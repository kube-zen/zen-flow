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

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestJobFlowReconciler_writeBinaryArtifact(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "binary.bin")

	binaryData := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD}

	err := reconciler.writeBinaryArtifact(binaryData, targetPath)
	if err != nil {
		t.Fatalf("Failed to write binary artifact: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Errorf("Binary artifact file was not created: %s", targetPath)
	}

	// Verify content
	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read binary artifact file: %v", err)
	}
	if len(content) != len(binaryData) {
		t.Errorf("Expected %d bytes, got %d", len(binaryData), len(content))
	}
	for i, b := range binaryData {
		if content[i] != b {
			t.Errorf("Byte mismatch at index %d: expected %x, got %x", i, b, content[i])
		}
	}
}

func TestJobFlowReconciler_writeBinaryArtifact_CreateDirectory(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tmpDir := t.TempDir()
	// Use nested directory to test directory creation
	targetPath := filepath.Join(tmpDir, "nested", "binary.bin")

	binaryData := []byte{0x01, 0x02, 0x03}

	err := reconciler.writeBinaryArtifact(binaryData, targetPath)
	if err != nil {
		t.Fatalf("Failed to write binary artifact (should create directory): %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		t.Errorf("Binary artifact file was not created: %s", targetPath)
	}
}

func TestJobFlowReconciler_storeArtifactInConfigMap_Binary(t *testing.T) {
	reconciler, fakeClient, _ := setupReconcilerTest(t)
	ctx := context.Background()

	tmpDir := t.TempDir()
	artifactPath := filepath.Join(tmpDir, "binary.bin")
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF}
	if err := os.WriteFile(artifactPath, binaryContent, 0644); err != nil {
		t.Fatalf("Failed to create binary artifact file: %v", err)
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

	err := reconciler.storeArtifactInConfigMap(ctx, jobFlow, "step1", "binary-artifact", artifactPath)
	if err != nil {
		t.Fatalf("Failed to store binary artifact in ConfigMap: %v", err)
	}

	// Verify ConfigMap was created
	configMapName := "test-flow-step1-binary-artifact"
	configMap := &corev1.ConfigMap{}
	configMapKey := client.ObjectKey{Namespace: "default", Name: configMapName}
	if err := fakeClient.Get(ctx, configMapKey, configMap); err != nil {
		t.Fatalf("Failed to get ConfigMap: %v", err)
	}

	// Artifacts are stored in Data as strings (base64 encoded for binary)
	if configMap.Data["binary-artifact"] == "" {
		t.Error("Expected artifact data in ConfigMap")
	}
}
