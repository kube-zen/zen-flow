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
	"os"
	"path/filepath"
	"testing"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestJobFlowReconciler_archiveArtifact_Tar(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	// Create temporary directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	archiveConfig := &v1alpha1.ArchiveConfig{
		Format:      "tar",
		Compression: "none",
	}

	archivePath, err := reconciler.archiveArtifact(testFile, archiveConfig)
	if err != nil {
		t.Fatalf("Failed to archive artifact: %v", err)
	}

	// Verify archive file exists
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Errorf("Archive file was not created: %s", archivePath)
	}

	// Verify extension
	if filepath.Ext(archivePath) != ".tar" {
		t.Errorf("Expected .tar extension, got %s", filepath.Ext(archivePath))
	}
}

func TestJobFlowReconciler_archiveArtifact_TarGzip(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	archiveConfig := &v1alpha1.ArchiveConfig{
		Format:      "tar",
		Compression: "gzip",
	}

	archivePath, err := reconciler.archiveArtifact(testFile, archiveConfig)
	if err != nil {
		t.Fatalf("Failed to archive artifact: %v", err)
	}

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Errorf("Archive file was not created: %s", archivePath)
	}

	// Verify extension is .tar.gz
	if filepath.Ext(archivePath) != ".gz" {
		t.Errorf("Expected .gz extension for gzipped tar, got %s", filepath.Ext(archivePath))
	}
}

func TestJobFlowReconciler_archiveArtifact_Zip(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	archiveConfig := &v1alpha1.ArchiveConfig{
		Format:      "zip",
		Compression: "none",
	}

	archivePath, err := reconciler.archiveArtifact(testFile, archiveConfig)
	if err != nil {
		t.Fatalf("Failed to archive artifact: %v", err)
	}

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Errorf("Archive file was not created: %s", archivePath)
	}

	if filepath.Ext(archivePath) != ".zip" {
		t.Errorf("Expected .zip extension, got %s", filepath.Ext(archivePath))
	}
}

func TestJobFlowReconciler_archiveArtifact_UnsupportedFormat(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	archiveConfig := &v1alpha1.ArchiveConfig{
		Format: "rar", // Unsupported format
	}

	_, err := reconciler.archiveArtifact(testFile, archiveConfig)
	if err == nil {
		t.Error("Expected error for unsupported archive format")
	}
}

func TestJobFlowReconciler_archiveArtifact_DefaultFormat(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Empty archive config should use defaults
	archiveConfig := &v1alpha1.ArchiveConfig{}

	archivePath, err := reconciler.archiveArtifact(testFile, archiveConfig)
	if err != nil {
		t.Fatalf("Failed to archive artifact: %v", err)
	}

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Errorf("Archive file was not created: %s", archivePath)
	}
}

func TestJobFlowReconciler_isArchivePath(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tests := []struct {
		path     string
		expected bool
	}{
		{"file.tar", true},
		{"file.zip", true},
		{"file.gz", true},
		{"file.tar.gz", true},
		{"file.TAR", true},
		{"file.ZIP", true},
		{"file.txt", false},
		{"file", false},
		{"file.tar.gz.backup", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := reconciler.isArchivePath(tt.path)
			if result != tt.expected {
				t.Errorf("Expected %v for %s, got %v", tt.expected, tt.path, result)
			}
		})
	}
}
