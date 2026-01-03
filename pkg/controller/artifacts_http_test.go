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
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestJobFlowReconciler_fetchArtifactFromHTTP_WithHeaders(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)
	ctx := context.Background()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer token123" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "downloaded.txt")

	httpArtifact := &v1alpha1.HTTPArtifact{
		URL: server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer token123",
		},
	}

	err := reconciler.fetchArtifactFromHTTP(ctx, httpArtifact, targetPath)
	if err != nil {
		t.Fatalf("Failed to fetch artifact with headers: %v", err)
	}
}

func TestJobFlowReconciler_fetchArtifactFromHTTP_HTTPError(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)
	ctx := context.Background()

	// Create a test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	targetPath := filepath.Join(tmpDir, "downloaded.txt")

	httpArtifact := &v1alpha1.HTTPArtifact{
		URL: server.URL,
	}

	err := reconciler.fetchArtifactFromHTTP(ctx, httpArtifact, targetPath)
	if err == nil {
		t.Error("Expected error for HTTP error status")
	}
}

func TestJobFlowReconciler_fetchArtifactFromHTTP_CreateDirectory(t *testing.T) {
	reconciler, _, _ := setupReconcilerTest(t)
	ctx := context.Background()

	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test content"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	// Use nested directory to test directory creation
	targetPath := filepath.Join(tmpDir, "nested", "downloaded.txt")

	httpArtifact := &v1alpha1.HTTPArtifact{
		URL: server.URL,
	}

	err := reconciler.fetchArtifactFromHTTP(ctx, httpArtifact, targetPath)
	if err != nil {
		t.Fatalf("Failed to fetch artifact (should create directory): %v", err)
	}
}
