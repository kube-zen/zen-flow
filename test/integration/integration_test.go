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
	"net/http"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kubefake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

// setupTestReconciler creates a test reconciler with fake clients
func setupTestReconciler(t *testing.T) (*controller.JobFlowReconciler, client.Client, *kubefake.Clientset) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
	if err := batchv1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add batch scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add core scheme: %v", err)
	}

	fakeClient := clientfake.NewClientBuilder().WithScheme(scheme).Build()
	kubeClient := kubefake.NewSimpleClientset()

	metricsRecorder := metrics.NewRecorder()
	eventRecorder := controller.NewEventRecorder(kubeClient)

	// Create a fake manager
	mgr := &fakeManager{client: fakeClient, scheme: scheme}
	reconciler := controller.NewJobFlowReconciler(mgr, metricsRecorder, eventRecorder)

	return reconciler, fakeClient, kubeClient
}

// fakeManager implements ctrl.Manager for testing
type fakeManager struct {
	client client.Client
	scheme *runtime.Scheme
}

func (f *fakeManager) GetClient() client.Client {
	return f.client
}

func (f *fakeManager) GetScheme() *runtime.Scheme {
	return f.scheme
}

func (f *fakeManager) GetEventRecorderFor(name string) record.EventRecorder {
	return &fakeEventRecorder{}
}

func (f *fakeManager) GetRESTMapper() meta.RESTMapper {
	return &fakeRESTMapper{}
}

func (f *fakeManager) GetAPIReader() client.Reader {
	return f.client
}

func (f *fakeManager) GetConfig() *rest.Config {
	panic("not implemented")
}

func (f *fakeManager) GetCache() cache.Cache {
	panic("not implemented")
}

func (f *fakeManager) GetFieldIndexer() client.FieldIndexer {
	panic("not implemented")
}

func (f *fakeManager) Start(ctx context.Context) error {
	panic("not implemented")
}

func (f *fakeManager) Add(r manager.Runnable) error {
	return nil
}

func (f *fakeManager) Elected() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

func (f *fakeManager) AddMetricsServerExtraHandler(path string, handler http.Handler) error {
	return nil
}

func (f *fakeManager) AddHealthzCheck(name string, check healthz.Checker) error {
	return nil
}

func (f *fakeManager) AddReadyzCheck(name string, check healthz.Checker) error {
	return nil
}

func (f *fakeManager) GetWebhookServer() webhook.Server {
	panic("not implemented")
}

func (f *fakeManager) GetLogger() logr.Logger {
	return logr.Discard()
}

func (f *fakeManager) GetControllerOptions() config.Controller {
	return config.Controller{}
}

func (f *fakeManager) GetHTTPClient() *http.Client {
	return &http.Client{}
}

type fakeEventRecorder struct{}

