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

package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/kube-zen/zen-flow/pkg/api/v1alpha1"
	"github.com/kube-zen/zen-flow/pkg/validation"
	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <examples-directory>\n", os.Args[0])
		os.Exit(1)
	}

	examplesDir := os.Args[1]
	if _, err := os.Stat(examplesDir); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: directory %s does not exist\n", examplesDir)
		os.Exit(1)
	}

	var errors []string
	var validated int

	err := filepath.WalkDir(examplesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to read file: %v", path, err))
			return nil
		}

		var obj map[string]interface{}
		if err := yaml.Unmarshal(data, &obj); err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to parse YAML: %v", path, err))
			return nil
		}

		// Check if it's a JobFlow
		apiVersion, ok := obj["apiVersion"].(string)
		if !ok || apiVersion != "workflow.zen.io/v1alpha1" {
			return nil // Skip non-JobFlow files
		}

		kind, ok := obj["kind"].(string)
		if !ok || kind != "JobFlow" {
			return nil // Skip non-JobFlow files
		}

		// Convert to JobFlow
		jsonData, err := json.Marshal(obj)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to convert to JSON: %v", path, err))
			return nil
		}

		var jobFlow v1alpha1.JobFlow
		if err := json.Unmarshal(jsonData, &jobFlow); err != nil {
			errors = append(errors, fmt.Sprintf("%s: failed to unmarshal JobFlow: %v", path, err))
			return nil
		}

		// Validate
		if err := validation.ValidateJobFlow(&jobFlow); err != nil {
			errors = append(errors, fmt.Sprintf("%s: validation failed: %v", path, err))
			return nil
		}

		validated++
		fmt.Printf("✅ %s: valid\n", path)
		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nValidated %d JobFlow(s)\n", validated)

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors:\n")
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "  %s\n", err)
		}
		os.Exit(1)
	}

	fmt.Println("All examples are valid! ✅")
}

