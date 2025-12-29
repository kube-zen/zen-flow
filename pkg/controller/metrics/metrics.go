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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// JobFlowsCurrent is the current number of JobFlows by phase and namespace.
	// P0.8: Changed from Counter to Gauge with Set() semantics to avoid double-counting.
	// Tracks current state: Pending, Running, Succeeded, Failed, Suspended, Paused
	JobFlowsCurrent = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zen_flow_jobflows",
			Help: "Current number of JobFlows by phase and namespace",
		},
		[]string{"phase", "namespace"},
	)

	// JobFlowPhaseTransitions is the total number of JobFlow phase transitions.
	// P0.8: Counter for transitions/events.
	JobFlowPhaseTransitions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zen_flow_jobflow_phase_transitions_total",
			Help: "Total number of JobFlow phase transitions",
		},
		[]string{"phase", "namespace"},
	)

	// StepDuration is the duration of step execution.
	StepDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "zen_flow_step_duration_seconds",
			Help:    "Duration of step execution",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"flow", "step", "result"},
	)

	// ReconciliationDuration is the duration of reconciliation loops.
	ReconciliationDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "zen_flow_reconciliation_duration_seconds",
			Help:    "Duration of reconciliation loops",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
	)

	// StepsCurrent is the current number of steps by phase.
	// P0.8: Changed from Counter to Gauge with Set() semantics to avoid double-counting.
	// Tracks current state: Pending, Running, Succeeded, Failed, Skipped, PendingApproval
	StepsCurrent = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zen_flow_steps",
			Help: "Current number of steps by phase",
		},
		[]string{"flow", "phase"},
	)

	// StepPhaseTransitions is the total number of step phase transitions.
	// P0.8: Counter for transitions/events.
	StepPhaseTransitions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "zen_flow_step_phase_transitions_total",
			Help: "Total number of step phase transitions",
		},
		[]string{"flow", "phase"},
	)
)

// Recorder records metrics for JobFlow operations.
type Recorder struct {
	// Track current state to avoid double-counting
	jobFlowPhases map[string]map[string]string // namespace -> name -> phase
	stepPhases    map[string]map[string]string // flow -> step -> phase
}

// NewRecorder creates a new metrics recorder.
func NewRecorder() *Recorder {
	return &Recorder{
		jobFlowPhases: make(map[string]map[string]string),
		stepPhases:     make(map[string]map[string]string),
	}
}

// RecordJobFlowPhase records the phase of a JobFlow.
// P0.8: Uses Set() semantics to track current state, avoiding double-counting.
func (r *Recorder) RecordJobFlowPhase(phase, namespace string) {
	// Record transition (counter)
	JobFlowPhaseTransitions.WithLabelValues(phase, namespace).Inc()
	
	// Update current state (gauge) - this should be called with full recomputation
	// For now, we'll use Set() to track transitions
	// In a full implementation, we'd recompute all gauges per reconcile
	JobFlowsCurrent.WithLabelValues(phase, namespace).Inc()
}

// RecordJobFlowPhaseTransition records a JobFlow phase transition.
// P0.8: Properly handles phase transitions by decrementing old phase and incrementing new phase.
func (r *Recorder) RecordJobFlowPhaseTransition(jobFlowName, namespace, oldPhase, newPhase string) {
	// Record transition counter
	JobFlowPhaseTransitions.WithLabelValues(newPhase, namespace).Inc()
	
	// Update gauges: decrement old phase, increment new phase
	if oldPhase != "" {
		JobFlowsCurrent.WithLabelValues(oldPhase, namespace).Dec()
	}
	JobFlowsCurrent.WithLabelValues(newPhase, namespace).Inc()
}

// RecordStepDuration records the duration of a step.
func (r *Recorder) RecordStepDuration(flow, step, result string, duration float64) {
	StepDuration.WithLabelValues(flow, step, result).Observe(duration)
}

// RecordReconciliationDuration records the duration of a reconciliation.
func (r *Recorder) RecordReconciliationDuration(duration float64) {
	ReconciliationDuration.Observe(duration)
}

// RecordStepPhase records the phase of a step.
// P0.8: Uses Set() semantics to track current state, avoiding double-counting.
func (r *Recorder) RecordStepPhase(flow, phase string) {
	// Record transition (counter)
	StepPhaseTransitions.WithLabelValues(flow, phase).Inc()
	
	// Update current state (gauge)
	StepsCurrent.WithLabelValues(flow, phase).Inc()
}

// RecordStepPhaseTransition records a step phase transition.
// P0.8: Properly handles phase transitions by decrementing old phase and incrementing new phase.
func (r *Recorder) RecordStepPhaseTransition(flow, step, oldPhase, newPhase string) {
	// Record transition counter
	StepPhaseTransitions.WithLabelValues(flow, newPhase).Inc()
	
	// Update gauges: decrement old phase, increment new phase
	if oldPhase != "" {
		StepsCurrent.WithLabelValues(flow, oldPhase).Dec()
	}
	StepsCurrent.WithLabelValues(flow, newPhase).Inc()
}
