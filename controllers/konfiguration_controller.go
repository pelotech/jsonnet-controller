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
	"os"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/events"
	"github.com/fluxcd/pkg/runtime/metrics"
	"github.com/fluxcd/pkg/runtime/predicates"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kuberecorder "k8s.io/client-go/tools/record"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/hashicorp/go-retryablehttp"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1"
	"github.com/pelotech/jsonnet-controller/pkg/jsonnet"
	"github.com/pelotech/jsonnet-controller/pkg/resources"
)

// KonfigurationReconciler reconciles a Konfiguration object
type KonfigurationReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	EventRecorder         kuberecorder.EventRecorder
	ExternalEventRecorder *events.Recorder
	MetricsRecorder       *metrics.Recorder
	StatusPoller          *polling.StatusPoller

	httpClient                *retryablehttp.Client
	dependencyRequeueDuration time.Duration
	jsonnetCache              string
	dryRunTimeout             time.Duration

	httpLog logr.Logger
}

type ReconcilerOptions struct {
	MaxConcurrentReconciles   int
	HTTPRetryMax              int
	DependencyRequeueInterval time.Duration
	JsonnetCacheDirectory     string
	DryRunRequestTimeout      time.Duration
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
	r.dependencyRequeueDuration = opts.DependencyRequeueInterval
	r.jsonnetCache = opts.JsonnetCacheDirectory
	r.dryRunTimeout = opts.DryRunRequestTimeout
	r.httpLog = log.WithName("webhook")

	// Index the Kustomizations by the GitRepository references they (may) point at.
	if err := mgr.GetCache().IndexField(context.TODO(), &konfigurationv1.Konfiguration{}, konfigurationv1.GitRepositoryIndexKey,
		r.indexBy(sourcev1.GitRepositoryKind)); err != nil {
		return fmt.Errorf("failed setting index fields: %w", err)
	}

	// Index the Kustomizations by the Bucket references they (may) point at.
	if err := mgr.GetCache().IndexField(context.TODO(), &konfigurationv1.Konfiguration{}, konfigurationv1.BucketIndexKey,
		r.indexBy(sourcev1.BucketKind)); err != nil {
		return fmt.Errorf("failed setting index fields: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&konfigurationv1.Konfiguration{}, builder.WithPredicates(
			predicate.Or(predicate.GenerationChangedPredicate{}, predicates.ReconcileRequestedPredicate{}),
		)).Watches(
		&source.Kind{Type: &sourcev1.GitRepository{}},
		handler.EnqueueRequestsFromMapFunc(r.requestsForRevisionChangeOf(konfigurationv1.GitRepositoryIndexKey)),
		builder.WithPredicates(SourceRevisionChangePredicate{}),
	).Watches(
		&source.Kind{Type: &sourcev1.Bucket{}},
		handler.EnqueueRequestsFromMapFunc(r.requestsForRevisionChangeOf(konfigurationv1.BucketIndexKey)),
		builder.WithPredicates(SourceRevisionChangePredicate{}),
	).WithOptions(
		controller.Options{MaxConcurrentReconciles: opts.MaxConcurrentReconciles},
	).Complete(r)
}

// The below do not cover all needed rbac permissions. It should really be defined by the user
// what they want the manager to be capable of.

