# Data Model: Helm-Managed Component Registry

**Phase 1 Output** | **Date**: 2025-10-02

## Core Entities

### 1. HelmManagedComponentRegistry

**Purpose**: Central registry storing all registered Helm-managed components and coordinating rendering

**Fields**:
- `components`: `map[string]*HelmManagedComponent` - Registry of components by name
- `chartCache`: `map[string]*chart.Chart` - Loaded chart metadata cache

**Operations**:
- `Register(name string, config ComponentConfig) error` - Register component at init time
- `Render(name string, values chartutil.Values) (map[string]string, error)` - Render component templates
- `GetComponent(name string) (*HelmManagedComponent, bool)` - Retrieve registered component
- `ListComponents() []string` - List all registered component names

**Lifecycle**: Singleton instance created at operator startup, populated during init()

**Validation Rules**:
- Component names must be unique
- Chart must load successfully during registration
- Component name must match Chart.yaml name

### 2. HelmManagedComponent

**Purpose**: Represents a single Helm-managed component configuration

**Fields**:
- `ChartName`: `string` - Name of Helm chart (matches Chart.yaml dependency)
- `Chart`: `*chart.Chart` - Loaded Helm chart metadata
- `ValuesGenerator`: `func(ComponentSpec) (chartutil.Values, error)` - Function to generate values from component spec
- `Watches`: `[]schema.GroupVersionKind` - Resource types to watch for this component
- `RHOAIValues`: `chartutil.Values` - RHOAI-specific value overrides (from values.rhoai.yaml)

**Operations**:
- `LoadChart(chartPath string) error` - Load chart using LoadArchive
- `GenerateValues(spec ComponentSpec) (chartutil.Values, error)` - Create values from component configuration
- `MergeValues(componentValues chartutil.Values) chartutil.Values` - Apply merge precedence
- `AddWatches(controller controller.Controller) error` - Register watches with controller

**Lifecycle**: Created during Register(), persists for operator lifetime

**Validation Rules**:
- ChartName must not be empty
- ValuesGenerator must not be nil
- Watches list can be empty (no additional watches)
- Chart must be valid Helm v3 chart

### 3. ComponentSpec (CRD Extension)

**Purpose**: Component configuration within DataScienceCluster

**Fields** (Example for Langfuse):
```go
type DSCLangfuse struct {
    // Common across all Helm components
    common.ManagementSpec `json:",inline"`  // Managed, Removed, Unmanaged states

    // Component-specific features
    Features LangfuseFeatures `json:"features,omitempty"`
}

type LangfuseFeatures struct {
    ExperimentalFeaturesEnabled bool `json:"experimentalFeaturesEnabled"`
    // Additional feature flags...
}
```

