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

	componentsv1alpha1 "github.com/opendatahub-io/opendatahub-operator/v2/api/components/v1alpha1"
	"github.com/opendatahub-io/opendatahub-operator/v2/internal/controller/helmregistry"
)

var _ = Describe("LangfuseValuesFromSpec", func() {
	Context("T011: ValuesGenerator function tests", func() {
		It("should generate values from DSCLangfuse spec", func() {
			spec := &componentsv1alpha1.DSCLangfuse{
				LangfuseCommonSpec: componentsv1alpha1.LangfuseCommonSpec{
					Features: componentsv1alpha1.LangfuseFeatures{
						ExperimentalFeaturesEnabled: true,
						TracingEnabled:              false,
						StorageSize:                 "20Gi",
					},
				},
			}

			values, err := helmregistry.LangfuseValuesFromSpec(spec)
			Expect(err).ToNot(HaveOccurred())
			Expect(values).ToNot(BeNil())

			// Verify nested structure
			langfuseValues, ok := values["langfuse"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "langfuse key should exist and be a map")

			features, ok := langfuseValues["features"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "features should be a map")
			Expect(features["experimentalEnabled"]).To(Equal(true))
			Expect(features["tracingEnabled"]).To(Equal(false))

			persistence, ok := langfuseValues["persistence"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "persistence should be a map")
			Expect(persistence["size"]).To(Equal("20Gi"))
		})

		It("should return error for invalid spec type", func() {
			invalidSpec := "not a DSCLangfuse struct"
			values, err := helmregistry.LangfuseValuesFromSpec(invalidSpec)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected *componentsv1alpha1.DSCLangfuse"))
			Expect(values).To(BeNil())
		})

		It("should handle default values when features are not specified", func() {
			spec := &componentsv1alpha1.DSCLangfuse{
				LangfuseCommonSpec: componentsv1alpha1.LangfuseCommonSpec{
					Features: componentsv1alpha1.LangfuseFeatures{
						// All defaults from CRD
					},
				},
			}

			values, err := helmregistry.LangfuseValuesFromSpec(spec)
			Expect(err).ToNot(HaveOccurred())

			langfuseValues := values["langfuse"].(map[string]interface{})
			features := langfuseValues["features"].(map[string]interface{})

			// Verify defaults are set correctly
			Expect(features["experimentalEnabled"]).To(Equal(false))
			Expect(features["tracingEnabled"]).To(Equal(false))
		})

		It("should generate deterministic values for same input", func() {
			spec := &componentsv1alpha1.DSCLangfuse{
				LangfuseCommonSpec: componentsv1alpha1.LangfuseCommonSpec{
					Features: componentsv1alpha1.LangfuseFeatures{
						ExperimentalFeaturesEnabled: true,
						TracingEnabled:              true,
						StorageSize:                 "10Gi",
					},
				},
			}

			values1, err1 := helmregistry.LangfuseValuesFromSpec(spec)
			values2, err2 := helmregistry.LangfuseValuesFromSpec(spec)

			Expect(err1).ToNot(HaveOccurred())
			Expect(err2).ToNot(HaveOccurred())
			Expect(values1).To(Equal(values2), "Same input should produce identical values")
		})

		It("should handle storage size variations", func() {
			testCases := []string{"10Gi", "100Mi", "1Ti", "500M"}

			for _, size := range testCases {
				spec := &componentsv1alpha1.DSCLangfuse{
					LangfuseCommonSpec: componentsv1alpha1.LangfuseCommonSpec{
						Features: componentsv1alpha1.LangfuseFeatures{
							StorageSize: size,
						},
					},
				}

				values, err := helmregistry.LangfuseValuesFromSpec(spec)
				Expect(err).ToNot(HaveOccurred())

				langfuseValues := values["langfuse"].(map[string]interface{})
				persistence := langfuseValues["persistence"].(map[string]interface{})
				Expect(persistence["size"]).To(Equal(size))
			}
		})
	})
})
