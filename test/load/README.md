# Load Tests

Load tests validate zen-flow performance under high load conditions.

## Overview

Load tests cover:
- **Concurrent JobFlow Creation**: Creating many JobFlows simultaneously
- **Large DAG Execution**: Testing JobFlows with complex dependency graphs
- **Many Steps Per Flow**: Testing JobFlows with many sequential steps
- **Sustained Load**: Testing continuous load over time

## Prerequisites

- Kubernetes cluster (kind/k3d for local testing)
- zen-flow controller running
- Test namespace configured

## Running Load Tests

### Run All Load Tests

```bash
# Using Makefile
make test-load

# Directly
go test -v -timeout=30m ./test/load/...
```

### Run Specific Test

```bash
# Concurrent creation test
go test -v ./test/load/... -run TestConcurrentJobFlowCreation

# Large DAG test
go test -v ./test/load/... -run TestLargeDAGExecution

# Many steps test
go test -v ./test/load/... -run TestManyStepsPerFlow

# Sustained load test
go test -v ./test/load/... -run TestSustainedLoad
```

### Skip Load Tests (Short Mode)

```bash
go test -short ./test/load/...
```

### Run Benchmarks

```bash
go test -bench=. ./test/load/... -benchmem
```

## Test Configuration

### Default Parameters

- **JobFlowCount**: 100
- **StepsPerFlow**: 10
- **ConcurrentFlows**: 50
- **TestDuration**: 5 minutes
- **LargeDAGSteps**: 50

### Environment Variables

```bash
export LOAD_TEST_NAMESPACE=load-test
export LOAD_TEST_JOBFLOW_COUNT=100
export LOAD_TEST_STEPS_PER_FLOW=10
```

## Performance Targets

**Concurrent Creation**:
- Throughput: ≥ 50 JobFlows/second
- Average creation time: < 100ms
- Error rate: 0%

**Large DAGs**:
- 50-step DAG creation: < 500ms
- DAG validation: < 100ms

**Many Steps**:
- 100-step JobFlow creation: < 1s
- Memory usage: < 500MB for 20 flows

**Sustained Load**:
- Sustained rate: ≥ 10 JobFlows/second
- No memory leaks over 5 minutes
- Error rate: < 1%

## Test Structure

### TestConcurrentJobFlowCreation
Creates many JobFlows concurrently to test:
- API server load
- Controller reconciliation rate
- Resource creation throughput

### TestLargeDAGExecution
Tests JobFlows with complex dependency graphs:
- DAG validation performance
- Topological sort performance
- Parallel step execution

### TestManyStepsPerFlow
Tests JobFlows with many sequential steps:
- Large spec handling
- Step management overhead
- Memory usage

### TestSustainedLoad
Tests continuous load over time:
- Memory stability
- Performance consistency
- Error handling under load

## CI/CD Integration

Load tests are integrated into CI/CD:
- Run on pull requests (optional)
- Run before releases
- Performance regression detection

## Troubleshooting

### High Memory Usage
1. Check for memory leaks
2. Review controller resource limits
3. Check for goroutine leaks

### Slow Performance
1. Check API server load
2. Review controller reconciliation rate
3. Check network latency

### High Error Rate
1. Check controller logs
2. Review API server throttling
3. Check resource limits