func (f *fakeEventRecorder) Event(object runtime.Object, eventtype, reason, message string) {}
func (f *fakeEventRecorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {}
func (f *fakeEventRecorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {}

type fakeRESTMapper struct{}

func (f *fakeRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	panic("not implemented")
}
func (f *fakeRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	panic("not implemented")
}
func (f *fakeRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	panic("not implemented")
}
func (f *fakeRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	panic("not implemented")
}
func (f *fakeRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	panic("not implemented")
}
func (f *fakeRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	panic("not implemented")
}
func (f *fakeRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	panic("not implemented")
}

func mustMarshalJob(job *batchv1.Job) []byte {
	data, err := json.Marshal(job)
	if err != nil {
		panic(err)
	}
	return data
}

func TestIntegration_SimpleLinearFlow(t *testing.T) {
	reconciler, fakeClient, _ := setupTestReconciler(t)
	ctx := context.Background()

	// Create a simple linear JobFlow
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
												Command: []string{"sh", "-c"},
												Args:    []string{"echo 'Step 1' && sleep 1"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
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
						Raw: mustMarshalJob(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "main",
												Image:   "busybox:latest",
												Command: []string{"sh", "-c"},
												Args:    []string{"echo 'Step 2' && sleep 1"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
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
	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-flow",
			Namespace: "default",
		},
	}

	result, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify result
	if result.Requeue {
		t.Log("Reconcile requested requeue (expected for initial setup)")
	}

	// Verify JobFlow status was initialized
	updated := &v1alpha1.JobFlow{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatalf("Failed to get updated JobFlow: %v", err)
	}

	if updated.Status.Phase == "" {
		t.Error("Expected phase to be set")
	}
	if updated.Status.StartTime == nil {
		t.Error("Expected StartTime to be set")
	}
	if updated.Status.Progress == nil {
		t.Error("Expected Progress to be set")
	}
	if updated.Status.Progress.TotalSteps != 2 {
		t.Errorf("Expected TotalSteps 2, got %d", updated.Status.Progress.TotalSteps)
	}

	// Verify step1 Job was created
	jobList := &batchv1.JobList{}
	if err := fakeClient.List(ctx, jobList, client.InNamespace("default")); err != nil {
		t.Fatalf("Failed to list Jobs: %v", err)
	}

	// Should have at least one Job created for step1
	if len(jobList.Items) == 0 {
		t.Error("Expected at least one Job to be created")
	}

	// Verify Jobs have correct labels
	for _, job := range jobList.Items {
		if job.Labels["workflow.kube-zen.io/flow"] != "test-flow" {
			t.Errorf("Expected job to have flow label, got: %v", job.Labels)
		}
		if job.Labels["workflow.kube-zen.io/managed-by"] != "zen-flow" {
			t.Errorf("Expected job to have managed-by label, got: %v", job.Labels)
		}
	}
}

func TestIntegration_DAGFlow(t *testing.T) {
	reconciler, fakeClient, _ := setupTestReconciler(t)
	ctx := context.Background()

	// Create a DAG flow: step1 -> step2, step1 -> step3
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
						Raw: mustMarshalJob(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"echo", "step1"}},
										},
										RestartPolicy: corev1.RestartPolicyNever,
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
						Raw: mustMarshalJob(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"echo", "step2"}},
										},
										RestartPolicy: corev1.RestartPolicyNever,
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
						Raw: mustMarshalJob(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"echo", "step3"}},
										},
										RestartPolicy: corev1.RestartPolicyNever,
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
	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "dag-flow",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify JobFlow was initialized
	updated := &v1alpha1.JobFlow{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatalf("Failed to get updated JobFlow: %v", err)
	}

	if updated.Status.Progress.TotalSteps != 3 {
		t.Errorf("Expected TotalSteps 3, got %d", updated.Status.Progress.TotalSteps)
	}

	// Verify step1 Job was created (it has no dependencies)
	jobList := &batchv1.JobList{}
	if err := fakeClient.List(ctx, jobList, client.InNamespace("default")); err != nil {
		t.Fatalf("Failed to list Jobs: %v", err)
	}

	if len(jobList.Items) == 0 {
		t.Error("Expected at least one Job to be created for step1")
	}
}

func TestIntegration_ContinueOnFailure(t *testing.T) {
	reconciler, fakeClient, _ := setupTestReconciler(t)
	ctx := context.Background()

	// Create a flow where step1 fails but step2 continues
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "continue-flow",
			Namespace: "default",
			UID:       "test-uid-789",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:             "step1",
					Dependencies:     []string{},
					ContinueOnFailure: true,
					Template: runtime.RawExtension{
						Raw: mustMarshalJob(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"false"}}, // Will fail
										},
										RestartPolicy: corev1.RestartPolicyNever,
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
						Raw: mustMarshalJob(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"echo", "step2"}},
										},
										RestartPolicy: corev1.RestartPolicyNever,
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
	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "continue-flow",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify JobFlow was initialized
	updated := &v1alpha1.JobFlow{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatalf("Failed to get updated JobFlow: %v", err)
	}

	if updated.Status.Progress.TotalSteps != 2 {
		t.Errorf("Expected TotalSteps 2, got %d", updated.Status.Progress.TotalSteps)
	}
}

func TestIntegration_JobFlowLifecycle(t *testing.T) {
	reconciler, fakeClient, _ := setupTestReconciler(t)
	ctx := context.Background()

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
						Raw: mustMarshalJob(&batchv1.Job{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{Name: "main", Image: "busybox:latest", Command: []string{"echo", "step1"}},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						}),
					},
				},
			},
		},
	}

	// Test 1: Create JobFlow
	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Test 2: Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "lifecycle-flow",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Test 3: Verify JobFlow exists and has status
	updated := &v1alpha1.JobFlow{}
	if err := fakeClient.Get(ctx, req.NamespacedName, updated); err != nil {
		t.Fatalf("Failed to get JobFlow: %v", err)
	}

	if updated.Name != "lifecycle-flow" {
		t.Errorf("Expected name 'lifecycle-flow', got '%s'", updated.Name)
	}
	if updated.Status.Phase == "" {
		t.Error("Expected phase to be set")
	}
}

