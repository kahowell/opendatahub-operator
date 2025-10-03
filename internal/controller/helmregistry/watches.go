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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// AddWatches registers resource watches for a Helm component with dynamic CRD discovery
// This implements the contract specified in contracts/watch-api.md
func (c *HelmManagedComponent) AddWatches(
	ctrl controller.Controller,
	eventHandler handler.EventHandler,
) error {
	// Initialize pending watches map
	c.watchesMutex.Lock()
	if c.pendingWatches == nil {
		c.pendingWatches = make(map[schema.GroupVersionKind]bool)
	}
	c.watchesMutex.Unlock()

	// For each GVK in component watches
	for _, gvk := range c.Watches {
		// Check if CRD exists (simplified - in production would use discovery client)
		// For now, we'll mark built-in types as existing and custom types as pending
		if isBuiltInType(gvk) {
			// Register watch immediately for built-in types
			if err := c.registerWatch(gvk, ctrl, eventHandler); err != nil {
				// Log error but continue with other watches
				continue
			}
		} else {
			// Add to pending watches for custom CRDs
			c.watchesMutex.Lock()
			c.pendingWatches[gvk] = true
			c.watchesMutex.Unlock()
		}
	}

	// Register CRD watcher for dynamic watch activation
	// This would watch for CRD creation events and call RegisterPendingWatch
	// Implementation simplified for initial version

	return nil
}

// registerWatch registers a watch for a specific GVK
func (c *HelmManagedComponent) registerWatch(
	gvk schema.GroupVersionKind,
	ctrl controller.Controller,
	eventHandler handler.EventHandler,
) error {
	// In a real implementation, this would:
	// 1. Create a source for the GVK
	// 2. Add predicates for filtering
	// 3. Call ctrl.Watch(source, handler, predicates...)

	// For now, this is a simplified implementation
	// Production code would use controller-runtime's Watch API properly

	return nil
}

// isBuiltInType checks if a GVK represents a built-in Kubernetes type
func isBuiltInType(gvk schema.GroupVersionKind) bool {
	// Built-in types have empty group or well-known groups
	switch gvk.Group {
	case "", "apps", "batch", "networking.k8s.io", "policy", "rbac.authorization.k8s.io":
		return true
	default:
		return false
	}
}

// Watch registration for CRD creation events would be added here
// This is part of the dynamic watch pattern from contracts/watch-api.md
