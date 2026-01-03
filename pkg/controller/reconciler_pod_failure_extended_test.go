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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestJobFlowReconciler_getContainerExitCodes(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tests := []struct {
		name           string
		pod            *corev1.Pod
		expectedCodes  map[string]int32
		expectedLength int
	}{
		{
			name: "pod with terminated container",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "main",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 0,
								},
							},
						},
					},
				},
			},
			expectedCodes: map[string]int32{
				"main": 0,
			},
			expectedLength: 1,
		},
		{
			name: "pod with multiple terminated containers",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "main",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 1,
								},
							},
						},
						{
							Name: "sidecar",
							State: corev1.ContainerState{
								Terminated: &corev1.ContainerStateTerminated{
									ExitCode: 2,
								},
							},
						},
					},
				},
			},
			expectedCodes: map[string]int32{
				"main":   1,
				"sidecar": 2,
			},
			expectedLength: 2,
		},
		{
			name: "pod with running container (no exit code)",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "main",
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{},
							},
						},
					},
				},
			},
			expectedCodes:  map[string]int32{},
			expectedLength: 0,
		},
		{
			name: "pod with no container statuses",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{},
				},
			},
			expectedCodes:  map[string]int32{},
			expectedLength: 0,
		},
		{
			name: "pod with waiting container (no exit code)",
			pod: &corev1.Pod{
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{
						{
							Name: "main",
							State: corev1.ContainerState{
								Waiting: &corev1.ContainerStateWaiting{},
							},
						},
					},
				},
			},
			expectedCodes:  map[string]int32{},
			expectedLength: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCodes := reconciler.getContainerExitCodes(tt.pod)
			if len(exitCodes) != tt.expectedLength {
				t.Errorf("Expected %d exit codes, got %d", tt.expectedLength, len(exitCodes))
			}
			for containerName, expectedCode := range tt.expectedCodes {
				if actualCode, exists := exitCodes[containerName]; !exists {
					t.Errorf("Expected exit code for container %s, but not found", containerName)
				} else if actualCode != expectedCode {
					t.Errorf("Expected exit code %d for container %s, got %d", expectedCode, containerName, actualCode)
				}
			}
		})
	}
}

func TestJobFlowReconciler_matchExitCodes(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tests := []struct {
		name       string
		exitCodes  map[string]int32
		rule       *v1alpha1.PodFailurePolicyOnExitCodes
		expected   bool
	}{
		{
			name: "In operator - matching exit code",
			exitCodes: map[string]int32{
				"main": 1,
			},
			rule: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "In",
				Values:   []int32{1, 2, 3},
			},
			expected: true,
		},
		{
			name: "In operator - non-matching exit code",
			exitCodes: map[string]int32{
				"main": 5,
			},
			rule: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "In",
				Values:   []int32{1, 2, 3},
			},
			expected: false,
		},
		{
			name: "NotIn operator - exit code not in list",
			exitCodes: map[string]int32{
				"main": 5,
			},
			rule: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "NotIn",
				Values:   []int32{1, 2, 3},
			},
			expected: true,
		},
		{
			name: "NotIn operator - exit code in list",
			exitCodes: map[string]int32{
				"main": 2,
			},
			rule: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "NotIn",
				Values:   []int32{1, 2, 3},
			},
			expected: false,
		},
		{
			name: "In operator with specific container name",
			exitCodes: map[string]int32{
				"sidecar": 1,
				"main":    2,
			},
			rule: &v1alpha1.PodFailurePolicyOnExitCodes{
				ContainerName: "sidecar",
				Operator:      "In",
				Values:        []int32{1, 2, 3},
			},
			expected: true,
		},
		{
			name: "In operator with non-existent container",
			exitCodes: map[string]int32{
				"main": 1,
			},
			rule: &v1alpha1.PodFailurePolicyOnExitCodes{
				ContainerName: "nonexistent",
				Operator:      "In",
				Values:        []int32{1, 2, 3},
			},
			expected: false,
		},
		{
			name: "default container name (main)",
			exitCodes: map[string]int32{
				"main": 1,
			},
			rule: &v1alpha1.PodFailurePolicyOnExitCodes{
				ContainerName: "", // Empty should default to "main"
				Operator:      "In",
				Values:        []int32{1, 2, 3},
			},
			expected: true,
		},
		{
			name: "invalid operator defaults to false",
			exitCodes: map[string]int32{
				"main": 1,
			},
			rule: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "Invalid",
				Values:   []int32{1, 2, 3},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reconciler.matchExitCodes(tt.exitCodes, tt.rule)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestJobFlowReconciler_checkPodFailurePolicy_NoPolicy(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-policy-test",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}
	if err := fakeClient.Create(ctx, job); err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}

	// No policy should default to fail
	shouldFail, err := reconciler.checkPodFailurePolicy(ctx, jobFlow, "step1", job)
	if err != nil {
		t.Fatalf("checkPodFailurePolicy returned error: %v", err)
	}
	if !shouldFail {
		t.Error("Expected shouldFail=true when no policy is set")
	}
}

