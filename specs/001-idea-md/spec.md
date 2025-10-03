# Feature Specification: Helm-Managed Component Registry

**Feature Branch**: `001-idea-md`
**Created**: 2025-10-02
**Status**: Draft
**Input**: User description: "@idea.md"

## Execution Flow (main)
```
1. Parse user description from Input
   → Completed: Extracted Helm component management system design
2. Extract key concepts from description
   → Identified: component registration, Helm chart rendering, dynamic watches, value mapping
3. For each unclear aspect:
   → Marked with [NEEDS CLARIFICATION: specific question]
4. Fill User Scenarios & Testing section
   → Completed: Primary user stories for component developers
5. Generate Functional Requirements
   → Each requirement must be testable
   → Marked ambiguous requirements
6. Identify Key Entities (if data involved)
   → Completed: Component registry, chart metadata, value mappings
7. Run Review Checklist
   → WARN "Spec has uncertainties" - several clarifications needed
8. Return: SUCCESS (spec ready for planning)
```

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT users need and WHY
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for business stakeholders, not developers

---

## Clarifications

### Session 2025-10-02
- Q: How should charts be loaded at runtime? → A: Via helm.sh/helm/v3/pkg/chart/loader's LoadArchive function
- Q: What Helm features should be supported? → A: Template rendering only, no advanced features (ArgoCD-style integration)
- Q: How reusable should the integration be? → A: Minimal code changes to add new helm-rendered components
- Q: How should helm components be added to the operator? → A: Via dependencies in Chart.yaml file in operator repo
- Q: When a Helm chart archive fails to load during initialization, what should happen? → A: Fail operator startup (block initialization until resolved)
- Q: When a component's Helm chart contains invalid templates during rendering (at reconciliation time), what should happen? → A: Fail the reconciliation, emit event, retry with backoff
- Q: When RHOAI value overrides conflict with component-specific values, which takes precedence? → A: Component values override RHOAI values (user config wins)
- Q: When value mapping fails due to type mismatches (e.g., string provided for boolean field), what should happen? → A: Fail validation at spec admission time (reject invalid DataScienceCluster)
- Q: When a component specifies watches for resource types that don't exist in the cluster, what should happen? → A: Dynamically add watch when CRD becomes available

---

## User Scenarios & Testing

### Primary User Story
As a platform component developer, I need a standardized way to add new Helm-managed components to the DataScienceCluster so that I can deploy and configure components without writing custom controller logic for each one.

### Acceptance Scenarios
1. **Given** a new component Helm chart added as a Chart.yaml dependency, **When** the operator initializes, **Then** the component becomes available for deployment in DataScienceClusters
2. **Given** a DataScienceCluster with a Helm-managed component enabled, **When** the component configuration is updated, **Then** the system renders new Kubernetes manifests using template rendering only (no hooks or advanced features)
3. **Given** a component with specific resource types to monitor, **When** the component is deployed, **Then** the system automatically watches the specified Kubernetes resource types for changes
4. **Given** a component with RHOAI-specific value overrides, **When** the component is rendered, **Then** the system merges RHOAI values with component-specific values in the correct precedence order
5. **Given** component configuration fields with value path annotations, **When** users configure those fields, **Then** the system automatically maps field values to the correct Helm chart paths
6. **Given** a component developer wants to add a new Helm component, **When** they add it via Chart.yaml dependency, **Then** minimal additional code changes are required beyond configuration

### Edge Cases
- System MUST fail operator startup when a Helm chart archive fails to load during initialization
- System MUST fail reconciliation, emit event, and retry with backoff when Helm chart rendering produces invalid templates
- Component-specific configuration values MUST take precedence over RHOAI default values when conflicts occur
- System MUST reject DataScienceCluster specs at admission time when component configuration contains type mismatches
- System MUST dynamically add watches for component resource types when the corresponding CRDs become available in the cluster

## Requirements

### Functional Requirements
- **FR-001**: System MUST allow component developers to add new Helm-managed components via Chart.yaml dependencies
- **FR-002**: System MUST support specifying which Kubernetes resource types each component should watch, and dynamically add watches when CRDs become available
- **FR-003**: System MUST automatically create controller instances for each registered Helm-managed component
- **FR-004**: System MUST render Kubernetes manifests from Helm charts using template rendering only (no hooks, tests, or other advanced Helm features)
- **FR-005**: System MUST merge RHOAI-specific value overrides (from values.rhoai.yaml) with component configuration, where component-specific values take precedence over RHOAI defaults
- **FR-006**: System MUST map component configuration fields to Helm chart value paths [NEEDS CLARIFICATION: How should nested paths be handled? Should there be validation of path correctness?]
- **FR-007**: System MUST load chart archives at runtime using standard Helm loader mechanisms
- **FR-008**: System MUST fail operator startup if any Helm chart archive fails to load during initialization
- **FR-009**: System MUST provide a way to specify component-specific features and configuration options
- **FR-010**: System MUST integrate registered components into the DataScienceCluster Components structure
- **FR-011**: System MUST validate type compatibility at admission time and reject DataScienceCluster specs with type mismatches in component configuration
- **FR-012**: System MUST fail reconciliation when Helm chart rendering produces invalid templates, emit error event, and retry with backoff
- **FR-013**: System MUST require minimal code changes to add new helm-rendered components (reusable design)

### Non-Functional Requirements
- **NFR-001**: Component registration MUST occur during system initialization [NEEDS CLARIFICATION: What happens to components registered after initialization?]
- **NFR-002**: Helm chart rendering MUST be deterministic for the same input values
- **NFR-003**: System MUST support multiple Helm-managed components running concurrently [NEEDS CLARIFICATION: Are there limits on the number of components? Resource constraints?]
- **NFR-004**: Integration design MUST minimize per-component code overhead to maintain reusability

### Key Entities

- **HelmManagedComponentRegistry**: Central registry that stores all registered Helm-managed components and provides rendering capabilities
- **HelmManagedComponent**: Represents a single component with its Helm chart, value generation function, and resource watch specifications
- **Component Specification**: The configuration structure for a component within DataScienceCluster (e.g., DSCLangfuse) containing management settings and component-specific features
- **Chart Metadata**: Information about the Helm chart including templates, files, and RHOAI-specific value overrides
- **Value Path Mapping**: Relationship between component configuration fields and their corresponding paths in the Helm chart values structure
- **Resource Watch Specification**: Defines which Kubernetes resource types (by GroupVersionKind) a component needs to monitor

---

## Review & Acceptance Checklist

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [x] No [NEEDS CLARIFICATION] markers remain (9 clarifications provided, 3 deferred to planning)
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

---

## Execution Status

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked and resolved (9 clarifications provided)
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [x] Review checklist passed

---

## Outstanding Clarifications

1. **Value Path Handling**: How should nested value paths be handled and validated? [DEFERRED: Implementation detail for planning phase]
2. **Post-Initialization Registration**: Can components be registered after initialization? [DEFERRED: Low impact - registration is initialization-time only per NFR-001]
3. **Concurrency Limits**: Are there resource constraints on concurrent components? [DEFERRED: Performance optimization detail for planning phase]
