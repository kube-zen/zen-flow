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
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

func TestJobFlowController_SetMaxConcurrentReconciles(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.kube-zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	jobInformer := informerFactory.Batch().V1().Jobs()
	pvcInformer := informerFactory.Core().V1().PersistentVolumeClaims()

	metricsRecorder := metrics.NewRecorder()
	eventRecorder := NewEventRecorder(kubeClient)
	statusUpdater := NewStatusUpdater(dynamicClient)

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		pvcInformer,
		statusUpdater,
		metricsRecorder,
		eventRecorder,
	)

	// Test setting max concurrent reconciles
	controller.SetMaxConcurrentReconciles(5)
	if controller.maxConcurrentReconciles != 5 {
		t.Errorf("Expected maxConcurrentReconciles to be 5, got %d", controller.maxConcurrentReconciles)
	}

	controller.SetMaxConcurrentReconciles(10)
	if controller.maxConcurrentReconciles != 10 {
		t.Errorf("Expected maxConcurrentReconciles to be 10, got %d", controller.maxConcurrentReconciles)
	}
}

func TestJobFlowController_refreshStepStatuses(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.kube-zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	jobInformer := informerFactory.Batch().V1().Jobs()
	pvcInformer := informerFactory.Core().V1().PersistentVolumeClaims()

	metricsRecorder := metrics.NewRecorder()
	eventRecorder := NewEventRecorder(kubeClient)
	statusUpdater := NewStatusUpdater(dynamicClient)

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		pvcInformer,
		statusUpdater,
		metricsRecorder,
		eventRecorder,
	)

	ctx := context.Background()

	// Create a JobFlow with a step that has a JobRef
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseRunning,
					JobRef: &corev1.ObjectReference{
						Name:      "step1-job",
						Namespace: "default",
					},
				},
			},
		},
	}

	// Create a Job in the informer
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "step1-job",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}

	// Add Job to informer store
	if err := jobInformer.Informer().GetStore().Add(job); err != nil {
		t.Fatalf("Failed to add job to store: %v", err)
	}

	// Test refreshStepStatuses
	err := controller.refreshStepStatuses(ctx, jobFlow)
	if err != nil {
		t.Fatalf("refreshStepStatuses returned error: %v", err)
	}

	// Verify step status was updated
	stepStatus := controller.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseSucceeded {
		t.Errorf("Expected step phase to be Succeeded, got %s", stepStatus.Phase)
	}
}

func TestJobFlowController_refreshStepStatuses_JobNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.kube-zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	jobInformer := informerFactory.Batch().V1().Jobs()
	pvcInformer := informerFactory.Core().V1().PersistentVolumeClaims()

	metricsRecorder := metrics.NewRecorder()
	eventRecorder := NewEventRecorder(kubeClient)
	statusUpdater := NewStatusUpdater(dynamicClient)

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		pvcInformer,
		statusUpdater,
		metricsRecorder,
		eventRecorder,
	)

	ctx := context.Background()

	// Create a JobFlow with a step that has a JobRef pointing to non-existent job
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseRunning,
					JobRef: &corev1.ObjectReference{
						Name:      "non-existent-job",
						Namespace: "default",
					},
				},
			},
		},
	}

	// Test refreshStepStatuses with non-existent job
	err := controller.refreshStepStatuses(ctx, jobFlow)
	if err != nil {
		t.Fatalf("refreshStepStatuses returned error: %v", err)
	}

	// Verify step status was marked as failed
	stepStatus := controller.getStepStatus(jobFlow.Status, "step1")
	if stepStatus == nil {
		t.Fatal("Step status not found")
	}
	if stepStatus.Phase != v1alpha1.StepPhaseFailed {
		t.Errorf("Expected step phase to be Failed, got %s", stepStatus.Phase)
	}
}

func TestJobFlowController_refreshStepStatusFromJob(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.kube-zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	jobInformer := informerFactory.Batch().V1().Jobs()
	pvcInformer := informerFactory.Core().V1().PersistentVolumeClaims()

	metricsRecorder := metrics.NewRecorder()
	eventRecorder := NewEventRecorder(kubeClient)
	statusUpdater := NewStatusUpdater(dynamicClient)

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		pvcInformer,
		statusUpdater,
		metricsRecorder,
		eventRecorder,
	)

	// Create a JobFlow with a step
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseRunning,
				},
			},
		},
	}

	// Test with succeeded job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "step1-job",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}

	controller.refreshStepStatusFromJob(jobFlow, "step1", job)
	stepStatus := controller.getStepStatus(jobFlow.Status, "step1")
	if stepStatus.Phase != v1alpha1.StepPhaseSucceeded {
		t.Errorf("Expected step phase to be Succeeded, got %s", stepStatus.Phase)
	}

	// Test with failed job
	job.Status.Succeeded = 0
	job.Status.Failed = 1
	jobFlow.Status.Steps[0].Phase = v1alpha1.StepPhaseRunning
	controller.refreshStepStatusFromJob(jobFlow, "step1", job)
	stepStatus = controller.getStepStatus(jobFlow.Status, "step1")
	if stepStatus.Phase != v1alpha1.StepPhaseFailed {
		t.Errorf("Expected step phase to be Failed, got %s", stepStatus.Phase)
	}

	// Test with running job
	job.Status.Failed = 0
	jobFlow.Status.Steps[0].Phase = v1alpha1.StepPhasePending
	controller.refreshStepStatusFromJob(jobFlow, "step1", job)
	stepStatus = controller.getStepStatus(jobFlow.Status, "step1")
	if stepStatus.Phase != v1alpha1.StepPhaseRunning {
		t.Errorf("Expected step phase to be Running, got %s", stepStatus.Phase)
	}
}

func TestJobFlowController_refreshStepStatuses_NoJobRef(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.kube-zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	jobInformer := informerFactory.Batch().V1().Jobs()
	pvcInformer := informerFactory.Core().V1().PersistentVolumeClaims()

	metricsRecorder := metrics.NewRecorder()
	eventRecorder := NewEventRecorder(kubeClient)
	statusUpdater := NewStatusUpdater(dynamicClient)

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		pvcInformer,
		statusUpdater,
		metricsRecorder,
		eventRecorder,
	)

	ctx := context.Background()

	// Create a JobFlow with a step without JobRef
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhasePending,
					// No JobRef
				},
			},
		},
	}

	// Test refreshStepStatuses - should skip step without JobRef
	err := controller.refreshStepStatuses(ctx, jobFlow)
	if err != nil {
		t.Fatalf("refreshStepStatuses returned error: %v", err)
	}
}

