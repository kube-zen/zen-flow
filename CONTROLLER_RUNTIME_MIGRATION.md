# Controller-Runtime Migration Plan

This document tracks the migration from raw client-go to controller-runtime framework.

## Benefits

- Built-in conveniences (predicates, status patching, finalizers)
- Structured reconciliation contracts
- Faster iteration and fewer edge-case regressions
- Better testability with envtest
- Standard patterns used across Kubernetes ecosystem

## Migration Steps

### Phase 1: Dependencies and Setup
- [x] Add controller-runtime dependencies
- [ ] Update go.mod with controller-runtime packages
- [ ] Add kubebuilder markers for CRD generation

### Phase 2: Controller Refactoring
- [ ] Create new Reconciler interface implementation
- [ ] Migrate to manager.Manager pattern
- [ ] Replace informers with controller-runtime cache
- [ ] Update event handling to use controller-runtime events
- [ ] Migrate status updates to use StatusWriter

### Phase 3: Main Function Refactoring
- [ ] Update main.go to use manager setup
- [ ] Migrate leader election to controller-runtime
- [ ] Update webhook server integration
- [ ] Update metrics server integration

### Phase 4: Testing Updates
- [ ] Update unit tests for controller-runtime
- [ ] Add envtest integration tests
- [ ] Update E2E tests

### Phase 5: Feature Implementation
- [ ] Implement missing API features on new foundation
- [ ] Add finalizers for cleanup
- [ ] Implement TTL cleanup
- [ ] Implement retry policies
- [ ] Implement timeouts
- [ ] Implement concurrency policies

## Implementation Notes

### Key Changes

1. **Reconciler Interface**: Implement `reconcile.Reconciler` instead of custom reconciliation
2. **Manager**: Use `manager.Manager` to coordinate controllers, caches, and clients
3. **Builder**: Use `controller.NewControllerManagedBy()` for controller setup
4. **Cache**: Use manager's cache instead of manual informers
5. **Client**: Use manager's client (with caching) instead of raw clients

### Code Structure

```
pkg/controller/
  ├── reconciler.go          # New: Reconciler implementation
  ├── controller.go          # Updated: Controller setup with builder
  ├── manager.go             # New: Manager setup
  └── ... (existing files)
```

## Progress Tracking

- [ ] Phase 1: Dependencies
- [ ] Phase 2: Controller Refactoring  
- [ ] Phase 3: Main Function
- [ ] Phase 4: Testing
- [ ] Phase 5: Features

