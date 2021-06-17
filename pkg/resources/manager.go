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

package resources

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"time"

	"github.com/go-logr/logr"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1beta1"
	"github.com/pelotech/jsonnet-controller/pkg/diff"
)

// ReconcileeWithTimeout is an interface extending client.Object that includes
// a method for retrieving a reconciliation timeout.
type ReconcileeWithTimeout interface {
	client.Object
	// GetTimeout should return the timeout for reconciliation options
	GetTimeout() time.Duration
	// ShouldValidate should return whether objects should be validated with a dry-run
	// before creation or updating.
	// TODO: This just triggers a dry-run first, should be more intelligent
	ShouldValidate() bool
}

// Manager is the main interface for reconciling resources from built manifests.
type Manager interface {
	// Reconcile will reconcile the provided manifest of one or more objects with
	// the API server. The snapshot provided must match the manifest.
	Reconcile(ctx context.Context, snapshot *konfigurationv1.Snapshot, manifest []byte) (changeSet string, err error)
	// Prune will attempt to garbage-collect resources represented in the lastSnapshot that
	// were not created in the newSnapshot.
	Prune(ctx context.Context, lastSnapshot, newSnapshot *konfigurationv1.Snapshot) (changeSet string, success bool)
}

// NewKonfigurationManager creates a new resource manager for the given konfiguration
// using the given client.
func NewResourceManager(cl client.Client, parent ReconcileeWithTimeout) Manager {
	return &manager{Client: cl, parent: parent}
}

// manager implements the Manager interface.
type manager struct {
	client.Client
	parent ReconcileeWithTimeout
}

func (m *manager) Reconcile(ctx context.Context, snapshot *konfigurationv1.Snapshot, manifest []byte) (changeSet string, err error) {
	log := log.FromContext(ctx)

	reconcileCtx, cancel := context.WithTimeout(ctx, m.parent.GetTimeout())
	defer cancel()

	reader := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(manifest), 2048)

	for {
		// Read an object off the stream
		toReconcile := &unstructured.Unstructured{}
		err = reader.Decode(toReconcile)
		if err != nil {
			// If we've reached end of file, break
			if err == io.EOF {
				err = nil
				break
			}
			// Return any decoding errors, though this should not happen
			// as at this point the manifest has likely already gone through
			// a decoding.
			return
		}

		// If it's a list, iterate each object
		var thischange string

		if toReconcile.IsList() {
			err = toReconcile.EachListItem(func(item runtime.Object) error {
				obj := item.(*unstructured.Unstructured)
				thischange, err = m.reconcileUnstructured(reconcileCtx, log, obj, snapshot.Checksum)
				if thischange != "" {
					changeSet += thischange
				}
				return err
			})
			if err != nil {
				return
			}
			// Decode next object in the stream
			continue
		}

		// Reconcile the object
		thischange, err = m.reconcileUnstructured(reconcileCtx, log, toReconcile, snapshot.Checksum)
		if thischange != "" {
			changeSet += thischange
		}
		if err != nil {
			return
		}
	}

	return
}

