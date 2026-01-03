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
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestJobFlowReconciler_parseJSONPathWithStepName(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tests := []struct {
		name         string
		jsonPathStr  string
		expectedStep string
		expectedPath string
	}{
		{
			name:         "step name with JSONPath",
			jsonPathStr:  "step1:$.status.succeeded",
			expectedStep: "step1",
			expectedPath: "$.status.succeeded",
		},
		{
			name:         "step name with spaces",
			jsonPathStr:  "step1 : $.status.succeeded",
			expectedStep: "step1",
			expectedPath: "$.status.succeeded",
		},
		{
			name:         "no step name",
			jsonPathStr:  "$.status.succeeded",
			expectedStep: "",
			expectedPath: "$.status.succeeded",
		},
		{
			name:         "complex JSONPath",
			jsonPathStr:  "step1:$.status.conditions[?(@.type=='Ready')].status",
			expectedStep: "step1",
			expectedPath: "$.status.conditions[?(@.type=='Ready')].status",
		},
		{
			name:         "colon in JSONPath (not step separator)",
			jsonPathStr:  "$.metadata.annotations['key:value']",
			expectedStep: "",
			expectedPath: "$.metadata.annotations['key:value']",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stepName, jsonPath := reconciler.parseJSONPathWithStepName(tt.jsonPathStr)
			if stepName != tt.expectedStep {
				t.Errorf("Expected step name %q, got %q", tt.expectedStep, stepName)
			}
			if jsonPath != tt.expectedPath {
				t.Errorf("Expected JSONPath %q, got %q", tt.expectedPath, jsonPath)
			}
		})
	}
}

func TestJobFlowReconciler_resolveParameter_JSONPathWithStepName(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	// Create a JobFlow with a completed step
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:     "step1",
					Template: runtime.RawExtension{},
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseSucceeded,
					JobRef: &corev1.ObjectReference{
						Name:      "step1-job",
						Namespace: "default",
					},
				},
			},
		},
	}

	// Create a Job with status
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "step1-job",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
			Conditions: []batchv1.JobCondition{
				{
					Type:   batchv1.JobComplete,
					Status: corev1.ConditionTrue,
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

	// Test JSONPath with step name
	// Note: The code marshals job.Status, so the JSONPath should be $.succeeded, not $.status.succeeded
	param := &v1alpha1.ParameterInput{
		Name: "test-param",
		ValueFrom: &v1alpha1.ParameterValueFrom{
			JSONPath: "step1:$.succeeded",
		},
	}

	value, err := reconciler.resolveParameter(ctx, jobFlow, param)
	if err != nil {
		t.Fatalf("Failed to resolve parameter: %v", err)
	}

	// Should extract the succeeded count (1) as string
	if value != "1" {
		t.Errorf("Expected parameter value '1', got %q", value)
	}
}

func TestJobFlowReconciler_resolveParameter_JSONPathWithoutStepName(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1", Template: runtime.RawExtension{}},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Test JSONPath without step name (should fail)
	param := &v1alpha1.ParameterInput{
		Name: "test-param",
		ValueFrom: &v1alpha1.ParameterValueFrom{
			JSONPath: "$.status.succeeded",
		},
	}

	_, err := reconciler.resolveParameter(ctx, jobFlow, param)
	if err == nil {
		t.Error("Expected error when JSONPath doesn't include step name")
	}
	if err != nil && err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestJobFlowReconciler_evaluateJSONPath(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tests := []struct {
		name        string
		jsonData    string
		jsonPath    string
		expectError bool
		expected    string
	}{
		{
			name:        "simple string value",
			jsonData:    `{"status": "succeeded"}`,
			jsonPath:    "$.status",
			expectError: false,
			expected:    "succeeded",
		},
		{
			name:        "numeric value",
			jsonData:    `{"count": 42}`,
			jsonPath:    "$.count",
			expectError: false,
			expected:    "42",
		},
		{
			name:        "boolean value",
			jsonData:    `{"ready": true}`,
			jsonPath:    "$.ready",
			expectError: false,
			expected:    "true",
		},
		{
			name:        "nested path",
			jsonData:    `{"status": {"phase": "Running"}}`,
			jsonPath:    "$.status.phase",
			expectError: false,
			expected:    "Running",
		},
		{
			name:        "invalid JSONPath",
			jsonData:    `{"status": "succeeded"}`,
			jsonPath:    "$.invalid.path",
			expectError: true,
		},
		{
			name:        "invalid JSON",
			jsonData:    `{invalid json}`,
			jsonPath:    "$.status",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reconciler.evaluateJSONPath(tt.jsonData, tt.jsonPath)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}
