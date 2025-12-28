# Testing Guide

This document describes the testing infrastructure and how to run tests for zen-flow.

## Overview

zen-flow has three types of tests:

1. **Unit Tests**: Fast, isolated tests for individual components
2. **Integration Tests**: Tests that verify component interactions using fake clients
3. **E2E Tests**: End-to-end tests that require a real Kubernetes cluster

## Unit Tests

Unit tests are located in `pkg/*/*_test.go` files and test individual functions and components in isolation.

### Running Unit Tests

```bash
# Run all unit tests
make test-unit

# Run tests for a specific package
go test -v ./pkg/controller/...

# Run tests with race detection
go test -race ./pkg/...

# Run tests with coverage
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out
```

### Coverage Requirements

- **Minimum**: 75% code coverage
- **Target**: >80% coverage
- **Critical paths**: >85% coverage

Coverage is checked automatically in CI and will fail if below 75%.

## Test Structure

### Package Coverage

- `pkg/controller/dag`: 100% coverage ✅
- `pkg/errors`: 79.2% coverage ✅
- `pkg/logging`: 58.9% coverage (improving)
- `pkg/controller`: 17.1% coverage (needs improvement)
- `pkg/controller/metrics`: 0% coverage (needs tests)

## Integration Tests

Integration tests are located in `test/integration/` and test component interactions using fake Kubernetes clients.

### Test Coverage

Integration tests cover:

- ✅ Controller startup and shutdown
- ✅ JobFlow CRUD operations
- ✅ Step execution
- ✅ DAG execution
- ✅ Status updates
- ✅ Error recovery scenarios

### Running Integration Tests

```bash
# Run all integration tests
make test-integration

# Run specific integration test
go test -v ./test/integration/... -run TestJobFlowController_StepExecution
```

## Test Best Practices

### Unit Tests

- ✅ Test one thing at a time
- ✅ Use table-driven tests for multiple scenarios
- ✅ Mock external dependencies
- ✅ Test error cases
- ✅ Test edge cases

### Integration Tests

- ✅ Use fake clients for isolation
- ✅ Test component interactions
- ✅ Verify cleanup and resource management
- ✅ Test error recovery

## Troubleshooting

### Tests Fail with "scheme not found"

**Solution**: Ensure `v1alpha1.AddToScheme(scheme)` is called before creating fake clients.

### Coverage Below Threshold

**Solution**: Add more test cases, especially for error paths and edge cases.

