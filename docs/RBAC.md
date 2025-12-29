# RBAC Permissions Documentation

This document explains all RBAC permissions required by the zen-flow controller and why each permission is necessary.

## Overview

The zen-flow controller requires permissions to:
1. Manage `JobFlow` CRDs
2. Create and manage Kubernetes Jobs
3. Create and manage PVCs and ConfigMaps
4. Perform leader election
5. Emit Kubernetes events

## Required Permissions

### 1. JobFlow CRD Permissions

```yaml
- apiGroups:
    - workflow.kube-zen.io
  resources:
    - jobflows
  verbs:
    - get
    - list
    - watch
    - delete
- apiGroups:
    - workflow.kube-zen.io
  resources:
    - jobflows/status
  verbs:
    - get
    - update
    - patch
```

**Why Required:**
- **get, list, watch**: Controller needs to discover and monitor JobFlows
- **delete**: Controller deletes JobFlows for TTL cleanup and concurrency policy Replace
- **jobflows/status**: Controller updates JobFlow status subresource (read-only access to spec)

### 2. Job Permissions

```yaml
- apiGroups:
    - batch
  resources:
    - jobs
  verbs:
    - get
    - list
    - watch
    - create
    - delete
```

**Why Required:**
- **get, list, watch**: Controller monitors Job status
- **create**: Controller creates Jobs from step templates
- **delete**: Controller deletes Jobs when steps timeout or for cleanup

### 3. PVC Permissions

```yaml
- apiGroups:
    - ""
  resources:
    - persistentvolumeclaims
  verbs:
    - get
    - list
    - watch
    - create
```

**Why Required:**
- Controller creates PVCs from `resourceTemplates.volumeClaimTemplates`
- Controller monitors PVC status (read-only after creation)

### 4. ConfigMap Permissions

```yaml
- apiGroups:
    - ""
  resources:
    - configmaps
  verbs:
    - get
    - list
    - watch
    - create
```

**Why Required:**
- Controller creates ConfigMaps from `resourceTemplates.configMapTemplates`
- Controller monitors ConfigMap status (read-only after creation)

### 5. Event Permissions

```yaml
- apiGroups:
    - ""
  resources:
    - events
  verbs:
    - create
    - patch
```

**Why Required:**
- Controller emits events for JobFlow and step status changes

### 6. Leader Election Permissions

```yaml
- apiGroups:
    - coordination.k8s.io
  resources:
    - leases
  verbs:
    - get
    - list
    - watch
    - create
    - update
    - patch
```

**Why Required:**
- Controller uses leader election for HA
- Only leader processes JobFlows

## Security Considerations

### Cluster-Scoped vs Namespace-Scoped

The default RBAC is **cluster-scoped** because:
- JobFlows can create Jobs in any namespace
- Controller needs to watch Jobs across namespaces

### Namespace-Scoped Alternative

For multi-tenant environments, you can use namespace-scoped RBAC:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: zen-flow-controller
  namespace: tenant-namespace
rules:
- apiGroups: ["workflow.kube-zen.io"]
  resources: ["jobflows"]
  verbs: ["*"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["create", "get", "list", "watch", "update", "patch", "delete"]
```

Deploy separate controller instances per namespace/tenant.

## Mitigation Strategies

1. **Webhook Validation**: Use validating webhooks to restrict JobFlow creation
2. **Resource Quotas**: Set quotas on Jobs, PVCs, ConfigMaps
3. **Network Policies**: Restrict network access
4. **Audit Logging**: Track all operations
5. **Monitoring**: Alert on unusual activity

## See Also

- [Security](SECURITY.md) - Security best practices
- [Operator Guide](OPERATOR_GUIDE.md) - Operations guide

