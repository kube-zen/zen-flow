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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	jferrors "github.com/kube-zen/zen-flow/pkg/errors"
	sdklog "github.com/kube-zen/zen-sdk/pkg/logging"
)

// ParameterContext provides context for parameter template substitution
type ParameterContext struct {
	Parameters map[string]string // parameter name -> value
	JobFlow    *v1alpha1.JobFlow
	StepName   string
}

// applyParametersToJobTemplate applies resolved parameters to a job template
func (r *JobFlowReconciler) applyParametersToJobTemplate(ctx context.Context, jobFlow *v1alpha1.JobFlow, step *v1alpha1.Step, job *batchv1.Job, resolvedParams map[string]string) error {
	if len(resolvedParams) == 0 {
		return nil // No parameters to apply
	}

	logger := sdklog.NewLogger("zen-flow-controller")

	// Build parameter context
	paramCtx := &ParameterContext{
		Parameters: resolvedParams,
		JobFlow:    jobFlow,
		StepName:   step.Name,
	}

	// Apply parameters to job spec using template substitution
	// We'll serialize the job spec, apply template, then deserialize
	jobSpecJSON, err := json.Marshal(job.Spec)
	if err != nil {
		return jferrors.Wrapf(err, "json_marshal_failed", "failed to marshal job spec for parameter substitution")
	}

	// Create template from job spec JSON
	tmpl, err := template.New("jobSpec").Parse(string(jobSpecJSON))
	if err != nil {
		// If template parsing fails, try direct substitution on string
		return r.applyParametersToString(job, resolvedParams)
	}

	// Execute template with parameters
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, paramCtx); err != nil {
		// If template execution fails, try direct substitution
		logger.Debug("Template execution failed, using direct substitution", sdklog.Error(err))
		return r.applyParametersToString(job, resolvedParams)
	}

	// Deserialize back to job spec
	var newSpec batchv1.JobSpec
	if err := json.Unmarshal(buf.Bytes(), &newSpec); err != nil {
		// If deserialization fails, try direct substitution
		logger.Debug("Failed to deserialize templated spec, using direct substitution", sdklog.Error(err))
		return r.applyParametersToString(job, resolvedParams)
	}

	job.Spec = newSpec
	return nil
}

// applyParametersToString applies parameters using string replacement
// This is a fallback when template parsing fails
func (r *JobFlowReconciler) applyParametersToString(job *batchv1.Job, resolvedParams map[string]string) error {
	// Convert job to JSON string
	jobJSON, err := json.Marshal(job)
	if err != nil {
		return jferrors.Wrapf(err, "json_marshal_failed", "failed to marshal job for parameter substitution")
	}

	jobStr := string(jobJSON)

	// Replace parameter placeholders: {{.parameters.paramName}} or {{parameters.paramName}}
	for paramName, paramValue := range resolvedParams {
		// Replace {{.parameters.paramName}}
		oldStr := fmt.Sprintf("{{.parameters.%s}}", paramName)
		jobStr = string(bytes.ReplaceAll([]byte(jobStr), []byte(oldStr), []byte(paramValue)))
		// Replace {{parameters.paramName}}
		oldStr2 := fmt.Sprintf("{{parameters.%s}}", paramName)
		jobStr = string(bytes.ReplaceAll([]byte(jobStr), []byte(oldStr2), []byte(paramValue)))
		// Replace ${parameters.paramName}
		oldStr3 := fmt.Sprintf("${parameters.%s}", paramName)
		jobStr = string(bytes.ReplaceAll([]byte(jobStr), []byte(oldStr3), []byte(paramValue)))
	}

	// Deserialize back to job
	var newJob batchv1.Job
	if err := json.Unmarshal([]byte(jobStr), &newJob); err != nil {
		return jferrors.Wrapf(err, "json_unmarshal_failed", "failed to unmarshal job after parameter substitution")
	}

	// Update job spec and metadata
	job.Spec = newJob.Spec
	// Preserve name, namespace, labels, etc. that we set
	// Only update spec

	return nil
}

// applyParametersToContainerArgs applies parameters to container command/args
func (r *JobFlowReconciler) applyParametersToContainerArgs(containers []corev1.Container, resolvedParams map[string]string) {
	for i := range containers {
		// Apply to command
		for j := range containers[i].Command {
			containers[i].Command[j] = r.substituteParameter(containers[i].Command[j], resolvedParams)
		}
		// Apply to args
		for j := range containers[i].Args {
			containers[i].Args[j] = r.substituteParameter(containers[i].Args[j], resolvedParams)
		}
		// Apply to env vars
		for j := range containers[i].Env {
			if containers[i].Env[j].Value != "" {
				containers[i].Env[j].Value = r.substituteParameter(containers[i].Env[j].Value, resolvedParams)
			}
		}
	}
}

// substituteParameter substitutes parameter placeholders in a string
func (r *JobFlowReconciler) substituteParameter(str string, resolvedParams map[string]string) string {
	result := str
	for paramName, paramValue := range resolvedParams {
		// Replace {{.parameters.paramName}}
		result = string(bytes.ReplaceAll([]byte(result), []byte(fmt.Sprintf("{{.parameters.%s}}", paramName)), []byte(paramValue)))
		// Replace {{parameters.paramName}}
		result = string(bytes.ReplaceAll([]byte(result), []byte(fmt.Sprintf("{{parameters.%s}}", paramName)), []byte(paramValue)))
		// Replace ${parameters.paramName}
		result = string(bytes.ReplaceAll([]byte(result), []byte(fmt.Sprintf("${parameters.%s}", paramName)), []byte(paramValue)))
	}
	return result
}