// Prune will prune all resources reconciled by this manager that are not present in the provided
// snapshot. Namespaced objects are removed first, followed by global ones. An object is determined
// orphaned (for deletion) if it matches a label selector but with an old checksum. If an object is already
// marked for deletion it is ignored. Items marked
func (m *manager) Prune(ctx context.Context, lastSnapshot, newSnapshot *konfigurationv1.Snapshot) (changeSet string, success bool) {
	// any failure will set this to false
	success = true

	if lastSnapshot == nil {
		// there is nothing to do
		return
	}

	var checksum string
	if newSnapshot != nil {
		checksum = newSnapshot.Checksum
	}

	log := log.FromContext(ctx)

	pruneCtx, cancel := context.WithTimeout(ctx, m.parent.GetTimeout())
	defer cancel()

	// Iterate namespaced objects
	for ns, gvks := range lastSnapshot.NamespacedKinds() {

	GVKs:
		for _, gvk := range gvks {
			ulist := &unstructured.UnstructuredList{}
			ulist.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   gvk.Group,
				Version: gvk.Version,
				Kind:    fmt.Sprintf("%sList", gvk.Kind),
			})
			log.Info(fmt.Sprintf("Checking for orphaned %ss in %s namespace", gvk.Kind, ns))

			err := m.List(pruneCtx, ulist, client.InNamespace(ns), m.matchingLabels())
			if err != nil {
				changeSet += fmt.Sprintf("failed to list objects for %s kind: %s\n", gvk.Kind, err.Error())
				success = false
				continue GVKs
			}

		Items:
			for _, item := range ulist.Items {
				id := fmt.Sprintf("%s/%s/%s", item.GetKind(), item.GetNamespace(), item.GetName())

				if m.shouldNotPrune(&item) {
					log.Info(fmt.Sprintf("GC is disabled for '%s'", id))
					continue Items
				}

				if m.isOrphaned(&item, checksum) && item.GetDeletionTimestamp().IsZero() {
					log.Info(fmt.Sprintf("Deleting orphaned object %s", id))
					err = m.Delete(pruneCtx, &item)
					if err != nil {
						changeSet += fmt.Sprintf("delete failed for %s: %v\n", id, err)
						success = false
						continue Items
					}
					if len(item.GetFinalizers()) > 0 {
						changeSet += fmt.Sprintf("%s marked for deletion\n", id)
					} else {
						changeSet += fmt.Sprintf("%s deleted\n", id)
					}
				}
			}
		}
	}

	for _, gvk := range lastSnapshot.NonNamespacedKinds() {
		ulist := &unstructured.UnstructuredList{}
		ulist.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   gvk.Group,
			Version: gvk.Version,
			Kind:    fmt.Sprintf("%sList", gvk.Kind),
		})

		log.Info(fmt.Sprintf("Checking for orphaned %ss", gvk.Kind))
		err := m.List(pruneCtx, ulist, m.matchingLabels())

		if err != nil {
			changeSet += fmt.Sprintf("failed to list objects for %s kind: %s\n", gvk.Kind, err.Error())
			success = false
		}

	ClusterItems:
		for _, item := range ulist.Items {
			id := fmt.Sprintf("%s/%s/%s", item.GetKind(), item.GetNamespace(), item.GetName())

			if m.shouldNotPrune(&item) {
				log.Info(fmt.Sprintf("GC is disabled for '%s'", id))
				continue ClusterItems
			}

			if m.isOrphaned(&item, checksum) && item.GetDeletionTimestamp().IsZero() {
				log.Info(fmt.Sprintf("Deleting orphaned object %s", id))
				err = m.Delete(pruneCtx, &item)
				if err != nil {
					changeSet += fmt.Sprintf("delete failed for %s: %v\n", id, err)
					success = false
					continue ClusterItems
				}
				if len(item.GetFinalizers()) > 0 {
					changeSet += fmt.Sprintf("%s marked for deletion\n", id)
				} else {
					changeSet += fmt.Sprintf("%s deleted\n", id)
				}
			}
		}
	}

	return
}

func (m *manager) reconcileUnstructured(ctx context.Context, log logr.Logger, obj *unstructured.Unstructured, fullChecksum string) (string, error) {
	nn := client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}
	id := fmt.Sprintf("%s/%s", obj.GetKind(), nn.String())

	// Set the garbage collection labels on the object.
	// This needs to happen before computing the checksum for the object as it will ensure
	// labels are updated and the object is not pruned later.
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range m.gcLabels(fullChecksum) {
		labels[k] = v
	}
	obj.SetLabels(labels)

	// Compute the checksum for this object
	checksum, err := m.computeObjectChecksum(obj)
	if err != nil {
		return fmt.Sprintf("could not compute checksum for '%s': %s\n", id, err.Error()), err
	}

	// Set the checksum annotation on the object
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[konfigurationv1.LastAppliedConfigAnnotation] = checksum
	obj.SetAnnotations(annotations)

	// Attempt to look up the object
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	err = m.Get(ctx, nn, found)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// The object doesn't exist, create it
			log.Info(fmt.Sprintf("Creating %s '%s'", obj.GetKind(), nn.String()))
			if m.parent.ShouldValidate() {
				if err := m.serverSideApply(ctx, obj, true); err != nil {
					return fmt.Sprintf("create failed for '%s' (dry-run): %s\n", id, err.Error()), err
				}
			}
			if err := m.serverSideApply(ctx, obj, false); err != nil {
				return fmt.Sprintf("create failed for '%s': %s\n", id, err.Error()), err
			}
			return fmt.Sprintf("%s created\n", id), nil
		}
		// Return any other error
		return fmt.Sprintf("create failed for '%s': %s\n", id, err.Error()), err
	}

	// The object was found, check that its checksum matches that computed above
	foundAnnotations := found.GetAnnotations()

	if foundAnnotations == nil {
		// No annotations - we need to patch the object
		log.Info(fmt.Sprintf("Existing %s '%s' has no annotations, updating", obj.GetKind(), nn.String()))
		return m.patch(ctx, found, obj, id)
	}

	foundChecksum, ok := foundAnnotations[konfigurationv1.LastAppliedConfigAnnotation]
	if !ok {
		// No checksum annotation - we need to patch the object
		log.Info(fmt.Sprintf("Existing %s '%s' has no last-applied-checksum annotation, updating", obj.GetKind(), nn.String()))
		return m.patch(ctx, found, obj, id)
	}

	// Check if checksum has changed - this is easier then doing a diff and will tell us
	// if a change has happened since the last apply by the controller
	if foundChecksum != checksum {
		log.Info(fmt.Sprintf("%s '%s' definition has a new checksum, updating", obj.GetKind(), nn.String()),
			"OldChecksum", foundChecksum, "NewChecksum", checksum)
		return m.patch(ctx, found, obj, id)
	}

	// Do a full diff - this will attempt to detect drift
	if res, err := diff.Diff(obj, found); err != nil {
		return fmt.Sprintf("computing diff failed for '%s': %s\n", id, err.Error()), err
	} else if res.Modified {
		log.Info(fmt.Sprintf("%s '%s' definition has drifted, updating", obj.GetKind(), nn.String()))
		return m.patch(ctx, found, obj, id)
	}

	log.Info(fmt.Sprintf("%s '%s' is up to date", obj.GetKind(), nn.String()))
	return "", nil
}

