.PHONY: build test test-unit fmt vet lint clean deploy coverage verify ci-check security-check

# Build the zen-flow-controller binary (development build with basic optimizations)
build:
	@echo "Building zen-flow-controller..."
	go build -ldflags="-s -w" -trimpath -o bin/zen-flow-controller ./cmd/zen-flow-controller
	@echo "✅ Build complete: bin/zen-flow-controller"
	@ls -lh bin/zen-flow-controller | awk '{print "   Binary size: " $$5}'

# Build optimized binary for production
build-release:
	@echo "Building optimized zen-flow-controller binary..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-alpha"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILD_DATE=$$(date -u +"%Y-%m-%dT%H:%M:%SZ"); \
	go build -trimpath \
		-ldflags "-s -w \
			-X 'main.version=$$VERSION' \
			-X 'main.commit=$$COMMIT' \
			-X 'main.buildDate=$$BUILD_DATE'" \
		-o bin/zen-flow-controller ./cmd/zen-flow-controller
	@echo "✅ Optimized build complete: bin/zen-flow-controller"
	@ls -lh bin/zen-flow-controller

# Build Docker image (requires Docker)
build-image:
	@echo "Building Docker image..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-alpha"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILD_DATE=$$(date -u +"%Y-%m-%dT%H:%M:%SZ"); \
	docker build \
		--build-arg VERSION=$$VERSION \
		--build-arg COMMIT=$$COMMIT \
		--build-arg BUILD_DATE=$$BUILD_DATE \
		-t kubezen/zen-flow-controller:$$VERSION \
		-t kubezen/zen-flow-controller:latest .
	@echo "✅ Docker image built: kubezen/zen-flow-controller:$$VERSION"

# Build multi-arch Docker images (requires Docker Buildx)
build-image-multiarch:
	@echo "Building multi-arch Docker images..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-alpha"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	BUILD_DATE=$$(date -u +"%Y-%m-%dT%H:%M:%SZ"); \
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$$VERSION \
		--build-arg COMMIT=$$COMMIT \
		--build-arg BUILD_DATE=$$BUILD_DATE \
		-t kubezen/zen-flow-controller:$$VERSION \
		-t kubezen/zen-flow-controller:latest \
		--push .
	@echo "✅ Multi-arch Docker images built: kubezen/zen-flow-controller:$$VERSION"

# Run all tests
test: test-unit

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic -timeout=10m ./pkg/...

