# Webhook Setup Guide

This guide explains how to set up and configure the zen-flow webhook server for validating and mutating JobFlow resources.

## Overview

zen-flow provides two types of admission webhooks:

1. **Validating Webhook**: Validates JobFlow resources before they are created or updated
2. **Mutating Webhook**: Sets default values for JobFlow resources on creation

**Installation Methods:**
- **Helm (Recommended)**: Webhooks are managed via Helm chart templates. See [Operator Guide](OPERATOR_GUIDE.md) for Helm installation.
- **kubectl-only**: This guide covers manual webhook setup for kubectl installations.

## Prerequisites

- Kubernetes cluster (1.20+)
- cert-manager installed (for automatic certificate management)
- zen-flow controller deployed

## Installation

### 1. Install cert-manager (if not already installed)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

### 2. Deploy Certificate (cert-manager)

**Note**: For kubectl-only installations, you need to create a cert-manager Issuer first, then the Certificate.

```bash
# Create cert-manager Issuer (self-signed for testing, or use your own ClusterIssuer)
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: zen-flow-webhook-cert
  namespace: zen-flow-system
spec:
  selfSigned: {}
EOF

# Create Certificate
cat <<EOF | kubectl apply -f -
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: zen-flow-webhook-cert
  namespace: zen-flow-system
spec:
  dnsNames:
    - zen-flow-controller-metrics.zen-flow-system.svc
    - zen-flow-controller-metrics.zen-flow-system.svc.cluster.local
  issuerRef:
    name: zen-flow-webhook-cert
    kind: Issuer
  secretName: zen-flow-webhook-cert
EOF

# Wait for certificate to be ready
kubectl wait --for=condition=Ready certificate/zen-flow-webhook-cert -n zen-flow-system --timeout=60s
```

**Alternative**: Use the certificate file in `deploy/webhook/certificate.yaml` but note it references `zen-flow-webhook` service. For kubectl-only installs using `deploy/manifests/`, update the DNS names to match `zen-flow-controller-metrics`.

### 3. Deploy Webhook Configurations

```bash
# Deploy webhook configurations (with cert-manager CA injection)
kubectl apply -f deploy/manifests/webhook.yaml
```

**Canonical Path**: `deploy/manifests/webhook.yaml` is the canonical webhook configuration for kubectl-only installations. It points to the `zen-flow-controller-metrics` service.

**Note**: The webhook configurations in `deploy/manifests/webhook.yaml` use cert-manager annotations to automatically inject the CA bundle. If you're not using cert-manager, you must either:
- Manually set the `caBundle` field in the webhook configurations, or
- Set `failurePolicy: Ignore` to allow requests even if webhook is unreachable

### 4. Verify Webhook Installation

```bash
# Check certificate
kubectl get certificate -n zen-flow-system

# Check webhook configurations
kubectl get validatingwebhookconfiguration zen-flow-validating-webhook
kubectl get mutatingwebhookconfiguration zen-flow-mutating-webhook

# Check webhook service (webhook port is on the metrics service)
kubectl get svc zen-flow-controller-metrics -n zen-flow-system
```

## Configuration

### Webhook Flags

The webhook server supports the following command-line flags:

- `--webhook-addr`: Address to bind the webhook server (default: `:9443`)
- `--webhook-cert-file`: Path to TLS certificate file (default: `/etc/webhook/certs/tls.crt`)
- `--webhook-key-file`: Path to TLS private key file (default: `/etc/webhook/certs/tls.key`)
- `--enable-webhook`: Enable webhook server (default: `true`)
- `--insecure-webhook`: Allow webhook to start without TLS (testing only, NOT for production)
  - **WARNING**: If using `--insecure-webhook=true`, you MUST either:
    - Disable webhooks entirely (`--enable-webhook=false`), or
    - Set `failurePolicy: Ignore` in webhook configurations
  - Kubernetes admission webhooks require TLS. HTTP webhooks will fail TLS handshake and block resource creation.

### Certificate Management

The webhook uses cert-manager to automatically manage TLS certificates:

1. **Certificate Issuer**: Creates a self-signed certificate issuer
2. **Certificate**: Requests a certificate for the webhook service
3. **Secret**: cert-manager creates a secret with the certificate
4. **Volume Mount**: The deployment mounts the secret as a volume

The certificate is automatically renewed by cert-manager before expiration.

## Validation Rules

The validating webhook enforces the following rules:

1. **At least one step**: JobFlow must have at least one step
2. **Unique step names**: Step names must be unique within a JobFlow
3. **Valid dependencies**: All step dependencies must reference existing steps
4. **Non-empty step names**: Step names cannot be empty

## Mutation Rules

The mutating webhook sets the following defaults:

