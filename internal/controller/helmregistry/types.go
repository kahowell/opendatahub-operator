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
	"context"
	"errors"
	"sync"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Common errors
var (
	ErrChartNotFound      = errors.New("chart not found")
	ErrChartLoadFailed    = errors.New("chart load failed")
	ErrDuplicateComponent = errors.New("duplicate component")
	ErrInvalidConfig      = errors.New("invalid component config")
	ErrComponentNotFound  = errors.New("component not found")
	ErrValuesGeneration   = errors.New("values generation failed")
	ErrTemplateRendering  = errors.New("template rendering failed")
	ErrInvalidManifest    = errors.New("invalid manifest")
	ErrWatchRegistration  = errors.New("watch registration failed")
	ErrInvalidGVK         = errors.New("invalid GVK")
	ErrDiscoveryFailed    = errors.New("discovery failed")
)

// ComponentConfig defines configuration for registering a Helm-managed component
type ComponentConfig struct {
	// ChartName is the name of the Helm chart (must match Chart.yaml dependency)
	ChartName string

	// ValuesGenerator generates Helm values from component spec
	ValuesGenerator func(spec interface{}) (chartutil.Values, error)

	// Watches defines resource types to watch for this component
	Watches []schema.GroupVersionKind
}

// HelmManagedComponent represents a single Helm-managed component
type HelmManagedComponent struct {
	// ChartName is the name of the Helm chart
	ChartName string

	// Chart is the loaded Helm chart metadata
	Chart *chart.Chart

	// ValuesGenerator generates values from component configuration
	ValuesGenerator func(spec interface{}) (chartutil.Values, error)

	// Watches defines resource types to watch
	Watches []schema.GroupVersionKind

	// RHOAIValues contains RHOAI-specific value overrides from values.rhoai.yaml
	RHOAIValues chartutil.Values

	// pendingWatches tracks watches waiting for CRD creation
	pendingWatches map[schema.GroupVersionKind]bool
	watchesMutex   sync.RWMutex
}

// HelmManagedComponentRegistry stores all registered Helm-managed components
type HelmManagedComponentRegistry struct {
	components map[string]*HelmManagedComponent
	mu         sync.RWMutex
}

// Global singleton registry instance
var HelmManagedComponents = NewHelmManagedComponentRegistry()

// NewHelmManagedComponentRegistry creates a new component registry
func NewHelmManagedComponentRegistry() *HelmManagedComponentRegistry {
	return &HelmManagedComponentRegistry{
		components: make(map[string]*HelmManagedComponent),
	}
}

// GetComponent retrieves a registered component by name
func (r *HelmManagedComponentRegistry) GetComponent(name string) (*HelmManagedComponent, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	component, exists := r.components[name]
	return component, exists
}

// ListComponents returns all registered component names
func (r *HelmManagedComponentRegistry) ListComponents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.components))
	for name := range r.components {
		names = append(names, name)
	}
	return names
}

// HasPendingWatches checks if component has any pending watches
func (c *HelmManagedComponent) HasPendingWatches() bool {
	c.watchesMutex.RLock()
	defer c.watchesMutex.RUnlock()

	return len(c.pendingWatches) > 0
}

// HasPendingWatchForCRD checks if a CRD matches any pending component watches
func (c *HelmManagedComponent) HasPendingWatchForCRD(crd *apiextensionsv1.CustomResourceDefinition) bool {
	c.watchesMutex.RLock()
	defer c.watchesMutex.RUnlock()

	crdGVK := schema.GroupVersionKind{
		Group:   crd.Spec.Group,
		Version: getServedVersion(crd),
		Kind:    crd.Spec.Names.Kind,
	}

	pending, exists := c.pendingWatches[crdGVK]
	return exists && pending
}

// MarkWatchRegistered marks a watch as successfully registered
func (c *HelmManagedComponent) MarkWatchRegistered(gvk schema.GroupVersionKind) {
	c.watchesMutex.Lock()
	defer c.watchesMutex.Unlock()

	delete(c.pendingWatches, gvk)
}

// MapCRDToComponent maps CRD creation event to component reconciliation request
func (c *HelmManagedComponent) MapCRDToComponent(ctx context.Context, obj client.Object) []reconcile.Request {
	crd, ok := obj.(*apiextensionsv1.CustomResourceDefinition)
	if !ok {
		return nil
	}

	if !c.HasPendingWatchForCRD(crd) {
		return nil
	}

	// Return reconcile request for this component
	return []reconcile.Request{
		{NamespacedName: client.ObjectKey{Name: c.ChartName}},
	}
}

// RegisterPendingWatch registers a watch that was waiting for CRD creation
func (c *HelmManagedComponent) RegisterPendingWatch(
	crd *apiextensionsv1.CustomResourceDefinition,
	ctrl controller.Controller,
	handler handler.EventHandler,
) error {
	if !c.HasPendingWatchForCRD(crd) {
		return nil
	}

	gvk := schema.GroupVersionKind{
		Group:   crd.Spec.Group,
		Version: getServedVersion(crd),
		Kind:    crd.Spec.Names.Kind,
	}

	// Register the watch (implementation in watches.go)
	// For now, mark as registered
	c.MarkWatchRegistered(gvk)

	return nil
}

// Helper function to get served version from CRD
func getServedVersion(crd *apiextensionsv1.CustomResourceDefinition) string {
	for _, version := range crd.Spec.Versions {
		if version.Served {
			return version.Name
		}
	}
	if len(crd.Spec.Versions) > 0 {
		return crd.Spec.Versions[0].Name
	}
	return ""
}

// Mock types for testing (will be removed when actual implementation is complete)

// MockChart represents a mock Helm chart for testing
type MockChart struct {
	DefaultValues chartutil.Values
	Templates     []*MockTemplate
}

// Metadata returns mock chart metadata
func (m *MockChart) Metadata() *chart.Metadata {
	return &chart.Metadata{
		Name:    "mock-chart",
		Version: "1.0.0",
	}
}

// MockTemplate represents a mock template file
type MockTemplate struct {
	Name string
	Data string
}

// MockController represents a mock controller for testing
type MockController struct {
	watches    []schema.GroupVersionKind
	predicates []predicate.Predicate
	mu         sync.Mutex
}

// NewMockController creates a new mock controller
func NewMockController() *MockController {
	return &MockController{
		watches:    make([]schema.GroupVersionKind, 0),
		predicates: make([]predicate.Predicate, 0),
	}
}

// Watch records a watch registration
func (m *MockController) Watch(src interface{}, handler handler.EventHandler, predicates ...predicate.Predicate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.predicates = append(m.predicates, predicates...)
	return nil
}

// WatchCount returns the number of registered watches
func (m *MockController) WatchCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.watches)
}

// GetPredicates returns registered predicates
func (m *MockController) GetPredicates() []predicate.Predicate {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]predicate.Predicate{}, m.predicates...)
}

// MockEventHandler represents a mock event handler for testing
type MockEventHandler struct{}

// NewMockEventHandler creates a new mock event handler
func NewMockEventHandler() *MockEventHandler {
	return &MockEventHandler{}
}
