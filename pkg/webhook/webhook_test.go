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

package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
)

func TestWebhookServer_handleValidate(t *testing.T) {
	server, err := NewWebhookServer(":0", "", "")
	if err != nil {
		t.Fatalf("Failed to create webhook server: %v", err)
	}

	tests := []struct {
		name            string
		request         admissionv1.AdmissionReview
		expectedAllowed bool
		expectedCode    int
	}{
		{
			name: "valid JobFlow",
			request: admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID: "test-uid",
					Kind: metav1.GroupVersionKind{
						Group:   "workflow.zen.io",
						Version: "v1alpha1",
						Kind:    "JobFlow",
					},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalJobFlow(t, &v1alpha1.JobFlow{
							Spec: v1alpha1.JobFlowSpec{
								Steps: []v1alpha1.Step{
									{
										Name:         "step1",
										Dependencies: []string{},
										Template: runtime.RawExtension{
											Raw: []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}`),
										},
									},
								},
							},
						}),
					},
				},
			},
			expectedAllowed: true,
			expectedCode:    http.StatusOK,
		},
		{
			name: "invalid JobFlow - no steps",
			request: admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID: "test-uid-2",
					Kind: metav1.GroupVersionKind{
						Group:   "workflow.zen.io",
						Version: "v1alpha1",
						Kind:    "JobFlow",
					},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalJobFlow(t, &v1alpha1.JobFlow{
							Spec: v1alpha1.JobFlowSpec{
								Steps: []v1alpha1.Step{},
							},
						}),
					},
				},
			},
			expectedAllowed: false,
			expectedCode:    http.StatusOK,
		},
		{
			name: "invalid JobFlow - duplicate step names",
			request: admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID: "test-uid-3",
					Kind: metav1.GroupVersionKind{
						Group:   "workflow.zen.io",
						Version: "v1alpha1",
						Kind:    "JobFlow",
					},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalJobFlow(t, &v1alpha1.JobFlow{
							Spec: v1alpha1.JobFlowSpec{
								Steps: []v1alpha1.Step{
									{Name: "step1"},
									{Name: "step1"}, // Duplicate
								},
							},
						}),
					},
				},
			},
			expectedAllowed: false,
			expectedCode:    http.StatusOK,
		},
		{
			name: "invalid JobFlow - invalid dependency",
			request: admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID: "test-uid-4",
					Kind: metav1.GroupVersionKind{
						Group:   "workflow.zen.io",
						Version: "v1alpha1",
						Kind:    "JobFlow",
					},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalJobFlow(t, &v1alpha1.JobFlow{
							Spec: v1alpha1.JobFlowSpec{
								Steps: []v1alpha1.Step{
									{
										Name:         "step1",
										Dependencies: []string{"nonexistent-step"},
									},
								},
							},
						}),
					},
				},
			},
			expectedAllowed: false,
			expectedCode:    http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/validate-jobflow", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleValidate(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			var response admissionv1.AdmissionReview
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Response.Allowed != tt.expectedAllowed {
				t.Errorf("Expected allowed=%v, got %v", tt.expectedAllowed, response.Response.Allowed)
			}
		})
	}
}

func TestWebhookServer_handleMutate(t *testing.T) {
	server, err := NewWebhookServer(":0", "", "")
	if err != nil {
		t.Fatalf("Failed to create webhook server: %v", err)
	}

	tests := []struct {
		name         string
		request      admissionv1.AdmissionReview
		expectPatch  bool
		expectedCode int
	}{
		{
			name: "JobFlow without execution policy - should add defaults",
			request: admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID: "test-uid",
					Kind: metav1.GroupVersionKind{
						Group:   "workflow.zen.io",
						Version: "v1alpha1",
						Kind:    "JobFlow",
					},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalJobFlow(t, &v1alpha1.JobFlow{
							Spec: v1alpha1.JobFlowSpec{
								Steps: []v1alpha1.Step{
									{Name: "step1"},
								},
							},
						}),
					},
				},
			},
			expectPatch:  true,
			expectedCode: http.StatusOK,
		},
		{
			name: "JobFlow with execution policy - no patch needed",
			request: admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID: "test-uid-2",
					Kind: metav1.GroupVersionKind{
						Group:   "workflow.zen.io",
						Version: "v1alpha1",
						Kind:    "JobFlow",
					},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Raw: marshalJobFlow(t, &v1alpha1.JobFlow{
							Spec: v1alpha1.JobFlowSpec{
								ExecutionPolicy: &v1alpha1.ExecutionPolicy{
									ConcurrencyPolicy: "Forbid",
								},
								Steps: []v1alpha1.Step{
									{Name: "step1"},
								},
							},
						}),
					},
				},
			},
			expectPatch:  true, // Will add missing defaults
			expectedCode: http.StatusOK,
		},
		{
			name: "UPDATE operation - no mutation",
			request: admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					UID: "test-uid-3",
					Kind: metav1.GroupVersionKind{
						Group:   "workflow.zen.io",
						Version: "v1alpha1",
						Kind:    "JobFlow",
					},
					Operation: admissionv1.Update,
					Object: runtime.RawExtension{
						Raw: marshalJobFlow(t, &v1alpha1.JobFlow{
							Spec: v1alpha1.JobFlowSpec{
								Steps: []v1alpha1.Step{
									{Name: "step1"},
								},
							},
						}),
					},
				},
			},
			expectPatch:  false,
			expectedCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/mutate-jobflow", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleMutate(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("Expected status code %d, got %d", tt.expectedCode, w.Code)
			}

			var response admissionv1.AdmissionReview
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if !response.Response.Allowed {
				t.Errorf("Expected mutation to be allowed")
			}

			hasPatch := len(response.Response.Patch) > 0
			if hasPatch != tt.expectPatch {
				t.Errorf("Expected patch=%v, got %v", tt.expectPatch, hasPatch)
			}
		})
	}
}

func TestWebhookServer_healthz(t *testing.T) {
	server, err := NewWebhookServer(":0", "", "")
	if err != nil {
		t.Fatalf("Failed to create webhook server: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	// Access the handler through the server's mux
	server.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", w.Body.String())
	}
}

func TestWebhookServer_Start(t *testing.T) {
	server, err := NewWebhookServer(":0", "", "")
	if err != nil {
		t.Fatalf("Failed to create webhook server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(ctx)
	}()

	// Wait for context to cancel
	<-ctx.Done()

	// Check that server stopped gracefully
	select {
	case err := <-errChan:
		if err != nil && err.Error() != "webhook server error: context canceled" {
			t.Errorf("Unexpected error: %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("Server did not stop within timeout")
	}
}

func marshalJobFlow(t *testing.T, jobFlow *v1alpha1.JobFlow) []byte {
	jobFlow.TypeMeta = metav1.TypeMeta{
		APIVersion: "workflow.zen.io/v1alpha1",
		Kind:       "JobFlow",
	}
	raw, err := json.Marshal(jobFlow)
	if err != nil {
		t.Fatalf("Failed to marshal JobFlow: %v", err)
	}
	return raw
}
