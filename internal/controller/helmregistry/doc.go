// Package helmregistry provides a registry for Helm-managed components
// that integrates with the OpenDataHub operator.
//
// This package implements the Helm-managed component registry system for the OpenDataHub operator,
// enabling component developers to add new Helm-based components to DataScienceCluster with
// minimal code changes. The registry loads Helm charts as dependencies, renders templates using
// the Helm SDK, merges RHOAI-specific value overrides with component configuration, and
// automatically creates controllers with dynamic resource watches.
//
// The integration follows an ArgoCD-style approach supporting only template rendering
// without advanced Helm features like hooks or tests.
package helmregistry

import (
	// Import Helm SDK packages to ensure dependencies are maintained
	_ "helm.sh/helm/v3/pkg/chart"
	_ "helm.sh/helm/v3/pkg/chart/loader"
	_ "helm.sh/helm/v3/pkg/chartutil"
	_ "helm.sh/helm/v3/pkg/engine"
)
