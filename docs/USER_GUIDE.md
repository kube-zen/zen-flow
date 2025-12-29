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
- [Step Retries](#step-retries)
- [Step Timeouts](#step-timeouts)
- [Concurrency Control](#concurrency-control)
- [Pod Failure Policies](#pod-failure-policies)
- [When Conditions](#when-conditions)
- [Manual Approval Steps](#manual-approval-steps)
- [TTL Cleanup](#ttl-cleanup)
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

## Step Retries

Configure automatic retries for failed steps with configurable backoff strategies.

### Retry Policy

```yaml
steps:
  - name: api-call
    retryPolicy:
      limit: 3  # Maximum retries (default: 3)
      backoff:
        type: Exponential  # Exponential, Linear, or Fixed
        factor: 2.0  # Backoff factor for exponential (default: 2.0)
        duration: "10s"  # Base duration
    template: ...
```

### Backoff Types

- **Exponential**: `duration * factor^retryCount` (e.g., 10s, 20s, 40s, 80s)
- **Linear**: `duration * (retryCount + 1)` (e.g., 10s, 20s, 30s, 40s)
- **Fixed**: `duration` (e.g., 10s, 10s, 10s, 10s)

### Example

```yaml
steps:
  - name: flaky-service
    retryPolicy:
      limit: 5
      backoff:
        type: Exponential
        factor: 2.0
        duration: "5s"
    template:
      apiVersion: batch/v1
      kind: Job
      spec:
        template:
          spec:
            containers:
              - name: main
                image: myapp:latest
                command: ["curl", "https://api.example.com"]
            restartPolicy: OnFailure
```

---

## Step Timeouts

Enforce maximum execution time for individual steps.

### Configuration

```yaml
steps:
  - name: data-processing
    timeoutSeconds: 3600  # 1 hour timeout
    template: ...
```

### Behavior

- Step is marked as failed if timeout is exceeded
- Job is automatically deleted when timeout occurs
- Flow continues or fails based on `continueOnFailure` setting

### Example

```yaml
steps:
  - name: long-task
    timeoutSeconds: 7200  # 2 hours
    continueOnFailure: false
    template:
      apiVersion: batch/v1
      kind: Job
      spec:
        template:
          spec:
            containers:
              - name: main
                image: processor:latest
                command: ["process", "--input", "/data"]
            restartPolicy: OnFailure
```

---

## Concurrency Control

Control how multiple JobFlow instances with the same name are handled.

### Policies

- **Allow**: Multiple executions can run concurrently
- **Forbid**: Only one execution at a time (default)
- **Replace**: Cancel existing execution and start new one

### Example

```yaml
apiVersion: workflow.zen.io/v1alpha1
kind: JobFlow
metadata:
  name: scheduled-job
spec:
  executionPolicy:
    concurrencyPolicy: Forbid  # Prevent concurrent executions
  steps:
    - name: task
      template: ...
```

---

## Pod Failure Policies

Handle pod failures based on exit codes and container names.

### Configuration

```yaml
executionPolicy:
  podFailurePolicy:
    rules:
      - action: Ignore  # Ignore, Count, or FailJob
        onExitCodes:
          containerName: main  # Optional, defaults to "main"
          operator: In  # In or NotIn
          values: [1, 2]  # Exit codes to match
```

### Actions

- **Ignore**: Don't mark step as failed
- **Count**: Count the failure but don't fail the step
- **FailJob**: Mark step as failed (default)

### Example

```yaml
executionPolicy:
  podFailurePolicy:
    rules:
      # Ignore expected exit codes
      - action: Ignore
        onExitCodes:
          operator: In
          values: [1, 2]
      # Fail on any other non-zero exit code
      - action: FailJob
        onExitCodes:
          operator: NotIn
          values: [0]
```

---

## When Conditions

Conditionally execute steps based on conditions (basic implementation).

### Configuration

```yaml
steps:
  - name: conditional-step
    when: "always"  # Condition expression
    template: ...
```

### Supported Values

- `always` or `true`: Always execute
- `never` or `false`: Never execute

**Note**: Full template engine support for step status evaluation is planned for future releases.

---

## Manual Approval Steps

Pause a workflow and wait for human approval before proceeding. This is essential for critical operations like database migrations, production deployments, or any step that requires manual verification.

### Configuration

```yaml
steps:
  - name: approve-migration
    type: Manual
    message: "Please verify backup before proceeding with schema migration"
    dependencies:
      - backup-db
```

### How It Works

1. When a manual approval step is reached, the JobFlow pauses (`status.phase: Paused`)
2. The step status shows `phase: PendingApproval` with the approval message
3. An administrator approves the step by annotating the JobFlow:
   ```bash
   kubectl annotate jobflow <name> workflow.zen.io/approved/<step-name>=true
   ```
4. The controller detects the annotation and marks the step as succeeded
5. The JobFlow resumes execution

### Approval Annotation Format

The annotation key follows the pattern: `workflow.zen.io/approved/<step-name>`

```bash
# Approve a specific step
kubectl annotate jobflow migration-prod workflow.zen.io/approved/approve-migration=true

# Check approval status
kubectl get jobflow migration-prod -o jsonpath='{.status.steps[?(@.name=="approve-migration")]}'
```

### Example: Database Migration with Approval

```yaml
apiVersion: workflow.zen.io/v1alpha1
kind: JobFlow
metadata:
  name: migration-with-approval
spec:
  steps:
    - name: backup-db
      type: Job
      template:
        apiVersion: batch/v1
        kind: Job
        spec:
          template:
            spec:
              containers:
                - name: backup
                  image: postgres:15
                  command:
                    - /bin/bash
                    - -c
                    - |
                      pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME > /backup/db-backup.sql
                      echo "Backup completed successfully"
                  env:
                    - name: DB_HOST
                      value: "postgres-service"
                    - name: DB_USER
                      valueFrom:
                        secretKeyRef:
                          name: db-credentials
                          key: username
                    - name: DB_NAME
                      value: "production"
                  volumeMounts:
                    - name: backup-volume
                      mountPath: /backup
              volumes:
                - name: backup-volume
                  persistentVolumeClaim:
                    claimName: backup-pvc
              restartPolicy: OnFailure

    - name: approve-migration
      type: Manual
      dependencies:
        - backup-db
      message: "Please verify backup before proceeding with schema migration"

    - name: migrate-schema
      type: Job
      dependencies:
        - approve-migration
      template:
        apiVersion: batch/v1
        kind: Job
        spec:
          template:
            spec:
              containers:
                - name: migrate
                  image: postgres:15
                  command:
                    - /bin/bash
                    - -c
                    - |
                      psql -h $DB_HOST -U $DB_USER -d $DB_NAME -f /migrations/schema.sql
                      echo "Migration completed successfully"
                  env:
                    - name: DB_HOST
                      value: "postgres-service"
                    - name: DB_USER
                      valueFrom:
                        secretKeyRef:
                          name: db-credentials
                          key: username
                    - name: DB_NAME
                      value: "production"
                  volumeMounts:
                    - name: migrations
                      mountPath: /migrations
              volumes:
                - name: migrations
                  configMap:
                    name: schema-migrations
              restartPolicy: OnFailure
```

### Workflow

1. **Backup runs**: The `backup-db` step executes and completes
2. **Flow pauses**: The `approve-migration` step is reached, JobFlow enters `Paused` phase
3. **Admin reviews**: Administrator checks backup logs and verifies backup was successful
4. **Admin approves**: 
   ```bash
   kubectl annotate jobflow migration-with-approval workflow.zen.io/approved/approve-migration=true
   ```
5. **Flow resumes**: Controller detects approval, marks step as succeeded, and continues to `migrate-schema`

### Status During Approval

```yaml
status:
  phase: Paused
  steps:
    - name: backup-db
      phase: Succeeded
    - name: approve-migration
      phase: PendingApproval
      message: "Please verify backup before proceeding with schema migration"
      startTime: "2025-12-28T10:00:00Z"
```

### Best Practices

- **Use descriptive messages**: Provide clear context about what needs to be verified
- **Place approvals strategically**: Put them after critical operations (backups, validations)
- **Monitor paused flows**: Set up alerts for JobFlows stuck in `Paused` phase
- **Document approval process**: Create runbooks for common approval scenarios

---

## TTL Cleanup

Automatically delete completed JobFlows after a specified time.

### Configuration

```yaml
executionPolicy:
  ttlSecondsAfterFinished: 3600  # 1 hour (default: 86400 = 24 hours)
```

### Behavior

- JobFlow is automatically deleted after TTL expires
- TTL starts counting from `completionTime`
- Set to `0` for immediate deletion after completion
- Default is 24 hours (86400 seconds)

### Example

```yaml
apiVersion: workflow.zen.io/v1alpha1
kind: JobFlow
metadata:
  name: temporary-job
spec:
  executionPolicy:
    ttlSecondsAfterFinished: 3600  # Clean up after 1 hour
  steps:
    - name: task
      template: ...
```

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
        factor: 2.0
        duration: "10s"
    template: ...
```

### With Timeouts

```yaml
steps:
  - name: long-running-step
    timeoutSeconds: 3600  # 1 hour
    template: ...
```

### With Pod Failure Policy

```yaml
executionPolicy:
  podFailurePolicy:
    rules:
      - action: Ignore
        onExitCodes:
          operator: In
          values: [1, 2]  # Ignore exit codes 1 and 2
      - action: FailJob
        onExitCodes:
          operator: NotIn
          values: [0]  # Fail on any non-zero exit code
```

### With When Conditions

```yaml
steps:
  - name: conditional-step
    when: "always"  # Basic condition (can be enhanced with template engine)
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

