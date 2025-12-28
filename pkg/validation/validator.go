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
	"errors"
	"fmt"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

var (
	// ErrNoSteps indicates JobFlow must have at least one step.
	ErrNoSteps = errors.New("JobFlow must have at least one step")

	// ErrEmptyStepName indicates step name cannot be empty.
	ErrEmptyStepName = errors.New("step name cannot be empty")

	// ErrDuplicateStepName indicates duplicate step names are not allowed.
	ErrDuplicateStepName = errors.New("duplicate step name")

	// ErrInvalidDependency indicates a step dependency references a non-existent step.
	ErrInvalidDependency = errors.New("invalid dependency: step does not exist")

	// ErrDAGCycle indicates a cycle was detected in the step dependencies.
	ErrDAGCycle = errors.New("cycle detected in step dependencies")

	// ErrInvalidConcurrencyPolicy indicates invalid concurrency policy value.
	ErrInvalidConcurrencyPolicy = errors.New("invalid concurrencyPolicy (must be Allow, Forbid, or Replace)")

	// ErrInvalidTTLSecondsAfterFinished indicates invalid TTL value.
	ErrInvalidTTLSecondsAfterFinished = errors.New("ttlSecondsAfterFinished must be non-negative")

	// ErrInvalidBackoffLimit indicates invalid backoff limit.
	ErrInvalidBackoffLimit = errors.New("backoffLimit must be non-negative")

	// ErrInvalidActiveDeadlineSeconds indicates invalid active deadline seconds.
	ErrInvalidActiveDeadlineSeconds = errors.New("activeDeadlineSeconds must be positive")

	// ErrInvalidJobTemplate indicates invalid Job template.
	ErrInvalidJobTemplate = errors.New("invalid Job template")

	// ErrEmptyJobTemplate indicates Job template cannot be empty.
	ErrEmptyJobTemplate = errors.New("Job template cannot be empty")

	// ErrInvalidResourceTemplate indicates invalid resource template.
	ErrInvalidResourceTemplate = errors.New("invalid resource template")
)

// ValidateJobFlow validates a JobFlow resource.
func ValidateJobFlow(jobFlow *v1alpha1.JobFlow) error {
	// Validate steps
	if err := validateSteps(jobFlow.Spec.Steps); err != nil {
		return fmt.Errorf("invalid steps: %w", err)
	}

	// Validate DAG (check for cycles)
	if err := validateDAG(jobFlow.Spec.Steps); err != nil {
		return fmt.Errorf("invalid DAG: %w", err)
	}

	// Validate execution policy
	if jobFlow.Spec.ExecutionPolicy != nil {
		if err := validateExecutionPolicy(jobFlow.Spec.ExecutionPolicy); err != nil {
			return fmt.Errorf("invalid executionPolicy: %w", err)
		}
	}

	// Validate resource templates
	if jobFlow.Spec.ResourceTemplates != nil {
		if err := validateResourceTemplates(jobFlow.Spec.ResourceTemplates); err != nil {
			return fmt.Errorf("invalid resourceTemplates: %w", err)
		}
	}

	// Validate step templates
	for i, step := range jobFlow.Spec.Steps {
		if err := validateStepTemplate(&step, i); err != nil {
			return fmt.Errorf("invalid step template at index %d: %w", i, err)
		}
	}

	return nil
}

// validateSteps validates the steps array.
func validateSteps(steps []v1alpha1.Step) error {
	if len(steps) == 0 {
		return ErrNoSteps
	}

	stepNames := make(map[string]bool)
	for i, step := range steps {
		// Validate step name
		if step.Name == "" {
			return fmt.Errorf("step at index %d: %w", i, ErrEmptyStepName)
		}

		// Check for duplicate names
		if stepNames[step.Name] {
			return fmt.Errorf("%w: %s", ErrDuplicateStepName, step.Name)
		}
		stepNames[step.Name] = true

		// Validate dependencies exist
		for _, dep := range step.Dependencies {
			if dep != "" && !stepNames[dep] {
				// Check if dependency exists in any step
				found := false
				for _, s := range steps {
					if s.Name == dep {
						found = true
						break
					}
				}
				if !found {
					return fmt.Errorf("step %q: %w: %s", step.Name, ErrInvalidDependency, dep)
				}
			}
		}
	}

	return nil
}

