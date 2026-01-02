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
	"fmt"
	"strings"
	"text/template"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

// TemplateContext provides context for template evaluation
type TemplateContext struct {
	JobFlow    *v1alpha1.JobFlow
	StepStatus map[string]*v1alpha1.StepStatus // step name -> status
}

// evaluateTemplate evaluates a template string with access to JobFlow and step statuses
func (r *JobFlowReconciler) evaluateTemplate(tmplStr string, ctx *TemplateContext) (string, error) {
	if tmplStr == "" {
		return "", nil
	}

	// Parse template
	tmpl, err := template.New("condition").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// evaluateWhenCondition evaluates a "When" condition with template support
func (r *JobFlowReconciler) evaluateWhenCondition(jobFlow *v1alpha1.JobFlow, condition string) (bool, error) {
	if condition == "" {
		return true, nil
	}

	// Basic keyword-based evaluation
	if condition == "always" || condition == "true" {
		return true, nil
	}
	if condition == "never" || condition == "false" {
		return false, nil
	}

	// Build template context with step statuses
	ctx := &TemplateContext{
		JobFlow:    jobFlow,
		StepStatus: make(map[string]*v1alpha1.StepStatus),
	}

	// Populate step statuses
	for i := range jobFlow.Status.Steps {
		stepStatus := &jobFlow.Status.Steps[i]
		ctx.StepStatus[stepStatus.Name] = stepStatus
	}

	// Try template evaluation
	result, err := r.evaluateTemplate(condition, ctx)
	if err != nil {
		// If template evaluation fails, try simple string matching
		// Check for step status patterns like "steps.step1.phase == 'Succeeded'"
		return r.evaluateSimpleCondition(condition, ctx), nil
	}

	// Parse boolean result
	result = strings.TrimSpace(strings.ToLower(result))
	if result == "true" || result == "1" {
		return true, nil
	}
	if result == "false" || result == "0" {
		return false, nil
	}
	
	// If template evaluation succeeds but doesn't produce a clear boolean,
	// fall back to simple condition evaluation
	return r.evaluateSimpleCondition(condition, ctx), nil
}

// evaluateSimpleCondition evaluates simple condition patterns without full template engine
func (r *JobFlowReconciler) evaluateSimpleCondition(condition string, ctx *TemplateContext) bool {
	// Simple pattern matching for common cases
	// Example: "steps.step1.phase == 'Succeeded'"
	condition = strings.TrimSpace(condition)

	// Check for step phase patterns
	if strings.Contains(condition, "steps.") && strings.Contains(condition, ".phase") {
		// Extract step name and phase
		parts := strings.Split(condition, ".")
		if len(parts) >= 3 {
			stepName := parts[1]
			if stepStatus, exists := ctx.StepStatus[stepName]; exists {
				// Check for equality patterns
				if strings.Contains(condition, "== 'Succeeded'") || strings.Contains(condition, "== \"Succeeded\"") {
					return stepStatus.Phase == v1alpha1.StepPhaseSucceeded
				}
				if strings.Contains(condition, "== 'Failed'") || strings.Contains(condition, "== \"Failed\"") {
					return stepStatus.Phase == v1alpha1.StepPhaseFailed
				}
				if strings.Contains(condition, "!= 'Failed'") || strings.Contains(condition, "!= \"Failed\"") {
					return stepStatus.Phase != v1alpha1.StepPhaseFailed
				}
			}
		}
	}

	// Default to true if condition exists but doesn't match patterns
	return true
}

