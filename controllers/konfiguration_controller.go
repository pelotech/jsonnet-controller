/*
Copyright 2021 Pelotech.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/predicates"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-retryablehttp"

	appsv1 "github.com/pelotech/kubecfg-operator/api/v1"
)

// KonfigurationReconciler reconciles a Konfiguration object
type KonfigurationReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	httpClient *retryablehttp.Client
	opts       *ReconcilerOptions
}

type ReconcilerOptions struct {
	HTTPRetryMax int
}

// SetupWithManager sets up the controller with the Manager.
func (r *KonfigurationReconciler) SetupWithManager(log logr.Logger, mgr ctrl.Manager, opts *ReconcilerOptions) error {
	// Set up an http client for fetching artifacts
	httpClient := retryablehttp.NewClient()
	httpClient.RetryWaitMin = 5 * time.Second
	httpClient.RetryWaitMax = 30 * time.Second
	httpClient.RetryMax = opts.HTTPRetryMax
	httpClient.Logger = nil
	r.httpClient = httpClient
	r.opts = opts

	// Index the Kustomizations by the GitRepository references they (may) point at.
	if err := mgr.GetCache().IndexField(context.TODO(), &appsv1.Konfiguration{}, appsv1.GitRepositoryIndexKey,
		r.indexBy(sourcev1.GitRepositoryKind)); err != nil {
		return fmt.Errorf("failed setting index fields: %w", err)
	}

	// Index the Kustomizations by the Bucket references they (may) point at.
	if err := mgr.GetCache().IndexField(context.TODO(), &appsv1.Konfiguration{}, appsv1.BucketIndexKey,
		r.indexBy(sourcev1.BucketKind)); err != nil {
		return fmt.Errorf("failed setting index fields: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Konfiguration{}, builder.WithPredicates(
			predicate.Or(predicate.GenerationChangedPredicate{}, predicates.ReconcileRequestedPredicate{}),
		)).Watches(
		&source.Kind{Type: &sourcev1.GitRepository{}},
		handler.EnqueueRequestsFromMapFunc(r.requestsForRevisionChangeOf(appsv1.GitRepositoryIndexKey)),
		builder.WithPredicates(SourceRevisionChangePredicate{}),
	).Watches(
		&source.Kind{Type: &sourcev1.Bucket{}},
		handler.EnqueueRequestsFromMapFunc(r.requestsForRevisionChangeOf(appsv1.BucketIndexKey)),
		builder.WithPredicates(SourceRevisionChangePredicate{}),
	).Complete(r)
}

// The below do not cover all needed rbac permissions. It should really be defined by the user
// what they want the manager to be capable of.

// +kubebuilder:rbac:groups=apps.kubecfg.io,resources=konfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.kubecfg.io,resources=konfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.kubecfg.io,resources=konfigurations/finalizers,verbs=update
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=buckets;gitrepositories,verbs=get;list;watch
// +kubebuilder:rbac:groups=source.toolkit.fluxcd.io,resources=buckets/status;gitrepositories/status,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets;serviceaccounts,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *KonfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	reqLogger.Info("Reconciling konfiguration")

	// Look up the konfiguration that triggered this request
	konfig := &appsv1.Konfiguration{}
	if err := r.Client.Get(ctx, req.NamespacedName, konfig); err != nil {
		// Check if object was deleted
		if client.IgnoreNotFound(err) == nil {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Add our finalizer if it does not exist
	if !controllerutil.ContainsFinalizer(konfig, appsv1.KonfigurationFinalizer) {
		reqLogger.Info("Registering finalizer to Konfiguration")
		controllerutil.AddFinalizer(konfig, appsv1.KonfigurationFinalizer)
		if err := r.Update(ctx, konfig); err != nil {
			reqLogger.Error(err, "failed to register finalizer")
			return ctrl.Result{}, err
		}
	}

	// Examine if the object is under deletion
	if !konfig.ObjectMeta.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, konfig)
	}

	// Check if the konfiguration is suspended
	if konfig.IsSuspended() {
		return ctrl.Result{
			RequeueAfter: konfig.GetInterval(),
		}, nil
	}

	// set the status to progressing
	if err := konfig.SetProgressing(ctx, r.Client); err != nil {
		reqLogger.Error(err, "unable to update status to progressing")
		return ctrl.Result{Requeue: true}, err
	}

	// Get the revision and the path we are going to operate on
	revision, path, clean, err := r.prepareSource(ctx, konfig)
	if err != nil {
		return ctrl.Result{
			RequeueAfter: konfig.GetRetryInterval(),
		}, nil
	}
	defer clean()

	// Build the jsonnet and compute a checksum
	manifests, checksum, err := r.build(ctx, konfig, path)
	if err != nil {
		meta := appsv1.NewStatusMeta(revision, appsv1.BuildFailedReason, err.Error())
		if statusErr := konfig.SetNotReady(ctx, r.Client, meta); statusErr != nil {
			reqLogger.Error(statusErr, "unable to update status after build error")
		}
		reqLogger.Error(err, "Error building the jsonnet")
		return ctrl.Result{
			RequeueAfter: konfig.GetRetryInterval(),
		}, nil
	}

	// Create a snapshot from the build output
	snapshot, err := appsv1.NewSnapshot(manifests, checksum)
	if err != nil {
		if statusErr := konfig.SetNotReady(ctx, r.Client, appsv1.NewStatusMeta(revision, meta.ReconciliationFailedReason, err.Error())); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		reqLogger.Error(err, "Error creating snapshot of manifests")
		return ctrl.Result{
			RequeueAfter: konfig.GetRetryInterval(),
		}, nil
	}

	// Do reconciliation
	err = r.reconcile(ctx, konfig, snapshot, revision, manifests)
	if err != nil {
		reqLogger.Error(err, "Error during reconciliation")
		return ctrl.Result{
			RequeueAfter: konfig.GetRetryInterval(),
		}, nil
	}

	// Set the konfiguration as ready
	msg := fmt.Sprintf("Applied revision: %s", revision)
	if err := konfig.SetReady(ctx, r.Client, snapshot, appsv1.NewStatusMeta(revision, meta.ReconciliationSucceededReason, msg)); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	return ctrl.Result{
		RequeueAfter: konfig.GetInterval(),
	}, nil
}

func (r *KonfigurationReconciler) reconcile(ctx context.Context, konfig *appsv1.Konfiguration, snapshot *appsv1.Snapshot, revision string, manifests []byte) error {
	reqLogger := log.FromContext(ctx)

	// Allocate a new temp directory for the generated manifest
	dir, err := ioutil.TempDir("", konfig.GetName())
	if err != nil {
		if statusErr := konfig.SetNotReadySnapshot(ctx, r.Client, snapshot, appsv1.NewStatusMeta(revision, sourcev1.StorageOperationFailedReason, err.Error())); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		reqLogger.Info("Could not allocate a temp directory for the generated manifest")
		return err
	}

	// Write the manifest to the temp directory - kubecfg needs to know the file extension
	// to know it's yaml at the moment.
	path := filepath.Join(dir, "manifest.yaml")
	if err := ioutil.WriteFile(path, manifests, 0600); err != nil {
		if statusErr := konfig.SetNotReadySnapshot(ctx, r.Client, snapshot, appsv1.NewStatusMeta(revision, sourcev1.StorageOperationFailedReason, err.Error())); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		reqLogger.Info("Could not write the generated manifest to disk")
		return err
	}

	// Run a diff first to determine if any actions are necessary
	updateRequired, err := runKubecfgDiff(ctx, konfig, path)
	if err != nil {
		if statusErr := konfig.SetNotReadySnapshot(ctx, r.Client, snapshot, appsv1.NewStatusMeta(revision, appsv1.ValidationFailedReason, err.Error())); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		return err
	}

	// If no update required, check on the next interval.
	if !updateRequired {
		return nil
	}

	// Run a dry-run - is also validation to some extant but this should all be cleaned up
	if err := runKubecfgUpdate(ctx, konfig, path, true); err != nil {
		if statusErr := konfig.SetNotReadySnapshot(ctx, r.Client, snapshot, appsv1.NewStatusMeta(revision, appsv1.ValidationFailedReason, err.Error())); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		return err
	}

	// Run an update
	if err := runKubecfgUpdate(ctx, konfig, path, false); err != nil {
		if statusErr := konfig.SetNotReadySnapshot(ctx, r.Client, snapshot, appsv1.NewStatusMeta(revision, meta.ReconciliationFailedReason, err.Error())); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		return err
	}

	return nil
}

func (r *KonfigurationReconciler) reconcileDelete(ctx context.Context, konfig *appsv1.Konfiguration) (ctrl.Result, error) {
	// If the konfig had prunening enabled and wasn't suspended for deletion
	// Run a kubecfg delete.
	if konfig.GCEnabled() && !konfig.IsSuspended() {
		_, path, clean, err := r.prepareSource(ctx, konfig)
		if err != nil {
			return ctrl.Result{
				RequeueAfter: konfig.GetRetryInterval(),
			}, nil
		}
		defer clean()

		if err := runKubecfgDelete(ctx, konfig, path); err != nil {
			return ctrl.Result{
				RequeueAfter: konfig.GetRetryInterval(),
			}, nil
		}
	}

	// Remove our finalizer from the list and update it
	controllerutil.RemoveFinalizer(konfig, appsv1.KonfigurationFinalizer)
	if err := r.Update(ctx, konfig); err != nil {
		return ctrl.Result{}, err
	}

	// Stop reconciliation as the object is being deleted
	return ctrl.Result{}, nil
}
