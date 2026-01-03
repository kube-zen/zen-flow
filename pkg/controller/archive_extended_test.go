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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestJobFlowReconciler_createTarArchive_WithMultipleFiles(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0750); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	archivePath := filepath.Join(tmpDir, "test.tar")
	err := reconciler.createTarArchive(sourceDir, archivePath, false)
	if err != nil {
		t.Fatalf("Failed to create tar archive: %v", err)
	}

	// Verify archive can be read
	//nolint:gosec // archivePath comes from test temp directory, safe
	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file: %v", err)
		}
	}()

	tarReader := tar.NewReader(file)
	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}

	// The file name might be relative to the source directory
	if header.Name != "test.txt" && header.Name != "source/test.txt" {
		t.Errorf("Expected file name 'test.txt' or 'source/test.txt', got %s", header.Name)
	}
}

func TestJobFlowReconciler_createTarArchive_Gzip(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0750); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	err := reconciler.createTarArchive(sourceDir, archivePath, true)
	if err != nil {
		t.Fatalf("Failed to create gzipped tar archive: %v", err)
	}

	// Verify archive can be read
	//nolint:gosec // archivePath comes from test temp directory, safe
	file, err := os.Open(archivePath)
	if err != nil {
		t.Fatalf("Failed to open archive: %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Logf("Failed to close file: %v", err)
		}
	}()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatalf("Failed to create gzip reader: %v", err)
	}
	defer func() {
		if err := gzipReader.Close(); err != nil {
			t.Logf("Failed to close gzip reader: %v", err)
		}
	}()

	tarReader := tar.NewReader(gzipReader)
	header, err := tarReader.Next()
	if err != nil {
		t.Fatalf("Failed to read tar header: %v", err)
	}

	// The file name might be relative to the source directory
	if header.Name != "test.txt" && header.Name != "source/test.txt" {
		t.Errorf("Expected file name 'test.txt' or 'source/test.txt', got %s", header.Name)
	}
}

func TestJobFlowReconciler_createZipArchive_WithMultipleFiles(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	if err := os.MkdirAll(sourceDir, 0750); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	testFile := filepath.Join(sourceDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	archivePath := filepath.Join(tmpDir, "test.zip")
	err := reconciler.createZipArchive(sourceDir, archivePath)
	if err != nil {
		t.Fatalf("Failed to create zip archive: %v", err)
	}

	// Verify archive can be read
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		t.Fatalf("Failed to open zip archive: %v", err)
	}
	defer func() {
		if err := zipReader.Close(); err != nil {
			t.Logf("Failed to close zip reader: %v", err)
		}
	}()

	if len(zipReader.File) == 0 {
		t.Error("Expected at least one file in zip archive")
	}

	found := false
	for _, f := range zipReader.File {
		if f.Name == "test.txt" || f.Name == "source/test.txt" {
			found = true
			// Verify content
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open file in zip: %v", err)
			}
			content, err := io.ReadAll(rc)
			if err := rc.Close(); err != nil {
				t.Logf("Failed to close zip file reader: %v", err)
			}
			if err != nil {
				t.Fatalf("Failed to read file content: %v", err)
			}
			if string(content) != "test content" {
				t.Errorf("Expected content 'test content', got %s", string(content))
			}
			break
		}
	}
	if !found {
		t.Errorf("Expected file 'test.txt' in zip archive, found files: %v", func() []string {
			var names []string
			for _, f := range zipReader.File {
				names = append(names, f.Name)
			}
			return names
		}())
	}
}

func TestJobFlowReconciler_archiveArtifact_ZipGzip(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	archiveConfig := &v1alpha1.ArchiveConfig{
		Format:      "zip",
		Compression: "gzip",
	}

	// Zip format doesn't support gzip compression, should use zip compression
	archivePath, err := reconciler.archiveArtifact(testFile, archiveConfig)
	if err != nil {
		t.Fatalf("Failed to archive artifact: %v", err)
	}

	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Errorf("Archive file was not created: %s", archivePath)
	}

	// Verify it's a zip file
	if filepath.Ext(archivePath) != ".zip" {
		t.Errorf("Expected .zip extension, got %s", filepath.Ext(archivePath))
	}
}
