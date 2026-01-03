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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

func TestJobFlowReconciler_hasInitialized(t *testing.T) {
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		want    bool
	}{
		{
			name: "not initialized - empty phase and no steps",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{},
			},
			want: false,
		},
		{
			name: "initialized - has phase",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Phase: v1alpha1.JobFlowPhasePending,
				},
			},
			want: true,
		},
		{
			name: "initialized - has steps",
			jobFlow: &v1alpha1.JobFlow{
				Status: v1alpha1.JobFlowStatus{
					Steps: []v1alpha1.StepStatus{
						{Name: "step1", Phase: v1alpha1.StepPhasePending},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &JobFlowReconciler{}
			got := r.hasInitialized(tt.jobFlow)
			if got != tt.want {
				t.Errorf("hasInitialized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobFlowReconciler_initializeJobFlow(t *testing.T) {
	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		wantErr bool
	}{
		{
			name: "successful initialization",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
						{Name: "step2"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "initialization with resource templates",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-flow",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
					},
					ResourceTemplates: &v1alpha1.ResourceTemplates{
						VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
							{
								ObjectMeta: metav1.ObjectMeta{Name: "pvc1"},
								Spec: corev1.PersistentVolumeClaimSpec{
									AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
									Resources: corev1.VolumeResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceStorage: *resource.NewQuantity(1, resource.DecimalSI),
										},
									},
								},
							},
						},
						ConfigMapTemplates: []corev1.ConfigMap{
							{
								ObjectMeta: metav1.ObjectMeta{Name: "cm1"},
								Data:       map[string]string{"key": "value"},
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
			_ = corev1.AddToScheme(scheme)

			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			r := &JobFlowReconciler{
				Client:          fakeClient,
				Scheme:          scheme,
				MetricsRecorder: metrics.NewRecorder(),
				EventRecorder:   NewEventRecorder(nil),
			}

			ctx := context.Background()
			err := r.initializeJobFlow(ctx, tt.jobFlow)

			if (err != nil) != tt.wantErr {
				t.Errorf("initializeJobFlow() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify initialization
				if tt.jobFlow.Status.Phase != v1alpha1.JobFlowPhasePending {
					t.Errorf("Status.Phase = %v, want %v", tt.jobFlow.Status.Phase, v1alpha1.JobFlowPhasePending)
				}
				if tt.jobFlow.Status.StartTime == nil {
					t.Error("Status.StartTime is nil")
				}
				if tt.jobFlow.Status.Progress == nil {
					t.Error("Status.Progress is nil")
				}
				if len(tt.jobFlow.Status.Steps) != len(tt.jobFlow.Spec.Steps) {
					t.Errorf("Status.Steps length = %d, want %d", len(tt.jobFlow.Status.Steps), len(tt.jobFlow.Spec.Steps))
				}
			}
		})
	}
}

// TestJobFlowReconciler_createResourceTemplates is tested in reconciler_test.go
