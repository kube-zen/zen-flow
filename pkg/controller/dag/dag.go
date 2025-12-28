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
	"fmt"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

// Graph represents a directed acyclic graph of steps.
type Graph struct {
	steps map[string]*StepNode
}

// StepNode represents a node in the DAG.
type StepNode struct {
	Name         string
	Dependencies []string
	Step         *v1alpha1.Step
}

// BuildDAG builds a DAG from a list of steps.
func BuildDAG(steps []v1alpha1.Step) *Graph {
	graph := &Graph{
		steps: make(map[string]*StepNode),
	}

	for i := range steps {
		step := &steps[i]
		node := &StepNode{
			Name:         step.Name,
			Dependencies: step.Dependencies,
			Step:         step,
		}
		graph.steps[step.Name] = node
	}

	return graph
}

// GetStep gets a step by name.
func (g *Graph) GetStep(name string) *StepNode {
	return g.steps[name]
}

// TopologicalSort performs a topological sort of the DAG.
// Returns the sorted step names and an error if a cycle is detected.
func (g *Graph) TopologicalSort() ([]string, error) {
	visited := make(map[string]bool)
	tempMark := make(map[string]bool)
	result := make([]string, 0)

	var visit func(string) error
	visit = func(name string) error {
		if tempMark[name] {
			return fmt.Errorf("cycle detected in DAG involving step: %s", name)
		}
		if visited[name] {
			return nil
		}

		tempMark[name] = true
		node := g.steps[name]
		if node != nil {
			for _, dep := range node.Dependencies {
				if err := visit(dep); err != nil {
					return err
				}
			}
		}
		tempMark[name] = false
		visited[name] = true
		result = append(result, name)
		return nil
	}

	for name := range g.steps {
		if !visited[name] {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}
