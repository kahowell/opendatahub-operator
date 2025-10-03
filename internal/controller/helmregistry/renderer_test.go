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

var _ = Describe("Template Rendering with Helm Engine", func() {
	var component *helmregistry.HelmManagedComponent

	BeforeEach(func() {
		component = &helmregistry.HelmManagedComponent{
			ChartName: "test-chart",
		}
	})

	Context("Render templates to map[string]string", func() {
		It("should render all templates to manifest map", func() {
			values := chartutil.Values{
				"replicas": 2,
				"image": map[string]interface{}{
					"repository": "test/app",
					"tag":        "v1.0.0",
				},
			}

			manifests, err := component.RenderTemplates(values)
			Expect(err).To(BeNil(), "Rendering should succeed")
			Expect(manifests).NotTo(BeNil())
			Expect(manifests).To(BeAssignableToTypeOf(map[string]string{}))
			Expect(len(manifests)).To(BeNumerically(">", 0), "Should have rendered manifests")
		})
	})

	Context("Exclude NOTES.txt, tests/, hooks/", func() {
		It("should filter out NOTES.txt from rendered output", func() {
			values := chartutil.Values{}

			manifests, err := component.RenderTemplates(values)
			Expect(err).To(BeNil())

			for filename := range manifests {
				Expect(filename).NotTo(ContainSubstring("NOTES.txt"),
					"NOTES.txt should be excluded")
			}
		})

		It("should filter out test files from rendered output", func() {
			values := chartutil.Values{}

			manifests, err := component.RenderTemplates(values)
			Expect(err).To(BeNil())

			for filename := range manifests {
				Expect(filename).NotTo(ContainSubstring("/tests/"),
					"Test files should be excluded")
			}
		})

		It("should filter out hooks from rendered output", func() {
			values := chartutil.Values{}

			manifests, err := component.RenderTemplates(values)
			Expect(err).To(BeNil())

			for filename := range manifests {
				Expect(filename).NotTo(ContainSubstring("/hooks/"),
					"Hook files should be excluded")
			}
		})
	})

	Context("Template syntax errors return ErrTemplateRendering", func() {
		It("should return error when template has syntax error", func() {
			// Component with invalid template
			component.Chart = &helmregistry.MockChart{
				Templates: []*helmregistry.MockTemplate{
					{
						Name: "deployment.yaml",
						Data: "{{ .Values.invalid syntax }}",
					},
				},
			}

			values := chartutil.Values{}

			manifests, err := component.RenderTemplates(values)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("render"),
				"Error should mention rendering failure")
			Expect(manifests).To(BeNil())
		})

		It("should return error when accessing undefined value in template", func() {
			component.Chart = &helmregistry.MockChart{
				Templates: []*helmregistry.MockTemplate{
					{
						Name: "service.yaml",
						Data: "{{ .Values.nonexistent.field }}",
					},
				},
			}

			values := chartutil.Values{}

			manifests, err := component.RenderTemplates(values)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Rendered output is valid YAML", func() {
		It("should produce valid YAML in each manifest", func() {
			values := chartutil.Values{
				"service": map[string]interface{}{
					"type": "ClusterIP",
					"port": 8080,
				},
			}

			manifests, err := component.RenderTemplates(values)
			Expect(err).To(BeNil())

			for filename, content := range manifests {
				// Verify it's parseable as YAML
				var parsed map[string]interface{}
				err := yaml.Unmarshal([]byte(content), &parsed)
				Expect(err).To(BeNil(), "Manifest %s should be valid YAML", filename)
			}
		})

		It("should produce Kubernetes-compatible manifests", func() {
			values := chartutil.Values{}

			manifests, err := component.RenderTemplates(values)
			Expect(err).To(BeNil())

			for filename, content := range manifests {
				// Basic check for Kubernetes resource structure
				var resource map[string]interface{}
				err := yaml.Unmarshal([]byte(content), &resource)
				Expect(err).To(BeNil())

				// Should have apiVersion and kind at minimum
				if len(resource) > 0 {
					_, hasAPIVersion := resource["apiVersion"]
					_, hasKind := resource["kind"]
					Expect(hasAPIVersion || hasKind).To(BeTrue(),
						"Manifest %s should have Kubernetes resource fields", filename)
				}
			}
		})
	})

	Context("Multiple template files rendered correctly", func() {
		It("should render all templates in chart", func() {
			component.Chart = &helmregistry.MockChart{
				Templates: []*helmregistry.MockTemplate{
					{Name: "deployment.yaml", Data: "apiVersion: apps/v1\nkind: Deployment"},
					{Name: "service.yaml", Data: "apiVersion: v1\nkind: Service"},
					{Name: "configmap.yaml", Data: "apiVersion: v1\nkind: ConfigMap"},
				},
			}

			values := chartutil.Values{}

			manifests, err := component.RenderTemplates(values)
			Expect(err).To(BeNil())
			Expect(len(manifests)).To(Equal(3), "Should render all templates")
		})
	})

	Context("Empty values produce chart defaults", func() {
		It("should render successfully with empty values", func() {
			values := chartutil.Values{}

			manifests, err := component.RenderTemplates(values)
			Expect(err).To(BeNil())
			Expect(manifests).NotTo(BeNil())
		})
	})
})
