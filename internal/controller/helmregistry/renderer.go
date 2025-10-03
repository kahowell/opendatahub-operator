/*
Copyright 2025.

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

package helmregistry

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"gopkg.in/yaml.v3"
)

// RenderTemplates renders Helm chart templates using the Helm engine
// This implements ArgoCD-style template-only rendering from research.md
func (c *HelmManagedComponent) RenderTemplates(values chartutil.Values) (map[string]string, error) {
	if c.Chart == nil {
		return nil, fmt.Errorf("chart not loaded")
	}

	// Create Helm rendering engine
	renderer := engine.Engine{}

	// Render all templates with provided values
	manifests, err := renderer.Render(c.Chart, values)
	if err != nil {
		return nil, fmt.Errorf("template rendering failed: %w", err)
	}

	// Filter out excluded files (ArgoCD pattern)
	filtered := make(map[string]string)
	for filename, content := range manifests {
		if shouldExcludeFile(filename) {
			continue
		}

		// Validate that output is valid YAML
		if err := validateYAML(content); err != nil {
			return nil, fmt.Errorf("invalid YAML in %s: %w", filename, err)
		}

		filtered[filename] = content
	}

	return filtered, nil
}

// shouldExcludeFile checks if a file should be excluded from rendered output
// Excludes: NOTES.txt, tests/, hooks/ (ArgoCD-style rendering)
func shouldExcludeFile(filename string) bool {
	// Exclude NOTES.txt
	if strings.Contains(filename, "NOTES.txt") {
		return true
	}

	// Exclude test files
	if strings.Contains(filename, "/tests/") {
		return true
	}

	// Exclude hooks
	if strings.Contains(filename, "/hooks/") {
		return true
	}

	return false
}

// validateYAML validates that content is parseable as YAML
func validateYAML(content string) error {
	var parsed interface{}
	if err := yaml.Unmarshal([]byte(content), &parsed); err != nil {
		return err
	}
	return nil
}
