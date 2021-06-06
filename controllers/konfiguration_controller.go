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

	"github.com/fluxcd/pkg/runtime/predicates"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	appsv1 "github.com/tinyzimmer/kubecfg-operator/api/v1"
)

// KonfigurationReconciler reconciles a Konfiguration object
type KonfigurationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.kubecfg.io,resources=konfigurations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.kubecfg.io,resources=konfigurations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.kubecfg.io,resources=konfigurations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *KonfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.FromContext(ctx)

	reqLogger.Info("Reconciling konfiguration")

	konfig := &appsv1.Konfiguration{}
	if err := r.Client.Get(ctx, req.NamespacedName, konfig); err != nil {
		// Check if object was deleted
		// TODO: Optional ownership of created resources?
		if client.IgnoreNotFound(err) == nil {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Do reconciliation

	// Run a diff first to determine if any actions are necessary
	updateRequired, err := runKubecfgDiff(ctx, reqLogger, konfig)
	if err != nil {
		return ctrl.Result{
			RequeueAfter: konfig.GetRetryInterval().Duration,
		}, err
	}

	// If no update required, check on the next interval.
	// TODO: check status
	if !updateRequired {
		return ctrl.Result{
			RequeueAfter: konfig.GetInterval().Duration,
		}, nil
	}

	// Run a dry-run
	if err := runKubecfgUpdate(ctx, reqLogger, konfig, true); err != nil {
		return ctrl.Result{
			RequeueAfter: konfig.GetRetryInterval().Duration,
		}, err
	}

	// Run an update
	if err := runKubecfgUpdate(ctx, reqLogger, konfig, false); err != nil {
		return ctrl.Result{
			RequeueAfter: konfig.GetRetryInterval().Duration,
		}, err
	}

	// TODO: Update status

	return ctrl.Result{
		RequeueAfter: konfig.GetInterval().Duration,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KonfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Konfiguration{}, builder.WithPredicates(
			predicate.Or(predicate.GenerationChangedPredicate{}, predicates.ReconcileRequestedPredicate{}),
		)).
		Complete(r)
}
