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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

var (
	// ErrInvalidStepTimeout indicates invalid step timeout value.
	ErrInvalidStepTimeout = fmt.Errorf("step timeout must be positive duration (e.g., '30s', '5m')")

	// ErrInvalidRetryPolicy indicates invalid retry policy configuration.
	ErrInvalidRetryPolicy = fmt.Errorf("invalid retry policy")

	// ErrInvalidParameterName indicates invalid parameter name.
	ErrInvalidParameterName = fmt.Errorf("parameter name must be a valid DNS-1123 subdomain")

	// ErrInvalidArtifactPath indicates invalid artifact path.
	ErrInvalidArtifactPath = fmt.Errorf("artifact path must be an absolute path")

	// ErrInvalidWhenCondition indicates invalid when condition.
	ErrInvalidWhenCondition = fmt.Errorf("invalid when condition")

	// ErrInvalidImagePullPolicy indicates invalid image pull policy.
	ErrInvalidImagePullPolicy = fmt.Errorf("invalid imagePullPolicy (must be Always, IfNotPresent, or Never)")

	// ErrInvalidRestartPolicy indicates invalid restart policy.
	ErrInvalidRestartPolicy = fmt.Errorf("invalid restartPolicy (must be Always, OnFailure, or Never)")
)

// validateStepEnhanced performs enhanced validation on a step including timeouts, retries, and inputs/outputs.
func validateStepEnhanced(step *v1alpha1.Step, index int) error {
	// Validate step name format (already done in validateSteps, but double-check)
	if err := ValidateStepName(step.Name); err != nil {
		return fmt.Errorf("step at index %d: %w", index, err)
	}

	// Validate timeout if specified (TimeoutSeconds is int64, not string)
	if step.TimeoutSeconds != nil && *step.TimeoutSeconds > 0 {
		// TimeoutSeconds is already validated as positive in basic validation
		// Additional validation: reasonable maximum (30 days = 2592000 seconds)
		maxTimeout := int64(30 * 24 * 3600)
		if *step.TimeoutSeconds > maxTimeout {
			return fmt.Errorf("step %q: timeoutSeconds %d exceeds maximum allowed duration of 30 days", step.Name, *step.TimeoutSeconds)
		}
	}

	// Validate retry policy if specified
	if step.RetryPolicy != nil {
		if err := validateRetryPolicy(step.RetryPolicy); err != nil {
			return fmt.Errorf("step %q: %w", step.Name, err)
		}
	}

	// Validate inputs
	if step.Inputs != nil {
		if err := validateStepInputs(step.Inputs, step.Name); err != nil {
			return fmt.Errorf("step %q: %w", step.Name, err)
		}
	}

	// Validate outputs
	if step.Outputs != nil {
		if err := validateStepOutputs(step.Outputs, step.Name); err != nil {
			return fmt.Errorf("step %q: %w", step.Name, err)
		}
	}

	// Validate when condition if specified
	if step.When != nil && *step.When != "" {
		if err := validateWhenCondition(*step.When); err != nil {
			return fmt.Errorf("step %q: %w", step.Name, err)
		}
	}

	return nil
}

// validateTimeout validates a timeout duration string.
func validateTimeout(timeout string) error {
	if timeout == "" {
		return nil // Empty timeout is valid (no timeout)
	}

	duration, err := time.ParseDuration(timeout)
	if err != nil {
		return fmt.Errorf("%w: %q is not a valid duration: %v", ErrInvalidStepTimeout, timeout, err)
	}

	if duration <= 0 {
		return fmt.Errorf("%w: %q must be positive", ErrInvalidStepTimeout, timeout)
	}

	// Reasonable maximum timeout (30 days)
	maxTimeout := 30 * 24 * time.Hour
	if duration > maxTimeout {
		return fmt.Errorf("timeout %q exceeds maximum allowed duration of 30 days", timeout)
	}

	return nil
}

