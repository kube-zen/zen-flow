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
	"os"
	"path/filepath"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	jferrors "github.com/kube-zen/zen-flow/pkg/errors"
	sdklog "github.com/kube-zen/zen-sdk/pkg/logging"
)

// copyArtifactFromConfigMap copies an artifact from a ConfigMap to a target path
func (r *JobFlowReconciler) copyArtifactFromConfigMap(ctx context.Context, configMap *corev1.ConfigMap, artifactName, targetPath string) error {
	logger := sdklog.NewLogger("zen-flow-controller")

	// Find artifact data in ConfigMap
	// ConfigMap keys might be: artifactName, artifactName.data, or artifactName.binary
	artifactData, found := configMap.Data[artifactName]
	if !found {
		// Try binary data
		if binaryData, found := configMap.BinaryData[artifactName]; found {
			return r.writeBinaryArtifact(binaryData, targetPath)
		}
		// Try with .data suffix
		if artifactData, found = configMap.Data[artifactName+".data"]; !found {
			return jferrors.New("artifact_not_in_configmap", fmt.Sprintf("artifact %s not found in ConfigMap %s", artifactName, configMap.Name))
		}
	}

	// Create target directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0750); err != nil {
		return jferrors.Wrapf(err, "mkdir_failed", "failed to create directory for %s", targetPath)
	}

	// Write artifact data to target path
	if err := os.WriteFile(targetPath, []byte(artifactData), DefaultFilePerm); err != nil {
		return jferrors.Wrapf(err, "file_write_failed", "failed to write artifact to %s", targetPath)
	}

	logger.Info("Copied artifact from ConfigMap",
		sdklog.String("artifact_name", artifactName),
		sdklog.String("configmap", configMap.Name),
		sdklog.String("target_path", targetPath))

	return nil
}

// writeBinaryArtifact writes binary artifact data to a file
func (r *JobFlowReconciler) writeBinaryArtifact(data []byte, targetPath string) error {
	// Create target directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0750); err != nil {
		return jferrors.Wrapf(err, "mkdir_failed", "failed to create directory for %s", targetPath)
	}

	// Write binary data to target path
	if err := os.WriteFile(targetPath, data, DefaultFilePerm); err != nil {
		return jferrors.Wrapf(err, "file_write_failed", "failed to write binary artifact to %s", targetPath)
	}

	return nil
}

// storeArtifactInConfigMap stores an artifact in a ConfigMap for sharing between steps
func (r *JobFlowReconciler) storeArtifactInConfigMap(ctx context.Context, jobFlow *v1alpha1.JobFlow, stepName, artifactName, artifactPath string) error {
	logger := sdklog.NewLogger("zen-flow-controller")

	// Read artifact file
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		return jferrors.Wrapf(err, "file_read_failed", "failed to read artifact file %s", artifactPath)
	}

	// Check file size - ConfigMaps have a 1MB limit
	if len(artifactData) > ConfigMapSizeLimit {
		logger.Warn("Artifact too large for ConfigMap, skipping ConfigMap storage",
			sdklog.String("artifact_name", artifactName),
			sdklog.String("size", fmt.Sprintf("%d bytes", len(artifactData))),
			sdklog.String("note", "Use PVC or S3 for large artifacts"))
		return nil // Don't fail, just skip ConfigMap storage
	}

	// Create or update ConfigMap
	configMapName := fmt.Sprintf("%s-%s-%s", jobFlow.Name, stepName, artifactName)
	configMapKey := types.NamespacedName{Namespace: jobFlow.Namespace, Name: configMapName}

	configMap := &corev1.ConfigMap{}
	err = r.Get(ctx, configMapKey, configMap)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			// Error other than not found
			return jferrors.Wrapf(err, "configmap_get_failed", "failed to get ConfigMap %s", configMapName)
		}
		// ConfigMap doesn't exist, create it
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configMapName,
				Namespace: jobFlow.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(jobFlow, v1alpha1.SchemeGroupVersion.WithKind("JobFlow")),
				},
			},
			Data: map[string]string{
				artifactName: string(artifactData),
			},
		}
		if err := r.Create(ctx, configMap); err != nil {
			return jferrors.Wrapf(err, "configmap_create_failed", "failed to create ConfigMap for artifact")
		}
	} else {
		// ConfigMap exists, update it
		if configMap.Data == nil {
			configMap.Data = make(map[string]string)
		}
		configMap.Data[artifactName] = string(artifactData)
		if err := r.Update(ctx, configMap); err != nil {
			return jferrors.Wrapf(err, "configmap_update_failed", "failed to update ConfigMap for artifact")
		}
	}

	logger.Info("Stored artifact in ConfigMap",
		sdklog.String("artifact_name", artifactName),
		sdklog.String("configmap", configMapName),
		sdklog.Int("size", len(artifactData)))

	return nil
}

// ensureArtifactVolumeMount ensures that a shared PVC is mounted in a job for artifact access
// This is a helper function that can be called to add volume mounts for shared PVCs
func (r *JobFlowReconciler) ensureArtifactVolumeMount(job *batchv1.Job, pvcName, mountPath string) {
	// Check if volume already exists
	volumeName := fmt.Sprintf("artifact-%s", pvcName)
	for _, vol := range job.Spec.Template.Spec.Volumes {
		if vol.Name == volumeName {
			return // Volume already mounted
		}
	}

	// Add volume
	if job.Spec.Template.Spec.Volumes == nil {
		job.Spec.Template.Spec.Volumes = []corev1.Volume{}
	}
	job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		},
	})

	// Add volume mount to all containers
	for i := range job.Spec.Template.Spec.Containers {
		if job.Spec.Template.Spec.Containers[i].VolumeMounts == nil {
			job.Spec.Template.Spec.Containers[i].VolumeMounts = []corev1.VolumeMount{}
		}
		job.Spec.Template.Spec.Containers[i].VolumeMounts = append(
			job.Spec.Template.Spec.Containers[i].VolumeMounts,
			corev1.VolumeMount{
				Name:      volumeName,
				MountPath: mountPath,
			},
		)
	}
}

