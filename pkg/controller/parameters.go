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
	"strings"

	"github.com/PaesslerAG/jsonpath"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	jferrors "github.com/kube-zen/zen-flow/pkg/errors"
	sdklog "github.com/kube-zen/zen-sdk/pkg/logging"
)

// resolveParameter resolves a parameter value from various sources
func (r *JobFlowReconciler) resolveParameter(ctx context.Context, jobFlow *v1alpha1.JobFlow, param *v1alpha1.ParameterInput) (string, error) {
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
		// JSONPath format: "stepName:$.jsonpath" or "$.jsonpath" (from current step)
		// Check if step name is explicitly specified
		stepName, jsonPathExpr := r.parseJSONPathWithStepName(valueFrom.JSONPath)
		if stepName != "" {
			// Step name is specified, extract from that step's job output
			return r.extractParameterFromJobOutput(ctx, jobFlow, stepName, jsonPathExpr)
		}
		// No step name specified - this requires context about which step we're in
		// For now, return error asking for explicit step name
		return "", jferrors.New("jsonpath_step_not_specified", "JSONPath parameter requires step name in format 'stepName:$.jsonpath' - use extractParameterFromJobOutput instead")
	}

	return "", jferrors.New("parameter_source_invalid", fmt.Sprintf("parameter %s has invalid valueFrom configuration", param.Name))
}

// parseJSONPathWithStepName parses a JSONPath string that may include a step name prefix.
// Format: "stepName:$.jsonpath" or "$.jsonpath"
// Returns: (stepName, jsonPathExpr)
// If no step name is found, returns ("", originalString)
func (r *JobFlowReconciler) parseJSONPathWithStepName(jsonPathStr string) (string, string) {
	// Check if the string contains a colon followed by $ (indicating stepName:$.jsonpath format)
	// We look for the pattern: stepName:$.jsonpath
	colonIndex := strings.Index(jsonPathStr, ":")
	if colonIndex > 0 && colonIndex < len(jsonPathStr)-1 {
		// Check if the part after colon starts with $ (JSONPath indicator)
		afterColon := jsonPathStr[colonIndex+1:]
		if strings.HasPrefix(strings.TrimSpace(afterColon), "$") {
			// Format is "stepName:$.jsonpath"
			stepName := strings.TrimSpace(jsonPathStr[:colonIndex])
			jsonPathExpr := strings.TrimSpace(afterColon)
			return stepName, jsonPathExpr
		}
	}
	// No step name prefix found, return empty step name and original string
	return "", jsonPathStr
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
		key = DefaultConfigMapKey
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

	// Extract parameter from job status using JSONPath
	// Full JSONPath evaluation is implemented via evaluateJSONPath function
	logger.Debug("Extracting parameter from job output using JSONPath",
		sdklog.String("step", stepName),
		sdklog.String("jsonpath", jsonPath))

	// Convert job status to JSON and apply JSONPath
	jobJSON, err := json.Marshal(job.Status)
	if err != nil {
		return "", jferrors.Wrapf(err, "json_marshal_failed", "failed to marshal job status to JSON")
	}

	// Use JSONPath library for full evaluation
	result, err := r.evaluateJSONPath(string(jobJSON), jsonPath)
	if err != nil {
		return "", jferrors.Wrapf(err, "jsonpath_evaluation_failed", "failed to evaluate JSONPath %s", jsonPath)
	}

	return result, nil
}

// evaluateJSONPath evaluates JSONPath expressions using github.com/PaesslerAG/jsonpath
func (r *JobFlowReconciler) evaluateJSONPath(jsonData, jsonPathExpr string) (string, error) {
	// Parse JSON
	var data interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return "", fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Evaluate JSONPath using Get function
	result, err := jsonpath.Get(jsonPathExpr, data)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate JSONPath %s: %w", jsonPathExpr, err)
	}

	// Convert result to string
	var resultStr string
	switch v := result.(type) {
	case string:
		resultStr = v
	case float64:
		resultStr = fmt.Sprintf("%.0f", v)
	case bool:
		resultStr = fmt.Sprintf("%t", v)
	case nil:
		resultStr = ""
	default:
		// For complex types, marshal to JSON
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSONPath result: %w", err)
		}
		resultStr = string(jsonBytes)
	}

	return resultStr, nil
}