// validateRetryPolicy validates a retry policy.
func validateRetryPolicy(policy *v1alpha1.RetryPolicy) error {
	if policy == nil {
		return nil
	}

	// Validate retry limit (Limit is int32, not pointer)
	if policy.Limit < 0 {
		return fmt.Errorf("%w: limit must be non-negative, got %d", ErrInvalidRetryPolicy, policy.Limit)
	}

	// Validate backoff
	if policy.Backoff != nil {
		// Validate backoff type
		validTypes := map[string]bool{
			"Exponential": true,
			"Linear":      true,
			"Fixed":       true,
		}
		if policy.Backoff.Type != "" && !validTypes[policy.Backoff.Type] {
			return fmt.Errorf("%w: invalid backoff.type %q (must be Exponential, Linear, or Fixed)", ErrInvalidRetryPolicy, policy.Backoff.Type)
		}

		// Validate base duration
		if policy.Backoff.Duration != "" {
			if err := validateTimeout(policy.Backoff.Duration); err != nil {
				return fmt.Errorf("%w: invalid backoff.duration: %w", ErrInvalidRetryPolicy, err)
			}
		}

		// Validate factor (must be >= 1.0)
		if policy.Backoff.Factor != nil && *policy.Backoff.Factor < 1.0 {
			return fmt.Errorf("%w: backoff.factor must be >= 1.0, got %f", ErrInvalidRetryPolicy, *policy.Backoff.Factor)
		}
	}

	return nil
}

// validateStepInputs validates step inputs including parameters and artifacts.
func validateStepInputs(inputs *v1alpha1.StepInputs, stepName string) error {
	if inputs == nil {
		return nil
	}

	// Validate parameters
	for i, param := range inputs.Parameters {
		if err := validateParameterInput(&param, stepName, i); err != nil {
			return fmt.Errorf("input parameter at index %d: %w", i, err)
		}
	}

	// Validate artifacts
	for i, artifact := range inputs.Artifacts {
		if err := validateArtifactInput(&artifact, stepName, i); err != nil {
			return fmt.Errorf("input artifact at index %d: %w", i, err)
		}
	}

	return nil
}

// validateStepOutputs validates step outputs including parameters and artifacts.
func validateStepOutputs(outputs *v1alpha1.StepOutputs, stepName string) error {
	if outputs == nil {
		return nil
	}

	// Validate parameters
	for i, param := range outputs.Parameters {
		if err := validateParameterOutput(&param, stepName, i); err != nil {
			return fmt.Errorf("output parameter at index %d: %w", i, err)
		}
	}

	// Validate artifacts
	for i, artifact := range outputs.Artifacts {
		if err := validateArtifactOutput(&artifact, stepName, i); err != nil {
			return fmt.Errorf("output artifact at index %d: %w", i, err)
		}
	}

	return nil
}

// validateParameterInput validates a parameter input.
func validateParameterInput(param *v1alpha1.ParameterInput, stepName string, index int) error {
	if param.Name == "" {
		return fmt.Errorf("parameter name cannot be empty")
	}

	// Validate parameter name format
	if errs := validation.IsDNS1123Subdomain(param.Name); len(errs) > 0 {
		return fmt.Errorf("%w: %q: %v", ErrInvalidParameterName, param.Name, errs)
	}

	// Either Value or ValueFrom must be specified
	if param.Value == "" && param.ValueFrom == nil {
		return fmt.Errorf("parameter %q: either value or valueFrom must be specified", param.Name)
	}

	// Both Value and ValueFrom cannot be specified
	if param.Value != "" && param.ValueFrom != nil {
		return fmt.Errorf("parameter %q: cannot specify both value and valueFrom", param.Name)
	}

	// Validate ValueFrom if specified
	if param.ValueFrom != nil {
		if err := validateParameterValueFrom(param.ValueFrom, param.Name); err != nil {
			return fmt.Errorf("parameter %q: %w", param.Name, err)
		}
	}

	return nil
}

