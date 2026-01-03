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
	"fmt"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

func TestJobFlowReconciler_refreshStepStatusFromJob(t *testing.T) {
	tests := []struct {
		name     string
		jobFlow  *v1alpha1.JobFlow
		stepName string
		job      *batchv1.Job
	}{
		{
			name: "job succeeded",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseRunning},
					},
				},
			},
			stepName: "step1",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{
					Succeeded: 1,
				},
			},
		},
		{
			name: "job failed",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseRunning},
					},
				},
			},
			stepName: "step1",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{
					Failed: 1,
				},
			},
		},
		{
			name: "job still running",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseRunning},
					},
				},
			},
			stepName: "step1",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{
				MetricsRecorder: metrics.NewRecorder(),
				EventRecorder:   NewEventRecorder(nil),
			}

			r.refreshStepStatusFromJob(tt.jobFlow, tt.stepName, tt.job)

			stepStatus := r.getStepStatus(tt.jobFlow.Status, tt.stepName)
			if stepStatus == nil {
				t.Fatal("Step status not found")
			}

			// Verify phase was updated based on job status
			if tt.job.Status.Succeeded > 0 {
				if stepStatus.Phase != v1alpha1.StepPhaseSucceeded {
					t.Errorf("Expected phase Succeeded, got %v", stepStatus.Phase)
				}
			} else if tt.job.Status.Failed > 0 {
				if stepStatus.Phase != v1alpha1.StepPhaseFailed {
					t.Errorf("Expected phase Failed, got %v", stepStatus.Phase)
				}
			} else if tt.job.Status.Active > 0 {
				if stepStatus.Phase != v1alpha1.StepPhaseRunning {
					t.Errorf("Expected phase Running, got %v", stepStatus.Phase)
				}
			}
		})
	}
}

func TestJobFlowReconciler_refreshStepStatusesParallel(t *testing.T) {
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		jobs    []*batchv1.Job
		wantErr bool
	}{
		{
			name: "no steps with JobRef",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhasePending},
					},
				},
			},
			jobs:    []*batchv1.Job{},
			wantErr: false,
		},
		{
			name: "refresh multiple steps in parallel",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{
							Name:  "step1",
							Phase: v1alpha1.StepPhaseRunning,
							JobRef: &corev1.ObjectReference{
								Name:      "job1",
								Namespace: "default",
							},
						},
						{
							Name:  "step2",
							Phase: v1alpha1.StepPhaseRunning,
							JobRef: &corev1.ObjectReference{
								Name:      "job2",
								Namespace: "default",
							},
						},
					},
				},
			},
			jobs: []*batchv1.Job{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "job1", Namespace: "default"},
					Status:     batchv1.JobStatus{Succeeded: 1},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "job2", Namespace: "default"},
					Status:     batchv1.JobStatus{Active: 1},
				},
			},
			wantErr: false,
		},
		{
			name: "job not found - should handle gracefully",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{
							Name:  "step1",
							Phase: v1alpha1.StepPhaseRunning,
							JobRef: &corev1.ObjectReference{
								Name:      "nonexistent-job",
								Namespace: "default",
							},
						},
					},
				},
			},
			jobs:    []*batchv1.Job{},
			wantErr: false, // Should handle missing job gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)
			_ = batchv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			objects := make([]client.Object, len(tt.jobs))
			for i, job := range tt.jobs {
				objects[i] = job
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			r := &JobFlowReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				MetricsRecorder: metrics.NewRecorder(),
				EventRecorder:   NewEventRecorder(nil),
			}

			ctx := context.Background()
			err := r.refreshStepStatusesParallel(ctx, tt.jobFlow)

			if (err != nil) != tt.wantErr {
				t.Errorf("refreshStepStatusesParallel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobFlowReconciler_isRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "conflict error - retryable",
			err:  k8serrors.NewConflict(schema.GroupResource{Resource: "jobflows"}, "test", nil),
			want: true,
		},
		{
			name: "server timeout - retryable",
			err:  k8serrors.NewServerTimeout(schema.GroupResource{Resource: "jobflows"}, "get", 0),
			want: true,
		},
		{
			name: "not found error - not retryable",
			err:  k8serrors.NewNotFound(schema.GroupResource{Resource: "jobflows"}, "test"),
			want: false,
		},
		{
			name: "generic error - not retryable",
			err:  fmt.Errorf("generic error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{}
			got := r.isRetryable(tt.err)
			if got != tt.want {
				t.Errorf("isRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}