// +kubebuilder:rbac:groups=jsonnet.io,resources=konfigurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=jsonnet.io,resources=konfigurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=jsonnet.io,resources=konfigurations/finalizers,verbs=update
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
	konfig := &konfigurationv1.Konfiguration{}
	if err := r.Client.Get(ctx, req.NamespacedName, konfig); err != nil {
		// Check if object was deleted
		if client.IgnoreNotFound(err) == nil {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Add our finalizer if it does not exist
	if !controllerutil.ContainsFinalizer(konfig, konfigurationv1.KonfigurationFinalizer) {
		reqLogger.Info("Registering finalizer to Konfiguration")
		controllerutil.AddFinalizer(konfig, konfigurationv1.KonfigurationFinalizer)
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

	// Check if there are any dependencies and that they are all ready
	if err := r.checkDependencies(ctx, konfig); err != nil {
		if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(
			revision, meta.DependencyNotReadyReason, err.Error(),
		)); statusErr != nil {
			reqLogger.Error(err, "failed to update status for dependency not ready")
		}
		msg := fmt.Sprintf("Dependencies do not meet ready condition, retrying in %s", r.dependencyRequeueDuration.String())
		reqLogger.Info(msg)
		r.event(ctx, konfig, &EventData{
			Revision: revision,
			Severity: events.EventSeverityInfo,
			Message:  msg,
		})
		return ctrl.Result{RequeueAfter: r.dependencyRequeueDuration}, nil
	}

	// Do reconciliation
	snapshot, err := r.reconcile(ctx, konfig, revision, path)
	if err != nil {
		reqLogger.Error(err, "Error during reconciliation")
		r.event(ctx, konfig, &EventData{
			Revision: revision,
			Severity: events.EventSeverityError,
			Message:  err.Error(),
		})
		return ctrl.Result{
			RequeueAfter: konfig.GetRetryInterval(),
		}, nil
	}

	updated := konfig.Status.Snapshot == nil || snapshot.Checksum != konfig.Status.Snapshot.Checksum

	// Set the konfiguration as ready
	msg := fmt.Sprintf("Applied revision: %s", revision)
	if err := konfig.SetReady(ctx, r.Client, snapshot, konfigurationv1.NewStatusMeta(
		revision, meta.ReconciliationSucceededReason, msg),
	); err != nil {
		return ctrl.Result{Requeue: true}, err
	}

	reqLogger.Info(fmt.Sprintf("Reconcile finished, next run in %s", konfig.GetInterval().String()), "Revision", revision)

	if updated {
		r.event(ctx, konfig, &EventData{
			Revision: revision,
			Severity: events.EventSeverityInfo,
			Message:  "Update Complete",
			Metadata: map[string]string{
				"commit_status": "update",
			},
		})
	}
	return ctrl.Result{
		RequeueAfter: konfig.GetInterval(),
	}, nil
}

func (r *KonfigurationReconciler) reconcile(ctx context.Context, konfig *konfigurationv1.Konfiguration, revision, path string) (*konfigurationv1.Snapshot, error) {
	reqLogger := log.FromContext(ctx)

	// Allocate a new temp directory for the current reconcile's workspace
	dirPath, err := ioutil.TempDir("", konfig.GetName())
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dirPath)

	// Create any necessary kube-clients for impersonation
	impersonation := NewKonfigurationImpersonation(konfig, r.Client, r.StatusPoller, dirPath)
	kubeClient, statusPoller, err := impersonation.GetClient(ctx)
	if err != nil {
		if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(
			revision, meta.ReconciliationFailedReason, err.Error()),
		); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		return nil, fmt.Errorf("failed to build kube client: %w", err)
	}

	// Create a builder to evaluate the jsonnet
	builder, err := jsonnet.NewBuilder(konfig, dirPath, r.jsonnetCache)
	if err != nil {
		if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(
			revision, meta.ReconciliationFailedReason, err.Error()),
		); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		return nil, fmt.Errorf("failed to initialize jsonnet builder: %w", err)
	}

	// Check is path is a directory. If so, assume a 'main.jsonnet' file.
	if strings.HasSuffix(path, "/") {
		path, err = securejoin.SecureJoin(path, "main.jsonnet")
		if err != nil {
			if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(
				revision, meta.ReconciliationFailedReason, err.Error()),
			); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			return nil, fmt.Errorf("failed to determine jsonnet path: %w", err)
		}
	}

	// Build the jsonnet
	buildOutput, err := builder.Build(ctx, kubeClient.RESTMapper(), path)
	if err != nil {
		if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(
			revision, meta.ReconciliationFailedReason, err.Error()),
		); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		return nil, fmt.Errorf("failed to build jsonnet: %w", err)
	}

	// Extract the yaml stream and checksum from the buildOutput
	manifests, err := buildOutput.YAMLStream()
	if err != nil {
		if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(
			revision, meta.ReconciliationFailedReason, err.Error()),
		); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		return nil, fmt.Errorf("failed to convert jsonnet output to yaml stream: %w", err)
	}
	checksum, err := buildOutput.SHA1Sum()
	if err != nil {
		if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(
			revision, meta.ReconciliationFailedReason, err.Error()),
		); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		return nil, fmt.Errorf("failed to compute checksum of jsonnet output: %w", err)
	}

	// Create a snapshot from the build output
	snapshot, err := konfigurationv1.NewSnapshot(manifests, checksum)
	if err != nil {
		if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(revision, meta.ReconciliationFailedReason, err.Error())); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		return nil, fmt.Errorf("failed to compute snapshot of manifests: %w", err)
	}

	// Create a resource manager for the konfiguration
	manager := resources.NewResourceManager(kubeClient, konfig)

	// Reconcile resources from the output
	if changeset, err := manager.Reconcile(ctx, snapshot, manifests); err != nil {
		if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(
			revision, meta.ReconciliationFailedReason, err.Error()),
		); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		r.event(ctx, konfig, &EventData{
			Revision: revision,
			Severity: events.EventSeverityError,
			Message:  changeset,
			Metadata: map[string]string{},
		})
		return nil, fmt.Errorf("failed to reconcile manifests: %w", err)
	} else if changeset != "" {
		r.event(ctx, konfig, &EventData{
			Revision: revision,
			Severity: events.EventSeverityInfo,
			Message:  changeset,
			Metadata: map[string]string{},
		})
	}

	// Prune any orphaned resources if enabled
	if konfig.GCEnabled() {
		if changeset, ok := manager.Prune(ctx, konfig.Status.Snapshot, snapshot); !ok {
			msg := fmt.Sprintf("failed to garbage-collect orphaned resources: %s", changeset)
			if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(
				revision, konfigurationv1.PruneFailedReason, msg),
			); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			r.event(ctx, konfig, &EventData{
				Revision: revision,
				Severity: events.EventSeverityError,
				Message:  changeset,
				Metadata: map[string]string{},
			})
			return nil, fmt.Errorf(msg)
		} else if changeset != "" {
			r.event(ctx, konfig, &EventData{
				Revision: revision,
				Severity: events.EventSeverityInfo,
				Message:  changeset,
				Metadata: map[string]string{},
			})
		}
	}

	// Check healthiness
	if err := r.checkHealth(ctx, statusPoller, konfig, revision); err != nil {
		if statusErr := konfig.SetNotReadySnapshot(ctx, r.Client, snapshot, konfigurationv1.NewStatusMeta(
			revision, konfigurationv1.HealthCheckFailedReason, err.Error()),
		); statusErr != nil {
			reqLogger.Error(statusErr, "Failed to update Konfiguration status")
		}
		return nil, err
	}

	return snapshot, nil
}

