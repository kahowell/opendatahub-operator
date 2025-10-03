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
)

var _ = Describe("MergeValues() Contract", func() {
	var component *helmregistry.HelmManagedComponent

	BeforeEach(func() {
		// Create a test component with chart defaults and RHOAI values
		component = &helmregistry.HelmManagedComponent{
			ChartName: "test-chart",
			Chart: &helmregistry.MockChart{
				DefaultValues: chartutil.Values{
					"replicas": 1,
					"image": map[string]interface{}{
						"tag": "latest",
						"pullPolicy": "Always",
					},
					"features": map[string]interface{}{
						"enabled": false,
					},
				},
			},
			RHOAIValues: chartutil.Values{
				"replicas": 2, // RHOAI override
				"image": map[string]interface{}{
					"pullPolicy": "IfNotPresent", // RHOAI override
				},
			},
		}
	})

	Context("Component values override RHOAI values", func() {
		It("should give precedence to component config over RHOAI", func() {
			componentValues := chartutil.Values{
				"replicas": 3, // Component wins
			}

			result := component.MergeValues(componentValues)

			Expect(result["replicas"]).To(Equal(3), "Component value should override RHOAI")
		})
	})

	Context("RHOAI values override chart defaults", func() {
		It("should give precedence to RHOAI over chart defaults", func() {
			componentValues := chartutil.Values{
				// No replicas override from component
			}

			result := component.MergeValues(componentValues)

			Expect(result["replicas"]).To(Equal(2), "RHOAI value should override chart default")
		})
	})

	Context("Deep nested map merging", func() {
		It("should merge nested maps recursively", func() {
			componentValues := chartutil.Values{
				"image": map[string]interface{}{
					"tag": "v1.2.3", // Component overrides tag
					// pullPolicy not specified, should come from RHOAI
				},
			}

			result := component.MergeValues(componentValues)

			imageMap := result["image"].(map[string]interface{})
			Expect(imageMap["tag"]).To(Equal("v1.2.3"), "Component value for nested field")
			Expect(imageMap["pullPolicy"]).To(Equal("IfNotPresent"), "RHOAI value for nested field")
		})
	})

	Context("Array replacement (not merge)", func() {
		It("should replace arrays rather than merging them", func() {
			component.Chart = &helmregistry.MockChart{
				DefaultValues: chartutil.Values{
					"tolerations": []interface{}{
						map[string]interface{}{"key": "default", "operator": "Exists"},
					},
				},
			}
			component.RHOAIValues = chartutil.Values{}

			componentValues := chartutil.Values{
				"tolerations": []interface{}{
					map[string]interface{}{"key": "component", "operator": "Equal", "value": "test"},
				},
			}

			result := component.MergeValues(componentValues)

			tolerations := result["tolerations"].([]interface{})
			Expect(len(tolerations)).To(Equal(1), "Array should be replaced, not merged")
			tolMap := tolerations[0].(map[string]interface{})
			Expect(tolMap["key"]).To(Equal("component"), "Array should be from component config")
		})
	})

	Context("Null value handling", func() {
		It("should remove values when higher precedence provides null", func() {
			component.Chart = &helmregistry.MockChart{
				DefaultValues: chartutil.Values{
					"serviceAccount": map[string]interface{}{
						"create": true,
						"name":   "default-sa",
					},
				},
			}
			component.RHOAIValues = chartutil.Values{}

			componentValues := chartutil.Values{
				"serviceAccount": map[string]interface{}{
					"create": false,
					"name":   nil, // Null removes the field
				},
			}

			result := component.MergeValues(componentValues)

			saMap := result["serviceAccount"].(map[string]interface{})
			Expect(saMap["create"]).To(Equal(false))
			_, hasName := saMap["name"]
			Expect(hasName).To(BeFalse(), "Null value should remove field")
		})
	})

	Context("Complete merge precedence validation", func() {
		It("should demonstrate full precedence: component > RHOAI > chart", func() {
			// Chart defaults
			component.Chart = &helmregistry.MockChart{
				DefaultValues: chartutil.Values{
					"replicas": 1,
					"features": map[string]interface{}{
						"experimental": false,
						"telemetry":    true,
					},
				},
			}

			// RHOAI overrides
			component.RHOAIValues = chartutil.Values{
				"replicas": 2, // Override chart default
			}

			// Component config
			componentValues := chartutil.Values{
				"features": map[string]interface{}{
					"experimental": true, // Override chart default
				},
			}

			result := component.MergeValues(componentValues)

			Expect(result["replicas"]).To(Equal(2), "From RHOAI")
			featuresMap := result["features"].(map[string]interface{})
			Expect(featuresMap["experimental"]).To(Equal(true), "From component")
			Expect(featuresMap["telemetry"]).To(Equal(true), "From chart default")
		})
	})
})

var _ = Describe("ValuesGenerator Contract", func() {
	Context("Valid values generation", func() {
		It("should generate proper chartutil.Values from component spec", func() {
			// Mock component spec
			type TestSpec struct {
				Replicas int
				Features map[string]bool
			}

			spec := TestSpec{
				Replicas: 3,
				Features: map[string]bool{
					"experimental": true,
				},
			}

			// Test generator function
			generator := func(spec interface{}) (chartutil.Values, error) {
				testSpec := spec.(TestSpec)
				return chartutil.Values{
					"replicas": testSpec.Replicas,
					"features": map[string]interface{}{
						"experimental": testSpec.Features["experimental"],
					},
				}, nil
			}

			values, err := generator(spec)
			Expect(err).To(BeNil())
			Expect(values["replicas"]).To(Equal(3))
			featuresMap := values["features"].(map[string]interface{})
			Expect(featuresMap["experimental"]).To(Equal(true))
		})
	})

	Context("Error handling in values generation", func() {
		It("should propagate errors from generator function", func() {
			generator := func(spec interface{}) (chartutil.Values, error) {
				return nil, Err("invalid spec")
			}

			values, err := generator(struct{}{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid"))
			Expect(values).To(BeNil())
		})
	})
})
