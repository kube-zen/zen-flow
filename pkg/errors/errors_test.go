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

package errors

import (
	"errors"
	"testing"
)

func TestJobFlowError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *JobFlowError
		wantErr string
	}{
		{
			name: "error with message only",
			err:  New("test_error", "test message"),
			wantErr: "test message",
		},
		{
			name: "error with underlying error",
			err:  Wrap(errors.New("underlying error"), "test_error", "test message"),
			wantErr: "test message: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantErr {
				t.Errorf("JobFlowError.Error() = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

func TestJobFlowError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := Wrap(underlying, "test_error", "test message")

	if unwrapped := err.Unwrap(); unwrapped != underlying {
		t.Errorf("JobFlowError.Unwrap() = %v, want %v", unwrapped, underlying)
	}

	errNoWrap := New("test_error", "test message")
	if unwrapped := errNoWrap.Unwrap(); unwrapped != nil {
		t.Errorf("JobFlowError.Unwrap() = %v, want nil", unwrapped)
	}
}

func TestWithJobFlow(t *testing.T) {
	originalErr := errors.New("original error")
	err := WithJobFlow(originalErr, "test-namespace", "test-name")

	if err.GetContext("jobflow_namespace") != "test-namespace" {
		t.Errorf("jobflow_namespace = %v, want test-namespace", err.GetContext("jobflow_namespace"))
	}
	if err.GetContext("jobflow_name") != "test-name" {
		t.Errorf("jobflow_name = %v, want test-name", err.GetContext("jobflow_name"))
	}
	if err.Unwrap() != originalErr {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), originalErr)
	}
}

func TestWithStep(t *testing.T) {
	originalErr := errors.New("original error")
	err := WithStep(originalErr, "step1")

	if err.GetContext("step_name") != "step1" {
		t.Errorf("step_name = %v, want step1", err.GetContext("step_name"))
	}
	if err.Unwrap() != originalErr {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), originalErr)
	}
}

func TestWithJob(t *testing.T) {
	originalErr := errors.New("original error")
	err := WithJob(originalErr, "test-namespace", "test-job")

	if err.GetContext("job_namespace") != "test-namespace" {
		t.Errorf("job_namespace = %v, want test-namespace", err.GetContext("job_namespace"))
	}
	if err.GetContext("job_name") != "test-job" {
		t.Errorf("job_name = %v, want test-job", err.GetContext("job_name"))
	}
	if err.Unwrap() != originalErr {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), originalErr)
	}
}

func TestNew(t *testing.T) {
	err := New("test_type", "test message")

	if err.Type != "test_type" {
		t.Errorf("Type = %v, want test_type", err.Type)
	}
	if err.Message != "test message" {
		t.Errorf("Message = %v, want test message", err.Message)
	}
	if err.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", err.Unwrap())
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := Wrap(originalErr, "test_type", "test message")

	if err.Type != "test_type" {
		t.Errorf("Type = %v, want test_type", err.Type)
	}
	if err.Message != "test message" {
		t.Errorf("Message = %v, want test message", err.Message)
	}
	if err.Unwrap() != originalErr {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), originalErr)
	}
}

func TestWrapf(t *testing.T) {
	originalErr := errors.New("original error")
	err := Wrapf(originalErr, "test_type", "test %s", "message")

	if err.Type != "test_type" {
		t.Errorf("Type = %v, want test_type", err.Type)
	}
	if err.Message != "test message" {
		t.Errorf("Message = %v, want test message", err.Message)
	}
	if err.Unwrap() != originalErr {
		t.Errorf("Unwrap() = %v, want %v", err.Unwrap(), originalErr)
	}
}

func TestWithJobFlow_PreservesExistingContext(t *testing.T) {
	originalErr := errors.New("original error")
	err1 := WithStep(originalErr, "step1")
	err2 := WithJobFlow(err1, "test-namespace", "test-name")

	if err2.GetContext("step_name") != "step1" {
		t.Errorf("step_name should be preserved: got %v, want step1", err2.GetContext("step_name"))
	}
	if err2.GetContext("jobflow_namespace") != "test-namespace" {
		t.Errorf("jobflow_namespace = %v, want test-namespace", err2.GetContext("jobflow_namespace"))
	}
	if err2.GetContext("jobflow_name") != "test-name" {
		t.Errorf("jobflow_name = %v, want test-name", err2.GetContext("jobflow_name"))
	}
}
