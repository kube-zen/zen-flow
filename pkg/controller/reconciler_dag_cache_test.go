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

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestJobFlowReconciler_getOrBuildDAG(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
				{Name: "step2", Dependencies: []string{"step1"}},
			},
		},
	}

	// First call - should build DAG
	dagGraph1, sortedSteps1, err := reconciler.getOrBuildDAG(jobFlow)
	if err != nil {
		t.Fatalf("getOrBuildDAG() error = %v", err)
	}
	if dagGraph1 == nil {
		t.Fatal("getOrBuildDAG() returned nil DAG")
	}
	if len(sortedSteps1) != 2 {
		t.Errorf("getOrBuildDAG() sortedSteps length = %d, want 2", len(sortedSteps1))
	}

	// Second call with same spec - should use cache
	dagGraph2, _, err := reconciler.getOrBuildDAG(jobFlow)
	if err != nil {
		t.Fatalf("getOrBuildDAG() error = %v", err)
	}
	if dagGraph2 != dagGraph1 {
		t.Error("getOrBuildDAG() should return cached DAG on second call")
	}

	// Call with different spec - should rebuild
	jobFlow2 := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
				{Name: "step3"}, // Different step
			},
		},
	}
	dagGraph3, sortedSteps3, err := reconciler.getOrBuildDAG(jobFlow2)
	if err != nil {
		t.Fatalf("getOrBuildDAG() error = %v", err)
	}
	if dagGraph3 == dagGraph1 {
		t.Error("getOrBuildDAG() should rebuild DAG when spec changes")
	}
	if len(sortedSteps3) != 2 {
		t.Errorf("getOrBuildDAG() sortedSteps length = %d, want 2", len(sortedSteps3))
	}
}

func TestJobFlowReconciler_computeStepsHash(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	steps1 := []v1alpha1.Step{
		{Name: "step1"},
		{Name: "step2"},
	}

	steps2 := []v1alpha1.Step{
		{Name: "step1"},
		{Name: "step2"},
	}

	steps3 := []v1alpha1.Step{
		{Name: "step1"},
		{Name: "step3"}, // Different
	}

	hash1 := reconciler.computeStepsHash(steps1)
	hash2 := reconciler.computeStepsHash(steps2)
	hash3 := reconciler.computeStepsHash(steps3)

	if hash1 == "" {
		t.Error("computeStepsHash() returned empty hash")
	}

	if hash1 != hash2 {
		t.Error("computeStepsHash() should return same hash for identical steps")
	}

	if hash1 == hash3 {
		t.Error("computeStepsHash() should return different hash for different steps")
	}
}

