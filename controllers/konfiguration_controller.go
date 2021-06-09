/*
Copyright 2021 Avi Zimmerman.

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
	"net/http"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/fluxcd/pkg/runtime/predicates"
	"github.com/fluxcd/pkg/untar"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-retryablehttp"

	appsv1 "github.com/pelotech/kubecfg-operator/api/v1"
)

// KonfigurationReconciler reconciles a Konfiguration object
type KonfigurationReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	httpClient *retryablehttp.Client
}

type ReconcilerOptions struct {
	FluxEnabled bool
}

// SetupWithManager sets up the controller with the Manager.
func (r *KonfigurationReconciler) SetupWithManager(log logr.Logger, mgr ctrl.Manager, opts *ReconcilerOptions) error {
	// Set up an http client for fetching artifacts
	httpClient := retryablehttp.NewClient()
	httpClient.RetryWaitMin = 5 * time.Second
	httpClient.RetryWaitMax = 30 * time.Second
	httpClient.RetryMax = 5
	httpClient.Logger = nil
	r.httpClient = httpClient

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

	log.Info("Setting up Konfigurations subscription")
	c := ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Konfiguration{}, builder.WithPredicates(
			predicate.Or(predicate.GenerationChangedPredicate{}, predicates.ReconcileRequestedPredicate{}),
		))

	if opts.FluxEnabled {
		log.Info("Subscribing to changes to GitRepositories")
		c = c.Watches(
			&source.Kind{Type: &sourcev1.GitRepository{}},
			handler.EnqueueRequestsFromMapFunc(r.requestsForRevisionChangeOf(appsv1.GitRepositoryIndexKey)),
			builder.WithPredicates(SourceRevisionChangePredicate{}),
		)
		log.Info("Subscribing to changes to Buckets")
		c = c.Watches(
			&source.Kind{Type: &sourcev1.Bucket{}},
			handler.EnqueueRequestsFromMapFunc(r.requestsForRevisionChangeOf(appsv1.BucketIndexKey)),
			builder.WithPredicates(SourceRevisionChangePredicate{}),
		)
	}

	return c.Complete(r)
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

var httpPathRegex = regexp.MustCompile("(https?)://")

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
		// TODO: Optional ownership of created resources?
		if client.IgnoreNotFound(err) == nil {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if the konfiguration is suspended
	if konfig.IsSuspended() {
		return ctrl.Result{
			RequeueAfter: konfig.GetInterval(),
		}, nil
	}

	// Initially set paths to those defined in spec. If we are running
	// against a source archive, they will be turned into absolute paths.
	// Otherwises they are probably http(s):// paths.
	path := konfig.GetPath()

	// Check if there is a reference to a source. This is a stop-gap solution
	// before full integration with source-controller.
	if sourceRef := konfig.GetSourceRef(); sourceRef != nil {
		source, err := sourceRef.GetSource(ctx, r.Client)
		if client.IgnoreNotFound(err) == nil {
			if err != nil {
				reqLogger.Error(err, "Failed to fetch source for Konfiguration")
				return ctrl.Result{
					RequeueAfter: konfig.GetRetryInterval(),
				}, nil
			}
		} else {
			return ctrl.Result{}, err
		}

		// Check if the artifact is not ready yet
		if source.GetArtifact() == nil {
			// TODO: status updates
			reqLogger.Info("Source is not ready, artifact not found")
			return ctrl.Result{RequeueAfter: konfig.GetRetryInterval()}, nil
		}

		// Create a temp directory for the artifact
		tmpDir, err := ioutil.TempDir("", konfig.GetName())
		if err != nil {
			reqLogger.Error(err, "Could not allocate a temp directory for source artifact")
			return ctrl.Result{
				RequeueAfter: konfig.GetRetryInterval(),
			}, nil
		}
		defer os.RemoveAll(tmpDir)

		// Download and extract the artifact
		if err := r.downloadAndExtractTo(source.GetArtifact().URL, tmpDir); err != nil {
			reqLogger.Error(err, "Failed to download source artifact")
			return ctrl.Result{
				RequeueAfter: konfig.GetRetryInterval(),
			}, nil
		}

		path, err = securejoin.SecureJoin(tmpDir, path)
		if err != nil {
			reqLogger.Error(err, "Failed to format path relative to tmp directory")
			return ctrl.Result{
				RequeueAfter: konfig.GetRetryInterval(),
			}, nil
		}

	}

	// Do reconciliation
	if err := r.reconcile(ctx, reqLogger, konfig, path); err != nil {
		reqLogger.Error(err, "Error during reconciliation")
		return ctrl.Result{
			RequeueAfter: konfig.GetRetryInterval(),
		}, nil
	}

	// TODO: Update status

	return ctrl.Result{
		RequeueAfter: konfig.GetInterval(),
	}, nil
}

func (r *KonfigurationReconciler) reconcile(ctx context.Context, reqLogger logr.Logger, konfig *appsv1.Konfiguration, path string) error {
	// Run a diff first to determine if any actions are necessary
	updateRequired, err := runKubecfgDiff(ctx, reqLogger, konfig, path)
	if err != nil {
		return err
	}

	// If no update required, check on the next interval.
	// TODO: check status
	if !updateRequired {
		return nil
	}

	// Run a dry-run
	if err := runKubecfgUpdate(ctx, reqLogger, konfig, path, true); err != nil {
		return err
	}

	// Run an update
	if err := runKubecfgUpdate(ctx, reqLogger, konfig, path, false); err != nil {
		return err
	}

	return nil
}

func (r *KonfigurationReconciler) downloadAndExtractTo(artifactURL, tmpDir string) error {
	if hostname := os.Getenv("SOURCE_CONTROLLER_LOCALHOST"); hostname != "" {
		u, err := url.Parse(artifactURL)
		if err != nil {
			return err
		}
		u.Host = hostname
		artifactURL = u.String()
	}

	req, err := retryablehttp.NewRequest(http.MethodGet, artifactURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create a new request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download artifact, error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download artifact from %s, status: %s", artifactURL, resp.Status)
	}

	if _, err = untar.Untar(resp.Body, tmpDir); err != nil {
		return fmt.Errorf("failed to untar artifact, error: %w", err)
	}

	return nil
}
