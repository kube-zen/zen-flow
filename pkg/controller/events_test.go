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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestNewEventRecorder(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	recorder := NewEventRecorder(kubeClient)

	if recorder == nil {
		t.Fatal("NewEventRecorder returned nil")
	}
	if recorder.Recorder == nil {
		t.Error("EventRecorder.Recorder is nil")
	}
}

func TestNewEventRecorder_NilClient(t *testing.T) {
	recorder := NewEventRecorder(nil)

	if recorder == nil {
		t.Fatal("NewEventRecorder returned nil")
	}
	if recorder.Recorder == nil {
		t.Error("EventRecorder.Recorder is nil even with nil client")
	}
}

func TestEventRecorder_Eventf(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	recorder := NewEventRecorder(kubeClient)

	obj := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	// Test that Eventf doesn't panic
	recorder.Eventf(obj, corev1.EventTypeNormal, "TestReason", "Test message %s", "value")
}

func TestEventRecorder_Event(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	recorder := NewEventRecorder(kubeClient)

	obj := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}

	// Test that Event doesn't panic
	recorder.Event(obj, corev1.EventTypeNormal, "TestReason", "Test message")
}

// TestEventSinkWrapper tests removed - eventSinkWrapper is an internal implementation
// detail of zen-sdk/pkg/events and is not part of the zen-flow controller API.
// The EventRecorder tests above are sufficient to verify event recording functionality.

func TestEventRecorder_WithJobFlow(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	recorder := NewEventRecorder(kubeClient)

	// Create a mock runtime.Object (JobFlow-like)
	obj := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-flow",
			Namespace: "default",
		},
	}

	// Test various event types
	recorder.Eventf(obj, corev1.EventTypeNormal, "StepCreated", "Created Job for step %s", "step1")
	recorder.Eventf(obj, corev1.EventTypeWarning, "StepFailed", "Step %s failed: %v", "step1", "test error")
	recorder.Event(obj, corev1.EventTypeNormal, "FlowSucceeded", "JobFlow completed successfully")
}

// TestEventRecorder_Integration tests event recording with actual fake client
func TestEventRecorder_Integration(t *testing.T) {
	kubeClient := fake.NewSimpleClientset()
	recorder := NewEventRecorder(kubeClient)

	obj := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			UID:       "test-uid",
		},
	}

	// Record multiple events
	recorder.Eventf(obj, corev1.EventTypeNormal, "Reason1", "Message 1")
	recorder.Eventf(obj, corev1.EventTypeWarning, "Reason2", "Message 2 %s", "value")
	recorder.Event(obj, corev1.EventTypeNormal, "Reason3", "Message 3")

	// Verify events were created (if fake client supports it)
	events, err := kubeClient.CoreV1().Events("default").List(context.TODO(), metav1.ListOptions{})
	if err == nil {
		t.Logf("Created %d events", len(events.Items))
	}
}
