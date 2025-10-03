# Research: Helm-Managed Component Registry

**Phase 0 Output** | **Date**: 2025-10-02

## Overview
Research findings for implementing Helm chart integration into OpenDataHub operator following ArgoCD-style template rendering approach.

## Key Technical Decisions

### 1. Helm Chart Loading Mechanism

**Decision**: Use `helm.sh/helm/v3/pkg/chart/loader.LoadArchive()` for runtime chart loading

**Rationale**:
- Official Helm v3 library function for loading chart archives
- Returns `*chart.Chart` with all metadata, templates, and files
- Supports both .tgz archives and unpacked chart directories
- Well-tested and maintained by Helm project
- Integrates with Helm's rendering engine

**Alternatives Considered**:
- Manual tar.gz extraction + template parsing: More complex, reimplements Helm logic
- Helm CLI invocation via exec: Process overhead, harder to test, less flexible
- Kustomize-only approach: Doesn't support Helm chart dependencies or upstream updates

**Implementation Notes**:
```go
import "helm.sh/helm/v3/pkg/chart/loader"

// Load from dependency in charts/ directory
chart, err := loader.LoadArchive(chartPath)
if err != nil {
    return fmt.Errorf("chart load failed: %w", err)
}
```

### 2. Template Rendering Approach

**Decision**: ArgoCD-style rendering using only `helm.sh/helm/v3/pkg/engine.Engine.Render()`

**Rationale**:
- ArgoCD pattern proven for GitOps declarative management
- Avoids Helm release state management (no Tiller/release secrets)
- Renders templates to manifests without lifecycle hooks
- Simpler reconciliation model (operator manages resources directly)
- No Helm test execution or hook processing overhead

**What's Excluded** (per clarifications):
- Helm hooks (pre-install, post-install, pre-delete, etc.)
- Helm tests (test connection, etc.)
- Release management (helm install/upgrade/rollback)
- Chart dependencies resolution (handled by Chart.yaml dependencies)

**Implementation Pattern**:
```go
import (
    "helm.sh/helm/v3/pkg/engine"
    "helm.sh/helm/v3/pkg/chartutil"
)

// Render templates to Kubernetes manifests
renderer := engine.Engine{}
values := chartutil.Values{...}
manifests, err := renderer.Render(chart, values)
// manifests is map[string]string (file path -> rendered YAML)
```

### 3. Value Merging Strategy

**Decision**: Use `helm.sh/helm/v3/pkg/chartutil.CoalesceValues()` with precedence: Component Config > RHOAI defaults > Chart defaults

**Rationale**:
- Matches clarified requirement: "Component values override RHOAI values (user config wins)"
- Helm's CoalesceValues handles deep merging of nested maps
- Preserves chart default values when not overridden
- RHOAI values.rhoai.yaml provides platform-specific overrides
- Component-specific configuration has final say

**Merge Order**:
1. Load chart default values (values.yaml from chart)
2. Load RHOAI overrides (values.rhoai.yaml if present in chart files)
3. Apply component-specific configuration from DataScienceCluster CR
4. Coalesce: `chartutil.CoalesceValues(chart, componentValues)`

**Implementation Pattern**:
```go
// Start with chart defaults
values := chart.Values

// Merge RHOAI overrides
for _, file := range chart.Files {
    if file.Name == "values.rhoai.yaml" {
        rhoaiValues := chartutil.Values{}
        yaml.Unmarshal(file.Data, &rhoaiValues)
        values = chartutil.CoalesceTables(rhoaiValues, values)
    }
}

// Component config wins
values = chartutil.CoalesceTables(componentConfig, values)
```

### 4. Dynamic Watch Registration

**Decision**: Use controller-runtime's `controller.Watch()` with predicate-based filtering and CRD discovery

**Rationale**:
- controller-runtime supports adding watches after controller start
- Predicate filters reduce unnecessary reconciliations
- Can watch for CRD creation events to register deferred watches
- Aligns with existing operator watch patterns

**CRD Discovery Pattern**:
```go
// Watch for CRD creations
err := c.Watch(
    source.Kind(cache, &apiextensionsv1.CustomResourceDefinition{}),
    handler.EnqueueRequestsFromMapFunc(r.mapCRDToComponents),
    predicate.Funcs{
        CreateFunc: func(e event.CreateEvent) bool {
            // Check if newly created CRD matches component watches
            return r.isWatchedGVK(e.Object)
        },
    },
)
```

### 5. Component Registration Pattern

**Decision**: Use init() functions with registry pattern, similar to database/sql drivers

**Rationale**:
- Go init() guarantees execution before main()
- Registry pattern allows discovery without explicit imports
- Matches existing Kubernetes scheme registration patterns
- Component packages self-register on import

