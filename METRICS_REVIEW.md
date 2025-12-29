# Metrics, Prometheus Rules, and Grafana Dashboard Review

## Summary

After implementing controller-runtime migration, new features (TTL, retries, timeouts, concurrency, pod failure policies, when conditions, manual approval), and fixing metrics semantics, this document summarizes the review and updates made to metrics, Prometheus rules, and Grafana dashboards.

## Changes Made

### 1. Metrics (`pkg/controller/metrics/metrics.go`)

**Fixed:**
- Renamed `JobFlowsTotal` to `JobFlowsCurrent` (better reflects gauge semantics)
- Updated comments to document all supported phases:
  - JobFlow phases: `Pending`, `Running`, `Succeeded`, `Failed`, `Suspended`, `Paused`
  - Step phases: `Pending`, `Running`, `Succeeded`, `Failed`, `Skipped`, `PendingApproval`

**Current Metrics:**
- `zen_flow_jobflows` (Gauge) - Current number of JobFlows by phase and namespace
- `zen_flow_jobflow_phase_transitions_total` (Counter) - Total phase transitions
- `zen_flow_steps` (Gauge) - Current number of steps by phase
- `zen_flow_step_phase_transitions_total` (Counter) - Total step phase transitions
- `zen_flow_step_duration_seconds` (Histogram) - Step execution duration
- `zen_flow_reconciliation_duration_seconds` (Histogram) - Reconciliation duration

### 2. Prometheus Rules (`deploy/prometheus/prometheus-rules.yaml`)

**Updated:**
- Fixed metric names from `zen_flow_jobflows_total` → `zen_flow_jobflows`
- Fixed metric names from `zen_flow_steps_total` → `zen_flow_steps`
- Fixed metric names from `zen_flow_step_phase_total` → `zen_flow_step_phase_transitions_total`
- Updated reconciliation error rate alert (removed non-existent `status` label)

**Added:**
- `ZenFlowJobFlowStuckWaitingApproval` - Alerts when JobFlows are stuck in `Paused` state for >24h
- `ZenFlowStepStuckPendingApproval` - Alerts when steps are waiting for approval for >12h

**Alerts:**
1. `ZenFlowControllerDown` - Controller is down
2. `ZenFlowHighReconciliationErrorRate` - High reconciliation rate
3. `ZenFlowStepExecutionFailures` - High step failure rate
4. `ZenFlowJobFlowStuck` - JobFlow stuck in Running >1h
5. `ZenFlowJobFlowStuckWaitingApproval` - JobFlow stuck in Paused >24h (NEW)
6. `ZenFlowStepStuckPendingApproval` - Step waiting for approval >12h (NEW)
7. `ZenFlowSlowStepExecution` - Slow step execution (P95 >300s)
8. `ZenFlowHighJobCreationFailureRate` - High step failure rate
9. `ZenFlowNoActiveJobFlows` - No active JobFlows

### 3. Grafana Dashboard (`deploy/grafana/dashboard.json`)

**Updated:**
- Fixed all metric names to match current implementation:
  - `zen_flow_jobflows_total` → `zen_flow_jobflows`
  - `zen_flow_steps_total` → `zen_flow_steps`
  - Updated transition rate queries to use `zen_flow_step_phase_transitions_total`

**Added:**
- **Paused** phase to "JobFlows by Phase" panel
- **Suspended** phase to "JobFlows by Phase" panel
- **PendingApproval** phase to "Step Execution Over Time" panel
- **Manual Approval Steps Waiting** panel - Shows steps waiting for approval
- **JobFlows Paused (Waiting Approval)** panel - Shows paused JobFlows
- **Phase Transition Rate** panel - Shows JobFlow phase transition rates
- **Step Phase Transitions** panel - Shows step phase transition rates

**Panels:**
1. JobFlows by Phase (includes Paused, Suspended)
2. Total Steps Executed
3. Step Execution Rate
4. Reconciliation Rate
5. Step Execution Over Time (includes PendingApproval)
6. Step Duration (P95/P50)
7. Reconciliation Duration
8. Steps by Phase (pie chart)
9. JobFlows by Namespace
10. Top JobFlows by Step Count
11. Step Success Rate
12. Manual Approval Steps Waiting (NEW)
13. JobFlows Paused (NEW)
14. Phase Transition Rate (NEW)
15. Step Phase Transitions (NEW)

### 4. Metrics Documentation (`docs/METRICS.md`)

**Updated:**
- Fixed metric names and descriptions
- Added documentation for new transition metrics
- Added examples for `Paused` and `PendingApproval` phases
- Updated Prometheus queries to use correct metric names
- Added queries for manual approval monitoring
- Updated alerting rules list

## Verification

✅ All code compiles successfully
✅ Dashboard JSON is valid
✅ Metric names are consistent across all files
✅ All phases are documented and included in dashboards
✅ New alerts for manual approval are added

## Next Steps

1. Deploy updated Prometheus rules and Grafana dashboard
2. Verify metrics are being collected correctly
3. Test alerts in a staging environment
4. Monitor manual approval metrics in production

