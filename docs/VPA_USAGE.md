# VPA (Vertical Pod Autoscaler) Usage Guide

## Overview

The Vertical Pod Autoscaler (VPA) automatically adjusts CPU and memory resource requests and limits for the `zen-flow` controller based on actual usage patterns. This helps optimize resource utilization and prevent OOM (Out of Memory) kills.

## Prerequisites

1. **VPA installed in cluster**: VPA must be installed in your Kubernetes cluster
   ```bash
   # Install VPA (example - adjust for your environment)
   kubectl apply -f https://github.com/kubernetes/autoscaler/releases/latest/download/vpa-release.yaml
   ```

2. **Metrics Server**: VPA requires metrics-server to collect resource usage data
   ```bash
   # Verify metrics-server is running
   kubectl get deployment metrics-server -n kube-system
   ```

## Installation

Apply the VPA configuration:

```bash
kubectl apply -f deploy/manifests/vpa.yaml
```

## Configuration

The VPA configuration (`deploy/manifests/vpa.yaml`) includes:

### Update Modes

- **Auto** (default): VPA automatically updates pod resource requests/limits
- **Off**: VPA only provides recommendations without applying them
- **Initial**: VPA only sets resources on pod creation

### Resource Limits

- **Minimum**: 50m CPU, 64Mi memory (prevents under-provisioning)
- **Maximum**: 2 CPU, 4Gi memory (prevents over-provisioning)

### Resource Recommendations by Scale

Based on typical usage patterns:

| Scale | JobFlows | Recommended CPU | Recommended Memory |
|-------|----------|----------------|-------------------|
| Small | < 10 | ~100m | ~128Mi |
| Medium | 10-100 | ~500m | ~512Mi |
| Large | 100-1000 | ~1 CPU | ~1Gi |
| High-Load | 1000+ | ~2 CPU | ~2Gi |

## Monitoring VPA

### Check VPA Status

```bash
# View VPA object
kubectl get vpa zen-flow-controller-vpa -n zen-flow-system

# Describe VPA for detailed information
kubectl describe vpa zen-flow-controller-vpa -n zen-flow-system
```

### View Recommendations

VPA provides recommendations in the status:

```bash
kubectl get vpa zen-flow-controller-vpa -n zen-flow-system -o yaml | grep -A 20 "recommendation"
```

### Monitor Pod Resources

```bash
# View current pod resource requests/limits
kubectl describe pod -l app.kubernetes.io/name=zen-flow -n zen-flow-system | grep -A 5 "Limits\|Requests"
```

## Troubleshooting

### VPA Not Updating Resources

1. **Check VPA is running**:
   ```bash
   kubectl get vpa zen-flow-controller-vpa -n zen-flow-system
   ```

2. **Check metrics-server**:
   ```bash
   kubectl top pods -n zen-flow-system
   ```

3. **Check VPA logs**:
   ```bash
   kubectl logs -n kube-system -l app=vpa-recommender
   kubectl logs -n kube-system -l app=vpa-updater
   ```

### Pods Being Evicted

If pods are being evicted due to resource constraints:

1. **Increase maxAllowed** in VPA configuration
2. **Check actual usage**:
   ```bash
   kubectl top pods -n zen-flow-system
   ```
3. **Review VPA recommendations**:
   ```bash
   kubectl describe vpa zen-flow-controller-vpa -n zen-flow-system
   ```

### VPA Recommendations Too Low/High

1. **Wait for metrics collection**: VPA needs at least 24 hours of metrics
2. **Review actual usage patterns**: Check `kubectl top pods` over time
3. **Adjust minAllowed/maxAllowed**: Update VPA configuration based on your needs

## Best Practices

1. **Start with Auto mode**: Let VPA learn usage patterns
2. **Monitor for 24-48 hours**: VPA needs time to collect metrics
3. **Review recommendations**: Check VPA recommendations before trusting Auto mode
4. **Set appropriate limits**: Use minAllowed/maxAllowed to prevent extreme values
5. **Combine with HPA**: For horizontal scaling, use HPA alongside VPA

## Disabling VPA

To disable VPA:

```bash
kubectl delete vpa zen-flow-controller-vpa -n zen-flow-system
```

Or set `updateMode: "Off"` to keep recommendations without applying them.

## Makefile Targets

Use the following Makefile targets:

```bash
# Check VPA configuration validity
make check-vpa

# View VPA status
make vpa-status

# Update VPA recommendations (after 24+ hours of metrics)
make vpa-update
```

## Additional Resources

- [VPA Documentation](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)
- [Resource Requirements Guide](./RESOURCE_REQUIREMENTS.md)
- [Performance Tuning Guide](./PERFORMANCE_TUNING.md)

