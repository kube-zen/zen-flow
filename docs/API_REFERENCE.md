# API Reference

Complete API reference for the JobFlow CRD.

## JobFlow

`JobFlow` is a namespaced resource that defines a workflow of Kubernetes Jobs executed in a directed acyclic graph (DAG) order.

### API Version

- **Group**: `workflow.zen.io`
- **Version**: `v1alpha1`
- **Kind**: `JobFlow`
- **Plural**: `jobflows`
- **Short Names**: `jf`, `flow`

### Schema

```yaml
apiVersion: workflow.zen.io/v1alpha1
kind: JobFlow
metadata:
  name: string
  namespace: string
spec:
  executionPolicy: ExecutionPolicy (optional)
  resourceTemplates: ResourceTemplates (optional)
  steps: []Step
status:
  phase: string
  startTime: string (RFC3339)
  completionTime: string (RFC3339)
  steps: []StepStatus
  progress: Progress
  conditions: []Condition
```

---

## ExecutionPolicy

Defines execution behavior for the JobFlow.

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `concurrencyPolicy` | string | No | "Forbid" | How to handle concurrent executions. Values: "Allow", "Forbid", "Replace" |
| `ttlSecondsAfterFinished` | int32 | No | 86400 | TTL in seconds after JobFlow finishes (24 hours) |
| `backoffLimit` | int32 | No | 6 | Maximum number of retries for the entire flow |
| `activeDeadlineSeconds` | int64 | No | - | Maximum duration in seconds for the entire flow |
| `podFailurePolicy` | PodFailurePolicy | No | - | Policy for handling pod failures |

### Example

```yaml
executionPolicy:
  concurrencyPolicy: Forbid
  ttlSecondsAfterFinished: 3600
  backoffLimit: 3
  activeDeadlineSeconds: 7200
```

---

## Step

Defines a single step in the JobFlow.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Unique name of the step within the flow |
| `dependencies` | []string | No | List of step names that must complete before this step starts |
| `template` | runtime.RawExtension | Yes | Kubernetes Job template |
| `retryPolicy` | RetryPolicy | No | Retry behavior for this step |
| `timeoutSeconds` | int64 | No | Maximum duration in seconds for this step |
| `continueOnFailure` | bool | No | Whether flow continues if this step fails (default: false) |
| `when` | string | No | Condition that must be met for step to execute |
| `inputs` | StepInputs | No | Input artifacts and parameters |
| `outputs` | StepOutputs | No | Output artifacts and parameters |
| `metadata` | StepMetadata | No | Metadata (annotations, labels) for the step's job |

### Example

```yaml
steps:
  - name: extract-data
    template:
      apiVersion: batch/v1
      kind: Job
      spec:
        template:
          spec:
            containers:
              - name: main
                image: busybox:latest
                command: ["sh", "-c"]
                args: ["echo 'Extracting data'"]
            restartPolicy: OnFailure
  - name: transform-data
    dependencies: ["extract-data"]
    continueOnFailure: false
    timeoutSeconds: 3600
    template:
      apiVersion: batch/v1
      kind: Job
      spec:
        template:
          spec:
            containers:
              - name: main
                image: python:3.9
                command: ["python", "transform.py"]
            restartPolicy: OnFailure
```

---

## RetryPolicy

Defines retry behavior for a step.

### Fields

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `limit` | int32 | No | 3 | Maximum number of retries |
| `backoff` | BackoffPolicy | No | - | Backoff strategy |

### Example

```yaml
retryPolicy:
  limit: 5
  backoff:
    type: Exponential
    factor: 2.0
    duration: "10s"
```

---

## BackoffPolicy

Defines backoff strategy for retries.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Backoff type: "Exponential", "Linear", "Fixed" |
| `factor` | float64 | No | Backoff factor (for Exponential) |
| `duration` | string | Yes | Base duration (e.g., "10s", "1m") |

### Example

```yaml
backoff:
  type: Exponential
  factor: 2.0
  duration: "10s"
```

---

## ResourceTemplates

Defines templates for resources created for the flow.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `volumeClaimTemplates` | []PersistentVolumeClaim | No | PVC templates to create |
| `configMapTemplates` | []ConfigMap | No | ConfigMap templates to create |

### Example

```yaml
resourceTemplates:
  volumeClaimTemplates:
    - metadata:
        name: data-volume
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 10Gi
  configMapTemplates:
    - metadata:
        name: config
      data:
        key: value
```

