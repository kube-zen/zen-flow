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
	"errors"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/informers"
	kubefake "k8s.io/client-go/kubernetes/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

func TestNewJobFlowController(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	jobInformer := informerFactory.Batch().V1().Jobs()
	pvcInformer := informerFactory.Core().V1().PersistentVolumeClaims()

	metricsRecorder := metrics.NewRecorder()
	eventRecorder := NewEventRecorder(kubeClient)
	statusUpdater := NewStatusUpdater(dynamicClient)

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		pvcInformer,
		statusUpdater,
		metricsRecorder,
		eventRecorder,
	)

	if controller == nil {
		t.Fatal("NewJobFlowController returned nil")
	}
	if controller.dynamicClient != dynamicClient {
		t.Error("Controller did not set dynamicClient correctly")
	}
	if controller.kubeClient != kubeClient {
		t.Error("Controller did not set kubeClient correctly")
	}
	if controller.jobFlowInformer != jobFlowInformer {
		t.Error("Controller did not set jobFlowInformer correctly")
	}
	if controller.statusUpdater != statusUpdater {
		t.Error("Controller did not set statusUpdater correctly")
	}
}

func TestJobFlowController_Stop(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	jobFlowGVR := schema.GroupVersionResource{
		Group:    "workflow.zen.io",
		Version:  "v1alpha1",
		Resource: "jobflows",
	}
	factory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute)
	jobFlowInformer := factory.ForResource(jobFlowGVR).Informer()

	informerFactory := informers.NewSharedInformerFactory(kubeClient, time.Minute)
	jobInformer := informerFactory.Batch().V1().Jobs()
	pvcInformer := informerFactory.Core().V1().PersistentVolumeClaims()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		jobFlowInformer,
		jobInformer,
		pvcInformer,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	// Stop should not panic
	controller.Stop()

	// Verify context is canceled
	select {
	case <-controller.ctx.Done():
		// Expected
	default:
		t.Error("Stop() did not cancel context")
	}
}

func TestJobFlowController_validateJobFlow(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		wantErr bool
	}{
		{
			name: "valid JobFlow",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1", Dependencies: []string{}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "empty steps",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{},
				},
			},
			wantErr: true,
		},
		{
			name: "empty step name",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "", Dependencies: []string{}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "duplicate step names",
			jobFlow: &v1alpha1.JobFlow{
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1", Dependencies: []string{}},
						{Name: "step1", Dependencies: []string{}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid dependency",
			jobFlow: &v1alpha1.JobFlow{
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
				Spec: v1alpha1.JobFlowSpec{
					Steps: []v1alpha1.Step{
						{Name: "step1", Dependencies: []string{}},
						{Name: "step2", Dependencies: []string{"step1"}},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := controller.validateJobFlow(tt.jobFlow)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJobFlow() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobFlowController_hasInitialized(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	tests := []struct {
		name    string
		jobFlow *v1alpha1.JobFlow
		want    bool
	}{
		{
			name: "not initialized - empty phase",
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
						{Name: "step1"},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := controller.hasInitialized(tt.jobFlow); got != tt.want {
				t.Errorf("hasInitialized() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJobFlowController_isRetryable(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	kubeClient := kubefake.NewSimpleClientset()

	controller := NewJobFlowController(
		dynamicClient,
		kubeClient,
		nil,
		nil,
		nil,
		NewStatusUpdater(dynamicClient),
		metrics.NewRecorder(),
		NewEventRecorder(kubeClient),
	)

	tests := []struct {
		name    string
		err     error
		want    bool
	}{
		{
			name: "conflict error",
			err:   k8serrors.NewConflict(schema.GroupResource{}, "test", errors.New("conflict")),
			want:  true,
		},
		{
			name: "server timeout error",
			err:   k8serrors.NewServerTimeout(schema.GroupResource{}, "test", 0),
			want:  true,
		},
		{
			name: "non-retryable error",
			err:   errors.New("some error"),
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := controller.isRetryable(tt.err); got != tt.want {
				t.Errorf("isRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

