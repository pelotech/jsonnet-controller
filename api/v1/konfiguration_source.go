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

package v1

import (
	"context"
	"fmt"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (sref *CrossNamespaceSourceReference) GetSource(ctx context.Context, c client.Client) (sourcev1.Source, error) {
	var source sourcev1.Source
	namespacedName := types.NamespacedName{
		Namespace: sref.Namespace,
		Name:      sref.Name,
	}
	switch sref.Kind {
	case sourcev1.GitRepositoryKind:
		var repository sourcev1.GitRepository
		err := c.Get(ctx, namespacedName, &repository)
		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				return source, err
			}
			return source, fmt.Errorf("unable to get source '%s': %w", namespacedName, err)
		}
		source = &repository
	case sourcev1.BucketKind:
		var bucket sourcev1.Bucket
		err := c.Get(ctx, namespacedName, &bucket)
		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				return source, err
			}
			return source, fmt.Errorf("unable to get source '%s': %w", namespacedName, err)
		}
		source = &bucket
	default:
		return source, fmt.Errorf("source `%s` kind '%s' not supported",
			sref.Name, sref.Kind)
	}
	return source, nil
}