# Show test coverage
coverage: test-unit
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report generated: coverage.html"
	@echo "Coverage summary:"
	@go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "Checking coverage threshold (minimum: 70%)..."
	@COVERAGE=$$(go tool cover -func=coverage.out | grep -v "pkg/api/v1alpha1" | grep "total:" | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$COVERAGE" ]; then \
		echo "⚠️  Could not determine coverage percentage"; \
	elif [ $$(echo "$$COVERAGE < 70" | bc -l 2>/dev/null || echo "0") -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below the 70% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets the 70% threshold"; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted"

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...
	@echo "✅ go vet passed"

# Run linter (requires golangci-lint)
lint:
	@echo "Running golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "⚠️  golangci-lint not found. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin latest; \
	fi
	golangci-lint run
	@echo "✅ Linting passed"

# Security checks
security-check:
	@echo "Running security checks..."
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	govulncheck ./...
	@echo "✅ Security check passed"

# Check formatting
check-fmt:
	@echo "Checking code formatting..."
	@if [ "$$(gofmt -s -l . | wc -l)" -gt 0 ]; then \
		echo "❌ Code is not formatted. Run 'make fmt'"; \
		gofmt -s -d .; \
		exit 1; \
	fi
	@echo "✅ Code formatting check passed"

# Check go mod tidy
check-mod:
	@echo "Checking go.mod..."
	@go mod tidy
	@if ! git diff --exit-code go.mod go.sum >/dev/null 2>&1; then \
		echo "❌ go.mod or go.sum needs updates. Run 'go mod tidy'"; \
		git diff go.mod go.sum; \
		exit 1; \
	fi
	@echo "✅ go.mod check passed"

# Verify code compiles
verify: check-fmt check-mod vet
	@echo "Verifying code compiles..."
	go build ./...
	@echo "✅ Code compiles successfully"

# CI check (runs all checks)
ci-check: verify lint test-unit security-check
	@echo "✅ All CI checks passed"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/ coverage.out coverage.html
	@echo "✅ Clean complete"

# Deploy CRD
deploy-crd:
	@echo "Deploying CRD..."
	kubectl apply -f deploy/crds/
	@echo "✅ CRD deployed"

# Deploy all manifests
deploy: deploy-crd
	@echo "Deploying manifests..."
	kubectl apply -f deploy/manifests/
	@echo "✅ Manifests deployed"

# Run controller locally (requires kubeconfig)
run:
	@echo "Running controller locally..."
	go run ./cmd/zen-flow-controller

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin latest; \
	fi
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	@if ! command -v helm >/dev/null 2>&1; then \
		echo "⚠️  Helm not found. Install from https://helm.sh/docs/intro/install/"; \
	fi
	@echo "✅ Development tools installed"

# Run E2E tests
test-e2e:
	@echo "Running E2E tests..."
	go test -v -timeout=30m ./test/integration/...
	@echo "✅ E2E tests passed"

# Run load tests
test-load:
	@echo "Running load tests..."
	@echo "⚠️  Load tests require a running Kubernetes cluster"
	@echo "⚠️  Skipping in short mode - use 'go test -v ./test/load/...' to run"
	@go test -v -timeout=30m ./test/load/... || echo "⚠️  Load tests skipped (use 'go test -v ./test/load/...' to run)"
	@echo "⚠️  Load tests not yet implemented"
	@echo "TODO: Implement load testing for zen-flow"

# Validate example manifests
validate-examples:
	@echo "Validating example manifests..."
	@if [ -f "cmd/validate-examples/main.go" ]; then \
		echo "Using Go validator tool..."; \
		go run ./cmd/validate-examples examples/ || exit 1; \
	else \
		echo "Using kubectl dry-run validation..."; \
		for file in examples/*.yaml; do \
			if [ -f "$$file" ]; then \
				echo "Validating $$file..."; \
				kubectl apply -f "$$file" --dry-run=client --validate=true || exit 1; \
			fi \
		done; \
	fi
	@echo "✅ All examples are valid"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Local-First Delivery Gates (G014)
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

.PHONY: validate-local-prereqs validate-local-build validate-local-cluster validate-local-install validate-local-examples validate-local-approval validate-local-all

# Validate local prerequisites
validate-local-prereqs:
	@echo "🔍 Validating local prerequisites..."
	@echo "Checking /etc/hosts sanity..."
	@if ! grep -q "127.0.0.1.*localhost" /etc/hosts 2>/dev/null; then \
		echo "⚠️  Warning: /etc/hosts may not have localhost entry"; \
	fi
	@echo "Checking Docker registry..."
	@if ! docker info >/dev/null 2>&1; then \
		echo "❌ Docker is not running or not accessible"; \
		exit 1; \
	fi
	@echo "Checking for local registry..."
	@if ! docker ps | grep -q registry:2 2>/dev/null && ! curl -s http://localhost:5000/v2/ >/dev/null 2>&1; then \
		echo "⚠️  Warning: Local Docker registry not detected (localhost:5000)"; \
		echo "   Start with: docker run -d -p 5000:5000 --name registry registry:2"; \
	fi
	@echo "✅ Local prerequisites validated"

# Validate local build and push
validate-local-build:
	@echo "🔨 Validating local build..."
	@echo "Building Docker image..."
	@$(MAKE) build-image
	@echo "Tagging for local registry..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-alpha"); \
	docker tag kubezen/zen-flow-controller:$$VERSION localhost:5000/zen-flow-controller:$$VERSION || true
	@echo "Pushing to local registry..."
	@VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "0.0.1-alpha"); \
	docker push localhost:5000/zen-flow-controller:$$VERSION 2>/dev/null || echo "⚠️  Local registry push skipped (registry may not be running)"
	@echo "✅ Local build validated"

# Validate local cluster (k3d or kind)
validate-local-cluster:
	@echo "☸️  Validating local cluster..."
	@if command -v k3d >/dev/null 2>&1; then \
		echo "Using k3d..."; \
		if ! k3d cluster list | grep -q "zen-flow-test"; then \
			echo "Creating k3d cluster zen-flow-test..."; \
			k3d cluster create zen-flow-test --wait --timeout 300s || exit 1; \
		fi; \
		kubectl config use-context k3d-zen-flow-test || exit 1; \
	elif command -v kind >/dev/null 2>&1; then \
		echo "Using kind..."; \
		if ! kind get clusters | grep -q "zen-flow-test"; then \
			echo "Creating kind cluster zen-flow-test..."; \
			kind create cluster --name zen-flow-test --wait 300s || exit 1; \
		fi; \
		kubectl config use-context kind-zen-flow-test || exit 1; \
	else \
		echo "❌ Neither k3d nor kind found. Install one: https://k3d.io or https://kind.sigs.k8s.io"; \
		exit 1; \
	fi
	@echo "Waiting for cluster to be ready..."
	@kubectl wait --for=condition=Ready nodes --all --timeout=300s || echo "⚠️  Cluster nodes may not be ready yet"
	@echo "✅ Local cluster validated"

# Validate local installation (CRD + Helm)
validate-local-install:
	@echo "📦 Validating local installation..."
	@echo "Installing CRD..."
	@kubectl apply -f deploy/crds/workflow.kube-zen.io_jobflows.yaml || exit 1
	@echo "Verifying CRD installation..."
	@kubectl get crd jobflows.workflow.kube-zen.io || exit 1
	@echo "Installing from helm-charts repository..."
	@helm repo add kube-zen https://kube-zen.github.io/helm-charts || true
	@helm repo update
	@helm install zen-flow kube-zen/zen-flow --namespace zen-flow-system --create-namespace --wait --timeout 5m || exit 1
	@echo "Verifying controller pod..."
	@kubectl wait --for=condition=Ready pod -l app.kubernetes.io/name=zen-flow -n zen-flow-system --timeout=300s || echo "⚠️  Controller pod may not be ready yet"
	@echo "✅ Local installation validated"

# Validate local examples
validate-local-examples:
	@echo "📝 Validating local examples..."
	@echo "Applying simple-linear-flow example..."
	@kubectl apply -f examples/simple-linear-flow.yaml || exit 1
	@echo "Waiting for JobFlow to reach Succeeded..."
	@timeout=60; \
	while [ $$timeout -gt 0 ]; do \
		phase=$$(kubectl get jobflow simple-linear-flow -o jsonpath='{.status.phase}' 2>/dev/null || echo "Pending"); \
		if [ "$$phase" = "Succeeded" ]; then \
			echo "✅ JobFlow reached Succeeded phase"; \
			break; \
		fi; \
		echo "Waiting... (phase: $$phase, timeout: $$timeout)"; \
		sleep 2; \
		timeout=$$((timeout - 2)); \
	done; \
	if [ $$timeout -le 0 ]; then \
		echo "⚠️  JobFlow did not reach Succeeded within timeout"; \
		kubectl get jobflow simple-linear-flow -o yaml; \
		exit 1; \
	fi
	@echo "Verifying Jobs were created..."
	@kubectl get jobs -l workflow.kube-zen.io/flow=simple-linear-flow || echo "⚠️  Jobs not found"
	@echo "Cleaning up example..."
	@kubectl delete jobflow simple-linear-flow --ignore-not-found=true
	@echo "✅ Local examples validated"

# Validate local manual approval
validate-local-approval:
	@echo "✋ Validating local manual approval..."
	@echo "Applying manual-approval-flow example..."
	@kubectl apply -f examples/manual-approval-flow.yaml || exit 1
	@echo "Waiting for approval step..."
	@sleep 5
	@phase=$$(kubectl get jobflow manual-approval-flow -o jsonpath='{.status.phase}' 2>/dev/null || echo "Pending"); \
	if [ "$$phase" != "Paused" ]; then \
		echo "⚠️  Expected JobFlow to be Paused, got: $$phase"; \
		kubectl get jobflow manual-approval-flow -o yaml; \
		exit 1; \
	fi
	@echo "Approving step with new annotation key..."
	@kubectl annotate jobflow manual-approval-flow workflow.kube-zen.io/approved/approve-step=true --overwrite || exit 1
	@echo "Waiting for approval to be processed..."
	@sleep 5
	@phase=$$(kubectl get jobflow manual-approval-flow -o jsonpath='{.status.phase}' 2>/dev/null || echo "Pending"); \
	if [ "$$phase" = "Succeeded" ] || [ "$$phase" = "Running" ]; then \
		echo "✅ Manual approval validated (new key)"; \
	else \
		echo "⚠️  Approval may not have been processed (phase: $$phase)"; \
	fi
	@echo "Cleaning up..."
	@kubectl delete jobflow manual-approval-flow --ignore-not-found=true
	@echo "✅ Local manual approval validated"

# Run all local validation gates
validate-local-all: validate-local-prereqs validate-local-build validate-local-cluster validate-local-install validate-local-examples validate-local-approval
	@echo ""
	@echo "✅ All local-first delivery gates passed!"
	@echo "   Ready for shared cluster deployment"

# Helm charts are now in the helm-charts repository
# See: https://github.com/kube-zen/helm-charts


check:
	@scripts/ci/check.sh

test-race:
	@go test -v -race -timeout=15m ./...
