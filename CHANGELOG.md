# Changelog

All notable changes to zen-flow will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **Helm chart defaults**: Webhooks disabled by default for safe installation without prerequisites
- **Helm RBAC**: Added delete permissions for jobflows and jobs to match controller behavior
- **Webhook safety**: Default failurePolicy set to Ignore to prevent bricking installations
- **Helm CRD installation**: CRDs now install automatically via Helm chart
- **Webhook TLS setup**: Consolidated to single approach with cert-manager support
- **Metrics gauge drift**: Added recompute methods for accurate metrics

### Changed
- **Installation**: Helm is now the recommended installation method
- **Quick Start**: Complete rewrite with Helm-first approach, troubleshooting, and uninstall instructions
- **Webhook configuration**: Production-ready TLS setup with cert-manager annotations

### Added
- Comprehensive controller test suite (85%+ coverage)
- E2E test suite for JobFlow lifecycle and DAG execution
- Helm chart for easy deployment (`charts/zen-flow/`)
- VPA configuration for resource recommendations
- GitHub Actions CI/CD workflows:
  - Lint job (golangci-lint, go vet, yamllint, Helm lint)
  - Test job with coverage reporting (75%+ threshold)
  - Build job
  - Security scanning (govulncheck, gosec, Trivy)
  - Multi-arch Docker image builds (linux/amd64, linux/arm64)
  - Helm chart publishing workflow
- Dependabot configuration for automated dependency updates
- CODEOWNERS file for code ownership
- Pull request template
- Issue templates (bug report, feature request)
- Additional Makefile targets:
  - `test-e2e`: Run E2E tests
  - `test-load`: Run load tests
  - `validate-examples`: Validate example YAML manifests
  - `helm-lint`, `helm-package`, `helm-test`, `helm-install`, `helm-all`
- VPA documentation (`docs/VPA.md`)

### Changed
- Improved controller test coverage from 45.6% to 85%+
- Updated documentation to include CI/CD information
- Updated roadmap to reflect Phase 3 completion

## [0.0.1-alpha] - 2015-12-29

### Added
- Initial release of zen-flow
- JobFlow CRD (workflow.kube-zen.io/v1alpha1)
- Basic sequential job execution
- DAG support with topological sorting
- Kubernetes Job creation and management
- Status reporting and progress tracking
- Prometheus metrics integration
- Kubernetes events support
- Leader election for HA
- RBAC manifests
- Deployment manifests
- Example JobFlow manifests
- Comprehensive documentation

### Architecture
- Kubernetes-native controller pattern
- CRD-based API design
- Zero external dependencies
- Production-ready security defaults (Pod Security Standards, RBAC)
- Observability built-in (Prometheus metrics, Kubernetes events)

### Known Limitations
- Status updates require dynamic client implementation (placeholder)
- JobFlow informer setup needs dynamic client configuration
- Retry policies not yet fully implemented
- Artifact management (PVC, S3) not yet implemented
- Parameter substitution not yet implemented
- Validating webhooks not yet implemented

### Next Steps
- Complete dynamic client setup for JobFlow CRD
- Implement full status update mechanism
- Add unit and integration tests
- Implement retry policies
- Add artifact management
- Add parameter substitution
- Add validating webhooks

