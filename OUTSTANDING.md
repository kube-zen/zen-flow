# Outstanding Items for zen-flow

This document tracks outstanding work items based on the P0 analysis and current state.

## ✅ Completed (P0 Functional Correctness & Packaging)

All P0 blockers have been addressed:
- ✅ P0.1: Running steps reconciliation
- ✅ P0.2: Validation failure status persistence
- ✅ P0.3: ContinueOnFailure dependency completion
- ✅ P0.4: DAG cycle detection
- ✅ P0.5: Webhook cert handling
- ✅ P0.6: Webhook configurations
- ✅ P0.7: Helm chart fixes
- ✅ P0.8: Metrics types (gauges vs counters)

## 🔴 Critical Outstanding Items

### Phase C: "Make the API Honest" (P1)

The API defines several features that are **not implemented** in the controller. This creates a mismatch between what users expect and what actually works.

#### Unimplemented API Features:

1. **TTL Cleanup** (`TTLSecondsAfterFinished`)
   - **Status**: Defined in API, not implemented
   - **Impact**: JobFlows and their resources are never cleaned up automatically
   - **Location**: `pkg/api/v1alpha1/types.go:82`
   - **Required**: Controller logic to delete JobFlow and associated resources after TTL expires

2. **Step Retry Policy** (`RetryPolicy`)
   - **Status**: Defined in API, not implemented
   - **Impact**: Steps cannot be retried with configurable backoff
   - **Location**: `pkg/api/v1alpha1/types.go:149-150`
   - **Required**: Controller logic to retry failed steps based on RetryPolicy configuration

3. **Step Timeouts** (`TimeoutSeconds`)
   - **Status**: Defined in API, not implemented
   - **Impact**: Steps can run indefinitely
   - **Location**: `pkg/api/v1alpha1/types.go:152-153`
   - **Required**: Controller logic to enforce step-level timeouts

4. **Concurrency Policy** (`ConcurrencyPolicy`)
   - **Status**: Defined in API, not implemented
   - **Impact**: Multiple JobFlow instances can run concurrently even if policy says otherwise
   - **Location**: `pkg/api/v1alpha1/types.go:76`
   - **Required**: Controller logic to enforce Allow/Forbid/Replace policies

5. **Flow-Level Backoff Limit** (`BackoffLimit`)
   - **Status**: Defined in API, not implemented
   - **Impact**: No limit on flow-level retries
   - **Location**: `pkg/api/v1alpha1/types.go:87-90`
   - **Required**: Controller logic to track and enforce flow-level retry limits

6. **Flow-Level Active Deadline** (`ActiveDeadlineSeconds`)
   - **Status**: Defined in API, not implemented
   - **Impact**: Flows can run indefinitely
   - **Location**: `pkg/api/v1alpha1/types.go:92-94`
   - **Required**: Controller logic to enforce flow-level timeouts

7. **Pod Failure Policy** (`PodFailurePolicy`)
   - **Status**: Defined in API, not implemented
   - **Impact**: Cannot handle pod failures based on exit codes
   - **Location**: `pkg/api/v1alpha1/types.go:84-85, 97-126`
   - **Required**: Controller logic to evaluate pod failure policies

8. **When Conditions** (`When`)
   - **Status**: Defined in API, not implemented
   - **Impact**: Cannot conditionally execute steps
   - **Location**: `pkg/api/v1alpha1/types.go:155-157`
   - **Required**: Template evaluation engine for step conditions

9. **Artifacts** (`Inputs.Artifacts`, `Outputs.Artifacts`)
   - **Status**: Defined in API, not implemented
   - **Impact**: Cannot pass artifacts between steps
   - **Location**: `pkg/api/v1alpha1/types.go:166-170, 224-255`
   - **Required**: Artifact management system (S3, HTTP, PVC-based)

10. **Parameters** (`Inputs.Parameters`, `Outputs.Parameters`)
    - **Status**: Defined in API, not implemented
    - **Impact**: Cannot pass parameters between steps
    - **Location**: `pkg/api/v1alpha1/types.go:166-170, 295-313`
    - **Required**: Parameter extraction and substitution system

#### Recommendation:

**Option 1**: Implement all advertised features (significant effort, 2-3 months)
- Pros: Full feature parity with API
- Cons: Large implementation effort, complexity

**Option 2**: Reduce API surface to match implementation (recommended for MVP)
- Remove or mark as "not implemented" the above features
- Document what's actually supported
- Re-expand API later as features are implemented
- Pros: Honest API, less user confusion
- Cons: Breaking changes if features are already in use

## 🟡 Important Outstanding Items

### Security: RBAC Scope Tightening

**Current RBAC** (`deploy/manifests/rbac.yaml`):
- Includes `create`, `delete` on `jobflows` - controller doesn't use these
- Includes `delete` on `jobs` - may not be needed
- Includes `create`, `update`, `patch` on `pods` - only needs `get`, `list`, `watch`

**Recommendation**:
- Tighten RBAC to minimum required permissions
- Add RBAC justification comments
- Consider adding CI check to validate RBAC against actual API calls

### Test Improvements

**Current State**:
- E2E tests exist but are mostly structural
- Don't fully simulate: job completion → controller updates step status → downstream steps start → flow completes

**Recommendation**:
- Add deterministic controller test harness using fake clients
- Test full lifecycle: create JobFlow → reconcile → Job created → Job succeeds → step status updated → next steps start → flow completes
- Add tests for edge cases (job deletion, job failure, etc.)

### Go Toolchain Constraint

**Current**: `go.mod` requires Go 1.24
- **Issue**: Go 1.24 doesn't exist yet (as of 2025, latest is Go 1.23)
- **Impact**: May cause issues in constrained/airgapped environments
- **Recommendation**: Verify if this is intentional or a typo, update to supported Go version

## 🟢 Nice-to-Have Items

### Design Refactor (Optional but Strategic)

**Current**: Raw client-go patterns + custom webhook server
**Option**: Migrate to controller-runtime
- Pros: Faster iteration, fewer edge-case regressions, built-in conveniences (predicates, status patching, finalizers)
- Cons: Significant refactoring effort
- **Recommendation**: Consider after MVP stabilization

### Documentation Updates

- Update API reference to clearly mark unimplemented features
- Add migration guide if reducing API surface
- Document RBAC requirements and justification

## 📊 Priority Summary

1. **P1 (High)**: Address API surface vs implementation gap (Phase C)
2. **P2 (Medium)**: Tighten RBAC scope
3. **P2 (Medium)**: Improve E2E tests
4. **P3 (Low)**: Fix Go version constraint
5. **P3 (Low)**: Consider controller-runtime migration

## Next Steps

1. **Decision Point**: Choose Option 1 or Option 2 for API surface
2. **If Option 2**: Create PR to remove/mark unimplemented features
3. **If Option 1**: Create implementation plan for each feature
4. **RBAC**: Audit and tighten permissions
5. **Tests**: Enhance E2E test coverage

