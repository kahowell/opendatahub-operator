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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("AddWatches() Contract", func() {
	var component *helmregistry.HelmManagedComponent
	var mockController *helmregistry.MockController
	var mockHandler *helmregistry.MockEventHandler

	BeforeEach(func() {
		component = &helmregistry.HelmManagedComponent{
			ChartName: "test-chart",
			Watches: []schema.GroupVersionKind{
				{Group: "apps", Version: "v1", Kind: "Deployment"},
				{Group: "", Version: "v1", Kind: "Service"},
				{Group: "custom.io", Version: "v1alpha1", Kind: "CustomResource"},
			},
		}
		mockController = helmregistry.NewMockController()
		mockHandler = helmregistry.NewMockEventHandler()
	})

	Context("Immediate watch registration for existing CRD", func() {
		It("should register watch immediately when CRD exists", func() {
			// Simulate existing CRDs for Deployment and Service (built-in types)
			err := component.AddWatches(mockController, mockHandler)
			Expect(err).To(BeNil())

			// Verify watches were registered for built-in types
			Expect(mockController.WatchCount()).To(BeNumerically(">=", 2),
				"Should register watches for Deployment and Service")
		})
	})

	Context("Deferred watch for missing CRD", func() {
		It("should defer watch registration for non-existent CRD", func() {
			err := component.AddWatches(mockController, mockHandler)
			Expect(err).To(BeNil())

			// CustomResource CRD doesn't exist, should be in pending watches
			hasPending := component.HasPendingWatches()
			Expect(hasPending).To(BeTrue(), "Should have pending watches for missing CRD")
		})
	})

	Context("CRD creation triggers pending watch", func() {
		It("should register pending watch when matching CRD is created", func() {
			// First, add watches with missing CRD
			err := component.AddWatches(mockController, mockHandler)
			Expect(err).To(BeNil())

			initialCount := mockController.WatchCount()

			// Simulate CRD creation
			crd := &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "customresources.custom.io",
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "custom.io",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "CustomResource",
					},
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{
							Name:   "v1alpha1",
							Served: true,
						},
					},
				},
			}

			// Component should detect matching CRD and register watch
			matches := component.HasPendingWatchForCRD(crd)
			Expect(matches).To(BeTrue(), "Should match pending watch")

			// Trigger watch registration
			err = component.RegisterPendingWatch(crd, mockController, mockHandler)
			Expect(err).To(BeNil())

			// Verify watch was added
			Expect(mockController.WatchCount()).To(BeNumerically(">", initialCount),
				"Should have registered the pending watch")
		})
	})

	Context("Predicate filtering reduces reconciliations", func() {
		It("should use predicate to filter watch events", func() {
			err := component.AddWatches(mockController, mockHandler)
			Expect(err).To(BeNil())

			// Verify predicates were set on watches
			predicates := mockController.GetPredicates()
			Expect(len(predicates)).To(BeNumerically(">", 0),
				"Watches should have predicates for filtering")
		})
	})
})

var _ = Describe("hasPendingWatchForCRD Contract", func() {
	var component *helmregistry.HelmManagedComponent

	BeforeEach(func() {
		component = &helmregistry.HelmManagedComponent{
			Watches: []schema.GroupVersionKind{
				{Group: "custom.io", Version: "v1alpha1", Kind: "CustomResource"},
			},
		}
	})

	Context("CRD matches pending watch", func() {
		It("should return true when CRD matches unregistered watch", func() {
			crd := &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "custom.io",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "CustomResource",
					},
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true},
					},
				},
			}

			matches := component.HasPendingWatchForCRD(crd)
			Expect(matches).To(BeTrue(), "Should match pending watch GVK")
		})
	})

	Context("CRD does not match any watch", func() {
		It("should return false when CRD doesn't match watches", func() {
			crd := &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "other.io",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "OtherResource",
					},
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1", Served: true},
					},
				},
			}

			matches := component.HasPendingWatchForCRD(crd)
			Expect(matches).To(BeFalse(), "Should not match different GVK")
		})
	})

	Context("Watch already registered", func() {
		It("should return false when watch is already registered", func() {
			crd := &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "custom.io",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "CustomResource",
					},
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true},
					},
				},
			}

			// Mark watch as registered
			component.MarkWatchRegistered(schema.GroupVersionKind{
				Group:   "custom.io",
				Version: "v1alpha1",
				Kind:    "CustomResource",
			})

			matches := component.HasPendingWatchForCRD(crd)
			Expect(matches).To(BeFalse(), "Should not match already registered watch")
		})
	})
})

var _ = Describe("mapCRDToComponent Contract", func() {
	var component *helmregistry.HelmManagedComponent

	BeforeEach(func() {
		component = &helmregistry.HelmManagedComponent{
			ChartName: "test-chart",
			Watches: []schema.GroupVersionKind{
				{Group: "custom.io", Version: "v1alpha1", Kind: "CustomResource"},
			},
		}
	})

	Context("CRD creation maps to component reconciliation", func() {
		It("should return reconcile request when matching CRD created", func() {
			crd := &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "customresources.custom.io",
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "custom.io",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "CustomResource",
					},
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true},
					},
				},
			}

			requests := component.MapCRDToComponent(ctx, crd)
			Expect(requests).NotTo(BeNil())
			Expect(len(requests)).To(BeNumerically(">", 0),
				"Should return reconcile request for component")
		})
	})

	Context("Non-matching CRD returns empty requests", func() {
		It("should return empty when CRD doesn't match watches", func() {
			crd := &apiextensionsv1.CustomResourceDefinition{
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "other.io",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "OtherResource",
					},
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1", Served: true},
					},
				},
			}

			requests := component.MapCRDToComponent(ctx, crd)
			Expect(len(requests)).To(Equal(0), "Should return empty for non-matching CRD")
		})
	})
})
