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

package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for zen-flow controller
type Config struct {
	// Default TTL (24 hours in seconds)
	DefaultTTLSeconds int32

	// ConfigMap size limit (1MB) - Kubernetes ConfigMap size limit
	ConfigMapSizeLimit int

	// File permissions
	DefaultDirPerm  os.FileMode
	DefaultFilePerm os.FileMode

	// UID truncation length for resource names
	UIDTruncateLength int

	// Default retry/backoff values
	DefaultBackoffLimit  int32
	DefaultRetryLimit    int
	DefaultBackoffBase   time.Duration
	DefaultBackoffFactor float64

	// Default strings
	DefaultConfigMapKey     string
	DefaultContainerName    string
	DefaultConcurrencyPolicy string
	DefaultContentType      string
	DefaultArchiveFormat    string
	DefaultCompression      string
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		DefaultTTLSeconds:     getEnvInt32("ZEN_FLOW_DEFAULT_TTL_SECONDS", 86400),
		ConfigMapSizeLimit:    getEnvInt("ZEN_FLOW_CONFIGMAP_SIZE_LIMIT", 1024*1024),
		DefaultDirPerm:        getEnvFileMode("ZEN_FLOW_DEFAULT_DIR_PERM", 0755),
		DefaultFilePerm:       getEnvFileMode("ZEN_FLOW_DEFAULT_FILE_PERM", 0644),
		UIDTruncateLength:     getEnvInt("ZEN_FLOW_UID_TRUNCATE_LENGTH", 8),
		DefaultBackoffLimit:   getEnvInt32("ZEN_FLOW_DEFAULT_BACKOFF_LIMIT", 6),
		DefaultRetryLimit:     getEnvInt("ZEN_FLOW_DEFAULT_RETRY_LIMIT", 3),
		DefaultBackoffBase:    getEnvDuration("ZEN_FLOW_DEFAULT_BACKOFF_BASE", 1*time.Second),
		DefaultBackoffFactor:  getEnvFloat64("ZEN_FLOW_DEFAULT_BACKOFF_FACTOR", 2.0),
		DefaultConfigMapKey:   getEnv("ZEN_FLOW_DEFAULT_CONFIGMAP_KEY", "value"),
		DefaultContainerName:  getEnv("ZEN_FLOW_DEFAULT_CONTAINER_NAME", "main"),
		DefaultConcurrencyPolicy: getEnv("ZEN_FLOW_DEFAULT_CONCURRENCY_POLICY", "Forbid"),
		DefaultContentType:    getEnv("ZEN_FLOW_DEFAULT_CONTENT_TYPE", "application/octet-stream"),
		DefaultArchiveFormat:  getEnv("ZEN_FLOW_DEFAULT_ARCHIVE_FORMAT", "tar"),
		DefaultCompression:    getEnv("ZEN_FLOW_DEFAULT_COMPRESSION", "none"),
	}
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as integer with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvInt32 gets an environment variable as int32 with a default value
func getEnvInt32(key string, defaultValue int32) int32 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 32); err == nil {
			return int32(intValue)
		}
	}
	return defaultValue
}

// getEnvFloat64 gets an environment variable as float64 with a default value
func getEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

// getEnvDuration gets an environment variable as duration with a default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// getEnvFileMode gets an environment variable as os.FileMode with a default value
func getEnvFileMode(key string, defaultValue os.FileMode) os.FileMode {
	if value := os.Getenv(key); value != "" {
		// Parse as octal (e.g., "0755")
		if intValue, err := strconv.ParseInt(value, 8, 32); err == nil {
			//nolint:gosec // FileMode is uint32, ParseInt with bitSize 32 ensures value fits
			if intValue >= 0 && intValue <= 0xFFFFFFFF {
				return os.FileMode(uint32(intValue))
			}
		}
		// Try decimal as fallback
		if intValue, err := strconv.ParseInt(value, 10, 32); err == nil {
			//nolint:gosec // FileMode is uint32, ParseInt with bitSize 32 ensures value fits
			if intValue >= 0 && intValue <= 0xFFFFFFFF {
				return os.FileMode(uint32(intValue))
			}
		}
	}
	return defaultValue
}

