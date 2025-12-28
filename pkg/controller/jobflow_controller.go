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
	"sync"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	batchinformers "k8s.io/client-go/informers/batch/v1"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/dag"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
	jferrors "github.com/kube-zen/zen-flow/pkg/errors"
	"github.com/kube-zen/zen-flow/pkg/logging"
)

const (
	// DefaultResyncPeriod is the default resync period for informers.
	DefaultResyncPeriod = 10 * time.Minute

	// DefaultMaxConcurrentReconciles is the default maximum number of concurrent reconciles.
	DefaultMaxConcurrentReconciles = 10

	// DefaultRequeueDelay is the default delay for requeuing failed reconciles.
	DefaultRequeueDelay = 5 * time.Second
)

// JobFlowController manages JobFlow resources.
type JobFlowController struct {
	dynamicClient dynamic.Interface
	kubeClient    kubernetes.Interface

	// Informers
	jobFlowInformer cache.SharedInformer
	jobInformer     batchinformers.JobInformer
	pvcInformer     coreinformers.PersistentVolumeClaimInformer

	// Work queues
	jobFlowQueue workqueue.RateLimitingInterface
	jobQueue     workqueue.RateLimitingInterface

	// Status updater
	statusUpdater *StatusUpdater

	// Metrics
	metricsRecorder *metrics.Recorder
	eventRecorder   *EventRecorder

	// Configuration
	maxConcurrentReconciles int
	resyncPeriod            time.Duration

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc

	// Shutdown synchronization
	shutdownComplete chan struct{}
	shutdownOnce     sync.Once
}

// NewJobFlowController creates a new JobFlow controller.
func NewJobFlowController(
	dynamicClient dynamic.Interface,
	kubeClient kubernetes.Interface,
	jobFlowInformer cache.SharedInformer,
	jobInformer batchinformers.JobInformer,
	pvcInformer coreinformers.PersistentVolumeClaimInformer,
	statusUpdater *StatusUpdater,
	metricsRecorder *metrics.Recorder,
	eventRecorder *EventRecorder,
) *JobFlowController {
	ctx, cancel := context.WithCancel(context.Background())

	return &JobFlowController{
		dynamicClient:           dynamicClient,
		kubeClient:              kubeClient,
		jobFlowInformer:         jobFlowInformer,
		jobInformer:             jobInformer,
		pvcInformer:             pvcInformer,
		jobFlowQueue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "jobflows"),
		jobQueue:                workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "jobs"),
		statusUpdater:           statusUpdater,
		metricsRecorder:         metricsRecorder,
		eventRecorder:           eventRecorder,
		maxConcurrentReconciles: DefaultMaxConcurrentReconciles,
		resyncPeriod:            DefaultResyncPeriod,
		ctx:                     ctx,
		cancel:                  cancel,
		shutdownComplete:        make(chan struct{}),
	}
}

// Start starts the controller.
func (c *JobFlowController) Start() error {
	logger := logging.NewLogger()
	logger.Info("Starting JobFlow controller...")

	// Set up event handlers
	_, err := c.jobFlowInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.enqueueJobFlow,
		UpdateFunc: func(oldObj, newObj interface{}) { c.enqueueJobFlow(newObj) },
		DeleteFunc: c.enqueueJobFlow,
	})
	if err != nil {
		return fmt.Errorf("failed to add JobFlow event handler: %w", err)
	}

	_, err = c.jobInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handleJobAdd,
		UpdateFunc: c.handleJobUpdate,
		DeleteFunc: c.handleJobDelete,
	})
	if err != nil {
		return fmt.Errorf("failed to add Job event handler: %w", err)
	}

	// Start workers
	for i := 0; i < c.maxConcurrentReconciles; i++ {
		go c.runWorker(c.ctx)
	}

	logger.Info("JobFlow controller started")
	return nil
}

// Stop stops the controller.
func (c *JobFlowController) Stop() {
	c.shutdownOnce.Do(func() {
		logger := logging.NewLogger()
		logger.Info("Stopping JobFlow controller...")
		c.cancel()
		c.jobFlowQueue.ShutDown()
		c.jobQueue.ShutDown()
		close(c.shutdownComplete)
		logger.Info("JobFlow controller stopped")
	})
}

