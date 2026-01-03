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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

// TestJobFlowReconciler_checkStepTimeouts is tested in reconciler_test.go

func TestJobFlowReconciler_checkStepTimeouts_Additional(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		wantErr bool
	}{
		{
			name: "no running steps",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhasePending},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "step with timeout not exceeded",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name:           "step1",
							TimeoutSeconds: int64Ptr(3600), // 1 hour
						},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{
							Name:      "step1",
							Phase:     v1alpha1.StepPhaseRunning,
							StartTime: &metav1.Time{Time: now.Add(-30 * time.Minute)},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "step timeout exceeded - continue on failure",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name:              "step1",
							TimeoutSeconds:    int64Ptr(60), // 1 minute
							ContinueOnFailure: true,
						},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{
							Name:      "step1",
							Phase:     v1alpha1.StepPhaseRunning,
							StartTime: &metav1.Time{Time: now.Add(-2 * time.Minute)},
							JobRef: &corev1.ObjectReference{
								Name:      "test-job",
								Namespace: "default",
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "step timeout exceeded - fail flow",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name:           "step1",
							TimeoutSeconds: int64Ptr(60), // 1 minute
						},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{
							Name:      "step1",
							Phase:     v1alpha1.StepPhaseRunning,
							StartTime: &metav1.Time{Time: now.Add(-2 * time.Minute)},
							JobRef: &corev1.ObjectReference{
								Name:      "test-job",
								Namespace: "default",
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = v1alpha1.AddToScheme(scheme)
			_ = batchv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			var objects []client.Object
			// Add job if JobRef exists
			for _, stepStatus := range tt.jobFlow.Status.Steps {
				if stepStatus.JobRef != nil {
					job := &batchv1.Job{
						ObjectMeta: metav1.ObjectMeta{
							Name:      stepStatus.JobRef.Name,
							Namespace: stepStatus.JobRef.Namespace,
						},
					}
					objects = append(objects, job)
				}
			}

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
			r := &JobFlowReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				MetricsRecorder: metrics.NewRecorder(),
				EventRecorder:   NewEventRecorder(nil),
			}

			ctx := context.Background()
			err := r.checkStepTimeouts(ctx, tt.jobFlow)

			if (err != nil) != tt.wantErr {
				t.Errorf("checkStepTimeouts() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify timeout handling
			for i := range tt.jobFlow.Status.Steps {
				stepStatus := &tt.jobFlow.Status.Steps[i]
				if stepStatus.Phase == v1alpha1.StepPhaseRunning {
					stepSpec := tt.jobFlow.Spec.Steps[i]
					if stepSpec.TimeoutSeconds != nil && stepStatus.StartTime != nil {
						timeoutTime := stepStatus.StartTime.Add(time.Duration(*stepSpec.TimeoutSeconds) * time.Second)
						if time.Now().After(timeoutTime) {
							// Step should be marked as failed
							if stepStatus.Phase != v1alpha1.StepPhaseFailed {
								t.Errorf("Step %s should be marked as failed due to timeout", stepStatus.Name)
							}
						}
					}
				}
			}
		})
	}
}

// TestJobFlowReconciler_handleStepRetry is tested in reconciler_test.go

