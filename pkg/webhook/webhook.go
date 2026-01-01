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

// Package webhook provides HTTP server for validating and mutating admission webhooks.
package webhook

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-sdk/pkg/logging"
	"github.com/kube-zen/zen-flow/pkg/validation"
)

var (
	// Codecs is the codec factory for deserializing admission requests.
	Codecs = serializer.NewCodecFactory(scheme.Scheme)

	// ErrUnexpectedObjectType indicates an unexpected object type was encountered.
	ErrUnexpectedObjectType = errors.New("expected JobFlow")
)

var (
	// schemeInitialized tracks if the scheme has been initialized
	schemeInitialized = false
	// schemeInitError stores any error from scheme initialization
	schemeInitError error
)

func init() {
	// Add JobFlow to scheme for deserialization
	// Note: This is called during package initialization, so we store the error
	// and check it when creating the webhook server
	if err := v1alpha1.AddToScheme(scheme.Scheme); err != nil {
		schemeInitError = fmt.Errorf("failed to add scheme: %w", err)
	} else {
		schemeInitialized = true
	}
}

// WebhookServer handles admission webhook requests.
type WebhookServer struct {
	server *http.Server
}

// NewWebhookServer creates a new webhook server.
func NewWebhookServer(addr, certFile, keyFile string) (*WebhookServer, error) {
	// Check if scheme was initialized successfully
	if !schemeInitialized {
		if schemeInitError != nil {
			return nil, fmt.Errorf("webhook server initialization failed: %w", schemeInitError)
		}
		return nil, fmt.Errorf("webhook server initialization failed: scheme not initialized")
	}

	mux := http.NewServeMux()
	ws := &WebhookServer{}

	// Register validation endpoint
	mux.HandleFunc("/validate-jobflow", ws.handleValidate)

	// Register mutation endpoint
	mux.HandleFunc("/mutate-jobflow", ws.handleMutate)

	// Health check endpoint for webhook
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	ws.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return ws, nil
}

// Start starts the webhook server without TLS (for testing).
func (ws *WebhookServer) Start(ctx context.Context) error {
	logger := logging.NewLogger("zen-flow-webhook")
	logger.WithContext(ctx).Info("Starting webhook server without TLS (testing mode)...")

	go func() {
		<-ctx.Done()
		logger.WithContext(ctx).Info("Shutting down webhook server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := ws.server.Shutdown(shutdownCtx); err != nil {
			logger.WithContext(ctx).Error(err, "Error shutting down webhook server")
		}
	}()

	if err := ws.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("webhook server error: %w", err)
	}

	return nil
}

// StartTLS starts the webhook server with TLS.
func (ws *WebhookServer) StartTLS(ctx context.Context, certFile, keyFile string) error {
	logger := logging.NewLogger("zen-flow-webhook")
	logger.WithContext(ctx).Info("Starting webhook server with TLS",
		logging.String("address", ws.server.Addr))

	go func() {
		<-ctx.Done()
		logger.WithContext(ctx).Info("Shutting down webhook server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := ws.server.Shutdown(shutdownCtx); err != nil {
			logger.WithContext(ctx).Error(err, "Error shutting down webhook server")
		}
	}()

	if err := ws.server.ListenAndServeTLS(certFile, keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("webhook server error: %w", err)
	}

	return nil
}

