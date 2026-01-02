# Resource Requirements Guide

This guide documents resource requirements for zen-flow controller at different scales.

## Overview

zen-flow controller resource usage depends on:
- Number of JobFlows being managed
- Number of steps per JobFlow
- Reconciliation frequency
- DAG complexity
- Concurrent reconciles

## Default Resource Limits

The default deployment sets conservative limits suitable for small to medium workloads:

```yaml
resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "256Mi"
    cpu: "500m"
```

## Resource Requirements by Scale

### Small Scale (< 50 JobFlows, < 10 steps each)

**Recommended Resources**:
```yaml
resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "128Mi"
    cpu: "200m"
```

**Characteristics**:
- Low reconciliation load
- Minimal DAG computation
- Infrequent status updates

### Medium Scale (50-200 JobFlows, 10-50 steps each)

**Recommended Resources**:
```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "200m"
  limits:
    memory: "512Mi"
    cpu: "1000m"
```

**Characteristics**:
- Moderate reconciliation load
- Regular DAG computation
- Frequent status updates
- Some parallel step execution

### Large Scale (200-1000 JobFlows, 50-100 steps each)

**Recommended Resources**:
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "2000m"
```

**Characteristics**:
- High reconciliation load
- Complex DAG computation
- Frequent status updates
- High parallel step execution
- Caching benefits become significant

### Very Large Scale (> 1000 JobFlows, > 100 steps each)

**Recommended Resources**:
```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "1000m"
  limits:
    memory: "2Gi"
    cpu: "4000m"
```

**Characteristics**:
- Very high reconciliation load
- Very complex DAG computation
- Continuous status updates
- Maximum parallel step execution
- Requires horizontal scaling consideration

## Vertical Pod Autoscaler (VPA)

VPA is recommended for automatic resource adjustment based on actual usage.

### VPA Configuration

The default VPA configuration (`deploy/manifests/vpa.yaml`) sets:

```yaml
minAllowed:
  cpu: 50m
  memory: 32Mi
maxAllowed:
  cpu: 2000m
  memory: 2Gi
```

### VPA Update Modes

- **Off**: VPA provides recommendations only (recommended for initial setup)
- **Initial**: VPA sets resources on pod creation
- **Recreate**: VPA updates by recreating pods (may cause downtime)
- **Auto**: VPA automatically updates resources (production-ready clusters)

### Monitoring VPA Recommendations

```bash
# View VPA recommendations
kubectl describe vpa zen-flow-controller-vpa -n zen-flow-system

# Check VPA status
kubectl get vpa -n zen-flow-system
```

## Horizontal Pod Autoscaler (HPA)

For very large scales, consider HPA for horizontal scaling:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: zen-flow-controller-hpa
  namespace: zen-flow-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: zen-flow-controller
  minReplicas: 1
  maxReplicas: 3
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

**Note**: HPA requires careful consideration of leader election and state management.

## Resource Monitoring

### Key Metrics

Monitor these metrics to determine resource needs:

- `zen_flow_reconciliation_duration_seconds`: Reconciliation time
- `zen_flow_dag_computation_duration_seconds`: DAG computation time
- `zen_flow_status_update_duration_seconds`: Status update latency
- `zen_flow_api_server_calls_total`: API server load
- `zen_flow_step_execution_queue_depth`: Queue depth

### Prometheus Queries

**Average CPU Usage**:
```promql
rate(container_cpu_usage_seconds_total{pod=~"zen-flow-controller-.*"}[5m])
```

**Average Memory Usage**:
```promql
container_memory_working_set_bytes{pod=~"zen-flow-controller-.*"}
```

**Reconciliation Rate**:
```promql
rate(zen_flow_reconciliation_total[5m])
```

## Performance Tuning

### Reconciliation Rate

Adjust `--max-concurrent-reconciles` based on workload:

```yaml
args:
  - --max-concurrent-reconciles=20  # Increase for high load
```

**Recommendations**:
- Small scale: 5-10
- Medium scale: 10-20
- Large scale: 20-50
- Very large scale: 50-100

### DAG Caching

DAG caching is enabled by default and significantly reduces computation for large DAGs. Monitor:

```promql
zen_flow_dag_computation_duration_seconds
```

If DAG computation is still high, consider:
- Reducing DAG complexity
- Increasing cache TTL
- Optimizing step dependencies

### Status Update Batching

Status update batching is enabled by default. Monitor:

```promql
zen_flow_status_update_duration_seconds
```

If status updates are slow, consider:
- Reducing status update frequency
- Optimizing status update logic
- Increasing API server capacity

## Troubleshooting

### High Memory Usage

**Symptoms**:
- Memory usage approaching limits
- OOMKilled pods

**Solutions**:
1. Increase memory limits
2. Enable VPA for automatic adjustment
3. Reduce `--max-concurrent-reconciles`
4. Check for memory leaks (review metrics)

### High CPU Usage

**Symptoms**:
- CPU throttling
- Slow reconciliation

**Solutions**:
1. Increase CPU limits
2. Enable VPA for automatic adjustment
3. Optimize DAG computation
4. Reduce reconciliation frequency

### Resource Starvation

**Symptoms**:
- Pods in Pending state
- Insufficient resources errors

**Solutions**:
1. Check cluster capacity
2. Reduce resource requests
3. Enable VPA with lower minAllowed
4. Consider node scaling

## Best Practices

1. **Start Conservative**: Begin with default limits and monitor
2. **Use VPA**: Enable VPA in "Off" mode first to observe recommendations
3. **Monitor Metrics**: Track resource usage and adjust accordingly
4. **Test Under Load**: Test resource requirements with realistic workloads
5. **Document Changes**: Document resource adjustments and rationale
6. **Review Regularly**: Review resource usage periodically as workload changes

## Example: Scaling from Small to Large

### Phase 1: Initial Deployment (Small Scale)
```yaml
resources:
  requests:
    memory: "64Mi"
    cpu: "100m"
  limits:
    memory: "128Mi"
    cpu: "200m"
```

### Phase 2: Growth (Medium Scale)
Enable VPA in "Off" mode, observe recommendations, then:
```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "200m"
  limits:
    memory: "512Mi"
    cpu: "1000m"
```

### Phase 3: Production (Large Scale)
Enable VPA in "Auto" mode:
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "500m"
  limits:
    memory: "1Gi"
    cpu: "2000m"
```

VPA will automatically adjust based on actual usage.

