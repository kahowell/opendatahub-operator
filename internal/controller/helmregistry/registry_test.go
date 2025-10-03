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

package helmregistry_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opendatahub-io/opendatahub-operator/v2/internal/controller/helmregistry"
	"helm.sh/helm/v3/pkg/chartutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("Registry.Register() Contract", func() {
	var registry *helmregistry.HelmManagedComponentRegistry

	BeforeEach(func() {
		// Create a new registry for each test
		registry = helmregistry.NewHelmManagedComponentRegistry()
	})

	Context("Successful registration with valid chart", func() {
		It("should register component with valid config", func() {
			config := helmregistry.ComponentConfig{
				ChartName: "test-chart",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
				Watches: []schema.GroupVersionKind{
					{Group: "apps", Version: "v1", Kind: "Deployment"},
				},
			}

			err := registry.Register("test-component", config)
			Expect(err).To(BeNil(), "Registration should succeed with valid config")

			// Verify component is in registry
			component, exists := registry.GetComponent("test-component")
			Expect(exists).To(BeTrue(), "Component should exist in registry")
			Expect(component).NotTo(BeNil())
			Expect(component.ChartName).To(Equal("test-chart"))
		})
	})

	Context("Error on duplicate component name", func() {
		It("should return ErrDuplicateComponent when registering same name twice", func() {
			config := helmregistry.ComponentConfig{
				ChartName: "test-chart",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
			}

			// First registration should succeed
			err := registry.Register("duplicate", config)
			Expect(err).To(BeNil())

			// Second registration with same name should fail
			err = registry.Register("duplicate", config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("duplicate"), "Error should mention duplicate component")
		})
	})

	Context("Error on chart not found", func() {
		It("should return ErrChartNotFound when chart doesn't exist", func() {
			config := helmregistry.ComponentConfig{
				ChartName: "nonexistent-chart",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
			}

			err := registry.Register("test", config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("chart"), "Error should mention chart not found")
		})
	})

	Context("Error on nil ValuesGenerator", func() {
		It("should return ErrInvalidConfig when ValuesGenerator is nil", func() {
			config := helmregistry.ComponentConfig{
				ChartName:       "test-chart",
				ValuesGenerator: nil, // Invalid: nil generator
			}

			err := registry.Register("test", config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid"), "Error should mention invalid config")
		})
	})

	Context("Error on empty ChartName", func() {
		It("should return ErrInvalidConfig when ChartName is empty", func() {
			config := helmregistry.ComponentConfig{
				ChartName: "", // Invalid: empty name
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
			}

			err := registry.Register("test", config)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid"), "Error should mention invalid config")
		})
	})
})

var _ = Describe("Registry.Render() Contract", func() {
	var registry *helmregistry.HelmManagedComponentRegistry

	BeforeEach(func() {
		registry = helmregistry.NewHelmManagedComponentRegistry()
	})

	Context("Successful rendering with valid values", func() {
		It("should render templates and return manifest map", func() {
			// Register a test component first
			config := helmregistry.ComponentConfig{
				ChartName: "test-chart",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{
						"replicas": 2,
					}, nil
				},
			}
			err := registry.Register("test-component", config)
			Expect(err).To(BeNil())

			// Render the component
			manifests, err := registry.Render("test-component", struct{}{})
			Expect(err).To(BeNil(), "Render should succeed with valid component")
			Expect(manifests).NotTo(BeNil())
			Expect(manifests).To(BeAssignableToTypeOf(map[string]string{}))
		})

		It("should return deterministic output for same inputs (NFR-002)", func() {
			config := helmregistry.ComponentConfig{
				ChartName: "test-chart",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{"replicas": 1}, nil
				},
			}
			err := registry.Register("deterministic-test", config)
			Expect(err).To(BeNil())

			// Render twice with same inputs
			manifests1, err1 := registry.Render("deterministic-test", struct{}{})
			Expect(err1).To(BeNil())

			manifests2, err2 := registry.Render("deterministic-test", struct{}{})
			Expect(err2).To(BeNil())

			// Results should be identical
			Expect(manifests1).To(Equal(manifests2), "Same inputs should produce same output")
		})
	})

	Context("Error on component not found", func() {
		It("should return ErrComponentNotFound when component not registered", func() {
			manifests, err := registry.Render("nonexistent", struct{}{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"), "Error should mention component not found")
			Expect(manifests).To(BeNil())
		})
	})

	Context("Error on template rendering failure", func() {
		It("should return ErrTemplateRendering when template has syntax error", func() {
			config := helmregistry.ComponentConfig{
				ChartName: "invalid-template-chart",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
			}
			err := registry.Register("bad-template", config)
			// Registration might succeed even with bad templates
			if err == nil {
				manifests, err := registry.Render("bad-template", struct{}{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("render"), "Error should mention rendering failure")
				Expect(manifests).To(BeNil())
			}
		})
	})

	Context("Error on values generation failure", func() {
		It("should return ErrValuesGeneration when ValuesGenerator fails", func() {
			config := helmregistry.ComponentConfig{
				ChartName: "test-chart",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return nil, Err("values generation failed")
				},
			}
			err := registry.Register("failing-generator", config)
			Expect(err).To(BeNil())

			manifests, err := registry.Render("failing-generator", struct{}{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("values"), "Error should mention values generation")
			Expect(manifests).To(BeNil())
		})
	})
})

// Helper function to create errors
func Err(msg string) error {
	return &testError{msg: msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
