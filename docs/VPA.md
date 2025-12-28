# Vertical Pod Autoscaler (VPA) Configuration

zen-flow supports Vertical Pod Autoscaler (VPA) for automatic resource recommendation and adjustment.

## Prerequisites

VPA requires the VPA operator to be installed in your cluster. See the [official VPA documentation](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler) for installation instructions.

## Installation

### Using Manifests

```bash
kubectl apply -f deploy/manifests/vpa.yaml
```

### Using Helm

The VPA configuration can be included in your Helm values:

```yaml
vpa:
  enabled: true
  updateMode: Auto
  minAllowed:
    cpu: 50m
    memory: 32Mi
  maxAllowed:
    cpu: 2000m
    memory: 2Gi
```

## Update Modes

VPA supports different update modes:

- **Off**: VPA only provides recommendations but doesn't update pods
- **Initial**: VPA sets resources only on pod creation
- **Recreate**: VPA updates resources by recreating pods (may cause downtime)
- **Auto**: VPA automatically updates resources (recommended for production)

## Resource Constraints

The default VPA configuration sets:

- **Minimum Allowed**: 50m CPU, 32Mi memory
- **Maximum Allowed**: 2000m CPU, 2Gi memory

Adjust these values based on your cluster's capacity and requirements.

## Monitoring VPA Recommendations

View VPA recommendations:

```bash
kubectl describe vpa zen-flow-controller-vpa -n zen-flow-system
```

## Best Practices

1. **Start with "Off" mode** to observe recommendations without automatic changes
2. **Monitor recommendations** for a period before enabling automatic updates
3. **Set appropriate min/max bounds** based on your cluster capacity
4. **Use "Recreate" or "Auto" mode** only after validating recommendations

## Troubleshooting

### VPA Not Working

Check if VPA operator is installed:

```bash
kubectl get deployment -n kube-system | grep vpa
```

### Pods Not Updating

Check VPA status:

```bash
kubectl get vpa -n zen-flow-system
kubectl describe vpa zen-flow-controller-vpa -n zen-flow-system
```

### Resource Recommendations

View recommendations:

```bash
kubectl get vpa zen-flow-controller-vpa -n zen-flow-system -o yaml
```

## References

- [VPA Documentation](https://github.com/kubernetes/autoscaler/tree/master/vertical-pod-autoscaler)
- [VPA Best Practices](https://github.com/kubernetes/autoscaler/blob/master/vertical-pod-autoscaler/README.md)

