# Contract: Value Merging API

**Package**: `internal/controller/helmregistry`

## MergeValues

**Purpose**: Merge component configuration with RHOAI and chart default values

**Signature**:
```go
func (c *HelmManagedComponent) MergeValues(
    componentValues chartutil.Values,
) chartutil.Values
```

**Request Contract**:
```go
// componentValues from ValuesGenerator function
type chartutil.Values map[string]interface{}
// Example:
{
    "langfuse": {
        "features": {
            "experimentalFeaturesEnabled": true,
        },
    },
}
```

**Merge Precedence** (FR-005 clarification):
1. Chart defaults (from chart.Values / values.yaml)
2. RHOAI overrides (from c.RHOAIValues / values.rhoai.yaml)
3. Component configuration (componentValues parameter) **← WINS CONFLICTS**

**Processing Algorithm**:
```go
// Step 1: Start with chart defaults
result := c.Chart.Values

// Step 2: Merge RHOAI overrides (RHOAI wins over chart)
result = chartutil.CoalesceTables(c.RHOAIValues, result)

// Step 3: Merge component config (component wins over all)
result = chartutil.CoalesceTables(componentValues, result)

return result
```

**Deep Merge Behavior**:
- Nested maps are merged recursively
- Arrays are replaced (not merged)
- Primitives (string, int, bool) are replaced
- Null values in higher precedence remove lower precedence values

**Example Merge**:
```go
// Chart defaults:
{
    "langfuse": {
        "replicas": 1,
        "features": {
            "experimentalFeaturesEnabled": false,
            "telemetryEnabled": true,
        },
    },
}

// RHOAI overrides:
{
    "langfuse": {
        "replicas": 2,  // Override default
    },
}

// Component config:
{
    "langfuse": {
        "features": {
            "experimentalFeaturesEnabled": true,  // Override RHOAI+default
        },
    },
}

// Result:
{
    "langfuse": {
        "replicas": 2,  // From RHOAI
        "features": {
            "experimentalFeaturesEnabled": true,   // From component
            "telemetryEnabled": true,              // From chart default
        },
    },
}
```

**Postconditions**:
- Result contains all keys from all sources
- Component values present in result (never overridden)
- RHOAI values present unless overridden by component
- Chart defaults present unless overridden by RHOAI or component
- Deep nested structures properly merged

**Validation**:
- No validation at merge time (values pre-validated at admission)
- Type mismatches handled by chart schema validation
- Invalid paths silently ignored by Helm rendering

## ValuesGenerator Contract

**Purpose**: Generate Helm values from component specification

**Function Signature Template**:
```go
func ComponentValuesFromSpec(spec ComponentSpec) (chartutil.Values, error)
```

**Request Contract**:
- spec: Component-specific configuration from DataScienceCluster CR
- Must be type-safe (already passed admission validation)

**Response Contract**:
```go
// Success: Values map suitable for Helm rendering
chartutil.Values{
    "component": {
        "nested": {
            "path": value,
        },
    },
}
// Error: nil + error describing generation failure
```

**Implementation Requirements**:
1. Extract fields from ComponentSpec
2. Map to Helm chart value paths (per +helmvalue markers)
3. Apply type conversions if needed (string->int, etc.)
4. Return nested map structure matching chart's values.yaml

**Example Implementation**:
```go
func LangfuseValuesFromSpec(spec DSCLangfuse) (chartutil.Values, error) {
    values := chartutil.Values{}

    // Map features to chart paths
    values["langfuse"] = map[string]interface{}{
        "features": map[string]interface{}{
            "experimentalFeaturesEnabled": spec.Features.ExperimentalFeaturesEnabled,
        },
    }

    // Conditional values
    if spec.Features.StorageSize != "" {
        values["langfuse"].(map[string]interface{})["persistence"] = map[string]interface{}{
            "size": spec.Features.StorageSize,
        }
    }

    return values, nil
}
```

**Error Cases**:
```go
ErrInvalidSpec        // Spec contains invalid data (shouldn't happen after admission)
ErrTypeConversion     // Failed to convert type
ErrMissingRequired    // Required field not present in spec
```

**Contract Enforcement**:
- Type safety enforced by Go compiler (spec is typed struct)
- Admission webhook validates spec before ValuesGenerator called
- ValuesGenerator should not perform business logic validation

## ValuePathMapping (Marker-Based)

**Purpose**: Declarative mapping of ComponentSpec fields to Helm value paths

**Marker Syntax**:
```go
// +helmvalue:path=<dot.separated.path>
// +helmvalue:required=<true|false>
// +helmvalue:default=<value>
```

**Example**:
```go
type LangfuseFeatures struct {
    // Maps to langfuse.features.experimentalEnabled in chart
    // +helmvalue:path=langfuse.features.experimentalEnabled
    // +helmvalue:default=false
    ExperimentalFeaturesEnabled bool `json:"experimentalFeaturesEnabled"`

    // Maps to langfuse.persistence.size in chart
    // +helmvalue:path=langfuse.persistence.size
    // +helmvalue:required=false
    StorageSize string `json:"storageSize,omitempty"`
}
```

**Code Generation** (Future Enhancement):
- Markers could generate ValuesGenerator function
- Current implementation: Manual ValuesGenerator functions

**Validation**:
- Path syntax validated at build time (controller-gen)
- Path existence in chart not validated (Helm rendering handles)
- Type compatibility checked at admission time

## RHOAI Values Override

**Purpose**: Platform-specific value overrides via values.rhoai.yaml

**File Location**: Bundled in Helm chart (charts/*/values.rhoai.yaml)

**Example values.rhoai.yaml**:
```yaml
# RHOAI platform overrides for Langfuse chart
langfuse:
  replicas: 2  # RHOAI default: 2 replicas
  image:
    pullPolicy: IfNotPresent  # RHOAI: reduce registry load
  resources:
    limits:
      memory: 512Mi
    requests:
      memory: 256Mi
  serviceAccount:
    create: false  # RHOAI manages service accounts centrally
```

**Loading**:
```go
// During Register(), extract RHOAI values from chart files
for _, file := range chart.Files {
    if file.Name == "values.rhoai.yaml" {
        rhoaiValues := chartutil.Values{}
        if err := yaml.Unmarshal(file.Data, &rhoaiValues); err != nil {
            return fmt.Errorf("invalid values.rhoai.yaml: %w", err)
        }
        component.RHOAIValues = rhoaiValues
        break
    }
}
```

**Override Scope**:
- Component-specific (per chart)
- Applied to all instances of that component
- Can be overridden by component configuration (component wins)

**Best Practices**:
- Document RHOAI overrides in chart README
- Keep overrides minimal (platform defaults only)
- Use component config for user-facing configuration