**Kubebuilder Markers**:
```go
// +kubebuilder:validation:Optional
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.managementState`
// +groupName=components.platform.opendatahub.io
```

**Validation Rules** (OpenAPI Schema):
- ManagementState: enum [Managed, Removed, Unmanaged]
- Feature flags: type validation (bool, string, int as declared)
- Required vs optional fields per component needs

**Admission Webhook Validation** (FR-011):
- Type compatibility check (reject string for bool field)
- Valid enum values
- Required field presence

### 4. ValuePathMapping

**Purpose**: Defines how component spec fields map to Helm chart value paths

**Representation**: Kubebuilder marker annotations

**Example**:
```go
type LangfuseFeatures struct {
    // +helmvalue:path=langfuse.features.experimentalEnabled
    ExperimentalFeaturesEnabled bool `json:"experimentalFeaturesEnabled"`

    // +helmvalue:path=langfuse.persistence.size
    StorageSize string `json:"storageSize,omitempty"`
}
```

**Validation Rules**:
- Path must use dot notation for nested values
- Path must correspond to actual chart values structure
- Type must be compatible (string->string, bool->bool, int->int)

### 5. ChartMetadata (from helm.sh/helm/v3/pkg/chart)

**Purpose**: Helm chart structure loaded by LoadArchive

**Key Fields** (from Helm library):
- `Metadata`: Chart.yaml content (name, version, dependencies)
- `Templates`: List of template files
- `Values`: Default values from values.yaml
- `Files`: Additional files (including values.rhoai.yaml)
- `Schema`: values.schema.json for validation

**Usage**: Read-only access to chart structure

### 6. ComponentStatus (extends existing pattern)

**Purpose**: Status conditions for Helm component reconciliation

**Status Conditions** (Kubernetes conventions):
```go
type ComponentStatus struct {
    // Existing fields...
    ManagementState ComponentState `json:"managementState"`

    // Standard Kubernetes conditions
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

**Condition Types**:
- `Ready`: Component successfully reconciled and resources deployed
- `Progressing`: Reconciliation in progress
- `Degraded`: Reconciliation failed (template rendering error, etc.)
- `Available`: Component resources are available in cluster

**Condition Reasons**:
- `ChartRenderError`: Template rendering failed (FR-012)
- `ValuesInvalid`: Component configuration validation failed
- `ResourcesApplied`: Manifests successfully applied
- `WatchRegistered`: Dynamic watch successfully added

### 7. WatchSpecification

**Purpose**: Describes resource types a component needs to watch

**Fields**:
```go
type WatchSpecification struct {
    GVK schema.GroupVersionKind
    Registered bool  // Whether watch is currently registered
    Predicate predicate.Predicate  // Optional filter
}
```

**Operations**:
- `RegisterWatch(controller controller.Controller) error` - Add watch to controller
- `IsRegistered() bool` - Check if watch active
- `MatchesCRD(crd *apiextensionsv1.CustomResourceDefinition) bool` - Check if CRD matches this watch

**Lifecycle**: Created during component registration, watches added dynamically

## Relationships

```
HelmManagedComponentRegistry (1)
  └─> HelmManagedComponent (n)
        ├─> chart.Chart (1) - loaded from charts/ directory
        ├─> ValuePathMapping (n) - via kubebuilder markers
        └─> WatchSpecification (n) - resource types to watch

DataScienceCluster (1)
  └─> ComponentSpec (n) - one per Helm component
        └─> Features (1) - component-specific configuration

ComponentController (n) - one per Helm component
  ├─> HelmManagedComponent (1) - from registry
  ├─> ComponentSpec (1) - from DataScienceCluster CR
  └─> WatchSpecification (n) - manages watches
```

## State Transitions

### Component Lifecycle

```
[Not Registered]
    └─> init() called
         └─> Register(name, config)
              └─> LoadChart(chartPath)
                   ├─ Success -> [Registered, Chart Loaded]
                   └─ Failure -> Operator startup fails (FR-008)

[Registered, Chart Loaded]
    └─> DataScienceCluster created with component enabled
         └─> Controller reconciles
              └─> GenerateValues(spec)
                   └─> MergeValues(component > RHOAI > chart defaults)
                        └─> Render(chart, values)
                             ├─ Success -> [Manifests Applied]
                             └─ Failure -> [Degraded] + Retry with backoff (FR-012)
```

### Watch Registration Lifecycle

```
[Watch Declared]
    └─> Component registered with GVK list
         └─> Controller starts
              └─> For each GVK:
                   ├─ CRD exists -> RegisterWatch() -> [Watch Active]
                   └─ CRD missing -> [Watch Pending]

[Watch Pending]
    └─> CRD created event
         └─> MatchesCRD() == true
              └─> RegisterWatch()
                   ├─ Success -> [Watch Active]
                   └─ Failure -> Log error, retry
```

### Value Merging Precedence

```
[Chart Default Values]
    └─> Load from chart.Values (values.yaml)
         └─> Merge RHOAI Overrides (values.rhoai.yaml if present)
              └─> chartutil.CoalesceTables(rhoai, chartDefaults)
                   └─> Merge Component Config (from ComponentSpec)
                        └─> chartutil.CoalesceTables(component, merged)
                             └─> [Final Values] - Component wins conflicts
```

## Invariants

1. **Registry Singleton**: Exactly one HelmManagedComponentRegistry instance per operator process
2. **Unique Component Names**: No two components with same name in registry
3. **Chart-Component Alignment**: Component name matches Chart.yaml dependency name
4. **Value Precedence**: Component values always override RHOAI values (FR-005 clarification)
5. **Startup Failure on Load Error**: Operator fails to start if any chart fails to load (FR-008)
6. **Idempotent Rendering**: Same inputs produce same rendered manifests (NFR-002)
7. **No Orphaned Watches**: All registered watches correspond to declared component GVKs
8. **Admission Validation First**: Type mismatches rejected before reconciliation (FR-011)

## Persistence

**In-Memory Only**:
- HelmManagedComponentRegistry
- chart.Chart metadata
- WatchSpecification registration state

**Kubernetes Persisted** (etcd via CRDs):
- DataScienceCluster with ComponentSpec fields
- Component status conditions
- Rendered Kubernetes resources (Deployments, Services, etc.)

**Container Image Bundled**:
- Helm charts in charts/ directory (from Chart.yaml dependencies)
- Chart.lock for dependency versions

**No Helm Release State**: Unlike helm install/upgrade, no release secrets or history

## Validation & Constraints

### Build-Time Validation
- Chart.yaml syntax validation
- Chart dependencies resolvable
- Chart.lock matches dependencies

### Startup-Time Validation (FR-008)
- All charts load successfully via LoadArchive
- Chart names match registry keys
- Required chart files present (Chart.yaml, templates/)

### Admission-Time Validation (FR-011)
- Component config field types match schema
- Required fields present
- Enum values valid
- Custom validation rules (e.g., storage size format)

### Reconciliation-Time Validation (FR-012)
- Template rendering succeeds
- Rendered manifests are valid YAML
- Kubernetes resource schemas valid
- No circular dependencies

## Extension Points

**Adding New Helm Component**:
1. Add chart dependency to Chart.yaml
2. Run `helm dependency update` (build time)
3. Define ComponentSpec type in api/components.platform.opendatahub.io/v1alpha1/
4. Implement ValuesGenerator function
5. Call Register() in init()
6. Generate CRDs with `make manifests`

**Minimal Code Changes** (FR-013 Reusability):
- ~50 lines: ComponentSpec type definition
- ~30 lines: ValuesGenerator function
- ~10 lines: Register() call in init()
- ~0 lines: Controller logic (auto-generated by factory)
