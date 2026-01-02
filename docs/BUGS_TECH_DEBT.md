# Bugs, Tech Debt, and Hardcoded Items Report

**Generated**: 2026-01-02  
**Scope**: zen-flow codebase analysis

## 🔴 Critical Bugs

### 1. Potential Panic: UID String Slicing
**Location**: `pkg/controller/reconciler.go:699`
```go
job.Name = fmt.Sprintf("%s-%s-%s", jobFlow.Name, step.Name, string(jobFlow.UID)[:8])
```
**Issue**: If `jobFlow.UID` is empty or shorter than 8 characters, this will panic.
**Fix**: Add length check before slicing.

### 2. Duplicate API Server Call Metrics
**Location**: `pkg/controller/reconciler.go:127, 131, 139`
**Issue**: `RecordAPIServerCall` is called multiple times for the same `Get` operation.
**Fix**: Remove duplicate calls, keep only one per operation.

### 3. Missing Error Type Check
**Location**: `pkg/controller/artifact_copy.go:110`
**Issue**: `r.Get` error is checked but `k8serrors.IsNotFound` is not explicitly checked before creating ConfigMap.
**Fix**: Explicitly check for `IsNotFound` before creating.

## 🟡 Tech Debt

### 1. TODO Comments
- `pkg/controller/parameters.go:63` - JSONPath step name support
- `pkg/controller/parameters.go:138` - Full JSONPath evaluation (partially implemented)

### 2. Placeholder Comments
- `pkg/controller/reconciler.go:1505` - "Currently a placeholder - actual artifact/parameter handling can be enhanced"
- `pkg/controller/reconciler.go:1597` - "Store outputs in step status (placeholder)"

### 3. Test Stubs
- `test/integration/integration_test.go` - Multiple `panic("not implemented")` stubs
- `pkg/controller/reconciler_test.go` - Multiple `panic("not implemented")` stubs

## 🟠 Hardcoded Values

### 1. Magic Numbers
| Value | Location | Description | Recommendation |
|-------|----------|-------------|----------------|
| `86400` | `reconciler.go:1065` | Default TTL (24 hours) | Extract to constant |
| `1024*1024` | `artifact_copy.go:97` | ConfigMap size limit (1MB) | Extract to constant |
| `0755` | Multiple files | Directory permissions | Extract to constant |
| `0644` | Multiple files | File permissions | Extract to constant |
| `8` | `reconciler.go:699` | UID truncation length | Extract to constant |
| `6` | `reconciler.go:1144` | Default backoff limit | Extract to constant |
| `3` | `reconciler.go:1262` | Default retry limit | Extract to constant |
| `1s` | `reconciler.go:1306` | Default backoff duration | Extract to constant |
| `2.0` | `reconciler.go:1322` | Default backoff factor | Extract to constant |

### 2. Hardcoded Strings
| Value | Location | Description | Recommendation |
|-------|----------|-------------|----------------|
| `"value"` | `artifacts.go:266`, `parameters.go:83` | Default ConfigMap/Secret key | Extract to constant |
| `"main"` | `reconciler.go:1381` | Default container name | Extract to constant |
| `"Forbid"` | `reconciler.go:1085` | Default concurrency policy | Extract to constant |
| `"application/octet-stream"` | `artifacts.go:234` | Default content type | Extract to constant |
| `"tar"` | `archive.go:40` | Default archive format | Extract to constant |
| `"none"` | `archive.go:45` | Default compression | Extract to constant |

### 3. Hardcoded Defaults
- Default retry policies
- Default backoff strategies
- Default timeouts
- Default resource limits

## 📋 Recommendations

### High Priority
1. **Fix UID slicing panic** - Add length validation
2. **Remove duplicate metrics calls** - Clean up API server call tracking
3. **Extract magic numbers to constants** - Create a `constants.go` file
4. **Extract hardcoded strings to constants** - Improve maintainability

### Medium Priority
1. **Complete TODO items** - Implement JSONPath step name support
2. **Remove placeholder comments** - Update or remove outdated comments
3. **Implement test stubs** - Complete integration test implementations

### Low Priority
1. **Create configuration package** - Move defaults to configurable constants
2. **Add validation for hardcoded limits** - Make limits configurable via environment variables

## 🔧 Proposed Fixes

### 1. Create Constants File
```go
// pkg/controller/constants.go
package controller

const (
    // Default TTL (24 hours)
    DefaultTTLSeconds int32 = 86400
    
    // ConfigMap size limit (1MB)
    ConfigMapSizeLimit = 1024 * 1024
    
    // File permissions
    DefaultDirPerm  = 0755
    DefaultFilePerm = 0644
    
    // UID truncation length
    UIDTruncateLength = 8
    
    // Default retry/backoff values
    DefaultBackoffLimit = 6
    DefaultRetryLimit   = 3
    DefaultBackoffBase  = 1 * time.Second
    DefaultBackoffFactor = 2.0
    
    // Default strings
    DefaultConfigMapKey = "value"
    DefaultContainerName = "main"
    DefaultConcurrencyPolicy = "Forbid"
    DefaultContentType = "application/octet-stream"
    DefaultArchiveFormat = "tar"
    DefaultCompression = "none"
)
```

### 2. Fix UID Slicing
```go
// Safe UID truncation
uidStr := string(jobFlow.UID)
if len(uidStr) < UIDTruncateLength {
    uidStr = uidStr // Use full UID if too short
} else {
    uidStr = uidStr[:UIDTruncateLength]
}
job.Name = fmt.Sprintf("%s-%s-%s", jobFlow.Name, step.Name, uidStr)
```

### 3. Fix Duplicate Metrics
Remove duplicate `RecordAPIServerCall` calls, keep only one per operation.