1. **ExecutionPolicy**:
   - `concurrencyPolicy`: `"Forbid"` (if not specified)
   - `ttlSecondsAfterFinished`: `86400` (24 hours, if not specified)
   - `backoffLimit`: `6` (if not specified)

## Development Setup

For development/testing without cert-manager, you have two options:

### Option 1: Disable Webhooks (Recommended for Dev)

```bash
# Deploy controller with webhooks disabled
# Edit deploy/manifests/deployment.yaml and set:
#   - --enable-webhook=false
# Then deploy:
kubectl apply -f deploy/manifests/deployment.yaml
```

### Option 2: Use Insecure Webhook with failurePolicy: Ignore

**WARNING**: This is for local development only. Kubernetes admission webhooks require TLS.

1. Deploy controller with `--insecure-webhook=true`
2. Edit webhook configurations to set `failurePolicy: Ignore`:

```bash
# Edit deploy/manifests/webhook.yaml and change:
# failurePolicy: Fail  ->  failurePolicy: Ignore
kubectl apply -f deploy/manifests/webhook.yaml
```

**Note**: With `failurePolicy: Ignore`, webhook failures won't block resource creation, but validation/mutation won't work.

## Testing

### Test Webhook Locally (With TLS)

```bash
# Start controller with TLS (requires certificates)
./bin/zen-flow-controller \
  --enable-webhook=true \
  --webhook-addr=:9443 \
  --webhook-cert-file=/path/to/tls.crt \
  --webhook-key-file=/path/to/tls.key
```

### Test Webhook Validation

```bash
# Create a valid JobFlow
kubectl apply -f examples/simple-linear-flow.yaml

# Try to create an invalid JobFlow (should be rejected)
kubectl apply -f - <<EOF
apiVersion: workflow.kube-zen.io/v1alpha1
kind: JobFlow
metadata:
  name: invalid-flow
spec:
  steps: []  # Invalid: no steps
EOF
```

### Test Webhook Mutation

```bash
# Create a JobFlow without ExecutionPolicy
kubectl apply -f - <<EOF
apiVersion: workflow.kube-zen.io/v1alpha1
kind: JobFlow
metadata:
  name: test-mutation
spec:
  steps:
    - name: step1
      template:
        raw: '{"spec":{"template":{"spec":{"containers":[{"name":"main","image":"busybox:latest"}]}}}}'
EOF

# Check that defaults were applied
kubectl get jobflow test-mutation -o yaml | grep -A 5 executionPolicy
```

## Troubleshooting

### Webhook Not Working

1. **Check webhook server logs**:
   ```bash
   kubectl logs -n zen-flow-system -l app=zen-flow-controller | grep webhook
   ```

2. **Check certificate status**:
   ```bash
   kubectl describe certificate zen-flow-webhook-cert -n zen-flow-system
   ```

3. **Check webhook configuration**:
   ```bash
   kubectl describe validatingwebhookconfiguration zen-flow-validating-webhook
   kubectl describe mutatingwebhookconfiguration zen-flow-mutating-webhook
   ```

4. **Test webhook endpoint**:
   ```bash
   kubectl port-forward -n zen-flow-system svc/zen-flow-controller-metrics 9443:9443
   curl -k https://localhost:9443/healthz
   ```

### Certificate Issues

If certificates are not being created:

1. Check cert-manager is installed:
   ```bash
   kubectl get pods -n cert-manager
   ```

2. Check certificate issuer:
   ```bash
   kubectl get issuer zen-flow-webhook-cert -n zen-flow-system
   ```

3. Check certificate events:
   ```bash
   kubectl describe certificate zen-flow-webhook-cert -n zen-flow-system
   ```

### Webhook Rejection

If webhook is rejecting valid resources:

1. Check webhook logs for validation errors
2. Verify the resource matches the validation rules
3. Check if mutating webhook is interfering

## Security Considerations

1. **TLS Required**: Never use `--insecure-webhook` in production
2. **Certificate Rotation**: cert-manager automatically rotates certificates
3. **Network Policies**: Consider restricting webhook access with NetworkPolicies
4. **RBAC**: Ensure webhook service account has minimal required permissions

## Disabling Webhooks

To disable webhooks:

1. **Disable in deployment**:
   ```yaml
   args:
     - --enable-webhook=false
   ```

2. **Delete webhook configurations**:
   ```bash
   kubectl delete validatingwebhookconfiguration zen-flow-validating-webhook
   kubectl delete mutatingwebhookconfiguration zen-flow-mutating-webhook
   ```

## References

- [Kubernetes Admission Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
- [cert-manager Documentation](https://cert-manager.io/docs/)
- [zen-flow API Reference](../API_REFERENCE.md)

