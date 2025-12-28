# Operator Guide

This guide is for operators who need to install, configure, and maintain the zen-flow controller.

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [Upgrading](#upgrading)
- [Security](#security)

---

## Installation

### Prerequisites

- Kubernetes cluster 1.24+
- kubectl configured
- Cluster admin permissions (for CRD installation)

### Step 1: Install CRD

```bash
kubectl apply -f deploy/crds/workflow.zen.io_jobflows.yaml
```

Verify installation:

```bash
kubectl get crd jobflows.workflow.zen.io
```

### Step 2: Install Controller

```bash
# Create namespace
kubectl apply -f deploy/manifests/namespace.yaml

# Install RBAC
kubectl apply -f deploy/manifests/rbac.yaml

# Install Deployment
kubectl apply -f deploy/manifests/deployment.yaml

# Install Service
kubectl apply -f deploy/manifests/service.yaml
```

Or use Helm:

```bash
helm install zen-flow charts/zen-flow/
```

### Step 3: Verify Installation

```bash
# Check controller is running
kubectl get pods -n zen-flow-system

# Check logs
kubectl logs -n zen-flow-system -l app=zen-flow-controller

# Check metrics endpoint
kubectl port-forward -n zen-flow-system svc/zen-flow-controller 8080:8080
curl http://localhost:8080/metrics
```

---

## Configuration

### Environment Variables

- `METRICS_ADDR` - Metrics server address (default: `:8080`)
- `KUBECONFIG` - Path to kubeconfig file (for local development)
- `POD_NAMESPACE` - Namespace for leader election
- `POD_NAME` - Pod name for leader election

### Command Line Flags

```bash
--kubeconfig=""                    # Path to kubeconfig file
--master=""                        # Kubernetes API server address
--metrics-addr=":8080"             # Metrics server address
--enable-leader-election=true      # Enable leader election
--webhook-addr=""                  # Webhook server address
--enable-webhook=false             # Enable webhooks
```

---

## Monitoring

### Metrics

The controller exposes Prometheus metrics on `/metrics`:

- `zen_flow_jobflows_total` - JobFlow counts by phase
- `zen_flow_steps_total` - Step counts by phase
- `zen_flow_step_duration_seconds` - Step execution duration
- `zen_flow_reconciliation_duration_seconds` - Reconciliation duration

### Grafana Dashboard

Import the dashboard from `deploy/grafana/dashboard.json`:

```bash
kubectl apply -f deploy/grafana/dashboard.json
```

### Prometheus Alerts

Apply alerting rules:

```bash
kubectl apply -f deploy/prometheus/prometheus-rules.yaml
```

---

## Troubleshooting

### Controller Not Starting

```bash
# Check pod status
kubectl get pods -n zen-flow-system

# Check logs
kubectl logs -n zen-flow-system -l app=zen-flow-controller

# Check events
kubectl get events -n zen-flow-system
```

### JobFlows Not Processing

```bash
# Check controller is leader
kubectl get lease -n zen-flow-system

# Check JobFlow status
kubectl get jobflow <name> -o yaml

# Check for validation errors
kubectl describe jobflow <name>
```

### High Resource Usage

```bash
# Check resource usage
kubectl top pods -n zen-flow-system

# Check reconciliation rate
curl http://localhost:8080/metrics | grep reconciliation
```

---

## Upgrading

### Upgrade Steps

1. Backup existing JobFlows:
   ```bash
   kubectl get jobflows -A -o yaml > backup.yaml
   ```

2. Upgrade CRD (if needed):
   ```bash
   kubectl apply -f deploy/crds/
   ```

3. Upgrade controller:
   ```bash
   kubectl apply -f deploy/manifests/deployment.yaml
   ```

4. Verify upgrade:
   ```bash
   kubectl get pods -n zen-flow-system
   kubectl logs -n zen-flow-system -l app=zen-flow-controller
   ```

---

## Security

### RBAC

The controller requires:
- Full access to `jobflows.workflow.zen.io` CRD
- Create/update/delete Jobs
- Create/update/delete PVCs and ConfigMaps
- Leader election permissions

See [RBAC.md](RBAC.md) for details.

### Pod Security

The controller runs with:
- Non-root user
- Read-only root filesystem
- Dropped capabilities
- Seccomp profile

### Network Policies

Restrict network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: zen-flow-controller
  namespace: zen-flow-system
spec:
  podSelector:
    matchLabels:
      app: zen-flow-controller
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080
```

---

## See Also

- [Metrics](METRICS.md) - Metrics documentation
- [RBAC](RBAC.md) - RBAC permissions
- [Security](SECURITY.md) - Security best practices

