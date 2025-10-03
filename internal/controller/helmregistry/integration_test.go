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

var _ = Describe("Component Registration at Startup Integration", func() {
	Context("Component registered during init()", func() {
		It("should successfully register component", func() {
			registry := helmregistry.NewHelmManagedComponentRegistry()

			config := helmregistry.ComponentConfig{
				ChartName: "langfuse",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
				Watches: []schema.GroupVersionKind{
					{Group: "apps", Version: "v1", Kind: "Deployment"},
				},
			}

			err := registry.Register("langfuse", config)
			Expect(err).To(BeNil(), "Registration should succeed")
		})

		It("should load chart successfully", func() {
			registry := helmregistry.NewHelmManagedComponentRegistry()

			config := helmregistry.ComponentConfig{
				ChartName: "langfuse",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
			}

			err := registry.Register("langfuse", config)
			Expect(err).To(BeNil())

			component, exists := registry.GetComponent("langfuse")
			Expect(exists).To(BeTrue())
			Expect(component.Chart).NotTo(BeNil(), "Chart should be loaded")
		})

		It("should have component in registry", func() {
			registry := helmregistry.NewHelmManagedComponentRegistry()

			config := helmregistry.ComponentConfig{
				ChartName: "langfuse",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
			}

			err := registry.Register("langfuse", config)
			Expect(err).To(BeNil())

			components := registry.ListComponents()
			Expect(components).To(ContainElement("langfuse"))
		})
	})

	Context("Operator startup succeeds with valid charts", func() {
		It("should not panic when all charts load successfully", func() {
			registry := helmregistry.NewHelmManagedComponentRegistry()

			registerFunc := func() {
				config := helmregistry.ComponentConfig{
					ChartName: "valid-chart",
					ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
						return chartutil.Values{}, nil
					},
				}

				err := registry.Register("test-component", config)
				if err != nil {
					panic(err)
				}
			}

			// Should not panic
			Expect(registerFunc).NotTo(Panic())
		})
	})

	Context("Operator startup FAILS on chart load error (FR-008)", func() {
		It("should panic when chart cannot be loaded", func() {
			registry := helmregistry.NewHelmManagedComponentRegistry()

			registerFunc := func() {
				config := helmregistry.ComponentConfig{
					ChartName: "nonexistent-chart",
					ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
						return chartutil.Values{}, nil
					},
				}

				err := registry.Register("failing-component", config)
				if err != nil {
					panic(err) // Fail-fast pattern
				}
			}

			// Should panic on chart load failure
			Expect(registerFunc).To(Panic(),
				"Operator should fail to start when chart load fails")
		})

		It("should include chart name in error message", func() {
			registry := helmregistry.NewHelmManagedComponentRegistry()

			config := helmregistry.ComponentConfig{
				ChartName: "missing-chart",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
			}

			err := registry.Register("test", config)
			if err != nil {
				Expect(err.Error()).To(ContainSubstring("missing-chart"),
					"Error should mention chart name")
			}
		})
	})
})

var _ = Describe("Component Lifecycle Integration", func() {
	var registry *helmregistry.HelmManagedComponentRegistry

	BeforeEach(func() {
		registry = helmregistry.NewHelmManagedComponentRegistry()
	})

	Context("Full component workflow", func() {
		It("should support register -> render -> apply workflow", func() {
			// Step 1: Register component
			config := helmregistry.ComponentConfig{
				ChartName: "test-chart",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{
						"replicas": 2,
					}, nil
				},
			}

			err := registry.Register("workflow-test", config)
			Expect(err).To(BeNil())

			// Step 2: Render component
			manifests, err := registry.Render("workflow-test", struct{}{})
			Expect(err).To(BeNil())
			Expect(manifests).NotTo(BeNil())

			// Step 3: Verify manifests are usable
			Expect(len(manifests)).To(BeNumerically(">", 0))
		})
	})

	Context("Multiple components can coexist", func() {
		It("should support multiple registered components", func() {
			config1 := helmregistry.ComponentConfig{
				ChartName: "component-a",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
			}

			config2 := helmregistry.ComponentConfig{
				ChartName: "component-b",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{}, nil
				},
			}

			err := registry.Register("comp-a", config1)
			Expect(err).To(BeNil())

			err = registry.Register("comp-b", config2)
			Expect(err).To(BeNil())

			components := registry.ListComponents()
			Expect(len(components)).To(Equal(2))
			Expect(components).To(ContainElement("comp-a"))
			Expect(components).To(ContainElement("comp-b"))
		})
	})
})
