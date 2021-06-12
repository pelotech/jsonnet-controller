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

package controllers

import (
	"context"
	"io/ioutil"

	"github.com/fluxcd/pkg/apis/meta"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appsv1 "github.com/pelotech/kubecfg-operator/api/v1"
)

func (r *KonfigurationReconciler) getKubeConfig(ctx context.Context, konfig *appsv1.Konfiguration) (string, error) {
	reqLogger := log.FromContext(ctx)

	var kubeconfig string
	// Write a kubeconfig file if necessary
	if config := konfig.GetKubeConfig(); config != nil {
		nn := types.NamespacedName{
			Name:      config.SecretRef.Name,
			Namespace: konfig.GetNamespace(),
		}
		reqLogger.Info("Fetching KubeConfig from secret", "secret", nn.String())
		var secret corev1.Secret
		if err := r.Get(ctx, nn, &secret); err != nil {
			if statusErr := konfig.SetNotReady(ctx, r.Client, appsv1.NewStatusMeta("", meta.ReconciliationFailedReason, err.Error())); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			reqLogger.Info("Could not retrieve kubeconfig")
			return "", err
		}
		tempFile, err := ioutil.TempFile("", "")
		if err != nil {
			if statusErr := konfig.SetNotReady(ctx, r.Client, appsv1.NewStatusMeta("", sourcev1.StorageOperationFailedReason, err.Error())); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			reqLogger.Info("Could not allocate a file for a kubeconfig")
			return "", err
		}
		defer tempFile.Close()

		kubeconfigData, ok := secret.Data["value"]
		if !ok {
			msg := "Kubeconfig secret does not contain a 'value' key"
			if statusErr := konfig.SetNotReady(ctx, r.Client, appsv1.NewStatusMeta("", meta.ReconciliationFailedReason, msg)); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			reqLogger.Info(msg)
		}

		if _, err := tempFile.Write(kubeconfigData); err != nil {
			if statusErr := konfig.SetNotReady(ctx, r.Client, appsv1.NewStatusMeta("", sourcev1.StorageOperationFailedReason, err.Error())); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			reqLogger.Info("Could not write kubeconfig contents to file")
			return "", err
		}

		kubeconfig = tempFile.Name()
	}

	return kubeconfig, nil
}
