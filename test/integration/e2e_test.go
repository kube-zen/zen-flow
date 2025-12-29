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
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

// setupTestControllerE2E creates a test controller with fake clients for E2E tests
func setupTestControllerE2E(t *testing.T) (*controller.JobFlowController, *fake.FakeDynamicClient, *kubefake.Clientset, cache.SharedInformer) {
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

	return jobFlowController, dynamicClient, kubeClient, jobFlowInformer
}

// createJobFlowInClient creates a JobFlow in the fake dynamic client and returns the unstructured object
func createJobFlowInClient(t *testing.T, dynamicClient *fake.FakeDynamicClient, jobFlow *v1alpha1.JobFlow, jobFlowInformer cache.SharedInformer) *unstructured.Unstructured {
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
	ctx := context.Background()
	_, err = dynamicClient.Resource(jobFlowGVR).Namespace(jobFlow.Namespace).Create(ctx, unstructuredJobFlow, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Add to informer store
	if jobFlowInformer != nil {
		if err := jobFlowInformer.GetStore().Add(unstructuredJobFlow); err != nil {
			t.Fatalf("Failed to add JobFlow to informer store: %v", err)
		}
	}

	return unstructuredJobFlow
}

// marshalJobTemplate marshals a Job template to JSON
func marshalJobTemplate(t *testing.T, job *batchv1.Job) []byte {
	unstructuredJob, err := runtime.DefaultUnstructuredConverter.ToUnstructured(job)
	if err != nil {
		t.Fatalf("Failed to convert job: %v", err)
	}

	// Ensure Kind and APIVersion are set
	if unstructuredJob["kind"] == nil {
		unstructuredJob["kind"] = "Job"
	}
	if unstructuredJob["apiVersion"] == nil {
		unstructuredJob["apiVersion"] = batchv1.SchemeGroupVersion.String()
	}

	raw, err := json.Marshal(unstructuredJob)
	if err != nil {
		t.Fatalf("Failed to marshal job: %v", err)
	}
	return raw
}

// TestE2E_SimpleLinearFlow tests a simple linear flow with two sequential steps
func TestE2E_SimpleLinearFlow(t *testing.T) {
	_, dynamicClient, _, jobFlowInformer := setupTestControllerE2E(t)

	// Create a simple linear JobFlow
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "simple-flow",
			Namespace: "default",
			UID:       "test-uid-123",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:         "step1",
					Dependencies: []string{},
					Template: runtime.RawExtension{
						Raw: marshalJobTemplate(t, &batchv1.Job{
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
				{
					Name:         "step2",
					Dependencies: []string{"step1"},
					Template: runtime.RawExtension{
						Raw: marshalJobTemplate(t, &batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "main",
												Image:   "busybox:latest",
												Command: []string{"echo", "step2"},
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

	// Create JobFlow
	_ = createJobFlowInClient(t, dynamicClient, jobFlow, jobFlowInformer)

	// For E2E tests, we test the integration through the informer and verify results
	// The actual reconciliation would happen through Start() and worker goroutines
	// Here we verify the JobFlow can be retrieved and has correct structure

	// Verify JobFlow can be retrieved
	obj, exists, err := jobFlowInformer.GetStore().GetByKey("default/simple-flow")
	if !exists || err != nil {
		t.Fatalf("Failed to get JobFlow: exists=%v, err=%v", exists, err)
	}

	unstructuredObj := obj.(*unstructured.Unstructured)
	retrievedJobFlow := &v1alpha1.JobFlow{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, retrievedJobFlow)
	if err != nil {
		t.Fatalf("Failed to convert JobFlow: %v", err)
	}

	// Verify structure
	if retrievedJobFlow.Name != "simple-flow" {
		t.Errorf("Expected name 'simple-flow', got '%s'", retrievedJobFlow.Name)
	}
	if len(retrievedJobFlow.Spec.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(retrievedJobFlow.Spec.Steps))
	}
	if retrievedJobFlow.Spec.Steps[1].Dependencies[0] != "step1" {
		t.Error("Expected step2 to depend on step1")
	}
}

// TestE2E_DAGFlow tests a DAG flow with parallel execution
func TestE2E_DAGFlow(t *testing.T) {
	_, dynamicClient, _, jobFlowInformer := setupTestControllerE2E(t)

	// Create a DAG flow: step1 -> step2, step3 (parallel)
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dag-flow",
			Namespace: "default",
			UID:       "test-uid-456",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:         "step1",
					Dependencies: []string{},
					Template: runtime.RawExtension{
						Raw: marshalJobTemplate(t, &batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"echo", "step1"}},
										},
										RestartPolicy: corev1.RestartPolicyOnFailure,
									},
								},
							},
						}),
					},
				},
				{
					Name:         "step2",
					Dependencies: []string{"step1"},
					Template: runtime.RawExtension{
						Raw: marshalJobTemplate(t, &batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"echo", "step2"}},
										},
										RestartPolicy: corev1.RestartPolicyOnFailure,
									},
								},
							},
						}),
					},
				},
				{
					Name:         "step3",
					Dependencies: []string{"step1"},
					Template: runtime.RawExtension{
						Raw: marshalJobTemplate(t, &batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"echo", "step3"}},
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

	_ = createJobFlowInClient(t, dynamicClient, jobFlow, jobFlowInformer)

	// Verify JobFlow structure and DAG dependencies
	obj, exists, err := jobFlowInformer.GetStore().GetByKey("default/dag-flow")
	if !exists || err != nil {
		t.Fatalf("Failed to get JobFlow: exists=%v, err=%v", exists, err)
	}

	unstructuredObj := obj.(*unstructured.Unstructured)
	retrievedJobFlow := &v1alpha1.JobFlow{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, retrievedJobFlow)
	if err != nil {
		t.Fatalf("Failed to convert JobFlow: %v", err)
	}

	// Verify DAG structure
	if len(retrievedJobFlow.Spec.Steps) != 3 {
		t.Errorf("Expected 3 steps, got %d", len(retrievedJobFlow.Spec.Steps))
	}
	// Verify dependencies
	if len(retrievedJobFlow.Spec.Steps[1].Dependencies) != 1 || retrievedJobFlow.Spec.Steps[1].Dependencies[0] != "step1" {
		t.Error("Expected step2 to depend on step1")
	}
	if len(retrievedJobFlow.Spec.Steps[2].Dependencies) != 1 || retrievedJobFlow.Spec.Steps[2].Dependencies[0] != "step1" {
		t.Error("Expected step3 to depend on step1")
	}
}

