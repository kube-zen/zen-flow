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
	"context"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

// updateStatusWithMetrics updates JobFlow status and records metrics
func (r *JobFlowReconciler) updateStatusWithMetrics(ctx context.Context, jobFlow *v1alpha1.JobFlow) error {
	statusStart := time.Now()
	err := r.Status().Update(ctx, jobFlow)
	statusDuration := time.Since(statusStart)
	r.MetricsRecorder.RecordStatusUpdateDuration(statusDuration.Seconds())
	r.MetricsRecorder.RecordAPIServerCall("update", "jobflow")
	return err
}

// getStepStatusOrCreate gets step status or creates it if it doesn't exist
func (r *JobFlowReconciler) getStepStatusOrCreate(jobFlow *v1alpha1.JobFlow, stepName string) *v1alpha1.StepStatus {
	stepStatus := r.getStepStatus(jobFlow.Status, stepName)
	if stepStatus == nil {
		// Create new step status
		stepStatus = &v1alpha1.StepStatus{
			Name:  stepName,
			Phase: v1alpha1.StepPhasePending,
		}
		jobFlow.Status.Steps = append(jobFlow.Status.Steps, *stepStatus)
		// Get reference to the newly added status
		stepStatus = &jobFlow.Status.Steps[len(jobFlow.Status.Steps)-1]
	}
	return stepStatus
}

// markStepFailed marks a step as failed with optional message
func (r *JobFlowReconciler) markStepFailed(jobFlow *v1alpha1.JobFlow, stepName string, message string) {
	stepStatus := r.getStepStatus(jobFlow.Status, stepName)
	if stepStatus != nil {
		stepStatus.Phase = v1alpha1.StepPhaseFailed
		now := metav1.Now()
		stepStatus.CompletionTime = &now
		if message != "" {
			stepStatus.Message = message
		}
	}
}

// markStepSucceeded marks a step as succeeded
func (r *JobFlowReconciler) markStepSucceeded(jobFlow *v1alpha1.JobFlow, stepName string) {
	stepStatus := r.getStepStatus(jobFlow.Status, stepName)
	if stepStatus != nil {
		stepStatus.Phase = v1alpha1.StepPhaseSucceeded
		now := metav1.Now()
		stepStatus.CompletionTime = &now
	}
}

// requeueWithError returns a requeue result with error
func requeueWithError(err error) (ctrl.Result, error) {
	return ctrl.Result{Requeue: true}, err
}

// requeueAfter returns a requeue result after specified duration
func requeueAfter(duration time.Duration) ctrl.Result {
	return ctrl.Result{RequeueAfter: duration}
}

// noRequeue returns a no-requeue result
func noRequeue() ctrl.Result {
	return ctrl.Result{}
}

// getJobFlow gets a JobFlow by namespaced name and records metrics
func (r *JobFlowReconciler) getJobFlow(ctx context.Context, name client.ObjectKey) (*v1alpha1.JobFlow, error) {
	jobFlow := &v1alpha1.JobFlow{}
	err := r.Get(ctx, name, jobFlow)
	r.MetricsRecorder.RecordAPIServerCall("get", "jobflow")
	return jobFlow, err
}

// getJob gets a Job by namespaced name and records metrics
func (r *JobFlowReconciler) getJob(ctx context.Context, name client.ObjectKey) (*batchv1.Job, error) {
	job := &batchv1.Job{}
	err := r.Get(ctx, name, job)
	r.MetricsRecorder.RecordAPIServerCall("get", "job")
	return job, err
}

// createJob creates a Job and records metrics
func (r *JobFlowReconciler) createJob(ctx context.Context, job *batchv1.Job) error {
	err := r.Create(ctx, job)
	r.MetricsRecorder.RecordAPIServerCall("create", "job")
	return err
}

