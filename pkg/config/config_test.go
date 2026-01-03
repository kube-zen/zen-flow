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

package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	// Clear environment variables
	os.Clearenv()

	cfg := Load()

	// Verify defaults
	if cfg.DefaultTTLSeconds != 86400 {
		t.Errorf("Expected DefaultTTLSeconds 86400, got %d", cfg.DefaultTTLSeconds)
	}
	if cfg.ConfigMapSizeLimit != 1024*1024 {
		t.Errorf("Expected ConfigMapSizeLimit %d, got %d", 1024*1024, cfg.ConfigMapSizeLimit)
	}
	if cfg.DefaultDirPerm != 0755 {
		t.Errorf("Expected DefaultDirPerm 0755, got %o", cfg.DefaultDirPerm)
	}
	if cfg.DefaultFilePerm != 0644 {
		t.Errorf("Expected DefaultFilePerm 0644, got %o", cfg.DefaultFilePerm)
	}
	if cfg.UIDTruncateLength != 8 {
		t.Errorf("Expected UIDTruncateLength 8, got %d", cfg.UIDTruncateLength)
	}
	if cfg.DefaultBackoffLimit != 6 {
		t.Errorf("Expected DefaultBackoffLimit 6, got %d", cfg.DefaultBackoffLimit)
	}
	if cfg.DefaultRetryLimit != 3 {
		t.Errorf("Expected DefaultRetryLimit 3, got %d", cfg.DefaultRetryLimit)
	}
	if cfg.DefaultBackoffBase != 1*time.Second {
		t.Errorf("Expected DefaultBackoffBase 1s, got %v", cfg.DefaultBackoffBase)
	}
	if cfg.DefaultBackoffFactor != 2.0 {
		t.Errorf("Expected DefaultBackoffFactor 2.0, got %f", cfg.DefaultBackoffFactor)
	}
	if cfg.DefaultConfigMapKey != "value" {
		t.Errorf("Expected DefaultConfigMapKey 'value', got %s", cfg.DefaultConfigMapKey)
	}
	if cfg.DefaultContainerName != "main" {
		t.Errorf("Expected DefaultContainerName 'main', got %s", cfg.DefaultContainerName)
	}
	if cfg.DefaultConcurrencyPolicy != "Forbid" {
		t.Errorf("Expected DefaultConcurrencyPolicy 'Forbid', got %s", cfg.DefaultConcurrencyPolicy)
	}
	if cfg.DefaultContentType != "application/octet-stream" {
		t.Errorf("Expected DefaultContentType 'application/octet-stream', got %s", cfg.DefaultContentType)
	}
	if cfg.DefaultArchiveFormat != "tar" {
		t.Errorf("Expected DefaultArchiveFormat 'tar', got %s", cfg.DefaultArchiveFormat)
	}
	if cfg.DefaultCompression != "none" {
		t.Errorf("Expected DefaultCompression 'none', got %s", cfg.DefaultCompression)
	}
}

