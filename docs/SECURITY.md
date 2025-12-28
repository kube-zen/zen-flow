# Security Best Practices

This document provides comprehensive security guidance for deploying and operating zen-flow.

## Table of Contents

- [Pod Security Standards](#pod-security-standards)
- [Network Policies](#network-policies)
- [RBAC Best Practices](#rbac-best-practices)
- [Secrets Management](#secrets-management)
- [Audit Logging](#audit-logging)
- [Security Checklist](#security-checklist)

---

## Pod Security Standards

zen-flow follows Kubernetes Pod Security Standards to ensure secure deployments.

### Recommended Security Context

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65534  # nobody
  fsGroup: 65534
  seccompProfile:
    type: RuntimeDefault
  capabilities:
    drop:
      - ALL
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
```

### Pod Security Standards Compliance

zen-flow is compliant with **Restricted** Pod Security Standard:

- ✅ Runs as non-root user
- ✅ No privileged containers
- ✅ No host network access
- ✅ No host PID/IPC namespaces
- ✅ Read-only root filesystem
- ✅ No capabilities granted
- ✅ Seccomp profile enforced

---

## Network Policies

Network policies restrict network access to the controller, following the principle of least privilege.

### Recommended Network Policy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: zen-flow-controller-netpol
  namespace: zen-flow-system
spec:
  podSelector:
    matchLabels:
      app: zen-flow-controller
  policyTypes:
  - Ingress
  - Egress
  
  # Allow ingress from Prometheus (metrics scraping)
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
    ports:
    - protocol: TCP
      port: 8080
  
  # Allow egress to API server and DNS
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: UDP
      port: 53
```

---

## RBAC Best Practices

### Minimal Permissions

The controller requires:
- Full access to `jobflows.workflow.zen.io` CRD
- Create/update/delete Jobs
- Create/update/delete PVCs and ConfigMaps
- Leader election permissions

See [RBAC.md](RBAC.md) for complete RBAC documentation.

### Namespace Scoping

For multi-tenant environments, consider namespace-scoped RBAC:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: zen-flow-controller
  namespace: tenant-namespace
rules:
- apiGroups: ["workflow.zen.io"]
  resources: ["jobflows"]
  verbs: ["*"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["create", "get", "list", "watch", "update", "patch", "delete"]
```

---

## Secrets Management

### Job Templates with Secrets

Use Kubernetes secrets in Job templates:

```yaml
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
                image: myapp:latest
                env:
                  - name: API_KEY
                    valueFrom:
                      secretKeyRef:
                        name: api-secret
                        key: api-key
```

### External Secret Managers

For production, consider:
- External Secrets Operator
- Sealed Secrets
- Vault integration

---

## Audit Logging

Enable Kubernetes audit logging to track:
- JobFlow creation/updates
- Job creation/deletion
- Status updates

### Audit Policy Example

```yaml
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Metadata
  resources:
  - group: workflow.zen.io
    resources: ["jobflows"]
- level: RequestResponse
  resources:
  - group: batch
    resources: ["jobs"]
```

---

## Security Checklist

Before deploying:

- [ ] RBAC permissions reviewed and minimized
- [ ] Security context configured (non-root, read-only filesystem)
- [ ] Network policies applied
- [ ] Secrets properly managed
- [ ] Dependencies scanned for vulnerabilities
- [ ] Audit logging enabled
- [ ] Webhooks enabled for validation
- [ ] Resource quotas configured
- [ ] Monitoring and alerting configured

---

## Vulnerability Scanning

### Container Images

Scan container images regularly:

```bash
trivy image kube-zen/zen-flow-controller:latest
```

### Dependencies

Scan Go dependencies:

```bash
govulncheck ./...
gosec ./...
```

---

## See Also

- [RBAC.md](RBAC.md) - RBAC permissions
- [Operator Guide](OPERATOR_GUIDE.md) - Operations guide
- [Security Policy](../SECURITY.md) - Security vulnerability reporting

