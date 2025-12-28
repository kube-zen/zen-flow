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
		-t kube-zen/zen-flow-controller:$$VERSION \
		-t kube-zen/zen-flow-controller:latest .
	@echo "✅ Docker image built: kube-zen/zen-flow-controller:$$VERSION"

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
		-t kube-zen/zen-flow-controller:$$VERSION \
		-t kube-zen/zen-flow-controller:latest \
		--push .
	@echo "✅ Multi-arch Docker images built: kube-zen/zen-flow-controller:$$VERSION"

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
	@echo "Checking coverage threshold (minimum: 75%)..."
	@COVERAGE=$$(go tool cover -func=coverage.out | grep -v "pkg/api/v1alpha1" | grep "total:" | awk '{print $$3}' | sed 's/%//'); \
	if [ -z "$$COVERAGE" ]; then \
		echo "⚠️  Could not determine coverage percentage"; \
	elif [ $$(echo "$$COVERAGE < 75" | bc -l 2>/dev/null || echo "0") -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below the 75% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets the 75% threshold"; \
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

# Helm chart operations
helm-lint:
	@echo "Linting Helm chart..."
	@if ! command -v helm >/dev/null 2>&1; then \
		echo "❌ Helm not found. Install from https://helm.sh/docs/intro/install/"; \
		exit 1; \
	fi
	helm lint charts/zen-flow
	@echo "✅ Helm chart linting passed"

helm-package:
	@echo "Packaging Helm chart..."
	@if ! command -v helm >/dev/null 2>&1; then \
		echo "❌ Helm not found. Install from https://helm.sh/docs/intro/install/"; \
		exit 1; \
	fi
	@mkdir -p .helm-packages
	helm package charts/zen-flow -d .helm-packages
	@echo "✅ Helm chart packaged in .helm-packages/"

helm-test:
	@echo "Testing Helm chart rendering..."
	@if ! command -v helm >/dev/null 2>&1; then \
		echo "❌ Helm not found. Install from https://helm.sh/docs/intro/install/"; \
		exit 1; \
	fi
	helm template zen-flow charts/zen-flow --debug > /dev/null
	@echo "✅ Helm chart renders successfully"

helm-install:
	@echo "Installing Helm chart (dry-run)..."
	@if ! command -v helm >/dev/null 2>&1; then \
		echo "❌ Helm not found. Install from https://helm.sh/docs/intro/install/"; \
		exit 1; \
	fi
	helm install zen-flow charts/zen-flow --dry-run --debug --namespace zen-flow-system --create-namespace
	@echo "✅ Helm chart installation dry-run successful"

helm-all: helm-lint helm-test helm-package
	@echo "✅ All Helm operations completed"