---

## StepInputs

Defines inputs for a step.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `artifacts` | []ArtifactInput | No | Artifact inputs |
| `parameters` | []ParameterInput | No | Parameter inputs |

---

## StepOutputs

Defines outputs from a step.

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `artifacts` | []ArtifactOutput | No | Artifact outputs |
| `parameters` | []ParameterOutput | No | Parameter outputs |

---

## JobFlowStatus

Status of the JobFlow.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current phase: "Pending", "Running", "Succeeded", "Failed", "Suspended" |
| `startTime` | string | When the JobFlow started (RFC3339) |
| `completionTime` | string | When the JobFlow completed (RFC3339) |
| `steps` | []StepStatus | Status of each step |
| `progress` | Progress | Progress information |
| `conditions` | []Condition | Conditions describing the JobFlow state |

### Phase Values

- **Pending**: JobFlow is waiting to start
- **Running**: JobFlow is executing steps
- **Succeeded**: All steps completed successfully
- **Failed**: One or more steps failed and flow stopped
- **Suspended**: JobFlow is suspended

---

## StepStatus

Status of a step.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Step name |
| `phase` | string | Step phase: "Pending", "Running", "Succeeded", "Failed", "Skipped" |
| `startTime` | string | When the step started (RFC3339) |
| `completionTime` | string | When the step completed (RFC3339) |
| `jobRef` | JobReference | Reference to the created Job |
| `retryCount` | int32 | Number of retries attempted |
| `message` | string | Human-readable message |

---

## Progress

Progress information for the JobFlow.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `completedSteps` | int32 | Number of completed steps |
| `totalSteps` | int32 | Total number of steps |
| `successfulSteps` | int32 | Number of successful steps |
| `failedSteps` | int32 | Number of failed steps |

---

## Complete Example

```yaml
apiVersion: workflow.zen.io/v1alpha1
kind: JobFlow
metadata:
  name: data-pipeline
  namespace: default
spec:
  executionPolicy:
    concurrencyPolicy: Forbid
    ttlSecondsAfterFinished: 3600
    backoffLimit: 3
  resourceTemplates:
    volumeClaimTemplates:
      - metadata:
          name: data-volume
        spec:
          accessModes: ["ReadWriteOnce"]
          resources:
            requests:
              storage: 10Gi
  steps:
    - name: extract
      template:
        apiVersion: batch/v1
        kind: Job
        spec:
          template:
            spec:
              containers:
                - name: main
                  image: extractor:latest
                  volumeMounts:
                    - name: data
                      mountPath: /data
              volumes:
                - name: data
                  persistentVolumeClaim:
                    claimName: data-volume
              restartPolicy: OnFailure
    - name: transform
      dependencies: ["extract"]
      retryPolicy:
        limit: 3
        backoff:
          type: Exponential
          duration: "10s"
      template:
        apiVersion: batch/v1
        kind: Job
        spec:
          template:
            spec:
              containers:
                - name: main
                  image: transformer:latest
                  volumeMounts:
                    - name: data
                      mountPath: /data
              volumes:
                - name: data
                  persistentVolumeClaim:
                    claimName: data-volume
              restartPolicy: OnFailure
    - name: load
      dependencies: ["transform"]
      template:
        apiVersion: batch/v1
        kind: Job
        spec:
          template:
            spec:
              containers:
                - name: main
                  image: loader:latest
              restartPolicy: OnFailure
```

---

## Validation Rules

### Step Names
- Must be unique within a JobFlow
- Must be non-empty
- Must be valid Kubernetes resource names

### Dependencies
- Must reference existing step names
- Cannot create cycles (validated by DAG engine)
- Self-dependencies are not allowed

### Templates
- Must be valid Kubernetes Job specs
- Must have at least one container
- Must specify `restartPolicy`

### Execution Policy
- `concurrencyPolicy` must be one of: "Allow", "Forbid", "Replace"
- `ttlSecondsAfterFinished` must be non-negative
- `backoffLimit` must be non-negative
- `activeDeadlineSeconds` must be positive if specified

---

## See Also

- [User Guide](USER_GUIDE.md) - How to use JobFlow
- [Architecture](ARCHITECTURE.md) - How JobFlow works internally
- [Examples](../examples/) - Example JobFlow manifests

