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

//go:build integration && experimental

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

// TestExperimentalFeatures_JSONv2 tests JSON v2 performance improvements
func TestExperimentalFeatures_JSONv2(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping experimental features test in short mode")
	}

	ctx := context.Background()
	_, fakeClient, _ := setupTestReconciler(t)

	// Create a JobFlow with multiple steps
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jsonv2-test",
			Namespace: env.namespace,
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: &v1alpha1.JobTemplate{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									RestartPolicy: corev1.RestartPolicyNever,
									Containers: []corev1.Container{
										{
											Name:    "test",
											Image:   "busybox:latest",
											Command: []string{"sh", "-c", "echo 'test'"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := env.client.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Wait for reconciliation
	time.Sleep(2 * time.Second)

	// Test JSON marshaling performance (simulating status updates)
	iterations := 1000
	start := time.Now()

	for i := 0; i < iterations; i++ {
		// Simulate status marshaling (what happens during reconciliation)
		status := jobFlow.Status
		status.Phase = v1alpha1.JobFlowPhaseRunning
		status.Steps = []v1alpha1.StepStatus{
			{
				Name:  "step1",
				Phase: v1alpha1.StepPhaseRunning,
			},
		}

		_, err := json.Marshal(status)
		if err != nil {
			t.Fatalf("Failed to marshal status: %v", err)
		}
	}

	duration := time.Since(start)
	avgTime := duration / time.Duration(iterations)

	t.Logf("JSON marshaling performance:")
	t.Logf("  Iterations: %d", iterations)
	t.Logf("  Total time: %v", duration)
	t.Logf("  Average time per operation: %v", avgTime)
	t.Logf("  Operations per second: %.2f", float64(iterations)/duration.Seconds())

	// With JSON v2, we expect 2-3x faster operations
	// This test documents baseline performance
}

// TestExperimentalFeatures_GreenTeaGC tests Green Tea GC performance
func TestExperimentalFeatures_GreenTeaGC(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping experimental features test in short mode")
	}

	ctx := context.Background()
	_, fakeClient, _ := setupTestReconciler(t)

	// Create multiple JobFlows to trigger GC
	numJobFlows := 50
	jobFlows := make([]*v1alpha1.JobFlow, numJobFlows)

	start := time.Now()

	for i := 0; i < numJobFlows; i++ {
		jobFlow := &v1alpha1.JobFlow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("gctest-%d", i),
				Namespace: env.namespace,
			},
			Spec: v1alpha1.JobFlowSpec{
				Steps: []v1alpha1.Step{
					{
						Name: "step1",
						Template: &v1alpha1.JobTemplate{
							Spec: batchv1.JobSpec{
								Template: corev1.PodTemplateSpec{
									Spec: corev1.PodSpec{
										RestartPolicy: corev1.RestartPolicyNever,
										Containers: []corev1.Container{
											{
												Name:    "test",
												Image:   "busybox:latest",
												Command: []string{"sh", "-c", "echo 'test'"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		if err := fakeClient.Create(ctx, jobFlow); err != nil {
			t.Fatalf("Failed to create JobFlow %d: %v", i, err)
		}

		jobFlows[i] = jobFlow

		// Simulate frequent status updates (triggers allocations)
		jobFlow.Status.Phase = v1alpha1.JobFlowPhaseRunning
		jobFlow.Status.Steps = []v1alpha1.StepStatus{
			{
				Name:  "step1",
				Phase: v1alpha1.StepPhaseRunning,
			},
		}
	}

	duration := time.Since(start)

	t.Logf("GC performance test:")
	t.Logf("  JobFlows created: %d", numJobFlows)
	t.Logf("  Total time: %v", duration)
	t.Logf("  Average time per JobFlow: %v", duration/time.Duration(numJobFlows))

	// Cleanup
	for _, jf := range jobFlows {
		_ = fakeClient.Delete(ctx, jf)
	}

	// With Green Tea GC, we expect lower GC overhead
	// This test documents baseline performance
}

// TestExperimentalFeatures_Combined tests combined experimental features
func TestExperimentalFeatures_Combined(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping experimental features test in short mode")
	}

	ctx := context.Background()
	_, fakeClient, _ := setupTestReconciler(t)

	// Create JobFlow with parameters (uses JSON processing)
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "combined-test",
			Namespace: env.namespace,
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{
					Name: "step1",
					Template: &v1alpha1.JobTemplate{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									RestartPolicy: corev1.RestartPolicyNever,
									Containers: []corev1.Container{
										{
											Name:    "test",
											Image:   "busybox:latest",
											Command: []string{"sh", "-c", "echo 'test'"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := env.client.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Simulate reconciliation loop with JSON operations and allocations
	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		// JSON marshaling (JSON v2 benefit)
		status := jobFlow.Status
		status.Phase = v1alpha1.JobFlowPhaseRunning
		statusJSON, _ := json.Marshal(status)
		_ = json.Unmarshal(statusJSON, &status)

		// Allocations (Green Tea GC benefit)
		steps := make([]v1alpha1.StepStatus, 10)
		for j := range steps {
			steps[j] = v1alpha1.StepStatus{
				Name:  fmt.Sprintf("step-%d", j),
				Phase: v1alpha1.StepPhaseRunning,
			}
		}
		status.Steps = steps
	}

	duration := time.Since(start)

	t.Logf("Combined experimental features test:")
	t.Logf("  Iterations: %d", iterations)
	t.Logf("  Total time: %v", duration)
	t.Logf("  Average time per iteration: %v", duration/time.Duration(iterations))

	// With both features, we expect 15-25% overall improvement
	// This test documents baseline performance
}

// BenchmarkExperimentalFeatures_JSON benchmarks JSON operations
func BenchmarkExperimentalFeatures_JSON(b *testing.B) {
	status := v1alpha1.JobFlowStatus{
		Phase: v1alpha1.JobFlowPhaseRunning,
		Steps: []v1alpha1.StepStatus{
			{
				Name:  "step1",
				Phase: v1alpha1.StepPhaseRunning,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, _ := json.Marshal(status)
		_ = json.Unmarshal(data, &status)
	}
}

// BenchmarkExperimentalFeatures_Allocations benchmarks allocation patterns
func BenchmarkExperimentalFeatures_Allocations(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		steps := make([]v1alpha1.StepStatus, 100)
		for j := range steps {
			steps[j] = v1alpha1.StepStatus{
				Name:  fmt.Sprintf("step-%d", j),
				Phase: v1alpha1.StepPhaseRunning,
			}
		}
		_ = steps
	}
}

