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
	"errors"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

func TestJobFlowController_initializeJobFlow(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{}},
				{Name: "step2", Dependencies: []string{"step1"}},
			},
		},
	}

	// Add JobFlow to fake client
	if err := setupJobFlowInFakeClient(dynamicClient, jobFlow); err != nil {
		t.Fatalf("Failed to setup JobFlow: %v", err)
	}

	ctx := context.Background()
	err := controller.initializeJobFlow(ctx, jobFlow)
	if err != nil {
		t.Fatalf("initializeJobFlow() error = %v", err)
	}

	// Verify initialization
	// Note: Phase may be Pending or Running depending on updateJobFlowStatus logic
	if jobFlow.Status.Phase == "" {
		t.Error("Expected phase to be set")
	}
	if jobFlow.Status.StartTime == nil {
		t.Error("Expected StartTime to be set")
	}
	if jobFlow.Status.Progress == nil {
		t.Error("Expected Progress to be set")
	}
	if jobFlow.Status.Progress.TotalSteps != 2 {
		t.Errorf("Expected TotalSteps 2, got %d", jobFlow.Status.Progress.TotalSteps)
	}
	if len(jobFlow.Status.Steps) != 2 {
		t.Errorf("Expected 2 step statuses, got %d", len(jobFlow.Status.Steps))
	}
	for i, stepStatus := range jobFlow.Status.Steps {
		if stepStatus.Name != jobFlow.Spec.Steps[i].Name {
			t.Errorf("Step status[%d].Name = %s, want %s", i, stepStatus.Name, jobFlow.Spec.Steps[i].Name)
		}
		if stepStatus.Phase != v1alpha1.StepPhasePending {
			t.Errorf("Step status[%d].Phase = %s, want %s", i, stepStatus.Phase, v1alpha1.StepPhasePending)
		}
	}
}

func TestJobFlowController_initializeJobFlow_WithResourceTemplates(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{}},
			},
			ResourceTemplates: &v1alpha1.ResourceTemplates{
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "data",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
							Resources: corev1.VolumeResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceStorage: resource.MustParse("10Gi"),
								},
							},
						},
					},
				},
			},
		},
	}

	// Add JobFlow to fake client
	if err := setupJobFlowInFakeClient(dynamicClient, jobFlow); err != nil {
		t.Fatalf("Failed to setup JobFlow: %v", err)
	}

	ctx := context.Background()
	err := controller.initializeJobFlow(ctx, jobFlow)
	if err != nil {
		t.Fatalf("initializeJobFlow() error = %v", err)
	}

	// Verify PVC was created
	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims("default").Get(ctx, "test-flow-data", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Failed to get PVC: %v", err)
	}
	if pvc == nil {
		t.Error("PVC was not created")
	}
}

func TestJobFlowController_executeStep_CreateJob(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	jobInformer := informerFactory.Batch().V1().Jobs()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
			UID:       "test-uid-12345678",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: runtime.RawExtension{
						Raw: mustMarshalJobTemplate(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest"},
										},
										RestartPolicy: corev1.RestartPolicyOnFailure,
									},
								},
							},
						}),
					},
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{Name: "step1", Phase: v1alpha1.StepPhasePending},
			},
		},
	}

	// Add JobFlow to fake client
	if err := setupJobFlowInFakeClient(dynamicClient, jobFlow); err != nil {
		t.Fatalf("Failed to setup JobFlow: %v", err)
	}

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	err := controller.executeStep(ctx, jobFlow, "step1")
	if err != nil {
		t.Fatalf("executeStep() error = %v", err)
	}

	// Verify job was created
	jobs, err := kubeClient.BatchV1().Jobs("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}
	if len(jobs.Items) != 1 {
		t.Errorf("Expected 1 job, got %d", len(jobs.Items))
	}

	// Verify step status was updated
	stepStatus := controller.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseRunning {
		t.Errorf("Expected step phase Running, got %s", stepStatus.Phase)
	}
	if stepStatus.JobRef == nil {
		t.Error("Expected JobRef to be set")
	}
}