func TestLoad_FromEnvironment(t *testing.T) {
	// Set environment variables
	if err := os.Setenv("ZEN_FLOW_DEFAULT_TTL_SECONDS", "3600"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	if err := os.Setenv("ZEN_FLOW_CONFIGMAP_SIZE_LIMIT", "2048000"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	if err := os.Setenv("ZEN_FLOW_UID_TRUNCATE_LENGTH", "12"); err != nil {
		t.Fatalf("Failed to set environment variable: %v", err)
	}
	os.Setenv("ZEN_FLOW_DEFAULT_BACKOFF_LIMIT", "10")
	os.Setenv("ZEN_FLOW_DEFAULT_RETRY_LIMIT", "5")
	os.Setenv("ZEN_FLOW_DEFAULT_BACKOFF_BASE", "2s")
	os.Setenv("ZEN_FLOW_DEFAULT_BACKOFF_FACTOR", "3.0")
	os.Setenv("ZEN_FLOW_DEFAULT_CONFIGMAP_KEY", "data")
	os.Setenv("ZEN_FLOW_DEFAULT_CONTAINER_NAME", "worker")
	os.Setenv("ZEN_FLOW_DEFAULT_CONCURRENCY_POLICY", "Allow")
	os.Setenv("ZEN_FLOW_DEFAULT_CONTENT_TYPE", "application/json")
	os.Setenv("ZEN_FLOW_DEFAULT_ARCHIVE_FORMAT", "zip")
	os.Setenv("ZEN_FLOW_DEFAULT_COMPRESSION", "gzip")
	os.Setenv("ZEN_FLOW_DEFAULT_DIR_PERM", "0777")
	os.Setenv("ZEN_FLOW_DEFAULT_FILE_PERM", "0666")

	defer func() {
		os.Clearenv()
	}()

	cfg := Load()

	// Verify environment values
	if cfg.DefaultTTLSeconds != 3600 {
		t.Errorf("Expected DefaultTTLSeconds 3600, got %d", cfg.DefaultTTLSeconds)
	}
	if cfg.ConfigMapSizeLimit != 2048000 {
		t.Errorf("Expected ConfigMapSizeLimit 2048000, got %d", cfg.ConfigMapSizeLimit)
	}
	if cfg.UIDTruncateLength != 12 {
		t.Errorf("Expected UIDTruncateLength 12, got %d", cfg.UIDTruncateLength)
	}
	if cfg.DefaultBackoffLimit != 10 {
		t.Errorf("Expected DefaultBackoffLimit 10, got %d", cfg.DefaultBackoffLimit)
	}
	if cfg.DefaultRetryLimit != 5 {
		t.Errorf("Expected DefaultRetryLimit 5, got %d", cfg.DefaultRetryLimit)
	}
	if cfg.DefaultBackoffBase != 2*time.Second {
		t.Errorf("Expected DefaultBackoffBase 2s, got %v", cfg.DefaultBackoffBase)
	}
	if cfg.DefaultBackoffFactor != 3.0 {
		t.Errorf("Expected DefaultBackoffFactor 3.0, got %f", cfg.DefaultBackoffFactor)
	}
	if cfg.DefaultConfigMapKey != "data" {
		t.Errorf("Expected DefaultConfigMapKey 'data', got %s", cfg.DefaultConfigMapKey)
	}
	if cfg.DefaultContainerName != "worker" {
		t.Errorf("Expected DefaultContainerName 'worker', got %s", cfg.DefaultContainerName)
	}
	if cfg.DefaultConcurrencyPolicy != "Allow" {
		t.Errorf("Expected DefaultConcurrencyPolicy 'Allow', got %s", cfg.DefaultConcurrencyPolicy)
	}
	if cfg.DefaultContentType != "application/json" {
		t.Errorf("Expected DefaultContentType 'application/json', got %s", cfg.DefaultContentType)
	}
	if cfg.DefaultArchiveFormat != "zip" {
		t.Errorf("Expected DefaultArchiveFormat 'zip', got %s", cfg.DefaultArchiveFormat)
	}
	if cfg.DefaultCompression != "gzip" {
		t.Errorf("Expected DefaultCompression 'gzip', got %s", cfg.DefaultCompression)
	}
	if cfg.DefaultDirPerm != 0777 {
		t.Errorf("Expected DefaultDirPerm 0777, got %o", cfg.DefaultDirPerm)
	}
	if cfg.DefaultFilePerm != 0666 {
		t.Errorf("Expected DefaultFilePerm 0666, got %o", cfg.DefaultFilePerm)
	}
}

func TestLoad_InvalidValues(t *testing.T) {
	// Set invalid environment variables - should fall back to defaults
	os.Setenv("ZEN_FLOW_DEFAULT_TTL_SECONDS", "invalid")
	os.Setenv("ZEN_FLOW_CONFIGMAP_SIZE_LIMIT", "not-a-number")
	os.Setenv("ZEN_FLOW_DEFAULT_BACKOFF_BASE", "invalid-duration")

	defer func() {
		os.Clearenv()
	}()

	cfg := Load()

	// Should use defaults for invalid values
	if cfg.DefaultTTLSeconds != 86400 {
		t.Errorf("Expected default DefaultTTLSeconds 86400, got %d", cfg.DefaultTTLSeconds)
	}
	if cfg.ConfigMapSizeLimit != 1024*1024 {
		t.Errorf("Expected default ConfigMapSizeLimit %d, got %d", 1024*1024, cfg.ConfigMapSizeLimit)
	}
	if cfg.DefaultBackoffBase != 1*time.Second {
		t.Errorf("Expected default DefaultBackoffBase 1s, got %v", cfg.DefaultBackoffBase)
	}
}

