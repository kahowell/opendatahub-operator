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
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opendatahub-io/opendatahub-operator/v2/internal/controller/helmregistry"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Dynamic Watch Registration Integration", func() {
	var component *helmregistry.HelmManagedComponent
	var mockController *helmregistry.MockController
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		component = &helmregistry.HelmManagedComponent{
			ChartName: "test-component",
			Watches: []schema.GroupVersionKind{
				{Group: "test.io", Version: "v1alpha1", Kind: "TestResource"},
			},
		}
		mockController = helmregistry.NewMockController()
	})

	Context("Component registers with watch for non-existent CRD", func() {
		It("should mark watch as pending initially", func() {
			handler := helmregistry.NewMockEventHandler()

			err := component.AddWatches(mockController, handler)
			Expect(err).To(BeNil())

			// Watch should be pending
			hasPending := component.HasPendingWatches()
			Expect(hasPending).To(BeTrue(),
				"Watch should be pending for non-existent CRD")
		})
	})

	Context("Create CRD matching watch GVK", func() {
		It("should detect matching CRD creation", func() {
			handler := helmregistry.NewMockEventHandler()

			err := component.AddWatches(mockController, handler)
			Expect(err).To(BeNil())

			// Create matching CRD
			crd := &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testresources.test.io",
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "test.io",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind:   "TestResource",
						Plural: "testresources",
					},
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{
							Name:   "v1alpha1",
							Served: true,
							Storage: true,
						},
					},
					Scope: apiextensionsv1.NamespaceScoped,
				},
			}

			// Component should recognize this CRD
			matches := component.HasPendingWatchForCRD(crd)
			Expect(matches).To(BeTrue(),
				"Component should recognize matching CRD")
		})
	})

	Context("Watch automatically registers", func() {
		It("should register watch when CRD becomes available", func() {
			handler := helmregistry.NewMockEventHandler()

			err := component.AddWatches(mockController, handler)
			Expect(err).To(BeNil())

			initialCount := mockController.WatchCount()

			// Create CRD
			crd := &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testresources.test.io",
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "test.io",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "TestResource",
					},
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true, Storage: true},
					},
				},
			}

			// Trigger watch registration
			err = component.RegisterPendingWatch(crd, mockController, handler)
			Expect(err).To(BeNil())

			// Verify watch count increased
			Expect(mockController.WatchCount()).To(BeNumerically(">", initialCount),
				"Watch should be registered after CRD creation")
		})
	})

	Context("Reconciliation triggered", func() {
		It("should enqueue reconcile request when CRD created", func() {
			handler := helmregistry.NewMockEventHandler()

			err := component.AddWatches(mockController, handler)
			Expect(err).To(BeNil())

			crd := &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testresources.test.io",
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "test.io",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind: "TestResource",
					},
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{Name: "v1alpha1", Served: true},
					},
				},
			}

			// Map CRD to reconcile requests
			requests := component.MapCRDToComponent(ctx, crd)
			Expect(requests).NotTo(BeNil())
			Expect(len(requests)).To(BeNumerically(">", 0),
				"Should return reconcile requests")
		})
	})
})

var _ = Describe("Watch Lifecycle with envtest", func() {
	var component *helmregistry.HelmManagedComponent
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
		component = &helmregistry.HelmManagedComponent{
			ChartName: "envtest-component",
			Watches: []schema.GroupVersionKind{
				{Group: "example.com", Version: "v1", Kind: "ExampleCR"},
			},
		}
	})

	Context("CRD lifecycle in real cluster", func() {
		It("should handle CRD creation and deletion", func() {
			// Create CRD
			crd := &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: "examplecrs.example.com",
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Group: "example.com",
					Names: apiextensionsv1.CustomResourceDefinitionNames{
						Kind:     "ExampleCR",
						Plural:   "examplecrs",
						Singular: "examplecr",
					},
					Scope: apiextensionsv1.NamespaceScoped,
					Versions: []apiextensionsv1.CustomResourceDefinitionVersion{
						{
							Name:    "v1",
							Served:  true,
							Storage: true,
							Schema: &apiextensionsv1.CustomResourceValidation{
								OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
									Type: "object",
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(ctx, crd)
			Expect(err).To(BeNil(), "Should create CRD")

			// Wait for CRD to be established
			Eventually(func() bool {
				var created apiextensionsv1.CustomResourceDefinition
				err := k8sClient.Get(ctx, client.ObjectKeyFromObject(crd), &created)
				if err != nil {
					return false
				}
				for _, cond := range created.Status.Conditions {
					if cond.Type == apiextensionsv1.Established && cond.Status == apiextensionsv1.ConditionTrue {
						return true
					}
				}
				return false
			}, 10*time.Second, 100*time.Millisecond).Should(BeTrue(),
				"CRD should become established")

			// Component should detect it
			matches := component.HasPendingWatchForCRD(crd)
			Expect(matches).To(BeTrue())

			// Cleanup
			err = k8sClient.Delete(ctx, crd)
			Expect(err).To(BeNil())
		})
	})
})
