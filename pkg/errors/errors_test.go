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
			err: &JobFlowError{
				Type:    "test_error",
				Message: "test message",
			},
			wantErr: "test message",
		},
		{
			name: "error with underlying error",
			err: &JobFlowError{
				Type:    "test_error",
				Message: "test message",
				Err:     errors.New("underlying error"),
			},
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
	err := &JobFlowError{
		Type:    "test_error",
		Message: "test message",
		Err:     underlying,
	}

	if unwrapped := err.Unwrap(); unwrapped != underlying {
		t.Errorf("JobFlowError.Unwrap() = %v, want %v", unwrapped, underlying)
	}

	errNoWrap := &JobFlowError{
		Type:    "test_error",
		Message: "test message",
	}
	if unwrapped := errNoWrap.Unwrap(); unwrapped != nil {
		t.Errorf("JobFlowError.Unwrap() = %v, want nil", unwrapped)
	}
}

func TestWithJobFlow(t *testing.T) {
	originalErr := errors.New("original error")
	err := WithJobFlow(originalErr, "test-namespace", "test-name")

	if err.JobFlowNamespace != "test-namespace" {
		t.Errorf("JobFlowNamespace = %v, want test-namespace", err.JobFlowNamespace)
	}
	if err.JobFlowName != "test-name" {
		t.Errorf("JobFlowName = %v, want test-name", err.JobFlowName)
	}
	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
	}
}

func TestWithStep(t *testing.T) {
	originalErr := errors.New("original error")
	err := WithStep(originalErr, "step1")

	if err.StepName != "step1" {
		t.Errorf("StepName = %v, want step1", err.StepName)
	}
	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
	}
}

func TestWithJob(t *testing.T) {
	originalErr := errors.New("original error")
	err := WithJob(originalErr, "test-namespace", "test-job")

	if err.JobNamespace != "test-namespace" {
		t.Errorf("JobNamespace = %v, want test-namespace", err.JobNamespace)
	}
	if err.JobName != "test-job" {
		t.Errorf("JobName = %v, want test-job", err.JobName)
	}
	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
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
	if err.Err != nil {
		t.Errorf("Err = %v, want nil", err.Err)
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
	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
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
	if err.Err != originalErr {
		t.Errorf("Err = %v, want %v", err.Err, originalErr)
	}
}

func TestWithJobFlow_PreservesExistingContext(t *testing.T) {
	originalErr := errors.New("original error")
	err1 := WithStep(originalErr, "step1")
	err2 := WithJobFlow(err1, "test-namespace", "test-name")

	if err2.StepName != "step1" {
		t.Errorf("StepName should be preserved: got %v, want step1", err2.StepName)
	}
	if err2.JobFlowNamespace != "test-namespace" {
		t.Errorf("JobFlowNamespace = %v, want test-namespace", err2.JobFlowNamespace)
	}
	if err2.JobFlowName != "test-name" {
		t.Errorf("JobFlowName = %v, want test-name", err2.JobFlowName)
	}
}
