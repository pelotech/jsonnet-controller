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

package jsonnet

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ObjectSorter []*unstructured.Unstructured

func (o ObjectSorter) Len() int      { return len(o) }
func (o ObjectSorter) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o ObjectSorter) Less(i, j int) bool {
	// Ensure namespaces and crds are first
	for _, t := range []string{"Namespace", "CustomResourceDefinition"} {
		if o[i].GetKind() == t {
			if o[j].GetKind() == t {
				return o[i].GetName() < o[j].GetName()
			}
			return true
		}
		if o[j].GetKind() == t {
			if o[i].GetKind() == t {
				return o[i].GetName() < o[j].GetName()
			}
			return false
		}
	}
	return fmt.Sprintf("%s/%s", o[i].GetNamespace(), o[i].GetName()) <
		fmt.Sprintf("%s/%s", o[j].GetNamespace(), o[j].GetName())
}
