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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1beta1"
	"github.com/pelotech/jsonnet-controller/pkg/impersonation"
	"github.com/pelotech/jsonnet-controller/pkg/jsonnet"
)

func (r *KonfigurationReconciler) DryRunFunc() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), r.dryRunTimeout)
		defer cancel()

		defer req.Body.Close()

		reader := yaml.NewYAMLOrJSONDecoder(req.Body, 2048)
		var konfig konfigurationv1.Konfiguration
		err := reader.Decode(&konfig)
		if err != nil {
			r.returnError(w, http.StatusBadRequest, err.Error())
			return
		}

		r.HTTPLog.Info(fmt.Sprintf("Dry run request for %s/%s", konfig.GetNamespace(), konfig.GetName()))

		var lastErr error
		for {
			select {
			case <-ctx.Done():
				r.returnError(w, http.StatusRequestTimeout, ctx.Err().Error()+", last error: "+lastErr.Error())
				return
			default:
				if lastErr != nil {
					time.Sleep(time.Second)
				}
				_, path, clean, err := r.prepareSource(ctx, &konfig)
				if err != nil {
					if client.IgnoreNotFound(err) == nil {
						r.returnError(w, http.StatusInternalServerError, err.Error())
						return
					}
					lastErr = err
					continue
				}
				defer clean()
				dirPath, err := ioutil.TempDir("", konfig.GetName())
				if err != nil {
					r.returnError(w, http.StatusInternalServerError, err.Error())
					return
				}
				defer os.RemoveAll(dirPath)

				imp := impersonation.NewImpersonation(&konfig, r.Client)
				kubeClient, err := imp.GetClient(ctx)

				if err != nil {
					r.returnError(w, http.StatusInternalServerError, err.Error())
					return
				}

				builder, err := jsonnet.NewBuilder(&konfig, dirPath, r.jsonnetCache)
				if err != nil {
					r.returnError(w, http.StatusInternalServerError, err.Error())
					return
				}

				buildOutput, err := builder.Build(ctx, kubeClient.RESTMapper(), path)
				if err != nil {
					r.returnError(w, http.StatusInternalServerError, err.Error())
					return
				}

				stream, err := buildOutput.YAMLStream()
				if err != nil {
					r.returnError(w, http.StatusInternalServerError, err.Error())
					return
				}

				if _, err := w.Write(append(stream, []byte("\n")...)); err != nil {
					r.HTTPLog.Error(err, "Error writing yaml stream to response")
				}

				return
			}
		}
	})
}

func (r *KonfigurationReconciler) returnError(w http.ResponseWriter, statusCode int, message string) {
	r.HTTPLog.Info(fmt.Sprintf("Konfiguration dry-run error: %s", message))
	out, err := json.MarshalIndent(map[string]string{
		"error": message,
	}, "", "  ")
	if err != nil {
		r.HTTPLog.Error(err, "Error marshalling json return")
		return
	}
	w.WriteHeader(statusCode)
	if _, err := w.Write(append(out, []byte("\n")...)); err != nil {
		r.HTTPLog.Error(err, "Error writing response")
	}
}
