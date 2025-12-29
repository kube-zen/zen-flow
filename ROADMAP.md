# zen-flow Roadmap

This roadmap outlines future enhancements and improvements for zen-flow.

## 🎯 Current Status

**Version**: 0.0.1-alpha

**Status**: ✅ All initial roadmap phases complete!

All planned phases (Critical Infrastructure, Validation & Quality, Deployment & Operations, Documentation & Governance) have been successfully completed. The project is production-ready.


## 🚀 Future Enhancements

### Testing & Quality
- ✅ **Completed**: Added comprehensive unit tests for error handling (`isRetryable`, `applyPodFailureAction`, `evaluateWhenCondition`)
- ✅ **Completed**: Added tests for execution planning (`createExecutionPlan` with DAG, ContinueOnFailure, When conditions)
- ✅ **Completed**: Added tests for status refresh logic (`refreshStepStatusFromJob`, `refreshStepStatuses`)
- ✅ **Completed**: Test coverage improved from 64.4% to 68.8%
- Enhance E2E test coverage with new features (TTL, retries, timeouts, concurrency, pod failure policies, when conditions, manual approval)
- Test full lifecycle: create JobFlow → reconcile → Job created → Job succeeds → step status updated → next steps start → flow completes
- Add tests for edge cases (job deletion, job failure, etc.)
- Target: Achieve 75%+ test coverage

### Feature Enhancements
- **In Progress**: Artifact/parameter handling structure in place (see [Limitations](README.md#limitations))
- **In Progress**: When condition evaluation with basic support (see [Limitations](README.md#limitations))
- Enhance artifact/parameter handling with actual storage/transfer (S3, GCS, HTTP endpoints)
- Enhance when condition evaluation with full template engine (step status evaluation, expressions)
- Performance optimizations for large-scale deployments
- Multi-cluster support
- Advanced workflow features (suspension/resumption, parameter substitution)
- Additional observability integrations (OpenTelemetry tracing)
- Community-driven feature requests

## 📝 Notes

- This roadmap tracked the initial production-readiness milestones
- All core functionality is complete and production-ready
- Future enhancements will be tracked via GitHub Issues and community feedback

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to contribute to zen-flow.

