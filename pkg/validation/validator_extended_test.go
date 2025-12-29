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

package validation

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestValidateJobSpec(t *testing.T) {
	tests := []struct {
		name    string
		jobSpec *batchv1.JobSpec
		wantErr bool
	}{
		{
			name: "valid job spec",
			jobSpec: &batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "main", Image: "busybox:latest"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no containers",
			jobSpec: &batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty container name",
			jobSpec: &batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "", Image: "busybox:latest"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "empty container image",
			jobSpec: &batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "main", Image: ""},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "negative backoff limit",
			jobSpec: &batchv1.JobSpec{
				BackoffLimit: int32Ptr(-1),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "main", Image: "busybox:latest"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "zero active deadline seconds",
			jobSpec: &batchv1.JobSpec{
				ActiveDeadlineSeconds: int64Ptr(0),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "main", Image: "busybox:latest"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "negative active deadline seconds",
			jobSpec: &batchv1.JobSpec{
				ActiveDeadlineSeconds: int64Ptr(-1),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "main", Image: "busybox:latest"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "valid backoff limit",
			jobSpec: &batchv1.JobSpec{
				BackoffLimit: int32Ptr(3),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "main", Image: "busybox:latest"},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid active deadline seconds",
			jobSpec: &batchv1.JobSpec{
				ActiveDeadlineSeconds: int64Ptr(300),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "main", Image: "busybox:latest"},
						},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateJobSpec(tt.jobSpec)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateJobSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateContainerResources(t *testing.T) {
	tests := []struct {
		name      string
		resources *corev1.ResourceRequirements
		wantErr   bool
	}{
		{
			name:      "nil resources",
			resources: nil,
			wantErr:   false,
		},
		{
			name: "valid resources",
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("200m"),
					corev1.ResourceMemory: resource.MustParse("256Mi"),
				},
			},
			wantErr: false,
		},
		{
			name: "limit less than request",
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("200m"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("100m"),
				},
			},
			wantErr: true,
		},
		{
			name: "zero request",
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("0"),
				},
			},
			wantErr: true,
		},
		{
			name: "zero limit",
			resources: &corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("0"),
				},
			},
			wantErr: true,
		},
		{
			name: "negative request",
			resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("-100m"),
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateContainerResources(tt.resources)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateContainerResources() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateResourceList(t *testing.T) {
	tests := []struct {
		name      string
		resources corev1.ResourceList
		wantErr   bool
	}{
		{
			name:      "empty resource list",
			resources: corev1.ResourceList{},
			wantErr:   false,
		},
		{
			name: "valid CPU",
			resources: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("100m"),
			},
			wantErr: false,
		},
		{
			name: "valid memory",
			resources: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
			wantErr: false,
		},
		{
			name: "zero CPU",
			resources: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("0"),
			},
			wantErr: true,
		},
		{
			name: "zero memory",
			resources: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("0"),
			},
			wantErr: true,
		},
		{
			name: "negative CPU",
			resources: corev1.ResourceList{
				corev1.ResourceCPU: resource.MustParse("-100m"),
			},
			wantErr: true,
		},
		{
			name: "negative memory",
			resources: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("-128Mi"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResourceList(tt.resources)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateResourceList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCPUQuantity(t *testing.T) {
	tests := []struct {
		name     string
		quantity resource.Quantity
		wantErr  bool
	}{
		{
			name:     "valid CPU",
			quantity: resource.MustParse("100m"),
			wantErr:  false,
		},
		{
			name:     "valid CPU in cores",
			quantity: resource.MustParse("1"),
			wantErr:  false,
		},
		{
			name:     "zero CPU",
			quantity: resource.MustParse("0"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCPUQuantity(tt.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCPUQuantity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMemoryQuantity(t *testing.T) {
	tests := []struct {
		name     string
		quantity resource.Quantity
		wantErr  bool
	}{
		{
			name:     "valid memory",
			quantity: resource.MustParse("128Mi"),
			wantErr:  false,
		},
		{
			name:     "valid memory in bytes",
			quantity: resource.MustParse("134217728"),
			wantErr:  false,
		},
		{
			name:     "zero memory",
			quantity: resource.MustParse("0"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMemoryQuantity(tt.quantity)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMemoryQuantity() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
