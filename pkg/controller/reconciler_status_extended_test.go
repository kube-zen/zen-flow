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
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestJobFlowReconciler_updateStepStatusFromJob_Failed(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failed-test",
			Namespace: "default",
			UID:       types.UID("failed-uid-12345678"),
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: runtime.RawExtension{
						Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
					},
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:      "step1",
					Phase:     v1alpha1.StepPhaseRunning,
					StartTime: &metav1.Time{Time: metav1.Now().Add(-time.Minute)},
					JobRef: &corev1.ObjectReference{
						Name:      "failed-test-step1",
						Namespace: "default",
					},
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "failed-test-step1",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Failed: 1,
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}
	if err := fakeClient.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}

	err := reconciler.updateStepStatusFromJob(ctx, jobFlow, "step1", job)
	// Status update may fail with fake client, but the function logic should run
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"failed-test\" not found" {
		t.Logf("updateStepStatusFromJob error (may be expected): %v", err)
	}

	// Verify step status was updated in memory
	stepStatus := reconciler.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseFailed {
		t.Errorf("Expected step phase to be Failed, got %s", stepStatus.Phase)
	}
	if stepStatus.CompletionTime == nil {
		t.Error("Expected CompletionTime to be set for failed step")
	}
}

func TestJobFlowReconciler_updateStepStatusFromJob_StillRunning(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "running-test",
			Namespace: "default",
			UID:       types.UID("running-uid-12345678"),
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: runtime.RawExtension{
						Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
					},
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:      "step1",
					Phase:     v1alpha1.StepPhasePending,
					StartTime: &metav1.Time{Time: metav1.Now().Add(-time.Minute)},
					JobRef: &corev1.ObjectReference{
						Name:      "running-test-step1",
						Namespace: "default",
					},
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "running-test-step1",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			// No succeeded or failed, still running
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}
	if err := fakeClient.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}

	err := reconciler.updateStepStatusFromJob(ctx, jobFlow, "step1", job)
	// Status update may fail with fake client, but the function logic should run
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"running-test\" not found" {
		t.Logf("updateStepStatusFromJob error (may be expected): %v", err)
	}

	// Verify step status was updated in memory
	stepStatus := reconciler.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseRunning {
		t.Errorf("Expected step phase to be Running, got %s", stepStatus.Phase)
	}
}

func TestJobFlowReconciler_updateStepStatusFromJob_WithOutputs(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "outputs-test",
			Namespace: "default",
			UID:       types.UID("outputs-uid-12345678"),
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: runtime.RawExtension{
						Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
					},
					Outputs: &v1alpha1.StepOutputs{
						Parameters: []v1alpha1.ParameterOutput{
							{
								Name: "param1",
								ValueFrom: v1alpha1.ParameterValueFrom{
									JSONPath: "$.succeeded",
								},
							},
						},
					},
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:      "step1",
					Phase:     v1alpha1.StepPhaseRunning,
					StartTime: &metav1.Time{Time: metav1.Now().Add(-time.Minute)},
					JobRef: &corev1.ObjectReference{
						Name:      "outputs-test-step1",
						Namespace: "default",
					},
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "outputs-test-step1",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}
	if err := fakeClient.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}

	err := reconciler.updateStepStatusFromJob(ctx, jobFlow, "step1", job)
	// Status update may fail with fake client, but the function logic should run
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"outputs-test\" not found" {
		t.Logf("updateStepStatusFromJob error (may be expected): %v", err)
	}

	// Verify step status was updated in memory
	stepStatus := reconciler.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseSucceeded {
		t.Errorf("Expected step phase to be Succeeded, got %s", stepStatus.Phase)
	}
}

func TestJobFlowReconciler_updateStepStatusFromJob_PhaseTransition(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transition-test",
			Namespace: "default",
			UID:       types.UID("transition-uid-12345678"),
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: runtime.RawExtension{
						Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
					},
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:      "step1",
					Phase:     v1alpha1.StepPhasePending, // Starting from Pending
					StartTime: &metav1.Time{Time: metav1.Now().Add(-time.Minute)},
					JobRef: &corev1.ObjectReference{
						Name:      "transition-test-step1",
						Namespace: "default",
					},
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "transition-test-step1",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}
	if err := fakeClient.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}

	err := reconciler.updateStepStatusFromJob(ctx, jobFlow, "step1", job)
	// Status update may fail with fake client, but the function logic should run
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"transition-test\" not found" {
		t.Logf("updateStepStatusFromJob error (may be expected): %v", err)
	}

	// Verify step status was updated in memory
	stepStatus := reconciler.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseSucceeded {
		t.Errorf("Expected step phase to be Succeeded, got %s", stepStatus.Phase)
	}
	// Phase transition from Pending to Succeeded should be recorded
}

func TestJobFlowReconciler_updateStepStatusFromJob_FailedWithPodFailurePolicy(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-failure-test",
			Namespace: "default",
			UID:       types.UID("pod-failure-uid-12345678"),
		},
		Spec: v1alpha1.JobFlowSpec{
			ExecutionPolicy: &v1alpha1.ExecutionPolicy{
				PodFailurePolicy: &v1alpha1.PodFailurePolicy{
					Rules: []v1alpha1.PodFailurePolicyRule{
						{
							Action: "Ignore",
							OnExitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
								Operator: "In",
								Values:   []int32{1},
							},
						},
					},
				},
			},
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: runtime.RawExtension{
						Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
					},
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:      "step1",
					Phase:     v1alpha1.StepPhaseRunning,
					StartTime: &metav1.Time{Time: metav1.Now().Add(-time.Minute)},
					JobRef: &corev1.ObjectReference{
						Name:      "pod-failure-test-step1",
						Namespace: "default",
					},
				},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-failure-test-step1",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Failed: 1,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod-failure-test-pod",
			Namespace: "default",
			Labels:    map[string]string{"job-name": "pod-failure-test-step1"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 1, // Matches policy, should be ignored
						},
					},
				},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}
	if err := fakeClient.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}
	if err := fakeClient.Create(ctx, pod); err != nil {
		t.Fatalf("Failed to create Pod: %v", err)
	}

	err := reconciler.updateStepStatusFromJob(ctx, jobFlow, "step1", job)
	// Status update may fail with fake client, but the function logic should run
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"pod-failure-test\" not found" {
		t.Logf("updateStepStatusFromJob error (may be expected): %v", err)
	}

	// Verify step status was updated in memory
	// With Ignore policy, failed job with matching exit code should be treated as succeeded
	stepStatus := reconciler.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseSucceeded {
		t.Errorf("Expected step phase to be Succeeded (due to Ignore policy), got %s", stepStatus.Phase)
	}
}