func TestJobFlowReconciler_handleStepRetry_Additional(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		stepName string
		wantErr bool
	}{
		{
			name: "no retry policy",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseFailed},
					},
				},
			},
			stepName: "step1",
			wantErr: false,
		},
		{
			name: "step not failed",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name: "step1",
							RetryPolicy: &v1alpha1.RetryPolicy{
								Limit: 3,
							},
						},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseRunning},
					},
				},
			},
			stepName: "step1",
			wantErr: false,
		},
		{
			name: "retry limit exceeded",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name: "step1",
							RetryPolicy: &v1alpha1.RetryPolicy{
								Limit: 3,
							},
						},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhaseFailed, RetryCount: 3},
					},
				},
			},
			stepName: "step1",
			wantErr: false,
		},
		{
			name: "retry with backoff - not time yet",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name: "step1",
							RetryPolicy: &v1alpha1.RetryPolicy{
								Limit: 3,
								Backoff: &v1alpha1.BackoffPolicy{
									Type:     "Fixed",
									Duration: "10s",
								},
							},
						},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{
							Name:           "step1",
							Phase:          v1alpha1.StepPhaseFailed,
							RetryCount:     0,
							CompletionTime: &metav1.Time{Time: now.Add(-5 * time.Second)}, // Too soon
						},
					},
				},
			},
			stepName: "step1",
			wantErr: false,
		},
		{
			name: "retry with backoff - time to retry",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{
							Name: "step1",
							RetryPolicy: &v1alpha1.RetryPolicy{
								Limit: 3,
								Backoff: &v1alpha1.BackoffPolicy{
									Type:     "Fixed",
									Duration: "10s",
								},
							},
						},
					},
				},
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{
							Name:           "step1",
							Phase:          v1alpha1.StepPhaseFailed,
							RetryCount:     0,
							CompletionTime: &metav1.Time{Time: now.Add(-15 * time.Second)}, // Enough time passed
						},
					},
				},
			},
			stepName: "step1",
			wantErr: false,
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

			ctx := context.Background()
			err := r.handleStepRetry(ctx, tt.jobFlow, tt.stepName)

			if (err != nil) != tt.wantErr {
				t.Errorf("handleStepRetry() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Verify retry logic
			stepStatus := r.getStepStatus(tt.jobFlow.Status, tt.stepName)
			if stepStatus != nil {
				stepSpec := tt.jobFlow.Spec.Steps[0]
				if stepSpec.RetryPolicy != nil && stepStatus.Phase == v1alpha1.StepPhaseFailed {
					if stepStatus.RetryCount < stepSpec.RetryPolicy.Limit {
						// Should be retried if enough time has passed
						if stepStatus.CompletionTime != nil {
							backoffDuration := r.calculateBackoff(stepSpec.RetryPolicy, stepStatus.RetryCount)
							nextRetryTime := stepStatus.CompletionTime.Add(backoffDuration)
							if time.Now().After(nextRetryTime) {
								// Should be reset to pending
								if stepStatus.Phase != v1alpha1.StepPhasePending {
									t.Errorf("Step should be reset to pending for retry")
								}
							}
						}
					}
				}
			}
		})
	}
}

// TestJobFlowReconciler_calculateBackoff is tested in reconciler_test.go