// Wait waits for the controller to stop.
func (c *JobFlowController) Wait() {
	<-c.shutdownComplete
}

// enqueueJobFlow enqueues a JobFlow for reconciliation.
func (c *JobFlowController) enqueueJobFlow(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger := logging.NewLogger()
		logger.WithError(err).Error("Error getting key for JobFlow")
		return
	}
	c.jobFlowQueue.Add(key)
}

// handleJobAdd handles Job add events.
func (c *JobFlowController) handleJobAdd(obj interface{}) {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return
	}
	c.handleJobChange(job)
}

// handleJobUpdate handles Job update events.
func (c *JobFlowController) handleJobUpdate(oldObj, newObj interface{}) {
	job, ok := newObj.(*batchv1.Job)
	if !ok {
		return
	}
	c.handleJobChange(job)
}

// handleJobDelete handles Job delete events.
func (c *JobFlowController) handleJobDelete(obj interface{}) {
	job, ok := obj.(*batchv1.Job)
	if !ok {
		return
	}
	c.handleJobChange(job)
}

// handleJobChange handles Job changes by finding the owning JobFlow and enqueuing it.
func (c *JobFlowController) handleJobChange(job *batchv1.Job) {
	ownerRef := metav1.GetControllerOf(job)
	if ownerRef == nil {
		return
	}

	if ownerRef.Kind != "JobFlow" || ownerRef.APIVersion != v1alpha1.SchemeGroupVersion.String() {
		return
	}

	key := fmt.Sprintf("%s/%s", job.Namespace, ownerRef.Name)
	c.jobFlowQueue.Add(key)
}

// runWorker runs a worker that processes items from the queue.
func (c *JobFlowController) runWorker(ctx context.Context) {
	for c.processNextJobFlow(ctx) {
	}
}

// processNextJobFlow processes the next JobFlow from the queue.
func (c *JobFlowController) processNextJobFlow(ctx context.Context) bool {
	obj, shutdown := c.jobFlowQueue.Get()
	if shutdown {
		return false
	}
	defer c.jobFlowQueue.Done(obj)

	key := obj.(string)

	// Add correlation ID for this reconciliation
	correlationID := logging.GenerateCorrelationID()
	reconcileCtx := logging.WithCorrelationID(ctx, correlationID)
	logger := logging.FromContext(reconcileCtx).WithField("jobflow_key", key)

	reconcileStart := time.Now()
	if err := c.reconcileJobFlow(reconcileCtx, key); err != nil {
		logger.WithError(err).WithDuration(time.Since(reconcileStart)).Error("Error reconciling JobFlow")
		c.metricsRecorder.RecordReconciliationDuration(time.Since(reconcileStart).Seconds())
		c.jobFlowQueue.AddRateLimited(key)
		return true
	}

	c.metricsRecorder.RecordReconciliationDuration(time.Since(reconcileStart).Seconds())
	c.jobFlowQueue.Forget(obj)
	return true
}

