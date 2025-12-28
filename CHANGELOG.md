# Changelog

All notable changes to zen-flow will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.0.1-alpha] - 2025-01-15

### Added
- Initial release of zen-flow
- JobFlow CRD (workflow.zen.io/v1alpha1)
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

