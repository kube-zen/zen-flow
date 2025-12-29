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
	"fmt"
	"math"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/dag"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
	jferrors "github.com/kube-zen/zen-flow/pkg/errors"
	"github.com/kube-zen/zen-flow/pkg/logging"
)

// JobFlowReconciler reconciles a JobFlow object
type JobFlowReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	MetricsRecorder *metrics.Recorder
	EventRecorder   *EventRecorder
}

// NewJobFlowReconciler creates a new JobFlowReconciler
func NewJobFlowReconciler(mgr ctrl.Manager, metricsRecorder *metrics.Recorder, eventRecorder *EventRecorder) *JobFlowReconciler {
	return &JobFlowReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		MetricsRecorder: metricsRecorder,
		EventRecorder:   eventRecorder,
	}
}

// Reconcile is part of the main kubernetes reconciliation loop
// +kubebuilder:rbac:groups=workflow.zen.io,resources=jobflows,verbs=get;list;watch
// +kubebuilder:rbac:groups=workflow.zen.io,resources=jobflows/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create
// +kubebuilder:rbac:groups="",resources=events,verbs=create
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
func (r *JobFlowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger = logger.WithValues("jobflow", req.NamespacedName)
	ctx = log.IntoContext(ctx, logger)

	// Add correlation ID for this reconciliation
	correlationID := logging.GenerateCorrelationID()
	reconcileCtx := logging.WithCorrelationID(ctx, correlationID)
	reconcileLogger := logging.FromContext(reconcileCtx).WithJobFlow(req.Namespace, req.Name)

	reconcileStart := time.Now()
	defer func() {
		duration := time.Since(reconcileStart)
		r.MetricsRecorder.RecordReconciliationDuration(duration.Seconds())
		reconcileLogger.WithDuration(duration).V(4).Info("Reconciliation completed")
	}()

	// Fetch the JobFlow instance
	jobFlow := &v1alpha1.JobFlow{}
	if err := r.Get(ctx, req.NamespacedName, jobFlow); err != nil {
		if k8serrors.IsNotFound(err) {
			// JobFlow was deleted, nothing to do
			reconcileLogger.V(4).Info("JobFlow was deleted, skipping reconciliation")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request
		reconcileLogger.WithError(err).Error("Failed to get JobFlow")
		return ctrl.Result{}, err
	}

	// Validate JobFlow
	if err := r.validateJobFlow(jobFlow); err != nil {
		reconcileLogger.WithError(err).Warning("JobFlow validation failed")
		// P0.2: Persist validation failure status
		jobFlow.Status.Phase = v1alpha1.JobFlowPhaseFailed
		jobFlow.Status.Conditions = append(jobFlow.Status.Conditions, v1alpha1.JobFlowCondition{
			Type:               "Ready",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "ValidationFailed",
			Message:            err.Error(),
		})
		if updateErr := r.Status().Update(ctx, jobFlow); updateErr != nil {
			reconcileLogger.WithError(updateErr).Error("Failed to update JobFlow status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, nil
	}

	// Check concurrency policy before processing
	if err := r.checkConcurrencyPolicy(ctx, jobFlow); err != nil {
		reconcileLogger.WithError(err).Warning("Concurrency policy violation")
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// Check flow-level active deadline
	if exceeded, err := r.checkActiveDeadline(jobFlow); err != nil {
		reconcileLogger.WithError(err).Error("Failed to check active deadline")
		return ctrl.Result{}, err
	} else if exceeded {
		reconcileLogger.Warning("Active deadline exceeded, marking JobFlow as failed")
		jobFlow.Status.Phase = v1alpha1.JobFlowPhaseFailed
		now := metav1.Now()
		jobFlow.Status.CompletionTime = &now
		jobFlow.Status.Conditions = append(jobFlow.Status.Conditions, v1alpha1.JobFlowCondition{
			Type:               "Ready",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "ActiveDeadlineExceeded",
			Message:            "JobFlow exceeded active deadline",
		})
		if updateErr := r.Status().Update(ctx, jobFlow); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, nil
	}

	// Check flow-level backoff limit
	if exceeded, err := r.checkBackoffLimit(jobFlow); err != nil {
		reconcileLogger.WithError(err).Error("Failed to check backoff limit")
		return ctrl.Result{}, err
	} else if exceeded {
		reconcileLogger.Warning("Backoff limit exceeded, marking JobFlow as failed")
		jobFlow.Status.Phase = v1alpha1.JobFlowPhaseFailed
		now := metav1.Now()
		jobFlow.Status.CompletionTime = &now
		jobFlow.Status.Conditions = append(jobFlow.Status.Conditions, v1alpha1.JobFlowCondition{
			Type:               "Ready",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "BackoffLimitExceeded",
			Message:            "JobFlow exceeded backoff limit",
		})
		if updateErr := r.Status().Update(ctx, jobFlow); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, nil
	}

	// Initialize if needed
	if !r.hasInitialized(jobFlow) {
		reconcileLogger.Info("Initializing JobFlow")
		if err := r.initializeJobFlow(reconcileCtx, jobFlow); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Build DAG
	dagGraph := dag.BuildDAG(jobFlow.Spec.Steps)

	// P0.4: Check for cycles in DAG
	sortedSteps, err := dagGraph.TopologicalSort()
	if err != nil {
		reconcileLogger.WithError(err).Error("DAG cycle detected")
		jobFlow.Status.Phase = v1alpha1.JobFlowPhaseFailed
		jobFlow.Status.Conditions = append(jobFlow.Status.Conditions, v1alpha1.JobFlowCondition{
			Type:               "Ready",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "DAGCycle",
			Message:            err.Error(),
		})
		if updateErr := r.Status().Update(ctx, jobFlow); updateErr != nil {
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, nil
	}

	// Check for manual approval steps and handle approvals
	if err := r.checkManualApprovals(reconcileCtx, jobFlow); err != nil {
		reconcileLogger.WithError(err).Error("Failed to check manual approvals")
		return ctrl.Result{}, err
	}

	// P0.1: Refresh step statuses from Jobs for any step with JobRef (Running/Pending-with-JobRef)
	if err := r.refreshStepStatuses(reconcileCtx, jobFlow); err != nil {
		if r.isRetryable(err) {
			reconcileLogger.WithError(err).V(4).Info("Retryable error refreshing step statuses, will retry")
			return ctrl.Result{Requeue: true}, err
		}
		reconcileLogger.WithError(err).Error("Failed to refresh step statuses")
		// Continue with execution plan despite refresh errors
	}

	// Create execution plan
	executionPlan := r.createExecutionPlan(dagGraph, jobFlow, sortedSteps)

	// Check step timeouts for running steps
	if err := r.checkStepTimeouts(reconcileCtx, jobFlow); err != nil {
		reconcileLogger.WithError(err).Error("Failed to check step timeouts")
		return ctrl.Result{}, err
	}

	// Handle step retries for failed steps
	for _, stepStatus := range jobFlow.Status.Steps {
		if stepStatus.Phase == v1alpha1.StepPhaseFailed {
			if err := r.handleStepRetry(reconcileCtx, jobFlow, stepStatus.Name); err != nil {
				reconcileLogger.WithStep(stepStatus.Name).WithError(err).Error("Failed to handle step retry")
				return ctrl.Result{}, err
			}
		}
	}

	// Execute ready steps
	for _, stepName := range executionPlan.ReadySteps {
		reconcileLogger.WithStep(stepName).V(4).Info("Executing step")
		if err := r.executeStep(reconcileCtx, jobFlow, stepName); err != nil {
			if r.isRetryable(err) {
				reconcileLogger.WithStep(stepName).WithError(err).V(4).Info("Retryable error, will retry")
				return ctrl.Result{Requeue: true}, err
			}
			if err := r.handleStepFailure(reconcileCtx, jobFlow, stepName, err); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Update status
	if err := r.updateJobFlowStatus(ctx, jobFlow, executionPlan); err != nil {
		reconcileLogger.WithError(err).Error("Failed to update JobFlow status")
		return ctrl.Result{}, err
	}

	// Check TTL cleanup if JobFlow is finished
	if jobFlow.Status.Phase == v1alpha1.JobFlowPhaseSucceeded || jobFlow.Status.Phase == v1alpha1.JobFlowPhaseFailed {
		if shouldDelete, err := r.shouldDeleteJobFlow(jobFlow); err != nil {
			reconcileLogger.WithError(err).Error("Failed to check TTL")
			return ctrl.Result{}, err
		} else if shouldDelete {
			reconcileLogger.Info("TTL expired, deleting JobFlow")
			if err := r.Delete(ctx, jobFlow); err != nil {
				reconcileLogger.WithError(err).Error("Failed to delete JobFlow")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *JobFlowReconciler) SetupWithManager(b *builder.Builder) error {
	return b.Complete(r)
}

// validateJobFlow validates a JobFlow.
func (r *JobFlowReconciler) validateJobFlow(jobFlow *v1alpha1.JobFlow) error {
	if len(jobFlow.Spec.Steps) == 0 {
		return jferrors.WithJobFlow(jferrors.New("validation_failed", "JobFlow must have at least one step"), jobFlow.Namespace, jobFlow.Name)
	}

	stepNames := make(map[string]bool)
	for _, step := range jobFlow.Spec.Steps {
		if step.Name == "" {
			return jferrors.WithStep(jferrors.WithJobFlow(jferrors.New("validation_failed", "step name cannot be empty"), jobFlow.Namespace, jobFlow.Name), step.Name)
		}
		if stepNames[step.Name] {
			return jferrors.WithStep(jferrors.WithJobFlow(jferrors.New("validation_failed", fmt.Sprintf("duplicate step name: %s", step.Name)), jobFlow.Namespace, jobFlow.Name), step.Name)
		}
		stepNames[step.Name] = true

		// Validate dependencies
		for _, dep := range step.Dependencies {
			if !stepNames[dep] && dep != "" {
				// Check if dependency exists in spec
				found := false
				for _, s := range jobFlow.Spec.Steps {
					if s.Name == dep {
						found = true
						break
					}
				}
				if !found {
					return jferrors.WithStep(jferrors.WithJobFlow(jferrors.New("validation_failed", fmt.Sprintf("step %s has invalid dependency: %s", step.Name, dep)), jobFlow.Namespace, jobFlow.Name), step.Name)
				}
			}
		}
	}

	return nil
}

// hasInitialized checks if the JobFlow has been initialized.
func (r *JobFlowReconciler) hasInitialized(jobFlow *v1alpha1.JobFlow) bool {
	return jobFlow.Status.Phase != "" || len(jobFlow.Status.Steps) > 0
}

// initializeJobFlow initializes a JobFlow.
func (r *JobFlowReconciler) initializeJobFlow(ctx context.Context, jobFlow *v1alpha1.JobFlow) error {
	now := metav1.Now()
	jobFlow.Status.Phase = v1alpha1.JobFlowPhasePending
	jobFlow.Status.StartTime = &now
	jobFlow.Status.Progress = &v1alpha1.ProgressStatus{
		TotalSteps: int32(len(jobFlow.Spec.Steps)),
	}

	// Initialize step statuses
	jobFlow.Status.Steps = make([]v1alpha1.StepStatus, len(jobFlow.Spec.Steps))
	for i, step := range jobFlow.Spec.Steps {
		jobFlow.Status.Steps[i] = v1alpha1.StepStatus{
			Name:  step.Name,
			Phase: v1alpha1.StepPhasePending,
		}
	}

	// Create resource templates if specified
	if jobFlow.Spec.ResourceTemplates != nil {
		if err := r.createResourceTemplates(ctx, jobFlow); err != nil {
			return err
		}
	}

	return r.Status().Update(ctx, jobFlow)
}

// createResourceTemplates creates resource templates for the JobFlow.
func (r *JobFlowReconciler) createResourceTemplates(ctx context.Context, jobFlow *v1alpha1.JobFlow) error {
	if jobFlow.Spec.ResourceTemplates == nil {
		return nil
	}

	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name)

	// Create PVCs
	for _, pvcTemplate := range jobFlow.Spec.ResourceTemplates.VolumeClaimTemplates {
		pvc := pvcTemplate.DeepCopy()
		pvc.Name = fmt.Sprintf("%s-%s", jobFlow.Name, pvc.Name)
		pvc.Namespace = jobFlow.Namespace
		pvc.OwnerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(jobFlow, v1alpha1.SchemeGroupVersion.WithKind("JobFlow")),
		}

		if err := r.Create(ctx, pvc); err != nil {
			if !k8serrors.IsAlreadyExists(err) {
				return jferrors.WithJobFlow(jferrors.Wrapf(err, "pvc_creation_failed", "failed to create PVC %s", pvc.Name), jobFlow.Namespace, jobFlow.Name)
			}
		} else {
			logger.WithField("pvc_name", pvc.Name).V(4).Info("Created PVC for JobFlow")
		}
	}

	// Create ConfigMaps
	for _, cmTemplate := range jobFlow.Spec.ResourceTemplates.ConfigMapTemplates {
		cm := cmTemplate.DeepCopy()
		cm.Name = fmt.Sprintf("%s-%s", jobFlow.Name, cm.Name)
		cm.Namespace = jobFlow.Namespace
		cm.OwnerReferences = []metav1.OwnerReference{
			*metav1.NewControllerRef(jobFlow, v1alpha1.SchemeGroupVersion.WithKind("JobFlow")),
		}

		if err := r.Create(ctx, cm); err != nil {
			if !k8serrors.IsAlreadyExists(err) {
				return jferrors.WithJobFlow(jferrors.Wrapf(err, "configmap_creation_failed", "failed to create ConfigMap %s", cm.Name), jobFlow.Namespace, jobFlow.Name)
			}
		} else {
			logger.WithField("configmap_name", cm.Name).V(4).Info("Created ConfigMap for JobFlow")
		}
	}

	return nil
}

// refreshStepStatuses refreshes step statuses from Jobs for any step with a JobRef.
// P0.1: This ensures Running steps are reconciled to completion.
func (r *JobFlowReconciler) refreshStepStatuses(ctx context.Context, jobFlow *v1alpha1.JobFlow) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name)

	for i := range jobFlow.Status.Steps {
		stepStatus := &jobFlow.Status.Steps[i]
		// Only refresh steps that have a JobRef (Running or Pending-with-JobRef)
		if stepStatus.JobRef == nil {
			continue
		}

		// Get the Job to check its current status
		job := &batchv1.Job{}
		jobKey := types.NamespacedName{Namespace: jobFlow.Namespace, Name: stepStatus.JobRef.Name}
		if err := r.Get(ctx, jobKey, job); err != nil {
			if k8serrors.IsNotFound(err) {
				// Job was deleted, mark step as failed
				logger.WithStep(stepStatus.Name).Warning("Job not found, marking step as failed")
				stepStatus.Phase = v1alpha1.StepPhaseFailed
				now := metav1.Now()
				stepStatus.CompletionTime = &now
				stepStatus.Message = fmt.Sprintf("Job %s not found", stepStatus.JobRef.Name)
				continue
			}
			return jferrors.WithStep(jferrors.WithJobFlow(jferrors.Wrapf(err, "job_get_failed", "failed to get Job %s", stepStatus.JobRef.Name), jobFlow.Namespace, jobFlow.Name), stepStatus.Name)
		}

		// Update step status based on job status (in memory only, don't write to API yet)
		r.refreshStepStatusFromJob(jobFlow, stepStatus.Name, job)
	}

	return nil
}

// refreshStepStatusFromJob updates step status in memory based on job status.
// This is used during refresh to avoid multiple API writes.
// P0.8: Tracks phase transitions properly.
func (r *JobFlowReconciler) refreshStepStatusFromJob(jobFlow *v1alpha1.JobFlow, stepName string, job *batchv1.Job) {
	stepStatus := r.getStepStatus(jobFlow.Status, stepName)
	if stepStatus == nil {
		return
	}

	// Update phase based on job conditions
	oldPhase := stepStatus.Phase
	var newPhase string

	if job.Status.Succeeded > 0 {
		newPhase = v1alpha1.StepPhaseSucceeded
		now := metav1.Now()
		stepStatus.CompletionTime = &now
	} else if job.Status.Failed > 0 {
		newPhase = v1alpha1.StepPhaseFailed
		now := metav1.Now()
		stepStatus.CompletionTime = &now
	} else {
		newPhase = v1alpha1.StepPhaseRunning
	}

	stepStatus.Phase = newPhase
	if oldPhase != newPhase {
		r.MetricsRecorder.RecordStepPhaseTransition(jobFlow.Name, stepName, oldPhase, newPhase)
	}
}

// createExecutionPlan creates an execution plan based on the DAG and current status.
func (r *JobFlowReconciler) createExecutionPlan(dagGraph *dag.Graph, jobFlow *v1alpha1.JobFlow, sortedSteps []string) *ExecutionPlan {
	plan := &ExecutionPlan{}

	// P0.3: Find completed steps (Succeeded OR Failed with ContinueOnFailure)
	completedSteps := make(map[string]bool)
	stepSpecs := make(map[string]*v1alpha1.Step)

	// Build step spec map for ContinueOnFailure lookup from DAG graph
	for _, stepName := range sortedSteps {
		node := dagGraph.GetStep(stepName)
		if node != nil && node.Step != nil {
			stepSpecs[stepName] = node.Step
		}
	}

	// Mark steps as completed based on their phase and ContinueOnFailure setting
	for i := range jobFlow.Status.Steps {
		stepStatus := &jobFlow.Status.Steps[i]

		// Mark as completed if succeeded
		if stepStatus.Phase == v1alpha1.StepPhaseSucceeded {
			completedSteps[stepStatus.Name] = true
		} else if stepStatus.Phase == v1alpha1.StepPhaseFailed {
			// P0.3: Failed steps complete if ContinueOnFailure is true
			if spec, ok := stepSpecs[stepStatus.Name]; ok && spec.ContinueOnFailure {
				completedSteps[stepStatus.Name] = true
			}
		}
	}

	// Find steps that are ready to execute
	for _, stepName := range sortedSteps {
		stepStatus := r.getStepStatus(jobFlow.Status, stepName)
		if stepStatus != nil && stepStatus.Phase != v1alpha1.StepPhasePending && stepStatus.Phase != v1alpha1.StepPhasePendingApproval {
			continue // Already started or completed
		}

		// Check if dependencies are satisfied
		step := dagGraph.GetStep(stepName)
		if step == nil {
			continue
		}

		ready := true
		for _, dep := range step.Dependencies {
			if !completedSteps[dep] {
				ready = false
				break
			}
		}

		// Check "When" condition if specified
		if ready && step.Step != nil && step.Step.When != "" {
			// Evaluate when condition
			evaluated, err := r.evaluateWhenCondition(jobFlow, step.Step.When)
			if err != nil {
				// If evaluation fails, skip the step
				ready = false
			} else {
				ready = evaluated
			}
		}

		if ready {
			plan.ReadySteps = append(plan.ReadySteps, stepName)
		}
	}

	return plan
}

// getStepStatus gets the status for a step.
func (r *JobFlowReconciler) getStepStatus(status v1alpha1.JobFlowStatus, stepName string) *v1alpha1.StepStatus {
	for i := range status.Steps {
		if status.Steps[i].Name == stepName {
			return &status.Steps[i]
		}
	}
	return nil
}

// executeStep executes a step.
func (r *JobFlowReconciler) executeStep(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepName string) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(stepName)

	// Find step spec
	var stepSpec *v1alpha1.Step
	for i := range jobFlow.Spec.Steps {
		if jobFlow.Spec.Steps[i].Name == stepName {
			stepSpec = &jobFlow.Spec.Steps[i]
			break
		}
	}
	if stepSpec == nil {
		return jferrors.WithStep(jferrors.WithJobFlow(jferrors.New("step_not_found", fmt.Sprintf("step not found: %s", stepName)), jobFlow.Namespace, jobFlow.Name), stepName)
	}

	// Get step status
	stepStatus := r.getStepStatus(jobFlow.Status, stepName)
	if stepStatus == nil {
		return jferrors.WithStep(jferrors.WithJobFlow(jferrors.New("step_status_not_found", fmt.Sprintf("step status not found: %s", stepName)), jobFlow.Namespace, jobFlow.Name), stepName)
	}

	// Check if step already has a job
	if stepStatus.JobRef != nil {
		// Job already exists, check its status
		job := &batchv1.Job{}
		jobKey := types.NamespacedName{Namespace: jobFlow.Namespace, Name: stepStatus.JobRef.Name}
		if err := r.Get(ctx, jobKey, job); err != nil {
			if !k8serrors.IsNotFound(err) {
				return jferrors.WithJob(jferrors.WithStep(jferrors.WithJobFlow(jferrors.Wrapf(err, "job_get_failed", "failed to get Job %s", stepStatus.JobRef.Name), jobFlow.Namespace, jobFlow.Name), stepName), jobFlow.Namespace, stepStatus.JobRef.Name)
			}
		} else {
			// Update step status based on job status
			logger.WithJob(jobFlow.Namespace, stepStatus.JobRef.Name).V(4).Info("Job already exists, updating step status")
			return r.updateStepStatusFromJob(ctx, jobFlow, stepName, job)
		}
	}

	// Handle manual approval steps
	if stepSpec.Type == v1alpha1.StepTypeManual || (stepSpec.Type == "" && len(stepSpec.Template.Raw) == 0) {
		return r.handleManualApprovalStep(ctx, jobFlow, stepSpec, stepStatus)
	}

	// Handle step inputs before creating job
	if err := r.handleStepInputs(ctx, jobFlow, stepSpec); err != nil {
		logger.WithError(err).Warning("Failed to handle step inputs, continuing")
		// Continue even if inputs fail (can be enhanced to fail fast)
	}

	// Create job for step
	job, err := r.createJobForStep(ctx, jobFlow, stepSpec)
	if err != nil {
		return jferrors.WithStep(jferrors.WithJobFlow(jferrors.Wrapf(err, "job_creation_failed", "failed to create Job for step"), jobFlow.Namespace, jobFlow.Name), stepName)
	}

	// Update step status
	stepStatus.Phase = v1alpha1.StepPhaseRunning
	stepStatus.StartTime = &metav1.Time{Time: time.Now()}
	stepStatus.JobRef = &corev1.ObjectReference{
		APIVersion: batchv1.SchemeGroupVersion.String(),
		Kind:       "Job",
		Name:       job.Name,
		Namespace:  job.Namespace,
		UID:        job.UID,
	}

	logger.WithJob(jobFlow.Namespace, job.Name).Info("Created Job for step")
	r.EventRecorder.Eventf(jobFlow, corev1.EventTypeNormal, "StepCreated", "Created Job for step %s", stepName)
	return r.Status().Update(ctx, jobFlow)
}

// createJobForStep creates a Kubernetes Job for a step.
func (r *JobFlowReconciler) createJobForStep(ctx context.Context, jobFlow *v1alpha1.JobFlow, step *v1alpha1.Step) (*batchv1.Job, error) {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(step.Name)

	// Get job template from step
	jobTemplate, err := step.GetJobTemplate()
	if err != nil {
		return nil, jferrors.WithStep(jferrors.WithJobFlow(jferrors.Wrapf(err, "template_parse_failed", "failed to get job template"), jobFlow.Namespace, jobFlow.Name), step.Name)
	}

	// Create job
	job := jobTemplate.DeepCopy()
	job.Name = fmt.Sprintf("%s-%s-%s", jobFlow.Name, step.Name, string(jobFlow.UID)[:8])
	job.Namespace = jobFlow.Namespace

	// Set owner reference
	job.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(jobFlow, v1alpha1.SchemeGroupVersion.WithKind("JobFlow")),
	}

	// Add labels
	if job.Labels == nil {
		job.Labels = make(map[string]string)
	}
	job.Labels["workflow.zen.io/step"] = step.Name
	job.Labels["workflow.zen.io/flow"] = jobFlow.Name
	job.Labels["workflow.zen.io/managed-by"] = "zen-flow"

	// Add step metadata labels/annotations
	if step.Metadata != nil {
		for k, v := range step.Metadata.Labels {
			job.Labels[k] = v
		}
		if len(step.Metadata.Annotations) > 0 {
			if job.Annotations == nil {
				job.Annotations = make(map[string]string)
			}
			for k, v := range step.Metadata.Annotations {
				job.Annotations[k] = v
			}
		}
	}

	if err := r.Create(ctx, job); err != nil {
		return nil, jferrors.WithStep(jferrors.WithJobFlow(jferrors.Wrapf(err, "job_create_failed", "failed to create Job"), jobFlow.Namespace, jobFlow.Name), step.Name)
	}

	logger.WithJob(jobFlow.Namespace, job.Name).V(4).Info("Job created successfully")
	return job, nil
}

// updateStepStatusFromJob updates step status based on job status.
func (r *JobFlowReconciler) updateStepStatusFromJob(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepName string, job *batchv1.Job) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(stepName).WithJob(jobFlow.Namespace, job.Name)

	stepStatus := r.getStepStatus(jobFlow.Status, stepName)
	if stepStatus == nil {
		return jferrors.WithStep(jferrors.WithJobFlow(jferrors.New("step_status_not_found", fmt.Sprintf("step status not found: %s", stepName)), jobFlow.Namespace, jobFlow.Name), stepName)
	}

	// Update phase based on job conditions
	// P0.8: Track phase transitions properly
	oldPhase := stepStatus.Phase
	newPhase := stepStatus.Phase

	if job.Status.Succeeded > 0 {
		newPhase = v1alpha1.StepPhaseSucceeded
		now := metav1.Now()
		stepStatus.CompletionTime = &now
		logger.Info("Step succeeded")

		// Handle step outputs after success
		var stepSpec *v1alpha1.Step
		for i := range jobFlow.Spec.Steps {
			if jobFlow.Spec.Steps[i].Name == stepName {
				stepSpec = &jobFlow.Spec.Steps[i]
				break
			}
		}
		if stepSpec != nil {
			if err := r.handleStepOutputs(ctx, jobFlow, stepSpec, stepStatus); err != nil {
				logger.WithError(err).Warning("Failed to handle step outputs")
			}
		}
	} else if job.Status.Failed > 0 {
		// Check pod failure policy before marking as failed
		if shouldFail, err := r.checkPodFailurePolicy(ctx, jobFlow, stepName, job); err != nil {
			logger.WithError(err).Warning("Failed to check pod failure policy, defaulting to fail")
			newPhase = v1alpha1.StepPhaseFailed
		} else if !shouldFail {
			// Pod failure policy says to ignore or count, don't mark as failed
			logger.Info("Pod failure policy indicates step should not fail")
			newPhase = v1alpha1.StepPhaseSucceeded // Treat as succeeded if policy says ignore
		} else {
			newPhase = v1alpha1.StepPhaseFailed
		}
		if newPhase == v1alpha1.StepPhaseFailed {
			now := metav1.Now()
			stepStatus.CompletionTime = &now
			logger.Warning("Step failed")
		}
	} else {
		newPhase = v1alpha1.StepPhaseRunning
		logger.V(4).Info("Step still running")
	}

	stepStatus.Phase = newPhase
	if oldPhase != newPhase {
		r.MetricsRecorder.RecordStepPhaseTransition(jobFlow.Name, stepName, oldPhase, newPhase)
	}

	return r.Status().Update(ctx, jobFlow)
}

// handleStepFailure handles a step failure.
func (r *JobFlowReconciler) handleStepFailure(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepName string, err error) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(stepName).WithError(err)

	stepStatus := r.getStepStatus(jobFlow.Status, stepName)
	if stepStatus == nil {
		return jferrors.WithStep(jferrors.WithJobFlow(jferrors.New("step_status_not_found", fmt.Sprintf("step status not found: %s", stepName)), jobFlow.Namespace, jobFlow.Name), stepName)
	}

	// Find step spec to check ContinueOnFailure
	var stepSpec *v1alpha1.Step
	for i := range jobFlow.Spec.Steps {
		if jobFlow.Spec.Steps[i].Name == stepName {
			stepSpec = &jobFlow.Spec.Steps[i]
			break
		}
	}

	if stepSpec != nil && stepSpec.ContinueOnFailure {
		stepStatus.Phase = v1alpha1.StepPhaseFailed
		stepStatus.Message = err.Error()
		logger.Warning("Step failed but continuing due to ContinueOnFailure")
		r.EventRecorder.Eventf(jobFlow, corev1.EventTypeWarning, "StepFailed", "Step %s failed but continuing: %v", stepName, err)
		return r.Status().Update(ctx, jobFlow)
	}

	// Mark flow as failed
	jobFlow.Status.Phase = v1alpha1.JobFlowPhaseFailed
	now := metav1.Now()
	jobFlow.Status.CompletionTime = &now
	stepStatus.Phase = v1alpha1.StepPhaseFailed
	stepStatus.Message = err.Error()

	logger.Error("Step failed, marking JobFlow as failed")
	r.EventRecorder.Eventf(jobFlow, corev1.EventTypeWarning, "StepFailed", "Step %s failed: %v", stepName, err)
	return r.Status().Update(ctx, jobFlow)
}

// isRetryable checks if an error is retryable.
func (r *JobFlowReconciler) isRetryable(err error) bool {
	return k8serrors.IsConflict(err) || k8serrors.IsServerTimeout(err)
}

// updateJobFlowStatus updates the JobFlow status.
func (r *JobFlowReconciler) updateJobFlowStatus(ctx context.Context, jobFlow *v1alpha1.JobFlow, plan *ExecutionPlan) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name)

	// Update progress
	if jobFlow.Status.Progress != nil {
		completed := int32(0)
		successful := int32(0)
		failed := int32(0)

		for _, stepStatus := range jobFlow.Status.Steps {
			if stepStatus.Phase == v1alpha1.StepPhaseSucceeded {
				completed++
				successful++
			} else if stepStatus.Phase == v1alpha1.StepPhaseFailed {
				completed++
				failed++
			}
		}

		jobFlow.Status.Progress.CompletedSteps = completed
		jobFlow.Status.Progress.SuccessfulSteps = successful
		jobFlow.Status.Progress.FailedSteps = failed

		logger.WithFields(
			logging.Field{Key: "completed_steps", Value: completed},
			logging.Field{Key: "successful_steps", Value: successful},
			logging.Field{Key: "failed_steps", Value: failed},
		).V(4).Info("Updated JobFlow progress")
	}

	// Update phase
	if jobFlow.Status.Phase == "" {
		jobFlow.Status.Phase = v1alpha1.JobFlowPhasePending
	}

	// Check if all steps are complete
	allComplete := true
	allSucceeded := true
	for _, stepStatus := range jobFlow.Status.Steps {
		if stepStatus.Phase != v1alpha1.StepPhaseSucceeded && stepStatus.Phase != v1alpha1.StepPhaseFailed {
			allComplete = false
			break
		}
		if stepStatus.Phase == v1alpha1.StepPhaseFailed {
			allSucceeded = false
		}
	}

	// P0.8: Track phase transitions properly
	oldPhase := jobFlow.Status.Phase
	if allComplete {
		if allSucceeded {
			jobFlow.Status.Phase = v1alpha1.JobFlowPhaseSucceeded
			logger.Info("JobFlow completed successfully")
		} else {
			jobFlow.Status.Phase = v1alpha1.JobFlowPhaseFailed
			logger.Warning("JobFlow completed with failures")
		}
		now := metav1.Now()
		jobFlow.Status.CompletionTime = &now
		if oldPhase != jobFlow.Status.Phase {
			r.MetricsRecorder.RecordJobFlowPhaseTransition(jobFlow.Name, jobFlow.Namespace, oldPhase, jobFlow.Status.Phase)
		}
	} else {
		if jobFlow.Status.Phase == v1alpha1.JobFlowPhasePending {
			newPhase := v1alpha1.JobFlowPhaseRunning
			if oldPhase != newPhase {
				r.MetricsRecorder.RecordJobFlowPhaseTransition(jobFlow.Name, jobFlow.Namespace, oldPhase, newPhase)
			}
			jobFlow.Status.Phase = newPhase
		}
		// Only record if phase changed
		if oldPhase != jobFlow.Status.Phase {
			r.MetricsRecorder.RecordJobFlowPhaseTransition(jobFlow.Name, jobFlow.Namespace, oldPhase, jobFlow.Status.Phase)
		}
	}

	// Update conditions
	r.updateConditions(jobFlow)

	// Update status via controller-runtime StatusWriter
	return r.Status().Update(ctx, jobFlow)
}

// updateConditions updates JobFlow conditions.
func (r *JobFlowReconciler) updateConditions(jobFlow *v1alpha1.JobFlow) {
	// Update Ready condition
	readyCondition := v1alpha1.JobFlowCondition{
		Type:               "Ready",
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
	}

	if jobFlow.Status.Phase == v1alpha1.JobFlowPhaseSucceeded {
		readyCondition.Status = corev1.ConditionTrue
		readyCondition.Reason = "FlowSucceeded"
		readyCondition.Message = "JobFlow completed successfully"
	} else if jobFlow.Status.Phase == v1alpha1.JobFlowPhaseFailed {
		readyCondition.Status = corev1.ConditionFalse
		readyCondition.Reason = "FlowFailed"
		readyCondition.Message = "JobFlow failed"
	} else if jobFlow.Status.Phase == v1alpha1.JobFlowPhaseRunning {
		readyCondition.Status = corev1.ConditionTrue
		readyCondition.Reason = "FlowRunning"
		readyCondition.Message = "JobFlow is executing"
	}

	// Update or add condition
	found := false
	for i := range jobFlow.Status.Conditions {
		if jobFlow.Status.Conditions[i].Type == readyCondition.Type {
			jobFlow.Status.Conditions[i] = readyCondition
			found = true
			break
		}
	}
	if !found {
		jobFlow.Status.Conditions = append(jobFlow.Status.Conditions, readyCondition)
	}
}

// shouldDeleteJobFlow checks if a JobFlow should be deleted based on TTLSecondsAfterFinished.
func (r *JobFlowReconciler) shouldDeleteJobFlow(jobFlow *v1alpha1.JobFlow) (bool, error) {
	// Check if JobFlow is finished
	if jobFlow.Status.Phase != v1alpha1.JobFlowPhaseSucceeded && jobFlow.Status.Phase != v1alpha1.JobFlowPhaseFailed {
		return false, nil
	}

	// Check if completion time is set
	if jobFlow.Status.CompletionTime == nil {
		return false, nil
	}

	// Get TTL from execution policy
	var ttlSeconds int32 = 86400 // Default 24 hours
	if jobFlow.Spec.ExecutionPolicy != nil && jobFlow.Spec.ExecutionPolicy.TTLSecondsAfterFinished != nil {
		ttlSeconds = *jobFlow.Spec.ExecutionPolicy.TTLSecondsAfterFinished
	}

	// If TTL is 0, delete immediately
	if ttlSeconds == 0 {
		return true, nil
	}

	// Calculate expiration time
	expirationTime := jobFlow.Status.CompletionTime.Add(time.Duration(ttlSeconds) * time.Second)
	now := time.Now()

	// Check if TTL has expired
	return now.After(expirationTime), nil
}

// checkConcurrencyPolicy checks if concurrent executions are allowed.
func (r *JobFlowReconciler) checkConcurrencyPolicy(ctx context.Context, jobFlow *v1alpha1.JobFlow) error {
	policy := "Forbid" // Default
	if jobFlow.Spec.ExecutionPolicy != nil && jobFlow.Spec.ExecutionPolicy.ConcurrencyPolicy != "" {
		policy = jobFlow.Spec.ExecutionPolicy.ConcurrencyPolicy
	}

	if policy == "Allow" {
		return nil // Allow concurrent executions
	}

	// For "Forbid" or "Replace", check for other running JobFlows with the same name
	jobFlowList := &v1alpha1.JobFlowList{}
	if err := r.List(ctx, jobFlowList, client.InNamespace(jobFlow.Namespace)); err != nil {
		return jferrors.WithJobFlow(jferrors.Wrapf(err, "list_failed", "failed to list JobFlows"), jobFlow.Namespace, jobFlow.Name)
	}

	for _, existing := range jobFlowList.Items {
		// Skip self
		if existing.UID == jobFlow.UID {
			continue
		}

		// Check if same name and running
		if existing.Name == jobFlow.Name && existing.Status.Phase == v1alpha1.JobFlowPhaseRunning {
			if policy == "Replace" {
				// Delete the existing JobFlow
				if err := r.Delete(ctx, &existing); err != nil {
					return jferrors.WithJobFlow(jferrors.Wrapf(err, "delete_failed", "failed to delete existing JobFlow"), jobFlow.Namespace, jobFlow.Name)
				}
				return nil
			}
			// Forbid: return error
			return jferrors.WithJobFlow(jferrors.New("concurrency_forbidden", fmt.Sprintf("concurrent execution forbidden for JobFlow %s", jobFlow.Name)), jobFlow.Namespace, jobFlow.Name)
		}
	}

	return nil
}

// checkActiveDeadline checks if the JobFlow has exceeded its active deadline.
func (r *JobFlowReconciler) checkActiveDeadline(jobFlow *v1alpha1.JobFlow) (bool, error) {
	if jobFlow.Spec.ExecutionPolicy == nil || jobFlow.Spec.ExecutionPolicy.ActiveDeadlineSeconds == nil {
		return false, nil // No deadline set
	}

	if jobFlow.Status.StartTime == nil {
		return false, nil // Not started yet
	}

	deadlineSeconds := *jobFlow.Spec.ExecutionPolicy.ActiveDeadlineSeconds
	deadlineTime := jobFlow.Status.StartTime.Add(time.Duration(deadlineSeconds) * time.Second)
	now := time.Now()

	return now.After(deadlineTime), nil
}

// checkBackoffLimit checks if the JobFlow has exceeded its backoff limit.
func (r *JobFlowReconciler) checkBackoffLimit(jobFlow *v1alpha1.JobFlow) (bool, error) {
	limit := int32(6) // Default
	if jobFlow.Spec.ExecutionPolicy != nil && jobFlow.Spec.ExecutionPolicy.BackoffLimit != nil {
		limit = *jobFlow.Spec.ExecutionPolicy.BackoffLimit
	}

	// Count total retries across all steps
	totalRetries := int32(0)
	for _, stepStatus := range jobFlow.Status.Steps {
		totalRetries += stepStatus.RetryCount
	}

	return totalRetries > limit, nil
}

// checkStepTimeouts checks if any running steps have exceeded their timeout.
func (r *JobFlowReconciler) checkStepTimeouts(ctx context.Context, jobFlow *v1alpha1.JobFlow) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name)

	for i := range jobFlow.Status.Steps {
		stepStatus := &jobFlow.Status.Steps[i]
		if stepStatus.Phase != v1alpha1.StepPhaseRunning {
			continue
		}

		// Find step spec
		var stepSpec *v1alpha1.Step
		for j := range jobFlow.Spec.Steps {
			if jobFlow.Spec.Steps[j].Name == stepStatus.Name {
				stepSpec = &jobFlow.Spec.Steps[j]
				break
			}
		}

		if stepSpec == nil || stepSpec.TimeoutSeconds == nil {
			continue // No timeout set
		}

		if stepStatus.StartTime == nil {
			continue // Not started yet
		}

		timeoutSeconds := *stepSpec.TimeoutSeconds
		timeoutTime := stepStatus.StartTime.Add(time.Duration(timeoutSeconds) * time.Second)
		now := time.Now()

		if now.After(timeoutTime) {
			logger.WithStep(stepStatus.Name).Warning("Step timeout exceeded")
			// Mark step as failed due to timeout
			stepStatus.Phase = v1alpha1.StepPhaseFailed
			now := metav1.Now()
			stepStatus.CompletionTime = &now
			stepStatus.Message = fmt.Sprintf("Step exceeded timeout of %d seconds", timeoutSeconds)

			// Delete the job if it exists
			if stepStatus.JobRef != nil {
				job := &batchv1.Job{}
				jobKey := types.NamespacedName{Namespace: jobFlow.Namespace, Name: stepStatus.JobRef.Name}
				if err := r.Get(ctx, jobKey, job); err == nil {
					if err := r.Delete(ctx, job); err != nil {
						logger.WithStep(stepStatus.Name).WithError(err).Error("Failed to delete timed-out job")
					}
				}
			}

			// Check if step should continue on failure
			if stepSpec.ContinueOnFailure {
				r.EventRecorder.Eventf(jobFlow, corev1.EventTypeWarning, "StepTimeout", "Step %s exceeded timeout but continuing", stepStatus.Name)
			} else {
				// Mark flow as failed
				jobFlow.Status.Phase = v1alpha1.JobFlowPhaseFailed
				now := metav1.Now()
				jobFlow.Status.CompletionTime = &now
				r.EventRecorder.Eventf(jobFlow, corev1.EventTypeWarning, "StepTimeout", "Step %s exceeded timeout, marking JobFlow as failed", stepStatus.Name)
			}

			return r.Status().Update(ctx, jobFlow)
		}
	}

	return nil
}