**Pattern**:
```go
// In component package
func init() {
    HelmManagedComponents.Register("langfuse", ComponentConfig{
        ChartName: "langfuse",
        ValuesGenerator: LangfuseValues,
        Watches: []schema.GroupVersionKind{...},
    })
}
```

### 6. Chart Dependency Management

**Decision**: Use Helm Chart.yaml dependencies field with `helm dependency update` in operator build

**Rationale**:
- Native Helm dependency mechanism
- Charts stored in charts/ directory as .tgz archives
- Dependencies versioned and locked (Chart.lock)
- Build-time dependency resolution (not runtime)
- Operator image contains all chart dependencies

**Chart.yaml Structure**:
```yaml
apiVersion: v2
name: opendatahub-operator
version: 2.x.x
dependencies:
  - name: langfuse
    version: "1.0.0"
    repository: "https://langfuse.github.io/langfuse-helm"
    condition: components.langfuse.enabled
```

## Integration Points

### Existing Operator Components

**Reconciliation Integration**:
- Helm rendering happens in reconciliation loop
- Uses existing `pkg/controller/actions` pattern
- New action: `render/helm/action_render_helm.go`
- Follows existing deploy action patterns

**Component Controller Pattern**:
- Extends existing component controller structure in `internal/controller/components/`
- Each Helm component gets controller in `internal/controller/components.platform.opendatahub.io/`
- Reuses ManagementState, DevFlags, and status condition logic

**CRD Extension**:
- Adds fields to DataScienceCluster.spec.components
- New API group: `components.platform.opendatahub.io/v1alpha1`
- Follows existing kubebuilder marker patterns
- Uses existing RBAC generation

### Testing Strategy

**Unit Tests** (Ginkgo + Gomega):
- Chart loading with valid/invalid archives
- Value merging with precedence validation
- Template rendering with various value combinations
- Watch registration for existing/non-existent CRDs

**Integration Tests** (envtest):
- Component controller reconciliation
- Dynamic watch activation on CRD creation
- Admission webhook validation
- Status condition updates

**E2E Tests**:
- Full component lifecycle (create, update, delete)
- RHOAI value override application
- Multiple concurrent Helm components
- Chart upgrade scenarios

## Performance Considerations

**Chart Loading**:
- Charts loaded once at operator startup from charts/ directory
- Metadata cached in memory (HelmManagedComponentRegistry)
- LoadArchive reads from filesystem (charts packaged in container image)
- Expected <5s total for all component charts

**Template Rendering**:
- Rendering happens per reconciliation when component config changes
- Helm engine rendering is CPU-bound template processing
- Expect <1s for typical component chart (10-20 templates)
- Results cached between reconciliations if values unchanged

**Watch Overhead**:
- Dynamic watches only added for declared GVKs
- Predicate filters reduce event processing
- Informer caching shared across components
- Watch count: O(components × resource types per component)

## Security Considerations

**Chart Validation**:
- Charts included at build time (supply chain controlled)
- Chart.lock ensures dependency integrity
- No runtime chart downloads (all in operator image)

**Value Injection**:
- Component config validated at admission time (FR-011)
- Type safety via OpenAPI schema validation
- No arbitrary value injection from user input

**RBAC**:
- Component-specific RBAC generated from kubebuilder markers
- Least privilege for each component's resource types
- Dynamic watch registration doesn't grant new permissions

## Open Questions Resolved

All technical unknowns from spec clarification session are resolved:
1. ✅ Chart loading: LoadArchive function
2. ✅ Helm features: Template rendering only (ArgoCD-style)
3. ✅ Reusability: Registry pattern with init() self-registration
4. ✅ Component addition: Chart.yaml dependencies
5. ✅ Chart load failure: Fail operator startup
6. ✅ Template render failure: Fail reconciliation, retry with backoff
7. ✅ Value precedence: Component > RHOAI > Chart defaults
8. ✅ Type mismatches: Admission validation
9. ✅ Missing CRD watches: Dynamic registration on CRD creation

## References

**Helm Documentation**:
- Helm Go SDK: https://helm.sh/docs/topics/advanced/
- Chart Structure: https://helm.sh/docs/topics/charts/
- Template Guide: https://helm.sh/docs/chart_template_guide/

**ArgoCD Helm Integration**:
- ArgoCD Helm Support: https://argo-cd.readthedocs.io/en/stable/user-guide/helm/
- Template-only rendering pattern reference

**OpenDataHub Operator**:
- Existing component patterns in internal/controller/components/
- Deploy action patterns in pkg/controller/actions/deploy/
- Manifest rendering in pkg/controller/actions/render/

**controller-runtime**:
- Dynamic watches: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/controller
- Predicate filtering: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/predicate
