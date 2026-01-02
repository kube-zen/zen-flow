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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     string
		expectError bool
	}{
		{
			name:        "valid timeout - seconds",
			timeout:     "30s",
			expectError: false,
		},
		{
			name:        "valid timeout - minutes",
			timeout:     "5m",
			expectError: false,
		},
		{
			name:        "valid timeout - hours",
			timeout:     "2h",
			expectError: false,
		},
		{
			name:        "empty timeout (valid)",
			timeout:     "",
			expectError: false,
		},
		{
			name:        "invalid timeout format",
			timeout:     "invalid",
			expectError: true,
		},
		{
			name:        "zero timeout",
			timeout:     "0s",
			expectError: true,
		},
		{
			name:        "negative timeout",
			timeout:     "-5s",
			expectError: true,
		},
		{
			name:        "timeout exceeds maximum",
			timeout:     "31d", // 31 days > 30 days max
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTimeout(tt.timeout)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateRetryPolicy(t *testing.T) {
	tests := []struct {
		name        string
		policy      *v1alpha1.RetryPolicy
		expectError bool
	}{
		{
			name:        "valid retry policy",
			policy:      &v1alpha1.RetryPolicy{Limit: 3},
			expectError: false,
		},
		{
			name:        "negative retry limit",
			policy:      &v1alpha1.RetryPolicy{Limit: -1},
			expectError: true,
		},
		{
			name: "valid backoff",
			policy: &v1alpha1.RetryPolicy{
				Backoff: &v1alpha1.BackoffPolicy{
					Type:     "Exponential",
					Duration: "1s",
					Factor:   float64Ptr(2.0),
				},
			},
			expectError: false,
		},
		{
			name: "invalid backoff factor",
			policy: &v1alpha1.RetryPolicy{
				Backoff: &v1alpha1.BackoffPolicy{
					Type:     "Exponential",
					Duration: "1s",
					Factor:   float64Ptr(0.5), // < 1.0
				},
			},
			expectError: true,
		},
		{
			name: "invalid backoff duration",
			policy: &v1alpha1.RetryPolicy{
				Backoff: &v1alpha1.BackoffPolicy{
					Type:     "Exponential",
					Duration: "invalid",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRetryPolicy(tt.policy)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateParameterInput(t *testing.T) {
	tests := []struct {
		name        string
		param       *v1alpha1.ParameterInput
		expectError bool
	}{
		{
			name:        "valid parameter with value",
			param:       &v1alpha1.ParameterInput{Name: "param1", Value: "value1"},
			expectError: false,
		},
		{
			name: "valid parameter with configMapKeyRef",
			param: &v1alpha1.ParameterInput{
				Name: "param1",
				ValueFrom: &v1alpha1.ParameterValueFrom{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "config",
						},
						Key: "key",
					},
				},
			},
			expectError: false,
		},
		{
			name:        "empty parameter name",
			param:       &v1alpha1.ParameterInput{Name: ""},
			expectError: true,
		},
		{
			name:        "invalid parameter name",
			param:       &v1alpha1.ParameterInput{Name: "param_1"}, // underscore not allowed
			expectError: true,
		},
		{
			name:        "neither value nor valueFrom",
			param:       &v1alpha1.ParameterInput{Name: "param1"},
			expectError: true,
		},
		{
			name: "both value and valueFrom",
			param: &v1alpha1.ParameterInput{
				Name:  "param1",
				Value: "value1",
				ValueFrom: &v1alpha1.ParameterValueFrom{
					ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "config",
						},
						Key: "key",
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParameterInput(tt.param, "step1", 0)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateJSONPath(t *testing.T) {
	tests := []struct {
		name        string
		jsonPath    string
		expectError bool
	}{
		{
			name:        "valid JSONPath",
			jsonPath:    "$.status.succeeded",
			expectError: false,
		},
		{
			name:        "valid JSONPath with step name",
			jsonPath:    "step1:$.status.succeeded",
			expectError: false,
		},
		{
			name:        "empty JSONPath",
			jsonPath:    "",
			expectError: true,
		},
		{
			name:        "JSONPath without $",
			jsonPath:    "status.succeeded",
			expectError: true,
		},
		{
			name:        "invalid step name in JSONPath",
			jsonPath:    "step_1:$.status", // underscore not allowed
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJSONPath(tt.jsonPath)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateArtifactPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "valid absolute path",
			path:        "/output/data.txt",
			expectError: false,
		},
		{
			name:        "empty path (valid)",
			path:        "",
			expectError: false,
		},
		{
			name:        "relative path",
			path:        "output/data.txt",
			expectError: true,
		},
		{
			name:        "path with ..",
			path:        "/output/../data.txt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateArtifactPath(tt.path)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateWhenCondition(t *testing.T) {
	tests := []struct {
		name        string
		when        string
		expectError bool
	}{
		{
			name:        "valid keyword - always",
			when:        "always",
			expectError: false,
		},
		{
			name:        "valid keyword - never",
			when:        "never",
			expectError: false,
		},
		{
			name:        "valid template expression",
			when:        "{{.parameters.value}} == 'test'",
			expectError: false,
		},
		{
			name:        "unbalanced template braces",
			when:        "{{.parameters.value}",
			expectError: true,
		},
		{
			name:        "empty condition (valid)",
			when:        "",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWhenCondition(tt.when)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateJobFlowEnhanced(t *testing.T) {
	timeoutSeconds := int64(30)
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:         "step1",
					Dependencies: []string{},
					Template: runtime.RawExtension{
						Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
					},
					TimeoutSeconds: &timeoutSeconds,
					Inputs: &v1alpha1.StepInputs{
						Parameters: []v1alpha1.ParameterInput{
							{Name: "param1", Value: "value1"},
						},
					},
				},
			},
		},
	}

	// Should pass enhanced validation
	if err := ValidateJobFlowEnhanced(jobFlow); err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Test with invalid timeout (exceeds maximum)
	maxTimeout := int64(31 * 24 * 3600) // 31 days
	jobFlow.Spec.Steps[0].TimeoutSeconds = &maxTimeout
	if err := ValidateJobFlowEnhanced(jobFlow); err == nil {
		t.Error("Expected error for timeout exceeding maximum but got none")
	}
}

func float64Ptr(f float64) *float64 {
	return &f
}

