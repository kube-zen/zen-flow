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

// Package main implements the zen-flow controller command-line application.
package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kube-zen/zen-flow/pkg/controller"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
	"github.com/kube-zen/zen-flow/pkg/webhook"
	"github.com/kube-zen/zen-sdk/pkg/leader"
)

const (
	// DefaultShutdownTimeout is the default timeout for graceful shutdown.
	DefaultShutdownTimeout = 30 * time.Second
)

var (
	// Version information (set via build flags).
	version   = "0.0.1-alpha"
	commit    = "unknown"
	buildDate = "unknown"
)

var (
	kubeconfig              = flag.String("kubeconfig", "", "Path to kubeconfig file. If not set, uses in-cluster config")
	masterURL               = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig")
	metricsAddr             = flag.String("metrics-addr", ":8080", "The address the metric endpoint binds to")
	webhookAddr             = flag.String("webhook-addr", ":9443", "The address the webhook endpoint binds to")
	webhookCertFile         = flag.String("webhook-cert-file", "/etc/webhook/certs/tls.crt", "Path to TLS certificate file")
	webhookKeyFile          = flag.String("webhook-key-file", "/etc/webhook/certs/tls.key", "Path to TLS private key file")
	enableLeaderElection    = flag.Bool("enable-leader-election", false, "Enable leader election for HA (uses zen-lead when enabled)")
	leaderElectionNS        = flag.String("leader-election-namespace", "", "Namespace for leader election lease (defaults to POD_NAMESPACE)")
	enableWebhook           = flag.Bool("enable-webhook", true, "Enable validating webhook server")
	insecureWebhook         = flag.Bool("insecure-webhook", false, "Allow webhook to start without TLS (testing only, not recommended for production)")
	maxConcurrentReconciles = flag.Int("max-concurrent-reconciles", 10, "Maximum number of concurrent reconciles")
)

