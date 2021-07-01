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

package v1beta1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-jsonnet"

	"github.com/fluxcd/pkg/apis/meta"
	"github.com/fluxcd/pkg/runtime/dependency"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetNamespacedName returns the namespaced name for this Konfiguration.
func (k *Konfiguration) GetNamespacedName() types.NamespacedName {
	return types.NamespacedName{
		Name:      k.GetName(),
		Namespace: k.GetNamespace(),
	}
}

// GetInterval returns the interval at which to reconcile the Konfiguration.
func (k *Konfiguration) GetInterval() time.Duration { return k.Spec.Interval.Duration }

// GetRetryInterval returns the interval at which to retry a previously failed
// reconciliation.
func (k *Konfiguration) GetRetryInterval() time.Duration {
	if k.Spec.RetryInterval != nil {
		return k.Spec.RetryInterval.Duration
	}
	return k.GetInterval()
}

// GetTimeout returns the timeout for validation, apply and health checking
// operations.
func (k *Konfiguration) GetTimeout() time.Duration {
	if k.Spec.Timeout != nil {
		return k.Spec.Timeout.Duration
	}
	return k.GetInterval()
}

// GetKubeConfig retrieves the kubeconfig to use for the operation if defined.
// When nil, it is assumed to use any client the caller already has configured
// (usually that of the controller-runtime at launch).
func (k *Konfiguration) GetKubeConfig() *KubeConfig { return k.Spec.KubeConfig }

// GetKubeConfigSecretName satisfies the Impersonator interface and returns the kubeconfig
// secret name, if any.
func (k *Konfiguration) GetKubeConfigSecretName() string {
	if cfg := k.GetKubeConfig(); cfg != nil {
		return cfg.SecretRef.Name
	}
	return ""
}

// GetServiceAccountName satisfies the Impersonator interface and returns the service account
// to assume, if any.
func (k *Konfiguration) GetServiceAccountName() string {
	return k.Spec.ServiceAccountName
}

// Fetch will use the given client and namespace to retrieve the contents of the
// kubeconfig from the referenced secret.
func (k *KubeConfig) Fetch(ctx context.Context, c client.Client, namespace string) (string, error) {
	nn := types.NamespacedName{
		Name:      k.SecretRef.Name,
		Namespace: namespace,
	}
	var secret corev1.Secret
	if err := c.Get(ctx, nn, &secret); err != nil {
		return "", err
	}
	if secret.Data == nil {
		return "", fmt.Errorf("secret '%s/%s' contains no data", secret.GetNamespace(), secret.GetName())
	}
	bytes, ok := secret.Data["value"]
	if !ok {
		return "", fmt.Errorf("secret '%s/%s' contains no 'value' key", secret.GetNamespace(), secret.GetName())
	}
	return string(bytes), nil
}

// GetPath returns the Path to the jsonnet, json, or yaml to evaluate.
func (k *Konfiguration) GetPath() string { return k.Spec.Path }

// GetVariables returns the external and top level arguments to pass to kubecfg.
func (k *Konfiguration) GetVariables() *Variables { return k.Spec.Variables }

// InjectIntoVM will inject the configured variables into the provided vm.
func (v *Variables) InjectIntoVM(vm *jsonnet.VM) error {
	for k, v := range v.ExtStr {
		vm.ExtVar(k, v)
	}
	for k, v := range v.ExtCode {
		vm.ExtCode(k, v)
	}
	for k, v := range v.TLAStr {
		vm.TLAVar(k, v)
	}
	for k, v := range v.TLACode {
		vm.TLACode(k, v)
	}
	if v.ExtVars != nil {
		var vars map[string]interface{}
		if err := json.Unmarshal(v.ExtVars.Raw, &vars); err != nil {
			return err
		}
		if err := iterVarsIntoVM(vars, vm.ExtCode); err != nil {
			return err
		}
	}
	if v.TLAVars != nil {
		var vars map[string]interface{}
		if err := json.Unmarshal(v.TLAVars.Raw, &vars); err != nil {
			return err
		}
		if err := iterVarsIntoVM(vars, vm.TLACode); err != nil {
			return err
		}
	}
	return nil
}

func iterVarsIntoVM(vars map[string]interface{}, codeFunc func(string, string)) error {
	for k, v := range vars {
		j, err := json.Marshal(v)
		if err != nil {
			return err
		}
		codeFunc(k, string(j))
	}
	return nil
}

// GetInjectSnippet returns any configured jsonnet snippet to add to the end of the execution.
func (k *Konfiguration) GetInjectSnippet() string {
	if k.Spec.Inject != "" {
		return "\n" + k.Spec.Inject
	}
	return ""
}

// GCEnabled returns whether garbage collection should be conducted on kubecfg
// manifests.
func (k *Konfiguration) GCEnabled() bool { return k.Spec.Prune }

// ShouldValidate returns true if server-side validation is enabled.
func (k *Konfiguration) ShouldValidate() bool { return k.Spec.Validate }

// IsSuspended returns whether the controller should not apply any manifests
// at the moment.
func (k *Konfiguration) IsSuspended() bool { return k.Spec.Suspend }

// ForceCreate returns whether the controller should force recreating resources
// when patching fails due to an immutable field change.
func (k *Konfiguration) ForceCreate() bool { return k.Spec.Force }

// GetDependsOn returns the konfigurations this one depends on.
func (k Konfiguration) GetDependsOn() (types.NamespacedName, []dependency.CrossNamespaceDependencyReference) {
	return k.GetNamespacedName(), k.Spec.DependsOn
}

// GetHealthChecks returns the health checks for this Konfiguration.
func (k *Konfiguration) GetHealthChecks() []meta.NamespacedObjectKindReference {
	return k.Spec.HealthChecks
}

// GetSourceRef returns the source ref for this konfiguration.
func (k *Konfiguration) GetSourceRef() *meta.NamespacedObjectKindReference {
	if k.Spec.SourceRef != nil {
		if k.Spec.SourceRef.Namespace == "" {
			k.Spec.SourceRef.Namespace = k.GetNamespace()
		}
		return k.Spec.SourceRef
	}
	return nil
}

// GetJsonnetPaths returns the search paths to configure in the jsonnet VM.
func (k *Konfiguration) GetJsonnetPaths() []string { return k.Spec.JsonnetPaths }

// GetJsonnetURLs returns the URL paths to configure in the jsonnet VM.
func (k *Konfiguration) GetJsonnetURLs() []string { return k.Spec.JsonnetURLs }
