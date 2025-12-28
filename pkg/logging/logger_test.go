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

package logging

import (
	"context"
	"testing"
	"time"

	jferrors "github.com/kube-zen/zen-flow/pkg/errors"
)

func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}
	if logger.fields == nil {
		t.Error("Logger fields map is nil")
	}
}

func TestLogger_WithField(t *testing.T) {
	logger := NewLogger()
	logger = logger.WithField("key", "value")

	if logger.fields["key"] != "value" {
		t.Errorf("Expected field 'key' to be 'value', got %v", logger.fields["key"])
	}
}

func TestLogger_WithFields(t *testing.T) {
	logger := NewLogger()
	logger = logger.WithFields(
		Field{Key: "key1", Value: "value1"},
		Field{Key: "key2", Value: "value2"},
	)

	if logger.fields["key1"] != "value1" {
		t.Errorf("Expected field 'key1' to be 'value1', got %v", logger.fields["key1"])
	}
	if logger.fields["key2"] != "value2" {
		t.Errorf("Expected field 'key2' to be 'value2', got %v", logger.fields["key2"])
	}
}

func TestLogger_WithJobFlow(t *testing.T) {
	logger := NewLogger()
	logger = logger.WithJobFlow("test-namespace", "test-name")

	if logger.fields["jobflow_namespace"] != "test-namespace" {
		t.Errorf("Expected jobflow_namespace to be 'test-namespace', got %v", logger.fields["jobflow_namespace"])
	}
	if logger.fields["jobflow_name"] != "test-name" {
		t.Errorf("Expected jobflow_name to be 'test-name', got %v", logger.fields["jobflow_name"])
	}
}

func TestLogger_WithStep(t *testing.T) {
	logger := NewLogger()
	logger = logger.WithStep("step1")

	if logger.fields["step_name"] != "step1" {
		t.Errorf("Expected step_name to be 'step1', got %v", logger.fields["step_name"])
	}
}

func TestLogger_WithJob(t *testing.T) {
	logger := NewLogger()
	logger = logger.WithJob("test-namespace", "test-job")

	if logger.fields["job_namespace"] != "test-namespace" {
		t.Errorf("Expected job_namespace to be 'test-namespace', got %v", logger.fields["job_namespace"])
	}
	if logger.fields["job_name"] != "test-job" {
		t.Errorf("Expected job_name to be 'test-job', got %v", logger.fields["job_name"])
	}
}

func TestLogger_WithCorrelationID(t *testing.T) {
	logger := NewLogger()
	logger = logger.WithCorrelationID("test-correlation-id")

	if logger.fields["correlation_id"] != "test-correlation-id" {
		t.Errorf("Expected correlation_id to be 'test-correlation-id', got %v", logger.fields["correlation_id"])
	}
}

func TestLogger_WithError(t *testing.T) {
	logger := NewLogger()
	err := jferrors.New("test_type", "test error")
	logger = logger.WithError(err)

	if logger.fields["error"] == nil {
		t.Error("Expected error field to be set")
	}
	if logger.fields["error_type"] != "test_type" {
		t.Errorf("Expected error_type to be 'test_type', got %v", logger.fields["error_type"])
	}
}

func TestLogger_WithError_Nil(t *testing.T) {
	logger := NewLogger()
	logger = logger.WithError(nil)

	if logger.fields["error"] != nil {
		t.Error("Expected error field to be nil when error is nil")
	}
}

func TestLogger_WithDuration(t *testing.T) {
	logger := NewLogger()
	duration := 100 * time.Millisecond
	logger = logger.WithDuration(duration)

	if logger.fields["duration_ms"] != int64(100) {
		t.Errorf("Expected duration_ms to be 100, got %v", logger.fields["duration_ms"])
	}
}

func TestLogger_Chaining(t *testing.T) {
	logger := NewLogger()
	logger = logger.
		WithJobFlow("test-namespace", "test-name").
		WithStep("step1").
		WithCorrelationID("test-id")

	if logger.fields["jobflow_namespace"] != "test-namespace" {
		t.Error("JobFlow namespace not set correctly")
	}
	if logger.fields["step_name"] != "step1" {
		t.Error("Step name not set correctly")
	}
	if logger.fields["correlation_id"] != "test-id" {
		t.Error("Correlation ID not set correctly")
	}
}

func TestWithCorrelationID_Context(t *testing.T) {
	ctx := context.Background()
	ctx = WithCorrelationID(ctx, "test-id")

	id := GetCorrelationID(ctx)
	if id != "test-id" {
		t.Errorf("Expected correlation ID 'test-id', got '%s'", id)
	}
}

func TestGetCorrelationID_NoID(t *testing.T) {
	ctx := context.Background()
	id := GetCorrelationID(ctx)
	if id != "" {
		t.Errorf("Expected empty correlation ID, got '%s'", id)
	}
}

func TestGetCorrelationID_NilContext(t *testing.T) {
	id := GetCorrelationID(nil)
	if id != "" {
		t.Errorf("Expected empty correlation ID for nil context, got '%s'", id)
	}
}

func TestFromContext(t *testing.T) {
	ctx := context.Background()
	ctx = WithCorrelationID(ctx, "test-id")

	logger := FromContext(ctx)
	if logger.fields["correlation_id"] != "test-id" {
		t.Errorf("Expected correlation_id to be 'test-id', got %v", logger.fields["correlation_id"])
	}
}

func TestGenerateCorrelationID(t *testing.T) {
	id1 := GenerateCorrelationID()
	id2 := GenerateCorrelationID()

	if id1 == id2 {
		t.Error("Generated correlation IDs should be unique")
	}
	if len(id1) == 0 {
		t.Error("Generated correlation ID should not be empty")
	}
}

func TestVerboseLogger(t *testing.T) {
	logger := NewLogger()
	verboseLogger := logger.V(2)

	if verboseLogger.logger != logger {
		t.Error("VerboseLogger should reference the original logger")
	}
	if verboseLogger.level != 2 {
		t.Errorf("Expected verbosity level 2, got %d", verboseLogger.level)
	}
}

func TestSafeKlogLevel_Clamping(t *testing.T) {
	logger := NewLogger()

	// Test negative level
	vl := logger.V(-1)
	if vl.level != -1 {
		t.Errorf("Expected level -1, got %d", vl.level)
	}

	// Test high level
	vl = logger.V(15)
	if vl.level != 15 {
		t.Errorf("Expected level 15, got %d", vl.level)
	}
}

