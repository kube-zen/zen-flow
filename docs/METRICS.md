# Metrics Documentation

This document describes all Prometheus metrics exposed by the zen-flow controller.

## Metrics Endpoint

The zen-flow controller exposes metrics on the `/metrics` endpoint, defaulting to port `8080`.

## Available Metrics

### `zen_flow_jobflows_total`
**Type**: Gauge  
**Description**: Total number of JobFlows  
**Labels**:
- `phase`: JobFlow phase (Pending, Running, Succeeded, Failed, Suspended)
- `namespace`: Namespace of the JobFlow

**Example**:
```
zen_flow_jobflows_total{phase="Running",namespace="default"} 5
zen_flow_jobflows_total{phase="Succeeded",namespace="default"} 10
```

---

### `zen_flow_steps_total`
**Type**: Gauge  
**Description**: Total number of steps  
**Labels**:
- `flow`: Name of the JobFlow
- `phase`: Step phase (Pending, Running, Succeeded, Failed, Skipped)

**Example**:
```
zen_flow_steps_total{flow="data-pipeline",phase="Running"} 3
zen_flow_steps_total{flow="data-pipeline",phase="Succeeded"} 7
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

### JobFlows by Phase
```promql
sum by (phase) (zen_flow_jobflows_total)
```

### Step Success Rate
```promql
sum(rate(zen_flow_step_duration_seconds_count{result="success"}[5m])) 
/ 
sum(rate(zen_flow_step_duration_seconds_count[5m]))
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
- **HighReconciliationErrorRate**: Alerts on high error rates
- **StepExecutionFailures**: Alerts on step failures
- **JobFlowStuck**: Alerts on JobFlows stuck in Running
- **SlowStepExecution**: Alerts on slow step execution
- **HighJobCreationFailureRate**: Alerts on job creation failures

---

## See Also

- [Operator Guide](OPERATOR_GUIDE.md) - Operations and monitoring
- [Architecture](ARCHITECTURE.md) - How metrics are collected
- [Prometheus Rules](../deploy/prometheus/prometheus-rules.yaml) - Alerting rules

