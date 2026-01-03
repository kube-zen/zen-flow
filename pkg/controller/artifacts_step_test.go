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
	"os"
	"path/filepath"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestJobFlowReconciler_fetchArtifactFromStep(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
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
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseSucceeded,
					Outputs: &v1alpha1.StepOutputs{
						Artifacts: []v1alpha1.ArtifactOutput{
							{
								Name: "artifact1",
								Path: "/tmp/artifact1.txt",
							},
						},
					},
				},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "downloaded.txt")

	// Test successful fetch from ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow-step1-artifact1",
			Namespace: "default",
		},
		Data: map[string]string{
			"artifact1": "test content",
		},
	}
	if err := fakeClient.Create(ctx, configMap); err != nil {
		t.Fatalf("Failed to create ConfigMap: %v", err)
	}

	err := reconciler.fetchArtifactFromStep(ctx, jobFlow, "step1", "artifact1", targetPath)
	if err != nil {
		t.Fatalf("Failed to fetch artifact: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Errorf("Artifact file was not created: %s", targetPath)
	}

	// Verify content
	//nolint:gosec // targetPath comes from test temp directory, safe
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read artifact file: %v", err)
	}
	if string(content) != "test content" {
		t.Errorf("Expected content 'test content', got %s", string(content))
	}
}

func TestJobFlowReconciler_fetchArtifactFromStep_StepNotFound(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "downloaded.txt")

	err := reconciler.fetchArtifactFromStep(ctx, jobFlow, "nonexistent", "artifact1", targetPath)
	if err == nil {
		t.Error("Expected error when step not found")
	}
}

func TestJobFlowReconciler_fetchArtifactFromStep_StepNotCompleted(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseRunning,
				},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "downloaded.txt")

	err := reconciler.fetchArtifactFromStep(ctx, jobFlow, "step1", "artifact1", targetPath)
	if err == nil {
		t.Error("Expected error when step not completed")
	}
}

func TestJobFlowReconciler_fetchArtifactFromStep_ArtifactNotFound(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseSucceeded,
					Outputs: &v1alpha1.StepOutputs{
						Artifacts: []v1alpha1.ArtifactOutput{},
					},
				},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "downloaded.txt")

	err := reconciler.fetchArtifactFromStep(ctx, jobFlow, "step1", "nonexistent", targetPath)
	if err == nil {
		t.Error("Expected error when artifact not found")
	}
}

func TestJobFlowReconciler_fetchArtifactFromStep_NoOutputs(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseSucceeded,
					// No Outputs field
				},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "downloaded.txt")

	err := reconciler.fetchArtifactFromStep(ctx, jobFlow, "step1", "artifact1", targetPath)
	if err == nil {
		t.Error("Expected error when outputs is nil")
	}
}

func TestJobFlowReconciler_fetchArtifactFromStep_PVCBased(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
		Status: v1alpha1.JobFlowStatus{
			Steps: []v1alpha1.StepStatus{
				{
					Name:  "step1",
					Phase: v1alpha1.StepPhaseSucceeded,
					Outputs: &v1alpha1.StepOutputs{
						Artifacts: []v1alpha1.ArtifactOutput{
							{
								Name: "artifact1",
								Path: "/pvc/artifact1.txt",
							},
						},
					},
				},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "downloaded.txt")

	// No ConfigMap exists, should return nil (PVC-based, handled by job containers)
	err := reconciler.fetchArtifactFromStep(ctx, jobFlow, "step1", "artifact1", targetPath)
	if err != nil {
		t.Errorf("Expected nil for PVC-based artifacts, got: %v", err)
	}
}

