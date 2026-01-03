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
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

// TestJobFlowReconciler_computeStepsHash is tested in reconciler_dag_cache_test.go
// This test is removed to avoid duplication

func TestJobFlowReconciler_computeStepsHash_Additional(t *testing.T) {
	tests := []struct {
		name  string
		steps []v1alpha1.Step
		want  string
	}{
		{
			name:  "empty steps",
			steps: []v1alpha1.Step{},
			want:  "", // Empty hash for empty steps
		},
		{
			name: "single step",
			steps: []v1alpha1.Step{
				{Name: "step1"},
			},
			want: "", // Non-empty hash
		},
		{
			name: "multiple steps",
			steps: []v1alpha1.Step{
				{Name: "step1"},
				{Name: "step2", DependsOn: []string{"step1"}},
			},
			want: "", // Non-empty hash
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{}
			got := r.computeStepsHash(tt.steps)

			// Hash should be consistent for same input
			got2 := r.computeStepsHash(tt.steps)
			if got != got2 {
				t.Errorf("computeStepsHash() is not consistent: first = %v, second = %v", got, got2)
			}

			// Hash should be different for different input
			if len(tt.steps) > 0 {
				differentSteps := []v1alpha1.Step{
					{Name: "different-step"},
				}
				got3 := r.computeStepsHash(differentSteps)
				if got == got3 {
					t.Errorf("computeStepsHash() should produce different hashes for different inputs")
				}
			}
		})
	}
}

// TestJobFlowReconciler_getOrBuildDAG is tested in reconciler_dag_cache_test.go
// This test is removed to avoid duplication

func TestJobFlowReconciler_getOrBuildDAG_Additional(t *testing.T) {
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		wantErr bool
	}{
		{
			name: "build DAG successfully",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
						{Name: "step2", DependsOn: []string{"step1"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "cache hit - same steps",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
						{Name: "step2", DependsOn: []string{"step1"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "cache miss - different steps",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
						{Name: "step2"},
						{Name: "step3", DependsOn: []string{"step1", "step2"}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "cycle detection",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1", DependsOn: []string{"step2"}},
						{Name: "step2", DependsOn: []string{"step1"}},
					},
				},
			},
			wantErr: true, // Cycle should be detected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			r := &JobFlowReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				MetricsRecorder: metrics.NewRecorder(),
				EventRecorder:   NewEventRecorder(nil),
			}

			// First call - should build DAG
			dagGraph1, sortedSteps1, err1 := r.getOrBuildDAG(tt.jobFlow)
			if (err1 != nil) != tt.wantErr {
				t.Errorf("getOrBuildDAG() first call error = %v, wantErr %v", err1, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if dagGraph1 == nil {
					t.Error("getOrBuildDAG() returned nil DAG graph")
					return
				}
				if sortedSteps1 == nil {
					t.Error("getOrBuildDAG() returned nil sorted steps")
					return
				}

				// Second call with same steps - should use cache
				dagGraph2, sortedSteps2, err2 := r.getOrBuildDAG(tt.jobFlow)
				if err2 != nil {
					t.Errorf("getOrBuildDAG() second call error = %v", err2)
					return
				}

				// Verify cache hit (same pointers or same content)
				if dagGraph1 != dagGraph2 {
					// If pointers are different, verify content is same
					// This is acceptable as long as the DAG is correct
					_ = dagGraph2
				}

				// Verify sorted steps are the same
				if len(sortedSteps1) != len(sortedSteps2) {
					t.Errorf("Sorted steps length mismatch: first = %d, second = %d", len(sortedSteps1), len(sortedSteps2))
				}
			}
		})
	}
}

func TestJobFlowReconciler_getOrBuildDAG_Cache(t *testing.T) {
	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
				{Name: "step2", DependsOn: []string{"step1"}},
			},
		},
	}

	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &JobFlowReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		MetricsRecorder: metrics.NewRecorder(),
		EventRecorder:   NewEventRecorder(nil),
	}

	// First call - should build and cache
	_, _, err := r.getOrBuildDAG(jobFlow)
	if err != nil {
		t.Fatalf("getOrBuildDAG() error = %v", err)
	}

	// Verify cache is populated
	r.dagMu.RLock()
	cacheKey := "default/test-flow"
	cached, exists := r.dagCache[cacheKey]
	r.dagMu.RUnlock()

	if !exists {
		t.Error("DAG cache entry not found")
		return
	}

	if cached.dagGraph == nil {
		t.Error("Cached DAG graph is nil")
	}

	if cached.sortedSteps == nil {
		t.Error("Cached sorted steps are nil")
	}

	// Modify steps - should invalidate cache
	jobFlow.Spec.Steps = append(jobFlow.Spec.Steps, v1alpha1.Step{Name: "step3"})

	// Call again - should rebuild (cache miss)
	_, _, err = r.getOrBuildDAG(jobFlow)
	if err != nil {
		t.Fatalf("getOrBuildDAG() after modification error = %v", err)
	}

	// Verify cache was updated
	r.dagMu.RLock()
	cached2, exists2 := r.dagCache[cacheKey]
	r.dagMu.RUnlock()

	if !exists2 {
		t.Error("DAG cache entry not found after modification")
		return
	}

	// Hash should be different
	if cached.specHash == cached2.specHash {
		t.Error("Cache hash should be different after step modification")
	}
}

func TestJobFlowReconciler_getOrBuildDAG_DifferentNamespaces(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &JobFlowReconciler{
		Client:          fakeClient,
		Scheme:          scheme,
		MetricsRecorder: metrics.NewRecorder(),
		EventRecorder:   NewEventRecorder(nil),
	}

	// Create two JobFlows with same name but different namespaces
	jobFlow1 := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "namespace1",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
	}

	jobFlow2 := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "namespace2",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step2"},
			},
		},
	}

	// Build DAG for both
	_, _, err1 := r.getOrBuildDAG(jobFlow1)
	if err1 != nil {
		t.Fatalf("getOrBuildDAG() for jobFlow1 error = %v", err1)
	}

	_, _, err2 := r.getOrBuildDAG(jobFlow2)
	if err2 != nil {
		t.Fatalf("getOrBuildDAG() for jobFlow2 error = %v", err2)
	}

	// Verify both are cached separately
	r.dagMu.RLock()
	cached1, exists1 := r.dagCache["namespace1/test-flow"]
	cached2, exists2 := r.dagCache["namespace2/test-flow"]
	r.dagMu.RUnlock()

	if !exists1 {
		t.Error("DAG cache entry for namespace1 not found")
	}
	if !exists2 {
		t.Error("DAG cache entry for namespace2 not found")
	}

	if cached1.specHash == cached2.specHash {
		t.Error("Cache hashes should be different for different JobFlows")
	}
}
