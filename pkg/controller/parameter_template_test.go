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

func TestJobFlowReconciler_substituteParameter(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tests := []struct {
		name     string
		input    string
		params   map[string]string
		expected string
	}{
		{
			name:     "dot notation",
			input:    "echo {{.parameters.param1}}",
			params:   map[string]string{"param1": "value1"},
			expected: "echo value1",
		},
		{
			name:     "no dot notation",
			input:    "echo {{parameters.param1}}",
			params:   map[string]string{"param1": "value1"},
			expected: "echo value1",
		},
		{
			name:     "dollar notation",
			input:    "echo ${parameters.param1}",
			params:   map[string]string{"param1": "value1"},
			expected: "echo value1",
		},
		{
			name:     "multiple parameters",
			input:    "{{.parameters.param1}} and {{parameters.param2}}",
			params:   map[string]string{"param1": "value1", "param2": "value2"},
			expected: "value1 and value2",
		},
		{
			name:     "no parameters",
			input:    "echo hello",
			params:   map[string]string{},
			expected: "echo hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reconciler.substituteParameter(tt.input, tt.params)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestJobFlowReconciler_applyParametersToContainerArgs(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	containers := []corev1.Container{
		{
			Name:    "main",
			Command: []string{"echo", "{{.parameters.message}}"},
			Args:    []string{"{{parameters.arg1}}", "${parameters.arg2}"},
			Env: []corev1.EnvVar{
				{Name: "VAR1", Value: "{{.parameters.env1}}"},
				{Name: "VAR2", Value: "static"},
			},
		},
	}

	params := map[string]string{
		"message": "Hello",
		"arg1":    "world",
		"arg2":    "test",
		"env1":    "value1",
	}

	reconciler.applyParametersToContainerArgs(containers, params)

	// Verify command substitution
	if containers[0].Command[1] != "Hello" {
		t.Errorf("Expected command arg 'Hello', got %s", containers[0].Command[1])
	}

	// Verify args substitution
	if containers[0].Args[0] != "world" {
		t.Errorf("Expected arg[0] 'world', got %s", containers[0].Args[0])
	}
	if containers[0].Args[1] != "test" {
		t.Errorf("Expected arg[1] 'test', got %s", containers[0].Args[1])
	}

	// Verify env substitution
	if containers[0].Env[0].Value != "value1" {
		t.Errorf("Expected env VAR1 'value1', got %s", containers[0].Env[0].Value)
	}
	if containers[0].Env[1].Value != "static" {
		t.Errorf("Expected env VAR2 'static', got %s", containers[0].Env[1].Value)
	}
}

func TestJobFlowReconciler_applyParametersToJobTemplate(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: runtime.RawExtension{
						Raw: mustMarshalJobTemplate(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "main",
												Command: []string{"echo", "{{.parameters.message}}"},
											},
										},
									},
								},
							},
						}),
					},
				},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	step := &jobFlow.Spec.Steps[0]
	job := &batchv1.Job{
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "main",
							Command: []string{"echo", "{{.parameters.message}}"},
						},
					},
				},
			},
		},
	}

	params := map[string]string{
		"message": "Hello World",
	}

	err := reconciler.applyParametersToJobTemplate(ctx, jobFlow, step, job, params)
	if err != nil {
		t.Fatalf("Failed to apply parameters: %v", err)
	}

	// Verify parameter was substituted in container command
	if len(job.Spec.Template.Spec.Containers) > 0 {
		if len(job.Spec.Template.Spec.Containers[0].Command) > 1 {
			// The substitution might happen via string replacement
			// Check that the placeholder is gone
			cmd := job.Spec.Template.Spec.Containers[0].Command[1]
			if cmd == "{{.parameters.message}}" {
				t.Error("Parameter placeholder was not substituted")
			}
		}
	}
}

func TestJobFlowReconciler_applyParametersToJobTemplate_NoParams(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	step := &jobFlow.Spec.Steps[0]
	job := &batchv1.Job{
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "main"},
					},
				},
			},
		},
	}

	// No parameters
	err := reconciler.applyParametersToJobTemplate(ctx, jobFlow, step, job, nil)
	if err != nil {
		t.Fatalf("Should not fail with no parameters: %v", err)
	}

	// Should also work with empty map
	err = reconciler.applyParametersToJobTemplate(ctx, jobFlow, step, job, map[string]string{})
	if err != nil {
		t.Fatalf("Should not fail with empty parameters: %v", err)
	}
}
