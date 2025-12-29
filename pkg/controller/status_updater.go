//go:build legacy
// +build legacy

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
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/klog/v2"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

// JobFlowGVR is the GroupVersionResource for JobFlow CRDs.
var JobFlowGVR = schema.GroupVersionResource{
	Group:    "workflow.kube-zen.io",
	Version:  "v1alpha1",
	Resource: "jobflows",
}

// StatusUpdater updates JobFlow CRD status subresource.
type StatusUpdater struct {
	dynClient dynamic.Interface
}

// NewStatusUpdater creates a new status updater.
func NewStatusUpdater(dynClient dynamic.Interface) *StatusUpdater {
	return &StatusUpdater{
		dynClient: dynClient,
	}
}

// UpdateStatus updates the JobFlow CRD status subresource.
func (s *StatusUpdater) UpdateStatus(
	ctx context.Context,
	jobFlow *v1alpha1.JobFlow,
) error {
	// Get the current JobFlow CRD
	unstructuredJobFlow, err := s.dynClient.Resource(JobFlowGVR).
		Namespace(jobFlow.Namespace).
		Get(ctx, jobFlow.Name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get JobFlow CRD %s/%s: %v", jobFlow.Namespace, jobFlow.Name, err)
		return err
	}

	// Convert JobFlow status to unstructured format
	statusBytes, err := json.Marshal(jobFlow.Status)
	if err != nil {
		klog.Errorf("Failed to marshal JobFlow status %s/%s: %v", jobFlow.Namespace, jobFlow.Name, err)
		return err
	}

	var statusObj map[string]interface{}
	if err := json.Unmarshal(statusBytes, &statusObj); err != nil {
		klog.Errorf("Failed to unmarshal JobFlow status %s/%s: %v", jobFlow.Namespace, jobFlow.Name, err)
		return err
	}

	// Set status
	unstructuredJobFlow.Object["status"] = statusObj

	// Update status subresource
	_, err = s.dynClient.Resource(JobFlowGVR).
		Namespace(jobFlow.Namespace).
		UpdateStatus(ctx, unstructuredJobFlow, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("Failed to update JobFlow status %s/%s: %v", jobFlow.Namespace, jobFlow.Name, err)
		return err
	}

	klog.V(4).Infof("Updated JobFlow status: %s/%s (phase=%s)", jobFlow.Namespace, jobFlow.Name, jobFlow.Status.Phase)
	return nil
}