// validateParameterOutput validates a parameter output.
func validateParameterOutput(param *v1alpha1.ParameterOutput, stepName string, index int) error {
	if param.Name == "" {
		return fmt.Errorf("parameter name cannot be empty")
	}

	// Validate parameter name format
	if errs := validation.IsDNS1123Subdomain(param.Name); len(errs) > 0 {
		return fmt.Errorf("%w: %q: %v", ErrInvalidParameterName, param.Name, errs)
	}

	// ValueFrom must be specified for outputs
	if param.ValueFrom.JSONPath == "" {
		return fmt.Errorf("parameter %q: valueFrom.jsonPath must be specified for output parameters", param.Name)
	}

	// Validate JSONPath expression
	if err := validateJSONPath(param.ValueFrom.JSONPath); err != nil {
		return fmt.Errorf("parameter %q: invalid JSONPath: %w", param.Name, err)
	}

	return nil
}

// validateParameterValueFrom validates a parameter value source.
func validateParameterValueFrom(valueFrom *v1alpha1.ParameterValueFrom, paramName string) error {
	if valueFrom == nil {
		return fmt.Errorf("valueFrom cannot be nil")
	}

	// At least one source must be specified
	sources := 0
	if valueFrom.ConfigMapKeyRef != nil {
		sources++
	}
	if valueFrom.SecretKeyRef != nil {
		sources++
	}
	if valueFrom.JSONPath != "" {
		sources++
	}

	if sources == 0 {
		return fmt.Errorf("at least one of configMapKeyRef, secretKeyRef, or jsonPath must be specified")
	}

	if sources > 1 {
		return fmt.Errorf("only one of configMapKeyRef, secretKeyRef, or jsonPath can be specified")
	}

	// Validate ConfigMapKeyRef
	if valueFrom.ConfigMapKeyRef != nil {
		if valueFrom.ConfigMapKeyRef.Name == "" {
			return fmt.Errorf("configMapKeyRef.name cannot be empty")
		}
		if valueFrom.ConfigMapKeyRef.Key == "" {
			return fmt.Errorf("configMapKeyRef.key cannot be empty")
		}
	}

	// Validate SecretKeyRef
	if valueFrom.SecretKeyRef != nil {
		if valueFrom.SecretKeyRef.Name == "" {
			return fmt.Errorf("secretKeyRef.name cannot be empty")
		}
		if valueFrom.SecretKeyRef.Key == "" {
			return fmt.Errorf("secretKeyRef.key cannot be empty")
		}
	}

	// Validate JSONPath
	if valueFrom.JSONPath != "" {
		if err := validateJSONPath(valueFrom.JSONPath); err != nil {
			return fmt.Errorf("invalid JSONPath: %w", err)
		}
	}

	return nil
}

// validateJSONPath validates a JSONPath expression.
func validateJSONPath(jsonPath string) error {
	if jsonPath == "" {
		return fmt.Errorf("JSONPath cannot be empty")
	}

	// Basic validation: must start with $ or contain step name prefix
	if !strings.HasPrefix(jsonPath, "$") && !strings.Contains(jsonPath, ":") {
		return fmt.Errorf("JSONPath must start with $ or contain step name prefix (e.g., 'step1:$.status.succeeded')")
	}

	// If it contains a colon, validate the step name prefix
	if strings.Contains(jsonPath, ":") {
		parts := strings.SplitN(jsonPath, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid JSONPath format with step name: expected 'stepName:$.path'")
		}
		stepName := parts[0]
		if errs := validation.IsDNS1123Subdomain(stepName); len(errs) > 0 {
			return fmt.Errorf("invalid step name in JSONPath: %q: %v", stepName, errs)
		}
		jsonPath = parts[1]
	}

	// Validate JSONPath starts with $
	if !strings.HasPrefix(jsonPath, "$") {
		return fmt.Errorf("JSONPath must start with $")
	}

	return nil
}

