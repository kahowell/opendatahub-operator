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

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Register registers a Helm-managed component at operator init time
// This function implements the contract specified in contracts/registry-api.md
func (r *HelmManagedComponentRegistry) Register(name string, config ComponentConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate component config
	if err := validateComponentConfig(config); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}

	// Check for duplicate registration
	if _, exists := r.components[name]; exists {
		return fmt.Errorf("%w: component '%s' already registered", ErrDuplicateComponent, name)
	}

	// Create component
	component := &HelmManagedComponent{
		ChartName:       config.ChartName,
		ValuesGenerator: config.ValuesGenerator,
		Watches:         config.Watches,
		pendingWatches:  make(map[schema.GroupVersionKind]bool),
	}

	// Load chart (will be implemented in loader.go)
	// For now, this will fail with chart not found error
	// which is the expected behavior for TDD
	if err := component.LoadChart(config.ChartName); err != nil {
		return fmt.Errorf("%w: %v", ErrChartLoadFailed, err)
	}

	// Store component in registry
	r.components[name] = component

	return nil
}

// Render renders Helm chart templates to Kubernetes manifests
// This function implements the contract specified in contracts/registry-api.md
func (r *HelmManagedComponentRegistry) Render(name string, spec interface{}) (map[string]string, error) {
	// Retrieve component from registry
	component, exists := r.GetComponent(name)
	if !exists {
		return nil, fmt.Errorf("%w: component '%s' not registered", ErrComponentNotFound, name)
	}

	// Generate values from component spec
	componentValues, err := component.ValuesGenerator(spec)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrValuesGeneration, err)
	}

	// Merge values with precedence: component > RHOAI > chart defaults
	finalValues := component.MergeValues(componentValues)

	// Render templates using Helm engine
	manifests, err := component.RenderTemplates(finalValues)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemplateRendering, err)
	}

	return manifests, nil
}

// validateComponentConfig validates component configuration
func validateComponentConfig(config ComponentConfig) error {
	if config.ChartName == "" {
		return fmt.Errorf("ChartName cannot be empty")
	}

	if config.ValuesGenerator == nil {
		return fmt.Errorf("ValuesGenerator cannot be nil")
	}

	return nil
}
