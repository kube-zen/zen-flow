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
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	jferrors "github.com/kube-zen/zen-flow/pkg/errors"
	sdklog "github.com/kube-zen/zen-sdk/pkg/logging"
)

// archiveArtifact archives an artifact according to ArchiveConfig
func (r *JobFlowReconciler) archiveArtifact(artifactPath string, archiveConfig *v1alpha1.ArchiveConfig) (string, error) {
	logger := sdklog.NewLogger("zen-flow-controller")

	format := archiveConfig.Format
	if format == "" {
		format = DefaultArchiveFormat
	}

	compression := archiveConfig.Compression
	if compression == "" {
		compression = DefaultCompression
	}

	// Determine archive file extension
	ext := format
	if compression == "gzip" && format == "tar" {
		ext = "tar.gz"
	}

	archivePath := artifactPath + "." + ext

	logger.Info("Archiving artifact",
		sdklog.String("artifact_path", artifactPath),
		sdklog.String("archive_path", archivePath),
		sdklog.String("format", format),
		sdklog.String("compression", compression))

	switch format {
	case "tar":
		if err := r.createTarArchive(artifactPath, archivePath, compression == "gzip"); err != nil {
			return "", jferrors.Wrapf(err, "tar_archive_failed", "failed to create tar archive")
		}
	case "zip":
		if err := r.createZipArchive(artifactPath, archivePath); err != nil {
			return "", jferrors.Wrapf(err, "zip_archive_failed", "failed to create zip archive")
		}
	default:
		return "", jferrors.New("unsupported_archive_format", fmt.Sprintf("unsupported archive format: %s", format))
	}

	logger.Info("Successfully archived artifact", sdklog.String("archive_path", archivePath))
	return archivePath, nil
}

// createTarArchive creates a tar archive (optionally gzipped)
func (r *JobFlowReconciler) createTarArchive(sourcePath, archivePath string, gzipCompress bool) error {
	// Create archive file
	//nolint:gosec // archivePath is validated and comes from trusted source (JobFlow spec)
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
		defer func() {
			if err := archiveFile.Close(); err != nil {
				logger := sdklog.NewLogger("zen-flow-controller")
				logger.Error(err, "Failed to close archive file", sdklog.String("archive_path", archivePath))
			}
		}()

	var writer io.Writer = archiveFile
	if gzipCompress {
		gzipWriter := gzip.NewWriter(archiveFile)
			defer func() {
				if err := gzipWriter.Close(); err != nil {
					logger := sdklog.NewLogger("zen-flow-controller")
					logger.Error(err, "Failed to close gzip writer", sdklog.String("archive_path", archivePath))
				}
			}()
		writer = gzipWriter
	}

	tarWriter := tar.NewWriter(writer)
		defer func() {
			if err := tarWriter.Close(); err != nil {
				logger := sdklog.NewLogger("zen-flow-controller")
				logger.Error(err, "Failed to close tar writer", sdklog.String("archive_path", archivePath))
			}
		}()

	// Walk source path and add files to archive
	return filepath.Walk(sourcePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories (we'll add files)
		if info.IsDir() {
			return nil
		}

		// Open file
		//nolint:gosec // filePath comes from filepath.Walk which validates paths
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filePath, err)
		}
			defer func() {
				if err := file.Close(); err != nil {
					logger := sdklog.NewLogger("zen-flow-controller")
					logger.Error(err, "Failed to close file", sdklog.String("file_path", filePath))
				}
			}()

		// Get relative path for archive
		relPath, err := filepath.Rel(sourcePath, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// Copy file content
		if _, err := io.Copy(tarWriter, file); err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}

		return nil
	})
}

// createZipArchive creates a zip archive
func (r *JobFlowReconciler) createZipArchive(sourcePath, archivePath string) error {
	// Create archive file
	//nolint:gosec // archivePath is validated and comes from trusted source (JobFlow spec)
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
		defer func() {
			if err := archiveFile.Close(); err != nil {
				logger := sdklog.NewLogger("zen-flow-controller")
				logger.Error(err, "Failed to close archive file", sdklog.String("archive_path", archivePath))
			}
		}()

	zipWriter := zip.NewWriter(archiveFile)
		defer func() {
			if err := zipWriter.Close(); err != nil {
				logger := sdklog.NewLogger("zen-flow-controller")
				logger.Error(err, "Failed to close zip writer", sdklog.String("archive_path", archivePath))
			}
		}()

	// Walk source path and add files to archive
	return filepath.Walk(sourcePath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Open file
		//nolint:gosec // filePath comes from filepath.Walk which validates paths
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filePath, err)
		}
			defer func() {
				if err := file.Close(); err != nil {
					logger := sdklog.NewLogger("zen-flow-controller")
					logger.Error(err, "Failed to close file", sdklog.String("file_path", filePath))
				}
			}()

		// Get relative path for archive
		relPath, err := filepath.Rel(sourcePath, filePath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Create zip file header
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("failed to create zip header: %w", err)
		}
		header.Name = relPath
		header.Method = zip.Deflate

		// Create zip file writer
		zipFile, err := zipWriter.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("failed to create zip file: %w", err)
		}

		// Copy file content
		if _, err := io.Copy(zipFile, file); err != nil {
			return fmt.Errorf("failed to copy file content: %w", err)
		}

		return nil
	})
}

// isArchivePath checks if a path is an archive file
func (r *JobFlowReconciler) isArchivePath(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".tar" || ext == ".zip" || ext == ".gz" || strings.HasSuffix(strings.ToLower(path), ".tar.gz")
}