// handleStepRetry handles retrying a failed step based on RetryPolicy.
func (r *JobFlowReconciler) handleStepRetry(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepName string) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(stepName)

	// Find step spec
	var stepSpec *v1alpha1.Step
	for i := range jobFlow.Spec.Steps {
		if jobFlow.Spec.Steps[i].Name == stepName {
			stepSpec = &jobFlow.Spec.Steps[i]
			break
		}
	}
	if stepSpec == nil || stepSpec.RetryPolicy == nil {
		return nil // No retry policy
	}

	stepStatus := r.getStepStatus(jobFlow.Status, stepName)
	if stepStatus == nil {
		return jferrors.WithStep(jferrors.WithJobFlow(jferrors.New("step_status_not_found", fmt.Sprintf("step status not found: %s", stepName)), jobFlow.Namespace, jobFlow.Name), stepName)
	}

	// Check if step has failed
	if stepStatus.Phase != v1alpha1.StepPhaseFailed {
		return nil // Not failed, no retry needed
	}

	// Check retry limit
	limit := int32(3) // Default
	if stepSpec.RetryPolicy.Limit > 0 {
		limit = stepSpec.RetryPolicy.Limit
	}

	if stepStatus.RetryCount >= limit {
		logger.Warning("Step retry limit exceeded")
		return nil // Retry limit exceeded
	}

	// Calculate backoff delay
	backoffDuration := r.calculateBackoff(stepSpec.RetryPolicy, stepStatus.RetryCount)

	// Check if enough time has passed since last failure
	if stepStatus.CompletionTime != nil {
		nextRetryTime := stepStatus.CompletionTime.Add(backoffDuration)
		if time.Now().Before(nextRetryTime) {
			// Not time to retry yet
			return nil
		}
	}

	// Increment retry count
	stepStatus.RetryCount++
	stepStatus.Phase = v1alpha1.StepPhasePending
	stepStatus.StartTime = nil
	stepStatus.CompletionTime = nil
	stepStatus.JobRef = nil
	stepStatus.Message = ""

	logger.WithField("retry_count", stepStatus.RetryCount).Info("Retrying step")
	r.EventRecorder.Eventf(jobFlow, corev1.EventTypeNormal, "StepRetry", "Retrying step %s (attempt %d)", stepName, stepStatus.RetryCount)

	return r.Status().Update(ctx, jobFlow)
}

