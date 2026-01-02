# OpenAPI/Swagger Documentation

This directory contains the OpenAPI 3.0 specification for the zen-flow CRD API.

## Files

- `openapi.yaml` - OpenAPI 3.0 specification in YAML format
- `openapi.json` - OpenAPI 3.0 specification in JSON format

## Generation

The OpenAPI specification is generated from the CRD definition using:

```bash
make generate-openapi
```

Or directly:

```bash
./scripts/generate-openapi.sh
```

The script extracts the OpenAPI schema from `deploy/crds/workflow.kube-zen.io_jobflows.yaml` and creates a full OpenAPI 3.0 specification with:

- API paths for CRUD operations
- Request/response schemas
- Component schemas extracted from the CRD
- Standard Kubernetes metadata types

## Viewing the Documentation

### Option 1: Swagger UI (Local)

Install and run Swagger UI:

```bash
npm install -g swagger-ui-serve
make serve-swagger
```

Or using Docker:

```bash
docker run -p 8080:8080 \
  -e SWAGGER_JSON=/openapi.yaml \
  -v $(pwd)/docs/openapi:/usr/share/nginx/html \
  swaggerapi/swagger-ui
```

Then open http://localhost:8080 in your browser.

### Option 2: Online Swagger Editor

1. Go to https://editor.swagger.io/
2. Click "File" → "Import file"
3. Upload `docs/openapi/openapi.yaml`

### Option 3: Redoc

```bash
npm install -g redoc-cli
redoc-cli serve docs/openapi/openapi.yaml
```

## API Endpoints

The OpenAPI spec documents the following Kubernetes API endpoints:

- `GET /apis/workflow.kube-zen.io/v1alpha1/namespaces/{namespace}/jobflows` - List JobFlows
- `POST /apis/workflow.kube-zen.io/v1alpha1/namespaces/{namespace}/jobflows` - Create JobFlow
- `GET /apis/workflow.kube-zen.io/v1alpha1/namespaces/{namespace}/jobflows/{name}` - Get JobFlow
- `PUT /apis/workflow.kube-zen.io/v1alpha1/namespaces/{namespace}/jobflows/{name}` - Update JobFlow
- `DELETE /apis/workflow.kube-zen.io/v1alpha1/namespaces/{namespace}/jobflows/{name}` - Delete JobFlow

## Schema

The OpenAPI spec includes complete schemas for:

- `JobFlow` - Main CRD resource
- `JobFlowSpec` - JobFlow specification (extracted from CRD)
- `JobFlowStatus` - JobFlow status (extracted from CRD)
- `JobFlowList` - List of JobFlows
- `ObjectMeta` - Standard Kubernetes metadata
- `ListMeta` - Standard Kubernetes list metadata

## Integration

The OpenAPI specification can be used for:

- API client generation (using tools like `openapi-generator`)
- API documentation hosting
- API validation
- Testing and mocking

## Updating

When the CRD schema changes, regenerate the OpenAPI spec:

```bash
make generate-openapi
```

The script automatically extracts the latest schema from the CRD definition.

