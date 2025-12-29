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
	"encoding/json"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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
