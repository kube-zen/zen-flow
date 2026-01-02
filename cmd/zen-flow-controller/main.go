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
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	flowv1alpha1 "github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/controller"
	"github.com/kube-zen/zen-flow/pkg/controller/metrics"
	"github.com/kube-zen/zen-flow/pkg/webhook"
	"github.com/kube-zen/zen-sdk/pkg/leader"
	sdklifecycle "github.com/kube-zen/zen-sdk/pkg/lifecycle"
	sdklog "github.com/kube-zen/zen-sdk/pkg/logging"
	"github.com/kube-zen/zen-sdk/pkg/zenlead"
)

var (
	scheme   = runtime.NewScheme()
	logger   *sdklog.Logger
	setupLog *sdklog.Logger
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(flowv1alpha1.AddToScheme(scheme))
}

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
	leaderElectionMode      = flag.String("leader-election-mode", "builtin", "Leader election mode: builtin (default), zenlead, or disabled")
	leaderElectionID        = flag.String("leader-election-id", "", "The ID for leader election (default: zen-flow-controller-leader-election). Required for builtin mode.")
	leaderElectionLeaseName = flag.String("leader-election-lease-name", "", "The LeaderGroup CRD name (required for zenlead mode)")
	enableWebhook           = flag.Bool("enable-webhook", true, "Enable validating webhook server")
	insecureWebhook         = flag.Bool("insecure-webhook", false, "Allow webhook to start without TLS (testing only, not recommended for production)")
	maxConcurrentReconciles = flag.Int("max-concurrent-reconciles", 10, "Maximum number of concurrent reconciles")
)

