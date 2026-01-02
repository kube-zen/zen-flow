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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	jferrors "github.com/kube-zen/zen-flow/pkg/errors"
	sdklog "github.com/kube-zen/zen-sdk/pkg/logging"
)

// fetchArtifactFromStep fetches an artifact from a previous step
func (r *JobFlowReconciler) fetchArtifactFromStep(ctx context.Context, jobFlow *v1alpha1.JobFlow, fromStep, artifactName, targetPath string) error {
	logger := sdklog.NewLogger("zen-flow-controller")

	// Find the source step status
	sourceStepStatus := r.getStepStatus(jobFlow.Status, fromStep)
	if sourceStepStatus == nil {
		return jferrors.WithStep(jferrors.WithJobFlow(
			jferrors.New("step_not_found", fmt.Sprintf("source step %s not found", fromStep)),
			jobFlow.Namespace, jobFlow.Name), fromStep)
	}

	// Check if source step has completed
	if sourceStepStatus.Phase != v1alpha1.StepPhaseSucceeded {
		return jferrors.WithStep(jferrors.WithJobFlow(
			jferrors.New("step_not_completed", fmt.Sprintf("source step %s has not completed", fromStep)),
			jobFlow.Namespace, jobFlow.Name), fromStep)
	}

	// Find artifact in source step outputs
	if sourceStepStatus.Outputs == nil {
		return jferrors.WithStep(jferrors.WithJobFlow(
			jferrors.New("artifact_not_found", fmt.Sprintf("artifact %s not found in step %s outputs", artifactName, fromStep)),
			jobFlow.Namespace, jobFlow.Name), fromStep)
	}

	var artifactPath string
	for _, artifact := range sourceStepStatus.Outputs.Artifacts {
		if artifact.Name == artifactName {
			artifactPath = artifact.Path
			break
		}
	}

	if artifactPath == "" {
		return jferrors.WithStep(jferrors.WithJobFlow(
			jferrors.New("artifact_not_found", fmt.Sprintf("artifact %s not found in step %s", artifactName, fromStep)),
			jobFlow.Namespace, jobFlow.Name), fromStep)
	}

	// In a real implementation, artifacts would be stored in a shared volume or ConfigMap
	// For now, we'll use a placeholder that logs the operation
	logger.Info("Fetching artifact from step",
		sdklog.String("source_step", fromStep),
		sdklog.String("artifact_name", artifactName),
		sdklog.String("source_path", artifactPath),
		sdklog.String("target_path", targetPath))

	// TODO: Implement actual artifact copying from shared storage
	// This would typically involve:
	// 1. Reading from shared volume (e.g., PVC) or ConfigMap
	// 2. Copying to target path in current step's container
	// 3. Handling different artifact types (files, directories, archives)

	return nil
}

// fetchArtifactFromHTTP fetches an artifact from an HTTP source
func (r *JobFlowReconciler) fetchArtifactFromHTTP(ctx context.Context, httpArtifact *v1alpha1.HTTPArtifact, targetPath string) error {
	logger := sdklog.NewLogger("zen-flow-controller")

	logger.Info("Fetching artifact from HTTP",
		sdklog.String("url", httpArtifact.URL),
		sdklog.String("target_path", targetPath))

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", httpArtifact.URL, nil)
	if err != nil {
		return jferrors.Wrapf(err, "http_request_failed", "failed to create HTTP request for %s", httpArtifact.URL)
	}

	// Add headers
	for k, v := range httpArtifact.Headers {
		req.Header.Set(k, v)
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return jferrors.Wrapf(err, "http_fetch_failed", "failed to fetch artifact from %s", httpArtifact.URL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return jferrors.New("http_error", fmt.Sprintf("HTTP request failed with status %d", resp.StatusCode))
	}

	// Create target directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return jferrors.Wrapf(err, "mkdir_failed", "failed to create directory for %s", targetPath)
	}

	// Create target file
	outFile, err := os.Create(targetPath)
	if err != nil {
		return jferrors.Wrapf(err, "file_create_failed", "failed to create file %s", targetPath)
	}
	defer outFile.Close()

	// Copy response body to file
	if _, err := io.Copy(outFile, resp.Body); err != nil {
		return jferrors.Wrapf(err, "file_write_failed", "failed to write artifact to %s", targetPath)
	}

	logger.Info("Successfully fetched artifact from HTTP", sdklog.String("target_path", targetPath))
	return nil
}

// uploadArtifactToS3 uploads an artifact to S3 storage
func (r *JobFlowReconciler) uploadArtifactToS3(ctx context.Context, jobFlow *v1alpha1.JobFlow, artifactPath string, s3Config *v1alpha1.S3Config) error {
	logger := sdklog.NewLogger("zen-flow-controller")

	logger.Info("Uploading artifact to S3",
		sdklog.String("artifact_path", artifactPath),
		sdklog.String("bucket", s3Config.Bucket),
		sdklog.String("key", s3Config.Key))

	// Get S3 credentials from secrets
	_, err := r.getSecretValue(ctx, jobFlow.Namespace, s3Config.AccessKeyIDSecretRef)
	if err != nil {
		return jferrors.Wrapf(err, "secret_get_failed", "failed to get access key ID from secret")
	}

	_, err = r.getSecretValue(ctx, jobFlow.Namespace, s3Config.SecretAccessKeySecretRef)
	if err != nil {
		return jferrors.Wrapf(err, "secret_get_failed", "failed to get secret access key from secret")
	}

	// TODO: Implement actual S3 upload
	// This would typically use aws-sdk-go or minio client:
	// 1. Initialize S3 client with credentials
	// 2. Read artifact file
	// 3. Upload to S3 bucket with specified key
	// 4. Handle errors and retries

	logger.Info("S3 upload placeholder - actual implementation requires S3 client library",
		sdklog.String("endpoint", s3Config.Endpoint),
		sdklog.String("bucket", s3Config.Bucket),
		sdklog.String("key", s3Config.Key))

	return nil
}

// getSecretValue retrieves a value from a Kubernetes Secret
func (r *JobFlowReconciler) getSecretValue(ctx context.Context, namespace string, secretRef *corev1.SecretKeySelector) (string, error) {
	if secretRef == nil {
		return "", jferrors.New("secret_ref_nil", "secret reference is nil")
	}

	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{Namespace: namespace, Name: secretRef.Name}
	if err := r.Get(ctx, secretKey, secret); err != nil {
		if k8serrors.IsNotFound(err) {
			return "", jferrors.Wrapf(err, "secret_not_found", "secret %s not found", secretRef.Name)
		}
		return "", jferrors.Wrapf(err, "secret_get_failed", "failed to get secret %s", secretRef.Name)
	}

	key := secretRef.Key
	if key == "" {
		key = "value" // Default key name
	}

	value, exists := secret.Data[key]
	if !exists {
		return "", jferrors.New("secret_key_not_found", fmt.Sprintf("key %s not found in secret %s", key, secretRef.Name))
	}

	return string(value), nil
}

