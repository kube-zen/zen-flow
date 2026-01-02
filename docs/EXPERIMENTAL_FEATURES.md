# Experimental Go 1.25 Features

**Status:** Opt-in, Enabled by Default  
**Warning:** Experimental features are not production-ready and may have stability issues.

## Overview

zen-flow **default images include experimental Go 1.25 features** (`jsonv2`, `greenteagc`) for better performance.

**Default Behavior:** All images built with `make build-image` or standard `docker build` include experimental features. To build GA-only images, use `make build-image-no-experimental` or `docker build --build-arg GOEXPERIMENT=""`.

## Available Experimental Features

### 1. JSON v2 (`jsonv2`)

**Impact:** 2-3x faster JSON serialization/deserialization  
**Use Case:** High-throughput reconciliation, Kubernetes API operations, parameter/artifact processing  
**Status:** Experimental

**Benefits:**
- Faster Kubernetes resource serialization (JobFlow status updates)
- Reduced reconciliation latency
- Lower CPU usage for JSON operations (parameter extraction, template processing)
- Better performance for artifact handling

**Risks:**
- Experimental API may change
- Potential compatibility issues
- Not production-ready

**Relevant Code:**
- `pkg/controller/reconciler.go:1753` - JobFlow status marshaling
- `pkg/controller/parameters.go:165,183,206` - Parameter JSON extraction
- `pkg/controller/parameter_template.go:58,80,94,116` - Template JSON processing
- `pkg/webhook/webhook.go:272` - Webhook patch marshaling

### 2. Green Tea GC (`greenteagc`)

**Impact:** 10-40% reduction in GC overhead, lower pause times  
**Use Case:** High-frequency reconciliations, parallel step status refresh, low-latency operations  
**Status:** Experimental

**Benefits:**
- Reduced GC pause times (important for reconciliation loops)
- Lower latency for parallel operations (step status refresh)
- Better memory efficiency (DAG caching, status updates)
- Improved performance under load

**Risks:**
- Experimental implementation
- Potential memory leaks
- Not production-ready

**Relevant Code:**
- `pkg/controller/reconciler.go:1784-1827` - Parallel step status refresh
- `pkg/controller/reconciler.go:60-77` - DAG caching with frequent allocations
- Continuous reconciliation loops

## Building Images

### Default Build (Builds Both Variants)

**Default behavior:** `make build-image` builds both GA-only and experimental variants.

```bash
# Build both variants
make build-image
```

**Result:** Creates two image tags:
- `kubezen/zen-flow-controller:<version>` (GA-only, default)
- `kubezen/zen-flow-controller:<version>-experimental` (experimental, opt-in)

### Build GA-Only Variant (Default)

```bash
# Build GA-only variant only
make build-image-ga-only

# Or directly
docker build --build-arg GOEXPERIMENT="" -t kubezen/zen-flow-controller:latest .
```

**Result:** Creates GA-only image with tags: `<version>`, `latest`, `<version>-ga-only`

### Build Experimental Variant (Opt-in)

```bash
# Build experimental variant only
make build-image-experimental

# Or directly
docker build --build-arg GOEXPERIMENT=jsonv2,greenteagc -t kubezen/zen-flow-controller:<version>-experimental .
```

**Result:** Creates experimental image with tag: `<version>-experimental`

**Use Case:** Performance-critical deployments where 15-25% improvement is valuable.

### Build With Specific Experiments

```bash
# Only JSON v2
docker build --build-arg GOEXPERIMENT=jsonv2 -t kubezen/zen-flow-controller:jsonv2 .

# Only Green Tea GC
docker build --build-arg GOEXPERIMENT=greenteagc -t kubezen/zen-flow-controller:greenteagc .

# Both (default)
docker build --build-arg GOEXPERIMENT=jsonv2,greenteagc -t kubezen/zen-flow-controller:latest .
```

## Local Development

### Build Binary with Experimental Features

```bash
# Build with experimental features
GOEXPERIMENT=jsonv2,greenteagc make build-release

# Build GA-only
make build-release-ga
```

### Testing Experimental Features

