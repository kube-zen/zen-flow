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

func TestJobFlowReconciler_validateJobFlow(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		wantErr bool
	}{
		{
			name: "valid jobflow",
			jobFlow: &v1alpha1.JobFlow{
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
			},
			wantErr: false,
		},
		{
			name: "empty steps",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{},
				},
			},
			wantErr: true,
		},
		{
			name: "empty step name",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: ""},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate step names",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1"},
						{Name: "step1"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid dependency",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1", Dependencies: []string{"nonexistent"}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid dependency",
			jobFlow: &v1alpha1.JobFlow{
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
			},
			wantErr: false,
		},
		{
			name: "empty dependency",
			jobFlow: &v1alpha1.JobFlow{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "default",
				},
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1", Dependencies: []string{""}},
					},
				},
			},
			wantErr: false, // Empty dependency is allowed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reconciler.validateJobFlow(tt.jobFlow)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJobFlow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

