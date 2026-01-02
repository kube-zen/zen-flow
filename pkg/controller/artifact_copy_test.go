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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestJobFlowReconciler_copyArtifactFromConfigMap(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "artifact.txt")

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "default",
		},
		Data: map[string]string{
			"artifact": "test artifact content",
		},
	}

	err := reconciler.copyArtifactFromConfigMap(ctx, configMap, "artifact", targetPath)
	if err != nil {
		t.Fatalf("Failed to copy artifact: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Errorf("Artifact file was not created: %s", targetPath)
	}

	// Verify content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read artifact file: %v", err)
	}
	if string(content) != "test artifact content" {
		t.Errorf("Expected content 'test artifact content', got %s", string(content))
	}
}

func TestJobFlowReconciler_copyArtifactFromConfigMap_BinaryData(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "artifact.bin")

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "default",
		},
		BinaryData: map[string][]byte{
			"artifact": []byte{0x01, 0x02, 0x03, 0x04},
		},
	}

	err := reconciler.copyArtifactFromConfigMap(ctx, configMap, "artifact", targetPath)
	if err != nil {
		t.Fatalf("Failed to copy binary artifact: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read binary artifact file: %v", err)
	}
	if len(content) != 4 || content[0] != 0x01 {
		t.Errorf("Binary content mismatch")
	}
}

func TestJobFlowReconciler_copyArtifactFromConfigMap_NotFound(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "artifact.txt")

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "default",
		},
		Data: map[string]string{},
	}

	err := reconciler.copyArtifactFromConfigMap(ctx, configMap, "nonexistent", targetPath)
	if err == nil {
		t.Error("Expected error when artifact not found in ConfigMap")
	}
}

func TestJobFlowReconciler_storeArtifactInConfigMap(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "artifact.txt")
	artifactContent := "test artifact content"
	if err := os.WriteFile(artifactPath, []byte(artifactContent), 0644); err != nil {
		t.Fatalf("Failed to create artifact file: %v", err)
	}

	jobFlow := &v1alpha1.JobFlow{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
			UID:       "test-uid",
		},
		Spec: v1alpha1.JobFlowSpec{
			Steps: []v1alpha1.Step{
				{Name: "step1"},
			},
		},
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	err := reconciler.storeArtifactInConfigMap(ctx, jobFlow, "step1", "artifact", artifactPath)
	if err != nil {
		t.Fatalf("Failed to store artifact in ConfigMap: %v", err)
	}

	// Verify ConfigMap was created
	configMapName := "test-flow-step1-artifact"
	configMap := &corev1.ConfigMap{}
	configMapKey := types.NamespacedName{Namespace: "default", Name: configMapName}
	if err := fakeClient.Get(ctx, configMapKey, configMap); err != nil {
		t.Fatalf("Failed to get ConfigMap: %v", err)
	}

	if configMap.Data["artifact"] != artifactContent {
		t.Errorf("Expected artifact content %s, got %s", artifactContent, configMap.Data["artifact"])
	}
}

func TestJobFlowReconciler_storeArtifactInConfigMap_TooLarge(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "large-artifact.txt")
	// Create a file larger than ConfigMapSizeLimit (1MB)
	largeContent := make([]byte, ConfigMapSizeLimit+1)
	for i := range largeContent {
		largeContent[i] = 'A'
	}
	if err := os.WriteFile(artifactPath, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large artifact file: %v", err)
	}

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
	}

	if err := fakeClient.Create(ctx, jobFlow); err != nil {
		t.Fatalf("Failed to create JobFlow: %v", err)
	}

	// Should not fail, but should skip ConfigMap storage
	err := reconciler.storeArtifactInConfigMap(ctx, jobFlow, "step1", "large-artifact", artifactPath)
	if err != nil {
		t.Fatalf("storeArtifactInConfigMap should not fail for large artifacts: %v", err)
	}

	// Verify ConfigMap was NOT created
	configMapName := "test-flow-step1-large-artifact"
	configMap := &corev1.ConfigMap{}
	configMapKey := types.NamespacedName{Namespace: "default", Name: configMapName}
	err = fakeClient.Get(ctx, configMapKey, configMap)
	if err == nil {
		t.Error("Expected ConfigMap not to be created for large artifacts")
	}
}

func TestJobFlowReconciler_ensureArtifactVolumeMount(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	job := &batchv1.Job{
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "main"},
					},
				},
			},
		},
	}

	reconciler.ensureArtifactVolumeMount(job, "shared-pvc", "/artifacts")

	// Verify volume was added
	if len(job.Spec.Template.Spec.Volumes) == 0 {
		t.Error("Expected volume to be added")
	}

	// Verify volume mount was added to container
	if len(job.Spec.Template.Spec.Containers[0].VolumeMounts) == 0 {
		t.Error("Expected volume mount to be added to container")
	}

	// Verify volume mount path
	if job.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath != "/artifacts" {
		t.Errorf("Expected mount path /artifacts, got %s", job.Spec.Template.Spec.Containers[0].VolumeMounts[0].MountPath)
	}
}
