# Webhook Setup Guide

This guide explains how to set up and configure the zen-flow webhook server for validating and mutating JobFlow resources.

## Overview

zen-flow provides two types of admission webhooks:

1. **Validating Webhook**: Validates JobFlow resources before they are created or updated
2. **Mutating Webhook**: Sets default values for JobFlow resources on creation

## Prerequisites

- Kubernetes cluster (1.20+)
- cert-manager installed (for automatic certificate management)
- zen-flow controller deployed

## Installation

### 1. Install cert-manager (if not already installed)

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

### 2. Deploy Webhook Manifests

```bash
# Deploy certificate issuer and certificate
kubectl apply -f deploy/webhook/certificate.yaml

# Wait for certificate to be ready
kubectl wait --for=condition=Ready certificate/zen-flow-webhook-cert -n zen-flow-system --timeout=60s

# Deploy webhook configurations
kubectl apply -f deploy/webhook/validating-webhook.yaml
kubectl apply -f deploy/webhook/mutating-webhook.yaml
```

### 3. Verify Webhook Installation

```bash
# Check certificate
kubectl get certificate -n zen-flow-system

# Check webhook configurations
kubectl get validatingwebhookconfiguration zen-flow-validating-webhook
kubectl get mutatingwebhookconfiguration zen-flow-mutating-webhook

# Check webhook service
kubectl get svc zen-flow-webhook -n zen-flow-system
```

## Configuration

### Webhook Flags

The webhook server supports the following command-line flags:

- `--webhook-addr`: Address to bind the webhook server (default: `:9443`)
- `--webhook-cert-file`: Path to TLS certificate file (default: `/etc/webhook/certs/tls.crt`)
- `--webhook-key-file`: Path to TLS private key file (default: `/etc/webhook/certs/tls.key`)
- `--enable-webhook`: Enable webhook server (default: `true`)
- `--insecure-webhook`: Allow webhook to start without TLS (testing only, NOT for production)

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

## Testing

### Test Webhook Locally (Without TLS)

```bash
# Start controller with insecure webhook
./bin/zen-flow-controller \
  --insecure-webhook=true \
  --enable-webhook=true \
  --webhook-addr=:9443
```

### Test Webhook Validation

```bash
# Create a valid JobFlow
kubectl apply -f examples/simple-linear-flow.yaml

# Try to create an invalid JobFlow (should be rejected)
kubectl apply -f - <<EOF
apiVersion: workflow.zen.io/v1alpha1
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
apiVersion: workflow.zen.io/v1alpha1
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
   kubectl port-forward -n zen-flow-system svc/zen-flow-webhook 9443:443
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

