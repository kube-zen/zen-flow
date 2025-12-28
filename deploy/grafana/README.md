# zen-flow Grafana Dashboard

This directory contains the Grafana dashboard for monitoring zen-flow controller.

## Installation

### Option 1: Using Grafana UI

1. Open Grafana and navigate to Dashboards → Import
2. Copy the contents of `dashboard.json`
3. Paste into the import dialog
4. Click "Load"
5. Select your Prometheus data source
6. Click "Import"

### Option 2: Using kubectl

```bash
# Create ConfigMap from dashboard JSON
kubectl create configmap zen-flow-dashboard \
  --from-file=dashboard.json=deploy/grafana/dashboard.json \
  -n grafana

# Or if using Grafana Operator
kubectl apply -f - <<EOF
apiVersion: integreatly.org/v1alpha1
kind: GrafanaDashboard
metadata:
  name: zen-flow-dashboard
  namespace: grafana
spec:
  json: |
    $(cat deploy/grafana/dashboard.json | sed 's/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')
EOF
```

## Dashboard Panels

The dashboard includes the following panels:

1. **JobFlows by Phase** - Current count of JobFlows by phase (Running, Succeeded, Failed, Pending)
2. **Total Steps Executed** - Total number of steps executed in the last 5 minutes
3. **Step Execution Rate** - Rate of step execution per second
4. **Reconciliation Rate** - Rate of reconciliation loops per second
5. **Step Execution Over Time** - Graph showing step execution trends
6. **Step Duration (P95)** - 95th percentile step execution duration
7. **Reconciliation Duration** - Duration of reconciliation loops
8. **Steps by Phase** - Pie chart showing step distribution by phase
9. **JobFlows by Namespace** - JobFlow distribution across namespaces
10. **Top JobFlows by Step Count** - Top 10 JobFlows by number of steps
11. **Step Success Rate** - Overall step success rate percentage

## Metrics Used

The dashboard uses the following Prometheus metrics:

- `zen_flow_jobflows_total` - Total JobFlows by phase and namespace
- `zen_flow_steps_total` - Total steps by flow and phase
- `zen_flow_step_duration_seconds` - Step execution duration histogram
- `zen_flow_reconciliation_duration_seconds` - Reconciliation duration histogram

## Requirements

- Prometheus configured to scrape zen-flow controller metrics
- Grafana with Prometheus data source configured
- zen-flow controller running with metrics enabled

