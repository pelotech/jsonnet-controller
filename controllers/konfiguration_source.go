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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	securejoin "github.com/cyphar/filepath-securejoin"
	"github.com/fluxcd/pkg/untar"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	"github.com/hashicorp/go-retryablehttp"
	konfigurationv1 "github.com/pelotech/kubecfg-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *KonfigurationReconciler) prepareSource(ctx context.Context, konfig *konfigurationv1.Konfiguration) (revision, path string, clean func(), err error) {
	reqLogger := log.FromContext(ctx)

	// Initially set paths to those defined in spec. If we are running
	// against a source archive, they will be turned into absolute paths.
	// Otherwises they are probably http(s):// paths.
	path = konfig.GetPath()
	revision = path
	clean = func() {}

	// Check if there is a reference to a source. This is a stop-gap solution
	// before full integration with source-controller.
	if sourceRef := konfig.GetSourceRef(); sourceRef != nil {
		var source sourcev1.Source

		source, err = sourceRef.GetSource(ctx, r.Client)
		if err != nil {
			msg := fmt.Sprintf("Source '%s' not found", konfig.Spec.SourceRef.String())
			if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta("", konfigurationv1.ArtifactFailedReason, msg)); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			reqLogger.Error(err, "Failed to fetch source for Konfiguration")
			return
		}

		// Check if the artifact is not ready yet
		if source.GetArtifact() == nil {
			msg := "source is not ready, artifact not found"
			if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta("", konfigurationv1.ArtifactFailedReason, msg)); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			reqLogger.Info(msg)
			err = errors.New(msg)
			return
		}

		artifact := source.GetArtifact()
		revision = artifact.Revision

		// Create a temp directory for the artifact
		var tmpDir string
		tmpDir, err = ioutil.TempDir("", konfig.GetName())
		if err != nil {
			if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(artifact.Revision, sourcev1.StorageOperationFailedReason, err.Error())); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			reqLogger.Error(err, "Could not allocate a temp directory for source artifact")
			return
		}

		// Download and extract the artifact
		if err = r.downloadAndExtractTo(artifact.URL, tmpDir); err != nil {
			if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(artifact.Revision, konfigurationv1.ArtifactFailedReason, err.Error())); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			reqLogger.Error(err, "Failed to download source artifact")
			return
		}

		path, err = securejoin.SecureJoin(tmpDir, path)
		if err != nil {
			if statusErr := konfig.SetNotReady(ctx, r.Client, konfigurationv1.NewStatusMeta(artifact.Revision, konfigurationv1.ArtifactFailedReason, err.Error())); statusErr != nil {
				reqLogger.Error(statusErr, "Failed to update Konfiguration status")
			}
			reqLogger.Error(err, "Failed to format path relative to tmp directory")
		}

		clean = func() { os.RemoveAll(tmpDir) }
	}

	return
}

func (r *KonfigurationReconciler) downloadAndExtractTo(artifactURL, tmpDir string) error {
	if hostname := os.Getenv("SOURCE_CONTROLLER_LOCALHOST"); hostname != "" {
		u, err := url.Parse(artifactURL)
		if err != nil {
			return err
		}
		u.Host = hostname
		artifactURL = u.String()
	}

	req, err := retryablehttp.NewRequest(http.MethodGet, artifactURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create a new request: %w", err)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download artifact, error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download artifact from %s, status: %s", artifactURL, resp.Status)
	}

	if _, err = untar.Untar(resp.Body, tmpDir); err != nil {
		return fmt.Errorf("failed to untar artifact, error: %w", err)
	}

	return nil
}
