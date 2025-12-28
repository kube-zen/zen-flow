# Disaster Recovery

This document describes disaster recovery procedures for zen-flow, including recovery from failures, backup strategies, emergency stop procedures, and rollback scenarios.

## Table of Contents

- [Recovery from Controller Failures](#recovery-from-controller-failures)
- [Recovery from JobFlow Failures](#recovery-from-jobflow-failures)
- [Backup Strategies](#backup-strategies)
- [Emergency Stop Procedures](#emergency-stop-procedures)
- [Rollback Scenarios](#rollback-scenarios)

---

## Recovery from Controller Failures

### Controller Pod Crash

If the controller pod crashes:

```bash
# Check pod status
kubectl get pods -n zen-flow-system

# Check logs
kubectl logs -n zen-flow-system -l app=zen-flow-controller --previous

# Restart controller
kubectl delete pod -n zen-flow-system -l app=zen-flow-controller
```

### Leader Election Failure

If leader election fails:

```bash
# Check lease
kubectl get lease -n zen-flow-system

# Delete lease to force re-election
kubectl delete lease zen-flow-controller-leader-election -n zen-flow-system
```

---

## Recovery from JobFlow Failures

### JobFlow Stuck in Running

If a JobFlow is stuck:

```bash
# Check JobFlow status
kubectl get jobflow <name> -o yaml

# Check step status
kubectl get jobflow <name> -o jsonpath='{.status.steps[*]}'

# Check created Jobs
kubectl get jobs -l jobflow=<name>

# Manually fix if needed
kubectl patch jobflow <name> --type=merge -p '{"status":{"phase":"Failed"}}'
```

### Step Execution Failure

If a step fails:

```bash
# Check step status
kubectl get jobflow <name> -o jsonpath='{.status.steps[?(@.name=="step-name")]}'

# Check Job logs
kubectl logs job/<job-name>

# Retry step (if retryPolicy allows)
# Controller will automatically retry based on retryPolicy
```

---

## Backup Strategies

### JobFlow Backup

Backup JobFlow definitions:

```bash
# Backup all JobFlows
kubectl get jobflows -A -o yaml > jobflows-backup.yaml

# Backup specific namespace
kubectl get jobflows -n <namespace> -o yaml > jobflows-<namespace>.yaml
```

### Regular Backups

Set up regular backups:

```bash
# CronJob for daily backups
apiVersion: batch/v1
kind: CronJob
metadata:
  name: jobflow-backup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: bitnami/kubectl
            command:
            - /bin/sh
            - -c
            - kubectl get jobflows -A -o yaml > /backup/jobflows-$(date +%Y%m%d).yaml
          volumes:
          - name: backup
            persistentVolumeClaim:
              claimName: backup-pvc
```

---

## Emergency Stop Procedures

### Stop All JobFlows

```bash
# Scale down controller
kubectl scale deployment zen-flow-controller -n zen-flow-system --replicas=0

# Or delete deployment
kubectl delete deployment zen-flow-controller -n zen-flow-system
```

### Suspend Specific JobFlow

```bash
# Annotate JobFlow to suspend
kubectl annotate jobflow <name> workflow.zen.io/suspended=true
```

---

## Rollback Scenarios

### Rollback Controller Version

```bash
# Restore previous deployment
kubectl rollout undo deployment/zen-flow-controller -n zen-flow-system

# Or restore from backup
kubectl apply -f deploy/manifests/deployment.yaml --previous
```

### Rollback CRD Version

```bash
# Backup current CRD
kubectl get crd jobflows.workflow.zen.io -o yaml > crd-backup.yaml

# Restore previous CRD
kubectl apply -f deploy/crds/workflow.zen.io_jobflows.yaml --previous
```

---

## Prevention Measures

1. **Regular Backups**: Maintain regular backups of JobFlows
2. **Monitoring**: Set up alerts for controller failures
3. **Testing**: Test recovery procedures regularly
4. **Documentation**: Keep recovery procedures documented
5. **Resource Quotas**: Set quotas to prevent resource exhaustion

---

## See Also

- [Operator Guide](OPERATOR_GUIDE.md) - Operations guide
- [Architecture](ARCHITECTURE.md) - System architecture

