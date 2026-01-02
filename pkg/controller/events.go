/*
Copyright 2025 Kube-ZEN Contributors

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

package controller

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	sdkevents "github.com/kube-zen/zen-sdk/pkg/events"
)

// EventRecorder wraps Kubernetes event recorder for JobFlow controller.
// This now uses zen-sdk/pkg/events as the base implementation.
type EventRecorder struct {
	*sdkevents.Recorder
}

// NewEventRecorder creates a new event recorder.
func NewEventRecorder(kubeClient kubernetes.Interface) *EventRecorder {
	return &EventRecorder{
		Recorder: sdkevents.NewRecorder(kubeClient, "zen-flow-controller"),
	}
}

// RecordJobFlowCreated records that a JobFlow was created.
// Events for CRDs may not be supported by all Kubernetes clusters.
// This function logs errors but does not fail if event recording fails.
func (er *EventRecorder) RecordJobFlowCreated(jobFlow *v1alpha1.JobFlow) {
	if er == nil || er.Recorder == nil {
		return
	}
	er.Eventf(
		jobFlow,
		corev1.EventTypeNormal,
		"JobFlowCreated",
		"JobFlow created",
	)
}

// RecordJobFlowStarted records that a JobFlow started execution.
func (er *EventRecorder) RecordJobFlowStarted(jobFlow *v1alpha1.JobFlow) {
	if er == nil || er.Recorder == nil {
		return
	}
	er.Eventf(
		jobFlow,
		corev1.EventTypeNormal,
		"JobFlowStarted",
		"JobFlow execution started",
	)
}

// RecordJobFlowCompleted records that a JobFlow completed successfully.
func (er *EventRecorder) RecordJobFlowCompleted(jobFlow *v1alpha1.JobFlow) {
	if er == nil || er.Recorder == nil {
		return
	}
	er.Eventf(
		jobFlow,
		corev1.EventTypeNormal,
		"JobFlowCompleted",
		"JobFlow completed successfully",
	)
}

// RecordJobFlowFailed records that a JobFlow failed.
func (er *EventRecorder) RecordJobFlowFailed(jobFlow *v1alpha1.JobFlow, reason string) {
	if er == nil || er.Recorder == nil {
		return
	}
	er.Eventf(
		jobFlow,
		corev1.EventTypeWarning,
		"JobFlowFailed",
		"JobFlow failed: %s",
		reason,
	)
}

// RecordStepCreated records that a step's Job was created.
func (er *EventRecorder) RecordStepCreated(jobFlow *v1alpha1.JobFlow, stepName string, job runtime.Object) {
	if er == nil || er.Recorder == nil {
		return
	}
	er.Eventf(
		jobFlow,
		corev1.EventTypeNormal,
		"StepCreated",
		"Created Job for step %s: %s",
		stepName,
		sdkevents.GetResourceName(job),
	)
}

// RecordStepSucceeded records that a step succeeded.
func (er *EventRecorder) RecordStepSucceeded(jobFlow *v1alpha1.JobFlow, stepName string) {
	if er == nil || er.Recorder == nil {
		return
	}
	er.Eventf(
		jobFlow,
		corev1.EventTypeNormal,
		"StepSucceeded",
		"Step %s completed successfully",
		stepName,
	)
}

// RecordStepFailed records that a step failed.
func (er *EventRecorder) RecordStepFailed(jobFlow *v1alpha1.JobFlow, stepName string, err error) {
	if er == nil || er.Recorder == nil {
		return
	}
	er.Eventf(
		jobFlow,
		corev1.EventTypeWarning,
		"StepFailed",
		"Step %s failed: %v",
		stepName,
		err,
	)
}
