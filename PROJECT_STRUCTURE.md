# zen-flow Project Structure

## Overview

zen-flow is a Kubernetes-native job orchestration controller that provides declarative, sequential execution of Kubernetes Jobs using standard CRDs.

## Project Goals

1. **Provide Value**: Solve the real-world problem of job sequencing in Kubernetes
2. **Production Ready**: Build a reliable, observable, and maintainable solution
3. **Kubernetes-Native**: Follow Kubernetes best practices and conventions
4. **Zero Dependencies**: No external databases or message queues

## Project Structure

```
zen-flow/
├── cmd/
│   └── zen-flow-controller/          # Main controller binary
│       └── main.go                   # Application entrypoint
├── pkg/
│   ├── api/
│   │   └── v1alpha1/                 # JobFlow CRD types
│   │       ├── groupversion_info.go  # API group/version info
│   │       ├── register.go           # Registration helpers
│   │       ├── types.go              # JobFlow type definitions
│   │       └── zz_generated.deepcopy.go  # Deep copy methods
│   └── controller/                   # Controller implementation
│       ├── dag/                      # DAG execution engine
│       │   └── dag.go               # DAG graph and topological sort
│       ├── metrics/                  # Prometheus metrics
│       │   └── metrics.go           # Metrics definitions
│       ├── events.go                 # Kubernetes event recorder
│       ├── jobflow_controller.go     # Main controller logic
│       └── leader_election.go       # Leader election for HA
├── deploy/
│   ├── crds/                        # CRD definitions
│   │   └── workflow.kube-zen.io_jobflows.yaml
│   └── manifests/                   # Deployment manifests
│       ├── namespace.yaml           # Namespace
│       ├── rbac.yaml                # RBAC (ServiceAccount, ClusterRole, ClusterRoleBinding)
│       ├── deployment.yaml          # Controller Deployment
│       └── service.yaml             # Metrics Service
├── examples/                         # Example JobFlow manifests
│   ├── simple-linear-flow.yaml     # Simple sequential flow
│   └── dag-flow.yaml                # DAG with parallel execution
├── build/                           # Build files (if needed)
├── docs/                            # Documentation (if needed)
├── .dockerignore                    # Docker ignore rules
├── .gitignore                       # Git ignore rules
├── Dockerfile                       # Multi-stage Dockerfile
├── go.mod                           # Go module definition
├── go.sum                           # Go module checksums
├── LICENSE                          # Apache 2.0 License
├── Makefile                         # Build and test targets
├── NOTICE                           # Copyright notice
├── PROJECT_STRUCTURE.md             # This file
└── README.md                        # Project documentation
```

## Directory Purposes

### `/cmd`
**Purpose**: Main application entry points

- Keep minimal - just wiring
- Each subdirectory = one binary
- Logic lives in `pkg/`

### `/pkg`
**Purpose**: All reusable code

- Well-organized packages
- Business logic
- Can be imported by other projects
- Includes CRD types in `pkg/api/v1alpha1/`

### `/pkg/api/v1alpha1`
**Purpose**: CRD type definitions

- JobFlow CRD types
- Type constants
- Deep copy methods
- API group/version registration

### `/pkg/controller`
**Purpose**: Controller implementation

- Main reconciliation logic
- DAG execution engine
- Metrics and events
- Leader election

### `/deploy`
**Purpose**: Kubernetes manifests

- CRD definitions
- Deployment manifests
- RBAC configuration
- Direct `kubectl apply` usage

### `/examples`
**Purpose**: Working examples

- Example JobFlow manifests
- Tutorial configs
- Integration examples

## Development Status

### Current Status: MVP (v1alpha1) ✅

- ✅ Basic sequential execution (linear flows)
- ✅ Kubernetes Job creation/management
- ✅ Status reporting
- ✅ Basic metrics and events
- ✅ DAG support (topological sort)
- ✅ Project structure and build system

### Next Steps

- ⬜ Dynamic client setup for JobFlow CRD
- ⬜ Full status update implementation
- ⬜ Unit test coverage
- ⬜ Integration tests
- ⬜ Validating webhooks
- ⬜ Artifact management
- ⬜ Retry policies implementation

## Design Philosophy

**Simple and Focused**: zen-flow is a job orchestrator, not a complex workflow engine.

### What We DON'T Need (for MVP)
- ❌ External databases
- ❌ Complex UI components
- ❌ Multiple API versions (yet)
- ❌ Webhook servers (yet)

### What We DO Have
- ✅ `pkg/api/v1alpha1/` - CRD definitions where they belong
- ✅ `deploy/crds/` - CRD manifests with other K8s resources
- ✅ Clean, simple structure
- ✅ Easy to understand and contribute to
- ✅ Kubernetes-native patterns

## Version

Current version: **0.0.1-alpha**

Even though this is production-grade architecture, we maintain the alpha version to indicate this is early-stage software that may have breaking changes.

