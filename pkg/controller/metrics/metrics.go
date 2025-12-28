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
	// JobFlowsTotal is the total number of JobFlows.
	JobFlowsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zen_flow_jobflows_total",
			Help: "Total number of JobFlows",
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

	// StepsTotal is the total number of steps.
	StepsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "zen_flow_steps_total",
			Help: "Total number of steps",
		},
		[]string{"flow", "phase"},
	)
)

// Recorder records metrics for JobFlow operations.
type Recorder struct{}

// NewRecorder creates a new metrics recorder.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// RecordJobFlowPhase records the phase of a JobFlow.
func (r *Recorder) RecordJobFlowPhase(phase, namespace string) {
	JobFlowsTotal.WithLabelValues(phase, namespace).Inc()
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
func (r *Recorder) RecordStepPhase(flow, phase string) {
	StepsTotal.WithLabelValues(flow, phase).Inc()
}

