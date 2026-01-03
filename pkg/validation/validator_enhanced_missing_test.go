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
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestValidateStepOutputs(t *testing.T) {
	tests := []struct {
		name        string
		outputs     *v1alpha1.StepOutputs
		stepName    string
		expectError bool
	}{
		{
			name:        "nil outputs (valid)",
			outputs:     nil,
			stepName:    "step1",
			expectError: false,
		},
		{
			name: "valid outputs with parameter",
			outputs: &v1alpha1.StepOutputs{
				Parameters: []v1alpha1.ParameterOutput{
					{
						Name: "param1",
						ValueFrom: v1alpha1.ParameterValueFrom{
							JSONPath: "$.status.succeeded",
						},
					},
				},
			},
			stepName:    "step1",
			expectError: false,
		},
		{
			name: "valid outputs with artifact",
			outputs: &v1alpha1.StepOutputs{
				Artifacts: []v1alpha1.ArtifactOutput{
					{
						Name: "artifact1",
						Path: "/tmp/artifact1.txt",
					},
				},
			},
			stepName:    "step1",
			expectError: false,
		},
		{
			name: "invalid parameter output - empty name",
			outputs: &v1alpha1.StepOutputs{
				Parameters: []v1alpha1.ParameterOutput{
					{
						Name: "",
						ValueFrom: v1alpha1.ParameterValueFrom{
							JSONPath: "$.status.succeeded",
						},
					},
				},
			},
			stepName:    "step1",
			expectError: true,
		},
		{
			name: "invalid parameter output - missing JSONPath",
			outputs: &v1alpha1.StepOutputs{
				Parameters: []v1alpha1.ParameterOutput{
					{
						Name:      "param1",
						ValueFrom: v1alpha1.ParameterValueFrom{},
					},
				},
			},
			stepName:    "step1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStepOutputs(tt.outputs, tt.stepName)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateParameterOutput(t *testing.T) {
	tests := []struct {
		name        string
		param       *v1alpha1.ParameterOutput
		stepName    string
		index       int
		expectError bool
	}{
		{
			name: "valid parameter output",
			param: &v1alpha1.ParameterOutput{
				Name: "param1",
				ValueFrom: v1alpha1.ParameterValueFrom{
					JSONPath: "$.status.succeeded",
				},
			},
			stepName:    "step1",
			index:       0,
			expectError: false,
		},
		{
			name: "empty parameter name",
			param: &v1alpha1.ParameterOutput{
				Name: "",
				ValueFrom: v1alpha1.ParameterValueFrom{
					JSONPath: "$.status.succeeded",
				},
			},
			stepName:    "step1",
			index:       0,
			expectError: true,
		},
		{
			name: "missing JSONPath",
			param: &v1alpha1.ParameterOutput{
				Name:      "param1",
				ValueFrom: v1alpha1.ParameterValueFrom{},
			},
			stepName:    "step1",
			index:       0,
			expectError: true,
		},
		{
			name: "invalid JSONPath",
			param: &v1alpha1.ParameterOutput{
				Name: "param1",
				ValueFrom: v1alpha1.ParameterValueFrom{
					JSONPath: "invalid-path",
				},
			},
			stepName:    "step1",
			index:       0,
			expectError: true,
		},
		{
			name: "invalid parameter name format",
			param: &v1alpha1.ParameterOutput{
				Name: "invalid name with spaces",
				ValueFrom: v1alpha1.ParameterValueFrom{
					JSONPath: "$.status.succeeded",
				},
			},
			stepName:    "step1",
			index:       0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParameterOutput(tt.param, tt.stepName, tt.index)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidatePodFailurePolicy(t *testing.T) {
	tests := []struct {
		name        string
		policy      *v1alpha1.PodFailurePolicy
		expectError bool
	}{
		{
			name:        "nil policy (valid)",
			policy:      nil,
			expectError: false,
		},
		{
			name: "valid policy with FailJob action",
			policy: &v1alpha1.PodFailurePolicy{
				Rules: []v1alpha1.PodFailurePolicyRule{
					{
						Action: "FailJob",
						OnExitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
							Operator: "In",
							Values:   []int32{1, 2, 3},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid policy with Ignore action",
			policy: &v1alpha1.PodFailurePolicy{
				Rules: []v1alpha1.PodFailurePolicyRule{
					{
						Action: "Ignore",
						OnExitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
							Operator: "NotIn",
							Values:   []int32{0},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid policy with Count action",
			policy: &v1alpha1.PodFailurePolicy{
				Rules: []v1alpha1.PodFailurePolicyRule{
					{
						Action: "Count",
						OnExitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
							Operator: "In",
							Values:   []int32{137},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty rules",
			policy: &v1alpha1.PodFailurePolicy{
				Rules: []v1alpha1.PodFailurePolicyRule{},
			},
			expectError: true,
		},
		{
			name: "invalid action",
			policy: &v1alpha1.PodFailurePolicy{
				Rules: []v1alpha1.PodFailurePolicyRule{
					{
						Action: "InvalidAction",
						OnExitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
							Operator: "In",
							Values:   []int32{1},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "missing OnExitCodes",
			policy: &v1alpha1.PodFailurePolicy{
				Rules: []v1alpha1.PodFailurePolicyRule{
					{
						Action:      "FailJob",
						OnExitCodes: nil,
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePodFailurePolicy(tt.policy)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateExitCodes(t *testing.T) {
	tests := []struct {
		name        string
		exitCodes   *v1alpha1.PodFailurePolicyOnExitCodes
		expectError bool
	}{
		{
			name:        "nil exit codes (valid)",
			exitCodes:   nil,
			expectError: false,
		},
		{
			name: "valid exit codes with In operator",
			exitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "In",
				Values:   []int32{1, 2, 3},
			},
			expectError: false,
		},
		{
			name: "valid exit codes with NotIn operator",
			exitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "NotIn",
				Values:   []int32{0},
			},
			expectError: false,
		},
		{
			name: "invalid operator",
			exitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "Invalid",
				Values:   []int32{1},
			},
			expectError: true,
		},
		{
			name: "empty values",
			exitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "In",
				Values:   []int32{},
			},
			expectError: true,
		},
		{
			name: "exit code out of range - negative",
			exitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "In",
				Values:   []int32{-1},
			},
			expectError: true,
		},
		{
			name: "exit code out of range - too large",
			exitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "In",
				Values:   []int32{256},
			},
			expectError: true,
		},
		{
			name: "exit code at boundary - 0",
			exitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "In",
				Values:   []int32{0},
			},
			expectError: false,
		},
		{
			name: "exit code at boundary - 255",
			exitCodes: &v1alpha1.PodFailurePolicyOnExitCodes{
				Operator: "In",
				Values:   []int32{255},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateExitCodes(tt.exitCodes)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateParameterValueFrom(t *testing.T) {
	tests := []struct {
		name        string
		valueFrom   *v1alpha1.ParameterValueFrom
		paramName   string
		expectError bool
	}{
		{
			name:        "nil valueFrom",
			valueFrom:   nil,
			paramName:   "param1",
			expectError: true,
		},
		{
			name: "valid ConfigMapKeyRef",
			valueFrom: &v1alpha1.ParameterValueFrom{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-configmap",
					},
					Key: "key1",
				},
			},
			paramName:   "param1",
			expectError: false,
		},
		{
			name: "valid SecretKeyRef",
			valueFrom: &v1alpha1.ParameterValueFrom{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "key1",
				},
			},
			paramName:   "param1",
			expectError: false,
		},
		{
			name: "valid JSONPath",
			valueFrom: &v1alpha1.ParameterValueFrom{
				JSONPath: "$.status.succeeded",
			},
			paramName:   "param1",
			expectError: false,
		},
		{
			name: "no source specified",
			valueFrom: &v1alpha1.ParameterValueFrom{},
			paramName:   "param1",
			expectError: true,
		},
		{
			name: "multiple sources specified",
			valueFrom: &v1alpha1.ParameterValueFrom{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-configmap",
					},
					Key: "key1",
				},
				JSONPath: "$.status.succeeded",
			},
			paramName:   "param1",
			expectError: true,
		},
		{
			name: "ConfigMapKeyRef with empty name",
			valueFrom: &v1alpha1.ParameterValueFrom{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "",
					},
					Key: "key1",
				},
			},
			paramName:   "param1",
			expectError: true,
		},
		{
			name: "ConfigMapKeyRef with empty key",
			valueFrom: &v1alpha1.ParameterValueFrom{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-configmap",
					},
					Key: "",
				},
			},
			paramName:   "param1",
			expectError: true,
		},
		{
			name: "SecretKeyRef with empty name",
			valueFrom: &v1alpha1.ParameterValueFrom{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "",
					},
					Key: "key1",
				},
			},
			paramName:   "param1",
			expectError: true,
		},
		{
			name: "SecretKeyRef with empty key",
			valueFrom: &v1alpha1.ParameterValueFrom{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "test-secret",
					},
					Key: "",
				},
			},
			paramName:   "param1",
			expectError: true,
		},
		{
			name: "invalid JSONPath",
			valueFrom: &v1alpha1.ParameterValueFrom{
				JSONPath: "invalid-path",
			},
			paramName:   "param1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateParameterValueFrom(tt.valueFrom, tt.paramName)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateStepInputs(t *testing.T) {
	tests := []struct {
		name        string
		inputs      *v1alpha1.StepInputs
		stepName    string
		expectError bool
	}{
		{
			name:        "nil inputs (valid)",
			inputs:      nil,
			stepName:    "step1",
			expectError: false,
		},
		{
			name: "valid inputs with parameter",
			inputs: &v1alpha1.StepInputs{
				Parameters: []v1alpha1.ParameterInput{
					{
						Name:  "param1",
						Value: "value1",
					},
				},
			},
			stepName:    "step1",
			expectError: false,
		},
		{
			name: "valid inputs with artifact",
			inputs: &v1alpha1.StepInputs{
				Artifacts: []v1alpha1.ArtifactInput{
					{
						Name: "artifact1",
						From: "step1/artifact1",
					},
				},
			},
			stepName:    "step1",
			expectError: false,
		},
		{
			name: "invalid parameter input - empty name",
			inputs: &v1alpha1.StepInputs{
				Parameters: []v1alpha1.ParameterInput{
					{
						Name:  "",
						Value: "value1",
					},
				},
			},
			stepName:    "step1",
			expectError: true,
		},
		{
			name: "invalid parameter input - neither value nor valueFrom",
			inputs: &v1alpha1.StepInputs{
				Parameters: []v1alpha1.ParameterInput{
					{
						Name: "param1",
					},
				},
			},
			stepName:    "step1",
			expectError: true,
		},
		{
			name: "invalid parameter input - both value and valueFrom",
			inputs: &v1alpha1.StepInputs{
				Parameters: []v1alpha1.ParameterInput{
					{
						Name:  "param1",
						Value: "value1",
						ValueFrom: &v1alpha1.ParameterValueFrom{
							JSONPath: "$.status.succeeded",
						},
					},
				},
			},
			stepName:    "step1",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStepInputs(tt.inputs, tt.stepName)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateStepEnhanced_WithOutputs(t *testing.T) {
	step := &v1alpha1.Step{
		Name: "step1",
		Outputs: &v1alpha1.StepOutputs{
			Parameters: []v1alpha1.ParameterOutput{
				{
					Name: "param1",
					ValueFrom: v1alpha1.ParameterValueFrom{
						JSONPath: "$.status.succeeded",
					},
				},
			},
		},
	}

	err := validateStepEnhanced(step, 0)
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}
}

func TestValidateStepEnhanced_WithInvalidOutputs(t *testing.T) {
	step := &v1alpha1.Step{
		Name: "step1",
		Outputs: &v1alpha1.StepOutputs{
			Parameters: []v1alpha1.ParameterOutput{
				{
					Name:      "",
					ValueFrom: v1alpha1.ParameterValueFrom{},
				},
			},
		},
	}

	err := validateStepEnhanced(step, 0)
	if err == nil {
		t.Error("Expected error for invalid outputs but got none")
	}
}

func TestValidateStepTemplate(t *testing.T) {
	tests := []struct {
		name        string
		step        *v1alpha1.Step
		index       int
		expectError bool
	}{
		{
			name: "valid step with template",
			step: &v1alpha1.Step{
				Name: "step1",
				Template: runtime.RawExtension{
					Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
				},
			},
			index:       0,
			expectError: false,
		},
		{
			name: "manual step without template (valid)",
			step: &v1alpha1.Step{
				Name: "step1",
				Type: "Manual",
			},
			index:       0,
			expectError: false,
		},
		{
			name: "job step without template (invalid)",
			step: &v1alpha1.Step{
				Name: "step1",
				Type: "Job",
			},
			index:       0,
			expectError: true,
		},
		{
			name: "empty template (invalid for job step)",
			step: &v1alpha1.Step{
				Name:     "step1",
				Type:     "Job",
				Template: runtime.RawExtension{},
			},
			index:       0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateStepTemplate(tt.step, tt.index)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