// handleValidate handles admission review requests for JobFlow validation.
func (ws *WebhookServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	logger := logging.NewLogger("zen-flow-webhook")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read admission review request
	ctx := r.Context()
	var review admissionv1.AdmissionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		logger.WithContext(ctx).Error(err, "Failed to decode admission review")
		http.Error(w, fmt.Sprintf("Failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	// Set response UID
	response := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: &admissionv1.AdmissionResponse{
			UID: review.Request.UID,
		},
	}

	// Validate the JobFlow
	if err := ws.validateJobFlow(review.Request); err != nil {
		logger.WithContext(ctx).Debug("JobFlow validation failed",
			logging.String("error", err.Error()))
		response.Response.Allowed = false
		response.Response.Result = &metav1.Status{
			Code:    http.StatusUnprocessableEntity,
			Message: err.Error(),
		}
	} else {
		response.Response.Allowed = true
		logger.WithContext(ctx).Debug("JobFlow validation succeeded")
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.WithContext(ctx).Error(err, "Failed to encode admission review response")
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// validateJobFlow validates a JobFlow from an admission request.
func (ws *WebhookServer) validateJobFlow(req *admissionv1.AdmissionRequest) error {
	// Only validate CREATE and UPDATE operations
	if req.Operation != admissionv1.Create && req.Operation != admissionv1.Update {
		return nil
	}

	// Deserialize the object
	var jobFlow v1alpha1.JobFlow
	decoder := Codecs.UniversalDeserializer()

	// For CREATE and UPDATE operations, use Object
	rawObj := req.Object

	obj, _, err := decoder.Decode(rawObj.Raw, nil, &jobFlow)
	if err != nil {
		return fmt.Errorf("failed to decode JobFlow: %w", err)
	}

	jobFlowObj, ok := obj.(*v1alpha1.JobFlow)
	if !ok {
		return fmt.Errorf("%w, got %T", ErrUnexpectedObjectType, obj)
	}

	// Use comprehensive validation package
	if err := validation.ValidateJobFlow(jobFlowObj); err != nil {
		return err
	}

	return nil
}

// handleMutate handles admission review requests for JobFlow mutation (defaults).
func (ws *WebhookServer) handleMutate(w http.ResponseWriter, r *http.Request) {
	logger := logging.NewLogger("zen-flow-webhook")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read admission review request
	ctx := r.Context()
	var review admissionv1.AdmissionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		logger.WithContext(ctx).Error(err, "Failed to decode admission review")
		http.Error(w, fmt.Sprintf("Failed to decode request: %v", err), http.StatusBadRequest)
		return
	}

	// Set response UID
	response := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Response: &admissionv1.AdmissionResponse{
			UID: review.Request.UID,
		},
	}

	// Mutate the JobFlow (set defaults)
	patches, err := ws.mutateJobFlow(review.Request)
	if err != nil {
		logger.WithContext(ctx).Error(err, "JobFlow mutation failed")
		response.Response.Allowed = false
		response.Response.Result = &metav1.Status{
			Code:    http.StatusInternalServerError,
			Message: fmt.Sprintf("Failed to mutate JobFlow: %v", err),
		}
	} else {
		response.Response.Allowed = true
		if len(patches) > 0 {
			patchBytes, err := json.Marshal(patches)
			if err != nil {
				logger.WithContext(ctx).Error(err, "Failed to marshal patches")
				response.Response.Allowed = false
				response.Response.Result = &metav1.Status{
					Code:    http.StatusInternalServerError,
					Message: fmt.Sprintf("Failed to marshal patches: %v", err),
				}
			} else {
				response.Response.Patch = patchBytes
				response.Response.PatchType = func() *admissionv1.PatchType {
					pt := admissionv1.PatchTypeJSONPatch
					return &pt
				}()
				logger.WithContext(ctx).Debug("JobFlow mutation succeeded",
					logging.Int("patches", len(patches)))
			}
		} else {
			logger.WithContext(ctx).Debug("JobFlow mutation succeeded (no patches needed)")
		}
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.WithContext(ctx).Error(err, "Failed to encode admission review response")
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// mutateJobFlow mutates a JobFlow to set default values.
func (ws *WebhookServer) mutateJobFlow(req *admissionv1.AdmissionRequest) ([]map[string]interface{}, error) {
	// Only mutate CREATE operations
	if req.Operation != admissionv1.Create {
		return nil, nil
	}

	// Deserialize the object
	var jobFlow v1alpha1.JobFlow
	decoder := Codecs.UniversalDeserializer()

	obj, _, err := decoder.Decode(req.Object.Raw, nil, &jobFlow)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JobFlow: %w", err)
	}

	jobFlowObj, ok := obj.(*v1alpha1.JobFlow)
	if !ok {
		return nil, fmt.Errorf("%w, got %T", ErrUnexpectedObjectType, obj)
	}

	// Collect patches for default values
	var patches []map[string]interface{}

	// Set default execution policy if not specified
	execPolicyPath := "/spec/executionPolicy"
	hasExecPolicy := jobFlowObj.Spec.ExecutionPolicy != nil &&
		(jobFlowObj.Spec.ExecutionPolicy.ConcurrencyPolicy != "" ||
			jobFlowObj.Spec.ExecutionPolicy.TTLSecondsAfterFinished != nil ||
			jobFlowObj.Spec.ExecutionPolicy.BackoffLimit != nil ||
			jobFlowObj.Spec.ExecutionPolicy.ActiveDeadlineSeconds != nil ||
			jobFlowObj.Spec.ExecutionPolicy.PodFailurePolicy != nil)

	if !hasExecPolicy {
		// Create execution policy with defaults
		patches = append(patches, map[string]interface{}{
			"op":   "add",
			"path": execPolicyPath,
			"value": map[string]interface{}{
				"concurrencyPolicy":       "Forbid",
				"ttlSecondsAfterFinished": 86400,
				"backoffLimit":            6,
			},
		})
	} else {
		// Set individual defaults if execution policy exists but fields are missing
		if jobFlowObj.Spec.ExecutionPolicy.ConcurrencyPolicy == "" {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  execPolicyPath + "/concurrencyPolicy",
				"value": "Forbid",
			})
		}
		if jobFlowObj.Spec.ExecutionPolicy.TTLSecondsAfterFinished == nil {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  execPolicyPath + "/ttlSecondsAfterFinished",
				"value": 86400,
			})
		}
		if jobFlowObj.Spec.ExecutionPolicy.BackoffLimit == nil {
			patches = append(patches, map[string]interface{}{
				"op":    "add",
				"path":  execPolicyPath + "/backoffLimit",
				"value": 6,
			})
		}
	}

	return patches, nil
}
