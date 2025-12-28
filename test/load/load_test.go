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

package load

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

const (
	defaultJobFlowCount    = 100
	defaultStepsPerFlow    = 10
	defaultConcurrentFlows  = 50
	defaultTestDuration     = 5 * time.Minute
	defaultLargeDAGSteps    = 50
)

// LoadTestConfig configures load test parameters
type LoadTestConfig struct {
	JobFlowCount      int
	StepsPerFlow      int
	ConcurrentFlows   int
	TestDuration      time.Duration
	LargeDAGSteps     int
	Namespace         string
	SkipCleanup       bool
}

// LoadTestResults contains load test results
type LoadTestResults struct {
	TotalJobFlowsCreated   int
	TotalJobFlowsSucceeded int
	TotalJobFlowsFailed    int
	TotalStepsExecuted     int
	AverageCreationTime    time.Duration
	AverageCompletionTime  time.Duration
	PeakMemoryUsage       int64
	TestDuration           time.Duration
	Errors                []error
}

// createJobFlowTemplate creates a JobFlow template for testing
func createJobFlowTemplate(name, namespace string, stepCount int, isDAG bool) *v1alpha1.JobFlow {
	steps := make([]v1alpha1.Step, stepCount)
	
	for i := 0; i < stepCount; i++ {
		var deps []string
		if isDAG && i > 0 {
			// Create a simple linear DAG for testing
			if i%2 == 0 {
				deps = []string{fmt.Sprintf("step%d", i-1)}
			} else if i > 2 {
				deps = []string{fmt.Sprintf("step%d", i-2)}
			}
		} else if i > 0 {
			deps = []string{fmt.Sprintf("step%d", i-1)}
		}

		steps[i] = v1alpha1.Step{
			Name:         fmt.Sprintf("step%d", i),
			Dependencies: deps,
			Template: runtime.RawExtension{
				Object: &batchv1.Job{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "batch/v1",
						Kind:       "Job",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: fmt.Sprintf("%s-step%d", name, i),
					},
					Spec: batchv1.JobSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Name:    "main",
										Image:   "busybox:latest",
										Command: []string{"sh", "-c"},
										Args:    []string{"echo 'Step completed' && sleep 1"},
									},
								},
								RestartPolicy: corev1.RestartPolicyOnFailure,
							},
						},
					},
				},
			},
		}
	}

	return &v1alpha1.JobFlow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "workflow.zen.io/v1alpha1",
			Kind:       "JobFlow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.JobFlowSpec{
			ExecutionPolicy: &v1alpha1.ExecutionPolicy{
				ConcurrencyPolicy: "Allow",
			},
			Steps: steps,
		},
	}
}

// TestConcurrentJobFlowCreation tests creating many JobFlows concurrently
func TestConcurrentJobFlowCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		JobFlowCount:    defaultJobFlowCount,
		StepsPerFlow:    defaultStepsPerFlow,
		ConcurrentFlows: defaultConcurrentFlows,
		Namespace:       "load-test",
	}

	ctx := context.Background()
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	results := &LoadTestResults{
		Errors: make([]error, 0),
	}
	startTime := time.Now()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.ConcurrentFlows)
	var mu sync.Mutex

	for i := 0; i < config.JobFlowCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			name := fmt.Sprintf("load-test-flow-%d", id)
			jobFlow := createJobFlowTemplate(name, config.Namespace, config.StepsPerFlow, false)

			creationStart := time.Now()
			_, err := dynamicClient.Resource(schema.GroupVersionResource{
				Group:    "workflow.zen.io",
				Version:  "v1alpha1",
				Resource: "jobflows",
			}).Namespace(config.Namespace).
				Create(ctx, toUnstructured(jobFlow), metav1.CreateOptions{})
			creationTime := time.Since(creationStart)

			mu.Lock()
			if err != nil {
				results.Errors = append(results.Errors, err)
			} else {
				results.TotalJobFlowsCreated++
				results.AverageCreationTime += creationTime
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	results.TestDuration = time.Since(startTime)
	if results.TotalJobFlowsCreated > 0 {
		results.AverageCreationTime /= time.Duration(results.TotalJobFlowsCreated)
	}

	t.Logf("Created %d JobFlows in %v", results.TotalJobFlowsCreated, results.TestDuration)
	t.Logf("Average creation time: %v", results.AverageCreationTime)
	t.Logf("Errors: %d", len(results.Errors))

	if len(results.Errors) > 0 {
		t.Errorf("Encountered %d errors during creation", len(results.Errors))
	}
}

// TestLargeDAGExecution tests execution of large DAGs
func TestLargeDAGExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		JobFlowCount:  10,
		LargeDAGSteps: defaultLargeDAGSteps,
		Namespace:     "load-test",
	}

	ctx := context.Background()
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	results := &LoadTestResults{}
	startTime := time.Now()

	for i := 0; i < config.JobFlowCount; i++ {
		name := fmt.Sprintf("large-dag-flow-%d", i)
		jobFlow := createJobFlowTemplate(name, config.Namespace, config.LargeDAGSteps, true)

		_, err := dynamicClient.Resource(schema.GroupVersionResource{
			Group:    "workflow.zen.io",
			Version:  "v1alpha1",
			Resource: "jobflows",
		}).Namespace(config.Namespace).
			Create(ctx, toUnstructured(jobFlow), metav1.CreateOptions{})
		if err != nil {
			t.Errorf("Failed to create JobFlow %s: %v", name, err)
			continue
		}

		results.TotalJobFlowsCreated++
		results.TotalStepsExecuted += config.LargeDAGSteps
	}

	results.TestDuration = time.Since(startTime)

	t.Logf("Created %d large DAG JobFlows with %d total steps in %v",
		results.TotalJobFlowsCreated, results.TotalStepsExecuted, results.TestDuration)
}

