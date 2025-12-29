# zen-flow

[![Version](https://img.shields.io/badge/version-0.0.1--alpha-blue)](https://github.com/kube-zen/zen-flow)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)
[![CI](https://github.com/kube-zen/zen-flow/workflows/CI/badge.svg)](https://github.com/kube-zen/zen-flow/actions)

**zen-flow** is a Kubernetes-native job orchestration controller that provides declarative, sequential execution of Kubernetes Jobs using standard CRDs. It addresses the critical gap between isolated Kubernetes Jobs and heavyweight workflow engines like Argo Workflows, offering production-grade orchestration with zero operational overhead.

## Features

- **Kubernetes-Native**: Built on standard Kubernetes APIs and CRDs
- **DAG Support**: Execute jobs with complex dependencies
- **Zero Dependencies**: No external databases or message queues
- **Observability**: Built-in Prometheus metrics and Kubernetes events
- **Security**: Pod Security Standards, RBAC, and audit logging
- **Production-Ready**: Designed for enterprise-grade deployments
- **Advanced Features**:
  - **Manual Approval Steps**: Pause workflows and wait for human approval
  - **TTL Cleanup**: Automatic cleanup of completed JobFlows
  - **Step Retries**: Configurable retry policies with exponential/linear/fixed backoff
  - **Timeouts**: Step-level and flow-level timeout enforcement
  - **Concurrency Control**: Allow/Forbid/Replace policies for concurrent executions
  - **Pod Failure Policies**: Handle pod failures based on exit codes
  - **Conditional Execution**: When conditions for conditional step execution
  - **Artifacts & Parameters**: Support for passing data between steps (structure in place)

## Quick Start

### Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured to access your cluster

### Installation

1. **Deploy the CRD:**
   ```bash
   kubectl apply -f deploy/crds/
   ```

2. **Deploy the controller:**
   ```bash
   kubectl apply -f deploy/manifests/
   ```

3. **Verify installation:**
   ```bash
   kubectl get pods -n zen-flow-system
   kubectl get jobflows
   ```

### Example Usage

Create a simple linear flow:

```yaml
apiVersion: workflow.zen.io/v1alpha1
kind: JobFlow
metadata:
  name: my-flow
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

Apply it:

```bash
kubectl apply -f examples/simple-linear-flow.yaml
```

Check status:

```bash
kubectl get jobflow my-flow
kubectl describe jobflow my-flow
```

## Architecture

zen-flow follows the standard Kubernetes controller pattern:

- **CRD**: `JobFlow` custom resource defines workflows
- **Controller**: Reconciles JobFlow resources and creates Jobs
- **DAG Engine**: Executes steps in topological order based on dependencies
- **Metrics**: Prometheus metrics for observability
- **Events**: Kubernetes events for debugging

## API Reference

### JobFlow Spec

```yaml
apiVersion: workflow.zen.io/v1alpha1
kind: JobFlow
metadata:
  name: example-flow
spec:
  executionPolicy:
    concurrencyPolicy: Forbid  # Allow, Forbid, Replace
    ttlSecondsAfterFinished: 86400
    backoffLimit: 6
    activeDeadlineSeconds: 21600
  steps:
    - name: step-name
      type: Job  # Job (default) or Manual
      dependencies: []  # List of step names that must complete first
      continueOnFailure: false
      message: "Approval message (for Manual type)"
      retryPolicy:
        limit: 3
        backoff:
          type: Exponential
          factor: 2
          duration: "10s"
      timeoutSeconds: 3600
      template:
        # Standard Kubernetes Job spec (required for Job type, ignored for Manual)
        apiVersion: batch/v1
        kind: Job
        spec:
          # ... Job specification
```

### JobFlow Status

```yaml
status:
  phase: Running  # Pending, Running, Succeeded, Failed, Suspended
  startTime: "2025-01-15T10:00:00Z"
  completionTime: null
  steps:
    - name: step-name
      phase: Running  # Pending, Running, Succeeded, Failed, Skipped
      startTime: "2025-01-15T10:00:05Z"
      jobRef:
        name: example-flow-step-name-abc123
  progress:
    completedSteps: 1
    totalSteps: 3
    successfulSteps: 1
    failedSteps: 0
```

## Development

### Building

```bash
# Build binary
make build

# Build Docker image
make build-image

# Run locally
make run
```

### Testing

```bash
# Run unit tests
make test-unit

# Run E2E tests
make test-e2e

# Check coverage
make coverage

# Run all checks
make ci-check
```

### CI/CD

The project uses GitHub Actions for continuous integration:

- **Lint**: Runs `golangci-lint`, `go vet`, formatting checks, and YAML linting
- **Test**: Runs unit and integration tests with coverage reporting
- **Build**: Builds the controller binary
- **Security**: Runs vulnerability scans (govulncheck, gosec, Trivy)
- **Multi-Arch Build**: Builds Docker images for linux/amd64 and linux/arm64
- **Helm Chart**: Publishes Helm charts to GitHub Pages

See `.github/workflows/` for workflow definitions.

### Project Structure

```
zen-flow/
├── cmd/
│   └── zen-flow-controller/    # Main controller binary
├── pkg/
│   ├── api/
│   │   └── v1alpha1/           # JobFlow CRD types
│   └── controller/             # Controller implementation
│       ├── dag/                # DAG execution engine
│       └── metrics/            # Prometheus metrics
├── deploy/
│   ├── crds/                   # CRD definitions
│   └── manifests/              # Deployment manifests
├── examples/                    # Example JobFlow manifests
└── README.md
```

## Comparison

| Feature | zen-flow | Argo Workflows | Kubernetes Jobs |
|---------|----------|----------------|-----------------|
| Installation | Single Deployment | Complex (DB, UI, MinIO) | Native |
| Learning Curve | Low (K8s-native) | High (DSL, concepts) | Low |
| DAG Support | ✅ | ✅ | ❌ |
| Artifact Management | ✅ (PVC, S3, HTTP) | ✅ (complex) | ❌ |
| Observability | Built-in (Prometheus) | Add-ons needed | Basic |
| Resource Footprint | ~50MB | ~1GB+ | None |

## Roadmap

### Phase 1: MVP (v1alpha1) ✅
- Basic sequential execution
- Kubernetes Job creation/management
- Status reporting
- Basic metrics and events

### Phase 2: Production Ready (v1beta1)
- DAG execution with complex dependencies
- Artifact passing (PVC-based)
- Retry policies with exponential backoff
- Validating webhooks
- Performance optimization

### Phase 3: Enterprise Features (v1beta2)
- Parameter substitution and templating
- Multi-cluster support
- Advanced artifact management (S3, GCS, HTTP)
- Workflow suspension/resumption
- Audit logging and compliance features

### Phase 4: Stable (v1)
- API stability guarantee
- Migration tools
- Production deployments validated
- Performance at scale (1000+ concurrent flows)

## Contributing

Contributions are welcome! Please see our contributing guidelines for details.

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/kube-zen/zen-flow/issues)
- **Documentation**: [GitHub Wiki](https://github.com/kube-zen/zen-flow/wiki)

## Acknowledgments

zen-flow is part of the Kube-ZEN suite of Kubernetes-native tools.