```bash
# Run tests with experimental features
GOEXPERIMENT=jsonv2,greenteagc go test ./...

# Run integration tests
GOEXPERIMENT=jsonv2,greenteagc go test -tags=integration ./test/integration/...
```

## Performance Expectations

### JSON v2

- **Expected Improvement:** 2-3x faster JSON operations
- **Observed Impact (from zen-lead):**
  - Kubernetes API serialization: 15-25% faster
  - Metrics export: Improved throughput
- **Relevant Operations in zen-flow:**
  - JobFlow status updates (frequent)
  - Parameter extraction from Job status
  - Template processing
  - Webhook patches

### Green Tea GC

- **Expected Improvement:** 10-40% reduction in GC overhead
- **Observed Impact (from zen-lead):**
  - Failover latency: 5-15% improvement
  - Memory efficiency: Improved allocation patterns
  - CPU usage: Lower overhead
- **Relevant Operations in zen-flow:**
  - Parallel step status refresh (many goroutines)
  - DAG computation and caching
  - Continuous reconciliation loops

### Combined Impact

- **Overall Performance:** Expected 15-25% improvement (based on zen-lead test results)
- **Stability:** To be validated through testing
- **Recommendation:** Safe for staging/testing; consider for production with monitoring

## Integration Testing

### Running Comparison Tests

```bash
# Build standard image
docker build --build-arg GOEXPERIMENT="" -t kubezen/zen-flow-controller:standard .

# Build experimental image
docker build --build-arg GOEXPERIMENT=jsonv2,greenteagc -t kubezen/zen-flow-controller:experimental .

# Deploy both versions and run comparison tests
# (See test scripts for detailed comparison)
```

### Metrics to Compare

1. **Reconciliation Latency:**
   - `zen_flow_reconciliation_duration_seconds`
   - Compare p50, p95, p99 between standard and experimental

2. **Step Status Refresh Time:**
   - Parallel refresh operations
   - Measure time for multiple step status updates

3. **GC Performance:**
   - GC pause times (via Go runtime metrics)
   - Memory allocation rates
   - GC frequency

4. **JSON Processing Time:**
   - Parameter extraction latency
   - Template processing time
   - Status update serialization

5. **Stability:**
   - Error rates
   - Memory leaks (long-running tests)
   - Crash frequency

## Recommendations

### For Production

**Status:** Experimental features show promising performance improvements. However, they remain experimental and should be used with caution in production until promoted to GA.

**Conservative Approach:** Use GA features only (`make build-image-no-experimental`)  
**Aggressive Approach:** Consider enabling in production with close monitoring if performance gains are critical

### For Staging/Testing

**✅ Recommended:** Enable experimental features in staging environments to benefit from performance improvements while monitoring for any issues.

### For Development

- Use experimental features for local development and testing
- Evaluate performance improvements
- Report issues to Go team if found

## Troubleshooting

### Feature Not Working

**Symptom:** No performance improvement observed

**Cause:** Binary not built with GOEXPERIMENT flag

**Solution:** Rebuild image with `--build-arg GOEXPERIMENT=jsonv2,greenteagc`

### Build Fails

**Symptom:** Docker build fails with GOEXPERIMENT

**Cause:** Invalid experiment name or Go version mismatch

**Solution:** 
- Verify Go 1.25+ is used
- Check experiment names: `jsonv2`, `greenteagc`
- Ensure comma-separated format: `jsonv2,greenteagc`

### Runtime Errors

**Symptom:** Controller crashes or errors with experimental features

**Cause:** Experimental feature incompatibility

**Solution:**
- Disable experimental features
- Report issue to Go team
- Use standard build for production

## References

- [Go 1.25 Release Notes](https://tip.golang.org/doc/go1.25)
- [Go Experiments](https://pkg.go.dev/internal/goexperiment)
- [JSON v2 Package](https://pkg.go.dev/encoding/json/v2) (experimental)
- [Green Tea GC](https://tip.golang.org/doc/go1.25#gc) (experimental)
- [Go 1.25 Opportunities](GO_1_25_OPPORTUNITIES.md) - Detailed feature analysis

