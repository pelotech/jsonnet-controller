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
 - Methods adapted from fluxcd/kustomize-controller
*/

package v1

import (
	"context"

	"github.com/fluxcd/pkg/apis/meta"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const MaxConditionMessageLength int = 20000

type StatusMeta struct {
	Revision, Reason, Message string
}

func NewStatusMeta(revision, reason, message string) *StatusMeta {
	return &StatusMeta{Revision: revision, Reason: reason, Message: message}
}

// GetStatusConditions returns the status conditions for this resource.
func (k *Konfiguration) GetStatusConditions() *[]metav1.Condition {
	return &k.Status.Conditions
}

// SetProgressing resets the conditions of this Kustomization to a single
// ReadyCondition with status ConditionUnknown.
func (k *Konfiguration) SetProgressing(ctx context.Context, cl client.Client) error {
	meta.SetResourceCondition(k, meta.ReadyCondition, metav1.ConditionUnknown, meta.ProgressingReason, "reconciliation in progress")
	return k.patchStatus(ctx, cl, k.Status)
}

func (k *Konfiguration) SetHealthiness(ctx context.Context, cl client.Client, status metav1.ConditionStatus, statusMeta *StatusMeta) error {
	switch len(k.Spec.HealthChecks) {
	case 0:
		apimeta.RemoveStatusCondition(k.GetStatusConditions(), HealthyCondition)
	default:
		meta.SetResourceCondition(k, HealthyCondition, status, statusMeta.Reason, trimString(statusMeta.Message, MaxConditionMessageLength))
	}
	return k.patchStatus(ctx, cl, k.Status)
}

// SetReadiness sets the ReadyCondition, ObservedGeneration, and LastAttemptedRevision,
// on the Konfiguration.
func (k *Konfiguration) SetReadiness(ctx context.Context, cl client.Client, status metav1.ConditionStatus, statusMeta *StatusMeta) error {
	meta.SetResourceCondition(k, meta.ReadyCondition, status, statusMeta.Reason, trimString(statusMeta.Message, MaxConditionMessageLength))
	k.Status.ObservedGeneration = k.Generation
	if statusMeta.Revision != "" {
		k.Status.LastAttemptedRevision = statusMeta.Revision
	}
	return k.patchStatus(ctx, cl, k.Status)
}

// SetNotReady registers a failed apply attempt of this Konfiguration.
func (k *Konfiguration) SetNotReady(ctx context.Context, cl client.Client, meta *StatusMeta) error {
	return k.SetReadiness(ctx, cl, metav1.ConditionFalse, meta)
}

// SetNotReadySnapshot registers a failed apply attempt of this Konfiguration,
// including a Snapshot.
func (k *Konfiguration) SetNotReadySnapshot(ctx context.Context, cl client.Client, snapshot *Snapshot, meta *StatusMeta) error {
	k.Status.Snapshot = snapshot
	k.Status.LastAttemptedRevision = meta.Revision
	if err := k.SetHealthiness(ctx, cl, metav1.ConditionFalse, meta); err != nil {
		return err
	}
	return k.SetReadiness(ctx, cl, metav1.ConditionFalse, meta)
}

// SetReady registers a successful apply attempt of this Konfiguration.
func (k *Konfiguration) SetReady(ctx context.Context, cl client.Client, snapshot *Snapshot, meta *StatusMeta) error {
	k.Status.Snapshot = snapshot
	k.Status.LastAppliedRevision = meta.Revision
	if err := k.SetHealthiness(ctx, cl, metav1.ConditionTrue, meta); err != nil {
		return err
	}
	return k.SetReadiness(ctx, cl, metav1.ConditionTrue, meta)
}

func (k *Konfiguration) patchStatus(ctx context.Context, cl client.Client, newStatus KonfigurationStatus) error {
	var konfig Konfiguration
	if err := cl.Get(ctx, k.GetNamespacedName(), &konfig); err != nil {
		return err
	}

	patch := client.MergeFrom(konfig.DeepCopy())
	konfig.Status = newStatus

	return cl.Status().Patch(ctx, &konfig, patch)
}

func trimString(str string, limit int) string {
	result := str
	chars := 0
	for i := range str {
		if chars >= limit {
			result = str[:i] + "..."
			break
		}
		chars++
	}
	return result
}
