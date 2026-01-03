# zen-flow

Kubernetes controller for managing workflow execution flows.

## Overview

`zen-flow` is a Kubernetes controller that manages workflow execution flows, ensuring proper sequencing and error handling for complex operations.

## Dependencies

This component uses `zen-sdk` for unified observability and infrastructure:

- **`zen-sdk/pkg/logging`** - Structured, context-aware logging
- **`zen-sdk/pkg/observability`** - OpenTelemetry distributed tracing
- **`zen-sdk/pkg/leader`** - Kubernetes leader election

See [zen-sdk README](https://github.com/kube-zen/zen-sdk/blob/main/README.md) for more information about the SDK packages.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/)
[![CI](https://github.com/kube-zen/zen-flow/workflows/CI/badge.svg)](https://github.com/kube-zen/zen-flow/actions)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.26+-326CE5?logo=kubernetes&logoColor=white)](https://kubernetes.io/)

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
  - **Artifacts & Parameters**: Structure in place for passing data between steps (implementation pending - see [Limitations](#limitations))

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

**Note**: The Helm chart is maintained in the centralized [helm-charts](https://github.com/kube-zen/helm-charts) repository.

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
helm upgrade zen-flow zen-flow/zen-flow \
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
helm upgrade zen-flow zen-flow/zen-flow \
  --namespace zen-flow-system \
  --set webhook.enabled=true \
  --set webhook.failurePolicy=Ignore
```

See [Webhook Setup Guide](docs/WEBHOOK_SETUP.md) for detailed instructions.

## High Availability

zen-flow supports High Availability (HA) to ensure continuous operation and prevent split-brain scenarios during upgrades or failures.

### Configuration

High Availability is **disabled by default** (`ha.enabled=false`) for simple development and testing setups. This deploys a single replica without leader election.

**⚠️ WARNING: Running with High Availability disabled (`ha.enabled=false`) is unsafe for production workloads due to risk of split-brain scenarios. If you disable HA but manually scale replicas > 1, you must implement your own locking mechanism.**

### Enabling High Availability

To enable HA for production deployments:

```bash
helm install zen-flow zen-flow/zen-flow \
  --namespace zen-flow-system \
  --create-namespace \
  --set ha.enabled=true
```

This automatically:
- Sets `replicas: 3` (instead of 1)
- Enables leader election
- Ensures only one replica processes reconciliation events

### Leader Election (Built-in)

When `ha.enabled=true`, zen-flow uses built-in leader election via `zen-sdk/pkg/leader` (controller-runtime Lease-based). This ensures only one replica processes reconciliation events, preventing split-brain scenarios.

**How it works:**
- Controller-runtime Manager uses Kubernetes Lease API for leader election
- Only the leader pod runs reconcilers
- On leader failure, another pod automatically becomes leader
- No pod mutation, no external dependencies—pure controller-runtime leader election

**Built-in Leader Election (Optional Fallback)**

Built-in Kubernetes leader election via Lease API remains available in the codebase but is not the default. It can be enabled by setting `--enable-leader-election=true` and configuring the manager options, but zen-lead is the recommended approach.

### Configuration Examples

**Development (HA Disabled):**
```yaml
ha:
  enabled: false  # Single replica, no leader election
```

**Production (HA Enabled with zen-lead):**
```yaml
ha:
  enabled: true  # Uses zen-lead for leader election (recommended)
```

**Note:** When `ha.enabled=true`, you must also annotate the zen-flow Service with `zen-lead.io/enabled: "true"` for leader election to work.

### Split-Brain Prevention

When `ha.enabled=true`, zen-flow ensures only one replica processes reconciliation events:

- **Built-in leader election**: Uses controller-runtime's Lease-based leader election (via zen-sdk/pkg/leader)
- **Disabled mode**: ⚠️ **No protection** - multiple replicas will all process events (split-brain risk)

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
- **CRDs**: Included in chart (available in helm-charts repository) or available in `deploy/crds/`

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

## Limitations

**Version 0.0.1-alpha** has the following limitations:

### Artifacts & Parameters
- **Status**: Structure in place, implementation pending
- **Current Behavior**: Artifact and parameter definitions are accepted but not processed
- **Impact**: Data cannot be passed between steps yet
- **Workaround**: Use ConfigMaps or PVCs directly in Job templates

### When Conditions
- **Status**: Basic keyword support only
- **Supported**: `always`, `never`, `true`, `false`
- **Not Supported**: Step status evaluation (e.g., `steps.step1.phase == 'Succeeded'`)
- **Impact**: Complex conditional execution is limited
- **Workaround**: Use dependencies to control execution flow

### Future Enhancements
See [ROADMAP.md](ROADMAP.md) for planned enhancements.

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

