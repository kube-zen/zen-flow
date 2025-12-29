# Metrics Documentation

This document describes all Prometheus metrics exposed by the zen-flow controller.

## Metrics Endpoint

The zen-flow controller exposes metrics on the `/metrics` endpoint, defaulting to port `8080`.

## Available Metrics

### `zen_flow_jobflows`
**Type**: Gauge  
**Description**: Current number of JobFlows by phase and namespace  
**Labels**:
- `phase`: JobFlow phase (Pending, Running, Succeeded, Failed, Suspended, Paused)
- `namespace`: Namespace of the JobFlow

**Example**:
```
zen_flow_jobflows{phase="Running",namespace="default"} 5
zen_flow_jobflows{phase="Succeeded",namespace="default"} 10
zen_flow_jobflows{phase="Paused",namespace="default"} 2
```

---

### `zen_flow_jobflow_phase_transitions_total`
**Type**: Counter  
**Description**: Total number of JobFlow phase transitions  
**Labels**:
- `phase`: Target phase of the transition
- `namespace`: Namespace of the JobFlow

**Example**:
```
zen_flow_jobflow_phase_transitions_total{phase="Running",namespace="default"} 150
zen_flow_jobflow_phase_transitions_total{phase="Succeeded",namespace="default"} 100
```

---

### `zen_flow_steps`
**Type**: Gauge  
**Description**: Current number of steps by phase  
**Labels**:
- `flow`: Name of the JobFlow
- `phase`: Step phase (Pending, Running, Succeeded, Failed, Skipped, PendingApproval)

**Example**:
```
zen_flow_steps{flow="data-pipeline",phase="Running"} 3
zen_flow_steps{flow="data-pipeline",phase="Succeeded"} 7
zen_flow_steps{flow="migration-prod",phase="PendingApproval"} 1
```

---

### `zen_flow_step_phase_transitions_total`
**Type**: Counter  
**Description**: Total number of step phase transitions  
**Labels**:
- `flow`: Name of the JobFlow
- `phase`: Target phase of the transition

**Example**:
```
zen_flow_step_phase_transitions_total{flow="data-pipeline",phase="Succeeded"} 500
zen_flow_step_phase_transitions_total{flow="data-pipeline",phase="Failed"} 10
zen_flow_step_phase_transitions_total{flow="migration-prod",phase="PendingApproval"} 5
```

---

### `zen_flow_step_duration_seconds`
**Type**: Histogram  
**Description**: Duration of step execution in seconds  
**Labels**:
- `flow`: Name of the JobFlow
- `step`: Name of the step
- `result`: Result of the step (success, failure)

**Buckets**: Default Prometheus buckets (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)

**Example**:
```
zen_flow_step_duration_seconds_bucket{flow="data-pipeline",step="extract",result="success",le="1.0"} 5
zen_flow_step_duration_seconds_sum{flow="data-pipeline",step="extract",result="success"} 3.5
zen_flow_step_duration_seconds_count{flow="data-pipeline",step="extract",result="success"} 5
```

---

### `zen_flow_reconciliation_duration_seconds`
**Type**: Histogram  
**Description**: Duration of reconciliation loops in seconds  
**Buckets**: [0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]

**Example**:
```
zen_flow_reconciliation_duration_seconds_bucket{le="0.1"} 1000
zen_flow_reconciliation_duration_seconds_sum 50.5
zen_flow_reconciliation_duration_seconds_count 1000
```

---

## Prometheus Queries

### JobFlows by Phase (Current State)
```promql
sum by (phase) (zen_flow_jobflows)
```

### JobFlows Waiting for Approval
```promql
sum(zen_flow_jobflows{phase="Paused"})
```

### Steps Waiting for Approval
```promql
sum(zen_flow_steps{phase="PendingApproval"})
```

### Step Success Rate
```promql
sum(rate(zen_flow_step_phase_transitions_total{phase="Succeeded"}[5m])) 
/ 
sum(rate(zen_flow_step_phase_transitions_total[5m])) * 100
```

### Phase Transition Rate
```promql
sum(rate(zen_flow_jobflow_phase_transitions_total[5m])) by (phase)
```

### Average Step Duration
```promql
rate(zen_flow_step_duration_seconds_sum[5m]) 
/ 
rate(zen_flow_step_duration_seconds_count[5m])
```

### Reconciliation Rate
```promql
rate(zen_flow_reconciliation_duration_seconds_count[5m])
```

### P95 Step Duration
```promql
histogram_quantile(0.95, 
  sum(rate(zen_flow_step_duration_seconds_bucket[5m])) by (le, flow, step)
)
```

---

## Grafana Dashboard

A Grafana dashboard is available at `deploy/grafana/dashboard.json` with pre-configured panels for:
- JobFlows by phase
- Step execution metrics
- Error rates
- Duration histograms
- Top JobFlows

See [Grafana Dashboard README](../deploy/grafana/README.md) for installation instructions.

---

## Alerting Rules

Prometheus alerting rules are available at `deploy/prometheus/prometheus-rules.yaml`:

- **ControllerDown**: Alerts when controller is down
- **HighReconciliationErrorRate**: Alerts on high reconciliation rates
- **StepExecutionFailures**: Alerts on step failures
- **JobFlowStuck**: Alerts on JobFlows stuck in Running (>1h)
- **JobFlowStuckWaitingApproval**: Alerts on JobFlows stuck in Paused (>24h)
- **StepStuckPendingApproval**: Alerts on steps waiting for approval (>12h)
- **SlowStepExecution**: Alerts on slow step execution
- **HighJobCreationFailureRate**: Alerts on high step failure rates

---

## See Also

- [Operator Guide](OPERATOR_GUIDE.md) - Operations and monitoring
- [Architecture](ARCHITECTURE.md) - How metrics are collected
- [Prometheus Rules](../deploy/prometheus/prometheus-rules.yaml) - Alerting rules