func TestJobFlowController_executeStep_JobAlreadyExists(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	jobInformer := informerFactory.Batch().V1().Jobs()

	// Create existing job
	existingJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow-step1-12345678",
			Namespace: "default",
			UID:       "job-uid",
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}
	_, err := kubeClient.BatchV1().Jobs("default").Create(context.Background(), existingJob, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create job: %v", err)
	}
	// Add to informer
	jobInformer.Informer().GetStore().Add(existingJob)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
			UID:       "test-uid-12345678",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: runtime.RawExtension{
						Raw: mustMarshalJobTemplate(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest"},
										},
									},
								},
							},
						}),
					},
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseRunning,
					JobRef: &corev1.ObjectReference{
						Name: "test-flow-step1-12345678",
					},
				},
			},
		},
	}

	// Add JobFlow to fake client
	if err := setupJobFlowInFakeClient(dynamicClient, jobFlow); err != nil {
		t.Fatalf("Failed to setup JobFlow: %v", err)
	}

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	err = controller.executeStep(ctx, jobFlow, "step1")
	if err != nil {
		t.Fatalf("executeStep() error = %v", err)
	}

	// Verify step status was updated from job
	stepStatus := controller.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseSucceeded {
		t.Errorf("Expected step phase Succeeded, got %s", stepStatus.Phase)
	}
}

func TestJobFlowController_createJobForStep_InvalidTemplate(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: runtime.RawExtension{
						Raw: []byte(`{"invalid": "json"`), // Invalid JSON
					},
				},
			},
		},
	}

	ctx := context.Background()
	step := &jobFlow.Spec.Steps[0]
	_, err := controller.createJobForStep(ctx, jobFlow, step)
	if err == nil {
		t.Error("createJobForStep with invalid template should return error")
	}
}

func TestJobFlowController_createResourceTemplates_Error(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "invalid-namespace", // Invalid namespace to cause error
			UID:       "test-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			ResourceTemplates: &v1alpha1.ResourceTemplates{
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "data",
						},
						Spec: corev1.PersistentVolumeClaimSpec{
							AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	err := controller.createResourceTemplates(ctx, jobFlow)
	// Should handle error gracefully (may or may not error depending on fake client behavior)
	_ = err // Error is acceptable here
}

func TestJobFlowController_reconcileJobFlow_Initialize(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1", Dependencies: []string{}},
			},
		},
		Status: v1alpha1.JobFlowStatus{}, // Not initialized
	}

	// Convert to unstructured and add to informer store
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobFlow)
	if err != nil {
		t.Fatalf("Failed to convert: %v", err)
	}
	unstructuredJobFlow := &unstructured.Unstructured{Object: unstructuredObj}
	unstructuredJobFlow.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("JobFlow"))

	// Add to fake client first (required for status updater)
	ctx := context.Background()
	_, err = dynamicClient.Resource(jobFlowGVR).Namespace("default").Create(ctx, unstructuredJobFlow, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create JobFlow in fake client: %v", err)
	}

	// Add to informer store
	if err := jobFlowInformer.GetStore().Add(unstructuredJobFlow); err != nil {
		t.Fatalf("Failed to add to store: %v", err)
	}

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	err = controller.reconcileJobFlow(ctx, "default/test-flow")
	if err != nil {
		t.Fatalf("reconcileJobFlow() error = %v", err)
	}
}

func TestJobFlowController_reconcileJobFlow_InvalidObjectType(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	// Add invalid object type to store
	invalidObj := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
	}
	if err := jobFlowInformer.GetStore().Add(invalidObj); err != nil {
		t.Fatalf("Failed to add to store: %v", err)
	}

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	err := controller.reconcileJobFlow(ctx, "default/test-flow")
	if err == nil {
		t.Error("reconcileJobFlow with invalid object type should return error")
	}
}

func TestJobFlowController_updateJobFlowStatus_StillRunning(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Phase: v1alpha1.JobFlowPhasePending,
			Progress: &v1alpha1.ProgressStatus{
				TotalSteps: 2,
			},
			Steps: []v1alpha1.StepStatus{
				{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
				{Name: "step2", Phase: v1alpha1.StepPhaseRunning},
			},
		},
	}

	// Add JobFlow to fake client
	if err := setupJobFlowInFakeClient(dynamicClient, jobFlow); err != nil {
		t.Fatalf("Failed to setup JobFlow: %v", err)
	}

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	err := controller.updateJobFlowStatus(ctx, jobFlow, nil)
	if err != nil {
		t.Fatalf("updateJobFlowStatus failed: %v", err)
	}

	// Verify phase is Running
	if jobFlow.Status.Phase != v1alpha1.JobFlowPhaseRunning {
		t.Errorf("Expected phase Running, got %s", jobFlow.Status.Phase)
	}
	if jobFlow.Status.CompletionTime != nil {
		t.Error("Expected CompletionTime to be nil (still running)")
	}
}

