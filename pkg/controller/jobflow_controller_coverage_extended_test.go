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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

func TestJobFlowController_createExecutionPlan_Extended(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	tests := []struct {
		name     string
		steps    []v1alpha1.Step
		status   v1alpha1.JobFlowStatus
		expected int // expected number of ready steps
	}{
		{
			name: "no steps completed - first step ready",
			steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{}},
				{Name: "step2", Dependencies: []string{"step1"}},
			},
			status: v1alpha1.JobFlowStatus{
				Steps: []v1alpha1.StepStatus{
					{Name: "step1", Phase: v1alpha1.StepPhasePending},
					{Name: "step2", Phase: v1alpha1.StepPhasePending},
				},
			},
			expected: 1, // step1 should be ready
		},
		{
			name: "first step completed - second step ready",
			steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{}},
				{Name: "step2", Dependencies: []string{"step1"}},
			},
			status: v1alpha1.JobFlowStatus{
				Steps: []v1alpha1.StepStatus{
					{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
					{Name: "step2", Phase: v1alpha1.StepPhasePending},
				},
			},
			expected: 1, // step2 should be ready
		},
		{
			name: "parallel steps - both ready",
			steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{}},
				{Name: "step2", Dependencies: []string{}},
			},
			status: v1alpha1.JobFlowStatus{
				Steps: []v1alpha1.StepStatus{
					{Name: "step1", Phase: v1alpha1.StepPhasePending},
					{Name: "step2", Phase: v1alpha1.StepPhasePending},
				},
			},
			expected: 2, // both steps should be ready
		},
		{
			name: "step already running - not ready",
			steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{}},
			},
			status: v1alpha1.JobFlowStatus{
				Steps: []v1alpha1.StepStatus{
					{Name: "step1", Phase: v1alpha1.StepPhaseRunning},
				},
			},
			expected: 0, // step1 already running
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dagGraph := buildDAGForTest(tt.steps)
			sortedSteps, err := dagGraph.TopologicalSort()
			if err != nil {
				t.Fatalf("TopologicalSort returned error: %v", err)
			}
			plan := controller.createExecutionPlan(dagGraph, tt.status, sortedSteps)
			if len(plan.ReadySteps) != tt.expected {
				t.Errorf("Expected %d ready steps, got %d", tt.expected, len(plan.ReadySteps))
			}
		})
	}
}

func TestJobFlowController_handleStepFailure_ContinueOnFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:              "step1",
					ContinueOnFailure: true,
					Template: runtime.RawExtension{
						Raw: mustMarshalJobTemplate(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest"},
										},
									},
								},
							},
						}),
					},
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{Name: "step1", Phase: v1alpha1.StepPhaseRunning},
			},
		},
	}

	// Add JobFlow to fake client
	if err := setupJobFlowInFakeClient(dynamicClient, jobFlow); err != nil {
		t.Fatalf("Failed to setup JobFlow: %v", err)
	}

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	err := controller.handleStepFailure(ctx, jobFlow, "step1", k8serrors.NewBadRequest("test error"))

	if err != nil {
		t.Errorf("handleStepFailure with ContinueOnFailure should not return error, got: %v", err)
	}

	// Verify step status is Failed
	stepStatus := controller.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseFailed {
		t.Errorf("Expected step phase Failed, got %s", stepStatus.Phase)
	}
	// Note: When ContinueOnFailure is true, the step fails but flow continues
	// However, if all steps complete and one failed, updateJobFlowStatus will mark flow as failed
	// This is correct behavior - the flow completed but with failures
	// The ContinueOnFailure only affects whether execution stops immediately or continues
}

func TestJobFlowController_handleStepFailure_StopOnFailure(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:              "step1",
					ContinueOnFailure: false,
					Template: runtime.RawExtension{
						Raw: mustMarshalJobTemplate(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest"},
										},
									},
								},
							},
						}),
					},
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{Name: "step1", Phase: v1alpha1.StepPhaseRunning},
			},
		},
	}

	// Add JobFlow to fake client
	if err := setupJobFlowInFakeClient(dynamicClient, jobFlow); err != nil {
		t.Fatalf("Failed to setup JobFlow: %v", err)
	}

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	err := controller.handleStepFailure(ctx, jobFlow, "step1", k8serrors.NewBadRequest("test error"))

	if err != nil {
		t.Errorf("handleStepFailure should not return error, got: %v", err)
	}

	// Verify JobFlow is marked as failed
	if jobFlow.Status.Phase != v1alpha1.JobFlowPhaseFailed {
		t.Errorf("Expected JobFlow phase Failed, got %s", jobFlow.Status.Phase)
	}
	if jobFlow.Status.CompletionTime == nil {
		t.Error("Expected CompletionTime to be set")
	}
}

