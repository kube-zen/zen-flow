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
// This package is a wrapper around zen-sdk/pkg/errors for backward compatibility.
package errors

import (
	sdkerrors "github.com/kube-zen/zen-sdk/pkg/errors"
)

// JobFlowError is an alias for zen-sdk's ContextError.
// This maintains backward compatibility while using the shared implementation.
type JobFlowError = sdkerrors.ContextError

// WithJobFlow adds JobFlow context to an error.
func WithJobFlow(err error, namespace, name string) *JobFlowError {
	return sdkerrors.WithMultipleContext(err, map[string]string{
		"jobflow_namespace": namespace,
		"jobflow_name":      name,
	})
}

// WithStep adds step context to an error.
func WithStep(err error, stepName string) *JobFlowError {
	return sdkerrors.WithContext(err, "step_name", stepName)
}

// WithJob adds job context to an error.
func WithJob(err error, namespace, name string) *JobFlowError {
	return sdkerrors.WithMultipleContext(err, map[string]string{
		"job_namespace": namespace,
		"job_name":      name,
	})
}

// New creates a new JobFlowError.
func New(errType, message string) *JobFlowError {
	return sdkerrors.New(errType, message)
}

// Wrap wraps an error with a message and type.
func Wrap(err error, errType, message string) *JobFlowError {
	return sdkerrors.Wrap(err, errType, message)
}

// Wrapf wraps an error with a formatted message and type.
func Wrapf(err error, errType, format string, args ...interface{}) *JobFlowError {
	return sdkerrors.Wrapf(err, errType, format, args...)
}
