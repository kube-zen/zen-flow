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

package v1alpha1

import (
	"encoding/json"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=jf;flow,categories=all
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Progress",type="string",JSONPath=".status.progress.completedSteps"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// JobFlow is the Schema for the jobflows API.
// JobFlow provides declarative, sequential execution of Kubernetes Jobs using standard CRDs.
type JobFlow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobFlowSpec   `json:"spec,omitempty"`
	Status JobFlowStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// JobFlowList contains a list of JobFlow.
type JobFlowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []JobFlow `json:"items"`
}

// JobFlowSpec defines the desired state of JobFlow.
type JobFlowSpec struct {
	// ExecutionPolicy defines how the JobFlow should be executed.
	ExecutionPolicy *ExecutionPolicy `json:"executionPolicy,omitempty"`

	// ResourceTemplates defines templates for resources that will be created for the flow.
	ResourceTemplates *ResourceTemplates `json:"resourceTemplates,omitempty"`

	// Steps defines the sequence of jobs to execute.
	// Steps are executed in topological order based on dependencies.
	Steps []Step `json:"steps"`
}

// ExecutionPolicy defines execution behavior for the JobFlow.
type ExecutionPolicy struct {
	// ConcurrencyPolicy specifies how to treat concurrent executions of the same JobFlow.
	// Valid values are: "Allow", "Forbid", "Replace".
	// Defaults to "Forbid".
	// +kubebuilder:validation:Enum=Allow;Forbid;Replace
	// +kubebuilder:default=Forbid
	ConcurrencyPolicy string `json:"concurrencyPolicy,omitempty"`

	// TTLSecondsAfterFinished specifies the TTL for the JobFlow after it finishes.
	// After this time, the JobFlow and its resources will be cleaned up.
	// Defaults to 86400 (24 hours).
	// +kubebuilder:default=86400
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// PodFailurePolicy defines how to handle pod failures.
	PodFailurePolicy *PodFailurePolicy `json:"podFailurePolicy,omitempty"`

	// BackoffLimit specifies the maximum number of retries for the entire flow.
	// Defaults to 6.
	// +kubebuilder:default=6
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`

	// ActiveDeadlineSeconds specifies the maximum duration in seconds for the entire flow.
	// If the flow runs longer than this, it will be terminated.
	ActiveDeadlineSeconds *int64 `json:"activeDeadlineSeconds,omitempty"`
}

// PodFailurePolicy defines how to handle pod failures.
type PodFailurePolicy struct {
	// Rules define failure handling rules.
	Rules []PodFailurePolicyRule `json:"rules,omitempty"`
}

// PodFailurePolicyRule defines a rule for handling pod failures.
type PodFailurePolicyRule struct {
	// Action specifies what action to take when the rule matches.
	// Valid values are: "FailJob", "Ignore", "Count".
	// +kubebuilder:validation:Enum=FailJob;Ignore;Count
	Action string `json:"action"`

	// OnExitCodes specifies exit codes that trigger this rule.
	OnExitCodes *PodFailurePolicyOnExitCodes `json:"onExitCodes,omitempty"`
}

// PodFailurePolicyOnExitCodes defines exit code matching.
type PodFailurePolicyOnExitCodes struct {
	// ContainerName specifies the container name (defaults to "main").
	ContainerName string `json:"containerName,omitempty"`

	// Operator specifies the operator for matching exit codes.
	// Valid values are: "In", "NotIn".
	// +kubebuilder:validation:Enum=In;NotIn
	Operator string `json:"operator"`

	// Values specifies the exit code values to match.
	Values []int32 `json:"values"`
}

// ResourceTemplates defines templates for resources created for the flow.
type ResourceTemplates struct {
	// VolumeClaimTemplates defines PVC templates.
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates,omitempty"`

	// ConfigMapTemplates defines ConfigMap templates.
	ConfigMapTemplates []corev1.ConfigMap `json:"configMapTemplates,omitempty"`
}

