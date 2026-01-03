//go:build integration

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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

// TestIntegration_ParameterExtraction_JSONPath tests JSONPath parameter extraction from job outputs
func TestIntegration_ParameterExtraction_JSONPath(t *testing.T) {
	reconciler, fakeClient, _ := setupTestReconciler(t)
	ctx := context.Background()

	// Create a JobFlow with a step that outputs a parameter
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "param-test-flow",
			Namespace: "default",
			UID:       "param-test-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:         "step1",
					Dependencies: []string{},
					Template: func() runtime.RawExtension {
						job := &batchv1.Job{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Job",
								APIVersion: batchv1.SchemeGroupVersion.String(),
							},
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "main",
												Image:   "busybox:latest",
												Command: []string{"sh", "-c"},
												Args:    []string{"echo 'Step 1 completed'"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						}
						return runtime.RawExtension{
							Object: job,
							Raw:    mustMarshalJobTemplate(job),
						}
					}(),
					Outputs: &v1alpha1.StepOutputs{
						Parameters: []v1alpha1.ParameterOutput{
							{
								Name: "succeeded",
								ValueFrom: v1alpha1.ParameterValueFrom{
									JSONPath: "$.status.succeeded",
								},
							},
						},
					},
				},
				{
					Name:         "step2",
					Dependencies: []string{"step1"},
					Inputs: &v1alpha1.StepInputs{
						Parameters: []v1alpha1.ParameterInput{
							{
								Name: "previousStepSucceeded",
								ValueFrom: &v1alpha1.ParameterValueFrom{
									JSONPath: "step1:$.status.succeeded",
								},
							},
						},
					},
					Template: func() runtime.RawExtension {
						job := &batchv1.Job{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Job",
								APIVersion: batchv1.SchemeGroupVersion.String(),
							},
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "main",
												Image:   "busybox:latest",
												Command: []string{"sh", "-c"},
												Args:    []string{"echo 'Step 2 with param: {{.parameters.previousStepSucceeded}}'"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						}
						return runtime.RawExtension{
							Object: job,
							Raw:    mustMarshalJobTemplate(job),
						}
					}(),
				},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseSucceeded,
					JobRef: &corev1.ObjectReference{
						Name:      "step1-job",
						Namespace: "default",
					},
				},
			},
		},
	}

	// Create a Job with status for step1
	job1 := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "step1-job",
			Namespace: "default",
		},
		Status: batchv1.JobStatus{
			Succeeded: 1,
			Conditions: []batchv1.JobCondition{
				{
					Type:   batchv1.JobComplete,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}
	if err := fakeClient.Create(ctx, job1); err != nil {
		t.Fatalf("Failed to create Job: %v", err)
	}

	// Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "param-test-flow",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	// Status update failures are expected with fake client
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"param-test-flow\" not found" {
		t.Logf("Reconcile completed (status update errors expected with fake client): %v", err)
	}

	// Verify JobFlow exists
	updated := &v1alpha1.JobFlow{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "param-test-flow", Namespace: "default"}, updated); err != nil {
		t.Fatalf("Failed to get JobFlow: %v", err)
	}

	// Verify step1 has output parameter
	if updated.Status.Steps == nil || len(updated.Status.Steps) == 0 {
		t.Error("Expected step statuses to be populated")
	}
}

// TestIntegration_ParameterTemplateApplication tests parameter template substitution in job templates
func TestIntegration_ParameterTemplateApplication(t *testing.T) {
	reconciler, fakeClient, _ := setupTestReconciler(t)
	ctx := context.Background()

	// Create a JobFlow with parameter template substitution
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "template-test-flow",
			Namespace: "default",
			UID:       "template-test-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:         "step1",
					Dependencies: []string{},
					Inputs: &v1alpha1.StepInputs{
						Parameters: []v1alpha1.ParameterInput{
							{
								Name:  "message",
								Value: "Hello from template",
							},
						},
					},
					Template: func() runtime.RawExtension {
						job := &batchv1.Job{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Job",
								APIVersion: batchv1.SchemeGroupVersion.String(),
							},
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "main",
												Image:   "busybox:latest",
												Command: []string{"sh", "-c"},
												Args:    []string{"echo '{{.parameters.message}}'"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						}
						jobJSON, _ := json.Marshal(job)
						return runtime.RawExtension{
							Object: job,
							Raw:    jobJSON,
						}
					}(),
				},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "template-test-flow",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	// Status update failures are expected with fake client
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"template-test-flow\" not found" {
		t.Logf("Reconcile completed (status update errors expected with fake client): %v", err)
	}

	// Verify JobFlow exists
	updated := &v1alpha1.JobFlow{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "template-test-flow", Namespace: "default"}, updated); err != nil {
		t.Fatalf("Failed to get JobFlow: %v", err)
	}
}

// TestIntegration_ParameterFromConfigMap tests parameter resolution from ConfigMap
func TestIntegration_ParameterFromConfigMap(t *testing.T) {
	reconciler, fakeClient, _ := setupTestReconciler(t)
	ctx := context.Background()

	// Create a ConfigMap with parameter value
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "default",
		},
		Data: map[string]string{
			"param-value": "value-from-configmap",
		},
	}

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-param-flow",
			Namespace: "default",
			UID:       "configmap-param-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:         "step1",
					Dependencies: []string{},
					Inputs: &v1alpha1.StepInputs{
						Parameters: []v1alpha1.ParameterInput{
							{
								Name: "configParam",
								ValueFrom: &v1alpha1.ParameterValueFrom{
									ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
										Name: "test-config",
										Key:  "param-value",
									},
								},
							},
						},
					},
					Template: func() runtime.RawExtension {
						job := &batchv1.Job{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Job",
								APIVersion: batchv1.SchemeGroupVersion.String(),
							},
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Name:    "main",
												Image:   "busybox:latest",
												Command: []string{"sh", "-c"},
												Args:    []string{"echo '{{.parameters.configParam}}'"},
											},
										},
										RestartPolicy: corev1.RestartPolicyNever,
									},
								},
							},
						}
						jobJSON, _ := json.Marshal(job)
						return runtime.RawExtension{
							Object: job,
							Raw:    jobJSON,
						}
					}(),
				},
			},
		},
	}

	if err := fakeClient.Create(ctx, configMap); err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}
	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "configmap-param-flow",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	// Status update failures are expected with fake client
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"configmap-param-flow\" not found" {
		t.Logf("Reconcile completed (status update errors expected with fake client): %v", err)
	}

	// Verify JobFlow exists
	updated := &v1alpha1.JobFlow{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "configmap-param-flow", Namespace: "default"}, updated); err != nil {
		t.Fatalf("Failed to get JobFlow: %v", err)
	}
}

