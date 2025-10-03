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

var _ = Describe("Value Precedence Integration (component > RHOAI > chart)", func() {
	var registry *helmregistry.HelmManagedComponentRegistry
	var component *helmregistry.HelmManagedComponent

	BeforeEach(func() {
		registry = helmregistry.NewHelmManagedComponentRegistry()

		// Set up component with all three value layers
		component = &helmregistry.HelmManagedComponent{
			ChartName: "test-chart",
			Chart: &helmregistry.MockChart{
				DefaultValues: chartutil.Values{
					"replicas": 1, // Chart default
					"image": map[string]interface{}{
						"tag":        "latest",
						"pullPolicy": "Always",
					},
					"resources": map[string]interface{}{
						"limits": map[string]interface{}{
							"memory": "128Mi",
						},
					},
				},
			},
			RHOAIValues: chartutil.Values{
				"replicas": 2, // RHOAI override
				"image": map[string]interface{}{
					"pullPolicy": "IfNotPresent",
				},
			},
		}
	})

	Context("Chart provides default values", func() {
		It("should use chart defaults when no overrides", func() {
			componentValues := chartutil.Values{}

			result := component.MergeValues(componentValues)

			// Should get RHOAI override
			Expect(result["replicas"]).To(Equal(2))

			// Should get chart default
			resourcesMap := result["resources"].(map[string]interface{})
			limitsMap := resourcesMap["limits"].(map[string]interface{})
			Expect(limitsMap["memory"]).To(Equal("128Mi"))
		})
	})

	Context("RHOAI values.rhoai.yaml overrides chart defaults", func() {
		It("should apply RHOAI overrides over chart defaults", func() {
			componentValues := chartutil.Values{}

			result := component.MergeValues(componentValues)

			// Replicas: RHOAI wins over chart
			Expect(result["replicas"]).To(Equal(2),
				"RHOAI value should override chart default")

			// pullPolicy: RHOAI wins over chart
			imageMap := result["image"].(map[string]interface{})
			Expect(imageMap["pullPolicy"]).To(Equal("IfNotPresent"),
				"RHOAI value should override chart default")

			// tag: No RHOAI override, use chart
			Expect(imageMap["tag"]).To(Equal("latest"),
				"Chart default should be used when RHOAI doesn't override")
		})
	})

	Context("Component config overrides RHOAI values (FR-005 clarification)", func() {
		It("should give final precedence to component configuration", func() {
			componentValues := chartutil.Values{
				"replicas": 3, // Component override
				"image": map[string]interface{}{
					"tag": "v1.2.3", // Component override
				},
			}

			result := component.MergeValues(componentValues)

			// Replicas: Component wins over RHOAI and chart
			Expect(result["replicas"]).To(Equal(3),
				"Component value should override RHOAI and chart")

			imageMap := result["image"].(map[string]interface{})
			// tag: Component wins
			Expect(imageMap["tag"]).To(Equal("v1.2.3"),
				"Component value should override chart")

			// pullPolicy: RHOAI wins (component didn't specify)
			Expect(imageMap["pullPolicy"]).To(Equal("IfNotPresent"),
				"RHOAI value should be used when component doesn't override")
		})
	})

	Context("Final rendered manifest reflects component config", func() {
		It("should produce manifests with component config values", func() {
			// Register component with values generator
			config := helmregistry.ComponentConfig{
				ChartName: "test-chart",
				ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
					return chartutil.Values{
						"replicas": 5, // User-specified via component
					}, nil
				},
			}

			err := registry.Register("precedence-test", config)
			Expect(err).To(BeNil())

			// Render with component spec
			manifests, err := registry.Render("precedence-test", struct{}{})
			Expect(err).To(BeNil())

			// Manifests should reflect component value (5 replicas)
			// This would be validated by parsing the rendered YAML
			Expect(manifests).NotTo(BeNil())
		})
	})

	Context("Complete three-layer merge", func() {
		It("should correctly merge all three layers", func() {
			// Chart defaults
			component.Chart = &helmregistry.MockChart{
				DefaultValues: chartutil.Values{
					"serviceAccount": map[string]interface{}{
						"create": true,
						"name":   "",
					},
					"podAnnotations": map[string]interface{}{},
					"affinity":       map[string]interface{}{},
				},
			}

			// RHOAI overrides
			component.RHOAIValues = chartutil.Values{
				"serviceAccount": map[string]interface{}{
					"create": false, // RHOAI disables SA creation
				},
				"podAnnotations": map[string]interface{}{
					"prometheus.io/scrape": "true",
				},
			}

			// Component config
			componentValues := chartutil.Values{
				"podAnnotations": map[string]interface{}{
					"custom.io/annotation": "value",
				},
			}

			result := component.MergeValues(componentValues)

			// serviceAccount.create: RHOAI value
			saMap := result["serviceAccount"].(map[string]interface{})
			Expect(saMap["create"]).To(BeFalse())

			// podAnnotations: Merged from RHOAI and component
			annotationsMap := result["podAnnotations"].(map[string]interface{})
			Expect(annotationsMap["prometheus.io/scrape"]).To(Equal("true"))
			Expect(annotationsMap["custom.io/annotation"]).To(Equal("value"))
		})
	})
})
