# Performance Tuning Guide

This guide provides recommendations for optimizing zen-flow controller performance.

## Overview

zen-flow performance depends on:
- Reconciliation rate
- DAG computation efficiency
- Status update batching
- API server load
- Resource allocation

## Key Performance Metrics

Monitor these metrics to identify bottlenecks:

- `zen_flow_reconciliation_duration_seconds`: Time per reconciliation
- `zen_flow_dag_computation_duration_seconds`: DAG computation time
- `zen_flow_status_update_duration_seconds`: Status update latency
- `zen_flow_api_server_calls_total`: API server call count
- `zen_flow_step_execution_queue_depth`: Queue depth

## Reconciliation Rate Tuning

### Max Concurrent Reconciles

Adjust `--max-concurrent-reconciles` based on workload:

```yaml
args:
  - --max-concurrent-reconciles=20
```

**Guidelines**:
- **Low load** (< 50 JobFlows): 5-10
- **Medium load** (50-200 JobFlows): 10-20
- **High load** (200-1000 JobFlows): 20-50
- **Very high load** (> 1000 JobFlows): 50-100

**Monitoring**:
```promql
# Reconciliation rate
rate(zen_flow_reconciliation_total[5m])

# Queue depth
zen_flow_step_execution_queue_depth

# Reconciliation duration
histogram_quantile(0.95, zen_flow_reconciliation_duration_seconds_bucket)
```

**Optimization**:
- Increase if queue depth is consistently high
- Decrease if API server is throttling
- Monitor reconciliation duration - should be < 1s for most cases

## DAG Computation Optimization

DAG caching is enabled by default and significantly improves performance.

### Cache Hit Rate

Monitor cache effectiveness:

```promql
# DAG computation time (should be low with caching)
zen_flow_dag_computation_duration_seconds

# Cache hits vs misses (if exposed)
zen_flow_dag_cache_hits_total / zen_flow_dag_cache_requests_total
```

### Optimization Strategies

1. **Reduce DAG Complexity**:
   - Minimize step dependencies
   - Use parallel execution where possible
   - Avoid deep dependency chains

2. **Optimize Step Dependencies**:
   - Group related steps
   - Use fan-out patterns instead of sequential chains
   - Minimize cross-step dependencies

3. **Cache Tuning**:
   - Cache is automatically invalidated on spec changes
   - No manual tuning needed currently

## Status Update Optimization

Status update batching is enabled by default, reducing API server load.

### Monitoring

```promql
# Status update duration
zen_flow_status_update_duration_seconds

# Status update frequency
rate(zen_flow_status_updates_total[5m])
```

### Optimization

1. **Batch Size**: Status updates are batched per reconciliation loop
2. **Update Frequency**: Updates occur once per reconciliation
3. **API Server Load**: Monitor `zen_flow_api_server_calls_total`

## API Server Load Reduction

### Strategies

1. **Reduce API Calls**:
   - Status update batching (enabled by default)
   - DAG caching (enabled by default)
   - Parallel step status refresh (enabled by default)

2. **Optimize Watch Filters**:
   - Use label selectors where possible
   - Minimize watch scope

3. **Rate Limiting**:
   - Monitor API server throttling
   - Adjust reconciliation rate if needed

### Monitoring

```promql
# API server call rate
rate(zen_flow_api_server_calls_total[5m])

# API server call duration
zen_flow_api_server_call_duration_seconds

# Failed API calls
rate(zen_flow_api_server_calls_total{success="false"}[5m])
```

## Step Execution Optimization

### Parallel Execution

Steps with no dependencies execute in parallel automatically.

**Optimization**:
- Design DAGs to maximize parallelism
- Minimize sequential dependencies
- Use fan-out patterns

### Queue Depth

Monitor step execution queue:

```promql
zen_flow_step_execution_queue_depth
```

**Optimization**:
- Increase `--max-concurrent-reconciles` if queue is deep
- Optimize step dependencies to reduce queue depth
- Consider horizontal scaling for very high loads

## Resource Optimization

### CPU

**Symptoms of CPU Bottleneck**:
- High CPU usage (> 80%)
- CPU throttling
- Slow reconciliation

**Solutions**:
1. Increase CPU limits
2. Enable VPA for automatic adjustment
3. Reduce `--max-concurrent-reconciles`
4. Optimize DAG computation

### Memory

**Symptoms of Memory Bottleneck**:
- High memory usage (> 80%)
- OOMKilled pods
- Memory pressure