//nolint:gocyclo // main function complexity is acceptable for initialization logic
func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// Set up signals so we handle shutdown gracefully
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Get namespace for leader election
	namespace := *leaderElectionNS
	if namespace == "" {
		namespace = os.Getenv("POD_NAMESPACE")
		if namespace == "" {
			// Try to read from service account namespace file
			if ns, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
				namespace = string(ns)
			} else {
				namespace = "zen-flow-system"
			}
		}
	}

	// Configure leader election: when enabled, use zen-lead (recommended)
	var externalWatcher *leader.Watcher
	var shouldReconcile func() bool = func() bool { return true }

	if *enableLeaderElection {
		// Use zen-lead for leader election (recommended)
		shouldReconcile = func() bool {
			if externalWatcher == nil {
				return false // Not initialized yet
			}
			return externalWatcher.GetIsLeader()
		}
		klog.Infof("Starting with zen-lead leader election. Waiting for leader role...")
	} else {
		// HA disabled - always reconcile (accept split-brain risk)
		shouldReconcile = func() bool { return true }
		klog.Warningf("Running with HA disabled. Accepting split-brain risk. Not recommended for production.")
	}

	// Create metrics recorder
	metricsRecorder := metrics.NewRecorder()

	// Build config for Kubernetes client (needed for event recorder)
	cfg, err := buildConfig(*masterURL, *kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %v", err)
	}

	// Create Kubernetes client for event recorder
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building Kubernetes client: %v", err)
	}

	// Create event recorder
	eventRecorder := controller.NewEventRecorder(kubeClient)

	// Setup controller-runtime manager (disable built-in leader election, use zen-lead instead)
	managerOptions := controller.ManagerOptions(namespace, false)
	mgr, err := controller.SetupManager(managerOptions)
	if err != nil {
		klog.Fatalf("Error setting up manager: %v", err)
	}

	// Setup external watcher for zen-lead (must be done after manager is created)
	if *enableLeaderElection {
		watcher, err := leader.NewWatcher(mgr.GetClient(), func(isLeader bool) {
			if isLeader {
				klog.Infof("Elected as leader via zen-lead. Starting reconciliation...")
			} else {
				klog.Infof("Lost leadership via zen-lead. Pausing reconciliation...")
			}
		})
		if err != nil {
			klog.Fatalf("Error creating external leader watcher: %v", err)
		}
		externalWatcher = watcher

		// Start watching in background
		go func() {
			if err := watcher.Watch(ctx); err != nil && err != context.Canceled {
				klog.Errorf("Error watching leader status: %v", err)
			}
		}()
	}

	// Setup controller with manager (use leader check when HA is enabled)
	var reconciler *controller.JobFlowReconciler
	if *enableLeaderElection {
		reconciler = controller.NewJobFlowReconcilerWithLeaderCheck(mgr, metricsRecorder, eventRecorder, shouldReconcile)
	} else {
		reconciler = controller.NewJobFlowReconciler(mgr, metricsRecorder, eventRecorder)
	}
	if err := controller.SetupControllerWithReconciler(mgr, *maxConcurrentReconciles, reconciler); err != nil {
		klog.Fatalf("Error setting up controller: %v", err)
	}

	// Start metrics server
	go startMetricsServer(*metricsAddr, mgr)

	// Start webhook server if enabled
	var webhookServer *webhook.WebhookServer
	if *enableWebhook {
		var err error
		webhookServer, err = webhook.NewWebhookServer(*webhookAddr, *webhookCertFile, *webhookKeyFile)
		if err != nil {
			klog.Fatalf("Error creating webhook server: %v", err)
		}

		// Check if TLS files exist
		certExists := false
		keyExists := false
		if _, err := os.Stat(*webhookCertFile); err == nil {
			certExists = true
		}
		if _, err := os.Stat(*webhookKeyFile); err == nil {
			keyExists = true
		}

		if certExists && keyExists {
			// TLS files exist, start with TLS
			go func() {
				if err := webhookServer.StartTLS(ctx, *webhookCertFile, *webhookKeyFile); err != nil {
					klog.Fatalf("Error starting webhook server: %v", err)
				}
			}()
			klog.Infof("Webhook server starting with TLS on %s", *webhookAddr)
		} else {
			// TLS files missing - check if insecure mode is allowed
			if !*insecureWebhook {
				klog.Fatalf("Webhook TLS certificates not found (cert: %s, key: %s). TLS is required for production. Use --insecure-webhook flag only for testing.", *webhookCertFile, *webhookKeyFile)
			}
			klog.Warningf("Webhook starting without TLS (insecure mode) - NOT RECOMMENDED FOR PRODUCTION")
			go func() {
				if err := webhookServer.Start(ctx); err != nil {
					klog.Fatalf("Error starting webhook server: %v", err)
				}
			}()
		}
	}

	// Start manager (handles leader election, cache sync, and controller lifecycle)
	klog.Info("Starting controller-runtime manager...")
	go func() {
		if err := mgr.Start(ctx); err != nil {
			klog.Fatalf("Error starting manager: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	klog.Info("Shutdown signal received, initiating graceful shutdown...")

	// Manager shutdown is handled automatically via context cancellation
	// No explicit Stop() call needed - controller-runtime handles graceful shutdown

	// Webhook server shutdown is handled automatically via context cancellation
	// No explicit Stop() call needed

	klog.Info("zen-flow controller shutdown complete")
}

// buildConfig builds a Kubernetes config from the given master URL and kubeconfig path.
func buildConfig(masterURL, kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath == "" {
		// Try in-cluster config first
		if config, err := rest.InClusterConfig(); err == nil {
			return config, nil
		}
		// Fall back to default kubeconfig location
		kubeconfigPath = clientcmd.RecommendedHomeFile
	}

	return clientcmd.BuildConfigFromFlags(masterURL, kubeconfigPath)
}

// startMetricsServer starts the Prometheus metrics server.
func startMetricsServer(addr string, mgr ctrl.Manager) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Use manager's health check endpoint if available
		// For now, just return OK - controller-runtime manager handles readiness internally
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"version":"` + version + `","commit":"` + commit + `","buildDate":"` + buildDate + `"}`))
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	klog.Infof("Starting metrics server on %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		klog.Fatalf("Error starting metrics server: %v", err)
	}
}
