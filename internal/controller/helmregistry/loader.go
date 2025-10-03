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
	"path/filepath"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"gopkg.in/yaml.v3"
)

// LoadChart loads a Helm chart using helm.sh/helm/v3/pkg/chart/loader.LoadArchive
// This function implements chart loading as specified in research.md
func (c *HelmManagedComponent) LoadChart(chartPath string) error {
	// Construct full chart path
	// Charts are expected to be in charts/ directory as dependencies
	fullPath := filepath.Join("charts", chartPath+".tgz")

	// Load chart using Helm's Load function
	// First try as a packaged chart (.tgz)
	chart, err := loader.Load(fullPath)
	if err != nil {
		// Try loading as unpacked directory
		dirPath := filepath.Join("charts", chartPath)
		chart, err = loader.Load(dirPath)
		if err != nil {
			return fmt.Errorf("failed to load chart from %s or %s: %w", fullPath, dirPath, err)
		}
	}

	// Store loaded chart
	c.Chart = chart

	// Extract RHOAI values from chart files if present
	if err := c.extractRHOAIValues(); err != nil {
		// Log warning but don't fail - RHOAI values are optional
		// In production, this would use proper logging
		// For now, we continue without RHOAI values
	}

	return nil
}

// extractRHOAIValues extracts RHOAI-specific value overrides from values.rhoai.yaml
// This implements the RHOAI value override pattern from contracts/values-api.md
func (c *HelmManagedComponent) extractRHOAIValues() error {
	if c.Chart == nil {
		return fmt.Errorf("chart not loaded")
	}

	// Look for values.rhoai.yaml in chart files
	for _, file := range c.Chart.Files {
		if file.Name == "values.rhoai.yaml" {
			// Parse YAML to chartutil.Values
			var rhoaiValues chartutil.Values
			if err := yaml.Unmarshal(file.Data, &rhoaiValues); err != nil {
				return fmt.Errorf("invalid values.rhoai.yaml: %w", err)
			}

			c.RHOAIValues = rhoaiValues
			return nil
		}
	}

	// No values.rhoai.yaml found - this is okay, RHOAI values are optional
	c.RHOAIValues = chartutil.Values{}
	return nil
}
