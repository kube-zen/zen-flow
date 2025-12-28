# Architecture

## Overview

`zen-flow` is a Kubernetes-native job orchestration controller that provides declarative, sequential execution of Kubernetes Jobs using standard CRDs. It follows the standard Kubernetes controller pattern using the **controller-runtime** framework with informers, work queues, and reconciliation loops.

## System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │  API Server  │  │  JobFlow CRD  │  │  Job Resources│    │
│  └──────┬───────┘  └──────┬───────┘  └──────┬────────┘    │
└─────────┼─────────────────┼─────────────────┼─────────────┘
          │                 │                 │
          │                 │                 │
┌─────────┴─────────────────┴─────────────────┴─────────────┐
│                  zen-flow Controller                        │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  JobFlow Informer  │  Job Informer  │  PVC Informer │  │
│  └──────────┬──────────┴───────┬────────┴───────┬──────┘  │
│             │                   │                 │         │
│  ┌──────────▼───────────────────▼─────────────────▼──────┐ │
│  │              Work Queues                                │ │
│  │  ┌──────────────┐  ┌──────────────┐                  │ │
│  │  │ JobFlow Queue│  │  Job Queue    │                  │ │
│  │  └──────┬───────┘  └───────┬───────┘                  │ │
│  └─────────┼──────────────────┼──────────────────────────┘ │
│            │                  │                             │
│  ┌─────────▼──────────────────▼──────────────────────────┐ │
│  │              Reconciler                                 │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐│ │
│  │  │ DAG Engine   │  │ Step Executor │  │ Status Updater││ │
│  │  └──────────────┘  └───────────────┘  └──────────────┘│ │
│  └─────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  Metrics Recorder  │  Event Recorder  │  Validator     │ │
│  └─────────────────────────────────────────────────────────┘ │
└───────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Main Controller (`cmd/zen-flow-controller/main.go`)

The entry point that:
- Initializes controller-runtime manager
- Sets up leader election for HA (via controller-runtime)
- Registers JobFlow reconciler
- Configures metrics server
- Handles graceful shutdown (via controller-runtime)

### 2. JobFlow Reconciler (`pkg/controller/reconciler.go`)

Core reconciliation logic using controller-runtime:

**Responsibilities:**
- Reconcile `JobFlow` CRDs
- Create execution plans from DAG
- Execute steps in topological order
- Monitor Job status
- Update JobFlow status
- Emit metrics and events
- Handle TTL cleanup, retries, timeouts, and policies

**Key Methods:**
- `Reconcile()`: Main reconciliation entry point (controller-runtime)
- `createExecutionPlan()`: Build DAG and execution plan
- `executeStep()`: Execute a single step
- `updateJobFlowStatus()`: Update status
- `checkStepTimeouts()`: Enforce step timeouts
- `handleStepRetry()`: Handle step retries
- `checkConcurrencyPolicy()`: Enforce concurrency policies
- `checkPodFailurePolicy()`: Evaluate pod failure policies
- `shouldDeleteJobFlow()`: Check TTL for cleanup

### 3. DAG Engine (`pkg/controller/dag/dag.go`)

Directed Acyclic Graph engine:

**Responsibilities:**
- Build DAG from steps
- Detect cycles
- Topological sort
- Find ready steps

**Key Methods:**
- `BuildDAG()`: Build graph from steps
- `TopologicalSort()`: Sort steps in execution order
- `GetReadySteps()`: Find steps ready to execute
- `HasCycles()`: Detect cycles

### 4. Reconciliation Flow

```
1. JobFlow Created/Updated
   ↓
2. Validate JobFlow (webhook or controller)
   ↓
3. Build DAG from steps
   ↓
4. Check for cycles
   ↓
5. Initialize JobFlow if needed
   ↓
6. Build DAG from steps
   ↓
7. Check for cycles
   ↓
8. Refresh step statuses from Jobs
   ↓
9. Check step timeouts
   ↓
10. Handle step retries
   ↓
11. Create execution plan (topological sort)
   ↓
12. For each ready step:
    - Evaluate "when" conditions
    - Handle step inputs (artifacts/parameters)
    - Check if Job exists
    - If not, create Job from template
    - Monitor Job status
    ↓
13. Update step status (check pod failure policy)
   ↓
14. Handle step outputs (artifacts/parameters)
   ↓
15. Update JobFlow status
   ↓
16. Check TTL cleanup if finished
   ↓
17. Emit metrics and events
```

### 5. Step Execution

```
Step Ready
  ↓
Create Job from Template
  ↓
Wait for Job Status
  ↓
Job Succeeded?
  ├─ Yes → Mark Step Succeeded → Check Dependencies
  └─ No → Check Retry Policy
      ├─ Retries Left? → Retry with Backoff
      └─ No Retries → Mark Step Failed
          ├─ ContinueOnFailure? → Continue Flow
          └─ Stop Flow → Mark JobFlow Failed
```

### 6. Status Management

The controller maintains status for:
- **JobFlow Phase**: Pending, Running, Succeeded, Failed, Suspended
- **Step Status**: Pending, Running, Succeeded, Failed, Skipped
- **Progress**: Completed steps, total steps, success/failure counts
- **Conditions**: Ready, Completed, Failed conditions

### 7. Resource Templates

Before executing steps, the controller creates:
- **PersistentVolumeClaims**: From `resourceTemplates.volumeClaimTemplates`
- **ConfigMaps**: From `resourceTemplates.configMapTemplates`

These resources are created in the same namespace as the JobFlow.

### 8. Metrics and Observability

The controller exposes:
- **Prometheus Metrics**: JobFlow counts, step durations, reconciliation duration
- **Kubernetes Events**: Step execution, failures, completions
- **Structured Logging**: Correlation IDs, context

## Data Flow

### JobFlow Creation

```
User creates JobFlow
  ↓
API Server validates (webhook)
  ↓
JobFlow stored in etcd
  ↓
JobFlow Informer receives event
  ↓
Controller enqueues for reconciliation
  ↓
Reconciler processes JobFlow
  ↓
Creates execution plan
  ↓
Executes steps
```

### Step Execution

```
Step ready to execute
  ↓
Create Kubernetes Job
  ↓
Job Informer watches Job
  ↓
Job status changes
  ↓
Update step status
  ↓
Update JobFlow status
  ↓
Emit metrics/events
```

## Concurrency and Scalability

### Leader Election

- Multiple controller replicas for HA
- Only leader processes JobFlows
- Automatic failover on leader loss

### Work Queue

- Rate-limited work queue
- Configurable concurrency
- Exponential backoff on errors

### Resource Limits

- Configurable max concurrent reconciles
- Rate limiting for API calls
- Resource quotas for created Jobs

## Security

- **RBAC**: Minimal required permissions
- **Pod Security**: Non-root, read-only filesystem
- **Network Policies**: Restricted network access
- **Webhooks**: Validation and mutation

## See Also

- [API Reference](API_REFERENCE.md) - Complete API documentation
- [User Guide](USER_GUIDE.md) - How to use JobFlow
- [Operator Guide](OPERATOR_GUIDE.md) - Operations and maintenance

