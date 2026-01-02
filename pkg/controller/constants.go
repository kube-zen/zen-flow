/*
Copyright 2025 Kube-ZEN Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import "time"

const (
	// Default TTL (24 hours in seconds)
	DefaultTTLSeconds int32 = 86400

	// ConfigMap size limit (1MB) - Kubernetes ConfigMap size limit
	ConfigMapSizeLimit = 1024 * 1024

	// File permissions
	DefaultDirPerm  = 0755
	DefaultFilePerm = 0644

	// UID truncation length for resource names
	UIDTruncateLength = 8

	// Default retry/backoff values
	DefaultBackoffLimit  = 6
	DefaultRetryLimit    = 3
	DefaultBackoffBase   = 1 * time.Second
	DefaultBackoffFactor = 2.0

	// Default strings
	DefaultConfigMapKey      = "value"
	DefaultContainerName      = "main"
	DefaultConcurrencyPolicy  = "Forbid"
	DefaultContentType        = "application/octet-stream"
	DefaultArchiveFormat      = "tar"
	DefaultCompression        = "none"
)

