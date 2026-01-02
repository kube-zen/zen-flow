# Outstanding Items

**Last Updated**: 2026-01-02

## Summary

Most major optimization opportunities have been completed. The following items remain:

---

## 🔴 High Priority

### 1. **Full JSONPath Implementation**
**Status**: ⚠️ **Partial** - Basic implementation exists, full library needed

**Current State**:
- Basic JSONPath evaluation exists in `parameters.go`
- Only supports simple patterns like `$.succeeded`
- Full JSONPath library integration needed

**Location**: `pkg/controller/parameters.go:140, 181`

**Recommendation**:
- Add `github.com/PaesslerAG/jsonpath` or similar library
- Implement full JSONPath evaluation for parameter extraction
- Support complex expressions like `$.status.conditions[?(@.type=='Ready')].status`

**Impact**: **MEDIUM** - Enables full parameter extraction from job outputs

---

### 2. **Parameter Template Application**
**Status**: ⚠️ **Partial** - Parameters resolved but not applied to job templates

**Current State**:
- Parameters are resolved from ConfigMap/Secret/JSONPath
- Resolved values are logged but not applied to job templates
- Job templates need parameter substitution

**Location**: `pkg/controller/reconciler.go:1482`

**Recommendation**:
- Use template engine to substitute parameters in job templates
- Apply resolved parameters before creating Jobs
- Support parameter references in job spec (e.g., `{{.parameters.paramName}}`)

**Impact**: **HIGH** - Enables dynamic job configuration

---

### 3. **Artifact Archiving**
**Status**: ⚠️ **Partial** - Structure exists, implementation needed

**Current State**:
- Artifact archiving structure exists
- Archive configuration supported (format, compression)
- Actual archiving not implemented

**Location**: `pkg/controller/reconciler.go:1503`

**Recommendation**:
- Implement tar/zip archiving based on `ArchiveConfig`
- Support compression (gzip, none)
- Archive artifacts before S3 upload or storage

**Impact**: **MEDIUM** - Enables artifact compression and archiving

---

## 🟡 Medium Priority

### 4. **S3 Upload Implementation**
**Status**: ⚠️ **Partial** - Structure exists, requires S3 client library

**Current State**:
- S3 upload structure exists in `artifacts.go`
- Credential retrieval implemented
- Actual S3 upload not implemented (requires library)

**Location**: `pkg/controller/artifacts.go:165`

**Recommendation**:
- Add `github.com/aws/aws-sdk-go-v2/service/s3` or `github.com/minio/minio-go`
- Implement S3 upload with retry logic
- Support S3-compatible storage (MinIO, etc.)

**Impact**: **MEDIUM** - Enables artifact storage in S3

---

### 5. **Artifact Copying from Shared Storage**
**Status**: ⚠️ **Partial** - Structure exists, implementation needed

**Current State**:
- Artifact fetching structure exists
- Supports fetching from previous steps
- Actual copying from shared storage not implemented

**Location**: `pkg/controller/artifacts.go:84`

**Recommendation**:
- Implement artifact copying from PVC or ConfigMap
- Support different artifact types (files, directories, archives)
- Handle artifact mounting in job containers

**Impact**: **MEDIUM** - Enables artifact passing between steps

---

### 6. **OpenAPI/Swagger Documentation**
**Status**: ⚠️ **Missing** - API documentation not generated

**Current State**:
- CRD definitions exist
- API reference documentation exists (markdown)
- No OpenAPI/Swagger spec generated

**Recommendation**:
- Generate OpenAPI spec from CRD definitions
- Add Swagger UI for API exploration
- Integrate with kubebuilder for automatic generation

**Impact**: **LOW** - Improves API discoverability

---

## 🟢 Low Priority / Future Enhancements

### 7. **Performance Optimizations (Already Done)**
**Status**: ✅ **Complete** - But document needs updating

**Note**: The following optimizations are already implemented but the optimization document hasn't been updated:
- ✅ Status update batching (implemented)
- ✅ DAG caching (implemented)
- ✅ Parallel step status refresh (implemented)
- ✅ Metrics coverage (all metrics added)

**Action**: Update `ZEN_FLOW_OPTIMIZATION_OPPORTUNITIES.md` to reflect completed work.

---

### 8. **Test Coverage**
**Status**: ✅ **Good** - 77.8% overall coverage

**Current Coverage**:
- `pkg/controller/dag`: 100%
- `pkg/controller/metrics`: 100%
- `pkg/controller`: ~77.8% (improved from 17.1%)

**Recommendation**:
- Target 80%+ coverage
- Add edge case tests
- Add integration tests for complex scenarios

**Impact**: **LOW** - Already at good coverage level

---

## 📋 Implementation Priority

1. **Parameter Template Application** (High) - Enables dynamic job configuration
2. **Full JSONPath Implementation** (High) - Enables parameter extraction
3. **Artifact Archiving** (Medium) - Enables artifact compression
4. **S3 Upload Implementation** (Medium) - Requires external library
5. **Artifact Copying** (Medium) - Enables artifact passing
6. **OpenAPI Documentation** (Low) - Nice to have

---

## ✅ Completed Items (For Reference)

- ✅ Load Testing Implementation
- ✅ Template Engine Support
- ✅ Artifact Fetching Structure
- ✅ Parameter Resolution (ConfigMap/Secret)
- ✅ Resource Requirements Documentation
- ✅ Performance Tuning Guide
- ✅ Webhook Certificate Monitoring
- ✅ RBAC Least Privilege Audit
- ✅ Status Update Batching
- ✅ DAG Caching
- ✅ Parallel Step Status Refresh
- ✅ Metrics Coverage

---

## Notes

- Most critical infrastructure is complete
- Remaining items are feature enhancements
- All items marked as "Partial" have structure in place, need implementation
- External libraries needed for: JSONPath (optional), S3 (optional)

