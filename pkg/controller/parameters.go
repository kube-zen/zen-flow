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
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	jferrors "github.com/kube-zen/zen-flow/pkg/errors"
	sdklog "github.com/kube-zen/zen-sdk/pkg/logging"
)

// resolveParameter resolves a parameter value from various sources
func (r *JobFlowReconciler) resolveParameter(ctx context.Context, jobFlow *v1alpha1.JobFlow, param *v1alpha1.ParameterInput) (string, error) {
	logger := sdklog.NewLogger("zen-flow-controller")

	// If value is directly provided, use it
	if param.Value != "" {
		return param.Value, nil
	}

	// If valueFrom is specified, resolve from source
	if param.ValueFrom == nil {
		return "", jferrors.New("parameter_value_missing", fmt.Sprintf("parameter %s has no value or valueFrom", param.Name))
	}

	valueFrom := param.ValueFrom

	// Resolve from ConfigMap
	if valueFrom.ConfigMapKeyRef != nil {
		return r.resolveParameterFromConfigMap(ctx, jobFlow.Namespace, valueFrom.ConfigMapKeyRef)
	}

	// Resolve from Secret
	if valueFrom.SecretKeyRef != nil {
		return r.resolveParameterFromSecret(ctx, jobFlow.Namespace, valueFrom.SecretKeyRef)
	}

	// Resolve from JSONPath (from previous step outputs)
	if valueFrom.JSONPath != "" {
		// TODO: Implement JSONPath extraction from step outputs
		// For now, return placeholder
		logger.Debug("JSONPath parameter resolution not yet fully implemented",
			sdklog.String("parameter_name", param.Name),
			sdklog.String("jsonpath", valueFrom.JSONPath))
		return "", jferrors.New("jsonpath_not_implemented", "JSONPath parameter resolution not yet implemented")
	}

	return "", jferrors.New("parameter_source_invalid", fmt.Sprintf("parameter %s has invalid valueFrom configuration", param.Name))
}

// resolveParameterFromConfigMap resolves a parameter from a ConfigMap
func (r *JobFlowReconciler) resolveParameterFromConfigMap(ctx context.Context, namespace string, configMapRef *corev1.ConfigMapKeySelector) (string, error) {
	configMap := &corev1.ConfigMap{}
	configMapKey := types.NamespacedName{Namespace: namespace, Name: configMapRef.Name}
	if err := r.Get(ctx, configMapKey, configMap); err != nil {
		if k8serrors.IsNotFound(err) {
			return "", jferrors.Wrapf(err, "configmap_not_found", "ConfigMap %s not found", configMapRef.Name)
		}
		return "", jferrors.Wrapf(err, "configmap_get_failed", "failed to get ConfigMap %s", configMapRef.Name)
	}

	key := configMapRef.Key
	if key == "" {
		key = "value" // Default key name
	}

	value, exists := configMap.Data[key]
	if !exists {
		// Try binary data
		if binaryValue, exists := configMap.BinaryData[key]; exists {
			return string(binaryValue), nil
		}
		return "", jferrors.New("configmap_key_not_found", fmt.Sprintf("key %s not found in ConfigMap %s", key, configMapRef.Name))
	}

	return value, nil
}

// resolveParameterFromSecret resolves a parameter from a Secret
func (r *JobFlowReconciler) resolveParameterFromSecret(ctx context.Context, namespace string, secretRef *corev1.SecretKeySelector) (string, error) {
	return r.getSecretValue(ctx, namespace, secretRef)
}

// extractParameterFromJobOutput extracts a parameter value from job output using JSONPath
func (r *JobFlowReconciler) extractParameterFromJobOutput(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepName string, jsonPath string) (string, error) {
	logger := sdklog.NewLogger("zen-flow-controller")

	// Get step status
	stepStatus := r.getStepStatus(jobFlow.Status, stepName)
	if stepStatus == nil {
		return "", jferrors.WithStep(jferrors.WithJobFlow(
			jferrors.New("step_not_found", fmt.Sprintf("step %s not found", stepName)),
			jobFlow.Namespace, jobFlow.Name), stepName)
	}

	// Get the Job
	if stepStatus.JobRef == nil {
		return "", jferrors.WithStep(jferrors.WithJobFlow(
			jferrors.New("job_ref_missing", fmt.Sprintf("step %s has no Job reference", stepName)),
			jobFlow.Namespace, jobFlow.Name), stepName)
	}

	job, err := r.getJob(ctx, types.NamespacedName{
		Namespace: jobFlow.Namespace,
		Name:      stepStatus.JobRef.Name,
	})
	if err != nil {
		return "", jferrors.Wrapf(err, "job_get_failed", "failed to get Job for step %s", stepName)
	}

	// Get job output (from logs or status)
	// In a real implementation, this would:
	// 1. Get job logs or status
	// 2. Parse as JSON
	// 3. Apply JSONPath expression
	// 4. Extract value

	// For now, we'll use a simple implementation that checks job status
	// TODO: Implement full JSONPath evaluation
	logger.Debug("Extracting parameter from job output using JSONPath",
		sdklog.String("step", stepName),
		sdklog.String("jsonpath", jsonPath))

	// Placeholder: convert job status to JSON and apply JSONPath
	jobJSON, err := json.Marshal(job.Status)
	if err != nil {
		return "", jferrors.Wrapf(err, "json_marshal_failed", "failed to marshal job status to JSON")
	}

	// Simple JSONPath implementation for common patterns
	// Full implementation would use a JSONPath library like github.com/PaesslerAG/jsonpath
	result, err := r.evaluateSimpleJSONPath(string(jobJSON), jsonPath)
	if err != nil {
		return "", jferrors.Wrapf(err, "jsonpath_evaluation_failed", "failed to evaluate JSONPath %s", jsonPath)
	}

	return result, nil
}

// evaluateSimpleJSONPath evaluates simple JSONPath expressions
// This is a basic implementation - for full JSONPath support, use a library
func (r *JobFlowReconciler) evaluateSimpleJSONPath(jsonData, jsonPath string) (string, error) {
	// Parse JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Simple JSONPath implementation for common patterns
	// Full implementation would use github.com/PaesslerAG/jsonpath or similar
	// For now, support simple paths like "$.succeeded" or "$.status.succeeded"
	if jsonPath == "$.succeeded" {
		if m, ok := data.(map[string]interface{}); ok {
			if succeeded, ok := m["succeeded"].(float64); ok {
				return fmt.Sprintf("%.0f", succeeded), nil
			}
		}
	}

	// TODO: Implement full JSONPath evaluation using a library
	return "", fmt.Errorf("JSONPath %s not yet fully supported - requires JSONPath library", jsonPath)
}