// validateDAG validates that the step dependencies form a valid DAG (no cycles).
func validateDAG(steps []v1alpha1.Step) error {
	// Build adjacency list
	adj := make(map[string][]string)
	stepMap := make(map[string]bool)

	for _, step := range steps {
		stepMap[step.Name] = true
		adj[step.Name] = []string{}
	}

	// Build edges
	for _, step := range steps {
		for _, dep := range step.Dependencies {
			if dep != "" && stepMap[dep] {
				adj[step.Name] = append(adj[step.Name], dep)
			}
		}
	}

	// Detect cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(string) error
	dfs = func(node string) error {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range adj[node] {
			if !visited[neighbor] {
				if err := dfs(neighbor); err != nil {
					return err
				}
			} else if recStack[neighbor] {
				return fmt.Errorf("%w: cycle involving steps %s and %s", ErrDAGCycle, node, neighbor)
			}
		}

		recStack[node] = false
		return nil
	}

	// Check all nodes
	for stepName := range stepMap {
		if !visited[stepName] {
			if err := dfs(stepName); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateExecutionPolicy validates the execution policy.
func validateExecutionPolicy(policy *v1alpha1.ExecutionPolicy) error {
	// Validate concurrency policy
	if policy.ConcurrencyPolicy != "" {
		validPolicies := map[string]bool{
			"Allow":   true,
			"Forbid":  true,
			"Replace": true,
		}
		if !validPolicies[policy.ConcurrencyPolicy] {
			return fmt.Errorf("%w: %s", ErrInvalidConcurrencyPolicy, policy.ConcurrencyPolicy)
		}
	}

	// Validate TTLSecondsAfterFinished
	if policy.TTLSecondsAfterFinished != nil {
		if *policy.TTLSecondsAfterFinished < 0 {
			return fmt.Errorf("%w: %d", ErrInvalidTTLSecondsAfterFinished, *policy.TTLSecondsAfterFinished)
		}
	}

	// Validate BackoffLimit
	if policy.BackoffLimit != nil {
		if *policy.BackoffLimit < 0 {
			return fmt.Errorf("%w: %d", ErrInvalidBackoffLimit, *policy.BackoffLimit)
		}
	}

	// Validate ActiveDeadlineSeconds
	if policy.ActiveDeadlineSeconds != nil {
		if *policy.ActiveDeadlineSeconds <= 0 {
			return fmt.Errorf("%w: %d", ErrInvalidActiveDeadlineSeconds, *policy.ActiveDeadlineSeconds)
		}
	}

	return nil
}

// validateStepTemplate validates a step's Job template.
func validateStepTemplate(step *v1alpha1.Step, index int) error {
	if len(step.Template.Raw) == 0 {
		return ErrEmptyJobTemplate
	}

	// Try to extract Job template using the Step's method
	job, err := step.GetJobTemplate()
	if err == nil && job != nil {
		// Validate Job spec
		if err := validateJobSpec(&job.Spec); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidJobTemplate, err)
		}
	}

	return nil
}

// validateJobSpec validates a Kubernetes Job spec.
func validateJobSpec(jobSpec *batchv1.JobSpec) error {
	// Validate template
	if jobSpec.Template.Spec.Containers == nil || len(jobSpec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("Job spec must have at least one container")
	}

	// Validate containers
	for i, container := range jobSpec.Template.Spec.Containers {
		if container.Name == "" {
			return fmt.Errorf("container at index %d: name cannot be empty", i)
		}
		if container.Image == "" {
			return fmt.Errorf("container %q: image cannot be empty", container.Name)
		}

		// Validate resources if specified
		if err := validateContainerResources(&container.Resources); err != nil {
			return fmt.Errorf("container %q: %w", container.Name, err)
		}
	}

	// Validate backoff limit
	if jobSpec.BackoffLimit != nil && *jobSpec.BackoffLimit < 0 {
		return fmt.Errorf("backoffLimit must be non-negative")
	}

	// Validate active deadline seconds
	if jobSpec.ActiveDeadlineSeconds != nil && *jobSpec.ActiveDeadlineSeconds <= 0 {
		return fmt.Errorf("activeDeadlineSeconds must be positive")
	}

	return nil
}

// validateContainerResources validates container resource requirements.
func validateContainerResources(resources *corev1.ResourceRequirements) error {
	if resources == nil {
		return nil
	}

	// Validate requests
	if err := validateResourceList(resources.Requests); err != nil {
		return fmt.Errorf("invalid resource requests: %w", err)
	}

	// Validate limits
	if err := validateResourceList(resources.Limits); err != nil {
		return fmt.Errorf("invalid resource limits: %w", err)
	}

	// Validate that limits are >= requests
	for resourceName, limit := range resources.Limits {
		if request, exists := resources.Requests[resourceName]; exists {
			if limit.Cmp(request) < 0 {
				return fmt.Errorf("limit for %s (%s) must be >= request (%s)", resourceName, limit.String(), request.String())
			}
		}
	}

	return nil
}

// validateResourceList validates a resource list.
func validateResourceList(resources corev1.ResourceList) error {
	for resourceName, quantity := range resources {
		if quantity.IsZero() {
			return fmt.Errorf("resource %s cannot be zero", resourceName)
		}
		if quantity.Value() < 0 {
			return fmt.Errorf("resource %s cannot be negative", resourceName)
		}

		// Validate CPU format
		if resourceName == corev1.ResourceCPU {
			if err := validateCPUQuantity(quantity); err != nil {
				return fmt.Errorf("invalid CPU quantity: %w", err)
			}
		}

		// Validate memory format
		if resourceName == corev1.ResourceMemory {
			if err := validateMemoryQuantity(quantity); err != nil {
				return fmt.Errorf("invalid memory quantity: %w", err)
			}
		}
	}
	return nil
}

// validateCPUQuantity validates a CPU quantity.
func validateCPUQuantity(q resource.Quantity) error {
	// CPU can be specified as millicores (m) or cores
	// Check that it's a valid format
	if q.IsZero() {
		return fmt.Errorf("CPU quantity cannot be zero")
	}
	return nil
}

// validateMemoryQuantity validates a memory quantity.
func validateMemoryQuantity(q resource.Quantity) error {
	// Memory must be positive
	if q.IsZero() {
		return fmt.Errorf("memory quantity cannot be zero")
	}
	return nil
}

// validateResourceTemplates validates resource templates.
func validateResourceTemplates(templates *v1alpha1.ResourceTemplates) error {
	// Validate PVCs
	for i, pvc := range templates.VolumeClaimTemplates {
		if err := validatePVCTemplate(&pvc, i); err != nil {
			return fmt.Errorf("invalid PVC template at index %d: %w", i, err)
		}
	}

	// Validate ConfigMaps
	for i, cm := range templates.ConfigMapTemplates {
		if err := validateConfigMapTemplate(&cm, i); err != nil {
			return fmt.Errorf("invalid ConfigMap template at index %d: %w", i, err)
		}
	}

	return nil
}

// validatePVCTemplate validates a PVC template.
func validatePVCTemplate(pvc *corev1.PersistentVolumeClaim, index int) error {
	if pvc.Name == "" {
		return fmt.Errorf("PVC name cannot be empty")
	}

	// Validate access modes
	if len(pvc.Spec.AccessModes) == 0 {
		return fmt.Errorf("PVC %q: accessModes cannot be empty", pvc.Name)
	}

	// Validate resources
	if pvc.Spec.Resources.Requests == nil || len(pvc.Spec.Resources.Requests) == 0 {
		return fmt.Errorf("PVC %q: resources.requests cannot be empty", pvc.Name)
	}

	// Validate storage request
	if storage, exists := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; !exists || storage.IsZero() {
		return fmt.Errorf("PVC %q: storage request cannot be empty", pvc.Name)
	}

	return nil
}

// validateConfigMapTemplate validates a ConfigMap template.
func validateConfigMapTemplate(cm *corev1.ConfigMap, index int) error {
	if cm.Name == "" {
		return fmt.Errorf("ConfigMap name cannot be empty")
	}

	// ConfigMap is valid if it has name (data is optional)
	return nil
}

// ValidateStepName validates a step name.
func ValidateStepName(name string) error {
	if name == "" {
		return ErrEmptyStepName
	}

	// Step names should be valid Kubernetes resource names
	if errs := validation.IsDNS1123Subdomain(name); len(errs) > 0 {
		return fmt.Errorf("invalid step name %q: %v", name, errs)
	}

	// Step names should not contain leading/trailing whitespace
	if strings.TrimSpace(name) != name {
		return fmt.Errorf("step name %q contains leading or trailing whitespace", name)
	}

	return nil
}
