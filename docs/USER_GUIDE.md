# User Guide

zen-flow makes it easy to orchestrate Kubernetes Jobs in a declarative way with DAG support. This guide shows you how to get started and create effective workflows.

## Why Use zen-flow?

- **Declarative**: Define workflows as Kubernetes resources
- **DAG Support**: Execute jobs with complex dependencies
- **Zero Dependencies**: No external databases or message queues
- **Kubernetes-Native**: Uses standard Kubernetes APIs
- **Production-Ready**: Built-in observability and error handling

## Table of Contents

- [Quick Start](#quick-start)
- [Creating JobFlows](#creating-jobflows)
- [Step Dependencies](#step-dependencies)
- [Execution Policies](#execution-policies)
- [Resource Templates](#resource-templates)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

### 1. Install zen-flow

```bash
# Install CRD
kubectl apply -f deploy/crds/

# Install controller
kubectl apply -f deploy/manifests/
```

### 2. Create Your First JobFlow

```yaml
apiVersion: workflow.zen.io/v1alpha1
kind: JobFlow
metadata:
  name: simple-pipeline
  namespace: default
spec:
  steps:
    - name: step1
      template:
        apiVersion: batch/v1
        kind: Job
        spec:
          template:
            spec:
              containers:
                - name: main
                  image: busybox:latest
                  command: ["echo", "Step 1"]
              restartPolicy: OnFailure
    - name: step2
      dependencies: ["step1"]
      template:
        apiVersion: batch/v1
        kind: Job
        spec:
          template:
            spec:
              containers:
                - name: main
                  image: busybox:latest
                  command: ["echo", "Step 2"]
              restartPolicy: OnFailure
```

### 3. Apply and Monitor

```bash
kubectl apply -f simple-pipeline.yaml
kubectl get jobflow simple-pipeline
kubectl describe jobflow simple-pipeline
```

---

## Creating JobFlows

### Basic Structure

```yaml
apiVersion: workflow.zen.io/v1alpha1
kind: JobFlow
metadata:
  name: <name>
  namespace: <namespace>
spec:
  executionPolicy:
    # Optional execution policy
  resourceTemplates:
    # Optional resource templates
  steps:
    # List of steps
```

### Steps

Each step defines:
- **name**: Unique step name
- **dependencies**: Steps that must complete first
- **template**: Kubernetes Job template
- **retryPolicy**: Optional retry configuration
- **timeoutSeconds**: Optional timeout
- **continueOnFailure**: Whether to continue on failure

---

## Step Dependencies

Steps can depend on other steps, creating a DAG:

```yaml
steps:
  - name: extract-a
    template: ...
  - name: extract-b
    template: ...
  - name: transform-a
    dependencies: ["extract-a"]
    template: ...
  - name: transform-b
    dependencies: ["extract-b"]
    template: ...
  - name: merge
    dependencies: ["transform-a", "transform-b"]
    template: ...
```

The controller executes steps in topological order based on dependencies.

---

## Execution Policies

### Concurrency Policy

```yaml
executionPolicy:
  concurrencyPolicy: Forbid  # Allow, Forbid, Replace
```

- **Allow**: Multiple executions allowed
- **Forbid**: Only one execution at a time
- **Replace**: Cancel existing execution

### TTL After Finished

```yaml
executionPolicy:
  ttlSecondsAfterFinished: 3600  # 1 hour
```

Automatically clean up completed JobFlows after specified time.

### Backoff Limit

```yaml
executionPolicy:
  backoffLimit: 3
```

Maximum number of retries for the entire flow.

### Active Deadline

```yaml
executionPolicy:
  activeDeadlineSeconds: 7200  # 2 hours
```

Maximum duration for the entire flow.

---

## Resource Templates

Create PVCs and ConfigMaps before executing steps:

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

## Examples

### Linear Pipeline

```yaml
apiVersion: workflow.zen.io/v1alpha1
kind: JobFlow
metadata:
  name: linear-pipeline
spec:
  steps:
    - name: step1
      template: ...
    - name: step2
      dependencies: ["step1"]
      template: ...
    - name: step3
      dependencies: ["step2"]
      template: ...
```

### Parallel Steps

```yaml
steps:
  - name: task-a
    template: ...
  - name: task-b
    template: ...
  - name: merge
    dependencies: ["task-a", "task-b"]
    template: ...
```

### With Retries

```yaml
steps:
  - name: unreliable-step
    retryPolicy:
      limit: 5
      backoff:
        type: Exponential
        duration: "10s"
    template: ...
```

### Continue on Failure

```yaml
steps:
  - name: optional-step
    continueOnFailure: true
    template: ...
```

---

## Troubleshooting

### Check JobFlow Status

```bash
kubectl get jobflow <name>
kubectl describe jobflow <name>
```

### View Step Status

```bash
kubectl get jobflow <name> -o yaml
# Check status.steps for step status
```

### Check Created Jobs

```bash
kubectl get jobs -l jobflow=<name>
```

### View Logs

```bash
kubectl logs job/<job-name>
```

### Common Issues

**JobFlow stuck in Pending:**
- Check if controller is running
- Check for validation errors
- Check DAG for cycles

**Step not executing:**
- Check dependencies are met
- Check step status
- Check Job creation errors

**JobFlow failed:**
- Check step failures
- Check retry policy
- Check timeout settings

---

## See Also

- [API Reference](API_REFERENCE.md) - Complete API documentation
- [Architecture](ARCHITECTURE.md) - How zen-flow works
- [Examples](../examples/) - More examples

