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

Copyright 2021 Avi Zimmerman - Apache License, Version 2.0.
 - Methods adapted from fluxcd/kustomize-controller to work on pointer receivers
*/

package v1

import (
	"github.com/fluxcd/pkg/apis/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const MaxConditionMessageLength int = 20000

// GetStatusConditions returns the status conditions for this resource.
func (k *Konfiguration) GetStatusConditions() *[]metav1.Condition {
	return &k.Status.Conditions
}

// SetProgressing resets the conditions of this Kustomization to a single
// ReadyCondition with status ConditionUnknown.
func (k *Konfiguration) SetProgressing() {
	meta.SetResourceCondition(k, meta.ReadyCondition, metav1.ConditionUnknown, meta.ProgressingReason, "reconciliation in progress")
}

// func (k *Konfiguration) SetHealthiness(status metav1.ConditionStatus, reason, message string) {
// 	switch len(k.Spec.HealthChecks) {
// 	case 0:
// 		apimeta.RemoveStatusCondition(k.GetStatusConditions(), HealthyCondition)
// 	default:
// 		meta.SetResourceCondition(k, HealthyCondition, status, reason, trimString(message, MaxConditionMessageLength))
// 	}
// }

// SetReadiness sets the ReadyCondition, ObservedGeneration, and LastAttemptedRevision,
// on the Konfiguration.
func (k *Konfiguration) SetReadiness(status metav1.ConditionStatus, revision, reason, message string) {
	meta.SetResourceCondition(k, meta.ReadyCondition, status, reason, trimString(message, MaxConditionMessageLength))
	k.Status.ObservedGeneration = k.Generation
	if revision != "" {
		k.Status.LastAttemptedRevision = revision
	}
}

// SetNotReady registers a failed apply attempt of this Konfiguration.
func (k *Konfiguration) SetNotReady(revision, reason, message string) {
	k.SetReadiness(metav1.ConditionFalse, revision, reason, message)
}

// SetNotReadySnapshot registers a failed apply attempt of this Konfiguration,
// including a Snapshot.
func (k *Konfiguration) SetNotReadySnapshot(snapshot *Snapshot, revision, reason, message string) {
	k.SetReadiness(metav1.ConditionFalse, revision, reason, message)
	// k.SetHealthiness(metav1.ConditionFalse, reason, reason)
	k.Status.Snapshot = snapshot
	k.Status.LastAttemptedRevision = revision
}

// SetReady registers a successful apply attempt of this Konfiguration.
func (k *Konfiguration) SetReady(snapshot *Snapshot, revision, reason, message string) {
	k.SetReadiness(metav1.ConditionTrue, revision, reason, message)
	// k.SetHealthiness(metav1.ConditionTrue, reason, reason)
	k.Status.Snapshot = snapshot
	k.Status.LastAppliedRevision = revision
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
