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

package validation

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestValidateJobFlow(t *testing.T) {
	tests := []struct {
		name        string
		jobFlow     *v1alpha1.JobFlow
		expectError bool
	}{
		{
			name: "valid JobFlow",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name:         "step1",
							Dependencies: []string{},
							Template: runtime.RawExtension{
								Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "no steps",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{},
				},
			},
			expectError: true,
		},
		{
			name: "duplicate step names",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
						{Name: "step1"}, // Duplicate
					},
				},
			},
			expectError: true,
		},
		{
			name: "empty step name",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: ""},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid dependency",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name:         "step1",
							Dependencies: []string{"nonexistent"},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "valid dependencies",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name:         "step1",
							Dependencies: []string{},
							Template: runtime.RawExtension{
								Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
							},
						},
						{
							Name:         "step2",
							Dependencies: []string{"step1"},
							Template: runtime.RawExtension{
								Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "cycle in dependencies",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name:         "step1",
							Dependencies: []string{"step2"},
							Template: runtime.RawExtension{
								Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
							},
						},
						{
							Name:         "step2",
							Dependencies: []string{"step1"},
							Template: runtime.RawExtension{
								Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid concurrency policy",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						ConcurrencyPolicy: "Invalid",
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
			},
			expectError: true,
		},
		{
			name: "negative TTL",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						TTLSecondsAfterFinished: int32Ptr(-1),
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
			},
			expectError: true,
		},
		{
			name: "negative backoff limit",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						BackoffLimit: int32Ptr(-1),
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
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJobFlow(tt.jobFlow)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateDAG(t *testing.T) {
	tests := []struct {
		name        string
		steps       []v1alpha1.Step
		expectError bool
	}{
		{
			name: "no cycle",
			steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{}},
				{Name: "step2", Dependencies: []string{"step1"}},
				{Name: "step3", Dependencies: []string{"step2"}},
			},
			expectError: false,
		},
		{
			name: "self-cycle",
			steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{"step1"}},
			},
			expectError: true,
		},
		{
			name: "two-step cycle",
			steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{"step2"}},
				{Name: "step2", Dependencies: []string{"step1"}},
			},
			expectError: true,
		},
		{
			name: "three-step cycle",
			steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{"step3"}},
				{Name: "step2", Dependencies: []string{"step1"}},
				{Name: "step3", Dependencies: []string{"step2"}},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDAG(tt.steps)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateExecutionPolicy(t *testing.T) {
	tests := []struct {
		name        string
		policy      *v1alpha1.ExecutionPolicy
		expectError bool
	}{
		{
			name:        "valid policy",
			policy:      &v1alpha1.ExecutionPolicy{ConcurrencyPolicy: "Forbid"},
			expectError: false,
		},
		{
			name:        "invalid concurrency policy",
			policy:      &v1alpha1.ExecutionPolicy{ConcurrencyPolicy: "Invalid"},
			expectError: true,
		},
		{
			name:        "negative TTL",
			policy:      &v1alpha1.ExecutionPolicy{TTLSecondsAfterFinished: int32Ptr(-1)},
			expectError: true,
		},
		{
			name:        "zero TTL (valid)",
			policy:      &v1alpha1.ExecutionPolicy{TTLSecondsAfterFinished: int32Ptr(0)},
			expectError: false,
		},
		{
			name:        "negative backoff limit",
			policy:      &v1alpha1.ExecutionPolicy{BackoffLimit: int32Ptr(-1)},
			expectError: true,
		},
		{
			name:        "zero backoff limit (valid)",
			policy:      &v1alpha1.ExecutionPolicy{BackoffLimit: int32Ptr(0)},
			expectError: false,
		},
		{
			name:        "zero active deadline seconds (invalid)",
			policy:      &v1alpha1.ExecutionPolicy{ActiveDeadlineSeconds: int64Ptr(0)},
			expectError: true,
		},
		{
			name:        "negative active deadline seconds (invalid)",
			policy:      &v1alpha1.ExecutionPolicy{ActiveDeadlineSeconds: int64Ptr(-1)},
			expectError: true,
		},
		{
			name:        "positive active deadline seconds (valid)",
			policy:      &v1alpha1.ExecutionPolicy{ActiveDeadlineSeconds: int64Ptr(3600)},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExecutionPolicy(tt.policy)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateStepName(t *testing.T) {
	tests := []struct {
		name        string
		stepName    string
		expectError bool
	}{
		{
			name:        "valid step name",
			stepName:    "step-1",
			expectError: false,
		},
		{
			name:        "empty step name",
			stepName:    "",
			expectError: true,
		},
		{
			name:        "step name with leading space",
			stepName:    " step1",
			expectError: true,
		},
		{
			name:        "step name with trailing space",
			stepName:    "step1 ",
			expectError: true,
		},
		{
			name:        "invalid DNS name",
			stepName:    "step_1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStepName(tt.stepName)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateResourceTemplates(t *testing.T) {
	tests := []struct {
		name        string
		templates   *v1alpha1.ResourceTemplates
		expectError bool
	}{
		{
			name: "valid PVC template",
			templates: &v1alpha1.ResourceTemplates{
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "pvc1"},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("10Gi"),
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "PVC without name",
			templates: &v1alpha1.ResourceTemplates{
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("10Gi"),
								},
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "valid ConfigMap template",
			templates: &v1alpha1.ResourceTemplates{
				ConfigMapTemplates: []corev1.ConfigMap{
					{
						ObjectMeta: metav1.ObjectMeta{Name: "cm1"},
					},
				},
			},
			expectError: false,
		},
		{
			name: "ConfigMap without name",
			templates: &v1alpha1.ResourceTemplates{
				ConfigMapTemplates: []corev1.ConfigMap{
					{},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResourceTemplates(tt.templates)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func int32Ptr(i int32) *int32 {
	return &i
}

func int64Ptr(i int64) *int64 {
	return &i
}