// reconcileJobFlow reconciles a JobFlow.
func (c *JobFlowController) reconcileJobFlow(ctx context.Context, key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return jferrors.Wrap(err, "invalid_key", "failed to split JobFlow key")
	}

	logger := logging.FromContext(ctx).WithJobFlow(namespace, name)
	logger.V(4).Info("Reconciling JobFlow")

	obj, exists, err := c.jobFlowInformer.GetStore().GetByKey(key)
	if err != nil {
		return jferrors.WithJobFlow(jferrors.Wrapf(err, "get_failed", "failed to get JobFlow %s", key), namespace, name)
	}

	if !exists {
		// JobFlow was deleted, nothing to do
		logger.V(4).Info("JobFlow was deleted, skipping reconciliation")
		return nil
	}

	// Convert unstructured to JobFlow
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return jferrors.WithJobFlow(jferrors.New("conversion_failed", fmt.Sprintf("object is not unstructured: %T", obj)), namespace, name)
	}

	jobFlow := &v1alpha1.JobFlow{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, jobFlow); err != nil {
		return jferrors.WithJobFlow(jferrors.Wrapf(err, "conversion_failed", "failed to convert unstructured to JobFlow"), namespace, name)
	}

	// Validate JobFlow
	if err := c.validateJobFlow(jobFlow); err != nil {
		logger.WithError(err).Warning("JobFlow validation failed")
		return c.updateStatus(jobFlow, v1alpha1.JobFlowPhaseFailed, err)
	}

	// Initialize if needed
	if !c.hasInitialized(jobFlow) {
		logger.Info("Initializing JobFlow")
		return c.initializeJobFlow(ctx, jobFlow)
	}

	// Build DAG
	dagGraph := dag.BuildDAG(jobFlow.Spec.Steps)

	// Create execution plan
	executionPlan := c.createExecutionPlan(dagGraph, jobFlow.Status)

	// Execute ready steps
	for _, stepName := range executionPlan.ReadySteps {
		logger.WithStep(stepName).V(4).Info("Executing step")
		if err := c.executeStep(ctx, jobFlow, stepName); err != nil {
			if c.isRetryable(err) {
				logger.WithStep(stepName).WithError(err).V(4).Info("Retryable error, will retry")
				return err
			}
			return c.handleStepFailure(ctx, jobFlow, stepName, err)
		}
	}

	// Update status
	return c.updateJobFlowStatus(ctx, jobFlow, executionPlan)
}

// validateJobFlow validates a JobFlow.
func (c *JobFlowController) validateJobFlow(jobFlow *v1alpha1.JobFlow) error {
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
func (c *JobFlowController) hasInitialized(jobFlow *v1alpha1.JobFlow) bool {
	return jobFlow.Status.Phase != "" || len(jobFlow.Status.Steps) > 0
}

// initializeJobFlow initializes a JobFlow.
func (c *JobFlowController) initializeJobFlow(ctx context.Context, jobFlow *v1alpha1.JobFlow) error {
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
		if err := c.createResourceTemplates(ctx, jobFlow); err != nil {
			return err
		}
	}

	return c.updateJobFlowStatus(ctx, jobFlow, nil)
}

// createResourceTemplates creates resource templates for the JobFlow.
func (c *JobFlowController) createResourceTemplates(ctx context.Context, jobFlow *v1alpha1.JobFlow) error {
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

		_, err := c.kubeClient.CoreV1().PersistentVolumeClaims(jobFlow.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			return jferrors.WithJobFlow(jferrors.Wrapf(err, "pvc_creation_failed", "failed to create PVC %s", pvc.Name), jobFlow.Namespace, jobFlow.Name)
		}
		if err == nil {
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

		_, err := c.kubeClient.CoreV1().ConfigMaps(jobFlow.Namespace).Create(ctx, cm, metav1.CreateOptions{})
		if err != nil && !k8serrors.IsAlreadyExists(err) {
			return jferrors.WithJobFlow(jferrors.Wrapf(err, "configmap_creation_failed", "failed to create ConfigMap %s", cm.Name), jobFlow.Namespace, jobFlow.Name)
		}
		if err == nil {
			logger.WithField("configmap_name", cm.Name).V(4).Info("Created ConfigMap for JobFlow")
		}
	}

	return nil
}

// ExecutionPlan represents an execution plan for a JobFlow.
type ExecutionPlan struct {
	ReadySteps []string
}