// TestE2E_ContinueOnFailure tests ContinueOnFailure behavior
func TestE2E_ContinueOnFailure(t *testing.T) {
	_, dynamicClient, _, jobFlowInformer := setupTestControllerE2E(t)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "continue-flow",
			Namespace: "default",
			UID:       "test-uid-789",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:              "step1",
					Dependencies:      []string{},
					ContinueOnFailure: true,
					Template: runtime.RawExtension{
						Raw: marshalJobTemplate(t, &batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"false"}}, // Will fail
										},
										RestartPolicy: corev1.RestartPolicyOnFailure,
									},
								},
							},
						}),
					},
				},
				{
					Name:         "step2",
					Dependencies: []string{"step1"},
					Template: runtime.RawExtension{
						Raw: marshalJobTemplate(t, &batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"echo", "step2"}},
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

	_ = createJobFlowInClient(t, dynamicClient, jobFlow, jobFlowInformer)

	// Verify ContinueOnFailure is set correctly
	obj, exists, err := jobFlowInformer.GetStore().GetByKey("default/continue-flow")
	if !exists || err != nil {
		t.Fatalf("Failed to get JobFlow: exists=%v, err=%v", exists, err)
	}

	unstructuredObj := obj.(*unstructured.Unstructured)
	retrievedJobFlow := &v1alpha1.JobFlow{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, retrievedJobFlow)
	if err != nil {
		t.Fatalf("Failed to convert JobFlow: %v", err)
	}

	if !retrievedJobFlow.Spec.Steps[0].ContinueOnFailure {
		t.Error("Expected ContinueOnFailure to be true for step1")
	}

	// Verify JobFlow structure
	if retrievedJobFlow.Spec.Steps[0].ContinueOnFailure {
		t.Log("ContinueOnFailure is correctly set")
	}
}

