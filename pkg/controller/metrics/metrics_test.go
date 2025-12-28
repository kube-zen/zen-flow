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

func TestNewRecorder(t *testing.T) {
	recorder := NewRecorder()
	if recorder == nil {
		t.Fatal("NewRecorder returned nil")
	}
}

func TestRecorder_RecordJobFlowPhase(t *testing.T) {
	recorder := NewRecorder()
	
	// Should not panic
	recorder.RecordJobFlowPhase("Running", "default")
	recorder.RecordJobFlowPhase("Succeeded", "test-namespace")
}

func TestRecorder_RecordStepDuration(t *testing.T) {
	recorder := NewRecorder()
	
	// Should not panic
	recorder.RecordStepDuration("flow1", "step1", "success", 1.5)
	recorder.RecordStepDuration("flow1", "step2", "failure", 2.3)
}

func TestRecorder_RecordReconciliationDuration(t *testing.T) {
	recorder := NewRecorder()
	
	// Should not panic
	recorder.RecordReconciliationDuration(0.1)
	recorder.RecordReconciliationDuration(0.5)
}

func TestRecorder_RecordStepPhase(t *testing.T) {
	recorder := NewRecorder()
	
	// Should not panic
	recorder.RecordStepPhase("flow1", "Running")
	recorder.RecordStepPhase("flow1", "Succeeded")
	recorder.RecordStepPhase("flow1", "Failed")
}