// validateArtifactInput validates an artifact input.
func validateArtifactInput(artifact *v1alpha1.ArtifactInput, stepName string, index int) error {
	if artifact.Name == "" {
		return fmt.Errorf("artifact name cannot be empty")
	}

	// Validate artifact name format
	if errs := validation.IsDNS1123Subdomain(artifact.Name); len(errs) > 0 {
		return fmt.Errorf("invalid artifact name %q: %v", artifact.Name, errs)
	}

	// Validate path (must be absolute)
	if artifact.Path != "" {
		if err := validateArtifactPath(artifact.Path); err != nil {
			return fmt.Errorf("artifact %q: %w", artifact.Name, err)
		}
	}

	// Either From or HTTP must be specified (From is the step name, HTTP is HTTP source)
	if artifact.From == "" && artifact.HTTP == nil {
		return fmt.Errorf("artifact %q: either from or http must be specified", artifact.Name)
	}

	// Both From and HTTP cannot be specified
	if artifact.From != "" && artifact.HTTP != nil {
		return fmt.Errorf("artifact %q: cannot specify both from and http", artifact.Name)
	}

	// Validate From step name format
	if artifact.From != "" {
		if errs := validation.IsDNS1123Subdomain(artifact.From); len(errs) > 0 {
			return fmt.Errorf("artifact %q: invalid from step name %q: %v", artifact.Name, artifact.From, errs)
		}
	}

	// Validate HTTP if specified
	if artifact.HTTP != nil {
		if artifact.HTTP.URL == "" {
			return fmt.Errorf("artifact %q: http.url cannot be empty", artifact.Name)
		}
		// Basic URL validation (must start with http:// or https://)
		if !strings.HasPrefix(artifact.HTTP.URL, "http://") && !strings.HasPrefix(artifact.HTTP.URL, "https://") {
			return fmt.Errorf("artifact %q: http.url must start with http:// or https://", artifact.Name)
		}
	}

	return nil
}

// validateArtifactOutput validates an artifact output.
func validateArtifactOutput(artifact *v1alpha1.ArtifactOutput, stepName string, index int) error {
	if artifact.Name == "" {
		return fmt.Errorf("artifact name cannot be empty")
	}

	// Validate artifact name format
	if errs := validation.IsDNS1123Subdomain(artifact.Name); len(errs) > 0 {
		return fmt.Errorf("invalid artifact name %q: %v", artifact.Name, errs)
	}

	// Validate path (must be absolute, and Path is required for outputs)
	if artifact.Path == "" {
		return fmt.Errorf("artifact %q: path is required for output artifacts", artifact.Name)
	}
	if err := validateArtifactPath(artifact.Path); err != nil {
		return fmt.Errorf("artifact %q: %w", artifact.Name, err)
	}

	// Validate archive config if specified
	if artifact.Archive != nil {
		if err := validateArchiveConfig(artifact.Archive, artifact.Name); err != nil {
			return fmt.Errorf("artifact %q: %w", artifact.Name, err)
		}
	}

	return nil
}

// validateArtifactPath validates an artifact path.
func validateArtifactPath(path string) error {
	if path == "" {
		return nil // Empty path is valid (default path will be used)
	}

	// Path must be absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%w: %q (must start with /)", ErrInvalidArtifactPath, path)
	}

	// Path should not contain ".." for security
	if strings.Contains(path, "..") {
		return fmt.Errorf("%w: %q (cannot contain '..')", ErrInvalidArtifactPath, path)
	}

	return nil
}

// validateArchiveConfig validates an archive configuration.
func validateArchiveConfig(archive *v1alpha1.ArchiveConfig, artifactName string) error {
	if archive == nil {
		return nil
	}

	// Validate format
	validFormats := map[string]bool{
		"tar": true,
		"zip": true,
	}
	if archive.Format != "" && !validFormats[archive.Format] {
		return fmt.Errorf("invalid archive format %q (must be 'tar' or 'zip')", archive.Format)
	}

	// Validate compression
	validCompressions := map[string]bool{
		"none": true,
		"gzip": true,
	}
	if archive.Compression != "" && !validCompressions[archive.Compression] {
		return fmt.Errorf("invalid compression %q (must be 'none' or 'gzip')", archive.Compression)
	}

	return nil
}

