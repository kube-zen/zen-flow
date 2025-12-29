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

	// Test phase transitions
	recorder.RecordJobFlowPhaseTransition("flow1", "default", "", "Pending")
	recorder.RecordJobFlowPhaseTransition("flow1", "default", "Pending", "Running")
	recorder.RecordJobFlowPhaseTransition("flow1", "default", "Running", "Succeeded")
	recorder.RecordJobFlowPhaseTransition("flow2", "test-ns", "Pending", "Failed")

	// Test with empty old phase (initial state)
	recorder.RecordJobFlowPhaseTransition("flow3", "default", "", "Running")
}

func TestRecorder_RecordStepPhaseTransition(t *testing.T) {
	recorder := NewRecorder()

	// Test phase transitions
	recorder.RecordStepPhaseTransition("flow1", "step1", "", "Pending")
	recorder.RecordStepPhaseTransition("flow1", "step1", "Pending", "Running")
	recorder.RecordStepPhaseTransition("flow1", "step1", "Running", "Succeeded")
	recorder.RecordStepPhaseTransition("flow1", "step2", "Pending", "Failed")
	recorder.RecordStepPhaseTransition("flow2", "step1", "Running", "PendingApproval")

	// Test with empty old phase (initial state)
	recorder.RecordStepPhaseTransition("flow3", "step1", "", "Running")
}
