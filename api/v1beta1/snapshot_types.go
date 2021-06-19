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
  - Adaption from fluxcd/kustomize-controller for jsonnet
  - Create snapshots with internal checksum compute from list
    of unstrucutred objects.
*/

package v1beta1

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Snapshot holds the metadata of the Kubernetes objects
// generated for a source revision
type Snapshot struct {
	// The manifests sha1 checksum.
	// +required
	Checksum string `json:"checksum"`

	// A list of Kubernetes kinds grouped by namespace.
	// +required
	Entries []SnapshotEntry `json:"entries"`
}

// SnapshotEntry holds the metadata of namespaced
// Kubernetes objects
type SnapshotEntry struct {
	// The namespace of this entry.
	// +optional
	Namespace string `json:"namespace"`

	// The list of Kubernetes kinds.
	// +required
	Kinds map[string]string `json:"kinds"`
}

// NewSnapshotFromUnstructured creates a new snapshot from the given list of unstructured
// objects.
func NewSnapshotFromUnstructured(objects []*unstructured.Unstructured) (*Snapshot, error) {
	bytes, err := json.Marshal(objects)
	if err != nil {
		return nil, err
	}
	h := sha1.New()
	if _, err := h.Write(bytes); err != nil {
		return nil, err
	}
	snapshot := &Snapshot{
		Checksum: fmt.Sprintf("%x", h.Sum(nil)),
		Entries:  []SnapshotEntry{},
	}
	for _, o := range objects {
		snapshot.addEntry(o)
	}
	return snapshot, nil
}

func (s *Snapshot) addEntry(item *unstructured.Unstructured) {
	found := false
	for _, tracker := range s.Entries {
		if tracker.Namespace == item.GetNamespace() {
			tracker.Kinds[item.GroupVersionKind().String()] = item.GetKind()
			found = true
			break
		}
	}
	if !found {
		s.Entries = append(s.Entries, SnapshotEntry{
			Namespace: item.GetNamespace(),
			Kinds: map[string]string{
				item.GroupVersionKind().String(): item.GetKind(),
			},
		})
	}
}

// NonNamespacedKinds returns the cluster-scoped kinds in this snapshot.
func (s *Snapshot) NonNamespacedKinds() []schema.GroupVersionKind {
	kinds := make([]schema.GroupVersionKind, 0)

	for _, tracker := range s.Entries {
		if tracker.Namespace == "" {
			for gvk, kind := range tracker.Kinds {
				if strings.Contains(gvk, ",") {
					gv, err := schema.ParseGroupVersion(strings.Split(gvk, ",")[0])
					if err == nil {
						kinds = append(kinds, gv.WithKind(kind))
					}
				}
			}
		}
	}
	return kinds
}

// NamespacedKinds returns the namespaced kinds in this snapshot.
func (s *Snapshot) NamespacedKinds() map[string][]schema.GroupVersionKind {
	nsk := make(map[string][]schema.GroupVersionKind)
	for _, tracker := range s.Entries {
		if tracker.Namespace != "" {
			var kinds []schema.GroupVersionKind
			for gvk, kind := range tracker.Kinds {
				if strings.Contains(gvk, ",") {
					gv, err := schema.ParseGroupVersion(strings.Split(gvk, ",")[0])
					if err == nil {
						kinds = append(kinds, gv.WithKind(kind))
					}
				}
			}
			nsk[tracker.Namespace] = kinds
		}
	}
	return nsk
}
