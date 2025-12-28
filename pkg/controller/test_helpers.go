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

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

// setupJobFlowInFakeClient adds a JobFlow to the fake dynamic client for testing.
func setupJobFlowInFakeClient(dynamicClient *fake.FakeDynamicClient, jobFlow *v1alpha1.JobFlow) error {
	// Convert to unstructured
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(jobFlow)
	if err != nil {
		return err
	}

	// Create unstructured object with proper GVK
	unstructuredJobFlow := &unstructured.Unstructured{Object: unstructuredObj}
	unstructuredJobFlow.SetGroupVersionKind(v1alpha1.SchemeGroupVersion.WithKind("JobFlow"))

	// Add to fake client - use context.Background() for testing
	ctx := context.Background()
	_, err = dynamicClient.Resource(v1alpha1.SchemeGroupVersion.WithResource("jobflows")).
		Namespace(jobFlow.Namespace).
		Create(ctx, unstructuredJobFlow, metav1.CreateOptions{})
	return err
}

// mustMarshalJobTemplate marshals a Job template to JSON with proper Kind and APIVersion.
func mustMarshalJobTemplate(job *batchv1.Job) []byte {
	unstructuredJob, err := runtime.DefaultUnstructuredConverter.ToUnstructured(job)
	if err != nil {
		panic(err)
	}

	// Ensure Kind and APIVersion are set in the unstructured object
	if unstructuredJob["kind"] == nil {
		unstructuredJob["kind"] = "Job"
	}
	if unstructuredJob["apiVersion"] == nil {
		unstructuredJob["apiVersion"] = batchv1.SchemeGroupVersion.String()
	}

	raw, err := json.Marshal(unstructuredJob)
	if err != nil {
		panic(err)
	}
	return raw
}