// createExecutionPlan creates an execution plan based on the DAG and current status.
func (c *JobFlowController) createExecutionPlan(dagGraph *dag.Graph, status v1alpha1.JobFlowStatus) *ExecutionPlan {
	plan := &ExecutionPlan{}

	// Find ready steps (steps whose dependencies are satisfied)
	completedSteps := make(map[string]bool)
	for _, stepStatus := range status.Steps {
		if stepStatus.Phase == v1alpha1.StepPhaseSucceeded {
			completedSteps[stepStatus.Name] = true
		}
	}

	// Find steps that are ready to execute
	for _, stepName := range dagGraph.TopologicalSort() {
		stepStatus := c.getStepStatus(status, stepName)
		if stepStatus != nil && stepStatus.Phase != v1alpha1.StepPhasePending {
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

		if ready {
			plan.ReadySteps = append(plan.ReadySteps, stepName)
		}
	}

	return plan
}

// getStepStatus gets the status for a step.
func (c *JobFlowController) getStepStatus(status v1alpha1.JobFlowStatus, stepName string) *v1alpha1.StepStatus {
	for i := range status.Steps {
		if status.Steps[i].Name == stepName {
			return &status.Steps[i]
		}
	}
	return nil
}

// executeStep executes a step.
func (c *JobFlowController) executeStep(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepName string) error {
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
	stepStatus := c.getStepStatus(jobFlow.Status, stepName)
	if stepStatus == nil {
		return jferrors.WithStep(jferrors.WithJobFlow(jferrors.New("step_status_not_found", fmt.Sprintf("step status not found: %s", stepName)), jobFlow.Namespace, jobFlow.Name), stepName)
	}

	// Check if step already has a job
	if stepStatus.JobRef != nil {
		// Job already exists, check its status
		job, err := c.jobInformer.Lister().Jobs(jobFlow.Namespace).Get(stepStatus.JobRef.Name)
		if err != nil && !k8serrors.IsNotFound(err) {
			return jferrors.WithJob(jferrors.WithStep(jferrors.WithJobFlow(jferrors.Wrapf(err, "job_get_failed", "failed to get Job %s", stepStatus.JobRef.Name), jobFlow.Namespace, jobFlow.Name), stepName), jobFlow.Namespace, stepStatus.JobRef.Name)
		}
		if job != nil {
			// Update step status based on job status
			logger.WithJob(jobFlow.Namespace, stepStatus.JobRef.Name).V(4).Info("Job already exists, updating step status")
			return c.updateStepStatusFromJob(ctx, jobFlow, stepName, job)
		}
	}

	// Create job for step
	job, err := c.createJobForStep(ctx, jobFlow, stepSpec)
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
	c.eventRecorder.Eventf(jobFlow, corev1.EventTypeNormal, "StepCreated", "Created Job for step %s", stepName)
	return c.updateJobFlowStatus(ctx, jobFlow, nil)
}

// createJobForStep creates a Kubernetes Job for a step.
func (c *JobFlowController) createJobForStep(ctx context.Context, jobFlow *v1alpha1.JobFlow, step *v1alpha1.Step) (*batchv1.Job, error) {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(step.Name)

	// Get job template from step
	jobTemplate, err := step.GetJobTemplate()
	if err != nil {
		return nil, jferrors.WithStep(jferrors.WithJobFlow(jferrors.Wrapf(err, "template_parse_failed", "failed to get job template"), jobFlow.Namespace, jobFlow.Name), step.Name)
	}

	// Create job
	job := jobTemplate.DeepCopy()
	job.Name = fmt.Sprintf("%s-%s-%s", jobFlow.Name, step.Name, jobFlow.UID[:8])
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

	createdJob, err := c.kubeClient.BatchV1().Jobs(jobFlow.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, jferrors.WithStep(jferrors.WithJobFlow(jferrors.Wrapf(err, "job_create_failed", "failed to create Job"), jobFlow.Namespace, jobFlow.Name), step.Name)
	}

	logger.WithJob(jobFlow.Namespace, createdJob.Name).V(4).Info("Job created successfully")
	return createdJob, nil
}

// updateStepStatusFromJob updates step status based on job status.
func (c *JobFlowController) updateStepStatusFromJob(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepName string, job *batchv1.Job) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(stepName).WithJob(jobFlow.Namespace, job.Name)

	stepStatus := c.getStepStatus(jobFlow.Status, stepName)
	if stepStatus == nil {
		return jferrors.WithStep(jferrors.WithJobFlow(jferrors.New("step_status_not_found", fmt.Sprintf("step status not found: %s", stepName)), jobFlow.Namespace, jobFlow.Name), stepName)
	}

	// Update phase based on job conditions
	if job.Status.Succeeded > 0 {
		stepStatus.Phase = v1alpha1.StepPhaseSucceeded
		now := metav1.Now()
		stepStatus.CompletionTime = &now
		logger.Info("Step succeeded")
		c.metricsRecorder.RecordStepPhase(jobFlow.Name, v1alpha1.StepPhaseSucceeded)
	} else if job.Status.Failed > 0 {
		stepStatus.Phase = v1alpha1.StepPhaseFailed
		now := metav1.Now()
		stepStatus.CompletionTime = &now
		logger.Warning("Step failed")
		c.metricsRecorder.RecordStepPhase(jobFlow.Name, v1alpha1.StepPhaseFailed)
	} else {
		stepStatus.Phase = v1alpha1.StepPhaseRunning
		logger.V(4).Info("Step still running")
	}

	return c.updateJobFlowStatus(ctx, jobFlow, nil)
}

