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
	"log"

	"k8s.io/apimachinery/pkg/runtime/schema"
	componentsv1alpha1 "github.com/opendatahub-io/opendatahub-operator/v2/api/components/v1alpha1"
)

// init registers Langfuse component with the global registry
// This follows the pattern from contracts/registry-api.md
func init() {
	// Register Langfuse component
	// Operator will fail to start if chart loading fails (FR-008)
	config := ComponentConfig{
		ChartName:       "langfuse",
		ValuesGenerator: LangfuseValuesFromSpec,
		Watches: []schema.GroupVersionKind{
			// Watch Langfuse CRD itself
			{
				Group:   "components.platform.opendatahub.io",
				Version: "v1alpha1",
				Kind:    componentsv1alpha1.LangfuseKind,
			},
			// Watch Deployments created by Langfuse chart
			{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			},
			// Watch Services created by Langfuse chart
			{
				Group:   "",
				Version: "v1",
				Kind:    "Service",
			},
			// Watch ConfigMaps for Langfuse configuration
			{
				Group:   "",
				Version: "v1",
				Kind:    "ConfigMap",
			},
		},
	}

	if err := HelmManagedComponents.Register("langfuse", config); err != nil {
		// Fail fast on registration error (FR-008: fail-fast pattern)
		log.Fatalf("Failed to register Langfuse component: %v", err)
	}
}