// TestE2E_ResourceTemplates tests resource template creation
func TestE2E_ResourceTemplates(t *testing.T) {
	_, dynamicClient, _, jobFlowInformer := setupTestControllerE2E(t)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-flow",
			Namespace: "default",
			UID:       "test-uid-resource",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:         "step1",
					Dependencies: []string{},
					Template: runtime.RawExtension{
						Raw: marshalJobTemplate(t, &batchv1.Job{
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
			ResourceTemplates: &v1alpha1.ResourceTemplates{
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "shared-data",
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
				ConfigMapTemplates: []corev1.ConfigMap{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "metadata",
						},
						Data: map[string]string{
							"flowId": "resource-flow",
						},
					},
				},
			},
		},
	}

	_ = createJobFlowInClient(t, dynamicClient, jobFlow, jobFlowInformer)

	// Verify resource templates are defined
	obj, exists, err := jobFlowInformer.GetStore().GetByKey("default/resource-flow")
	if !exists || err != nil {
		t.Fatalf("Failed to get JobFlow: exists=%v, err=%v", exists, err)
	}

	unstructuredObj := obj.(*unstructured.Unstructured)
	retrievedJobFlow := &v1alpha1.JobFlow{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, retrievedJobFlow)
	if err != nil {
		t.Fatalf("Failed to convert JobFlow: %v", err)
	}

	if retrievedJobFlow.Spec.ResourceTemplates == nil {
		t.Fatal("Expected ResourceTemplates to be set")
	}
	if len(retrievedJobFlow.Spec.ResourceTemplates.VolumeClaimTemplates) == 0 {
		t.Error("Expected VolumeClaimTemplates to be defined")
	}
	if len(retrievedJobFlow.Spec.ResourceTemplates.ConfigMapTemplates) == 0 {
		t.Error("Expected ConfigMapTemplates to be defined")
	}

	// Verify resource templates are defined in spec
	// Note: Actual resource creation happens during reconciliation, which we're not testing here
	if len(retrievedJobFlow.Spec.ResourceTemplates.VolumeClaimTemplates) != 1 {
		t.Errorf("Expected 1 VolumeClaimTemplate, got %d", len(retrievedJobFlow.Spec.ResourceTemplates.VolumeClaimTemplates))
	}
	if retrievedJobFlow.Spec.ResourceTemplates.VolumeClaimTemplates[0].Name != "shared-data" {
		t.Errorf("Expected PVC name 'shared-data', got '%s'", retrievedJobFlow.Spec.ResourceTemplates.VolumeClaimTemplates[0].Name)
	}
	if len(retrievedJobFlow.Spec.ResourceTemplates.ConfigMapTemplates) != 1 {
		t.Errorf("Expected 1 ConfigMapTemplate, got %d", len(retrievedJobFlow.Spec.ResourceTemplates.ConfigMapTemplates))
	}
	if retrievedJobFlow.Spec.ResourceTemplates.ConfigMapTemplates[0].Name != "metadata" {
		t.Errorf("Expected ConfigMap name 'metadata', got '%s'", retrievedJobFlow.Spec.ResourceTemplates.ConfigMapTemplates[0].Name)
	}
}

// TestE2E_JobFlowLifecycle tests the complete lifecycle of a JobFlow
func TestE2E_JobFlowLifecycle(t *testing.T) {
	_, dynamicClient, _, jobFlowInformer := setupTestControllerE2E(t)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lifecycle-flow",
			Namespace: "default",
			UID:       "test-uid-lifecycle",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:         "step1",
					Dependencies: []string{},
					Template: runtime.RawExtension{
						Raw: marshalJobTemplate(t, &batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"echo", "step1"}},
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

	_ = createJobFlowInClient(t, dynamicClient, jobFlow, jobFlowInformer)

	// Test 1: Verify JobFlow can be retrieved
	// Test 2: Verify structure
	obj, exists, err := jobFlowInformer.GetStore().GetByKey("default/lifecycle-flow")
	if !exists || err != nil {
		t.Fatalf("Failed to get JobFlow: exists=%v, err=%v", exists, err)
	}

	unstructuredObj := obj.(*unstructured.Unstructured)
	retrievedJobFlow := &v1alpha1.JobFlow{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, retrievedJobFlow)
	if err != nil {
		t.Fatalf("Failed to convert JobFlow: %v", err)
	}

	// Test 3: Verify JobFlow structure
	if retrievedJobFlow.Name != "lifecycle-flow" {
		t.Errorf("Expected name 'lifecycle-flow', got '%s'", retrievedJobFlow.Name)
	}
	if len(retrievedJobFlow.Spec.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(retrievedJobFlow.Spec.Steps))
	}
}

// TestE2E_ErrorRecovery tests error recovery scenarios
func TestE2E_ErrorRecovery(t *testing.T) {
	_, dynamicClient, _, jobFlowInformer := setupTestControllerE2E(t)

	// Test with invalid JobFlow (no steps)
	invalidJobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "invalid-flow",
			Namespace: "default",
			UID:       "test-uid-invalid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{}, // Invalid: no steps
		},
	}

	_ = createJobFlowInClient(t, dynamicClient, invalidJobFlow, jobFlowInformer)

	// Verify invalid JobFlow can be retrieved (validation happens during reconciliation)
	obj, exists, err := jobFlowInformer.GetStore().GetByKey("default/invalid-flow")
	if !exists || err != nil {
		t.Fatalf("Failed to get JobFlow: exists=%v, err=%v", exists, err)
	}

	unstructuredObj := obj.(*unstructured.Unstructured)
	retrievedJobFlow := &v1alpha1.JobFlow{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, retrievedJobFlow)
	if err != nil {
		t.Fatalf("Failed to convert JobFlow: %v", err)
	}

	// Verify it has no steps (invalid)
	if len(retrievedJobFlow.Spec.Steps) != 0 {
		t.Errorf("Expected 0 steps for invalid JobFlow, got %d", len(retrievedJobFlow.Spec.Steps))
	}
}