// handleStepFailure handles a step failure.
func (c *JobFlowController) handleStepFailure(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepName string, err error) error {
	logger := logging.FromContext(ctx).WithJobFlow(jobFlow.Namespace, jobFlow.Name).WithStep(stepName).WithError(err)

	stepStatus := c.getStepStatus(jobFlow.Status, stepName)
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
		c.eventRecorder.Eventf(jobFlow, corev1.EventTypeWarning, "StepFailed", "Step %s failed but continuing: %v", stepName, err)
		return c.updateJobFlowStatus(ctx, jobFlow, nil)
	}

	// Mark flow as failed
	jobFlow.Status.Phase = v1alpha1.JobFlowPhaseFailed
	now := metav1.Now()
	jobFlow.Status.CompletionTime = &now
	stepStatus.Phase = v1alpha1.StepPhaseFailed
	stepStatus.Message = err.Error()

	logger.Error("Step failed, marking JobFlow as failed")
	c.eventRecorder.Eventf(jobFlow, corev1.EventTypeWarning, "StepFailed", "Step %s failed: %v", stepName, err)
	return c.updateJobFlowStatus(ctx, jobFlow, nil)
}

// isRetryable checks if an error is retryable.
func (c *JobFlowController) isRetryable(err error) bool {
	return k8serrors.IsConflict(err) || k8serrors.IsServerTimeout(err)
}

// updateJobFlowStatus updates the JobFlow status.
func (c *JobFlowController) updateJobFlowStatus(ctx context.Context, jobFlow *v1alpha1.JobFlow, plan *ExecutionPlan) error {
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
		c.metricsRecorder.RecordJobFlowPhase(jobFlow.Status.Phase, jobFlow.Namespace)
	} else {
		if jobFlow.Status.Phase == v1alpha1.JobFlowPhasePending {
			jobFlow.Status.Phase = v1alpha1.JobFlowPhaseRunning
		}
		c.metricsRecorder.RecordJobFlowPhase(jobFlow.Status.Phase, jobFlow.Namespace)
	}

	// Update conditions
	c.updateConditions(jobFlow)

	// Update status via status updater
	if c.statusUpdater != nil {
		if err := c.statusUpdater.UpdateStatus(ctx, jobFlow); err != nil {
			logger.WithError(err).Error("Failed to update JobFlow status")
			return jferrors.WithJobFlow(jferrors.Wrapf(err, "status_update_failed", "failed to update JobFlow status"), jobFlow.Namespace, jobFlow.Name)
		}
	}

	return nil
}

// updateStatus is a helper to update status with phase and error.
func (c *JobFlowController) updateStatus(jobFlow *v1alpha1.JobFlow, phase string, err error) error {
	logger := logging.NewLogger().WithJobFlow(jobFlow.Namespace, jobFlow.Name)

	jobFlow.Status.Phase = phase
	if err != nil {
		logger.WithError(err).Error("Updating JobFlow status with error")
		jobFlow.Status.Conditions = append(jobFlow.Status.Conditions, v1alpha1.JobFlowCondition{
			Type:               "Ready",
			Status:             corev1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             "Error",
			Message:            err.Error(),
		})
	}
	return err
}

// updateConditions updates JobFlow conditions.
func (c *JobFlowController) updateConditions(jobFlow *v1alpha1.JobFlow) {
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
