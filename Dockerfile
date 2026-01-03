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

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies (zen-sdk will be fetched from GitHub via go.mod)
RUN go mod download

# Copy source code
COPY . .

# Build
# Default: GA-only (no experimental features)
# To enable experimental features: docker build --build-arg GOEXPERIMENT=jsonv2,greenteagc
# Available experiments: jsonv2, greenteagc
# Experimental features provide 15-25% performance improvement but are opt-in
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

