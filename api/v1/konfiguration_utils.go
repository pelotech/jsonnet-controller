package v1

import (
	"context"
	"fmt"

	"github.com/drone/envsubst"
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
func (k *Konfiguration) GetKubeConfig() *KubeConfig { return k.Spec.KubeConfig }

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
func (k *Konfiguration) GetPath() string { return k.Spec.Path }

// GetPostBuild returns any post build substitution to perform on the rendered
// manifests.
func (k *Konfiguration) GetPostBuild() *PostBuild { return k.Spec.PostBuild }

// Render will render the contents provided using envsubstr and the configured
// substitution variables.
func (p *PostBuild) Render(contents string) (string, error) {
	return envsubst.Eval(contents, func(k string) string {
		if val, ok := p.Substitute[k]; ok {
			return val
		}
		return ""
	})
}

// GCEnabled returns whether garbage collection should be conducted on kubecfg
// manifests.
func (k *Konfiguration) GCEnabled() bool { return k.Spec.Prune }

// IsSuspended returns whether the controller should not apply any manifests
// at the moment.
func (k *Konfiguration) IsSuspended() bool { return k.Spec.Suspend }

// GetValidation returns the type of validation strategy to use for the Konfiguration.
func (k *Konfiguration) GetValidation() Validation {
	if k.Spec.Validation != "" {
		return k.Spec.Validation
	}
	return ValidationNone
}

// ForceCreate returns whether the controller should force recreating resources
// when patching fails due to an immutable field change.
func (k *Konfiguration) ForceCreate() bool { return k.Spec.Force }
