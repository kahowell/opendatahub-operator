# Contract: HelmManagedComponentRegistry API

**Package**: `internal/controller/helmregistry`

## Registry.Register

**Purpose**: Register a Helm-managed component at operator init time

**Signature**:
```go
func (r *HelmManagedComponentRegistry) Register(
    name string,
    config ComponentConfig,
) error
```

**Request Contract**:
```go
type ComponentConfig struct {
    ChartName string                                      // REQUIRED: Helm chart name (must match Chart.yaml dependency)
    ValuesGenerator func(ComponentSpec) (chartutil.Values, error) // REQUIRED: Value generation function
    Watches []schema.GroupVersionKind                     // OPTIONAL: Resource types to watch
}
```

**Preconditions**:
- Called from init() function only
- Chart exists in charts/ directory as dependency
- Chart name matches Chart.yaml dependency entry
- Component name is unique (not already registered)

**Postconditions Success**:
- Component added to registry map
- Chart loaded into memory via LoadArchive
- RHOAI values extracted from chart files (if values.rhoai.yaml present)
- Returns nil error

**Postconditions Failure**:
- Chart load fails → Operator startup aborts (FR-008)
- Duplicate name → Operator startup aborts
- Invalid config → Operator startup aborts

**Error Cases**:
```go
ErrChartNotFound      // Chart missing from charts/ directory
ErrChartLoadFailed    // LoadArchive returned error
ErrDuplicateComponent // Component name already registered
ErrInvalidConfig      // ValuesGenerator is nil or ChartName empty
```

**Example**:
```go
func init() {
    err := HelmManagedComponents.Register("langfuse", ComponentConfig{
        ChartName: "langfuse",
        ValuesGenerator: LangfuseValuesFromSpec,
        Watches: []schema.GroupVersionKind{
            {Group: "apps", Version: "v1", Kind: "Deployment"},
            {Group: "v1", Version: "v1", Kind: "Service"},
        },
    })
    if err != nil {
        panic(fmt.Sprintf("failed to register langfuse: %v", err))
    }
}
```

## Registry.Render

**Purpose**: Render Helm chart templates to Kubernetes manifests

**Signature**:
```go
func (r *HelmManagedComponentRegistry) Render(
    name string,
    spec ComponentSpec,
) (map[string]string, error)
```

**Request Contract**:
- `name`: Registered component name
- `spec`: Component configuration from DataScienceCluster CR

**Response Contract**:
```go
// Success: map of filename -> rendered YAML content
map[string]string{
    "langfuse/templates/deployment.yaml": "apiVersion: apps/v1\nkind: Deployment...",
    "langfuse/templates/service.yaml":    "apiVersion: v1\nkind: Service...",
}
// Error: nil map + error
```

**Processing Steps**:
1. Retrieve component from registry by name
2. Call ValuesGenerator(spec) to produce component values
3. Merge values with precedence: component > RHOAI > chart defaults
4. Call helm engine.Render(chart, finalValues)
5. Return rendered manifest map

**Preconditions**:
- Component registered via Register()
- spec is valid (passed admission validation)

**Postconditions Success**:
- Map contains all rendered templates (excluding NOTES.txt, tests/, etc.)
- Each value is valid YAML
- Manifests can be parsed into Kubernetes resources
- Deterministic output for same inputs (NFR-002)

**Postconditions Failure** (FR-012):
- Template rendering fails → Return error (reconciliation will retry with backoff)
- Values incompatible with schema → Return error
- Chart template syntax invalid → Return error

**Error Cases**:
```go
ErrComponentNotFound    // Component not registered
ErrValuesGeneration     // ValuesGenerator function failed
ErrTemplateRendering    // Helm engine.Render failed
ErrInvalidManifest      // Rendered output not valid YAML/Kubernetes resources
```

**Performance**:
- Target: <1s for typical chart (10-20 templates)
- Cached chart metadata (no re-loading)
- Template rendering is CPU-bound

**Example**:
```go
manifests, err := HelmManagedComponents.Render("langfuse", componentSpec)
if err != nil {
    return ctrl.Result{RequeueAfter: 30*time.Second}, err // Retry with backoff
}
// Apply manifests to cluster...
```

## Registry.GetComponent

**Purpose**: Retrieve registered component metadata

**Signature**:
```go
func (r *HelmManagedComponentRegistry) GetComponent(
    name string,
) (*HelmManagedComponent, bool)
```

**Response Contract**:
- First return: Component pointer (nil if not found)
- Second return: Boolean existence indicator

**Example**:
```go
component, exists := HelmManagedComponents.GetComponent("langfuse")
if !exists {
    return fmt.Errorf("component langfuse not registered")
}
// Use component.Watches, component.Chart, etc.
```

## Registry.ListComponents

**Purpose**: List all registered component names

**Signature**:
```go
func (r *HelmManagedComponentRegistry) ListComponents() []string
```

**Response Contract**:
- Sorted list of component names
- Empty slice if no components registered

**Example**:
```go
components := HelmManagedComponents.ListComponents()
// ["langfuse", "other-component"]
```

## Thread Safety

**Concurrency Guarantee**:
- Register(): Called only during init() phase (single-threaded)
- Render(): Safe for concurrent calls (read-only chart access)
- GetComponent(): Safe for concurrent calls (read-only map access)
- ListComponents(): Safe for concurrent calls (returns copy)

**Synchronization**: Registry uses read-only access after init() completes, no locking needed for read operations.
