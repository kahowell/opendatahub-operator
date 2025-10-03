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
	"helm.sh/helm/v3/pkg/chartutil"
)

// MergeValues merges component configuration with RHOAI and chart default values
// Implements precedence: component > RHOAI > chart defaults (contracts/values-api.md)
func (c *HelmManagedComponent) MergeValues(componentValues chartutil.Values) chartutil.Values {
	if c.Chart == nil {
		return componentValues
	}

	// Step 1: Start with chart default values
	result := c.Chart.Values

	// Step 2: Merge RHOAI overrides (RHOAI wins over chart defaults)
	if c.RHOAIValues != nil && len(c.RHOAIValues) > 0 {
		result = chartutil.CoalesceTables(c.RHOAIValues, result)
	}

	// Step 3: Merge component config (component wins over all)
	if componentValues != nil && len(componentValues) > 0 {
		result = chartutil.CoalesceTables(componentValues, result)
	}

	return result
}
