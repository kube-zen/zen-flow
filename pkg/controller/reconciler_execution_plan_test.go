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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/dag"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

// TestJobFlowReconciler_createExecutionPlan is tested in reconciler_test.go

func TestJobFlowReconciler_createExecutionPlan_Additional(t *testing.T) {
	tests := []struct {
		name        string
		jobFlow     *v1alpha1.JobFlow
		dagGraph    *dag.Graph
		sortedSteps []string
		wantReady   int
	}{
		{
			name: "all steps pending - first step ready",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
						{Name: "step2", DependsOn: []string{"step1"}},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhasePending},
						{Name: "step2", Phase: v1alpha1.StepPhasePending},
					},
				},
			},
			dagGraph: func() *dag.Graph {
				g := dag.BuildDAG([]v1alpha1.Step{
					{Name: "step1"},
					{Name: "step2", DependsOn: []string{"step1"}},
				})
				return g
			}(),
			sortedSteps: []string{"step1", "step2"},
			wantReady:   1, // Only step1 is ready
		},
		{
			name: "first step completed - second step ready",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
						{Name: "step2", DependsOn: []string{"step1"}},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
						{Name: "step2", Phase: v1alpha1.StepPhasePending},
					},
				},
			},
			dagGraph: func() *dag.Graph {
				g := dag.BuildDAG([]v1alpha1.Step{
					{Name: "step1"},
					{Name: "step2", DependsOn: []string{"step1"}},
				})
				return g
			}(),
			sortedSteps: []string{"step1", "step2"},
			wantReady:   1, // step2 is ready
		},
		{
			name: "parallel steps - both ready",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
						{Name: "step2"},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhasePending},
						{Name: "step2", Phase: v1alpha1.StepPhasePending},
					},
				},
			},
			dagGraph: func() *dag.Graph {
				g := dag.BuildDAG([]v1alpha1.Step{
					{Name: "step1"},
					{Name: "step2"},
				})
				return g
			}(),
			sortedSteps: []string{"step1", "step2"},
			wantReady:   2, // Both steps are ready
		},
		{
			name: "failed step with ContinueOnFailure - dependency satisfied",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1", ContinueOnFailure: true},
						{Name: "step2", DependsOn: []string{"step1"}},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseFailed},
						{Name: "step2", Phase: v1alpha1.StepPhasePending},
					},
				},
			},
			dagGraph: func() *dag.Graph {
				g := dag.BuildDAG([]v1alpha1.Step{
					{Name: "step1", ContinueOnFailure: true},
					{Name: "step2", DependsOn: []string{"step1"}},
				})
				return g
			}(),
			sortedSteps: []string{"step1", "step2"},
			wantReady:   1, // step2 is ready because step1 failed but ContinueOnFailure
		},
		{
			name: "step with When condition - always",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
						{Name: "step2", When: "always"},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhasePending},
						{Name: "step2", Phase: v1alpha1.StepPhasePending},
					},
				},
			},
			dagGraph: func() *dag.Graph {
				g := dag.BuildDAG([]v1alpha1.Step{
					{Name: "step1"},
					{Name: "step2", When: "always"},
				})
				return g
			}(),
			sortedSteps: []string{"step1", "step2"},
			wantReady:   1, // Only step1 is ready (step2 has When condition but no deps)
		},
		{
			name: "step already running - not in ready list",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseRunning},
					},
				},
			},
			dagGraph: func() *dag.Graph {
				g := dag.BuildDAG([]v1alpha1.Step{
					{Name: "step1"},
				})
				return g
			}(),
			sortedSteps: []string{"step1"},
			wantReady:   0, // Step is already running
		},
		{
			name: "step pending approval - not in ready list",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhasePendingApproval},
					},
				},
			},
			dagGraph: func() *dag.Graph {
				g := dag.BuildDAG([]v1alpha1.Step{
					{Name: "step1"},
				})
				return g
			}(),
			sortedSteps: []string{"step1"},
			wantReady:   0, // Step is pending approval
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{
				MetricsRecorder: metrics.NewRecorder(),
				EventRecorder:   NewEventRecorder(nil),
			}

			plan := r.createExecutionPlan(tt.dagGraph, tt.jobFlow, tt.sortedSteps)

			if plan == nil {
				t.Fatal("createExecutionPlan() returned nil")
			}

			if len(plan.ReadySteps) != tt.wantReady {
				t.Errorf("createExecutionPlan() ReadySteps = %v, want %d ready steps", plan.ReadySteps, tt.wantReady)
			}
		})
	}
}

