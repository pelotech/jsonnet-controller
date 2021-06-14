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
	- Exposes method for returning kubecfg arguments
*/

package controllers

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/cli-utils/pkg/kstatus/polling"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1"
)

type KonfigurationImpersonation struct {
	client.Client

	workdir       string
	konfiguration *konfigurationv1.Konfiguration
	statusPoller  *polling.StatusPoller

	// cached assets
	serviceAccountToken string
	kubeconfigPath      string
}

func NewKonfigurationImpersonation(
	konfiguration *konfigurationv1.Konfiguration,
	kubeClient client.Client,
	statusPoller *polling.StatusPoller,
	workdir string) *KonfigurationImpersonation {
	return &KonfigurationImpersonation{
		workdir:       workdir,
		konfiguration: konfiguration,
		statusPoller:  statusPoller,
		Client:        kubeClient,
	}
}

func (ki *KonfigurationImpersonation) GetServiceAccountToken(ctx context.Context) (string, error) {
	// Return an already retrieved serviceAccountToken
	if ki.serviceAccountToken != "" {
		return ki.serviceAccountToken, nil
	}

	namespacedName := types.NamespacedName{
		Namespace: ki.konfiguration.Namespace,
		Name:      ki.konfiguration.Spec.ServiceAccountName,
	}

	var serviceAccount corev1.ServiceAccount
	err := ki.Client.Get(ctx, namespacedName, &serviceAccount)
	if err != nil {
		return "", err
	}

	secretName := types.NamespacedName{
		Namespace: ki.konfiguration.Namespace,
		Name:      ki.konfiguration.Spec.ServiceAccountName,
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

// GetClient creates a controller-runtime client for talking to a Kubernetes API server.
// If KubeConfig is set, will use the kubeconfig bytes from the Kubernetes secret.
// If ServiceAccountName is set, will use the cluster provided kubeconfig impersonating the SA.
// If --kubeconfig is set, will use the kubeconfig file at that location.
// Otherwise will assume running in cluster and use the cluster provided kubeconfig.
func (ki *KonfigurationImpersonation) GetClient(ctx context.Context) (client.Client, *polling.StatusPoller, error) {
	if ki.konfiguration.Spec.KubeConfig == nil {
		if ki.konfiguration.Spec.ServiceAccountName != "" {
			return ki.clientForServiceAccount(ctx)
		}

		return ki.Client, ki.statusPoller, nil
	}
	return ki.clientForKubeConfig(ctx)
}

// GetKubecfgArgs will retrieve/write any necessary assets and return a slice of arguments
// to pass to kubecfg invocations.
func (ki *KonfigurationImpersonation) GetKubecfgArgs(ctx context.Context) ([]string, error) {
	if ki.konfiguration.Spec.KubeConfig == nil {
		if ki.konfiguration.Spec.ServiceAccountName != "" {
			token, err := ki.GetServiceAccountToken(ctx)
			if err != nil {
				return nil, err
			}
			return []string{fmt.Sprintf("--token=%s", token)}, nil
		}

		return []string{}, nil
	}

	kubeconfig, err := ki.WriteKubeConfig(ctx)
	if err != nil {
		return nil, err
	}

	return []string{fmt.Sprintf("--kubeconfig=%s", kubeconfig)}, nil
}

func (ki *KonfigurationImpersonation) clientForServiceAccount(ctx context.Context) (client.Client, *polling.StatusPoller, error) {
	token, err := ki.GetServiceAccountToken(ctx)
	if err != nil {
		return nil, nil, err
	}
	restConfig, err := config.GetConfig()
	if err != nil {
		return nil, nil, err
	}
	restConfig.BearerToken = token
	restConfig.BearerTokenFile = "" // Clear, as it overrides BearerToken

	restMapper, err := apiutil.NewDynamicRESTMapper(restConfig)
	if err != nil {
		return nil, nil, err
	}

	client, err := client.New(restConfig, client.Options{Mapper: restMapper})
	if err != nil {
		return nil, nil, err
	}

	statusPoller := polling.NewStatusPoller(client, restMapper)
	return client, statusPoller, err

}

func (ki *KonfigurationImpersonation) kubeconfigSecretName() types.NamespacedName {
	return types.NamespacedName{
		Namespace: ki.konfiguration.GetNamespace(),
		Name:      ki.konfiguration.Spec.KubeConfig.SecretRef.Name,
	}

}

func (ki *KonfigurationImpersonation) clientForKubeConfig(ctx context.Context) (client.Client, *polling.StatusPoller, error) {
	kubeConfigBytes, err := ki.getKubeConfig(ctx)
	if err != nil {
		return nil, nil, err
	}

	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeConfigBytes)
	if err != nil {
		return nil, nil, err
	}

	restMapper, err := apiutil.NewDynamicRESTMapper(restConfig)
	if err != nil {
		return nil, nil, err
	}

	client, err := client.New(restConfig, client.Options{Mapper: restMapper})
	if err != nil {
		return nil, nil, err
	}

	statusPoller := polling.NewStatusPoller(client, restMapper)

	return client, statusPoller, err
}

func (ki *KonfigurationImpersonation) WriteKubeConfig(ctx context.Context) (string, error) {
	// See if we already wrote a kubeconfig and it still exists
	if ki.kubeconfigPath != "" {
		_, err := os.Stat(ki.kubeconfigPath)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		} else if err == nil {
			return ki.kubeconfigPath, nil
		}
	}

	sname := ki.kubeconfigSecretName()

	kubeConfig, err := ki.getKubeConfig(ctx)
	if err != nil {
		return "", err
	}

	f, err := ioutil.TempFile(ki.workdir, "kubeconfig")
	if err != nil {
		return "", fmt.Errorf("unable to write KubeConfig secret '%s' to storage: %w", sname.String(), err)
	}
	defer f.Close()

	if _, err := f.Write(kubeConfig); err != nil {
		return "", fmt.Errorf("unable to write KubeConfig secret '%s' to storage: %w", sname.String(), err)
	}

	// Set locally so file will be reused until the workdir is destroyed or
	// the impersonation is thrown away.
	ki.kubeconfigPath = f.Name()

	return ki.kubeconfigPath, nil
}

func (ki *KonfigurationImpersonation) getKubeConfig(ctx context.Context) ([]byte, error) {
	var secret corev1.Secret
	sname := ki.kubeconfigSecretName()
	if err := ki.Get(ctx, ki.kubeconfigSecretName(), &secret); err != nil {
		return nil, fmt.Errorf("unable to read KubeConfig secret '%s' error: %w", sname.String(), err)
	}

	kubeConfig, ok := secret.Data["value"]
	if !ok {
		return nil, fmt.Errorf("KubeConfig secret '%s' doesn't contain a 'value' key ", sname.String())
	}

	return kubeConfig, nil
}
