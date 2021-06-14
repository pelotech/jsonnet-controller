/*
Copyright 2020 The Flux authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright 2021 Pelotech - Apache License, Version 2.0.
  - Adaption from fluxcd/kustomize-controller
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/fluxcd/pkg/apis/meta"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1"
)

func (r *KonfigurationReconciler) checkDependencies(ctx context.Context, konfig *konfigurationv1.Konfiguration) error {
	log := log.FromContext(ctx)

	_, deps := konfig.GetDependsOn()

	if len(deps) == 0 {
		return nil
	}

	for _, dep := range deps {
		if dep.Namespace == "" {
			dep.Namespace = konfig.GetNamespace()
		}
		dName := types.NamespacedName(dep)

		log.Info(fmt.Sprintf("Checking dependency '%s'", dName.String()))
		var k konfigurationv1.Konfiguration
		err := r.Get(ctx, dName, &k)
		if err != nil {
			return fmt.Errorf("unable to get '%s' dependency: %w", dName, err)
		}

		if len(k.Status.Conditions) == 0 || k.Generation != k.Status.ObservedGeneration {
			return fmt.Errorf("dependency '%s' is not ready", dName)
		}

		if !apimeta.IsStatusConditionTrue(k.Status.Conditions, meta.ReadyCondition) {
			return fmt.Errorf("dependency '%s' is not ready", dName)
		}
	}

	log.Info("All dependencies area ready, proceeding with reconciliation")
	return nil
}
