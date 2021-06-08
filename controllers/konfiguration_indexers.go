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
*/

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

 - Adaption from fluxcd/kustomize-controller for kubecfg
*/

package controllers

import (
	"context"
	"fmt"

	appsv1 "github.com/tinyzimmer/kubecfg-operator/api/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/fluxcd/pkg/runtime/dependency"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
)

func (r *KonfigurationReconciler) requestsForRevisionChangeOf(indexKey string) func(obj client.Object) []reconcile.Request {
	return func(obj client.Object) []reconcile.Request {
		repo, ok := obj.(interface {
			GetArtifact() *sourcev1.Artifact
		})
		if !ok {
			panic(fmt.Sprintf("Expected an object conformed with GetArtifact() method, but got a %T", obj))
		}
		// If we do not have an artifact, we have no requests to make
		if repo.GetArtifact() == nil {
			return nil
		}

		ctx := context.Background()
		var list appsv1.KonfigurationList
		if err := r.List(ctx, &list, client.MatchingFields{
			indexKey: ObjectKey(obj).String(),
		}); err != nil {
			return nil
		}
		var dd []dependency.Dependent
		for _, d := range list.Items {
			// If the revision of the artifact equals to the last attempted revision,
			// we should not make a request for this Kustomization
			if repo.GetArtifact().Revision == d.Status.LastAttemptedRevision {
				continue
			}
			dd = append(dd, d)
		}
		sorted, err := dependency.Sort(dd)
		if err != nil {
			return nil
		}
		reqs := make([]reconcile.Request, len(sorted), len(sorted))
		for i := range sorted {
			reqs[i].NamespacedName.Name = sorted[i].Name
			reqs[i].NamespacedName.Namespace = sorted[i].Namespace
		}
		return reqs
	}
}

func (r *KonfigurationReconciler) indexBy(kind string) func(o client.Object) []string {
	return func(o client.Object) []string {
		k, ok := o.(*appsv1.Konfiguration)
		if !ok {
			panic(fmt.Sprintf("Expected a Konfiguration, got %T", o))
		}

		if k.Spec.SourceRef != nil && k.Spec.SourceRef.Kind == kind {
			namespace := k.GetNamespace()
			if k.Spec.SourceRef.Namespace != "" {
				namespace = k.Spec.SourceRef.Namespace
			}
			return []string{fmt.Sprintf("%s/%s", namespace, k.Spec.SourceRef.Name)}
		}

		return nil
	}
}

// ObjectKey returns client.ObjectKey for the object.
func ObjectKey(object metav1.Object) client.ObjectKey {
	return client.ObjectKey{
		Namespace: object.GetNamespace(),
		Name:      object.GetName(),
	}
}
