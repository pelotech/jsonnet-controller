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

package v1beta1

const (
	// GitRepositoryIndexKey is the key used for indexing kustomizations
	// based on their Git sources.
	GitRepositoryIndexKey string = ".metadata.gitRepository"
	// BucketIndexKey is the key used for indexing kustomizations
	// based on their S3 sources.
	BucketIndexKey string = ".metadata.bucket"
)

// The FieldOwner used for Server-Side Apply.
const ServerSideApplyOwner = "jsonnet-controller"

const (
	// The annotation added to objects containing the checksum of their last
	// applied configuration. Used to check if a patch is required.
	LastAppliedConfigAnnotation string = "jsonnet.io/last-applied-checksum"

	// KonfigurationNameLabel is the label added to objects to denote the Konfiguration
	// they belong to. Used for garbage collection.
	KonfigurationNameLabel string = "jsonnet.io/konfiguration-name"

	// KonfigurationNamespaceLabel is the label added to objects to denote the Konfiguration's
	// namespace they belong to. Used for garbage collection.
	KonfigurationNamespaceLabel string = "jsonnet.io/konfiguration-namespace"

	// KonfigurationChecksumLabel is the label added to objects containing the full checksum of
	// the built konfiguration being applied. Used for garbage collection.
	KonfigurationChecksumLabel string = "jsonnet.io/konfiguration-checksum"

	// ResourceSkipPruning is the label or annotation that a user can apply to resources to have
	// them skipped during pruning.
	ResourceSkipPruning string = "jsonnet.io/prune"

	// PruningDisabledValue is the value set to ResourceSkipPruningLabel to exclude an object from
	// pruning.
	PruningDisabledValue string = "disabled"
)

// KongifurationFinalizer is the finalizer placed on Konfiguration resources
const KonfigurationFinalizer string = "finalizers.jsonnet.io"
