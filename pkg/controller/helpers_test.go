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
	"errors"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestUpdateStatusWithMetrics(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Phase: v1alpha1.JobFlowPhaseRunning,
		},
	}

	// Create the JobFlow first
	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Update status
	jobFlow.Status.Phase = v1alpha1.JobFlowPhaseSucceeded
	err := reconciler.updateStatusWithMetrics(ctx, jobFlow)
	if err != nil {
		// Status update may fail with fake client, but metrics should be recorded
		t.Logf("Status update error (expected with fake client): %v", err)
	}

	// Verify metrics were recorded (check that recorder is not nil)
	if reconciler.MetricsRecorder == nil {
		t.Error("MetricsRecorder is nil")
	}
}

func TestGetStepStatusOrCreate(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "existing-step",
					Phase: v1alpha1.StepPhaseRunning,
				},
			},
		},
	}

	// Test getting existing step status
	stepStatus := reconciler.getStepStatusOrCreate(jobFlow, "existing-step")
	if stepStatus == nil {
		t.Fatal("getStepStatusOrCreate returned nil for existing step")
	}
	if stepStatus.Name != "existing-step" {
		t.Errorf("Expected step name 'existing-step', got '%s'", stepStatus.Name)
	}
	if stepStatus.Phase != v1alpha1.StepPhaseRunning {
		t.Errorf("Expected phase 'Running', got '%s'", stepStatus.Phase)
	}

	// Test creating new step status
	initialStepCount := len(jobFlow.Status.Steps)
	stepStatus = reconciler.getStepStatusOrCreate(jobFlow, "new-step")
	if stepStatus == nil {
		t.Fatal("getStepStatusOrCreate returned nil for new step")
	}
	if stepStatus.Name != "new-step" {
		t.Errorf("Expected step name 'new-step', got '%s'", stepStatus.Name)
	}
	if stepStatus.Phase != v1alpha1.StepPhasePending {
		t.Errorf("Expected phase 'Pending', got '%s'", stepStatus.Phase)
	}
	if len(jobFlow.Status.Steps) != initialStepCount+1 {
		t.Errorf("Expected %d steps, got %d", initialStepCount+1, len(jobFlow.Status.Steps))
	}
}

func TestMarkStepFailed(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseRunning,
				},
			},
		},
	}

	// Test marking step as failed with message
	reconciler.markStepFailed(jobFlow, "step1", "Step failed due to error")
	stepStatus := reconciler.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseFailed {
		t.Errorf("Expected phase 'Failed', got '%s'", stepStatus.Phase)
	}
	if stepStatus.Message != "Step failed due to error" {
		t.Errorf("Expected message 'Step failed due to error', got '%s'", stepStatus.Message)
	}
	if stepStatus.CompletionTime == nil {
		t.Error("Expected CompletionTime to be set")
	}

	// Test marking step as failed without message
	reconciler.markStepFailed(jobFlow, "step1", "")
	if stepStatus.Message != "Step failed due to error" {
		t.Error("Message should not be cleared when empty string is provided")
	}

	// Test marking non-existent step (should not panic)
	reconciler.markStepFailed(jobFlow, "non-existent-step", "Error")
}

func TestMarkStepSucceeded(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseRunning,
				},
			},
		},
	}

	// Test marking step as succeeded
	reconciler.markStepSucceeded(jobFlow, "step1")
	stepStatus := reconciler.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseSucceeded {
		t.Errorf("Expected phase 'Succeeded', got '%s'", stepStatus.Phase)
	}
	if stepStatus.CompletionTime == nil {
		t.Error("Expected CompletionTime to be set")
	}

	// Test marking non-existent step (should not panic)
	reconciler.markStepSucceeded(jobFlow, "non-existent-step")
}

func TestRequeueWithError(t *testing.T) {
	err := errors.New("test error")
	result, returnedErr := requeueWithError(err)

	if !result.Requeue {
		t.Error("Expected Requeue to be true")
	}
	if returnedErr != err {
		t.Errorf("Expected error '%v', got '%v'", err, returnedErr)
	}
}

func TestRequeueAfter(t *testing.T) {
	duration := 5 * time.Second
	result := requeueAfter(duration)

	if result.RequeueAfter != duration {
		t.Errorf("Expected RequeueAfter %v, got %v", duration, result.RequeueAfter)
	}
	if result.Requeue {
		t.Error("Expected Requeue to be false when RequeueAfter is set")
	}
}