func (m *manager) patch(ctx context.Context, old, new *unstructured.Unstructured, id string) (string, error) {
	if m.parent.ShouldValidate() {
		if err := m.serverSideApply(ctx, new, true); err != nil {
			return fmt.Sprintf("update failed for '%s' (dry-run): %s\n", id, err.Error()), err
		}
	}
	if err := m.serverSideApply(ctx, new, false); err != nil {
		return fmt.Sprintf("update failed for '%s': %s\n", id, err.Error()), err
	}
	return fmt.Sprintf("%s configured\n", id), nil
}

func (m *manager) serverSideApply(ctx context.Context, new *unstructured.Unstructured, dryRun bool) error {
	annotations := new.GetAnnotations()
	new.SetAnnotations(annotations)
	opts := []client.PatchOption{
		client.ForceOwnership,
		client.FieldOwner(konfigurationv1.ServerSideApplyOwner),
	}
	if dryRun {
		// DeepCopy the object for a dryrun or else managed fields will get populated
		return m.Patch(ctx, new.DeepCopy(), client.Apply, append(opts, client.DryRunAll)...)
	}
	return m.Patch(ctx, new, client.Apply, opts...)
}

func (m *manager) computeObjectChecksum(obj *unstructured.Unstructured) (checksum string, err error) {
	json, err := obj.MarshalJSON()
	if err != nil {
		return
	}
	h := sha1.New()
	if _, err = h.Write(json); err != nil {
		return
	}
	checksum = fmt.Sprintf("%x", h.Sum(nil))
	return
}

func (m *manager) shouldNotPrune(obj *unstructured.Unstructured) bool {
	labels := obj.GetLabels()
	annotations := obj.GetAnnotations()
	if labels == nil && annotations == nil {
		// All objects should have labels and annotations, but at least the skip one isn't there
		return false
	}

	for _, mp := range []map[string]string{labels, annotations} {
		if val, ok := mp[konfigurationv1.ResourceSkipPruning]; ok && val == konfigurationv1.PruningDisabledValue {
			return true
		}
	}

	return false
}

func (m *manager) isOrphaned(obj *unstructured.Unstructured, newChecksum string) bool {
	// If the parent is gone, then all selected objects are assumed orphaned
	if !m.parent.GetDeletionTimestamp().IsZero() {
		return true
	}
	labels := obj.GetLabels()
	if labels == nil {
		// All objects should have labels, we assume it is orphaned
		return true
	}
	if val, ok := labels[konfigurationv1.KonfigurationChecksumLabel]; ok {
		if val == newChecksum {
			return false
		}
	}
	return true
}

func (m *manager) matchingLabels() client.MatchingLabels { return m.selectorLabels() }

func (m *manager) selectorLabels() map[string]string {
	return map[string]string{
		konfigurationv1.KonfigurationNameLabel:      m.parent.GetName(),
		konfigurationv1.KonfigurationNamespaceLabel: m.parent.GetNamespace(),
	}
}

func (m *manager) gcLabels(checksum string) map[string]string {
	labels := m.selectorLabels()
	labels[konfigurationv1.KonfigurationChecksumLabel] = checksum
	return labels
}