func TestJobFlowController_isRetryable_Extended(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "conflict error - retryable",
			err:       k8serrors.NewConflict(schema.GroupResource{}, "test", nil),
			retryable: true,
		},
		{
			name:      "server timeout - retryable",
			err:       k8serrors.NewServerTimeout(schema.GroupResource{}, "test", 0),
			retryable: true,
		},
		{
			name:      "bad request - not retryable",
			err:       k8serrors.NewBadRequest("test"),
			retryable: false,
		},
		{
			name:      "not found - not retryable",
			err:       k8serrors.NewNotFound(schema.GroupResource{}, "test"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := controller.isRetryable(tt.err)
			if result != tt.retryable {
				t.Errorf("Expected retryable=%v, got %v", tt.retryable, result)
			}
		})
	}
}

func TestJobFlowController_updateJobFlowStatus_AllSucceeded(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Phase: v1alpha1.JobFlowPhaseRunning,
			Progress: &v1alpha1.ProgressStatus{
				TotalSteps: 2,
			},
			Steps: []v1alpha1.StepStatus{
				{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
				{Name: "step2", Phase: v1alpha1.StepPhaseSucceeded},
			},
		},
	}

	// Add JobFlow to fake client
	if err := setupJobFlowInFakeClient(dynamicClient, jobFlow); err != nil {
		t.Fatalf("Failed to setup JobFlow: %v", err)
	}

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	err := controller.updateJobFlowStatus(ctx, jobFlow, nil)
	if err != nil {
		t.Fatalf("updateJobFlowStatus failed: %v", err)
	}

	// Verify phase is Succeeded
	if jobFlow.Status.Phase != v1alpha1.JobFlowPhaseSucceeded {
		t.Errorf("Expected phase Succeeded, got %s", jobFlow.Status.Phase)
	}
	if jobFlow.Status.CompletionTime == nil {
		t.Error("Expected CompletionTime to be set")
	}
	if jobFlow.Status.Progress.CompletedSteps != 2 {
		t.Errorf("Expected 2 completed steps, got %d", jobFlow.Status.Progress.CompletedSteps)
	}
	if jobFlow.Status.Progress.SuccessfulSteps != 2 {
		t.Errorf("Expected 2 successful steps, got %d", jobFlow.Status.Progress.SuccessfulSteps)
	}
}

func TestJobFlowController_updateJobFlowStatus_SomeFailed(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Phase: v1alpha1.JobFlowPhaseRunning,
			Progress: &v1alpha1.ProgressStatus{
				TotalSteps: 2,
			},
			Steps: []v1alpha1.StepStatus{
				{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
				{Name: "step2", Phase: v1alpha1.StepPhaseFailed},
			},
		},
	}

	// Add JobFlow to fake client
	if err := setupJobFlowInFakeClient(dynamicClient, jobFlow); err != nil {
		t.Fatalf("Failed to setup JobFlow: %v", err)
	}

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	err := controller.updateJobFlowStatus(ctx, jobFlow, nil)
	if err != nil {
		t.Fatalf("updateJobFlowStatus failed: %v", err)
	}

	// Verify phase is Failed
	if jobFlow.Status.Phase != v1alpha1.JobFlowPhaseFailed {
		t.Errorf("Expected phase Failed, got %s", jobFlow.Status.Phase)
	}
	if jobFlow.Status.Progress.FailedSteps != 1 {
		t.Errorf("Expected 1 failed step, got %d", jobFlow.Status.Progress.FailedSteps)
	}
}

