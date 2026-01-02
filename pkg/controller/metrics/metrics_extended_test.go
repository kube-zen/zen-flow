/*
Copyright 2025 Kube-ZEN Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"testing"
)

func TestRecorder_RecordJobFlowPhaseTransition(t *testing.T) {
	recorder := NewRecorder()

	// Test phase transition
	recorder.RecordJobFlowPhaseTransition("flow1", "default", "", "Pending")
	recorder.RecordJobFlowPhaseTransition("flow1", "default", "Pending", "Running")
	recorder.RecordJobFlowPhaseTransition("flow1", "default", "Running", "Succeeded")
}

func TestRecorder_RecomputeJobFlowMetrics(t *testing.T) {
	recorder := NewRecorder()

	jobFlowsByPhase := map[string]map[string]int{
		"default": {
			"Pending":  2,
			"Running":   3,
			"Succeeded": 1,
		},
		"test": {
			"Pending": 1,
			"Failed":  1,
		},
	}

	recorder.RecomputeJobFlowMetrics(jobFlowsByPhase)
}

func TestRecorder_RecordStepPhaseTransition(t *testing.T) {
	recorder := NewRecorder()

	// Test step phase transition
	recorder.RecordStepPhaseTransition("flow1", "step1", "", "Pending")
	recorder.RecordStepPhaseTransition("flow1", "step1", "Pending", "Running")
	recorder.RecordStepPhaseTransition("flow1", "step1", "Running", "Succeeded")

	// Test with PendingApproval
	recorder.RecordStepPhaseTransition("flow1", "step2", "", "PendingApproval")
	recorder.RecordStepPhaseTransition("flow1", "step2", "PendingApproval", "Running")
}

func TestRecorder_RecomputeStepMetrics(t *testing.T) {
	recorder := NewRecorder()

	stepsByFlow := map[string]map[string]int{
		"flow1": {
			"Pending":  2,
			"Running":  1,
			"Succeeded": 1,
		},
		"flow2": {
			"Pending": 1,
			"Failed":  1,
		},
	}

	recorder.RecomputeStepMetrics(stepsByFlow)
}

func TestRecorder_RecordJobFlowCurrentState(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordJobFlowCurrentState("Running", "default", 5)
	recorder.RecordJobFlowCurrentState("Succeeded", "test", 10)
}

func TestRecorder_RecordStepCurrentState(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordStepCurrentState("flow1", "Running", 3)
	recorder.RecordStepCurrentState("flow1", "Succeeded", 2)
}

func TestRecorder_RecordStepError(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordStepError("flow1", "step1", "execution_failed")
	recorder.RecordStepError("flow1", "step2", "validation_failed")
	recorder.RecordStepError("flow1", "step3", "timeout")
}

func TestRecorder_RecordReconciliationError(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordReconciliationError("dag_cycle")
	recorder.RecordReconciliationError("status_update_failed")
	recorder.RecordReconciliationError("job_get_failed")
}

func TestRecorder_RecordStepRetry(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordStepRetry("flow1", "step1", 1.5)
	recorder.RecordStepRetry("flow1", "step1", 5.0)
	recorder.RecordStepRetry("flow2", "step2", 10.0)
}

func TestRecorder_RecordApprovalLatency(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordApprovalLatency("flow1", "step1", 60.0)   // 1 minute
	recorder.RecordApprovalLatency("flow1", "step2", 300.0)  // 5 minutes
	recorder.RecordApprovalLatency("flow2", "step1", 3600.0) // 1 hour
}

func TestRecorder_RecordTimeout(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordTimeout("flow1", "step1", "step_timeout")
	recorder.RecordTimeout("flow1", "", "jobflow_timeout")
	recorder.RecordTimeout("flow2", "step2", "step_timeout")
}

func TestRecorder_UpdateApprovalsPending(t *testing.T) {
	recorder := NewRecorder()

	recorder.UpdateApprovalsPending("flow1", 0)
	recorder.UpdateApprovalsPending("flow1", 2)
	recorder.UpdateApprovalsPending("flow2", 1)
}

func TestRecorder_RecordDAGComputationDuration(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordDAGComputationDuration(0.001)  // 1ms
	recorder.RecordDAGComputationDuration(0.01)   // 10ms
	recorder.RecordDAGComputationDuration(0.1)    // 100ms
}

func TestRecorder_RecordStatusUpdateDuration(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordStatusUpdateDuration(0.001)  // 1ms
	recorder.RecordStatusUpdateDuration(0.01)   // 10ms
	recorder.RecordStatusUpdateDuration(0.1)    // 100ms
}

func TestRecorder_RecordStepExecutionQueueDepth(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordStepExecutionQueueDepth("flow1", 0)
	recorder.RecordStepExecutionQueueDepth("flow1", 5)
	recorder.RecordStepExecutionQueueDepth("flow2", 10)
}

func TestRecorder_RecordAPIServerCall(t *testing.T) {
	recorder := NewRecorder()

	recorder.RecordAPIServerCall("get", "jobflow")
	recorder.RecordAPIServerCall("get", "job")
	recorder.RecordAPIServerCall("list", "jobflow")
	recorder.RecordAPIServerCall("create", "job")
	recorder.RecordAPIServerCall("update", "jobflow")
	recorder.RecordAPIServerCall("delete", "job")
}
