# Build stage
FROM golang:1.25-alpine AS builder

ARG VERSION=0.0.1-alpha
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
# GOEXPERIMENT: Enable experimental Go 1.25 features (jsonv2, greenteagc)
# Default: "" (GA-only, no experimental features)
# To enable: docker build --build-arg GOEXPERIMENT=jsonv2,greenteagc
ARG GOEXPERIMENT=""

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy zen-sdk first (needed for latest logging code)
# Build context should be from parent directory (zen/)
COPY zen-sdk /build/zen-sdk

# Ensure zen-sdk dependencies are resolved
WORKDIR /build/zen-sdk
RUN go mod tidy && go mod download

# Back to build directory
WORKDIR /build

# Copy go mod files
COPY zen-flow/go.mod zen-flow/go.sum* ./

# Download dependencies (may fail for zen-sdk if tag not available, that's OK)
RUN go mod download || true

# Add replace directive to use local zen-sdk during build
RUN go mod edit -replace github.com/kube-zen/zen-sdk=./zen-sdk

# Download dependencies with local replace (updates go.sum without removing requires)
RUN go mod download

# Copy source code
COPY zen-flow/ .

# Build
# Default: GA-only (no experimental features)
# To enable experimental features: docker build --build-arg GOEXPERIMENT=jsonv2,greenteagc
# Available experiments: jsonv2, greenteagc
# Experimental features provide 15-25% performance improvement but are opt-in
ARG GOEXPERIMENT=""
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    GOEXPERIMENT=${GOEXPERIMENT} \
    go build \
    -trimpath \
    -ldflags "-s -w \
        -X 'main.version=${VERSION}' \
        -X 'main.commit=${COMMIT}' \
        -X 'main.buildDate=${BUILD_DATE}'" \
    -o zen-flow-controller ./cmd/zen-flow-controller

# Runtime stage - use scratch (empty) base for minimal size
# The binary is statically linked (CGO_ENABLED=0), so no libc needed
FROM scratch

# Copy CA certificates from Alpine for HTTPS/TLS support (needed for Kubernetes API)
# This is much smaller than the full Alpine base (~200KB vs 8MB)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary
COPY --from=builder /build/zen-flow-controller /zen-flow-controller

EXPOSE 8080

ENTRYPOINT ["/zen-flow-controller"]