**Solutions**:
1. Increase memory limits
2. Enable VPA for automatic adjustment
3. Check for memory leaks
4. Reduce cache size (if configurable)

## Network Optimization

### API Server Latency

Monitor API server response times:

```promql
zen_flow_api_server_call_duration_seconds
```

**Optimization**:
- Ensure controller and API server are in same region/zone
- Use dedicated API server endpoints if available
- Monitor network latency

## JobFlow Design Best Practices

### Step Design

1. **Minimize Dependencies**:
   ```yaml
   # Good: Parallel execution
   steps:
     - name: step1
     - name: step2
     - name: step3
       dependencies: [step1, step2]
   ```

2. **Use Fan-out Patterns**:
   ```yaml
   # Good: Fan-out
   steps:
     - name: prepare
     - name: task1
       dependencies: [prepare]
     - name: task2
       dependencies: [prepare]
     - name: task3
       dependencies: [prepare]
   ```

3. **Avoid Deep Chains**:
   ```yaml
   # Avoid: Deep sequential chain
   steps:
     - name: step1
     - name: step2
       dependencies: [step1]
     - name: step3
       dependencies: [step2]
     # ... many more sequential steps
   ```

### Resource Templates

1. **Reuse Templates**: Define resource templates once, reuse across steps
2. **Minimize Template Complexity**: Keep templates simple
3. **Cache Templates**: Templates are cached automatically

## Monitoring and Alerting

### Key Alerts

1. **High Reconciliation Duration**:
   ```promql
   histogram_quantile(0.95, zen_flow_reconciliation_duration_seconds_bucket) > 5
   ```

2. **High Queue Depth**:
   ```promql
   zen_flow_step_execution_queue_depth > 100
   ```

3. **High API Server Call Rate**:
   ```promql
   rate(zen_flow_api_server_calls_total[5m]) > 100
   ```

4. **High DAG Computation Time**:
   ```promql
   zen_flow_dag_computation_duration_seconds > 1
   ```

### Performance Dashboard

Use the Grafana dashboard (`deploy/grafana/dashboard.json`) to monitor:
- Reconciliation rate and duration
- DAG computation time
- Status update latency
- API server call metrics
- Queue depth

## Troubleshooting Performance Issues

### Slow Reconciliation

**Symptoms**:
- High reconciliation duration
- Slow JobFlow processing

**Diagnosis**:
```bash
# Check reconciliation duration
kubectl get --raw /metrics | grep zen_flow_reconciliation_duration_seconds

# Check queue depth
kubectl get --raw /metrics | grep zen_flow_step_execution_queue_depth
```

**Solutions**:
1. Increase `--max-concurrent-reconciles`
2. Optimize DAG complexity
3. Increase CPU resources
4. Check API server latency

### High API Server Load

**Symptoms**:
- API server throttling
- Slow API responses

**Diagnosis**:
```bash
# Check API call rate
kubectl get --raw /metrics | grep zen_flow_api_server_calls_total

# Check API call duration
kubectl get --raw /metrics | grep zen_flow_api_server_call_duration_seconds
```

**Solutions**:
1. Reduce reconciliation rate
2. Ensure status update batching is enabled
3. Optimize watch filters
4. Scale API server if needed

### Memory Pressure

**Symptoms**:
- High memory usage
- OOMKilled pods

**Diagnosis**:
```bash
# Check memory usage
kubectl top pod -n zen-flow-system

# Check for memory leaks
kubectl get --raw /metrics | grep container_memory_working_set_bytes
```

**Solutions**:
1. Increase memory limits
2. Enable VPA
3. Check for memory leaks
4. Reduce cache size

## Performance Testing

### Load Testing

Use the load tests in `test/load/` to validate performance:

```bash
# Run load tests
make test-load

# Or directly
go test -v ./test/load/...
```

### Benchmarking

Benchmark specific operations:

```bash
# Benchmark JobFlow creation
go test -bench=BenchmarkJobFlowCreation ./test/load/...
```

## Performance Checklist

- [ ] Monitor key performance metrics
- [ ] Tune `--max-concurrent-reconciles` based on workload
- [ ] Ensure DAG caching is effective
- [ ] Verify status update batching is working
- [ ] Optimize JobFlow DAG design
- [ ] Set appropriate resource limits
- [ ] Enable VPA for automatic resource adjustment
- [ ] Configure performance alerts
- [ ] Run load tests regularly
- [ ] Review performance metrics periodically

