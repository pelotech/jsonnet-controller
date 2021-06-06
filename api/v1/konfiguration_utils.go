package v1

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetInterval returns the interval at which to reconcile the Konfiguration.
func (k *Konfiguration) GetInterval() metav1.Duration { return k.Spec.Interval }

// GetRetryInterval returns the interval at which to retry a previously failed
// reconciliation.
func (k *Konfiguration) GetRetryInterval() metav1.Duration {
	if k.Spec.RetryInterval != nil {
		return *k.Spec.RetryInterval
	}
	return k.GetInterval()
}

// GetTimeout returns the timeout for validation, apply and health checking
// operations.
func (k *Konfiguration) GetTimeout() metav1.Duration {
	if k.Spec.Timeout != nil {
		return *k.Spec.Timeout
	}
	return k.GetInterval()
}

// GetKubeConfig retrieves the kubeconfig to use for the operation if defined.
// When nil, it is assumed to use any client the caller already has configured
// (usually that of the controller-runtime at launch).
// func (k *Konfiguration) GetKubeConfig() *KubeConfig { return k.Spec.KubeConfig }

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
		return "", fmt.Errorf("Secret '%s/%s' contains no data", secret.GetNamespace(), secret.GetName())
	}
	bytes, ok := secret.Data["value"]
	if !ok {
		return "", fmt.Errorf("Secret '%s/%s' contains no 'value' key", secret.GetNamespace(), secret.GetName())
	}
	return string(bytes), nil
}

// GetPath returns the Path to the jsonnet, json, or yaml to evaluate.
func (k *Konfiguration) GetPaths() []string { return k.Spec.Paths }

// GetVariables returns the external and top level arguments to pass to kubecfg.
func (k *Konfiguration) GetVariables() *Variables {
	return k.Spec.Variables
}

// AppendToArgs formats the configured variables to kubecfg command line arguments.
func (v *Variables) AppendToArgs(args []string) []string {
	for k, v := range v.ExtStr {
		args = append(args, []string{"--ext-str", fmt.Sprintf("%s=%s", k, v)}...)
	}
	for k, v := range v.ExtCode {
		args = append(args, []string{"--ext-code", fmt.Sprintf("%s=%s", k, v)}...)
	}
	for k, v := range v.TLAStr {
		args = append(args, []string{"--tla-str", fmt.Sprintf("%s=%s", k, v)}...)
	}
	for k, v := range v.TLACode {
		args = append(args, []string{"--tla-code", fmt.Sprintf("%s=%s", k, v)}...)
	}
	return args
}

// GetKubecfgArgs returns user-defined arguments to pass to kubecfg.
func (k *Konfiguration) GetKubecfgArgs() []string { return k.Spec.KubecfgArgs }

// GCEnabled returns whether garbage collection should be conducted on kubecfg
// manifests.
func (k *Konfiguration) GCEnabled() bool { return k.Spec.Prune }

// ValidateEnabled returns true if server-side validation is enabled.
func (k *Konfiguration) ValidateEnabled() bool { return k.Spec.Validate }

// IsSuspended returns whether the controller should not apply any manifests
// at the moment.
func (k *Konfiguration) IsSuspended() bool { return k.Spec.Suspend }

// GetDiffStrategy retrieves the diff strategy to use.
func (k *Konfiguration) GetDiffStrategy() string { return k.Spec.DiffStrategy }

// ForceCreate returns whether the controller should force recreating resources
// when patching fails due to an immutable field change.
// func (k *Konfiguration) ForceCreate() bool { return k.Spec.Force }