// calculateBackoff calculates the backoff duration based on the retry policy.
func (r *JobFlowReconciler) calculateBackoff(retryPolicy *v1alpha1.RetryPolicy, retryCount int32) time.Duration {
	if retryPolicy.Backoff == nil {
		// Default exponential backoff: 1s, 2s, 4s, 8s...
		return time.Duration(1<<retryCount) * time.Second
	}

	backoff := retryPolicy.Backoff
	baseDuration, err := time.ParseDuration(backoff.Duration)
	if err != nil {
		// Default to 1s if parsing fails
		baseDuration = time.Second
	}

	switch backoff.Type {
	case "Fixed":
		return baseDuration
	case "Linear":
		return baseDuration * time.Duration(retryCount+1)
	case "Exponential":
		factor := 2.0 // Default factor
		if backoff.Factor != nil {
			factor = *backoff.Factor
		}
		return time.Duration(float64(baseDuration) * math.Pow(factor, float64(retryCount)))
	default:
		// Default exponential
		return baseDuration * time.Duration(1<<retryCount)
	}
}

// checkPodFailurePolicy checks pod failure policy rules against a failed job.
func (r *JobFlowReconciler) checkPodFailurePolicy(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepName string, job *batchv1.Job) (bool, error) {
	// Get pod failure policy from execution policy
	if jobFlow.Spec.ExecutionPolicy == nil || jobFlow.Spec.ExecutionPolicy.PodFailurePolicy == nil {
		return true, nil // Default: fail the step
	}

	policy := jobFlow.Spec.ExecutionPolicy.PodFailurePolicy

	// Get failed pods from the job
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(job.Namespace), client.MatchingLabels{
		"job-name": job.Name,
	}); err != nil {
		return true, jferrors.WithStep(jferrors.WithJobFlow(jferrors.Wrapf(err, "pod_list_failed", "failed to list pods"), jobFlow.Namespace, jobFlow.Name), stepName)
	}

	// Check each failed pod against policy rules
	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodFailed {
			continue
		}

		// Get container exit codes
		exitCodes := r.getContainerExitCodes(&pod)

		// Check each rule
		for _, rule := range policy.Rules {
			if rule.OnExitCodes == nil {
				// No exit code matching, apply action directly
				return r.applyPodFailureAction(rule.Action), nil
			}

			// Check exit codes
			matches := r.matchExitCodes(exitCodes, rule.OnExitCodes)
			if matches {
				return r.applyPodFailureAction(rule.Action), nil
			}
		}
	}

	// No rules matched, default to fail
	return true, nil
}

