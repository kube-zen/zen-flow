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

// Package errors provides structured error types for the JobFlow controller with context.
package errors

import (
	"errors"
	"fmt"
)

// JobFlowError represents a JobFlow error with context.
type JobFlowError struct {
	// Type categorizes the error (e.g., "reconciliation_failed", "step_execution_failed")
	Type string

	// JobFlowNamespace is the namespace of the JobFlow (if applicable)
	JobFlowNamespace string

	// JobFlowName is the name of the JobFlow (if applicable)
	JobFlowName string

	// StepName is the name of the step (if applicable)
	StepName string

	// JobNamespace is the namespace of the job (if applicable)
	JobNamespace string

	// JobName is the name of the job (if applicable)
	JobName string

	// Message is the error message
	Message string

	// Err is the underlying error
	Err error
}

// Error implements the error interface.
func (e *JobFlowError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *JobFlowError) Unwrap() error {
	return e.Err
}

// WithJobFlow adds JobFlow context to an error.
func WithJobFlow(err error, namespace, name string) *JobFlowError {
	var jferr *JobFlowError
	if errors.As(err, &jferr) && jferr != nil {
		jferr.JobFlowNamespace = namespace
		jferr.JobFlowName = name
		return jferr
	}
	return &JobFlowError{
		Message:          err.Error(),
		Err:              err,
		JobFlowNamespace: namespace,
		JobFlowName:      name,
	}
}

// WithStep adds step context to an error.
func WithStep(err error, stepName string) *JobFlowError {
	var jferr *JobFlowError
	if errors.As(err, &jferr) && jferr != nil {
		jferr.StepName = stepName
		return jferr
	}
	return &JobFlowError{
		Message: err.Error(),
		Err:     err,
		StepName: stepName,
	}
}

// WithJob adds job context to an error.
func WithJob(err error, namespace, name string) *JobFlowError {
	var jferr *JobFlowError
	if errors.As(err, &jferr) && jferr != nil {
		jferr.JobNamespace = namespace
		jferr.JobName = name
		return jferr
	}
	return &JobFlowError{
		Message:      err.Error(),
		Err:          err,
		JobNamespace: namespace,
		JobName:      name,
	}
}

// New creates a new JobFlowError.
func New(errType, message string) *JobFlowError {
	return &JobFlowError{
		Type:    errType,
		Message: message,
	}
}

// Wrap wraps an error with a message and type.
func Wrap(err error, errType, message string) *JobFlowError {
	return &JobFlowError{
		Type:    errType,
		Message: message,
		Err:     err,
	}
}

// Wrapf wraps an error with a formatted message and type.
func Wrapf(err error, errType, format string, args ...interface{}) *JobFlowError {
	return &JobFlowError{
		Type:    errType,
		Message: fmt.Sprintf(format, args...),
		Err:     err,
	}
}

