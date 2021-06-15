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
 - API adapted for jsonnet from fluxcd/kustomize-controller
*/

package v1

import (
	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/dependency"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KonfigurationSpec defines the desired state of a Konfiguration
type KonfigurationSpec struct {
	// DependsOn may contain a dependency.CrossNamespaceDependencyReference slice
	// with references to Konfiguration resources that must be ready before this
	// Konfiguration can be reconciled.
	// +optional
	DependsOn []dependency.CrossNamespaceDependencyReference `json:"dependsOn,omitempty"`

	// The interval at which to reconcile the Konfiguration.
	// +required
	Interval metav1.Duration `json:"interval"`

	// The interval at which to retry a previously failed reconciliation.
	// When not specified, the controller uses the KonfigurationSpec.Interval
	// value to retry failures.
	// +optional
	RetryInterval *metav1.Duration `json:"retryInterval,omitempty"`

	// The KubeConfig for reconciling the Konfiguration on a remote cluster.
	// Defaults to the in-cluster configuration.
	// +optional
	KubeConfig *KubeConfig `json:"kubeConfig,omitempty"`

	// Path to the jsonnet, json, or yaml that should be applied to the cluster.
	// Defaults to 'None', which translates to the root path of the SourceRef.
	// When declared as a file path it is assumed to be from the root path of the SourceRef.
	// You may also define a HTTP(S) link to fetch files from a remote location.
	// +required
	Path string `json:"path"`

	// Additional search paths to add to the jsonnet importer. These are relative to
	// the root of the sourceRef.
	// +optional
	JsonnetPaths []string `json:"jsonnerPaths,omitempty"`

	// Additional HTTP(S) URLs to add to the jsonnet importer.
	// +optional
	JsonnetURLs []string `json:"jsonnetURLs,omitempty"`

	// Variables to use when invoking kubecfg to render manifests.
	// +optional
	Variables *Variables `json:"variables,omitempty"`

	// The name of the Kubernetes service account to impersonate
	// when reconciling this Konfiguration.
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Reference of the source where the jsonnet, json, or yaml file(s) are.
	// +optional
	SourceRef *CrossNamespaceSourceReference `json:"sourceRef"`

	// Prune enables garbage collection. Note that this makes commands take
	// considerably longer, so you may want to adjust your timeouts accordingly.
	// +required
	Prune bool `json:"prune"`

	// A list of resources to be included in the health assessment.
	// +optional
	HealthChecks []meta.NamespacedObjectKindReference `json:"healthChecks,omitempty"`

	// This flag tells the controller to suspend subsequent kubecfg executions,
	// it does not apply to already started executions. Defaults to false.
	// +optional
	Suspend bool `json:"suspend,omitempty"`

	// Timeout for diff, validation, apply, and health checking operations.
	// Defaults to 'Interval' duration.
	// +optional
	Timeout *metav1.Duration `json:"timeout,omitempty"`

	// Validate input against the server schema, defaults to true. At the moment
	// this just implies a dry-run before patch/create operations.
	// This will be updated to support different methods of validation.
	// +kubebuilder:default:=true
	// +optional
	Validate bool `json:"validate,omitempty"`

	// Force instructs the controller to recreate resources
	// when patching fails due to an immutable field change.
	// +kubebuilder:default:=false
	// +optional
	// Force bool `json:"force,omitempty"`
}

// KubeConfig holds the configuration for where to fetch the contents of a
// kubeconfig file.
type KubeConfig struct {
	// SecretRef holds the name to a secret that contains a 'value' key with
	// the kubeconfig file as the value. It must be in the same namespace as
	// the Konfiguration.
	// It is recommended that the kubeconfig is self-contained, and the secret
	// is regularly updated if credentials such as a cloud-access-token expire.
	// Cloud specific `cmd-path` auth helpers will not function without adding
	// binaries and credentials to the Pod that is responsible for reconciling
	// the Konfiguration.
	// +required
	SecretRef corev1.LocalObjectReference `json:"secretRef,omitempty"`
}

// Variables describe code/strings for external variables and top-level arguments.
type Variables struct {
	// Values of external variables with string values.
	// +optional
	ExtStr map[string]string `json:"extStr,omitempty"`
	// Values of external variables with values supplied as Jsonnet code.
	// +optional
	ExtCode map[string]string `json:"extCode,omitempty"`
	// Values of top level arguments with string values.
	// +optional
	TLAStr map[string]string `json:"tlaStr,omitempty"`
	// Values of top level arguments with values supplied as Jsonnet code.
	// +optional
	TLACode map[string]string `json:"tlaCode,omitempty"`
}

// KonfigurationStatus defines the observed state of Konfiguration
type KonfigurationStatus struct {
	// ObservedGeneration is the last reconciled generation.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// The last successfully applied revision.
	// The revision format for Git sources is <branch|tag>/<commit-sha>.
	// For HTTP(S) paths it will just be the URL.
	// +optional
	LastAppliedRevision string `json:"lastAppliedRevision,omitempty"`

	// LastAttemptedRevision is the revision of the last reconciliation attempt.
	// For HTTP(S) paths it will just be the URL.
	// +optional
	LastAttemptedRevision string `json:"lastAttemptedRevision,omitempty"`

	// The last successfully applied revision metadata.
	// +optional
	Snapshot *Snapshot `json:"snapshot,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:shortName=konfig;konfigs;konf;konfs
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CurrentRevision",type="string",JSONPath=".status.lastAppliedRevision",priority=1
// +kubebuilder:printcolumn:name="Checksum",type="string",JSONPath=".status.snapshot.checksum",priority=1
// +kubebuilder:printcolumn:name="LastAttemptedRevision",type="string",JSONPath=".status.lastAttemptedRevision",priority=1

// Konfiguration is the Schema for the konfigurations API
type Konfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KonfigurationSpec   `json:"spec,omitempty"`
	Status KonfigurationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KonfigurationList contains a list of Konfiguration
type KonfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Konfiguration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Konfiguration{}, &KonfigurationList{})
}
