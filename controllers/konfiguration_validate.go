package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1"
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
			returnError(w, http.StatusBadRequest, err.Error())
			return
		}

		var lastErr error
		for {
			select {
			case <-ctx.Done():
				returnError(w, http.StatusRequestTimeout, ctx.Err().Error()+", last error: "+lastErr.Error())
				return
			default:
				if lastErr != nil {
					time.Sleep(time.Second)
				}
				_, path, clean, err := r.prepareSource(ctx, &konfig)
				if err != nil {
					if client.IgnoreNotFound(err) == nil {
						returnError(w, http.StatusInternalServerError, err.Error())
						return
					}
					lastErr = err
					continue
				}
				defer clean()
				dirPath, err := ioutil.TempDir("", konfig.GetName())
				if err != nil {
					returnError(w, http.StatusInternalServerError, err.Error())
					return
				}
				defer os.RemoveAll(dirPath)

				impersonation := NewKonfigurationImpersonation(&konfig, r.Client, r.StatusPoller, filepath.Dir(path))
				kubeClient, _, err := impersonation.GetClient(ctx)
				if err != nil {
					returnError(w, http.StatusInternalServerError, err.Error())
					return
				}

				builder, err := jsonnet.NewBuilder(&konfig, dirPath, r.jsonnetCache)
				if err != nil {
					returnError(w, http.StatusInternalServerError, err.Error())
					return
				}

				buildOutput, err := builder.Build(ctx, kubeClient.RESTMapper(), path)
				if err != nil {
					returnError(w, http.StatusInternalServerError, err.Error())
					return
				}

				stream, err := buildOutput.YAMLStream()
				if err != nil {
					returnError(w, http.StatusInternalServerError, err.Error())
					return
				}

				if _, err := w.Write(append(stream, []byte("\n")...)); err != nil {
					fmt.Println("Error writing yaml stream to response:", err)
				}

				return
			}
		}
	})
}

func returnError(w http.ResponseWriter, statusCode int, message string) {
	out, err := json.MarshalIndent(map[string]string{
		"error": message,
	}, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling json error:", err)
		return
	}
	w.WriteHeader(statusCode)
	if _, err := w.Write(append(out, []byte("\n")...)); err != nil {
		fmt.Println("Error writing json to response:", err)
	}
}
