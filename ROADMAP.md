# zen-flow Roadmap

This roadmap outlines the path to production-grade quality matching zen-gc standards.

## 🎯 Current Status

**Version**: 0.0.1-alpha

**Completed**:
- ✅ Core controller implementation
- ✅ CRD definitions
- ✅ DAG engine (100% test coverage)
- ✅ Structured logging
- ✅ Error handling
- ✅ Metrics infrastructure
- ✅ Basic test suite (45.6% controller coverage)
- ✅ Deployment manifests
- ✅ Basic documentation

## 📅 Roadmap Phases

### Phase 1: Critical Infrastructure (Weeks 1-2) 🔴

**Goal**: Production-ready webhooks and observability

#### 1.1 Webhook Infrastructure
- [ ] Implement validating webhook server (`pkg/webhook/webhook.go`)
- [ ] Implement mutating webhook server (defaults)
- [ ] Add webhook tests (`pkg/webhook/webhook_test.go`)
- [ ] Create webhook manifests (`deploy/webhook/`)
- [ ] Integrate cert-manager for TLS certificates
- [ ] Add webhook health check endpoint

**Deliverables**:
- Validating webhook for JobFlow CRD
- Mutating webhook for JobFlow defaults
- Certificate management
- Webhook tests (75%+ coverage)

**Priority**: 🔴 Critical

#### 1.2 Prometheus Rules & Alerts
- [ ] Create PrometheusRule resource
- [ ] Define alerts for:
  - Controller down
  - High reconciliation error rate
  - Step execution failures
  - JobFlow stuck states
  - Slow step execution
  - High job creation failure rate

**Deliverables**:
- `deploy/prometheus/prometheus-rules.yaml`
- Comprehensive alerting rules

**Priority**: 🔴 Critical

#### 1.3 Grafana Dashboard
- [ ] Create Grafana dashboard JSON
- [ ] Add panels for:
  - JobFlows by phase
  - Step execution metrics
  - Error rates
  - Duration histograms
  - Top JobFlows
- [ ] Add dashboard README

**Deliverables**:
- `deploy/grafana/dashboard.json`
- `deploy/grafana/README.md`

**Priority**: 🔴 Critical

### Phase 2: Validation & Quality (Weeks 3-4) 🟡

**Goal**: Comprehensive validation and improved test coverage

#### 2.1 Validation Package
- [ ] Create validation package (`pkg/validation/`)
- [ ] Implement JobFlow validator
- [ ] Add DAG validation (cycle detection)
- [ ] Add template validation (Job spec)
- [ ] Add resource template validation
- [ ] Add execution policy validation
- [ ] Write comprehensive tests

**Deliverables**:
- `pkg/validation/validator.go`
- `pkg/validation/validator_test.go`
- `pkg/validation/dag_validator.go`
- `pkg/validation/template_validator.go`
- 75%+ test coverage

**Priority**: 🟡 High

#### 2.2 Test Coverage Improvement
- [ ] Increase controller test coverage to 75%+
- [ ] Add integration tests for:
  - JobFlow lifecycle
  - Step execution
  - DAG execution
  - Error recovery
- [ ] Add E2E test suite
- [ ] Add load tests

**Deliverables**:
- Controller coverage: 75%+
- Integration test suite
- E2E test suite (`test/e2e/`)
- Load test scripts (`test/load/`)

**Priority**: 🟡 High

### Phase 3: Deployment & Operations (Weeks 5-6) 🟡

**Goal**: Production-ready deployment options

#### 3.1 Helm Chart
- [ ] Create Helm chart structure
- [ ] Add Chart.yaml
- [ ] Create values.yaml with configurable options
- [ ] Create templates for all resources
- [ ] Add chart README
- [ ] Add Helm linting to CI

**Deliverables**:
- `charts/zen-flow/Chart.yaml`
- `charts/zen-flow/values.yaml`
- `charts/zen-flow/templates/*.yaml`
- `charts/zen-flow/README.md`

**Priority**: 🟡 Medium

#### 3.2 VPA Configuration
- [ ] Create VPA manifest
- [ ] Configure resource recommendations
- [ ] Document VPA usage

**Deliverables**:
- `deploy/manifests/vpa.yaml`
- VPA documentation

**Priority**: 🟡 Medium

#### 3.3 Additional Makefile Targets
- [ ] Add `build-release` target
- [ ] Add `build-image-multiarch` target
- [ ] Add `test-e2e` target
- [ ] Add `test-load` target
- [ ] Add `validate-examples` target
- [ ] Add `helm-lint` target
- [ ] Add `helm-package` target

