# Build stage
FROM golang:1.24-alpine AS builder

ARG VERSION=0.0.1-alpha
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git make

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w \
        -X 'main.version=${VERSION}' \
        -X 'main.commit=${COMMIT}' \
        -X 'main.buildDate=${BUILD_DATE}'" \
    -o zen-flow-controller ./cmd/zen-flow-controller

# Runtime stage
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/zen-flow-controller /app/zen-flow-controller

# Create non-root user
RUN addgroup -g 65534 -S nonroot && \
    adduser -u 65534 -S nonroot -G nonroot && \
    chown -R nonroot:nonroot /app

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/app/zen-flow-controller"]