// getContainerExitCodes extracts exit codes from pod container statuses.
func (r *JobFlowReconciler) getContainerExitCodes(pod *corev1.Pod) map[string]int32 {
	exitCodes := make(map[string]int32)
	containerName := "main" // Default container name

	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Terminated != nil {
			exitCodes[status.Name] = status.State.Terminated.ExitCode
			if containerName == "main" {
				containerName = status.Name
			}
		}
	}

	return exitCodes
}

// matchExitCodes checks if exit codes match the policy rule.
func (r *JobFlowReconciler) matchExitCodes(exitCodes map[string]int32, rule *v1alpha1.PodFailurePolicyOnExitCodes) bool {
	containerName := rule.ContainerName
	if containerName == "" {
		containerName = "main"
	}

	exitCode, exists := exitCodes[containerName]
	if !exists {
		return false
	}

	switch rule.Operator {
	case "In":
		for _, val := range rule.Values {
			if exitCode == val {
				return true
			}
		}
		return false
	case "NotIn":
		for _, val := range rule.Values {
			if exitCode == val {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// applyPodFailureAction applies the action from a pod failure policy rule.
func (r *JobFlowReconciler) applyPodFailureAction(action string) bool {
	switch action {
	case "Ignore":
		return false // Don't fail the step
	case "Count":
		return false // Count but don't fail (could be enhanced to track counts)
	case "FailJob":
		return true // Fail the step
	default:
		return true // Default: fail
	}
}

// evaluateWhenCondition evaluates a "When" condition string.
// Currently supports simple boolean expressions and step status checks.
// TODO: Enhance with full template engine support.
func (r *JobFlowReconciler) evaluateWhenCondition(jobFlow *v1alpha1.JobFlow, condition string) (bool, error) {
	// Simple implementation: check if condition references step status
	// Examples: "steps.step1.phase == 'Succeeded'", "steps.step1.phase != 'Failed'"
	// For now, return true if condition is not empty (can be enhanced with proper parser)
	if condition == "" {
		return true, nil
	}

	// Basic keyword-based evaluation
	// Check for common patterns like "always", "never", step references
	if condition == "always" || condition == "true" {
		return true, nil
	}
	if condition == "never" || condition == "false" {
		return false, nil
	}

	// TODO: Implement proper template evaluation with step status access
	// For now, default to true if condition exists
	return true, nil
}

// handleStepInputs processes step inputs (artifacts, parameters) before step execution.
// Currently a placeholder - actual artifact/parameter handling can be enhanced.
func (r *JobFlowReconciler) handleStepInputs(ctx context.Context, jobFlow *v1alpha1.JobFlow, step *v1alpha1.Step) error {
	if step.Inputs == nil {
		return nil
	}

	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(step.Name)

	// Handle artifacts
	for _, artifact := range step.Inputs.Artifacts {
		logger.WithField("artifact_name", artifact.Name).V(4).Info("Processing artifact input")
		// TODO: Implement artifact fetching from previous steps or HTTP sources
		// For now, just log
	}

	// Handle parameters
	for _, param := range step.Inputs.Parameters {
		logger.WithField("parameter_name", param.Name).V(4).Info("Processing parameter input")
		// TODO: Implement parameter resolution from values or valueFrom
		// For now, just log
	}

	return nil
}

// handleStepOutputs processes step outputs (artifacts, parameters) after step completion.
// Currently a placeholder - actual artifact/parameter handling can be enhanced.
func (r *JobFlowReconciler) handleStepOutputs(ctx context.Context, jobFlow *v1alpha1.JobFlow, step *v1alpha1.Step, stepStatus *v1alpha1.StepStatus) error {
	if step.Outputs == nil {
		return nil
	}

	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(step.Name)

	// Handle artifacts
	for _, artifact := range step.Outputs.Artifacts {
		logger.WithField("artifact_name", artifact.Name).V(4).Info("Processing artifact output")
		// TODO: Implement artifact archiving/uploading to S3 or storage
		// For now, just log
	}

	// Handle parameters
	for _, param := range step.Outputs.Parameters {
		logger.WithField("parameter_name", param.Name).V(4).Info("Processing parameter output")
		// TODO: Implement parameter extraction from job outputs using JSONPath
		// For now, just log
	}

	// Store outputs in step status (placeholder)
	if stepStatus.Outputs == nil {
		stepStatus.Outputs = &v1alpha1.StepOutputs{}
	}

	return nil
}

// checkManualApprovals checks for manual approval steps and handles approvals.
func (r *JobFlowReconciler) checkManualApprovals(ctx context.Context, jobFlow *v1alpha1.JobFlow) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name)

	// Check if there are any steps waiting for approval
	hasPendingApproval := false
	for i := range jobFlow.Status.Steps {
		stepStatus := &jobFlow.Status.Steps[i]
		if stepStatus.Phase == v1alpha1.StepPhasePendingApproval {
			hasPendingApproval = true

			// Check if this step has been approved via annotation
			approvalKey := fmt.Sprintf("%s/%s", v1alpha1.ApprovalAnnotationKey, stepStatus.Name)
			if approved, exists := jobFlow.Annotations[approvalKey]; exists && approved == v1alpha1.ApprovalAnnotationValue {
				// Step has been approved, mark it as succeeded
				stepStatus.Phase = v1alpha1.StepPhaseSucceeded
				stepStatus.CompletionTime = &metav1.Time{Time: time.Now()}
				if stepStatus.StartTime == nil {
					stepStatus.StartTime = &metav1.Time{Time: time.Now()}
				}
				logger.WithStep(stepStatus.Name).Info("Manual approval step approved")
				r.EventRecorder.Eventf(jobFlow, corev1.EventTypeNormal, "StepApproved", "Step %s has been approved", stepStatus.Name)
			}
		}
	}

	// Update JobFlow phase based on approval status
	if hasPendingApproval && jobFlow.Status.Phase != v1alpha1.JobFlowPhasePaused {
		jobFlow.Status.Phase = v1alpha1.JobFlowPhasePaused
		logger.Info("JobFlow paused waiting for manual approval")
		r.EventRecorder.Eventf(jobFlow, corev1.EventTypeNormal, "FlowPaused", "JobFlow paused waiting for manual approval")
	} else if !hasPendingApproval && jobFlow.Status.Phase == v1alpha1.JobFlowPhasePaused {
		// No more pending approvals, resume the flow
		jobFlow.Status.Phase = v1alpha1.JobFlowPhaseRunning
		logger.Info("JobFlow resumed after approval")
		r.EventRecorder.Eventf(jobFlow, corev1.EventTypeNormal, "FlowResumed", "JobFlow resumed after approval")
	}

	return nil
}

// handleManualApprovalStep handles a manual approval step.
func (r *JobFlowReconciler) handleManualApprovalStep(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepSpec *v1alpha1.Step, stepStatus *v1alpha1.StepStatus) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(stepSpec.Name)

	// If step is already approved, mark it as succeeded
	approvalKey := fmt.Sprintf("%s/%s", v1alpha1.ApprovalAnnotationKey, stepSpec.Name)
	if approved, exists := jobFlow.Annotations[approvalKey]; exists && approved == v1alpha1.ApprovalAnnotationValue {
		if stepStatus.Phase != v1alpha1.StepPhaseSucceeded {
			stepStatus.Phase = v1alpha1.StepPhaseSucceeded
			stepStatus.CompletionTime = &metav1.Time{Time: time.Now()}
			if stepStatus.StartTime == nil {
				stepStatus.StartTime = &metav1.Time{Time: time.Now()}
			}
			logger.Info("Manual approval step approved")
			r.EventRecorder.Eventf(jobFlow, corev1.EventTypeNormal, "StepApproved", "Step %s has been approved", stepSpec.Name)
			return r.Status().Update(ctx, jobFlow)
		}
		return nil
	}

	// Step is waiting for approval
	if stepStatus.Phase != v1alpha1.StepPhasePendingApproval {
		stepStatus.Phase = v1alpha1.StepPhasePendingApproval
		stepStatus.StartTime = &metav1.Time{Time: time.Now()}
		if stepSpec.Message != "" {
			stepStatus.Message = stepSpec.Message
		} else {
			stepStatus.Message = fmt.Sprintf("Waiting for manual approval of step %s", stepSpec.Name)
		}
		logger.WithField("message", stepStatus.Message).Info("Manual approval step waiting for approval")
		r.EventRecorder.Eventf(jobFlow, corev1.EventTypeNormal, "StepPendingApproval", "Step %s is waiting for manual approval: %s", stepSpec.Name, stepStatus.Message)

		// Update JobFlow phase to Paused
		if jobFlow.Status.Phase != v1alpha1.JobFlowPhasePaused {
			jobFlow.Status.Phase = v1alpha1.JobFlowPhasePaused
			r.EventRecorder.Eventf(jobFlow, corev1.EventTypeNormal, "FlowPaused", "JobFlow paused waiting for manual approval of step %s", stepSpec.Name)
		}

		return r.Status().Update(ctx, jobFlow)
	}

	return nil
}
