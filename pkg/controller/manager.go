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

package controller

import (
	"time"

	batchv1 "k8s.io/api/batch/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
)

// SetupManager creates and configures a controller-runtime manager.
func SetupManager(options ctrl.Options) (ctrl.Manager, error) {
	// Get Kubernetes config
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	// Create manager
	mgr, err := ctrl.NewManager(cfg, options)
	if err != nil {
		return nil, err
	}

	// Add JobFlow to scheme
	if err := v1alpha1.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}

	return mgr, nil
}

// SetupController sets up the JobFlow controller with the manager.
func SetupController(mgr ctrl.Manager, maxConcurrentReconciles int, metricsRecorder *metrics.Recorder, eventRecorder *EventRecorder) error {
	reconciler := NewJobFlowReconciler(mgr, metricsRecorder, eventRecorder)

	// Setup controller with builder
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.JobFlow{}).
		Owns(&batchv1.Job{})

	// Set max concurrent reconciles if specified (controller-runtime uses default if not set)
	if maxConcurrentReconciles > 0 {
		controllerOpts := controller.Options{
			MaxConcurrentReconciles: maxConcurrentReconciles,
		}
		builder = builder.WithOptions(controllerOpts)
	}

	return reconciler.SetupWithManager(builder)
}

// ManagerOptions returns default manager options.
// Note: Metrics server is handled separately in main.go, not via manager Options.
func ManagerOptions(namespace string, enableLeaderElection bool) ctrl.Options {
	opts := ctrl.Options{
		Scheme:                  nil, // Will be set by SetupManager
		LeaderElection:          enableLeaderElection,
		LeaderElectionID:        "zen-flow-controller-leader-election",
		LeaderElectionNamespace: namespace,
		LeaseDuration:           func() *time.Duration { d := 15 * time.Second; return &d }(),
		RenewDeadline:           func() *time.Duration { d := 10 * time.Second; return &d }(),
		RetryPeriod:             func() *time.Duration { d := 2 * time.Second; return &d }(),
	}

	return opts
}
