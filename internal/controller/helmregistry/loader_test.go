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
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opendatahub-io/opendatahub-operator/v2/internal/controller/helmregistry"
)

var _ = Describe("Chart Loading via LoadArchive", func() {
	var component *helmregistry.HelmManagedComponent

	BeforeEach(func() {
		component = &helmregistry.HelmManagedComponent{
			ChartName: "test-chart",
		}
	})

	Context("Load valid .tgz chart archive", func() {
		It("should successfully load chart from .tgz file", func() {
			chartPath := filepath.Join("testdata", "charts", "test-chart-1.0.0.tgz")

			err := component.LoadChart(chartPath)
			Expect(err).To(BeNil(), "Should load valid chart archive")
			Expect(component.Chart).NotTo(BeNil())
			Expect(component.Chart.Metadata.Name).To(Equal("test-chart"))
		})
	})

	Context("Load unpacked chart directory", func() {
		It("should load chart from unpacked directory structure", func() {
			chartPath := filepath.Join("testdata", "charts", "test-chart")

			err := component.LoadChart(chartPath)
			Expect(err).To(BeNil(), "Should load unpacked chart")
			Expect(component.Chart).NotTo(BeNil())
		})
	})

	Context("Error on missing chart file", func() {
		It("should return error when chart file doesn't exist", func() {
			chartPath := filepath.Join("testdata", "charts", "nonexistent-chart.tgz")

			err := component.LoadChart(chartPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("chart"), "Error should mention chart")
		})
	})

	Context("Error on invalid chart structure", func() {
		It("should return error when chart structure is invalid", func() {
			chartPath := filepath.Join("testdata", "charts", "invalid-chart.tgz")

			err := component.LoadChart(chartPath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid"), "Error should mention invalid structure")
		})
	})

	Context("Extract RHOAI values from values.rhoai.yaml", func() {
		It("should extract RHOAI values if present in chart files", func() {
			chartPath := filepath.Join("testdata", "charts", "chart-with-rhoai-values.tgz")

			err := component.LoadChart(chartPath)
			Expect(err).To(BeNil())
			Expect(component.RHOAIValues).NotTo(BeNil(), "Should extract RHOAI values")
			Expect(component.RHOAIValues).To(HaveKey("replicas"))
		})

		It("should not error when values.rhoai.yaml is absent", func() {
			chartPath := filepath.Join("testdata", "charts", "test-chart-1.0.0.tgz")

			err := component.LoadChart(chartPath)
			Expect(err).To(BeNil())
			// RHOAIValues can be nil or empty when not present
		})
	})

	Context("Chart metadata validation", func() {
		It("should validate chart has required metadata fields", func() {
			chartPath := filepath.Join("testdata", "charts", "test-chart-1.0.0.tgz")

			err := component.LoadChart(chartPath)
			Expect(err).To(BeNil())

			Expect(component.Chart.Metadata).NotTo(BeNil())
			Expect(component.Chart.Metadata.Name).NotTo(BeEmpty())
			Expect(component.Chart.Metadata.Version).NotTo(BeEmpty())
		})
	})
})