**Deliverables**:
- Updated `Makefile` with all targets

**Priority**: 🟡 Medium

### Phase 4: Documentation & Governance (Weeks 7-8) 🟢

**Goal**: Complete documentation and community standards

#### 4.1 Technical Documentation
- [ ] API_REFERENCE.md - Complete API documentation
- [ ] ARCHITECTURE.md - Architecture deep dive
- [ ] USER_GUIDE.md - User-facing guide
- [ ] OPERATOR_GUIDE.md - Operations guide
- [ ] METRICS.md - Metrics documentation
- [ ] SECURITY.md - Security best practices
- [ ] RBAC.md - RBAC documentation
- [ ] CI_CD.md - CI/CD setup guide
- [ ] DISASTER_RECOVERY.md - DR procedures
- [ ] BENCHMARKS.md - Performance benchmarks
- [ ] VERSION_COMPATIBILITY.md - K8s compatibility

**Deliverables**:
- Complete `docs/` directory
- All technical documentation

**Priority**: 🟢 Low

#### 4.2 Governance & Community
- [ ] CODE_OF_CONDUCT.md
- [ ] CONTRIBUTING.md
- [ ] GOVERNANCE.md
- [ ] MAINTAINERS.md
- [ ] ADOPTERS.md
- [ ] RELEASING.md
- [ ] SECURITY.md (security policy)

**Deliverables**:
- Complete governance structure

**Priority**: 🟢 Low

#### 4.3 Example Validation Tool
- [ ] Create `cmd/validate-examples/`
- [ ] Implement example validator
- [ ] Add Makefile target

**Deliverables**:
- `cmd/validate-examples/main.go`
- Example validation tool

**Priority**: 🟢 Low

## 📊 Progress Tracking

### Phase 1: Critical Infrastructure
- [ ] Webhook Infrastructure (0/6 tasks)
- [ ] Prometheus Rules (0/1 tasks)
- [ ] Grafana Dashboard (0/2 tasks)
- **Progress**: 0/9 tasks (0%)

### Phase 2: Validation & Quality
- [ ] Validation Package (0/7 tasks)
- [ ] Test Coverage (0/4 tasks)
- **Progress**: 0/11 tasks (0%)

### Phase 3: Deployment & Operations
- [ ] Helm Chart (0/6 tasks)
- [ ] VPA Configuration (0/2 tasks)
- [ ] Makefile Targets (0/7 tasks)
- **Progress**: 0/15 tasks (0%)

### Phase 4: Documentation & Governance
- [ ] Technical Documentation (0/11 tasks)
- [ ] Governance (0/7 tasks)
- [ ] Example Validator (0/2 tasks)
- **Progress**: 0/20 tasks (0%)

**Overall Progress**: 0/55 tasks (0%)

## 🎯 Success Criteria

### Phase 1 Complete When:
- ✅ Webhooks deployed and tested
- ✅ Prometheus alerts configured
- ✅ Grafana dashboard operational
- ✅ All critical infrastructure in place

### Phase 2 Complete When:
- ✅ Validation package implemented (75%+ coverage)
- ✅ Controller test coverage: 75%+
- ✅ Integration tests passing
- ✅ E2E tests passing

### Phase 3 Complete When:
- ✅ Helm chart published
- ✅ VPA configured
- ✅ All Makefile targets working
- ✅ Multi-arch builds working

### Phase 4 Complete When:
- ✅ All documentation complete
- ✅ Governance structure in place
- ✅ Example validator working
- ✅ Ready for community contribution

## 🚀 Version Milestones

### v0.0.1-alpha (Current)
- Core functionality
- Basic tests
- Basic documentation

### v0.0.2-alpha (After Phase 1)
- Webhooks
- Prometheus alerts
- Grafana dashboard

### v0.0.3-alpha (After Phase 2)
- Comprehensive validation
- 75%+ test coverage
- E2E tests

### v0.1.0-beta (After Phase 3)
- Helm chart
- Production-ready deployment
- Multi-arch support

### v1.0.0 (After Phase 4)
- Complete documentation
- Community governance
- Production stable

## 📝 Notes

- This roadmap is flexible and may be adjusted based on feedback
- Priority levels indicate importance, not strict ordering
- Some tasks may be done in parallel
- Community feedback will influence prioritization

## 🤝 Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for how to contribute to zen-flow.

