# Webhook Configuration Files (Alternative Setup)

**Note**: These files are provided as an alternative webhook setup that creates a separate `zen-flow-webhook` service.

## Canonical Installation Paths

- **Helm (Recommended)**: Use Helm chart - webhooks are automatically configured
- **kubectl-only**: Use `deploy/manifests/webhook.yaml` which points to `zen-flow-controller-metrics` service

## When to Use These Files

These files in `deploy/webhook/` create a separate webhook service (`zen-flow-webhook`) and are provided for:
- Custom deployments requiring separate webhook service
- Testing alternative webhook configurations

For standard installations, use:
- Helm chart (recommended), OR
- `deploy/manifests/webhook.yaml` (kubectl-only)

## Files

- `certificate.yaml`: Cert-manager Issuer and Certificate (references `zen-flow-webhook` service)
- `validating-webhook.yaml`: Validating webhook config + Service (references `zen-flow-webhook`)
- `mutating-webhook.yaml`: Mutating webhook config (references `zen-flow-webhook`)

## Service Name Mismatch

⚠️ **Important**: These files reference `zen-flow-webhook` service, while `deploy/manifests/webhook.yaml` references `zen-flow-controller-metrics`. 

If using these files, ensure the Certificate DNS names match the service you're using.

