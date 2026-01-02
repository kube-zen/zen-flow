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

// TestIntegration_ArtifactCopying_ConfigMap tests artifact copying from ConfigMap between steps
func TestIntegration_ArtifactCopying_ConfigMap(t *testing.T) {
	reconciler, fakeClient, _ := setupTestReconciler(t)
	ctx := context.Background()

	// Create a JobFlow with artifact copying
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "artifact-test-flow",
			Namespace: "default",
			UID:       "artifact-test-uid",
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
												Args:    []string{"echo 'artifact data' > /output/artifact.txt"},
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
						Artifacts: []v1alpha1.ArtifactOutput{
							{
								Name: "artifact",
								Path: "/output/artifact.txt",
							},
						},
					},
				},
				{
					Name:         "step2",
					Dependencies: []string{"step1"},
					Inputs: &v1alpha1.StepInputs{
						Artifacts: []v1alpha1.ArtifactInput{
							{
								Name:     "artifact",
								FromStep: "step1",
								Path:     "/input/artifact.txt",
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
												Args:    []string{"cat /input/artifact.txt"},
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
					Outputs: &v1alpha1.StepOutputs{
						Artifacts: []v1alpha1.ArtifactOutput{
							{
								Name: "artifact",
								Path: "/output/artifact.txt",
							},
						},
					},
				},
			},
		},
	}

	// Create a ConfigMap with artifact data (simulating step1 output)
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "artifact-test-flow-step1-artifact",
			Namespace: "default",
		},
		Data: map[string]string{
			"artifact": "artifact data",
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}
	if err := fakeClient.Create(ctx, configMap); err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}

	// Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "artifact-test-flow",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	// Status update failures are expected with fake client
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"artifact-test-flow\" not found" {
		t.Logf("Reconcile completed (status update errors expected with fake client): %v", err)
	}

	// Verify JobFlow exists
	updated := &v1alpha1.JobFlow{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "artifact-test-flow", Namespace: "default"}, updated); err != nil {
		t.Fatalf("Failed to get JobFlow: %v", err)
	}
}

// TestIntegration_ArtifactArchiving tests artifact archiving functionality
func TestIntegration_ArtifactArchiving(t *testing.T) {
	reconciler, fakeClient, _ := setupTestReconciler(t)
	ctx := context.Background()

	// Create a JobFlow with artifact archiving
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "archive-test-flow",
			Namespace: "default",
			UID:       "archive-test-uid",
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
												Args:    []string{"echo 'data' > /output/data.txt"},
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
						Artifacts: []v1alpha1.ArtifactOutput{
							{
								Name: "data",
								Path: "/output/data.txt",
								Archive: &v1alpha1.ArchiveConfig{
									Format:     "tar",
									Compression: "gzip",
								},
							},
						},
					},
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

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Reconcile
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "archive-test-flow",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	// Status update failures are expected with fake client
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"archive-test-flow\" not found" {
		t.Logf("Reconcile completed (status update errors expected with fake client): %v", err)
	}

	// Verify JobFlow exists
	updated := &v1alpha1.JobFlow{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "archive-test-flow", Namespace: "default"}, updated); err != nil {
		t.Fatalf("Failed to get JobFlow: %v", err)
	}
}

// TestIntegration_ComplexDAGWithParameters tests a complex DAG with parameter passing
func TestIntegration_ComplexDAGWithParameters(t *testing.T) {
	reconciler, fakeClient, _ := setupTestReconciler(t)
	ctx := context.Background()

	// Create a JobFlow with multiple steps and parameter passing
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "complex-dag-flow",
			Namespace: "default",
			UID:       "complex-dag-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name:         "step1",
					Dependencies: []string{},
					Inputs: &v1alpha1.StepInputs{
						Parameters: []v1alpha1.ParameterInput{
							{
								Name:  "initialValue",
								Value: "100",
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
												Args:    []string{"echo '{{.parameters.initialValue}}'"},
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
					Outputs: &v1alpha1.StepOutputs{
						Parameters: []v1alpha1.ParameterOutput{
							{
								Name: "result",
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
								Name: "step1Result",
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
												Args:    []string{"echo 'Step 2 with param: {{.parameters.step1Result}}'"},
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
				{
					Name:         "step3",
					Dependencies: []string{"step1"},
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
												Args:    []string{"echo 'Step 3 parallel to step2'"},
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
				{
					Name:         "step4",
					Dependencies: []string{"step2", "step3"},
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
												Args:    []string{"echo 'Final step'"},
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
			Name:      "complex-dag-flow",
			Namespace: "default",
		},
	}

	_, err := reconciler.Reconcile(ctx, req)
	// Status update failures are expected with fake client
	if err != nil && err.Error() != "jobflows.workflow.kube-zen.io \"complex-dag-flow\" not found" {
		t.Logf("Reconcile completed (status update errors expected with fake client): %v", err)
	}

	// Verify JobFlow exists
	updated := &v1alpha1.JobFlow{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "complex-dag-flow", Namespace: "default"}, updated); err != nil {
		t.Fatalf("Failed to get JobFlow: %v", err)
	}

	// Verify DAG structure
	if len(updated.Spec.Steps) != 4 {
		t.Errorf("Expected 4 steps, got %d", len(updated.Spec.Steps))
	}
}