func TestJobFlowReconciler_checkPodFailurePolicy_NoExitCodes(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-exit-codes-test",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			ExecutionPolicy: &v1alpha1.ExecutionPolicy{
				PodFailurePolicy: &v1alpha1.PodFailurePolicy{
					Rules: []v1alpha1.PodFailurePolicyRule{
						{
							Action:      "Ignore",
							OnExitCodes: nil, // No exit codes specified
						},
					},
				},
			},
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels:    map[string]string{"job-name": "test-job"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
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

	// Should apply action directly when no exit codes
	shouldFail, err := reconciler.checkPodFailurePolicy(ctx, jobFlow, "step1", job)
	if err != nil {
		t.Fatalf("checkPodFailurePolicy returned error: %v", err)
	}
	// Ignore action should return false (don't fail)
	if shouldFail {
		t.Error("Expected shouldFail=false for Ignore action")
	}
}

func TestJobFlowReconciler_checkPodFailurePolicy_NotInOperator(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "notin-test",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			ExecutionPolicy: &v1alpha1.ExecutionPolicy{
				PodFailurePolicy: &v1alpha1.PodFailurePolicy{
					Rules: []v1alpha1.PodFailurePolicyRule{
						{
							Action: "Ignore",
							OnExitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
								Operator: "NotIn",
								Values:   []int32{1, 2},
							},
						},
					},
				},
			},
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels:    map[string]string{"job-name": "test-job"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 5, // Not in [1, 2], so should match NotIn
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

	shouldFail, err := reconciler.checkPodFailurePolicy(ctx, jobFlow, "step1", job)
	if err != nil {
		t.Fatalf("checkPodFailurePolicy returned error: %v", err)
	}
	// Exit code 5 is NotIn [1, 2], so rule matches, action is Ignore, should not fail
	if shouldFail {
		t.Error("Expected shouldFail=false for NotIn operator with matching exit code")
	}
}

func TestJobFlowReconciler_checkPodFailurePolicy_NoMatchingRules(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-match-test",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			ExecutionPolicy: &v1alpha1.ExecutionPolicy{
				PodFailurePolicy: &v1alpha1.PodFailurePolicy{
					Rules: []v1alpha1.PodFailurePolicyRule{
						{
							Action: "Ignore",
							OnExitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
								Operator: "In",
								Values:   []int32{1, 2},
							},
						},
					},
				},
			},
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels:    map[string]string{"job-name": "test-job"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name: "main",
					State: corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 99, // Not in [1, 2], so rule doesn't match
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

	// No rules matched, should default to fail
	shouldFail, err := reconciler.checkPodFailurePolicy(ctx, jobFlow, "step1", job)
	if err != nil {
		t.Fatalf("checkPodFailurePolicy returned error: %v", err)
	}
	if !shouldFail {
		t.Error("Expected shouldFail=true when no rules match")
	}
}

func TestJobFlowReconciler_checkPodFailurePolicy_NonFailedPod(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "non-failed-test",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			ExecutionPolicy: &v1alpha1.ExecutionPolicy{
				PodFailurePolicy: &v1alpha1.PodFailurePolicy{
					Rules: []v1alpha1.PodFailurePolicyRule{
						{
							Action: "Ignore",
							OnExitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
								Operator: "In",
								Values:   []int32{1, 2},
							},
						},
					},
				},
			},
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels:    map[string]string{"job-name": "test-job"},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodSucceeded, // Not failed, should be skipped
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

	// Non-failed pods should be skipped, no rules match, default to fail
	shouldFail, err := reconciler.checkPodFailurePolicy(ctx, jobFlow, "step1", job)
	if err != nil {
		t.Fatalf("checkPodFailurePolicy returned error: %v", err)
	}
	if !shouldFail {
		t.Error("Expected shouldFail=true when no failed pods match rules")
	}
}