//nolint:gocyclo // main function complexity is acceptable for initialization logic
func main() {
	flag.Parse()

	// Initialize zen-sdk logger (configures controller-runtime logger automatically)
	logger = sdklog.NewLogger("zen-flow")
	setupLog = logger.WithComponent("setup")

	// Set up signals so we handle shutdown gracefully using zen-sdk lifecycle
	ctx, cancel := sdklifecycle.ShutdownContext(context.Background(), "zen-flow")
	defer cancel()

	// OpenTelemetry tracing initialization can be added here when zen-sdk/pkg/observability is available
	// For now, continue without tracing

	// Build config for Kubernetes client (needed for event recorder)
	cfg, err := buildConfig(*masterURL, *kubeconfig)
	if err != nil {
		setupLog.Error(err, "Error building kubeconfig", sdklog.ErrorCode("CONFIG_ERROR"))
		os.Exit(1)
	}

	// Apply REST config defaults (via zen-sdk helper)
	zenlead.ControllerRuntimeDefaults(cfg)

	// Create Kubernetes client for event recorder
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		setupLog.Error(err, "Error building Kubernetes client", sdklog.ErrorCode("CLIENT_ERROR"))
		os.Exit(1)
	}

	// Create metrics recorder
	metricsRecorder := metrics.NewRecorder()

	// Create event recorder
	eventRecorder := controller.NewEventRecorder(kubeClient)

	// Setup controller-runtime manager
	baseOpts := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: *metricsAddr,
		},
		HealthProbeBindAddress: ":8081",
	}

	// Configure leader election using zenlead package (Profiles B/C)
	var leConfig zenlead.LeaderElectionConfig
	namespace, err := leader.RequirePodNamespace()
	if err != nil {
		setupLog.Error(err, "Failed to determine pod namespace", sdklog.ErrorCode("NAMESPACE_ERROR"))
		os.Exit(1)
	}

	// Determine election ID (default if not provided)
	electionID := *leaderElectionID
	if electionID == "" {
		electionID = "zen-flow-controller-leader-election"
	}

	// Configure based on mode
	switch *leaderElectionMode {
	case "builtin":
		leConfig = zenlead.LeaderElectionConfig{
			Mode:       zenlead.BuiltIn,
			ElectionID: electionID,
			Namespace:  namespace,
		}
		setupLog.Info("Leader election mode: builtin (Profile B)", sdklog.Operation("leader_election_config"))
	case "zenlead":
		if *leaderElectionLeaseName == "" {
			setupLog.Error(fmt.Errorf("--leader-election-lease-name is required when --leader-election-mode=zenlead"), "invalid configuration", sdklog.ErrorCode("INVALID_CONFIG"))
			os.Exit(1)
		}
		leConfig = zenlead.LeaderElectionConfig{
			Mode:      zenlead.ZenLeadManaged,
			LeaseName: *leaderElectionLeaseName,
			Namespace: namespace,
		}
		setupLog.Info("Leader election mode: zenlead managed (Profile C)", sdklog.Operation("leader_election_config"), sdklog.String("leaseName", *leaderElectionLeaseName))
	case "disabled":
		leConfig = zenlead.LeaderElectionConfig{
			Mode: zenlead.Disabled,
		}
		setupLog.Warn("Leader election disabled - single replica only (unsafe if replicas > 1)", sdklog.Operation("leader_election_config"))
	default:
		setupLog.Error(fmt.Errorf("invalid --leader-election-mode: %q (must be builtin, zenlead, or disabled)", *leaderElectionMode), "invalid configuration", sdklog.ErrorCode("INVALID_CONFIG"))
		os.Exit(1)
	}

	// Prepare manager options with leader election
	mgrOpts, err := zenlead.PrepareManagerOptions(&baseOpts, &leConfig)
	if err != nil {
		setupLog.Error(err, "Failed to prepare manager options", sdklog.ErrorCode("MANAGER_OPTIONS_ERROR"))
		os.Exit(1)
	}

	// Get replica count from environment (set by Helm/Kubernetes)
	replicaCount := 1
	if rcStr := os.Getenv("REPLICA_COUNT"); rcStr != "" {
		if rc, err := strconv.Atoi(rcStr); err == nil {
			replicaCount = rc
		}
	}

	// Enforce safe HA configuration
	if err := zenlead.EnforceSafeHA(replicaCount, mgrOpts.LeaderElection); err != nil {
		setupLog.Error(err, "Unsafe HA configuration", sdklog.ErrorCode("UNSAFE_HA_CONFIG"))
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(cfg, mgrOpts)
	if err != nil {
		setupLog.Error(err, "Error setting up manager", sdklog.ErrorCode("MANAGER_SETUP_ERROR"))
		os.Exit(1)
	}

	// Setup controller with manager (no leader check needed - Manager handles it)
	reconciler := controller.NewJobFlowReconciler(mgr, metricsRecorder, eventRecorder)
	if err := controller.SetupControllerWithReconciler(mgr, *maxConcurrentReconciles, reconciler); err != nil {
		setupLog.Error(err, "Error setting up controller", sdklog.ErrorCode("CONTROLLER_SETUP_ERROR"))
		os.Exit(1)
	}

	// Start metrics server
	go startMetricsServer(*metricsAddr, mgr)

	// Start webhook server if enabled
	var webhookServer *webhook.WebhookServer
	if *enableWebhook {
		var err error
		webhookServer, err = webhook.NewWebhookServer(*webhookAddr, *webhookCertFile, *webhookKeyFile)
		if err != nil {
			setupLog.Error(err, "Error creating webhook server", sdklog.ErrorCode("WEBHOOK_CREATE_ERROR"))
			os.Exit(1)
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
					setupLog.Error(err, "Error starting webhook server", sdklog.ErrorCode("WEBHOOK_START_ERROR"))
					os.Exit(1)
				}
			}()
			setupLog.Info("Webhook server starting with TLS", sdklog.String("address", *webhookAddr), sdklog.Component("webhook"))
		} else {
			// TLS files missing - check if insecure mode is allowed
			if !*insecureWebhook {
				setupLog.Error(fmt.Errorf("webhook TLS certificates not found (cert: %s, key: %s). TLS is required for production. Use --insecure-webhook flag only for testing", *webhookCertFile, *webhookKeyFile), "TLS certificates missing", sdklog.ErrorCode("TLS_CERT_MISSING"))
				os.Exit(1)
			}
			setupLog.Warn("Webhook starting without TLS (insecure mode) - NOT RECOMMENDED FOR PRODUCTION", sdklog.Component("webhook"))
			go func() {
				if err := webhookServer.Start(ctx); err != nil {
					setupLog.Error(err, "Error starting webhook server", sdklog.ErrorCode("WEBHOOK_START_ERROR"))
					os.Exit(1)
				}
			}()
		}
	}

	// Start manager (handles leader election, cache sync, and controller lifecycle)
	setupLog.Info("Starting controller-runtime manager", sdklog.Operation("start"))
	go func() {
		if err := mgr.Start(ctx); err != nil {
			setupLog.Error(err, "Error starting manager", sdklog.ErrorCode("MANAGER_START_ERROR"))
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	setupLog.Info("Shutdown signal received, initiating graceful shutdown", sdklog.Operation("shutdown"))

	// Manager shutdown is handled automatically via context cancellation
	// No explicit Stop() call needed - controller-runtime handles graceful shutdown

	// Webhook server shutdown is handled automatically via context cancellation
	// No explicit Stop() call needed

	setupLog.Info("zen-flow controller shutdown complete", sdklog.Operation("shutdown"))
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

	serverLogger := logger.WithComponent("metrics-server")
	serverLogger.Info("Starting metrics server", sdklog.String("address", addr))
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		serverLogger.Error(err, "Error starting metrics server", sdklog.ErrorCode("METRICS_SERVER_ERROR"))
		os.Exit(1)
	}
}
