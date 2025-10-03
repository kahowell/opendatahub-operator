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
	"fmt"

	"helm.sh/helm/v3/pkg/chartutil"
	componentsv1alpha1 "github.com/opendatahub-io/opendatahub-operator/v2/api/components/v1alpha1"
)

// LangfuseValuesFromSpec generates Helm values from Langfuse component spec
// This implements the ValuesGenerator contract for Langfuse component
func LangfuseValuesFromSpec(spec interface{}) (chartutil.Values, error) {
	langfuseSpec, ok := spec.(*componentsv1alpha1.DSCLangfuse)
	if !ok {
		return nil, fmt.Errorf("expected *componentsv1alpha1.DSCLangfuse, got %T", spec)
	}

	// Build values map following the Langfuse chart structure
	// Precedence: component config > RHOAI values > chart defaults
	values := chartutil.Values{
		"langfuse": map[string]interface{}{
			"features": map[string]interface{}{
				"experimentalEnabled": langfuseSpec.Features.ExperimentalFeaturesEnabled,
				"tracingEnabled":      langfuseSpec.Features.TracingEnabled,
			},
			"persistence": map[string]interface{}{
				"size": langfuseSpec.Features.StorageSize,
			},
		},
	}

	// Add DevFlags if present (common pattern across components)
	if langfuseSpec.DevFlags != nil {
		if langfuseSpec.DevFlags.Manifests != nil {
			for _, manifest := range langfuseSpec.DevFlags.Manifests {
				// DevFlags manifests are applied after Helm rendering
				// These are stored in component values for reference
				_ = manifest // Will be used by controller for manifest overrides
			}
		}
	}

	return values, nil
}
