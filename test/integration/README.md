# Integration Tests for zen-flow

This directory contains integration tests for zen-flow that validate controller behavior in a real Kubernetes environment.

## Test Structure

- `integration_test.go` - Core integration tests using fake clients (fast, no cluster required)
- `jobflow_lifecycle_test.go` - Full JobFlow lifecycle tests (requires cluster)
- `dag_execution_test.go` - DAG execution and step dependencies tests
- `experimental_features_test.go` - Experimental Go 1.25 features performance tests
- `setup_kind.sh` - Script to set up kind cluster for integration tests

## Quick Start

### Prerequisites

1. Kubernetes cluster (kind recommended)
2. kubectl configured
3. zen-flow controller deployed (or use `setup_kind.sh`)

### Running Tests

#### Fast Integration Tests (No Cluster Required)

These tests use fake clients and run quickly:

```bash
# Run all integration tests with fake clients
go test -tags=integration -v ./test/integration/ -run TestJobFlowReconciler

# Run specific test
go test -tags=integration -v ./test/integration/ -run TestJobFlowCRUD
```

#### Full Integration Tests (Requires Cluster)

These tests require a real Kubernetes cluster:

```bash
# Set up kind cluster
cd test/integration
./setup_kind.sh create

# Export kubeconfig
export KUBECONFIG=$(./setup_kind.sh kubeconfig)

# Run cluster-based tests
go test -tags=integration,cluster -v ./test/integration/ -run TestJobFlowLifecycle

# Cleanup
./setup_kind.sh delete
```

### Test Categories

#### 1. Fast Integration Tests (Fake Clients)

- `TestJobFlowReconciler_Integration` - Basic reconciler functionality
- `TestJobFlowCRUD_Integration` - CRUD operations
- `TestStepStatusUpdate_Integration` - Status update logic
- `TestDAGComputation_Integration` - DAG computation logic

**Build Tag:** `integration`  
**Cluster Required:** No  
**Speed:** Fast (< 1 second per test)

#### 2. Cluster Integration Tests

- `TestJobFlowLifecycle_Integration` - Full JobFlow lifecycle
- `TestDAGExecution_Integration` - DAG execution with real Jobs
- `TestStepDependencies_Integration` - Step dependency resolution
- `TestParallelSteps_Integration` - Parallel step execution
- `TestRetryLogic_Integration` - Retry and failure handling

**Build Tag:** `integration,cluster`  
**Cluster Required:** Yes  
**Speed:** Moderate (5-30 seconds per test)

#### 3. Experimental Features Tests

- `TestExperimentalFeatures_JSONv2` - JSON v2 performance
- `TestExperimentalFeatures_GreenTeaGC` - Green Tea GC performance
- `TestExperimentalFeatures_Combined` - Combined features

**Build Tag:** `integration,experimental`  
**Cluster Required:** No (uses fake clients)  
**Speed:** Fast (benchmark-style tests)

## Test Parameters

Some tests support environment variables for configuration:

| Parameter | Env Var | Default | Description |
|-----------|---------|---------|-------------|
| Test Namespace | `TEST_NAMESPACE` | `zen-flow-test` | Namespace for test resources |
| Test Timeout | `TEST_TIMEOUT` | `5m` | Test timeout duration |
| Cleanup Resources | `TEST_CLEANUP` | `true` | Whether to cleanup test resources |

## Setup Script

The `setup_kind.sh` script automates cluster setup:

```bash
# Create cluster and deploy zen-flow
./setup_kind.sh create

# Delete cluster
./setup_kind.sh delete

# Get kubeconfig path
./setup_kind.sh kubeconfig
```

## Test Coverage

### Core Functionality
- ✅ JobFlow CRUD operations
- ✅ Step status updates
- ✅ DAG computation
- ✅ Job creation and management
- ✅ Status reconciliation

### Advanced Features
- ✅ Step dependencies
- ✅ Parallel execution
- ✅ Retry logic
- ✅ Timeout handling
- ✅ ContinueOnFailure
- ✅ When conditions

### Edge Cases
- ✅ Job deletion handling
- ✅ Step failure propagation
- ✅ DAG cycle detection
- ✅ Invalid step references

## CI Integration

Integration tests can be run in CI:

```yaml
# Example GitHub Actions step
- name: Run Integration Tests
  run: |
    cd test/integration
    ./setup_kind.sh create
    export KUBECONFIG=$(./setup_kind.sh kubeconfig)
    go test -tags=integration,cluster -v ./test/integration/ -timeout=10m
  env:
    TEST_NAMESPACE: zen-flow-test
    TEST_CLEANUP: true
```

## Troubleshooting

### Tests Fail with "cluster not found"

**Solution:** Run `./setup_kind.sh create` first

### Tests Timeout

**Solution:** Increase timeout: `go test -timeout=10m ...`

### Resource Cleanup Issues

**Solution:** Manually cleanup: `kubectl delete namespace zen-flow-test`

## See Also

- [E2E Tests](../e2e/README.md) - End-to-end tests
- [Load Tests](../load/README.md) - Load and performance tests
- [Experimental Features Tests](experimental_features_test.go) - Go 1.25 features

