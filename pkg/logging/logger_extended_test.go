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
	"errors"
	"testing"

	jferrors "github.com/kube-zen/zen-flow/pkg/errors"
)

func TestLogger_Info(t *testing.T) {
	logger := NewLogger()
	// Test that Info doesn't panic
	logger.Info("test message")
}

func TestLogger_Infof(t *testing.T) {
	logger := NewLogger()
	// Test that Infof doesn't panic
	logger.Infof("test message %s", "value")
}

func TestLogger_Error(t *testing.T) {
	logger := NewLogger()
	// Test that Error doesn't panic
	logger.Error("test error")
}

func TestLogger_Errorf(t *testing.T) {
	logger := NewLogger()
	// Test that Errorf doesn't panic
	logger.Errorf("test error %s", "value")
}

func TestLogger_Warning(t *testing.T) {
	logger := NewLogger()
	// Test that Warning doesn't panic
	logger.Warning("test warning")
}

func TestLogger_Warningf(t *testing.T) {
	logger := NewLogger()
	// Test that Warningf doesn't panic
	logger.Warningf("test warning %s", "value")
}

func TestLogger_fieldsToKlogArgs(t *testing.T) {
	logger := NewLogger()
	logger = logger.WithField("key1", "value1")
	logger = logger.WithField("key2", "value2")

	args := logger.fieldsToKlogArgs()
	if len(args) != 4 { // 2 fields * 2 (key + value)
		t.Errorf("Expected 4 args, got %d", len(args))
	}

	// Verify key-value pairs
	argsMap := make(map[string]interface{})
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			argsMap[args[i].(string)] = args[i+1]
		}
	}

	if argsMap["key1"] != "value1" {
		t.Error("key1 not found in args")
	}
	if argsMap["key2"] != "value2" {
		t.Error("key2 not found in args")
	}
}

func TestSafeKlogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    int
		expected int
	}{
		{
			name:     "negative level",
			level:    -1,
			expected: 0,
		},
		{
			name:     "zero level",
			level:    0,
			expected: 0,
		},
		{
			name:     "valid level",
			level:    5,
			expected: 5,
		},
		{
			name:     "level above max",
			level:    15,
			expected: 10,
		},
		{
			name:     "max level",
			level:    10,
			expected: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := safeKlogLevel(tt.level)
			if int(result) != tt.expected {
				t.Errorf("safeKlogLevel(%d) = %d, want %d", tt.level, result, tt.expected)
			}
		})
	}
}

func TestVerboseLogger_Info(t *testing.T) {
	logger := NewLogger()
	verboseLogger := logger.V(2)
	// Test that Info doesn't panic
	verboseLogger.Info("test message")
}

func TestVerboseLogger_Infof(t *testing.T) {
	logger := NewLogger()
	verboseLogger := logger.V(2)
	// Test that Infof doesn't panic
	verboseLogger.Infof("test message %s", "value")
}

func TestGetErrorType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "regular error",
			err:      errors.New("test error"),
			expected: "unknown",
		},
		{
			name:     "JobFlowError with type",
			err:      jferrors.New("test_type", "test message"),
			expected: "test_type",
		},
		{
			name:     "JobFlowError without type",
			err:      jferrors.New("", "test message"),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getErrorType(tt.err)
			if result != tt.expected {
				t.Errorf("getErrorType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

