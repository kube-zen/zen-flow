#!/bin/bash
# Setup script for zen-flow integration tests with kind
# Creates a kind cluster, deploys CRDs, RBAC, and zen-flow controller

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CLUSTER_NAME="${CLUSTER_NAME:-zen-flow-integration}"
KUBECONFIG_PATH="${KUBECONFIG_PATH:-${HOME}/.kube/zen-flow-integration-config}"

log_info() {
    echo "[INFO] $*" >&2
}

log_error() {
    echo "[ERROR] $*" >&2
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing=0
    
    if ! command -v kind >/dev/null 2>&1; then
        log_error "kind is not installed. Install from https://kind.sigs.k8s.io/"
        missing=1
    fi
    
    if ! command -v kubectl >/dev/null 2>&1; then
        log_error "kubectl is not installed"
        missing=1
    fi
    
    if ! command -v docker >/dev/null 2>&1 && ! command -v podman >/dev/null 2>&1; then
        log_error "docker or podman is required"
        missing=1
    fi
    
    if [ $missing -eq 1 ]; then
        exit 1
    fi
    
    log_info "All prerequisites met"
}

create_cluster() {
    log_info "Creating kind cluster: $CLUSTER_NAME"
    
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_info "Cluster $CLUSTER_NAME already exists"
        return 0
    fi
    
    kind create cluster \
        --name "$CLUSTER_NAME" \
        --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 8080
    protocol: TCP
  - containerPort: 443
    hostPort: 8443
    protocol: TCP
EOF
    
    log_info "Cluster created successfully"
}

export_kubeconfig() {
    log_info "Exporting kubeconfig..."
    kind get kubeconfig --name "$CLUSTER_NAME" > "$KUBECONFIG_PATH"
    log_info "Kubeconfig exported to: $KUBECONFIG_PATH"
    log_info "To use this cluster, run: export KUBECONFIG=$KUBECONFIG_PATH"
}

install_crds() {
    log_info "Installing CRDs..."
    kubectl apply --kubeconfig="$KUBECONFIG_PATH" -f "$PROJECT_ROOT/deploy/crds/"
    log_info "Waiting for CRDs to be established..."
    kubectl wait --kubeconfig="$KUBECONFIG_PATH" --for=condition=established --timeout=60s \
        crd/jobflows.workflow.kube-zen.io || true
    log_info "CRDs installed"
}

install_rbac() {
    log_info "Installing RBAC..."
    kubectl apply --kubeconfig="$KUBECONFIG_PATH" -f "$PROJECT_ROOT/deploy/manifests/rbac.yaml"
    log_info "RBAC installed"
}

build_and_load_image() {
    log_info "Building zen-flow image..."
    
    local image_name="kubezen/zen-flow-controller:integration-test"
    
    # Build image from project root
    cd "$PROJECT_ROOT"
    docker build -t "$image_name" -f Dockerfile . || {
        log_error "Failed to build image"
        exit 1
    }
    
    # Load into kind
    log_info "Loading image into kind cluster..."
    kind load docker-image "$image_name" --name "$CLUSTER_NAME" || {
        log_error "Failed to load image into kind"
        exit 1
    }
    
    log_info "Image built and loaded: $image_name"
    echo "$image_name"
}

deploy_zen_flow() {
    log_info "Deploying zen-flow..."
    
    local image_name="${1:-kubezen/zen-flow-controller:integration-test}"
    local namespace="zen-flow-system"
    
    # Create namespace
    kubectl create namespace "$namespace" --kubeconfig="$KUBECONFIG_PATH" --dry-run=client -o yaml | \
        kubectl apply --kubeconfig="$KUBECONFIG_PATH" -f -
    
    # Create Deployment manifest
    cat <<EOF | kubectl apply --kubeconfig="$KUBECONFIG_PATH" -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zen-flow-controller
  namespace: $namespace
  labels:
    app.kubernetes.io/name: zen-flow
    app.kubernetes.io/component: controller
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: zen-flow
      app.kubernetes.io/component: controller
  template:
    metadata:
      labels:
        app.kubernetes.io/name: zen-flow
        app.kubernetes.io/component: controller
    spec:
      serviceAccountName: zen-flow-controller
      containers:
      - name: controller
        image: $image_name
        imagePullPolicy: Never
        ports:
        - containerPort: 8080
          name: metrics
          protocol: TCP
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 256Mi
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 5
          periodSeconds: 10
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
            scheme: HTTP
          initialDelaySeconds: 10
          periodSeconds: 30
EOF
    
    log_info "Waiting for deployment to be ready..."
    kubectl wait --kubeconfig="$KUBECONFIG_PATH" \
        --for=condition=available \
        --timeout=120s \
        deployment/zen-flow-controller \
        -n "$namespace" || {
        log_error "Deployment failed to become ready"
        kubectl get pods --kubeconfig="$KUBECONFIG_PATH" -n "$namespace"
        exit 1
    }
    
    log_info "zen-flow deployed successfully"
}

wait_for_ready() {
    log_info "Waiting for zen-flow to be ready..."
    local namespace="zen-flow-system"
    
    # Wait for controller pod
    kubectl wait --kubeconfig="$KUBECONFIG_PATH" \
        --for=condition=ready \
        --timeout=60s \
        pod -l app.kubernetes.io/name=zen-flow,app.kubernetes.io/component=controller \
        -n "$namespace" || {
        log_error "Controller pod not ready"
        kubectl logs --kubeconfig="$KUBECONFIG_PATH" \
            -l app.kubernetes.io/name=zen-flow,app.kubernetes.io/component=controller \
            -n "$namespace" || true
        exit 1
    }
    
    log_info "zen-flow is ready"
}

cleanup_cluster() {
    log_info "Cleaning up cluster..."
    kind delete cluster --name "$CLUSTER_NAME" || true
    rm -f "$KUBECONFIG_PATH"
    log_info "Cleanup complete"
}

print_usage() {
    cat <<EOF
Usage: $0 [COMMAND]

Commands:
    create      Create kind cluster and deploy zen-flow
    delete      Delete kind cluster
    kubeconfig  Show kubeconfig export command
    help        Show this help message

Environment Variables:
    CLUSTER_NAME       Name of the kind cluster (default: zen-flow-integration)
    KUBECONFIG_PATH    Path to kubeconfig file (default: ~/.kube/zen-flow-integration-config)

Examples:
    # Create cluster and deploy zen-flow
    $0 create

    # Delete cluster
    $0 delete

    # Export kubeconfig
    export KUBECONFIG=\$($0 kubeconfig)
EOF
}

main() {
    case "${1:-help}" in
        create)
            check_prerequisites
            create_cluster
            export_kubeconfig
            install_crds
            install_rbac
            image_name=$(build_and_load_image)
            deploy_zen_flow "$image_name"
            wait_for_ready
            log_info "✅ zen-flow integration test environment is ready!"
            log_info "Export kubeconfig: export KUBECONFIG=$KUBECONFIG_PATH"
            ;;
        delete)
            cleanup_cluster
            ;;
        kubeconfig)
            echo "$KUBECONFIG_PATH"
            ;;
        help|--help|-h)
            print_usage
            ;;
        *)
            log_error "Unknown command: $1"
            print_usage
            exit 1
            ;;
    esac
}

main "$@"