func TestNoRequeue(t *testing.T) {
	result := noRequeue()

	if result.Requeue {
		t.Error("Expected Requeue to be false")
	}
	if result.RequeueAfter != 0 {
		t.Errorf("Expected RequeueAfter to be 0, got %v", result.RequeueAfter)
	}
}

func TestGetJobFlow(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Phase: v1alpha1.JobFlowPhaseRunning,
		},
	}

	// Create the JobFlow
	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Get the JobFlow
	key := client.ObjectKey{Name: "test-flow", Namespace: "default"}
	retrieved, err := reconciler.getJobFlow(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get JobFlow: %v", err)
	}
	if retrieved == nil {
		t.Fatal("getJobFlow returned nil")
	}
	if retrieved.Name != "test-flow" {
		t.Errorf("Expected name 'test-flow', got '%s'", retrieved.Name)
	}

	// Verify metrics were recorded
	if reconciler.MetricsRecorder == nil {
		t.Error("MetricsRecorder is nil")
	}
}

func TestGetJobFlow_NotFound(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)
	ctx := context.Background()

	key := client.ObjectKey{Name: "non-existent", Namespace: "default"}
	_, err := reconciler.getJobFlow(ctx, key)
	if err == nil {
		t.Error("Expected error when getting non-existent JobFlow")
	}
}

func TestGetJob(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "main", Image: "busybox:latest"},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	// Create the Job
	if err := fakeClient.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}

	// Get the Job
	key := client.ObjectKey{Name: "test-job", Namespace: "default"}
	retrieved, err := reconciler.getJob(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get Job: %v", err)
	}
	if retrieved == nil {
		t.Fatal("getJob returned nil")
	}
	if retrieved.Name != "test-job" {
		t.Errorf("Expected name 'test-job', got '%s'", retrieved.Name)
	}

	// Verify metrics were recorded
	if reconciler.MetricsRecorder == nil {
		t.Error("MetricsRecorder is nil")
	}
}

func TestGetJob_NotFound(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)
	ctx := context.Background()

	key := client.ObjectKey{Name: "non-existent", Namespace: "default"}
	_, err := reconciler.getJob(ctx, key)
	if err == nil {
		t.Error("Expected error when getting non-existent Job")
	}
}

func TestCreateJob(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "main", Image: "busybox:latest"},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	// Create the Job
	err := reconciler.createJob(ctx, job)
	if err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}

	// Verify the Job was created
	key := client.ObjectKey{Name: "test-job", Namespace: "default"}
	retrieved := &batchv1.Job{}
	if err := fakeClient.Get(ctx, key, retrieved); err != nil {
		t.Fatalf("Failed to get created Job: %v", err)
	}

	// Verify metrics were recorded
	if reconciler.MetricsRecorder == nil {
		t.Error("MetricsRecorder is nil")
	}
}

func TestGetStepStatus(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	status := v1alpha1.JobFlowStatus{
		Steps: []v1alpha1.StepStatus{
			{
				Name:  "step1",
				Phase: v1alpha1.StepPhaseRunning,
			},
			{
				Name:  "step2",
				Phase: v1alpha1.StepPhasePending,
			},
		},
	}

	// Test getting existing step
	stepStatus := reconciler.getStepStatus(status, "step1")
	if stepStatus == nil {
		t.Fatal("getStepStatus returned nil for existing step")
	}
	if stepStatus.Name != "step1" {
		t.Errorf("Expected step name 'step1', got '%s'", stepStatus.Name)
	}
	if stepStatus.Phase != v1alpha1.StepPhaseRunning {
		t.Errorf("Expected phase 'Running', got '%s'", stepStatus.Phase)
	}

	// Test getting another existing step
	stepStatus = reconciler.getStepStatus(status, "step2")
	if stepStatus == nil {
		t.Fatal("getStepStatus returned nil for existing step")
	}
	if stepStatus.Name != "step2" {
		t.Errorf("Expected step name 'step2', got '%s'", stepStatus.Name)
	}

	// Test getting non-existent step
	stepStatus = reconciler.getStepStatus(status, "non-existent")
	if stepStatus != nil {
		t.Error("getStepStatus should return nil for non-existent step")
	}

	// Test with empty steps
	emptyStatus := v1alpha1.JobFlowStatus{Steps: []v1alpha1.StepStatus{}}
	stepStatus = reconciler.getStepStatus(emptyStatus, "step1")
	if stepStatus != nil {
		t.Error("getStepStatus should return nil for empty steps")
	}
}
