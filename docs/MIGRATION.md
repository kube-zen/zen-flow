# Migration and Upgrade Guide

This guide helps you migrate and upgrade zen-flow between versions.

## Table of Contents

- [Upgrading zen-flow](#upgrading-zen-flow)
- [API Version Changes](#api-version-changes)
- [Breaking Changes](#breaking-changes)
- [Deprecations](#deprecations)
- [Migration Steps](#migration-steps)

---

## Upgrading zen-flow

### Helm Upgrade (Recommended)

```bash
# Update Helm repository
helm repo update zen-flow

# Check current version
helm list -n zen-flow-system

# Upgrade to latest version
helm upgrade zen-flow zen-flow/zen-flow \
  --namespace zen-flow-system \
  --reuse-values

# Or specify version explicitly
helm upgrade zen-flow zen-flow/zen-flow \
  --namespace zen-flow-system \
  --version 0.0.1-alpha \
  --reuse-values
```

### kubectl Upgrade

```bash
# Backup existing resources
kubectl get jobflows --all-namespaces -o yaml > jobflows-backup.yaml

# Update CRDs
kubectl apply -f deploy/crds/

# Update controller deployment
kubectl apply -f deploy/manifests/

# Restart controller to pick up changes
kubectl rollout restart deployment/zen-flow-controller -n zen-flow-system
```

### Verification

After upgrading, verify the installation:

```bash
# Check controller version
kubectl get deployment -n zen-flow-system zen-flow-controller -o jsonpath='{.spec.template.spec.containers[0].image}'

# Check CRD version
kubectl get crd jobflows.workflow.kube-zen.io -o jsonpath='{.spec.versions[*].name}'

# Check controller logs
kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow | head -20
```

---

## API Version Changes

### Current Version: v1alpha1

**Status**: Alpha - API may change in future versions

**Stability**: No stability guarantees for alpha APIs

### Future Versions

- **v1beta1**: Planned for production-ready features
- **v1**: Stable API with backward compatibility guarantees

---

## Breaking Changes

### Version 0.0.1-alpha

**No breaking changes** - This is the initial release.

### Future Breaking Changes

When breaking changes are introduced, they will be:
1. Documented in [CHANGELOG.md](../CHANGELOG.md)
2. Announced in release notes
3. Provided with migration scripts/tools when possible

---

## Deprecations

### Current Version (0.0.1-alpha)

**No deprecations** - This is the initial release.

### Deprecation Policy

- Deprecated features will be supported for at least 2 minor versions
- Deprecation warnings will be logged
- Migration guides will be provided

---

## Migration Steps

### From Pre-release to 0.0.1-alpha

If you were using a pre-release version:

1. **Backup existing JobFlows**
   ```bash
   kubectl get jobflows --all-namespaces -o yaml > backup.yaml
   ```

2. **Uninstall old version**
   ```bash
   helm uninstall zen-flow -n zen-flow-system
   # or
   kubectl delete -f deploy/manifests/
   ```

3. **Install new version**
   ```bash
   helm install zen-flow zen-flow/zen-flow \
     --namespace zen-flow-system \
     --create-namespace
   ```

4. **Verify JobFlows**
   ```bash
   kubectl get jobflows --all-namespaces
   ```

### API Group Migration (if applicable)

If migrating from a different API group:

1. **Export existing resources**
   ```bash
   kubectl get jobflows --all-namespaces -o yaml > old-jobflows.yaml
   ```

2. **Update API group in YAML**
   ```bash
   # Replace old API group with new one
   sed -i 's/old-group.io/workflow.kube-zen.io/g' old-jobflows.yaml
   ```

3. **Delete old CRD and resources**
   ```bash
   kubectl delete crd jobflows.old-group.io
   kubectl delete jobflows --all-namespaces
   ```

4. **Apply new CRD**
   ```bash
   kubectl apply -f deploy/crds/
   ```

5. **Recreate resources with new API group**
   ```bash
   kubectl apply -f old-jobflows.yaml
   ```

---

## Rollback Procedure

If you need to rollback to a previous version:

### Helm Rollback

```bash
# List release history
helm history zen-flow -n zen-flow-system

# Rollback to previous version
helm rollback zen-flow -n zen-flow-system

# Or rollback to specific revision
helm rollback zen-flow <revision> -n zen-flow-system
```

### kubectl Rollback

```bash
# Rollback deployment
kubectl rollout undo deployment/zen-flow-controller -n zen-flow-system

# Rollback to specific revision
kubectl rollout undo deployment/zen-flow-controller -n zen-flow-system --to-revision=<number>
```

### CRD Rollback

**⚠️ Warning**: Rolling back CRDs can be risky if the new version introduced schema changes.

```bash
# Backup current CRD
kubectl get crd jobflows.workflow.kube-zen.io -o yaml > crd-backup.yaml

# Apply old CRD version
kubectl apply -f <old-crd-version>.yaml

# Verify
kubectl get crd jobflows.workflow.kube-zen.io
```

---

## Pre-Upgrade Checklist

Before upgrading:

- [ ] Review [CHANGELOG.md](../CHANGELOG.md) for changes
- [ ] Backup all JobFlows: `kubectl get jobflows --all-namespaces -o yaml > backup.yaml`
- [ ] Check running JobFlows: `kubectl get jobflows --all-namespaces`
- [ ] Review breaking changes (if any)
- [ ] Test upgrade in non-production environment first
- [ ] Ensure sufficient cluster resources
- [ ] Verify Helm repository is up to date

---

## Post-Upgrade Checklist

After upgrading:

- [ ] Verify controller is running: `kubectl get pods -n zen-flow-system`
- [ ] Check controller logs for errors: `kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow`
- [ ] Verify CRD version: `kubectl get crd jobflows.workflow.kube-zen.io`
- [ ] Test JobFlow creation: `kubectl apply -f examples/simple-linear-flow.yaml`
- [ ] Verify metrics endpoint: `kubectl port-forward -n zen-flow-system svc/zen-flow-controller-metrics 8080:8080 && curl http://localhost:8080/metrics`
- [ ] Check webhook (if enabled): `kubectl get validatingwebhookconfiguration,mutatingwebhookconfiguration | grep zen-flow`

---

## Troubleshooting Upgrades

### Controller Not Starting After Upgrade

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md#controller-not-starting)

### JobFlows Not Working After Upgrade

1. Check CRD compatibility
   ```bash
   kubectl get crd jobflows.workflow.kube-zen.io -o yaml | grep version
   ```

2. Check JobFlow status
   ```bash
   kubectl get jobflows --all-namespaces -o yaml | grep -A 5 status
   ```

3. Check controller logs
   ```bash
   kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow | tail -50
   ```

### Webhook Issues After Upgrade

1. Verify webhook certificates
   ```bash
   kubectl get secret -n zen-flow-system zen-flow-webhook-cert
   ```

2. Check webhook configuration
   ```bash
   kubectl get validatingwebhookconfiguration validate-jobflow.workflow.kube-zen.io -o yaml
   ```

3. Temporarily disable webhooks
   ```bash
   helm upgrade zen-flow zen-flow/zen-flow \
     --namespace zen-flow-system \
     --set webhook.enabled=false
   ```

---

## Version Compatibility Matrix

| Controller Version | CRD Version | Compatible | Notes |
|-------------------|-------------|------------|-------|
| 0.0.1-alpha | v1alpha1 | ✅ | Current version |

---

## Getting Help

If you encounter issues during migration:

1. Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
2. Review [CHANGELOG.md](../CHANGELOG.md) for known issues
3. Open an issue: [GitHub Issues](https://github.com/kube-zen/zen-flow/issues)

---

*Last updated: 2015-12-29*

