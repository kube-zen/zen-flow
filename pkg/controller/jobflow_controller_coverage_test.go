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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

func TestJobFlowController_updateStepStatusFromJob(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add scheme: %v", err)
	}
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
			Steps: []v1alpha1.StepStatus{
				{Name: "step1", Phase: v1alpha1.StepPhaseRunning},
			},
		},
	}

	// Add JobFlow to fake client
	if err := setupJobFlowInFakeClient(dynamicClient, jobFlow); err != nil {
		t.Fatalf("Failed to setup JobFlow in fake client: %v", err)
	}

	tests := []struct {
		name string
		job  *batchv1.Job
		want string
	}{
		{
			name: "succeeded job",
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Status: batchv1.JobStatus{
					Succeeded: 1,
				},
			},
			want: v1alpha1.StepPhaseSucceeded,
		},
		{
			name: "failed job",
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Status: batchv1.JobStatus{
					Failed: 1,
				},
			},
			want: v1alpha1.StepPhaseFailed,
		},
		{
			name: "running job",
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
			want: v1alpha1.StepPhaseRunning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := controller.updateStepStatusFromJob(ctx, jobFlow, "step1", tt.job)
			if err != nil {
				t.Errorf("updateStepStatusFromJob() error = %v", err)
			}

			stepStatus := controller.getStepStatus(jobFlow.Status, "step1")
			if stepStatus == nil {
				t.Fatal("Step status not found")
			}
			if stepStatus.Phase != tt.want {
				t.Errorf("Expected phase %s, got %s", tt.want, stepStatus.Phase)
			}
		})
	}
}

func TestJobFlowController_createJobForStep_WithMetadata(t *testing.T) {
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
			UID:       "test-uid-12345678",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Metadata: &v1alpha1.StepMetadata{
						Labels: map[string]string{
							"custom-label": "custom-value",
						},
						Annotations: map[string]string{
							"custom-annotation": "custom-annotation-value",
						},
					},
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
	}

	ctx := context.Background()
	step := &jobFlow.Spec.Steps[0]
	job, err := controller.createJobForStep(ctx, jobFlow, step)
	if err != nil {
		t.Fatalf("createJobForStep() error = %v", err)
	}

	// Verify job name format
	expectedPrefix := "test-flow-step1-"
	if len(job.Name) <= len(expectedPrefix) || job.Name[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Job name should start with %s, got %s", expectedPrefix, job.Name)
	}

	// Verify labels
	if job.Labels["workflow.kube-zen.io/step"] != "step1" {
		t.Errorf("Expected workflow.kube-zen.io/step label, got %v", job.Labels)
	}
	if job.Labels["custom-label"] != "custom-value" {
		t.Errorf("Expected custom-label, got %v", job.Labels)
	}

	// Verify annotations
	if job.Annotations["custom-annotation"] != "custom-annotation-value" {
		t.Errorf("Expected custom-annotation, got %v", job.Annotations)
	}

	// Verify owner reference
	if len(job.OwnerReferences) != 1 {
		t.Errorf("Expected 1 owner reference, got %d", len(job.OwnerReferences))
	}
	if job.OwnerReferences[0].Kind != "JobFlow" {
		t.Errorf("Expected owner reference kind JobFlow, got %s", job.OwnerReferences[0].Kind)
	}
}

func TestJobFlowController_createResourceTemplates(t *testing.T) {
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
							"flowId": "test-flow",
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	err := controller.createResourceTemplates(ctx, jobFlow)
	if err != nil {
		t.Fatalf("createResourceTemplates() error = %v", err)
	}

	// Verify PVC was created
	pvc, err := kubeClient.CoreV1().PersistentVolumeClaims("default").Get(ctx, "test-flow-shared-data", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Failed to get PVC: %v", err)
	}
	if pvc == nil {
		t.Error("PVC was not created")
	}

	// Verify ConfigMap was created
	cm, err := kubeClient.CoreV1().ConfigMaps("default").Get(ctx, "test-flow-metadata", metav1.GetOptions{})
	if err != nil {
		t.Errorf("Failed to get ConfigMap: %v", err)
	}
	if cm == nil {
		t.Error("ConfigMap was not created")
	}
}

func TestJobFlowController_createResourceTemplates_AlreadyExists(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	// Pre-create PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow-shared-data",
			Namespace: "default",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		},
	}
	_, _ = kubeClient.CoreV1().PersistentVolumeClaims("default").Create(context.Background(), pvc, metav1.CreateOptions{})

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
			ResourceTemplates: &v1alpha1.ResourceTemplates{
				VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "shared-data",
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
	// Should not error if PVC already exists
	if err != nil {
		t.Errorf("createResourceTemplates() should not error when PVC exists: %v", err)
	}
}

// mustMarshalJobTemplate is now in test_helpers.go
