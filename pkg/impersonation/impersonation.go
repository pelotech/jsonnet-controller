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
  - Adaption for Konfigurations from fluxcd/kustomize-controller
    - Caches kubeconfigs and serviceaccount tokens for subsequent calls
	- Standalone package operating on interfaces
*/

// Package impersonation contains an interface for impersonating different kubernetes
// clients based on the spec of a custom resource.
package impersonation

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// Impersonator is an interface to be implemented by CRs that need to assume the credentials
// of other Kubernetes entities during reconciliation.
type Impersonator interface {
	client.Object

	// GetKubeConfigSecretName should return the name of the secret in the object's namespace
	// containing a Kubeconfig. If no kubeconfig is configured, it should return an
	// empty string.
	GetKubeConfigSecretName() string
	// GetServiceAccountName should return the name of the service account to impersonate
	// in the object's namespace. If none is configured, it should return an empty string.
	GetServiceAccountName() string
}

// Impersonation provides methods for retrieving kubernetes clients and status pollers
// during a CR's reconciliation.
type Impersonation interface {
	// GetClient creates a controller-runtime client for talking to a Kubernetes API server.
	// If KubeConfig is set, will use the kubeconfig bytes from the Kubernetes secret.
	// If ServiceAccountName is set, will use the cluster provided kubeconfig impersonating the SA.
	// Otherwise will assume running in cluster and use the cluster provided kubeconfig.
	GetClient(ctx context.Context) (Client, error)
}

// NewImpersonation creates a new Impersonation using the given CR and client.
func NewImpersonation(imp Impersonator, kubeClient client.Client) Impersonation {
	return &impersonation{
		imp:    imp,
		Client: kubeClient,
	}
}

type impersonation struct {
	client.Client

	// The CR backing this impersonation
	imp Impersonator

	// cached assets
	serviceAccountToken string
	kubeconfigContents  []byte
}

// GetClient creates a controller-runtime client for talking to a Kubernetes API server.
// If KubeConfig is set, will use the kubeconfig bytes from the Kubernetes secret.
// If ServiceAccountName is set, will use the cluster provided kubeconfig impersonating the SA.
// Otherwise will assume running in cluster and use the cluster provided kubeconfig.
func (ki *impersonation) GetClient(ctx context.Context) (Client, error) {
	if kubeconfig := ki.imp.GetKubeConfigSecretName(); kubeconfig != "" {
		return ki.clientForKubeConfig(ctx)
	}
	if svcAccount := ki.imp.GetServiceAccountName(); svcAccount != "" {
		return ki.clientForServiceAccount(ctx)
	}
	return &clientWithPoller{ki.Client}, nil
}

func (ki *impersonation) clientForServiceAccount(ctx context.Context) (Client, error) {
	token, err := ki.getServiceAccountToken(ctx)
	if err != nil {
		return nil, err
	}
	restConfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	restConfig.BearerToken = token
	restConfig.BearerTokenFile = "" // Clear, as it overrides BearerToken

	restMapper, err := apiutil.NewDynamicRESTMapper(restConfig)
	if err != nil {
		return nil, err
	}

	client, err := client.New(restConfig, client.Options{Mapper: restMapper})
	if err != nil {
		return nil, err
	}

	return &clientWithPoller{client}, nil

}

func (ki *impersonation) clientForKubeConfig(ctx context.Context) (Client, error) {
	kubeConfigBytes, err := ki.getKubeConfig(ctx)
	if err != nil {
		return nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigBytes)
	if err != nil {
		return nil, err
	}

	restMapper, err := apiutil.NewDynamicRESTMapper(restConfig)
	if err != nil {
		return nil, err
	}

	client, err := client.New(restConfig, client.Options{Mapper: restMapper})
	if err != nil {
		return nil, err
	}

	return &clientWithPoller{client}, nil
}

func (ki *impersonation) getKubeConfig(ctx context.Context) ([]byte, error) {
	if ki.kubeconfigContents != nil {
		return ki.kubeconfigContents, nil
	}

	var secret corev1.Secret
	sname := types.NamespacedName{
		Namespace: ki.imp.GetNamespace(),
		Name:      ki.imp.GetKubeConfigSecretName(),
	}
	if err := ki.Get(ctx, sname, &secret); err != nil {
		return nil, fmt.Errorf("unable to read KubeConfig secret '%s' error: %w", sname.String(), err)
	}

	kubeConfig, ok := secret.Data["value"]
	if !ok {
		return nil, fmt.Errorf("KubeConfig secret '%s' doesn't contain a 'value' key ", sname.String())
	}

	ki.kubeconfigContents = kubeConfig
	return kubeConfig, nil
}

func (ki *impersonation) getServiceAccountToken(ctx context.Context) (string, error) {
	// Return an already retrieved serviceAccountToken
	if ki.serviceAccountToken != "" {
		return ki.serviceAccountToken, nil
	}

	namespacedName := types.NamespacedName{
		Namespace: ki.imp.GetNamespace(),
		Name:      ki.imp.GetServiceAccountName(),
	}

	var serviceAccount corev1.ServiceAccount
	err := ki.Client.Get(ctx, namespacedName, &serviceAccount)
	if err != nil {
		return "", err
	}

	secretName := types.NamespacedName{
		Namespace: ki.imp.GetNamespace(),
		Name:      ki.imp.GetServiceAccountName(),
	}

	for _, secret := range serviceAccount.Secrets {
		if strings.HasPrefix(secret.Name, fmt.Sprintf("%s-token", serviceAccount.Name)) {
			secretName.Name = secret.Name
			break
		}
	}

	var secret corev1.Secret
	err = ki.Client.Get(ctx, secretName, &secret)
	if err != nil {
		return "", err
	}

	var token string
	if data, ok := secret.Data["token"]; ok {
		token = string(data)
	} else {
		return "", fmt.Errorf("the service account secret '%s' does not containt a token", secretName.String())
	}

	// Reuse token for the life of this impersonation to avoid continued lookups
	ki.serviceAccountToken = token

	return ki.serviceAccountToken, nil
}
