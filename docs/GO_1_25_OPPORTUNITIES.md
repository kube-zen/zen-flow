# Go 1.25 Opportunities for zen-flow

This document outlines opportunities to leverage Go 1.25 features in the zen-flow project.

## Summary

Go 1.25 introduces several features that can improve code quality, performance, and maintainability in zen-flow.

## 1. WaitGroup.Go Method ⭐ **High Priority**

### Current Pattern
The codebase uses the traditional `WaitGroup` pattern with manual `Add(1)` and `go func()`:

```go
// Current pattern in reconciler.go:1784-1791
var wg sync.WaitGroup
for _, task := range tasks {
    wg.Add(1)
    go func(t refreshTask) {
        defer wg.Done()
        // ... work ...
    }(task)
}
wg.Wait()
```

### Go 1.25 Improvement
The new `WaitGroup.Go` method simplifies this pattern:

```go
// Improved pattern with Go 1.25
var wg sync.WaitGroup
for _, task := range tasks {
    wg.Go(func() {
        // ... work with task ...
    })
}
wg.Wait()
```

### Benefits
- **Reduced boilerplate**: No need for `Add(1)` and `defer wg.Done()`
- **Better readability**: More concise and idiomatic
- **Less error-prone**: Automatic counter management

### Files to Update
- `pkg/controller/reconciler.go:1784-1830` - Parallel step status refresh
- `test/load/load_test.go:160-170` - Load test goroutines
- `test/load/load_test.go:327-340` - Concurrent flow creation

### Implementation Notes
- The `WaitGroup.Go` method handles the counter increment automatically
- The closure captures loop variables correctly (no need for explicit parameter passing)
- This is a **non-breaking change** - can be adopted incrementally

---

## 2. JSON v2 Package (Experimental) ⭐ **Medium Priority**

### Current Usage
The codebase uses `encoding/json` extensively for:
- Job status marshaling (`reconciler.go:1753`)
- Webhook patches (`webhook.go:272`)
- Parameter extraction (`parameters.go:165,183,206`)
- Template processing (`parameter_template.go:58,80,94,116`)

### Go 1.25 Improvement
Enable experimental JSON v2 for better performance:

```bash
# Build with JSON v2
GOEXPERIMENT=jsonv2 go build ./cmd/zen-flow-controller
```

### Benefits
- **10-30% faster** JSON marshaling/unmarshaling
- **Better streaming** for large JSON documents
- **Custom marshaling** for external types

### Considerations
- **Experimental feature**: Requires `GOEXPERIMENT=jsonv2` build flag
- **API compatible**: Same `encoding/json` API, different implementation
- **Testing required**: Verify all JSON operations work correctly

### Recommended Approach
1. Test with `GOEXPERIMENT=jsonv2` in CI
2. Benchmark performance improvements
3. Monitor for any compatibility issues
4. Consider enabling in production builds if stable

---

## 3. Green Tea Garbage Collector (Experimental) ⭐ **Medium Priority**

### Current State
zen-flow is a Kubernetes controller that:
- Processes reconciliation loops continuously
- Manages many small objects (JobFlow statuses, step statuses)
- Performs frequent API calls (creates/updates Jobs)

### Go 1.25 Improvement
Enable experimental Green Tea GC:

```bash
# Build with Green Tea GC
GOEXPERIMENT=greenteagc go build ./cmd/zen-flow-controller
```

### Benefits
- **10-40% reduction** in GC overhead
- **Better performance** for small object allocations
- **Lower latency** for reconciliation loops

### Considerations
- **Experimental feature**: Requires `GOEXPERIMENT=greenteagc` build flag
- **Container-aware**: Works well with Kubernetes resource limits
- **Testing required**: Verify stability under load

### Recommended Approach
1. Test in staging environment with `GOEXPERIMENT=greenteagc`
2. Monitor GC pause times and throughput
3. Compare with standard GC under production-like load
4. Enable in production if stable and beneficial

---

## 4. Container-Aware GOMAXPROCS ⭐ **Automatic**

### Current State
zen-flow runs in Kubernetes containers with CPU limits.

### Go 1.25 Improvement
**Automatic** - No code changes needed!

Go 1.25 automatically detects CPU bandwidth limits from cgroups on Linux and adjusts `GOMAXPROCS` accordingly.

### Benefits
- **Optimal CPU utilization** without manual configuration
- **Respects container limits** automatically
- **Better resource efficiency** in Kubernetes

### Verification
Check that `GOMAXPROCS` is set correctly:
```bash
# In container
go tool pprof http://localhost:8080/debug/pprof/goroutine
```

---

## 5. Enhanced go vet Analyzers ⭐ **Automatic**

### Current State
The project already uses `go vet` in CI.

### Go 1.25 Improvement
**Automatic** - New analyzers catch more issues:

- **waitgroup**: Detects incorrect WaitGroup usage
- **hostport**: Detects common networking mistakes

### Benefits
- **Catches bugs earlier** in development
- **No code changes needed** - just run `go vet`
- **Better CI checks** automatically

### Action Required
- Ensure CI runs `go vet ./...` (already done)
- Review any new warnings from enhanced analyzers

---

## 6. Trace Flight Recorder API ⭐ **Low Priority (Debugging)**

### Use Case
Debug rare reconciliation issues or performance problems.

### Go 1.25 Improvement
Use the new trace flight recorder for lightweight execution tracing:

```go
import "runtime/trace"

// Enable trace recording
recorder := trace.NewFlightRecorder()
recorder.Start()

// ... reconciliation loop ...

// Capture trace on error
if err != nil {
    traceData := recorder.Stop()
    // Save trace data for analysis
}
```

### Benefits
- **Lightweight**: In-memory ring buffer
- **Practical**: Can be enabled in production
- **Debugging**: Helps diagnose rare events

### Recommended Approach
- Consider adding trace recording for error conditions
- Useful for debugging production issues
- Can be enabled via feature flag

---

## Implementation Priority

### High Priority (Immediate)
1. ✅ **WaitGroup.Go** - Simple refactoring, immediate readability improvement
   - Estimated effort: 1-2 hours
   - Risk: Low
   - Benefit: High

### Medium Priority (Testing Required)
2. **JSON v2** - Performance improvement, requires testing
   - Estimated effort: 2-4 hours (testing)
   - Risk: Low (experimental, can disable)
   - Benefit: Medium-High

3. **Green Tea GC** - Performance improvement, requires testing
   - Estimated effort: 2-4 hours (testing)
   - Risk: Low (experimental, can disable)
   - Benefit: Medium-High

### Low Priority (Optional)
4. **Trace Flight Recorder** - Debugging tool
   - Estimated effort: 4-8 hours
   - Risk: Low
   - Benefit: Medium (only when debugging)

### Automatic (No Action Needed)
5. ✅ **Container-aware GOMAXPROCS** - Already working
6. ✅ **Enhanced go vet** - Already benefiting

---

## Recommended Next Steps

1. **Immediate**: Refactor `WaitGroup` usage to use `WaitGroup.Go`
2. **Short-term**: Test JSON v2 and Green Tea GC in CI/staging
3. **Long-term**: Monitor Go 1.25 experimental features for stability
4. **Documentation**: Update build instructions if enabling experiments

---

## References

- [Go 1.25 Release Notes](https://tip.golang.org/doc/go1.25)
- [WaitGroup.Go Documentation](https://pkg.go.dev/sync#WaitGroup.Go)
- [JSON v2 Package](https://pkg.go.dev/encoding/json/v2)
- [Trace Flight Recorder](https://pkg.go.dev/runtime/trace#FlightRecorder)

