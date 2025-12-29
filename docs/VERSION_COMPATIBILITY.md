# Version Compatibility Matrix

This document provides version compatibility information for zen-flow, including supported Kubernetes versions.

## Table of Contents

- [Supported Kubernetes Versions](#supported-kubernetes-versions)

---

## Supported Kubernetes Versions

### Compatibility Matrix

| zen-flow Version | Kubernetes 1.24 | Kubernetes 1.25 | Kubernetes 1.26 | Kubernetes 1.27 | Kubernetes 1.28 | Kubernetes 1.29+ |
|------------------|-----------------|-----------------|-----------------|-----------------|-----------------|------------------|
| 0.0.1-alpha      | ✅              | ✅              | ✅              | ✅              | ✅              | ✅               |
| 0.1.x            | ✅              | ✅              | ✅              | ✅              | ✅              | ✅               |
| 1.0.x            | ❌              | ✅              | ✅              | ✅              | ✅              | ✅               |

**Legend:**
- ✅ Fully supported
- ⚠️ Supported with limitations
- ❌ Not supported

### Minimum Requirements

- **Kubernetes**: 1.24+ (required for Job API features)
- **CRD API**: v1 (apiextensions.k8s.io/v1)
- **RBAC**: v1 (rbac.authorization.k8s.io/v1)

### Tested Versions

The following Kubernetes versions are regularly tested:

- 1.24.x (EKS, GKE, AKS)
- 1.25.x (EKS, GKE, AKS)
- 1.26.x (EKS, GKE, AKS)
- 1.27.x (EKS, GKE, AKS)
- 1.28.x (EKS, GKE, AKS)
- 1.29.x (EKS, GKE, AKS)

### Version Support Policy

- **Current Version**: Fully supported
- **Previous Minor Version**: Supported with bug fixes
- **Older Versions**: Best effort support

---

## CRD Version History

| Version | Status | Kubernetes Version | Notes |
|---------|--------|-------------------|-------|
| v1alpha1 | ✅ Current | 1.24+ | Current release |
| v1beta1 | 🔜 Planned | TBD | API stabilized, no breaking changes |
| v1 | 🔜 Planned | TBD | Stable API, long-term support |

---

## See Also

- [API Reference](API_REFERENCE.md) - Complete API documentation
- [User Guide](USER_GUIDE.md) - User guide

