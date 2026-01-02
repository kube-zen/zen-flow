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

// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

const (
	defaultClusterName = "zen-flow-e2e"
	testNamespace      = "default"
	timeout            = 5 * time.Minute
	pollInterval       = 2 * time.Second
)

var (
	clusterName = getClusterName()
)

// getClusterName returns the cluster name from environment variable or default
func getClusterName() string {
	if name := os.Getenv("CLUSTER_NAME"); name != "" {
		return name
	}
	return defaultClusterName
}

// getKubeconfigPath returns the path to the kubeconfig file
func getKubeconfigPath() string {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kube", clusterName+"-config")
}

// getKubernetesClient returns a Kubernetes client for the test cluster
func getKubernetesClient() (*kubernetes.Clientset, error) {
	kubeconfig := getKubeconfigPath()
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}
	return clientset, nil
}

// getControllerClient returns a controller-runtime client for the test cluster
func getControllerClient() (client.Client, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add v1alpha1 scheme: %w", err)
	}
	if err := batchv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add batch scheme: %w", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add core scheme: %w", err)
	}

	cl, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}
	return cl, nil
}

// TestClusterExists verifies that the test cluster exists and is accessible
func TestClusterExists(t *testing.T) {
	kubeconfig := getKubeconfigPath()
	if _, err := os.Stat(kubeconfig); os.IsNotExist(err) {
		t.Skipf("Kubeconfig not found at %s. Run 'make test-e2e-setup' first", kubeconfig)
	}

	clientset, err := getKubernetesClient()
	if err != nil {
		t.Fatalf("Failed to get Kubernetes client: %v", err)
	}

	// Test cluster connectivity
	_, err = clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("Failed to connect to cluster: %v", err)
	}

	t.Log("Cluster is accessible")
}

// TestJobFlowLifecycle tests the full lifecycle of a JobFlow
func TestJobFlowLifecycle(t *testing.T) {
	cl, err := getControllerClient()
	if err != nil {
		t.Fatalf("Failed to get controller client: %v", err)
	}

	ctx := context.Background()

	// Create a simple JobFlow
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-test-flow",
			Namespace: testNamespace,
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
												Args:    []string{"echo 'Step 1 completed' && sleep 1"},
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
					Name:         "step2",
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
												Args:    []string{"echo 'Step 2 completed' && sleep 1"},
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

	// Create JobFlow
	if err := cl.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}
	t.Log("JobFlow created")

	// Cleanup
	defer func() {
		if err := cl.Delete(ctx, jobFlow); err != nil {
			t.Logf("Failed to delete JobFlow: %v", err)
		}
	}()

	// Wait for JobFlow to complete
	err = wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		updated := &v1alpha1.JobFlow{}
		if err := cl.Get(ctx, types.NamespacedName{Name: jobFlow.Name, Namespace: jobFlow.Namespace}, updated); err != nil {
			return false, err
		}

		if updated.Status.Phase == v1alpha1.JobFlowPhaseSucceeded {
			t.Log("JobFlow succeeded")
			return true, nil
		}
		if updated.Status.Phase == v1alpha1.JobFlowPhaseFailed {
			return false, fmt.Errorf("JobFlow failed: %v", updated.Status.Conditions)
		}

		return false, nil
	})

	if err != nil {
		t.Fatalf("JobFlow did not complete successfully: %v", err)
	}

	// Verify both steps completed
	updated := &v1alpha1.JobFlow{}
	if err := cl.Get(ctx, types.NamespacedName{Name: jobFlow.Name, Namespace: jobFlow.Namespace}, updated); err != nil {
		t.Fatalf("Failed to get JobFlow: %v", err)
	}

	if len(updated.Status.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(updated.Status.Steps))
	}

	for _, step := range updated.Status.Steps {
		if step.Phase != v1alpha1.StepPhaseSucceeded {
			t.Errorf("Step %s did not succeed, phase: %s", step.Name, step.Phase)
		}
	}
}

// TestJobFlowDAG tests DAG execution with multiple parallel steps
func TestJobFlowDAG(t *testing.T) {
	cl, err := getControllerClient()
	if err != nil {
		t.Fatalf("Failed to get controller client: %v", err)
	}

	ctx := context.Background()

	// Create a JobFlow with DAG structure: step1 -> step2, step3 -> step4
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-dag-test",
			Namespace: testNamespace,
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
												Args:    []string{"echo 'Step 1' && sleep 1"},
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
					Name:         "step2",
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
												Args:    []string{"echo 'Step 2' && sleep 1"},
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
												Args:    []string{"echo 'Step 3' && sleep 1"},
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
												Args:    []string{"echo 'Step 4' && sleep 1"},
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

	// Create JobFlow
	if err := cl.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}
	t.Log("JobFlow with DAG created")

	// Cleanup
	defer func() {
		if err := cl.Delete(ctx, jobFlow); err != nil {
			t.Logf("Failed to delete JobFlow: %v", err)
		}
	}()

	// Wait for JobFlow to complete
	err = wait.PollImmediate(pollInterval, timeout, func() (bool, error) {
		updated := &v1alpha1.JobFlow{}
		if err := cl.Get(ctx, types.NamespacedName{Name: jobFlow.Name, Namespace: jobFlow.Namespace}, updated); err != nil {
			return false, err
		}

		if updated.Status.Phase == v1alpha1.JobFlowPhaseSucceeded {
			t.Log("JobFlow with DAG succeeded")
			return true, nil
		}
		if updated.Status.Phase == v1alpha1.JobFlowPhaseFailed {
			return false, fmt.Errorf("JobFlow failed: %v", updated.Status.Conditions)
		}

		return false, nil
	})

	if err != nil {
		t.Fatalf("JobFlow did not complete successfully: %v", err)
	}

	// Verify all steps completed
	updated := &v1alpha1.JobFlow{}
	if err := cl.Get(ctx, types.NamespacedName{Name: jobFlow.Name, Namespace: jobFlow.Namespace}, updated); err != nil {
		t.Fatalf("Failed to get JobFlow: %v", err)
	}

	if len(updated.Status.Steps) != 4 {
		t.Errorf("Expected 4 steps, got %d", len(updated.Status.Steps))
	}

	for _, step := range updated.Status.Steps {
		if step.Phase != v1alpha1.StepPhaseSucceeded {
			t.Errorf("Step %s did not succeed, phase: %s", step.Name, step.Phase)
		}
	}
}

