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
  - Adaption from fluxcd/kustomize-controller
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/fluxcd/pkg/apis/meta"

	metav1 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/reference"
	"sigs.k8s.io/controller-runtime/pkg/log"

	konfigurationv1 "github.com/pelotech/kubecfg-operator/api/v1"
)

type EventData struct {
	Revision, Severity, Message string
	Metadata                    map[string]string
}

func (r *KonfigurationReconciler) event(ctx context.Context, konfig *konfigurationv1.Konfiguration, data *EventData) {
	log := log.FromContext(ctx)

	log.Info("Sending event", "Event", fmt.Sprintf("%+v", data))
	r.EventRecorder.Event(konfig, "Normal", data.Severity, data.Message)
	objRef, err := reference.GetReference(r.Scheme, konfig)
	if err != nil {
		log.Error(err, "unable to send event")
		return
	}

	if r.ExternalEventRecorder != nil {
		metadata := data.Metadata
		if metadata == nil {
			metadata = map[string]string{}
		}
		if data.Revision != "" {
			metadata["revision"] = data.Revision
		}

		reason := data.Severity
		if c := metav1.FindStatusCondition(konfig.Status.Conditions, meta.ReadyCondition); c != nil {
			reason = c.Reason
		}

		if err := r.ExternalEventRecorder.Eventf(*objRef, metadata, data.Severity, reason, data.Message); err != nil {
			log.Error(err, "unable to send event")
			return
		}
	}
}
