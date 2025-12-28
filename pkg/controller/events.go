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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// eventSinkWrapper wraps EventInterface to implement record.EventSink.
type eventSinkWrapper struct {
	events v1.EventInterface
}

func (e *eventSinkWrapper) Create(event *corev1.Event) (*corev1.Event, error) {
	return e.events.Create(context.Background(), event, metav1.CreateOptions{})
}

func (e *eventSinkWrapper) Update(event *corev1.Event) (*corev1.Event, error) {
	return e.events.Update(context.Background(), event, metav1.UpdateOptions{})
}

func (e *eventSinkWrapper) Patch(oldEvent *corev1.Event, data []byte) (*corev1.Event, error) {
	return e.events.Patch(context.Background(), oldEvent.Name, types.MergePatchType, data, metav1.PatchOptions{})
}

// EventRecorder records events for JobFlow resources.
type EventRecorder struct {
	recorder record.EventRecorder
}

// NewEventRecorder creates a new event recorder.
func NewEventRecorder(kubeClient kubernetes.Interface) *EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	if kubeClient != nil {
		eventBroadcaster.StartRecordingToSink(&eventSinkWrapper{
			events: kubeClient.CoreV1().Events(""),
		})
	}
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "zen-flow-controller"})

	return &EventRecorder{
		recorder: recorder,
	}
}

// Eventf records an event with formatting.
func (r *EventRecorder) Eventf(object runtime.Object, eventType, reason, messageFmt string, args ...interface{}) {
	r.recorder.Eventf(object, eventType, reason, messageFmt, args...)
}

// Event records an event.
func (r *EventRecorder) Event(object runtime.Object, eventType, reason, message string) {
	r.recorder.Event(object, eventType, reason, message)
}