func TestJobFlowReconciler_updateJobFlowStatus(t *testing.T) {
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		plan    *ExecutionPlan
		wantErr bool
	}{
		{
			name: "update progress - all succeeded",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Status: v1alpha1.JobFlowStatus{
					Progress: &v1alpha1.ProgressStatus{
						TotalSteps: 2,
					},
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
						{Name: "step2", Phase: v1alpha1.StepPhaseSucceeded},
					},
				},
			},
			plan:    &ExecutionPlan{ReadySteps: []string{}},
			wantErr: false,
		},
		{
			name: "update progress - mixed results",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Status: v1alpha1.JobFlowStatus{
					Progress: &v1alpha1.ProgressStatus{
						TotalSteps: 3,
					},
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
						{Name: "step2", Phase: v1alpha1.StepPhaseFailed},
						{Name: "step3", Phase: v1alpha1.StepPhaseRunning},
					},
				},
			},
			plan:    &ExecutionPlan{ReadySteps: []string{}},
			wantErr: false,
		},
		{
			name: "phase transition - pending to running",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Status: v1alpha1.JobFlowStatus{
					Phase: v1alpha1.JobFlowPhasePending,
					Progress: &v1alpha1.ProgressStatus{
						TotalSteps: 1,
					},
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseRunning},
					},
				},
			},
			plan:    &ExecutionPlan{ReadySteps: []string{}},
			wantErr: false,
		},
		{
			name: "phase transition - all succeeded",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Status: v1alpha1.JobFlowStatus{
					Phase: v1alpha1.JobFlowPhaseRunning,
					Progress: &v1alpha1.ProgressStatus{
						TotalSteps: 2,
					},
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
						{Name: "step2", Phase: v1alpha1.StepPhaseSucceeded},
					},
				},
			},
			plan:    &ExecutionPlan{ReadySteps: []string{}},
			wantErr: false,
		},
		{
			name: "phase transition - some failed",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Status: v1alpha1.JobFlowStatus{
					Phase: v1alpha1.JobFlowPhaseRunning,
					Progress: &v1alpha1.ProgressStatus{
						TotalSteps: 2,
					},
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
						{Name: "step2", Phase: v1alpha1.StepPhaseFailed},
					},
				},
			},
			plan:    &ExecutionPlan{ReadySteps: []string{}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.jobFlow).Build()
			r := &JobFlowReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				MetricsRecorder: metrics.NewRecorder(),
				EventRecorder:   NewEventRecorder(nil),
			}

			ctx := context.Background()
			err := r.updateJobFlowStatus(ctx, tt.jobFlow, tt.plan)

			if (err != nil) != tt.wantErr {
				t.Errorf("updateJobFlowStatus() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify progress was updated
			if tt.jobFlow.Status.Progress != nil {
				completed := int32(0)
				successful := int32(0)
				failed := int32(0)

				for _, stepStatus := range tt.jobFlow.Status.Steps {
					switch stepStatus.Phase {
					case v1alpha1.StepPhaseSucceeded:
						completed++
						successful++
					case v1alpha1.StepPhaseFailed:
						completed++
						failed++
					}
				}

				if tt.jobFlow.Status.Progress.CompletedSteps != completed {
					t.Errorf("Progress.CompletedSteps = %d, want %d", tt.jobFlow.Status.Progress.CompletedSteps, completed)
				}
				if tt.jobFlow.Status.Progress.SuccessfulSteps != successful {
					t.Errorf("Progress.SuccessfulSteps = %d, want %d", tt.jobFlow.Status.Progress.SuccessfulSteps, successful)
				}
				if tt.jobFlow.Status.Progress.FailedSteps != failed {
					t.Errorf("Progress.FailedSteps = %d, want %d", tt.jobFlow.Status.Progress.FailedSteps, failed)
				}
			}
		})
	}
}
