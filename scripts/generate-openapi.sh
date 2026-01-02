#!/bin/bash
# Generate OpenAPI 3.0 specification from CRD definitions
# This script extracts the OpenAPI schema from the CRD and creates a full OpenAPI spec

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/docs/openapi"
CRD_FILE="$PROJECT_ROOT/deploy/crds/workflow.kube-zen.io_jobflows.yaml"

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Generating OpenAPI 3.0 Specification${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Generate OpenAPI spec
cat > "$OUTPUT_DIR/openapi.yaml" << 'EOF'
openapi: 3.0.3
info:
  title: zen-flow API
  description: |
    Kubernetes CRD API for zen-flow - A declarative workflow orchestrator for Kubernetes Jobs.
    
    zen-flow provides a Kubernetes Custom Resource Definition (CRD) that allows you to define
    workflows of Kubernetes Jobs executed in a directed acyclic graph (DAG) order.
    
    ## Features
    - Sequential and parallel job execution
    - DAG-based dependency management
    - Retry policies with exponential backoff
    - Manual approval steps
    - Artifact and parameter passing between steps
    - Resource template support (PVCs, ConfigMaps)
    - TTL and cleanup policies
    
  version: v1alpha1
  contact:
    name: Kube-ZEN
    url: https://github.com/kube-zen/zen-flow
  license:
    name: Apache 2.0
    url: https://www.apache.org/licenses/LICENSE-2.0

servers:
  - url: https://kubernetes.default.svc
    description: Kubernetes API Server

tags:
  - name: JobFlow
    description: JobFlow CRD operations

paths:
  /apis/workflow.kube-zen.io/v1alpha1/namespaces/{namespace}/jobflows:
    get:
      tags:
        - JobFlow
      summary: List JobFlows
      description: List all JobFlows in a namespace
      operationId: listJobFlows
      parameters:
        - name: namespace
          in: path
          required: true
          schema:
            type: string
          description: Namespace name
        - name: labelSelector
          in: query
          required: false
          schema:
            type: string
          description: Label selector for filtering
        - name: fieldSelector
          in: query
          required: false
          schema:
            type: string
          description: Field selector for filtering
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/JobFlowList'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '403':
          $ref: '#/components/responses/Forbidden'
    
    post:
      tags:
        - JobFlow
      summary: Create JobFlow
      description: Create a new JobFlow
      operationId: createJobFlow
      parameters:
        - name: namespace
          in: path
          required: true
          schema:
            type: string
          description: Namespace name
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/JobFlow'
          application/yaml:
            schema:
              $ref: '#/components/schemas/JobFlow'
      responses:
        '201':
          description: JobFlow created successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/JobFlow'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '403':
          $ref: '#/components/responses/Forbidden'
  
  /apis/workflow.kube-zen.io/v1alpha1/namespaces/{namespace}/jobflows/{name}:
    get:
      tags:
        - JobFlow
      summary: Get JobFlow
      description: Get a specific JobFlow by name
      operationId: getJobFlow
      parameters:
        - name: namespace
          in: path
          required: true
          schema:
            type: string
          description: Namespace name
        - name: name
          in: path
          required: true
          schema:
            type: string
          description: JobFlow name
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/JobFlow'
        '404':
          $ref: '#/components/responses/NotFound'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '403':
          $ref: '#/components/responses/Forbidden'
    
    put:
      tags:
        - JobFlow
      summary: Update JobFlow
      description: Update an existing JobFlow
      operationId: updateJobFlow
      parameters:
        - name: namespace
          in: path
          required: true
          schema:
            type: string
          description: Namespace name
        - name: name
          in: path
          required: true
          schema:
            type: string
          description: JobFlow name
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/JobFlow'
          application/yaml:
            schema:
              $ref: '#/components/schemas/JobFlow'
      responses:
        '200':
          description: JobFlow updated successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/JobFlow'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '403':
          $ref: '#/components/responses/Forbidden'
        '404':
          $ref: '#/components/responses/NotFound'
    
    delete:
      tags:
        - JobFlow
      summary: Delete JobFlow
      description: Delete a JobFlow
      operationId: deleteJobFlow
      parameters:
        - name: namespace
          in: path
          required: true
          schema:
            type: string
          description: Namespace name
        - name: name
          in: path
          required: true
          schema:
            type: string
          description: JobFlow name
      responses:
        '200':
          description: JobFlow deleted successfully
          content:
            application/json:
              schema:
                type: object
        '401':
          $ref: '#/components/responses/Unauthorized'
        '403':
          $ref: '#/components/responses/Forbidden'
        '404':
          $ref: '#/components/responses/NotFound'
  
  /apis/workflow.kube-zen.io/v1alpha1/namespaces/{namespace}/jobflows/{name}/status:
    get:
      tags:
        - JobFlow
      summary: Get JobFlow Status
      description: Get the status subresource of a JobFlow
      operationId: getJobFlowStatus
      parameters:
        - name: namespace
          in: path
          required: true
          schema:
            type: string
          description: Namespace name
        - name: name
          in: path
          required: true
          schema:
            type: string
          description: JobFlow name
      responses:
        '200':
          description: Successful response
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/JobFlowStatus'
        '404':
          $ref: '#/components/responses/NotFound'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '403':
          $ref: '#/components/responses/Forbidden'
    
    patch:
      tags:
        - JobFlow
      summary: Update JobFlow Status
      description: Update the status subresource of a JobFlow
      operationId: updateJobFlowStatus
      parameters:
        - name: namespace
          in: path
          required: true
          schema:
            type: string
          description: Namespace name
        - name: name
          in: path
          required: true
          schema:
            type: string
          description: JobFlow name
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/JobFlowStatus'
          application/merge-patch+json:
            schema:
              $ref: '#/components/schemas/JobFlowStatus'
      responses:
        '200':
          description: Status updated successfully
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/JobFlowStatus'
        '400':
          $ref: '#/components/responses/BadRequest'
        '401':
          $ref: '#/components/responses/Unauthorized'
        '403':
          $ref: '#/components/responses/Forbidden'
        '404':
          $ref: '#/components/responses/NotFound'

components:
  schemas:
    JobFlow:
      type: object
      description: JobFlow is a namespaced resource that defines a workflow of Kubernetes Jobs executed in a DAG order.
      required:
        - apiVersion
        - kind
        - metadata
        - spec
      properties:
        apiVersion:
          type: string
          enum: [workflow.kube-zen.io/v1alpha1]
          description: API version
        kind:
          type: string
          enum: [JobFlow]
          description: Resource kind
        metadata:
          $ref: '#/components/schemas/ObjectMeta'
        spec:
          $ref: '#/components/schemas/JobFlowSpec'
        status:
          $ref: '#/components/schemas/JobFlowStatus'
    
    JobFlowList:
      type: object
      description: List of JobFlows
      required:
        - apiVersion
        - kind
        - items
      properties:
        apiVersion:
          type: string
          enum: [workflow.kube-zen.io/v1alpha1]
        kind:
          type: string
          enum: [JobFlowList]
        metadata:
          $ref: '#/components/schemas/ListMeta'
        items:
          type: array
          items:
            $ref: '#/components/schemas/JobFlow'
EOF

# Extract schema from CRD and append to OpenAPI spec
# We'll use yq or python to extract the openAPIV3Schema from the CRD
if command -v yq >/dev/null 2>&1; then
    echo -e "${GREEN}Extracting schema from CRD using yq...${NC}"
    CRD_SCHEMA=$(yq eval '.spec.versions[0].schema.openAPIV3Schema' "$CRD_FILE" 2>/dev/null || echo "")
    
    if [ -n "$CRD_SCHEMA" ] && [ "$CRD_SCHEMA" != "null" ]; then
        # Append the extracted schema to components.schemas
        echo "    JobFlowSpec:" >> "$OUTPUT_DIR/openapi.yaml"
        yq eval '.spec.versions[0].schema.openAPIV3Schema.properties.spec' "$CRD_FILE" | sed 's/^/      /' >> "$OUTPUT_DIR/openapi.yaml" 2>/dev/null || true
        echo "    JobFlowStatus:" >> "$OUTPUT_DIR/openapi.yaml"
        yq eval '.spec.versions[0].schema.openAPIV3Schema.properties.status' "$CRD_FILE" | sed 's/^/      /' >> "$OUTPUT_DIR/openapi.yaml" 2>/dev/null || true
    fi
elif command -v python3 >/dev/null 2>&1; then
    echo -e "${GREEN}Generating OpenAPI spec using Python...${NC}"
    python3 "$SCRIPT_DIR/generate-openapi.py" "$CRD_FILE" "$OUTPUT_DIR"
    exit $?
else
    echo -e "${YELLOW}⚠️  yq or python3 not found. Using basic schema structure.${NC}"
    echo "    JobFlowSpec:" >> "$OUTPUT_DIR/openapi.yaml"
    echo "      type: object" >> "$OUTPUT_DIR/openapi.yaml"
    echo "      description: JobFlow specification" >> "$OUTPUT_DIR/openapi.yaml"
    echo "      properties:" >> "$OUTPUT_DIR/openapi.yaml"
    echo "        steps:" >> "$OUTPUT_DIR/openapi.yaml"
    echo "          type: array" >> "$OUTPUT_DIR/openapi.yaml"
    echo "          description: List of steps to execute" >> "$OUTPUT_DIR/openapi.yaml"
    echo "          items:" >> "$OUTPUT_DIR/openapi.yaml"
    echo "            type: object" >> "$OUTPUT_DIR/openapi.yaml"
fi

# Add common Kubernetes types
cat >> "$OUTPUT_DIR/openapi.yaml" << 'EOF'
    
    ObjectMeta:
      type: object
      description: Standard Kubernetes object metadata
      properties:
        name:
          type: string
          description: Resource name
        namespace:
          type: string
          description: Resource namespace
        labels:
          type: object
          additionalProperties:
            type: string
          description: Resource labels
        annotations:
          type: object
          additionalProperties:
            type: string
          description: Resource annotations
        creationTimestamp:
          type: string
          format: date-time
          description: Creation timestamp
        uid:
          type: string
          description: Unique identifier
    
    ListMeta:
      type: object
      description: Standard Kubernetes list metadata
      properties:
        resourceVersion:
          type: string
          description: Resource version
        continue:
          type: string
          description: Continue token for pagination

  responses:
    BadRequest:
      description: Bad request
      content:
        application/json:
          schema:
            type: object
            properties:
              message:
                type: string
    
    Unauthorized:
      description: Unauthorized
      content:
        application/json:
          schema:
            type: object
            properties:
              message:
                type: string
    
    Forbidden:
      description: Forbidden
      content:
        application/json:
          schema:
            type: object
            properties:
              message:
                type: string
    
    NotFound:
      description: Resource not found
      content:
        application/json:
          schema:
            type: object
            properties:
              message:
                type: string
EOF

# Generate JSON version
if command -v yq >/dev/null 2>&1; then
    yq eval -o=json "$OUTPUT_DIR/openapi.yaml" > "$OUTPUT_DIR/openapi.json" 2>/dev/null || {
        echo -e "${YELLOW}⚠️  Failed to convert to JSON, install yq for JSON output${NC}"
    }
elif command -v python3 >/dev/null 2>&1; then
    python3 << 'PYTHON_SCRIPT' > "$OUTPUT_DIR/openapi.json"
import yaml
import json
import sys

try:
    with open('$OUTPUT_DIR/openapi.yaml', 'r') as f:
        spec = yaml.safe_load(f)
    print(json.dumps(spec, indent=2))
except Exception as e:
    print(f"{{'error': '{e}'}}", file=sys.stderr)
    sys.exit(1)
PYTHON_SCRIPT
else
    echo -e "${YELLOW}⚠️  yq or python3 not found. JSON version not generated.${NC}"
fi

echo -e "${GREEN}✅ OpenAPI specification generated${NC}"
echo -e "   → $OUTPUT_DIR/openapi.yaml"
if [ -f "$OUTPUT_DIR/openapi.json" ]; then
    echo -e "   → $OUTPUT_DIR/openapi.json"
fi

