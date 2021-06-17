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
*/

package controllers

import (
	"context"

	"github.com/fluxcd/pkg/apis/meta"

	"github.com/go-logr/logr"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1"
)

func (r *KonfigurationReconciler) recordReadiness(ctx context.Context, konfig *konfigurationv1.Konfiguration) {
	if r.MetricsRecorder == nil {
		return
	}
	log := logr.FromContext(ctx)

	objRef, err := reference.GetReference(r.Scheme, konfig)
	if err != nil {
		log.Error(err, "unable to record readiness metric")
		return
	}
	if rc := apimeta.FindStatusCondition(konfig.Status.Conditions, meta.ReadyCondition); rc != nil {
		r.MetricsRecorder.RecordCondition(*objRef, *rc, !konfig.DeletionTimestamp.IsZero())
	} else {
		r.MetricsRecorder.RecordCondition(*objRef, metav1.Condition{
			Type:   meta.ReadyCondition,
			Status: metav1.ConditionUnknown,
		}, !konfig.DeletionTimestamp.IsZero())
	}
}

func (r *KonfigurationReconciler) recordSuspension(ctx context.Context, konfig *konfigurationv1.Konfiguration) {
	if r.MetricsRecorder == nil {
		return
	}
	log := logr.FromContext(ctx)

	objRef, err := reference.GetReference(r.Scheme, konfig)
	if err != nil {
		log.Error(err, "unable to record suspended metric")
		return
	}

	if !konfig.DeletionTimestamp.IsZero() {
		r.MetricsRecorder.RecordSuspend(*objRef, false)
	} else {
		r.MetricsRecorder.RecordSuspend(*objRef, konfig.Spec.Suspend)
	}
}