// Step defines a single step in the JobFlow.
type Step struct {
	// Name is the unique name of the step within the flow.
	Name string `json:"name"`

	// Metadata contains metadata for the step.
	Metadata *StepMetadata `json:"metadata,omitempty"`

	// Dependencies specifies which steps must complete before this step can start.
	// Empty list means the step can start immediately.
	Dependencies []string `json:"dependencies,omitempty"`

	// RetryPolicy defines retry behavior for this step.
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`

	// TimeoutSeconds specifies the maximum duration in seconds for this step.
	TimeoutSeconds *int64 `json:"timeoutSeconds,omitempty"`

	// When specifies a condition that must be met for this step to execute.
	// Uses template syntax to evaluate conditions.
	When string `json:"when,omitempty"`

	// ContinueOnFailure specifies whether the flow should continue if this step fails.
	// Defaults to false.
	ContinueOnFailure bool `json:"continueOnFailure,omitempty"`

	// Template defines the Kubernetes Job template for this step.
	Template runtime.RawExtension `json:"template"`

	// Inputs defines inputs for this step (artifacts, parameters).
	Inputs *StepInputs `json:"inputs,omitempty"`

	// Outputs defines outputs from this step (artifacts, parameters).
	Outputs *StepOutputs `json:"outputs,omitempty"`
}

// StepMetadata contains metadata for a step.
type StepMetadata struct {
	// Annotations are annotations to apply to the step's job.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Labels are labels to apply to the step's job.
	Labels map[string]string `json:"labels,omitempty"`
}

// RetryPolicy defines retry behavior.
type RetryPolicy struct {
	// Limit specifies the maximum number of retries.
	// Defaults to 3.
	Limit int32 `json:"limit,omitempty"`

	// Backoff defines the backoff strategy.
	Backoff *BackoffPolicy `json:"backoff,omitempty"`
}

// BackoffPolicy defines backoff strategy.
type BackoffPolicy struct {
	// Type specifies the backoff type.
	// Valid values are: "Exponential", "Linear", "Fixed".
	// +kubebuilder:validation:Enum=Exponential;Linear;Fixed
	Type string `json:"type"`

	// Factor is the backoff factor (for Exponential).
	Factor *float64 `json:"factor,omitempty"`

	// Duration specifies the base duration for backoff.
	Duration string `json:"duration"`
}

// StepInputs defines inputs for a step.
type StepInputs struct {
	// Artifacts defines artifact inputs.
	Artifacts []ArtifactInput `json:"artifacts,omitempty"`

	// Parameters defines parameter inputs.
	Parameters []ParameterInput `json:"parameters,omitempty"`
}

// StepOutputs defines outputs from a step.
type StepOutputs struct {
	// Artifacts defines artifact outputs.
	Artifacts []ArtifactOutput `json:"artifacts,omitempty"`

	// Parameters defines parameter outputs.
	Parameters []ParameterOutput `json:"parameters,omitempty"`
}

// ArtifactInput defines an artifact input.
type ArtifactInput struct {
	// Name is the name of the artifact.
	Name string `json:"name"`

	// From specifies the source step and artifact.
	From string `json:"from,omitempty"`

	// Path specifies the path where the artifact should be placed.
	Path string `json:"path,omitempty"`

	// HTTP specifies HTTP source for the artifact.
	HTTP *HTTPArtifact `json:"http,omitempty"`
}

// ArtifactOutput defines an artifact output.
type ArtifactOutput struct {
	// Name is the name of the artifact.
	Name string `json:"name"`

	// Path specifies the path to the artifact.
	Path string `json:"path"`

	// Optional specifies whether the artifact is optional.
	Optional bool `json:"optional,omitempty"`

	// Archive specifies archive configuration.
	Archive *ArchiveConfig `json:"archive,omitempty"`

	// S3 specifies S3 storage configuration.
	S3 *S3Config `json:"s3,omitempty"`
}

// HTTPArtifact defines HTTP artifact source.
type HTTPArtifact struct {
	// URL is the HTTP URL to fetch the artifact from.
	URL string `json:"url"`

	// Headers are HTTP headers to include in the request.
	Headers map[string]string `json:"headers,omitempty"`
}

// ArchiveConfig defines archive configuration.
type ArchiveConfig struct {
	// Format specifies the archive format (tar, zip).
	// +kubebuilder:validation:Enum=tar;zip
	Format string `json:"format,omitempty"`

	// Compression specifies compression type (gzip, none).
	// +kubebuilder:validation:Enum=gzip;none
	Compression string `json:"compression,omitempty"`
}

// S3Config defines S3 storage configuration.
type S3Config struct {
	// Endpoint is the S3 endpoint.
	Endpoint string `json:"endpoint"`

	// Bucket is the S3 bucket name.
	Bucket string `json:"bucket"`

	// Key is the S3 object key.
	Key string `json:"key"`

	// AccessKeyIDSecretRef references a secret containing the access key ID.
	AccessKeyIDSecretRef *corev1.SecretKeySelector `json:"accessKeyIDSecretRef,omitempty"`

	// SecretAccessKeySecretRef references a secret containing the secret access key.
	SecretAccessKeySecretRef *corev1.SecretKeySelector `json:"secretAccessKeySecretRef,omitempty"`
}

// ParameterInput defines a parameter input.
type ParameterInput struct {
	// Name is the name of the parameter.
	Name string `json:"name"`

	// Value is the parameter value.
	Value string `json:"value,omitempty"`

	// ValueFrom specifies where to get the value from.
	ValueFrom *ParameterValueFrom `json:"valueFrom,omitempty"`
}

// ParameterOutput defines a parameter output.
type ParameterOutput struct {
	// Name is the name of the parameter.
	Name string `json:"name"`

	// ValueFrom specifies where to get the value from.
	ValueFrom ParameterValueFrom `json:"valueFrom"`
}

// ParameterValueFrom specifies where to get a parameter value.
type ParameterValueFrom struct {
	// JSONPath specifies a JSONPath expression to extract the value.
	JSONPath string `json:"jsonPath,omitempty"`

	// ConfigMapKeyRef references a ConfigMap key.
	ConfigMapKeyRef *corev1.ConfigMapKeySelector `json:"configMapKeyRef,omitempty"`

	// SecretKeyRef references a Secret key.
	SecretKeyRef *corev1.SecretKeySelector `json:"secretKeyRef,omitempty"`
}

// JobFlowStatus defines the observed state of JobFlow.
type JobFlowStatus struct {
	// Phase represents the current phase of the JobFlow.
	// Valid values are: "Pending", "Running", "Succeeded", "Failed", "Suspended".
	// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Suspended
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations of the JobFlow's state.
	Conditions []JobFlowCondition `json:"conditions,omitempty"`

	// StartTime is the time when the JobFlow started.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time when the JobFlow completed.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// EstimatedCompletionTime is the estimated time when the JobFlow will complete.
	EstimatedCompletionTime *metav1.Time `json:"estimatedCompletionTime,omitempty"`

	// Steps contains the status of each step.
	Steps []StepStatus `json:"steps,omitempty"`

	// ResourceRefs contains references to resources created by the flow.
	ResourceRefs []corev1.ObjectReference `json:"resourceRefs,omitempty"`

	// Progress contains progress information.
	Progress *ProgressStatus `json:"progress,omitempty"`

	// Audit contains audit information.
	Audit *AuditStatus `json:"audit,omitempty"`
}

// JobFlowCondition represents a condition of the JobFlow.
type JobFlowCondition struct {
	// Type is the type of condition.
	Type string `json:"type"`

	// Status is the status of the condition.
	// Valid values are: "True", "False", "Unknown".
	Status corev1.ConditionStatus `json:"status"`

	// LastTransitionTime is the last time the condition transitioned.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`

	// Reason is a brief reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`

	// Message is a human-readable message about the condition.
	Message string `json:"message,omitempty"`
}