func (r *KonfigurationReconciler) reconcileDelete(ctx context.Context, konfig *konfigurationv1.Konfiguration) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	// If the konfig had prunening enabled and wasn't suspended for deletion
	// Run a kubecfg delete.
	if konfig.GCEnabled() && !konfig.IsSuspended() {
		// Allocate a new temp directory for the current reconcile's workspace
		dirPath, err := ioutil.TempDir("", konfig.GetName())
		if err != nil {
			return ctrl.Result{}, err
		}
		defer os.RemoveAll(dirPath)

		if err != nil {
			r.event(ctx, konfig, &EventData{
				Revision: konfig.Status.LastAppliedRevision,
				Severity: events.EventSeverityError,
				Message:  err.Error(),
			})
			reqLogger.Info("Could not write the allocate a temp directory")
			return ctrl.Result{}, err
		}

		impersonation := NewKonfigurationImpersonation(konfig, r.Client, r.StatusPoller, dirPath)
		kubeClient, _, err := impersonation.GetClient(ctx)
		if err != nil {
			r.event(ctx, konfig, &EventData{
				Revision: konfig.Status.LastAppliedRevision,
				Severity: events.EventSeverityError,
				Message:  err.Error(),
			})
			return ctrl.Result{}, fmt.Errorf("failed to build kube client: %w", err)
		}

		// Create a resource manager for the konfiguration
		manager := resources.NewResourceManager(kubeClient, konfig)

		if changeset, ok := manager.Prune(ctx, konfig.Status.Snapshot, nil); !ok {
			r.event(ctx, konfig, &EventData{
				Revision: konfig.Status.LastAppliedRevision,
				Severity: events.EventSeverityError,
				Message:  changeset,
			})
			return ctrl.Result{}, fmt.Errorf("failed to garbage-collect orphaned resources: %s", changeset)
		} else if changeset != "" {
			r.event(ctx, konfig, &EventData{
				Revision: konfig.Status.LastAppliedRevision,
				Severity: events.EventSeverityInfo,
				Message:  changeset,
			})
		}
	}

	// Remove our finalizer from the list and update it
	controllerutil.RemoveFinalizer(konfig, konfigurationv1.KonfigurationFinalizer)
	if err := r.Update(ctx, konfig); err != nil {
		r.event(ctx, konfig, &EventData{
			Revision: konfig.Status.LastAppliedRevision,
			Severity: events.EventSeverityError,
			Message:  err.Error(),
		})
		return ctrl.Result{}, err
	}

	// Stop reconciliation as the object is being deleted
	return ctrl.Result{}, nil
}

// checkHealth checks the healthiness of the konfiguration after an apply
func (r *KonfigurationReconciler) checkHealth(ctx context.Context, statusPoller *polling.StatusPoller, konfig *konfigurationv1.Konfiguration, revision string) error {
	if len(konfig.GetHealthChecks()) == 0 {
		return nil
	}

	hc := NewHealthCheck(konfig, statusPoller)

	if err := hc.Assess(1 * time.Second); err != nil {
		return err
	}

	healthiness := apimeta.FindStatusCondition(konfig.Status.Conditions, konfigurationv1.HealthyCondition)
	healthy := healthiness != nil && healthiness.Status == metav1.ConditionTrue

	if !healthy || (konfig.Status.LastAppliedRevision != revision) {
		r.event(ctx, konfig, &EventData{
			Revision: revision,
			Severity: events.EventSeverityInfo,
			Message:  "Health check passed",
		})
	}

	return nil
}