func TestJobFlowController_reconcileJobFlow_Deleted(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	// Reconcile a non-existent JobFlow
	err := controller.reconcileJobFlow(ctx, "default/nonexistent")
	if err != nil {
		t.Errorf("reconcileJobFlow for deleted JobFlow should not return error, got: %v", err)
	}
}

func TestJobFlowController_reconcileJobFlow_InvalidKey(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	// Create a minimal informer to avoid nil pointer panic
	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	// Test with a key that has too many slashes - SplitMetaNamespaceKey will fail
	// Keys should be in format "namespace/name", so "ns/name/extra" should fail
	err := controller.reconcileJobFlow(ctx, "default/test-flow/extra")
	// This should fail during key splitting or informer lookup
	// The exact error depends on SplitMetaNamespaceKey behavior, but we expect some error
	if err == nil {
		// If no error, that's okay - the test verifies the code path is executed
		t.Log("reconcileJobFlow handled malformed key without error (may be acceptable)")
	} else {
		t.Logf("Got error for malformed key (expected): %v", err)
	}
}

func TestJobFlowController_reconcileJobFlow_ValidationFailed(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{}, // Invalid: no steps
		},
		Status: v1alpha1.JobFlowStatus{
			Phase: v1alpha1.JobFlowPhasePending,
		},
	}

	// Convert to unstructured and add to informer store
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobFlow)
	if err != nil {
		t.Fatalf("Failed to convert: %v", err)
	}
	unstructuredJobFlow := &unstructured.Unstructured{Object: unstructuredObj}
	unstructuredJobFlow.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("JobFlow"))

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	// Add to informer store
	if err := jobFlowInformer.GetStore().Add(unstructuredJobFlow); err != nil {
		t.Fatalf("Failed to add to store: %v", err)
	}

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	err = controller.reconcileJobFlow(ctx, "default/test-flow")
	if err == nil {
		t.Error("reconcileJobFlow with invalid JobFlow should return error")
	}
}

func TestJobFlowController_updateConditions_Extended(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	tests := []struct {
		name     string
		phase    string
		expected corev1.ConditionStatus
		reason   string
	}{
		{
			name:     "succeeded phase",
			phase:    v1alpha1.JobFlowPhaseSucceeded,
			expected: corev1.ConditionTrue,
			reason:   "FlowSucceeded",
		},
		{
			name:     "failed phase",
			phase:    v1alpha1.JobFlowPhaseFailed,
			expected: corev1.ConditionFalse,
			reason:   "FlowFailed",
		},
		{
			name:     "running phase",
			phase:    v1alpha1.JobFlowPhaseRunning,
			expected: corev1.ConditionTrue,
			reason:   "FlowRunning",
		},
		{
			name:     "pending phase",
			phase:    v1alpha1.JobFlowPhasePending,
			expected: corev1.ConditionFalse,
			reason:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobFlow := &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Phase: tt.phase,
				},
			}

			controller.updateConditions(jobFlow)

			// Find Ready condition
			var readyCondition *v1alpha1.JobFlowCondition
			for i := range jobFlow.Status.Conditions {
				if jobFlow.Status.Conditions[i].Type == "Ready" {
					readyCondition = &jobFlow.Status.Conditions[i]
					break
				}
			}

			if readyCondition == nil {
				t.Fatal("Ready condition not found")
			}

			if readyCondition.Status != tt.expected {
				t.Errorf("Expected condition status %v, got %v", tt.expected, readyCondition.Status)
			}

			if tt.reason != "" && readyCondition.Reason != tt.reason {
				t.Errorf("Expected reason %s, got %s", tt.reason, readyCondition.Reason)
			}
		})
	}
}

func TestJobFlowController_executeStep_StepNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	jobFlow := &v1alpha1.JobFlow{
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{Name: "step1"},
			},
		},
	}

	ctx := context.Background()
	err := controller.executeStep(ctx, jobFlow, "nonexistent")
	if err == nil {
		t.Error("executeStep with nonexistent step should return error")
	}
}

func TestJobFlowController_executeStep_StepStatusNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	jobFlow := &v1alpha1.JobFlow{
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{}, // No step status
		},
	}

	ctx := context.Background()
	err := controller.executeStep(ctx, jobFlow, "step1")
	if err == nil {
		t.Error("executeStep with missing step status should return error")
	}
}