func TestJobFlowReconciler_calculateBackoff_Additional(t *testing.T) {
	tests := []struct {
		name       string
		retryPolicy *v1alpha1.RetryPolicy
		retryCount  int32
		wantMin    time.Duration
		wantMax    time.Duration
	}{
		{
			name:        "default exponential backoff",
			retryPolicy: &v1alpha1.RetryPolicy{},
			retryCount:  0,
			wantMin:     time.Second,
			wantMax:     time.Second,
		},
		{
			name:        "default exponential backoff - retry 2",
			retryPolicy: &v1alpha1.RetryPolicy{},
			retryCount:  2,
			wantMin:     4 * time.Second,
			wantMax:     4 * time.Second,
		},
		{
			name: "Fixed backoff",
			retryPolicy: &v1alpha1.RetryPolicy{
				Backoff: &v1alpha1.BackoffPolicy{
					Type:     "Fixed",
					Duration: "5s",
				},
			},
			retryCount: 0,
			wantMin:    5 * time.Second,
			wantMax:    5 * time.Second,
		},
		{
			name: "Linear backoff",
			retryPolicy: &v1alpha1.RetryPolicy{
				Backoff: &v1alpha1.BackoffPolicy{
					Type:     "Linear",
					Duration: "2s",
				},
			},
			retryCount: 3,
			wantMin:    8 * time.Second, // 2s * (3+1)
			wantMax:    8 * time.Second,
		},
		{
			name: "Exponential backoff",
			retryPolicy: &v1alpha1.RetryPolicy{
				Backoff: &v1alpha1.BackoffPolicy{
					Type:     "Exponential",
					Duration: "1s",
					Factor:   floatPtr(2.0),
				},
			},
			retryCount: 2,
			wantMin:    4 * time.Second, // 1s * 2^2
			wantMax:    4 * time.Second,
		},
		{
			name: "Invalid duration - defaults to 1s",
			retryPolicy: &v1alpha1.RetryPolicy{
				Backoff: &v1alpha1.BackoffPolicy{
					Type:     "Fixed",
					Duration: "invalid",
				},
			},
			retryCount: 0,
			wantMin:    time.Second,
			wantMax:    time.Second,
		},
		{
			name: "Unknown type - defaults to exponential",
			retryPolicy: &v1alpha1.RetryPolicy{
				Backoff: &v1alpha1.BackoffPolicy{
					Type:     "Unknown",
					Duration: "1s",
				},
			},
			retryCount: 1,
			wantMin:    time.Second, // 1s * 2^1
			wantMax:    2 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{}
			got := r.calculateBackoff(tt.retryPolicy, tt.retryCount)

			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("calculateBackoff() = %v, want between %v and %v", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestJobFlowReconciler_applyPodFailureAction is tested in reconciler_test.go

func TestJobFlowReconciler_applyPodFailureAction_Additional(t *testing.T) {
	tests := []struct {
		name   string
		action string
		want   bool
	}{
		{
			name:   "Ignore action",
			action: "Ignore",
			want:   false,
		},
		{
			name:   "Count action",
			action: "Count",
			want:   false,
		},
		{
			name:   "FailJob action",
			action: "FailJob",
			want:   true,
		},
		{
			name:   "Unknown action - defaults to fail",
			action: "Unknown",
			want:   true,
		},
		{
			name:   "Empty action - defaults to fail",
			action: "",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{}
			got := r.applyPodFailureAction(tt.action)
			if got != tt.want {
				t.Errorf("applyPodFailureAction(%q) = %v, want %v", tt.action, got, tt.want)
			}
		})
	}
}

func TestJobFlowReconciler_updateConditions(t *testing.T) {
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		want    corev1.ConditionStatus
		wantReason string
	}{
		{
			name: "Succeeded phase",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Phase: v1alpha1.JobFlowPhaseSucceeded,
				},
			},
			want:        corev1.ConditionTrue,
			wantReason:  "FlowSucceeded",
		},
		{
			name: "Failed phase",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Phase: v1alpha1.JobFlowPhaseFailed,
				},
			},
			want:        corev1.ConditionFalse,
			wantReason:  "FlowFailed",
		},
		{
			name: "Running phase",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Phase: v1alpha1.JobFlowPhaseRunning,
				},
			},
			want:        corev1.ConditionTrue,
			wantReason:  "FlowRunning",
		},
		{
			name: "Pending phase",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Phase: v1alpha1.JobFlowPhasePending,
				},
			},
			want:        corev1.ConditionFalse,
			wantReason:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{}
			r.updateConditions(tt.jobFlow)

			// Find Ready condition
			var readyCondition *v1alpha1.JobFlowCondition
			for i := range tt.jobFlow.Status.Conditions {
				if tt.jobFlow.Status.Conditions[i].Type == "Ready" {
					readyCondition = &tt.jobFlow.Status.Conditions[i]
					break
				}
			}

			if readyCondition == nil {
				t.Error("Ready condition not found")
				return
			}

			if readyCondition.Status != tt.want {
				t.Errorf("Condition Status = %v, want %v", readyCondition.Status, tt.want)
			}

			if tt.wantReason != "" && readyCondition.Reason != tt.wantReason {
				t.Errorf("Condition Reason = %v, want %v", readyCondition.Reason, tt.wantReason)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}

// int64Ptr helper is defined in reconciler_test.go

func floatPtr(f float64) *float64 {
	return &f
}

