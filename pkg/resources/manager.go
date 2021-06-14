package resources

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/imdario/mergo"
	konfigurationv1 "github.com/pelotech/kubecfg-operator/api/v1"
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

// Manager is the main interface for reconciling resources from built
// manifests.
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
	ctx, cancel := context.WithTimeout(ctx, m.parent.GetTimeout())
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
				thischange, err = m.reconcileUnstructured(ctx, obj, snapshot.Checksum)
				changeSet += thischange
				return err
			})
			if err != nil {
				return
			}
			// Decode next object in the stream
			changeSet += thischange
			continue
		}

		// Reconcile the object
		thischange, err = m.reconcileUnstructured(ctx, toReconcile, snapshot.Checksum)
		changeSet += thischange
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

	ctx, cancel := context.WithTimeout(ctx, m.parent.GetTimeout())
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

			err := m.List(ctx, ulist, client.InNamespace(ns), m.matchingLabels())
			if err != nil {
				changeSet += fmt.Sprintf("failed to list objects for %s kind: %s\n", gvk.Kind, err.Error())
				success = false
				continue GVKs
			}

		Items:
			for _, item := range ulist.Items {
				id := fmt.Sprintf("%s/%s/%s", item.GetKind(), item.GetNamespace(), item.GetName())
				log.Info(fmt.Sprintf("Checking if '%s' should be pruned", id))

				if m.shouldNotPrune(&item) {
					log.Info(fmt.Sprintf("GC is disabled for '%s'", id))
					continue Items
				}

				if m.isOrphaned(&item, checksum) && item.GetDeletionTimestamp().IsZero() {
					log.Info(fmt.Sprintf("Deleting orphaned object %s", id))
					err = m.Delete(ctx, &item)
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
		err := m.List(ctx, ulist, m.matchingLabels())

		if err != nil {
			changeSet += fmt.Sprintf("failed to list objects for %s kind: %s\n", gvk.Kind, err.Error())
			success = false
		}

	ClusterItems:
		for _, item := range ulist.Items {
			id := fmt.Sprintf("%s/%s/%s", item.GetKind(), item.GetNamespace(), item.GetName())
			log.Info(fmt.Sprintf("Checking if '%s' should be pruned", id))

			if m.shouldNotPrune(&item) {
				log.Info(fmt.Sprintf("GC is disabled for '%s'", id))
				continue ClusterItems
			}

			if m.isOrphaned(&item, checksum) && item.GetDeletionTimestamp().IsZero() {
				log.Info(fmt.Sprintf("Deleting orphaned object %s", id))
				err = m.Delete(ctx, &item)
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

func (m *manager) reconcileUnstructured(ctx context.Context, obj *unstructured.Unstructured, fullChecksum string) (string, error) {
	nn := client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}
	id := fmt.Sprintf("%s/%s", obj.GetKind(), nn.String())
	log := log.FromContext(ctx)

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

	// Set the garbage collection labels on the object
	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	for k, v := range m.gcLabels(fullChecksum) {
		labels[k] = v
	}
	obj.SetLabels(labels)

	// Attempt to look up the object
	found := &unstructured.Unstructured{}
	found.SetGroupVersionKind(obj.GetObjectKind().GroupVersionKind())
	err = m.Get(ctx, nn, found)
	if err != nil {
		if client.IgnoreNotFound(err) == nil {
			// The object doesn't exist, create it
			// TODO: Support dry-run
			log.Info(fmt.Sprintf("Creating %s '%s'", obj.GetKind(), nn.String()))
			if m.parent.ShouldValidate() {
				if err := m.Create(ctx, obj, client.DryRunAll); err != nil {
					return fmt.Sprintf("create failed for '%s' (dry-run): %s\n", id, err.Error()), err
				}
			}
			if err := m.Create(ctx, obj); err != nil {
				return fmt.Sprintf("create failed for '%s': %s\n", id, err.Error()), err
			}
			return fmt.Sprintf("%s created\n", id), nil
		}
		// Return any other error
		return fmt.Sprintf("create failed for '%s': %s\n", id, err.Error()), err
	}

	// The object was found, check that its checksum matches that computed above
	foundAnnotations := found.GetAnnotations()

	// TODO: More intelligent patching

	if foundAnnotations == nil {
		// No annotations - we need to patch the object
		log.Info(fmt.Sprintf("Existing %s '%s' has no annotations, updating", obj.GetKind(), nn.String()))
		return m.patch(ctx, found, obj, id)
	}

	foundChecksum, ok := foundAnnotations[konfigurationv1.LastAppliedConfigAnnotation]
	if !ok {
		// No checksum annotation - we need to patch the object
		log.Info(fmt.Sprintf("Existing %s '%s' has no last-applied annotation, updating", obj.GetKind(), nn.String()))
		return m.patch(ctx, found, obj, id)
	}

	// Check if checksum has changed
	if foundChecksum != checksum {
		log.Info(fmt.Sprintf("%s '%s' definition has changed, updating", obj.GetKind(), nn.String()))
		return m.patch(ctx, found, obj, id)
	}

	log.Info(fmt.Sprintf("%s '%s' is up to date", obj.GetKind(), nn.String()))
	return "", nil
}

func (m *manager) patch(ctx context.Context, old, new *unstructured.Unstructured, id string) (string, error) {
	mergo.MergeWithOverwrite(old, new)
	if m.parent.ShouldValidate() {
		if err := m.Update(ctx, old, client.DryRunAll); err != nil {
			return fmt.Sprintf("update failed for '%s' (dry-run): %s\n", id, err.Error()), err
		}
	}
	if err := m.Update(ctx, old); err != nil {
		return fmt.Sprintf("update failed for '%s': %s\n", id, err.Error()), err
	}
	return fmt.Sprintf("%s configured\n", id), nil
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
