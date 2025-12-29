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
- Helm 3.x (for installation)

### Installation (Helm - Recommended)

Install zen-flow using Helm:

```bash
# Add the Helm repository
helm repo add zen-flow https://kube-zen.github.io/zen-flow/charts
helm repo update

# Install zen-flow
helm install zen-flow zen-flow/zen-flow \
  --namespace zen-flow-system \
  --create-namespace
```

**Alternative**: Install from local chart during development:
```bash
helm install zen-flow ./charts/zen-flow --namespace zen-flow-system --create-namespace
```

**Note**: Webhooks are disabled by default for safe installation. See [Enabling Webhooks](#enabling-webhooks) below.

### Verify Installation

```bash
# Check controller pod is running
kubectl get pods -n zen-flow-system

# Check CRDs are installed
kubectl get crd jobflows.workflow.kube-zen.io

# Check controller logs
kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow
```

### Create Your First JobFlow

Apply the example:

```bash
kubectl apply -f examples/simple-linear-flow.yaml
```

Watch the JobFlow progress:

```bash
# Watch JobFlow status
kubectl get jobflow simple-linear-flow -w

# Check detailed status
kubectl describe jobflow simple-linear-flow

# View created Jobs
kubectl get jobs -l jobflow=simple-linear-flow
```

### Expected Status Transitions

A JobFlow typically transitions through these phases:

1. **Pending** → JobFlow created, waiting to start
2. **Running** → Steps are executing
3. **Succeeded** → All steps completed successfully

Watch the transitions:

```bash
kubectl get jobflow simple-linear-flow -o jsonpath='{.status.phase}' && echo
```

### Troubleshooting

**Controller not starting:**
```bash
# Check pod status
kubectl get pods -n zen-flow-system

# Check logs
kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow

# Check events
kubectl get events -n zen-flow-system --sort-by='.lastTimestamp'
```

**JobFlow stuck in Pending:**
```bash
# Check for validation errors
kubectl describe jobflow <name>

# Check controller is running
kubectl get pods -n zen-flow-system
```

**Webhook issues (if enabled):**
```bash
# Check webhook configurations
kubectl get validatingwebhookconfiguration
kubectl get mutatingwebhookconfiguration

# Check certificate (if using cert-manager)
kubectl get certificate -n zen-flow-system
```

### Uninstall

```bash
# Uninstall Helm release
helm uninstall zen-flow --namespace zen-flow-system

# Clean up CRDs (optional - removes JobFlow CRD from cluster)
kubectl delete crd jobflows.workflow.kube-zen.io

# Clean up namespace (optional)
kubectl delete namespace zen-flow-system
```

### Enabling Webhooks

Webhooks are disabled by default for safe installation. To enable:

**Option 1: With cert-manager (Recommended for Production)**

```bash
# Install cert-manager first
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Upgrade Helm release with webhooks enabled
helm upgrade zen-flow ./charts/zen-flow \
  --namespace zen-flow-system \
  --set webhook.enabled=true \
  --set webhook.certManager.enabled=true \
  --set webhook.certManager.issuer.name=your-cluster-issuer \
  --set webhook.certManager.issuer.kind=ClusterIssuer \
  --set webhook.failurePolicy=Fail
```

**Option 2: Without cert-manager (Development)**

```bash
# Enable webhooks with Ignore failure policy (safe for dev)
helm upgrade zen-flow ./charts/zen-flow \
  --namespace zen-flow-system \
  --set webhook.enabled=true \
  --set webhook.failurePolicy=Ignore
```

See [Webhook Setup Guide](docs/WEBHOOK_SETUP.md) for detailed instructions.

### Alternative: kubectl Installation

For environments without Helm:

```bash
# Install CRDs
kubectl apply -f deploy/crds/

# Install controller
kubectl apply -f deploy/manifests/
```

**Note**: kubectl installation requires manual webhook certificate management. See [docs/WEBHOOK_SETUP.md](docs/WEBHOOK_SETUP.md).

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
apiVersion: workflow.kube-zen.io/v1alpha1
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
  startTime: "2015-12-29T10:00:00Z"
  completionTime: null
  steps:
    - name: step-name
      phase: Running  # Pending, Running, Succeeded, Failed, Skipped
      startTime: "2015-12-29T10:00:05Z"
      jobRef:
        name: example-flow-step-name-abc123
  progress:
    completedSteps: 1
    totalSteps: 3
    successfulSteps: 1
    failedSteps: 0
```

## Release Information

- **Docker Images**: Published to `docker.io/kubezen/zen-flow-controller`
  - Tags: `0.0.1-alpha`, `latest`
- **Helm Chart Repository**: `https://kube-zen.github.io/zen-flow/charts`
  - Chart version: `0.0.1-alpha`
  - App version: `0.0.1-alpha`
- **CRDs**: Included in chart (`charts/zen-flow/crds/`) or available in `deploy/crds/`

## Development

### Building

```bash
# Build binary
make build

# Build Docker image
make build-image

# Push to Docker Hub (requires login)
docker push kubezen/zen-flow-controller:0.0.1-alpha
docker push kubezen/zen-flow-controller:latest

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

