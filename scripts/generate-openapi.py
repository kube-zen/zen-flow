#!/usr/bin/env python3
"""
Generate OpenAPI 3.0 specification from CRD definitions.
Extracts the OpenAPI schema from the CRD and creates a full OpenAPI spec.
"""

import yaml
import json
import sys
from pathlib import Path

def extract_schema_from_crd(crd_file):
    """Extract OpenAPI schema from CRD file."""
    with open(crd_file, 'r') as f:
        crd = yaml.safe_load(f)
    
    if 'spec' not in crd or 'versions' not in crd['spec']:
        return None, None
    
    version = crd['spec']['versions'][0]
    if 'schema' not in version or 'openAPIV3Schema' not in version['schema']:
        return None, None
    
    schema = version['schema']['openAPIV3Schema']
    if 'properties' not in schema:
        return None, None
    
    spec_schema = schema['properties'].get('spec', {})
    status_schema = schema['properties'].get('status', {})
    
    return spec_schema, status_schema

def generate_openapi_spec(crd_file, output_dir):
    """Generate full OpenAPI 3.0 specification."""
    output_dir = Path(output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)
    
    # Extract schemas from CRD
    spec_schema, status_schema = extract_schema_from_crd(crd_file)
    
    # Base OpenAPI spec
    openapi_spec = {
        'openapi': '3.0.3',
        'info': {
            'title': 'zen-flow API',
            'description': '''Kubernetes CRD API for zen-flow - A declarative workflow orchestrator for Kubernetes Jobs.

zen-flow provides a Kubernetes Custom Resource Definition (CRD) that allows you to define
workflows of Kubernetes Jobs executed in a directed acyclic graph (DAG) order.

## Features
- Sequential and parallel job execution
- DAG-based dependency management
- Retry policies with exponential backoff
- Manual approval steps
- Artifact and parameter passing between steps
- Resource template support (PVCs, ConfigMaps)
- TTL and cleanup policies''',
            'version': 'v1alpha1',
            'contact': {
                'name': 'Kube-ZEN',
                'url': 'https://github.com/kube-zen/zen-flow'
            },
            'license': {
                'name': 'Apache 2.0',
                'url': 'https://www.apache.org/licenses/LICENSE-2.0'
            }
        },
        'servers': [
            {
                'url': 'https://kubernetes.default.svc',
                'description': 'Kubernetes API Server'
            }
        ],
        'tags': [
            {
                'name': 'JobFlow',
                'description': 'JobFlow CRD operations'
            }
        ],
        'paths': {
            '/apis/workflow.kube-zen.io/v1alpha1/namespaces/{namespace}/jobflows': {
                'get': {
                    'tags': ['JobFlow'],
                    'summary': 'List JobFlows',
                    'description': 'List all JobFlows in a namespace',
                    'operationId': 'listJobFlows',
                    'parameters': [
                        {
                            'name': 'namespace',
                            'in': 'path',
                            'required': True,
                            'schema': {'type': 'string'},
                            'description': 'Namespace name'
                        },
                        {
                            'name': 'labelSelector',
                            'in': 'query',
                            'required': False,
                            'schema': {'type': 'string'},
                            'description': 'Label selector for filtering'
                        }
                    ],
                    'responses': {
                        '200': {
                            'description': 'Successful response',
                            'content': {
                                'application/json': {
                                    'schema': {'$ref': '#/components/schemas/JobFlowList'}
                                }
                            }
                        }
                    }
                },
                'post': {
                    'tags': ['JobFlow'],
                    'summary': 'Create JobFlow',
                    'description': 'Create a new JobFlow',
                    'operationId': 'createJobFlow',
                    'parameters': [
                        {
                            'name': 'namespace',
                            'in': 'path',
                            'required': True,
                            'schema': {'type': 'string'},
                            'description': 'Namespace name'
                        }
                    ],
                    'requestBody': {
                        'required': True,
                        'content': {
                            'application/json': {
                                'schema': {'$ref': '#/components/schemas/JobFlow'}
                            },
                            'application/yaml': {
                                'schema': {'$ref': '#/components/schemas/JobFlow'}
                            }
                        }
                    },
                    'responses': {
                        '201': {
                            'description': 'JobFlow created successfully',
                            'content': {
                                'application/json': {
                                    'schema': {'$ref': '#/components/schemas/JobFlow'}
                                }
                            }
                        }
                    }
                }
            },
            '/apis/workflow.kube-zen.io/v1alpha1/namespaces/{namespace}/jobflows/{name}': {
                'get': {
                    'tags': ['JobFlow'],
                    'summary': 'Get JobFlow',
                    'description': 'Get a specific JobFlow by name',
                    'operationId': 'getJobFlow',
                    'parameters': [
                        {
                            'name': 'namespace',
                            'in': 'path',
                            'required': True,
                            'schema': {'type': 'string'}
                        },
                        {
                            'name': 'name',
                            'in': 'path',
                            'required': True,
                            'schema': {'type': 'string'}
                        }
                    ],
                    'responses': {
                        '200': {
                            'description': 'Successful response',
                            'content': {
                                'application/json': {
                                    'schema': {'$ref': '#/components/schemas/JobFlow'}
                                }
                            }
                        }
                    }
                },
                'put': {
                    'tags': ['JobFlow'],
                    'summary': 'Update JobFlow',
                    'description': 'Update an existing JobFlow',
                    'operationId': 'updateJobFlow',
                    'parameters': [
                        {
                            'name': 'namespace',
                            'in': 'path',
                            'required': True,
                            'schema': {'type': 'string'}
                        },
                        {
                            'name': 'name',
                            'in': 'path',
                            'required': True,
                            'schema': {'type': 'string'}
                        }
                    ],
                    'requestBody': {
                        'required': True,
                        'content': {
                            'application/json': {
                                'schema': {'$ref': '#/components/schemas/JobFlow'}
                            }
                        }
                    },
                    'responses': {
                        '200': {
                            'description': 'JobFlow updated successfully',
                            'content': {
                                'application/json': {
                                    'schema': {'$ref': '#/components/schemas/JobFlow'}
                                }
                            }
                        }
                    }
                },
                'delete': {
                    'tags': ['JobFlow'],
                    'summary': 'Delete JobFlow',
                    'description': 'Delete a JobFlow',
                    'operationId': 'deleteJobFlow',
                    'parameters': [
                        {
                            'name': 'namespace',
                            'in': 'path',
                            'required': True,
                            'schema': {'type': 'string'}
                        },
                        {
                            'name': 'name',
                            'in': 'path',
                            'required': True,
                            'schema': {'type': 'string'}
                        }
                    ],
                    'responses': {
                        '200': {
                            'description': 'JobFlow deleted successfully'
                        }
                    }
                }
            }
        },
        'components': {
            'schemas': {
                'JobFlow': {
                    'type': 'object',
                    'description': 'JobFlow is a namespaced resource that defines a workflow of Kubernetes Jobs executed in a DAG order.',
                    'required': ['apiVersion', 'kind', 'metadata', 'spec'],
                    'properties': {
                        'apiVersion': {
                            'type': 'string',
                            'enum': ['workflow.kube-zen.io/v1alpha1']
                        },
                        'kind': {
                            'type': 'string',
                            'enum': ['JobFlow']
                        },
                        'metadata': {'$ref': '#/components/schemas/ObjectMeta'},
                        'spec': {'$ref': '#/components/schemas/JobFlowSpec'},
                        'status': {'$ref': '#/components/schemas/JobFlowStatus'}
                    }
                },
                'JobFlowList': {
                    'type': 'object',
                    'required': ['apiVersion', 'kind', 'items'],
                    'properties': {
                        'apiVersion': {
                            'type': 'string',
                            'enum': ['workflow.kube-zen.io/v1alpha1']
                        },
                        'kind': {
                            'type': 'string',
                            'enum': ['JobFlowList']
                        },
                        'metadata': {'$ref': '#/components/schemas/ListMeta'},
                        'items': {
                            'type': 'array',
                            'items': {'$ref': '#/components/schemas/JobFlow'}
                        }
                    }
                },
                'ObjectMeta': {
                    'type': 'object',
                    'description': 'Standard Kubernetes object metadata',
                    'properties': {
                        'name': {'type': 'string'},
                        'namespace': {'type': 'string'},
                        'labels': {
                            'type': 'object',
                            'additionalProperties': {'type': 'string'}
                        },
                        'annotations': {
                            'type': 'object',
                            'additionalProperties': {'type': 'string'}
                        },
                        'creationTimestamp': {
                            'type': 'string',
                            'format': 'date-time'
                        },
                        'uid': {'type': 'string'}
                    }
                },
                'ListMeta': {
                    'type': 'object',
                    'properties': {
                        'resourceVersion': {'type': 'string'},
                        'continue': {'type': 'string'}
                    }
                }
            }
        }
    }
    
    # Add extracted schemas
    if spec_schema:
        openapi_spec['components']['schemas']['JobFlowSpec'] = spec_schema
    else:
        openapi_spec['components']['schemas']['JobFlowSpec'] = {
            'type': 'object',
            'description': 'JobFlow specification'
        }
    
    if status_schema:
        openapi_spec['components']['schemas']['JobFlowStatus'] = status_schema
    else:
        openapi_spec['components']['schemas']['JobFlowStatus'] = {
            'type': 'object',
            'description': 'JobFlow status'
        }
    
    # Write YAML
    yaml_file = output_dir / 'openapi.yaml'
    with open(yaml_file, 'w') as f:
        yaml.dump(openapi_spec, f, default_flow_style=False, sort_keys=False, allow_unicode=True)
    
    # Write JSON
    json_file = output_dir / 'openapi.json'
    with open(json_file, 'w') as f:
        json.dump(openapi_spec, f, indent=2)
    
    print(f"✅ OpenAPI specification generated")
    print(f"   → {yaml_file}")
    print(f"   → {json_file}")
    
    return yaml_file, json_file

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print("Usage: generate-openapi.py <crd_file> <output_dir>")
        sys.exit(1)
    
    crd_file = sys.argv[1]
    output_dir = sys.argv[2]
    
    try:
        generate_openapi_spec(crd_file, output_dir)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

