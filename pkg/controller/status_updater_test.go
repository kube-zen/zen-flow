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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestNewStatusUpdater(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)

	updater := NewStatusUpdater(dynamicClient)
	if updater == nil {
		t.Fatal("NewStatusUpdater returned nil")
	}
	if updater.dynClient != dynamicClient {
		t.Error("StatusUpdater did not set dynamic client correctly")
	}
}

func TestStatusUpdater_UpdateStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	v1alpha1.AddToScheme(scheme)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Phase: v1alpha1.JobFlowPhaseRunning,
		},
	}

	// Convert to unstructured for fake client
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobFlow)
	if err != nil {
		t.Fatalf("Failed to convert to unstructured: %v", err)
	}

	// Create unstructured object with proper GVK
	unstructuredJobFlow := &unstructured.Unstructured{Object: unstructuredObj}
	unstructuredJobFlow.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("JobFlow"))

	dynamicClient := fake.NewSimpleDynamicClient(scheme, unstructuredJobFlow)
	updater := NewStatusUpdater(dynamicClient)

	ctx := context.Background()
	err = updater.UpdateStatus(ctx, jobFlow)
	if err != nil {
		t.Errorf("UpdateStatus returned error: %v", err)
	}
}

func TestStatusUpdater_UpdateStatus_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	dynamicClient := fake.NewSimpleDynamicClient(scheme)
	updater := NewStatusUpdater(dynamicClient)

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "nonexistent",
			Namespace: "default",
		},
		Status: v1alpha1.JobFlowStatus{
			Phase: v1alpha1.JobFlowPhaseRunning,
		},
	}

	ctx := context.Background()
	err := updater.UpdateStatus(ctx, jobFlow)
	if err == nil {
		t.Error("UpdateStatus should return error for nonexistent JobFlow")
	}
}

