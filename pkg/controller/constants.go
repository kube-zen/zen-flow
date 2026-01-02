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

import (
	"github.com/kube-zen/zen-flow/pkg/config"
)

var (
	// Config holds the loaded configuration
	// Loaded once at package initialization
	Config = config.Load()
)

// Deprecated: Use Config.DefaultTTLSeconds instead
// Default TTL (24 hours in seconds)
var DefaultTTLSeconds = Config.DefaultTTLSeconds

// Deprecated: Use Config.ConfigMapSizeLimit instead
// ConfigMap size limit (1MB) - Kubernetes ConfigMap size limit
var ConfigMapSizeLimit = Config.ConfigMapSizeLimit

// Deprecated: Use Config.DefaultDirPerm instead
// File permissions
var DefaultDirPerm = Config.DefaultDirPerm

// Deprecated: Use Config.DefaultFilePerm instead
var DefaultFilePerm = Config.DefaultFilePerm

// Deprecated: Use Config.UIDTruncateLength instead
// UID truncation length for resource names
var UIDTruncateLength = Config.UIDTruncateLength

// Deprecated: Use Config.DefaultBackoffLimit instead
// Default retry/backoff values
var DefaultBackoffLimit = Config.DefaultBackoffLimit

// Deprecated: Use Config.DefaultRetryLimit instead
var DefaultRetryLimit = Config.DefaultRetryLimit

// Deprecated: Use Config.DefaultBackoffBase instead
var DefaultBackoffBase = Config.DefaultBackoffBase

// Deprecated: Use Config.DefaultBackoffFactor instead
var DefaultBackoffFactor = Config.DefaultBackoffFactor

// Deprecated: Use Config.DefaultConfigMapKey instead
// Default strings
var DefaultConfigMapKey = Config.DefaultConfigMapKey

// Deprecated: Use Config.DefaultContainerName instead
var DefaultContainerName = Config.DefaultContainerName

// Deprecated: Use Config.DefaultConcurrencyPolicy instead
var DefaultConcurrencyPolicy = Config.DefaultConcurrencyPolicy

// Deprecated: Use Config.DefaultContentType instead
var DefaultContentType = Config.DefaultContentType

// Deprecated: Use Config.DefaultArchiveFormat instead
var DefaultArchiveFormat = Config.DefaultArchiveFormat

// Deprecated: Use Config.DefaultCompression instead
var DefaultCompression = Config.DefaultCompression