// StepStatus represents the status of a step.
type StepStatus struct {
	// Name is the name of the step.
	Name string `json:"name"`

	// Phase represents the current phase of the step.
	// Valid values are: "Pending", "Running", "Succeeded", "Failed", "Skipped".
	Phase string `json:"phase,omitempty"`

	// StartTime is the time when the step started.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time when the step completed.
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// JobRef is a reference to the Kubernetes Job created for this step.
	JobRef *corev1.ObjectReference `json:"jobRef,omitempty"`

	// Outputs contains the outputs from this step.
	Outputs *StepOutputs `json:"outputs,omitempty"`

	// RetryCount is the number of times this step has been retried.
	RetryCount int32 `json:"retryCount,omitempty"`

	// Message contains a human-readable message about the step's state.
	Message string `json:"message,omitempty"`
}

// ProgressStatus contains progress information.
type ProgressStatus struct {
	// CompletedSteps is the number of completed steps.
	CompletedSteps int32 `json:"completedSteps,omitempty"`

	// TotalSteps is the total number of steps.
	TotalSteps int32 `json:"totalSteps,omitempty"`

	// SuccessfulSteps is the number of successful steps.
	SuccessfulSteps int32 `json:"successfulSteps,omitempty"`

	// FailedSteps is the number of failed steps.
	FailedSteps int32 `json:"failedSteps,omitempty"`
}

// AuditStatus contains audit information.
type AuditStatus struct {
	// InitiatedBy contains information about who initiated the flow.
	InitiatedBy *InitiatedBy `json:"initiatedBy,omitempty"`

	// CorrelationID is a correlation ID for tracking the flow.
	CorrelationID string `json:"correlationID,omitempty"`
}

// InitiatedBy contains information about who initiated the flow.
type InitiatedBy struct {
	// User is the user who initiated the flow.
	User string `json:"user,omitempty"`

	// Type is the type of initiator (User, ServiceAccount, etc.).
	Type string `json:"type,omitempty"`
}

// Step phase constants
const (
	StepPhasePending  = "Pending"
	StepPhaseRunning  = "Running"
	StepPhaseSucceeded = "Succeeded"
	StepPhaseFailed   = "Failed"
	StepPhaseSkipped  = "Skipped"
)

// JobFlow phase constants
const (
	JobFlowPhasePending   = "Pending"
	JobFlowPhaseRunning    = "Running"
	JobFlowPhaseSucceeded  = "Succeeded"
	JobFlowPhaseFailed     = "Failed"
	JobFlowPhaseSuspended  = "Suspended"
)

// GetJobTemplate extracts the Job template from a Step's Template field.
func (s *Step) GetJobTemplate() (*batchv1.Job, error) {
	// Convert RawExtension to unstructured first
	unstructuredObj := &unstructured.Unstructured{}
	if err := json.Unmarshal(s.Template.Raw, unstructuredObj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}

	// Convert unstructured to Job
	job := &batchv1.Job{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, job); err != nil {
		return nil, fmt.Errorf("failed to convert unstructured to Job: %w", err)
	}
	return job, nil
}

