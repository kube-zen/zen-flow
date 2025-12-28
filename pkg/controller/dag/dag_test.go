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

package dag

import (
	"testing"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestBuildDAG_EmptySteps(t *testing.T) {
	steps := []v1alpha1.Step{}
	graph := BuildDAG(steps)

	if graph == nil {
		t.Fatal("BuildDAG returned nil")
	}

	if len(graph.steps) != 0 {
		t.Errorf("Expected 0 steps, got %d", len(graph.steps))
	}
}

func TestBuildDAG_SingleStep(t *testing.T) {
	steps := []v1alpha1.Step{
		{Name: "step1", Dependencies: []string{}},
	}
	graph := BuildDAG(steps)

	if graph == nil {
		t.Fatal("BuildDAG returned nil")
	}

	if len(graph.steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(graph.steps))
	}

	node := graph.GetStep("step1")
	if node == nil {
		t.Fatal("GetStep returned nil")
	}
	if node.Name != "step1" {
		t.Errorf("Expected step name 'step1', got '%s'", node.Name)
	}
}

func TestBuildDAG_LinearDependencies(t *testing.T) {
	steps := []v1alpha1.Step{
		{Name: "step1", Dependencies: []string{}},
		{Name: "step2", Dependencies: []string{"step1"}},
		{Name: "step3", Dependencies: []string{"step2"}},
	}
	graph := BuildDAG(steps)

	sorted := graph.TopologicalSort()
	expected := []string{"step1", "step2", "step3"}

	if len(sorted) != len(expected) {
		t.Fatalf("Expected %d steps in sort, got %d", len(expected), len(sorted))
	}

	for i, step := range expected {
		if sorted[i] != step {
			t.Errorf("Position %d: expected '%s', got '%s'", i, step, sorted[i])
		}
	}
}

func TestBuildDAG_ParallelSteps(t *testing.T) {
	steps := []v1alpha1.Step{
		{Name: "step1", Dependencies: []string{}},
		{Name: "step2", Dependencies: []string{}},
		{Name: "step3", Dependencies: []string{"step1", "step2"}},
	}
	graph := BuildDAG(steps)

	sorted := graph.TopologicalSort()
	if sorted == nil {
		t.Fatal("TopologicalSort returned nil (cycle detected)")
	}

	// step1 and step2 should come before step3
	step1Index := -1
	step2Index := -1
	step3Index := -1

	for i, step := range sorted {
		switch step {
		case "step1":
			step1Index = i
		case "step2":
			step2Index = i
		case "step3":
			step3Index = i
		}
	}

	if step1Index == -1 || step2Index == -1 || step3Index == -1 {
		t.Fatalf("Not all steps found in sort: step1=%d, step2=%d, step3=%d", step1Index, step2Index, step3Index)
	}

	if step3Index <= step1Index || step3Index <= step2Index {
		t.Errorf("step3 should come after step1 and step2. step1=%d, step2=%d, step3=%d", step1Index, step2Index, step3Index)
	}
}

func TestBuildDAG_ComplexDAG(t *testing.T) {
	steps := []v1alpha1.Step{
		{Name: "extract-a", Dependencies: []string{}},
		{Name: "extract-b", Dependencies: []string{}},
		{Name: "transform-a", Dependencies: []string{"extract-a"}},
		{Name: "transform-b", Dependencies: []string{"extract-b"}},
		{Name: "merge", Dependencies: []string{"transform-a", "transform-b"}},
		{Name: "load", Dependencies: []string{"merge"}},
	}
	graph := BuildDAG(steps)

	sorted := graph.TopologicalSort()
	if sorted == nil {
		t.Fatal("TopologicalSort returned nil (cycle detected)")
	}

	// Verify dependencies are respected
	indices := make(map[string]int)
	for i, step := range sorted {
		indices[step] = i
	}

	// extract-a and extract-b should come first
	if indices["extract-a"] >= indices["transform-a"] {
		t.Error("extract-a should come before transform-a")
	}
	if indices["extract-b"] >= indices["transform-b"] {
		t.Error("extract-b should come before transform-b")
	}
	if indices["transform-a"] >= indices["merge"] {
		t.Error("transform-a should come before merge")
	}
	if indices["transform-b"] >= indices["merge"] {
		t.Error("transform-b should come before merge")
	}
	if indices["merge"] >= indices["load"] {
		t.Error("merge should come before load")
	}
}

func TestBuildDAG_CycleDetection(t *testing.T) {
	steps := []v1alpha1.Step{
		{Name: "step1", Dependencies: []string{"step2"}},
		{Name: "step2", Dependencies: []string{"step1"}},
	}
	graph := BuildDAG(steps)

	sorted := graph.TopologicalSort()
	if sorted != nil {
		t.Error("Expected cycle detection to return nil, but got sorted list")
	}
}

func TestBuildDAG_GetStep(t *testing.T) {
	steps := []v1alpha1.Step{
		{Name: "step1", Dependencies: []string{}},
		{Name: "step2", Dependencies: []string{"step1"}},
	}
	graph := BuildDAG(steps)

	node := graph.GetStep("step1")
	if node == nil {
		t.Fatal("GetStep returned nil for existing step")
	}
	if node.Name != "step1" {
		t.Errorf("Expected step name 'step1', got '%s'", node.Name)
	}

	node = graph.GetStep("nonexistent")
	if node != nil {
		t.Error("GetStep returned non-nil for nonexistent step")
	}
}

func TestTopologicalSort_EmptyGraph(t *testing.T) {
	graph := &Graph{steps: make(map[string]*StepNode)}
	sorted := graph.TopologicalSort()

	if sorted == nil {
		t.Error("TopologicalSort returned nil for empty graph")
	}
	if len(sorted) != 0 {
		t.Errorf("Expected empty sort result, got %d items", len(sorted))
	}
}

