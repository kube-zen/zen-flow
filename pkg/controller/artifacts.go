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
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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

	logger.Info("Fetching artifact from step",
		sdklog.String("source_step", fromStep),
		sdklog.String("artifact_name", artifactName),
		sdklog.String("source_path", artifactPath),
		sdklog.String("target_path", targetPath))

	// Artifact copying strategy:
	// 1. For PVC-based artifacts: The controller ensures PVCs are created via resourceTemplates.
	//    Users must mount the PVC in their job templates. File copying happens in job containers.
	// 2. For ConfigMap-based artifacts: The controller can copy data from ConfigMaps.
	// 3. For S3 artifacts: Use fetchArtifactFromS3 (if implemented) or HTTP.

	// Check if artifact is stored in a ConfigMap (by convention: <jobflow-name>-<step-name>-<artifact-name>)
	configMapName := fmt.Sprintf("%s-%s-%s", jobFlow.Name, fromStep, artifactName)
	configMapKey := types.NamespacedName{Namespace: jobFlow.Namespace, Name: configMapName}

	configMap := &corev1.ConfigMap{}
	if err := r.Get(ctx, configMapKey, configMap); err == nil {
		// Artifact found in ConfigMap - copy to target path
		return r.copyArtifactFromConfigMap(ctx, configMap, artifactName, targetPath)
	} else if !k8serrors.IsNotFound(err) {
		return jferrors.Wrapf(err, "configmap_get_failed", "failed to check ConfigMap for artifact")
	}

	// Artifact not in ConfigMap - assume it's in a shared PVC
	// The controller can't directly access PVC files, so we document that:
	// - Users should mount shared PVCs in their job templates
	// - File copying should happen in job containers using init containers or sidecars
	// - The controller ensures PVCs are created and available

	logger.Info("Artifact not found in ConfigMap, assuming PVC-based storage",
		sdklog.String("source_step", fromStep),
		sdklog.String("artifact_name", artifactName),
		sdklog.String("note", "Ensure shared PVC is mounted in job template for file access"))

	// For PVC-based artifacts, we can't copy files directly, but we can:
	// 1. Verify the PVC exists
	// 2. Document that users should mount it in their job templates
	// 3. The actual file copying happens in the job container

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
	accessKeyID, err := r.getSecretValue(ctx, jobFlow.Namespace, s3Config.AccessKeyIDSecretRef)
	if err != nil {
		return jferrors.Wrapf(err, "secret_get_failed", "failed to get access key ID from secret")
	}

	secretAccessKey, err := r.getSecretValue(ctx, jobFlow.Namespace, s3Config.SecretAccessKeySecretRef)
	if err != nil {
		return jferrors.Wrapf(err, "secret_get_failed", "failed to get secret access key from secret")
	}

	// Initialize MinIO client (S3-compatible)
	// Use SSL if endpoint uses https
	useSSL := strings.HasPrefix(s3Config.Endpoint, "https://")
	endpoint := strings.TrimPrefix(strings.TrimPrefix(s3Config.Endpoint, "https://"), "http://")

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return jferrors.Wrapf(err, "s3_client_init_failed", "failed to initialize S3 client")
	}

	// Check if bucket exists, create if not
	exists, err := minioClient.BucketExists(ctx, s3Config.Bucket)
	if err != nil {
		return jferrors.Wrapf(err, "s3_bucket_check_failed", "failed to check if bucket exists")
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, s3Config.Bucket, minio.MakeBucketOptions{}); err != nil {
			return jferrors.Wrapf(err, "s3_bucket_create_failed", "failed to create bucket %s", s3Config.Bucket)
		}
		logger.Info("Created S3 bucket", sdklog.String("bucket", s3Config.Bucket))
	}

	// Open artifact file
	file, err := os.Open(artifactPath)
	if err != nil {
		return jferrors.Wrapf(err, "file_open_failed", "failed to open artifact file %s", artifactPath)
	}
	defer file.Close()

	// Get file info for content type
	fileInfo, err := file.Stat()
	if err != nil {
		return jferrors.Wrapf(err, "file_stat_failed", "failed to get file info for %s", artifactPath)
	}

	// Upload file to S3
	uploadInfo, err := minioClient.PutObject(ctx, s3Config.Bucket, s3Config.Key, file, fileInfo.Size(), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return jferrors.Wrapf(err, "s3_upload_failed", "failed to upload artifact to S3")
	}

	logger.Info("Successfully uploaded artifact to S3",
		sdklog.String("bucket", s3Config.Bucket),
		sdklog.String("key", s3Config.Key),
		sdklog.String("etag", uploadInfo.ETag),
		sdklog.Int64("size", uploadInfo.Size))

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