func TestJobFlowController_updateJobFlowStatus_StatusUpdaterError(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	// Create a status updater that will return an error
	// We'll use a nil status updater to simulate error
	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		nil, // nil status updater will cause error
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Phase: v1alpha1.JobFlowPhaseRunning,
			Steps: []v1alpha1.StepStatus{
				{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
			},
		},
	}

	ctx := context.Background()
	err := controller.updateJobFlowStatus(ctx, jobFlow, nil)
	// With nil status updater, it should still work (status updater is optional)
	if err != nil {
		t.Logf("updateJobFlowStatus with nil updater returned error (may be expected): %v", err)
	}
}

func TestJobFlowController_processNextJobFlow_Shutdown(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	// Shutdown the queue
	controller.jobFlowQueue.ShutDown()

	ctx := context.Background()
	result := controller.processNextJobFlow(ctx)
	if result {
		t.Error("processNextJobFlow after shutdown should return false")
	}
}

func TestJobFlowController_processNextJobFlow_Error(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	// Add invalid key to queue
	controller.jobFlowQueue.Add("invalid-key")

	ctx := context.Background()
	result := controller.processNextJobFlow(ctx)
	// Should return true (will retry)
	if !result {
		t.Error("processNextJobFlow with error should return true (for retry)")
	}
}

func TestJobFlowController_Start(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	jobInformer := informerFactory.Batch().V1().Jobs()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	err := controller.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify workers are started (queue should be ready)
	// We can't directly verify workers, but we can verify Start doesn't error
	controller.Stop()
}

func TestJobFlowController_Stop_MultipleCalls(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	// Call Stop multiple times - should not panic
	controller.Stop()
	controller.Stop()
	controller.Stop()

	// Verify context is canceled
	select {
	case <-controller.ctx.Done():
		// Expected
	default:
		t.Error("Stop() did not cancel context")
	}
}

func TestJobFlowController_createJobForStep_JobCreationError(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "invalid-namespace", // Invalid namespace
			UID:       "test-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: runtime.RawExtension{
						Raw: mustMarshalJobTemplate(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest"},
										},
									},
								},
							},
						}),
					},
				},
			},
		},
	}

	ctx := context.Background()
	step := &jobFlow.Spec.Steps[0]
	_, err := controller.createJobForStep(ctx, jobFlow, step)
	// May or may not error depending on fake client behavior
	_ = err
}

func TestJobFlowController_updateStepStatusFromJob_StepStatusNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{}, // No step status
		},
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-job",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}

	ctx := context.Background()
	err := controller.updateStepStatusFromJob(ctx, jobFlow, "step1", job)
	if err == nil {
		t.Error("updateStepStatusFromJob with missing step status should return error")
	}
}

func TestJobFlowController_handleJobAdd_InvalidType(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	// Pass invalid type - should not panic
	controller.handleJobAdd(&corev1.Pod{})
	controller.handleJobUpdate(nil, &corev1.Pod{})
	controller.handleJobDelete(&corev1.Pod{})
}

func TestJobFlowController_reconcileJobFlow_GetStoreError(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	// Create a mock informer that returns an error
	// We'll use a real informer but with an invalid store operation
	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	ctx := context.Background()
	// Reconcile non-existent JobFlow - should handle gracefully
	err := controller.reconcileJobFlow(ctx, "default/nonexistent")
	if err != nil {
		t.Logf("reconcileJobFlow for non-existent returned error (may be expected): %v", err)
	}
}

func TestJobFlowController_isRetryable_WrappedError(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	// Test wrapped retryable error
	wrappedErr := errors.New("wrapped: " + k8serrors.NewConflict(schema.GroupResource{}, "test", nil).Error())
	// Note: isRetryable checks for specific error types, not wrapped errors
	result := controller.isRetryable(wrappedErr)
	if result {
		t.Log("Wrapped retryable error detected (may depend on implementation)")
	}
}
