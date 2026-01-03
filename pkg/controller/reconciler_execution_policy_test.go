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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

func TestJobFlowReconciler_checkConcurrencyPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policy   string
		existing []*v1alpha1.JobFlow
		wantErr  bool
	}{
		{
			name:   "Allow policy - no error",
			policy: "Allow",
			existing: []*v1alpha1.JobFlow{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "existing", Namespace: "default", UID: "existing-uid"},
					Status:     v1alpha1.JobFlowStatus{Phase: v1alpha1.JobFlowPhaseRunning},
				},
			},
			wantErr: false,
		},
		{
			name:   "Forbid policy - no concurrent",
			policy: "Forbid",
			existing: []*v1alpha1.JobFlow{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "default", UID: "other-uid"},
					Status:     v1alpha1.JobFlowStatus{Phase: v1alpha1.JobFlowPhaseSucceeded},
				},
			},
			wantErr: false,
		},
		{
			name:   "Forbid policy - concurrent exists",
			policy: "Forbid",
			existing: []*v1alpha1.JobFlow{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default", UID: "existing-uid"},
					Status:     v1alpha1.JobFlowStatus{Phase: v1alpha1.JobFlowPhaseRunning},
				},
			},
			wantErr: true,
		},
		{
			name:   "Replace policy - deletes existing",
			policy: "Replace",
			existing: []*v1alpha1.JobFlow{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default", UID: "existing-uid"},
					Status:     v1alpha1.JobFlowStatus{Phase: v1alpha1.JobFlowPhaseRunning},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)

			objects := make([]client.Object, len(tt.existing))
			for i, jf := range tt.existing {
				objects[i] = jf
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			r := &JobFlowReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				MetricsRecorder: metrics.NewRecorder(),
				EventRecorder:   NewEventRecorder(nil),
			}

			jobFlow := &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default", UID: "test-uid"},
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						ConcurrencyPolicy: tt.policy,
					},
				},
			}

			ctx := context.Background()
			err := r.checkConcurrencyPolicy(ctx, jobFlow)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkConcurrencyPolicy() error = %v, wantErr %v", err, tt.wantErr)
			}

			// For Replace policy, verify existing was deleted
			if tt.policy == "Replace" && !tt.wantErr {
				existing := &v1alpha1.JobFlow{}
				err := fakeClient.Get(ctx, client.ObjectKey{Name: "test-flow", Namespace: "default"}, existing)
				if err == nil && existing.UID == "existing-uid" {
					t.Error("Existing JobFlow was not deleted")
				}
			}
		})
	}
}

func TestJobFlowReconciler_checkActiveDeadline(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		want    bool
		wantErr bool
	}{
		{
			name: "no deadline set",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{},
			},
			want: false,
		},
		{
			name: "not started yet",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						ActiveDeadlineSeconds: intPtr(60),
					},
				},
			},
			want: false,
		},
		{
			name: "deadline not exceeded",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						ActiveDeadlineSeconds: intPtr(3600), // 1 hour
					},
				},
				Status: v1alpha1.JobFlowStatus{
					StartTime: &metav1.Time{Time: now.Add(-30 * time.Minute)},
				},
			},
			want: false,
		},
		{
			name: "deadline exceeded",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						ActiveDeadlineSeconds: intPtr(60), // 1 minute
					},
				},
				Status: v1alpha1.JobFlowStatus{
					StartTime: &metav1.Time{Time: now.Add(-2 * time.Minute)},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{}
			got, err := r.checkActiveDeadline(tt.jobFlow)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkActiveDeadline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkActiveDeadline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobFlowReconciler_checkBackoffLimit(t *testing.T) {
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		want    bool
		wantErr bool
	}{
		{
			name: "no retries - under limit",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", RetryCount: 0},
						{Name: "step2", RetryCount: 0},
					},
				},
			},
			want: false,
		},
		{
			name: "total retries under limit",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						BackoffLimit: int32Ptr(5),
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", RetryCount: 2},
						{Name: "step2", RetryCount: 2},
					},
				},
			},
			want: false,
		},
		{
			name: "total retries exceeds limit",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						BackoffLimit: int32Ptr(5),
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", RetryCount: 3},
						{Name: "step2", RetryCount: 3},
					},
				},
			},
			want: true,
		},
		{
			name: "uses default backoff limit",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", RetryCount: 10}, // Exceeds default of 6
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{}
			got, err := r.checkBackoffLimit(tt.jobFlow)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkBackoffLimit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("checkBackoffLimit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobFlowReconciler_shouldDeleteJobFlow(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		want    bool
		wantErr bool
	}{
		{
			name: "not finished - should not delete",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Phase: v1alpha1.JobFlowPhaseRunning,
				},
			},
			want: false,
		},
		{
			name: "finished but no completion time",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Phase: v1alpha1.JobFlowPhaseSucceeded,
				},
			},
			want: false,
		},
		{
			name: "TTL is 0 - delete immediately",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						TTLSecondsAfterFinished: int32Ptr(0),
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Phase:          v1alpha1.JobFlowPhaseSucceeded,
					CompletionTime: &metav1.Time{Time: now.Add(-1 * time.Hour)},
				},
			},
			want: true,
		},
		{
			name: "TTL not expired",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						TTLSecondsAfterFinished: int32Ptr(3600), // 1 hour
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Phase:          v1alpha1.JobFlowPhaseSucceeded,
					CompletionTime: &metav1.Time{Time: now.Add(-30 * time.Minute)},
				},
			},
			want: false,
		},
		{
			name: "TTL expired",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					ExecutionPolicy: &v1alpha1.ExecutionPolicy{
						TTLSecondsAfterFinished: int32Ptr(3600), // 1 hour
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Phase:          v1alpha1.JobFlowPhaseSucceeded,
					CompletionTime: &metav1.Time{Time: now.Add(-2 * time.Hour)},
				},
			},
			want: true,
		},
		{
			name: "uses default TTL",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Phase:          v1alpha1.JobFlowPhaseFailed,
					CompletionTime: &metav1.Time{Time: now.Add(-25 * time.Hour)}, // Default is 24 hours
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{}
			got, err := r.shouldDeleteJobFlow(tt.jobFlow)
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldDeleteJobFlow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("shouldDeleteJobFlow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

func int32Ptr(i int32) *int32 {
	return &i
}
