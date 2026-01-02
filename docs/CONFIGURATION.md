# Configuration Guide

This document describes how to configure zen-flow controller using environment variables.

## Overview

zen-flow supports configuration via environment variables. All configuration values have sensible defaults, so configuration is optional unless you need to customize behavior.

## Configuration Options

### Resource Limits

| Variable | Default | Description |
|----------|--------|-------------|
| `ZEN_FLOW_DEFAULT_TTL_SECONDS` | `86400` | Default TTL for completed JobFlows (24 hours in seconds) |
| `ZEN_FLOW_CONFIGMAP_SIZE_LIMIT` | `1048576` | Maximum size for ConfigMap artifacts (1MB in bytes) |
| `ZEN_FLOW_UID_TRUNCATE_LENGTH` | `8` | Length to truncate UIDs in resource names |

### Retry and Backoff

| Variable | Default | Description |
|----------|--------|-------------|
| `ZEN_FLOW_DEFAULT_BACKOFF_LIMIT` | `6` | Default backoff limit for Jobs |
| `ZEN_FLOW_DEFAULT_RETRY_LIMIT` | `3` | Default retry limit |
| `ZEN_FLOW_DEFAULT_BACKOFF_BASE` | `1s` | Base duration for exponential backoff |
| `ZEN_FLOW_DEFAULT_BACKOFF_FACTOR` | `2.0` | Multiplier for exponential backoff |

### File Permissions

| Variable | Default | Description |
|----------|--------|-------------|
| `ZEN_FLOW_DEFAULT_DIR_PERM` | `0755` | Default directory permissions (octal) |
| `ZEN_FLOW_DEFAULT_FILE_PERM` | `0644` | Default file permissions (octal) |

### Default Values

| Variable | Default | Description |
|----------|--------|-------------|
| `ZEN_FLOW_DEFAULT_CONFIGMAP_KEY` | `value` | Default key name for ConfigMap values |
| `ZEN_FLOW_DEFAULT_CONTAINER_NAME` | `main` | Default container name in Job templates |
| `ZEN_FLOW_DEFAULT_CONCURRENCY_POLICY` | `Forbid` | Default concurrency policy |
| `ZEN_FLOW_DEFAULT_CONTENT_TYPE` | `application/octet-stream` | Default content type for artifacts |
| `ZEN_FLOW_DEFAULT_ARCHIVE_FORMAT` | `tar` | Default archive format (tar or zip) |
| `ZEN_FLOW_DEFAULT_COMPRESSION` | `none` | Default compression (none or gzip) |

## Usage

### Setting Environment Variables

#### In Deployment Manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zen-flow-controller
spec:
  template:
    spec:
      containers:
      - name: controller
        env:
        - name: ZEN_FLOW_DEFAULT_TTL_SECONDS
          value: "3600"  # 1 hour
        - name: ZEN_FLOW_CONFIGMAP_SIZE_LIMIT
          value: "2097152"  # 2MB
        - name: ZEN_FLOW_DEFAULT_BACKOFF_LIMIT
          value: "10"
```

#### Using ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: zen-flow-config
data:
  ZEN_FLOW_DEFAULT_TTL_SECONDS: "3600"
  ZEN_FLOW_CONFIGMAP_SIZE_LIMIT: "2097152"
  ZEN_FLOW_DEFAULT_BACKOFF_LIMIT: "10"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zen-flow-controller
spec:
  template:
    spec:
      containers:
      - name: controller
        envFrom:
        - configMapRef:
            name: zen-flow-config
```

#### Using Secrets (for sensitive values)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: zen-flow-config
type: Opaque
stringData:
  ZEN_FLOW_DEFAULT_TTL_SECONDS: "3600"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zen-flow-controller
spec:
  template:
    spec:
      containers:
      - name: controller
        envFrom:
        - secretRef:
            name: zen-flow-config
```

## Examples

### Custom TTL for JobFlows

Set a shorter TTL for faster cleanup:

```bash
export ZEN_FLOW_DEFAULT_TTL_SECONDS=3600  # 1 hour
```

### Increase ConfigMap Size Limit

Allow larger artifacts in ConfigMaps:

```bash
export ZEN_FLOW_CONFIGMAP_SIZE_LIMIT=2097152  # 2MB
```

### Custom Backoff Strategy

Configure more aggressive retry strategy:

```bash
export ZEN_FLOW_DEFAULT_BACKOFF_LIMIT=10
export ZEN_FLOW_DEFAULT_BACKOFF_BASE=500ms
export ZEN_FLOW_DEFAULT_BACKOFF_FACTOR=1.5
```

### Custom File Permissions

Set more restrictive file permissions:

```bash
export ZEN_FLOW_DEFAULT_DIR_PERM=0700
export ZEN_FLOW_DEFAULT_FILE_PERM=0600
```

## Validation

Invalid environment variable values will fall back to defaults. For example:

- Invalid integer values → default used
- Invalid duration values → default used
- Invalid file mode values → default used

## Migration from Constants

If you were using the old constants directly in code, you can now use the configuration:

**Old way:**
```go
import "github.com/kube-zen/zen-flow/pkg/controller"

ttl := controller.DefaultTTLSeconds
```

**New way:**
```go
import "github.com/kube-zen/zen-flow/pkg/controller"

ttl := controller.Config.DefaultTTLSeconds
```

The old constants are still available for backward compatibility but are deprecated.

## See Also

- [Resource Requirements](./RESOURCE_REQUIREMENTS.md) - Resource limits and recommendations
- [Performance Tuning](./PERFORMANCE_TUNING.md) - Performance optimization guide
- [Operator Guide](./OPERATOR_GUIDE.md) - Operations and troubleshooting