// validateWhenCondition validates a when condition.
func validateWhenCondition(when string) error {
	if when == "" {
		return nil // Empty condition is valid (always execute)
	}

	// Basic validation: known keywords or template expression
	validKeywords := map[string]bool{
		"always": true,
		"never":  true,
		"true":   true,
		"false":  true,
	}

	whenTrimmed := strings.TrimSpace(when)
	if validKeywords[whenTrimmed] {
		return nil
	}

	// For template expressions, basic validation
	// Must contain template markers or be a valid expression
	if strings.Contains(whenTrimmed, "{{") && strings.Contains(whenTrimmed, "}}") {
		// Basic template validation - check for balanced braces
		openBraces := strings.Count(whenTrimmed, "{{")
		closeBraces := strings.Count(whenTrimmed, "}}")
		if openBraces != closeBraces {
			return fmt.Errorf("%w: unbalanced template braces in %q", ErrInvalidWhenCondition, when)
		}
		return nil
	}

	// If it's not a keyword and not a template, it might be invalid
	// But we allow it for now (will be evaluated at runtime)
	return nil
}

// validateExecutionPolicyEnhanced performs enhanced validation on execution policy.
func validateExecutionPolicyEnhanced(policy *v1alpha1.ExecutionPolicy) error {
	if policy == nil {
		return nil
	}

	// Use existing validation
	if err := validateExecutionPolicy(policy); err != nil {
		return err
	}

	// Additional validations

	// Validate pod failure policy if specified
	if policy.PodFailurePolicy != nil {
		if err := validatePodFailurePolicy(policy.PodFailurePolicy); err != nil {
			return fmt.Errorf("invalid podFailurePolicy: %w", err)
		}
	}

	return nil
}

// validatePodFailurePolicy validates a pod failure policy.
func validatePodFailurePolicy(policy *v1alpha1.PodFailurePolicy) error {
	if policy == nil {
		return nil
	}

	if len(policy.Rules) == 0 {
		return fmt.Errorf("podFailurePolicy must have at least one rule")
	}

	for i, rule := range policy.Rules {
		// Validate action (valid values: FailJob, Ignore, Count)
		validActions := map[string]bool{
			"FailJob": true,
			"Ignore":  true,
			"Count":   true,
		}
		if !validActions[rule.Action] {
			return fmt.Errorf("rule at index %d: invalid action %q (must be FailJob, Ignore, or Count)", i, rule.Action)
		}

		// At least one condition must be specified
		if rule.OnExitCodes == nil && rule.OnPodConditions == nil {
			return fmt.Errorf("rule at index %d: at least one of onExitCodes or onPodConditions must be specified", i)
		}

		// Validate OnExitCodes if specified
		if rule.OnExitCodes != nil {
			if err := validateExitCodes(rule.OnExitCodes); err != nil {
				return fmt.Errorf("rule at index %d: %w", i, err)
			}
		}
	}

	return nil
}

// validateExitCodes validates exit code conditions.
func validateExitCodes(exitCodes *v1alpha1.PodFailurePolicyOnExitCodes) error {
	if exitCodes == nil {
		return nil
	}

	validOperators := map[string]bool{
		"In":    true,
		"NotIn": true,
	}
	if !validOperators[exitCodes.Operator] {
		return fmt.Errorf("invalid operator %q (must be In or NotIn)", exitCodes.Operator)
	}

	if len(exitCodes.Values) == 0 {
		return fmt.Errorf("values cannot be empty")
	}

	// Validate exit code values are in valid range (0-255)
	for _, code := range exitCodes.Values {
		if code < 0 || code > 255 {
			return fmt.Errorf("exit code %d is out of valid range (0-255)", code)
		}
	}

	return nil
}

// ValidateJobFlowEnhanced performs enhanced validation including all edge cases.
func ValidateJobFlowEnhanced(jobFlow *v1alpha1.JobFlow) error {
	// Use existing validation first
	if err := ValidateJobFlow(jobFlow); err != nil {
		return err
	}

	// Enhanced validations

	// Validate each step with enhanced checks
	for i, step := range jobFlow.Spec.Steps {
		if err := validateStepEnhanced(&step, i); err != nil {
			return err
		}
	}

	// Enhanced execution policy validation
	if jobFlow.Spec.ExecutionPolicy != nil {
		if err := validateExecutionPolicyEnhanced(jobFlow.Spec.ExecutionPolicy); err != nil {
			return err
		}
	}

	return nil
}

