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

func TestJobFlowReconciler_evaluateTemplate(t *testing.T) {
	r := &JobFlowReconciler{}

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
			},
		},
	}

	ctx := &TemplateContext{
		JobFlow:    jobFlow,
		StepStatus: make(map[string]*v1alpha1.StepStatus),
	}
	ctx.StepStatus["step1"] = &jobFlow.Status.Steps[0]

	tests := []struct {
		name    string
		tmplStr string
		ctx     *TemplateContext
		want    string
		wantErr bool
	}{
		{
			name:    "empty template",
			tmplStr: "",
			ctx:     ctx,
			want:    "",
			wantErr: false,
		},
		{
			name:    "simple string template",
			tmplStr: "hello world",
			ctx:     ctx,
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "template with JobFlow name",
			tmplStr: "{{.JobFlow.Name}}",
			ctx:     ctx,
			want:    "test-flow",
			wantErr: false,
		},
		{
			name:    "template with step status",
			tmplStr: "{{.StepStatus.step1.Phase}}",
			ctx:     ctx,
			want:    "Succeeded",
			wantErr: false,
		},
		{
			name:    "invalid template syntax",
			tmplStr: "{{.Invalid",
			ctx:     ctx,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.evaluateTemplate(tt.tmplStr, tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("evaluateTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("evaluateTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobFlowReconciler_evaluateSimpleCondition(t *testing.T) {
	r := &JobFlowReconciler{}

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{Name: "test-flow", Namespace: "default"},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{Name: "step1", Phase: v1alpha1.StepPhaseSucceeded},
				{Name: "step2", Phase: v1alpha1.StepPhaseFailed},
			},
		},
	}

	ctx := &TemplateContext{
		JobFlow:    jobFlow,
		StepStatus: make(map[string]*v1alpha1.StepStatus),
	}
	ctx.StepStatus["step1"] = &jobFlow.Status.Steps[0]
	ctx.StepStatus["step2"] = &jobFlow.Status.Steps[1]

	tests := []struct {
		name      string
		condition string
		want      bool
	}{
		{
			name:      "step succeeded pattern",
			condition: "steps.step1.phase == 'Succeeded'",
			want:      true,
		},
		{
			name:      "step succeeded pattern with double quotes",
			condition: "steps.step1.phase == \"Succeeded\"",
			want:      true,
		},
		{
			name:      "step failed pattern",
			condition: "steps.step2.phase == 'Failed'",
			want:      true,
		},
		{
			name:      "step not failed pattern",
			condition: "steps.step1.phase != 'Failed'",
			want:      true,
		},
		{
			name:      "non-matching step",
			condition: "steps.step3.phase == 'Succeeded'",
			want:      true, // Defaults to true for non-matching patterns
		},
		{
			name:      "no step pattern",
			condition: "some other condition",
			want:      true, // Defaults to true
		},
		{
			name:      "empty condition",
			condition: "",
			want:      true, // Defaults to true
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.evaluateSimpleCondition(tt.condition, ctx)
			if got != tt.want {
				t.Errorf("evaluateSimpleCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}
