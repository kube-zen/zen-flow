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

package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

func setupTestController(t *testing.T) (*controller.JobFlowController, *fake.FakeDynamicClient, *kubefake.Clientset) {
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

	statusUpdater := controller.NewStatusUpdater(dynamicClient)
	metricsRecorder := metrics.NewRecorder()
	eventRecorder := controller.NewEventRecorder(kubeClient)

	jobFlowController := controller.NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		pvcInformer,
		statusUpdater,
		metricsRecorder,
		eventRecorder,
	)

	return jobFlowController, dynamicClient, kubeClient
}

func TestJobFlowController_Integration_SimpleFlow(t *testing.T) {
	_, dynamicClient, _ := setupTestController(t)
	ctx := context.Background()

	// Create a simple JobFlow
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:         "step1",
					Dependencies: []string{},
					Template: runtime.RawExtension{
						Raw: mustMarshalJob(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "main",
												Image:   "busybox:latest",
												Command: []string{"echo", "step1"},
											},
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
	}

	// Convert to unstructured and add to fake client
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobFlow)
	if err != nil {
		t.Fatalf("Failed to convert to unstructured: %v", err)
	}
	unstructuredJobFlow := &unstructured.Unstructured{Object: unstructuredObj}
	unstructuredJobFlow.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("JobFlow"))

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.kube-zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	_, err = dynamicClient.Resource(jobFlowGVR).Namespace("default").Create(ctx, unstructuredJobFlow, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Test that JobFlow was created successfully
	// Note: We can't access private informer fields, so we verify through the client
	retrievedUnstructured, err := dynamicClient.Resource(jobFlowGVR).Namespace("default").Get(ctx, "test-flow", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get JobFlow from client: %v", err)
	}

	// Verify we can convert it back
	retrievedJobFlow := &v1alpha1.JobFlow{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(retrievedUnstructured.Object, retrievedJobFlow)
	if err != nil {
		t.Fatalf("Failed to convert JobFlow: %v", err)
	}

	// Verify basic properties
	if retrievedJobFlow.Name != "test-flow" {
		t.Errorf("Expected name 'test-flow', got '%s'", retrievedJobFlow.Name)
	}
	if retrievedJobFlow.Namespace != "default" {
		t.Errorf("Expected namespace 'default', got '%s'", retrievedJobFlow.Namespace)
	}
	if len(retrievedJobFlow.Spec.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(retrievedJobFlow.Spec.Steps))
	}
}

func mustMarshalJob(job *batchv1.Job) []byte {
	unstructuredJob, err := runtime.DefaultUnstructuredConverter.ToUnstructured(job)
	if err != nil {
		panic(err)
	}
	// Ensure Kind and APIVersion are set
	if unstructuredJob["kind"] == nil {
		unstructuredJob["kind"] = "Job"
	}
	if unstructuredJob["apiVersion"] == nil {
		unstructuredJob["apiVersion"] = batchv1.SchemeGroupVersion.String()
	}
	// Use json.Marshal instead of runtime.Encode
	raw, err := json.Marshal(unstructuredJob)
	if err != nil {
		panic(err)
	}
	return raw
}