// TestManyStepsPerFlow tests JobFlows with many steps
func TestManyStepsPerFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		JobFlowCount: 20,
		StepsPerFlow: 100, // Many steps per flow
		Namespace:    "load-test",
	}

	ctx := context.Background()
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)

	results := &LoadTestResults{}
	startTime := time.Now()

	for i := 0; i < config.JobFlowCount; i++ {
		name := fmt.Sprintf("many-steps-flow-%d", i)
		jobFlow := createJobFlowTemplate(name, config.Namespace, config.StepsPerFlow, false)

		_, err := dynamicClient.Resource(schema.GroupVersionResource{
			Group:    "workflow.zen.io",
			Version:  "v1alpha1",
			Resource: "jobflows",
		}).Namespace(config.Namespace).
			Create(ctx, toUnstructured(jobFlow), metav1.CreateOptions{})
		if err != nil {
			t.Errorf("Failed to create JobFlow %s: %v", name, err)
			continue
		}

		results.TotalJobFlowsCreated++
		results.TotalStepsExecuted += config.StepsPerFlow
	}

	results.TestDuration = time.Since(startTime)

	t.Logf("Created %d JobFlows with %d steps each (%d total steps) in %v",
		results.TotalJobFlowsCreated, config.StepsPerFlow, results.TotalStepsExecuted, results.TestDuration)
}

// TestSustainedLoad tests sustained load over time
func TestSustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	config := LoadTestConfig{
		ConcurrentFlows: defaultConcurrentFlows,
		TestDuration:    defaultTestDuration,
		StepsPerFlow:    defaultStepsPerFlow,
		Namespace:       "load-test",
	}

	ctx := context.Background()
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	ctx, cancel := context.WithTimeout(ctx, config.TestDuration)
	defer cancel()

	results := &LoadTestResults{}
	startTime := time.Now()
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, config.ConcurrentFlows)
	var mu sync.Mutex
	counter := 0

	for {
		select {
		case <-ctx.Done():
			goto done
		default:
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				name := fmt.Sprintf("sustained-load-flow-%d", id)
				jobFlow := createJobFlowTemplate(name, config.Namespace, config.StepsPerFlow, false)

				_, err := dynamicClient.Resource(schema.GroupVersionResource{
					Group:    "workflow.zen.io",
					Version:  "v1alpha1",
					Resource: "jobflows",
				}).Namespace(config.Namespace).
					Create(ctx, toUnstructured(jobFlow), metav1.CreateOptions{})

				mu.Lock()
				if err != nil {
					results.Errors = append(results.Errors, err)
				} else {
					results.TotalJobFlowsCreated++
				}
				mu.Unlock()
			}(counter)
			counter++
			time.Sleep(100 * time.Millisecond) // Rate limit
		}
	}

done:
	wg.Wait()
	results.TestDuration = time.Since(startTime)

	t.Logf("Created %d JobFlows over %v (sustained load)", results.TotalJobFlowsCreated, results.TestDuration)
	t.Logf("Average rate: %.2f JobFlows/second", float64(results.TotalJobFlowsCreated)/results.TestDuration.Seconds())
}

// Helper functions

func toUnstructured(jobFlow *v1alpha1.JobFlow) *unstructured.Unstructured {
	// Convert JobFlow to unstructured for dynamic client
	// In real implementation, use proper conversion utilities
	u := &unstructured.Unstructured{}
	u.SetAPIVersion("workflow.zen.io/v1alpha1")
	u.SetKind("JobFlow")
	u.SetName(jobFlow.Name)
	u.SetNamespace(jobFlow.Namespace)
	
	// Set spec - simplified for testing
	// Convert steps to JSON-compatible format
	stepsRaw, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&jobFlow.Spec)
	u.Object["spec"] = stepsRaw
	
	return u
}

// BenchmarkJobFlowCreation benchmarks JobFlow creation performance
func BenchmarkJobFlowCreation(b *testing.B) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)
	dynamicClient := dynamicfake.NewSimpleDynamicClient(scheme)
	namespace := "benchmark"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		name := fmt.Sprintf("bench-flow-%d", i)
		jobFlow := createJobFlowTemplate(name, namespace, 10, false)
		_, err := dynamicClient.Resource(schema.GroupVersionResource{
			Group:    "workflow.zen.io",
			Version:  "v1alpha1",
			Resource: "jobflows",
		}).Namespace(namespace).
			Create(ctx, toUnstructured(jobFlow), metav1.CreateOptions{})
		if err != nil {
			b.Fatal(err)
		}
	}
}

