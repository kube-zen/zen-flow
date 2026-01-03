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

// TestJobFlowReconciler_checkConcurrencyPolicy is tested in reconciler_test.go
// TestJobFlowReconciler_checkActiveDeadline is tested in reconciler_test.go

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

// TestJobFlowReconciler_shouldDeleteJobFlow is tested in reconciler_test.go

func intPtr(i int) *int {
	return &i
}

func int32Ptr(i int32) *int32 {
	return &i
}
