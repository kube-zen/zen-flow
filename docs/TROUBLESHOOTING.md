# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with zen-flow.

## Table of Contents

- [Controller Not Starting](#controller-not-starting)
- [JobFlows Stuck in Pending](#jobflows-stuck-in-pending)
- [Jobs Not Created](#jobs-not-created)
- [Webhook Issues](#webhook-issues)
- [Permission Errors](#permission-errors)
- [Status Not Updating](#status-not-updating)
- [Performance Issues](#performance-issues)
- [Common Error Messages](#common-error-messages)

---

## Controller Not Starting

### Symptoms
- Controller pod is in `CrashLoopBackOff` or `Error` state
- No logs from controller
- `kubectl get pods -n zen-flow-system` shows unhealthy pod

### Diagnosis

```bash
# Check pod status
kubectl get pods -n zen-flow-system

# Check pod logs
kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow

# Check pod events
kubectl describe pod -n zen-flow-system -l app.kubernetes.io/name=zen-flow
```

### Common Causes

#### 1. Missing RBAC Permissions

**Error**: `Forbidden` or `Unauthorized` errors in logs

**Solution**: Verify RBAC is installed correctly

```bash
# Check ServiceAccount exists
kubectl get serviceaccount -n zen-flow-system zen-flow-controller

# Check Role/RoleBinding or ClusterRole/ClusterRoleBinding
kubectl get role,rolebinding -n zen-flow-system
kubectl get clusterrole,clusterrolebinding | grep zen-flow

# Reinstall RBAC
kubectl apply -f deploy/manifests/rbac.yaml
# or
helm upgrade zen-flow ./charts/zen-flow --namespace zen-flow-system
```

#### 2. Webhook Certificate Issues

**Error**: `failed to load certificate` or TLS errors

**Solution**: 

If using cert-manager:
```bash
# Check certificate status
kubectl get certificate -n zen-flow-system

# Check cert-manager logs
kubectl logs -n cert-manager -l app=cert-manager
```

If using manual certificates:
```bash
# Verify secret exists
kubectl get secret -n zen-flow-system zen-flow-webhook-cert

# Check certificate validity
kubectl get secret -n zen-flow-system zen-flow-webhook-cert -o jsonpath='{.data.tls\.crt}' | base64 -d | openssl x509 -text -noout
```

**Workaround**: Disable webhooks temporarily
```bash
helm upgrade zen-flow ./charts/zen-flow \
  --namespace zen-flow-system \
  --set webhook.enabled=false
```

#### 3. Leader Election Issues

**Error**: `leader election failed` or multiple controller instances

**Solution**: Check leader election namespace

```bash
# Verify namespace exists
kubectl get namespace zen-flow-system

# Check lease object
kubectl get lease -n zen-flow-system zen-flow-controller-leader-election

# If using custom namespace, set it correctly
helm upgrade zen-flow ./charts/zen-flow \
  --namespace zen-flow-system \
  --set leaderElection.namespace=zen-flow-system
```

#### 4. Resource Constraints

**Error**: `OOMKilled` or `Evicted` pod status

**Solution**: Increase resource limits

```bash
helm upgrade zen-flow ./charts/zen-flow \
  --namespace zen-flow-system \
  --set resources.limits.memory=512Mi \
  --set resources.requests.memory=128Mi
```

---

## JobFlows Stuck in Pending

### Symptoms
- JobFlow status shows `phase: Pending`
- No Jobs are created
- No errors in controller logs

### Diagnosis

```bash
# Check JobFlow status
kubectl get jobflow <name> -o yaml

# Check controller logs
kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow | grep <name>

# Check for validation errors
kubectl describe jobflow <name>
```

### Common Causes

#### 1. Invalid JobFlow Spec

**Solution**: Validate the JobFlow spec

```bash
# Check validation errors
kubectl get jobflow <name> -o jsonpath='{.status.conditions}'

# Validate manually
kubectl apply -f <jobflow.yaml> --dry-run=client -o yaml
```

**Common Issues**:
- Empty `steps` array
- Invalid step dependencies (circular or missing)
- Invalid Job template
- Missing required fields

#### 2. Manual Approval Required

**Solution**: Approve the step

```bash
# Check if step is waiting for approval
kubectl get jobflow <name> -o jsonpath='{.status.steps[*].phase}'

# Approve manual step
kubectl annotate jobflow <name> \
  workflow.kube-zen.io/approved/<step-name>=true
```

#### 3. When Condition Evaluates to False

**Solution**: Check when condition

```bash
# Check step when condition
kubectl get jobflow <name> -o jsonpath='{.spec.steps[*].when}'

# Note: v0.0.1-alpha only supports "always", "never", "true", "false"
# Complex conditions are not yet supported
```

#### 4. Concurrency Policy Blocking

**Solution**: Check for existing JobFlows

```bash
# List all JobFlows with same name
kubectl get jobflows --all-namespaces | grep <name>

# If using "Forbid" policy, delete or wait for existing JobFlow to complete
kubectl delete jobflow <name>
```

---

## Jobs Not Created

### Symptoms
- JobFlow is `Running` but no Jobs exist
- Step status shows `Pending` without `jobRef`

### Diagnosis

```bash
# Check if Jobs should be created
kubectl get jobflow <name> -o jsonpath='{.status.steps[*]}'

# Check controller logs for errors
kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow | grep <step-name>

# Check for permission errors
kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow | grep -i forbidden
```

### Common Causes

#### 1. Insufficient Permissions

**Error**: `Forbidden` when creating Jobs

**Solution**: Verify RBAC includes Job creation

```bash
# Check RBAC permissions
kubectl get clusterrole zen-flow-controller -o yaml | grep -A 5 jobs

# Should include: create, get, list, watch, delete
```

#### 2. Invalid Job Template

**Error**: `invalid template` or `failed to create job`

**Solution**: Validate Job template

```bash
# Extract and validate Job template
kubectl get jobflow <name> -o jsonpath='{.spec.steps[0].template}' | kubectl apply --dry-run=client -f -

# Check for common issues:
# - Missing apiVersion or kind
# - Invalid container spec
# - Missing required fields
```

#### 3. Resource Quotas

**Error**: `exceeded quota` or `insufficient resources`

**Solution**: Check resource quotas

```bash
# Check namespace quotas
kubectl describe quota -n <namespace>

# Check resource usage
kubectl top pods -n <namespace>
kubectl top nodes
```

---

## Webhook Issues

### Symptoms
- JobFlow creation fails with webhook errors
- `Internal error occurred: failed calling webhook`
- Webhook timeout errors

### Diagnosis

```bash
# Check webhook configuration
kubectl get validatingwebhookconfiguration,mutatingwebhookconfiguration | grep zen-flow

# Check webhook service
kubectl get svc -n zen-flow-system zen-flow-controller-metrics

# Check webhook pod
kubectl get pods -n zen-flow-system -l app.kubernetes.io/name=zen-flow

# Test webhook endpoint
kubectl run test-pod --image=curlimages/curl --rm -it --restart=Never -- \
  curl -k https://zen-flow-controller-metrics.zen-flow-system.svc:9443/healthz
```

### Common Causes

#### 1. Webhook Service Not Ready

**Solution**: Wait for service to be ready

```bash
# Check service endpoints
kubectl get endpoints -n zen-flow-system zen-flow-controller-metrics

# Should show at least one endpoint (controller pod IP)
```

#### 2. Certificate Issues

**Solution**: See [Controller Not Starting - Webhook Certificate Issues](#2-webhook-certificate-issues)

#### 3. Network Policy Blocking

**Solution**: Check NetworkPolicy

```bash
# Check NetworkPolicies
kubectl get networkpolicy -n zen-flow-system

# Temporarily disable to test
kubectl delete networkpolicy -n zen-flow-system --all
```

#### 4. Failure Policy Too Strict

**Solution**: Use `Ignore` policy for testing

```bash
# Update webhook failure policy
kubectl patch validatingwebhookconfiguration validate-jobflow.workflow.kube-zen.io \
  --type='json' -p='[{"op": "replace", "path": "/webhooks/0/failurePolicy", "value": "Ignore"}]'
```

---

## Permission Errors

### Symptoms
- `Forbidden` errors in logs
- Jobs not created
- Status updates fail

### Diagnosis

```bash
# Check ServiceAccount
kubectl get serviceaccount -n zen-flow-system zen-flow-controller

# Check RBAC bindings
kubectl get rolebinding,clusterrolebinding | grep zen-flow

# Test permissions
kubectl auth can-i create jobs --as=system:serviceaccount:zen-flow-system:zen-flow-controller -n default
```

### Solution

Reinstall RBAC with correct permissions:

```bash
# Using Helm
helm upgrade zen-flow ./charts/zen-flow --namespace zen-flow-system

# Using kubectl
kubectl apply -f deploy/manifests/rbac.yaml
```

---

## Status Not Updating

### Symptoms
- JobFlow status remains unchanged
- Step status doesn't reflect Job state
- Completion time not set

### Diagnosis

```bash
# Check JobFlow status
kubectl get jobflow <name> -o yaml | grep -A 20 status

# Check if Jobs exist
kubectl get jobs -l workflow.kube-zen.io/flow=<name>

# Check controller logs
kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow | grep <name>
```

### Common Causes

#### 1. Controller Not Reconciling

**Solution**: Check controller is running and reconciling

```bash
# Check reconciliation metrics
kubectl port-forward -n zen-flow-system svc/zen-flow-controller-metrics 8080:8080
curl http://localhost:8080/metrics | grep zen_flow_reconcile
```

#### 2. Status Update Conflicts

**Error**: `Conflict` errors in logs

**Solution**: This is normal - controller will retry. If persistent:

```bash
# Check for multiple controller instances
kubectl get pods -n zen-flow-system -l app.kubernetes.io/name=zen-flow

# Should be only 1 pod (or multiple with leader election)
```

#### 3. Job Not Found

**Error**: `Job not found` in logs

**Solution**: Check if Job was deleted

```bash
# Check Job history
kubectl get jobs --all-namespaces --show-labels | grep <name>

# Check if TTL expired
kubectl get jobflow <name> -o jsonpath='{.spec.executionPolicy.ttlSecondsAfterFinished}'
```

---

## Performance Issues

### Symptoms
- Slow reconciliation
- High CPU/memory usage
- Delayed Job creation

### Diagnosis

```bash
# Check resource usage
kubectl top pod -n zen-flow-system -l app.kubernetes.io/name=zen-flow

# Check metrics
kubectl port-forward -n zen-flow-system svc/zen-flow-controller-metrics 8080:8080
curl http://localhost:8080/metrics | grep zen_flow
```

### Solutions

#### 1. Increase Resources

```bash
helm upgrade zen-flow ./charts/zen-flow \
  --namespace zen-flow-system \
  --set resources.limits.cpu=1000m \
  --set resources.limits.memory=512Mi
```

#### 2. Adjust Concurrency

```bash
# Increase max concurrent reconciles
helm upgrade zen-flow ./charts/zen-flow \
  --namespace zen-flow-system \
  --set controller.maxConcurrentReconciles=20
```

#### 3. Reduce Reconciliation Frequency

This is controlled by Kubernetes - ensure controller is healthy and not restarting frequently.

---

## Common Error Messages

### `jobflows.workflow.kube-zen.io "X" not found`

**Cause**: JobFlow was deleted or doesn't exist

**Solution**: Verify JobFlow exists
```bash
kubectl get jobflow <name>
```

### `failed to create job: Forbidden`

**Cause**: Insufficient RBAC permissions

**Solution**: Reinstall RBAC (see [Permission Errors](#permission-errors))

### `validation failed: step name cannot be empty`

**Cause**: Invalid JobFlow spec

**Solution**: Ensure all steps have names
```bash
kubectl get jobflow <name> -o yaml | grep -A 5 steps
```

### `circular dependency detected`

**Cause**: Steps have circular dependencies

**Solution**: Review step dependencies
```bash
kubectl get jobflow <name> -o jsonpath='{.spec.steps[*].dependencies}'
```

### `webhook server error: failed to add scheme`

**Cause**: Webhook initialization failed (should be fixed in latest version)

**Solution**: Check controller logs and restart if needed
```bash
kubectl logs -n zen-flow-system -l app.kubernetes.io/name=zen-flow | grep -i scheme
kubectl delete pod -n zen-flow-system -l app.kubernetes.io/name=zen-flow
```

---

## Getting Help

If you're still experiencing issues:

1. **Check Logs**: Controller logs contain detailed error information
2. **Check Events**: Kubernetes events may have additional context
   ```bash
   kubectl get events -n <namespace> --sort-by='.lastTimestamp'
   ```
3. **Open an Issue**: [GitHub Issues](https://github.com/kube-zen/zen-flow/issues)
4. **Review Documentation**: See [docs/](docs/) for detailed guides

---

*Last updated: 2015-12-29*

