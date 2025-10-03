# Tasks: Helm-Managed Component Registry

**Input**: Design documents from `/home/khowell/github/rhoai/opendatahub-operator/specs/001-idea-md/`
**Prerequisites**: plan.md, research.md, data-model.md, contracts/, quickstart.md

## Execution Flow (main)
```
1. Load plan.md from feature directory
   → Extracted: Go 1.24.4, controller-runtime, Helm SDK, Ginkgo/Gomega
2. Load design documents:
   → data-model.md: 7 entities (HelmManagedComponentRegistry, HelmManagedComponent, etc.)
   → contracts/: 3 API contracts (registry, values, watch)
   → research.md: 6 technical decisions
   → quickstart.md: 10-step developer workflow
3. Generate tasks by category:
   → Setup: 4 tasks (dependencies, structure, test framework)
   → Tests: 16 tasks (contract tests, unit tests, integration tests)
   → Core: 14 tasks (registry, rendering, value merging, watches)
   → Integration: 6 tasks (controller, webhook, CRD)
   → Polish: 6 tasks (E2E, docs, cleanup)
4. Apply TDD: All tests before implementation
5. Mark [P] for parallel: Different packages/files
6. Total: 46 tasks
```

## Format: `[ID] [P?] Description`
- **[P]**: Can run in parallel (different files, no dependencies)
- All paths relative to repository root: `/home/khowell/github/rhoai/opendatahub-operator/`

## Phase 3.1: Setup & Dependencies

- [X] **T001** Add Helm SDK dependencies to go.mod
  - **Files**: `go.mod`, `go.sum`
  - **Action**: Add `helm.sh/helm/v3 v3.latest`, run `go mod tidy`
  - **Acceptance**: `go mod verify` succeeds, Helm packages importable

- [X] **T002** Create helmregistry package structure
  - **Files**: `internal/controller/helmregistry/`
  - **Action**: Create directory with empty `.gitkeep`
  - **Acceptance**: Directory exists in internal/controller/

- [X] **T003** Create helmchart package structure
  - **Files**: `pkg/helmchart/`
  - **Action**: Create directory with empty `.gitkeep`
  - **Acceptance**: Directory exists in pkg/

- [X] **T004** [P] Set up Ginkgo test suite for helmregistry
  - **Files**: `internal/controller/helmregistry/suite_test.go`
  - **Action**: Create Ginkgo test suite boilerplate with envtest setup
  - **Acceptance**: `make test` runs suite (0 specs)
  - **Template**:
    ```go
    var _ = BeforeSuite(func() {
        testEnv = &envtest.Environment{}
        cfg, err = testEnv.Start()
        // ...
    })
    ```

## Phase 3.2: Tests First (TDD) ⚠️ MUST COMPLETE BEFORE 3.3

**CRITICAL: These tests MUST be written and MUST FAIL before ANY implementation**

### Contract Tests (from contracts/)

- [X] **T005** [P] Contract test for Registry.Register() API
  - **Files**: `internal/controller/helmregistry/registry_test.go`
  - **Contract**: contracts/registry-api.md - Register()
  - **Action**: Write Ginkgo specs for Register() contract
  - **Test Cases**:
    - Successful registration with valid chart
    - Error on duplicate component name (ErrDuplicateComponent)
    - Error on chart not found (ErrChartNotFound)
    - Error on nil ValuesGenerator (ErrInvalidConfig)
  - **Acceptance**: Tests written, all FAIL (functions not implemented)

- [X] **T006** [P] Contract test for Registry.Render() API
  - **Files**: `internal/controller/helmregistry/registry_test.go`
  - **Contract**: contracts/registry-api.md - Render()
  - **Action**: Write Ginkgo specs for Render() contract
  - **Test Cases**:
    - Successful rendering with valid values
    - Returns map[string]string with rendered manifests
    - Error on component not found (ErrComponentNotFound)
    - Error on template rendering failure (ErrTemplateRendering)
    - Deterministic output for same inputs (NFR-002)
  - **Acceptance**: Tests written, all FAIL

- [X] **T007** [P] Contract test for ValuePathMapping and MergeValues()
  - **Files**: `internal/controller/helmregistry/values_test.go`
  - **Contract**: contracts/values-api.md - MergeValues(), ValuesGenerator
  - **Action**: Write Ginkgo specs for value merging precedence
  - **Test Cases**:
    - Component values override RHOAI values
    - RHOAI values override chart defaults
    - Deep nested map merging
    - Array replacement (not merge)
    - Null value handling
  - **Acceptance**: Tests written, all FAIL

- [X] **T008** [P] Contract test for AddWatches() dynamic registration
  - **Files**: `internal/controller/helmregistry/watches_test.go`
  - **Contract**: contracts/watch-api.md - AddWatches(), hasPendingWatchForCRD
  - **Action**: Write Ginkgo specs for watch lifecycle
  - **Test Cases**:
    - Immediate watch registration for existing CRD
    - Deferred watch for missing CRD
    - CRD creation triggers pending watch
    - Predicate filtering reduces reconciliations
  - **Acceptance**: Tests written, all FAIL

### Unit Tests (from data-model.md entities)

- [X] **T009** [P] Unit test for chart loading via LoadArchive
  - **Files**: `internal/controller/helmregistry/loader_test.go`
  - **Entity**: HelmManagedComponent.LoadChart()
  - **Action**: Write tests for chart loading from charts/ directory
  - **Test Cases**:
    - Load valid .tgz chart archive
    - Load unpacked chart directory
    - Error on missing chart file
    - Error on invalid chart structure
    - Extract RHOAI values from values.rhoai.yaml if present
  - **Acceptance**: Tests written, all FAIL

- [X] **T010** [P] Unit test for template rendering with Helm engine
  - **Files**: `internal/controller/helmregistry/renderer_test.go`
  - **Entity**: HelmManagedComponent - template rendering
  - **Action**: Write tests for engine.Render() wrapper
  - **Test Cases**:
    - Render templates to map[string]string
    - Exclude NOTES.txt, tests/, hooks/
    - Template syntax errors return ErrTemplateRendering
    - Rendered output is valid YAML
  - **Acceptance**: Tests written, all FAIL

- [X] **T011** [P] Unit test for ValuesGenerator functions
  - **Files**: `internal/controller/helmregistry/langfuse_values_test.go`
  - **Entity**: ComponentSpec - values generation
  - **Action**: Write tests for Langfuse example ValuesGenerator
  - **Test Cases**:
    - Map Features fields to Helm value paths
    - Handle optional fields (omitempty)
    - Type conversions (if needed)
    - Return proper chartutil.Values structure
  - **Acceptance**: Tests written, all FAIL

### Integration Tests (from quickstart.md scenarios)

- [X] **T012** [P] Integration test for component registration at startup
  - **Files**: `internal/controller/helmregistry/integration_test.go`
  - **Scenario**: Quickstart step 8 - operator startup
  - **Action**: Write envtest integration test for init() registration
  - **Test Cases**:
    - Component registered during init()
    - Chart loaded successfully
    - Registry contains component
    - Operator startup succeeds
    - Operator startup FAILS on chart load error (FR-008)
  - **Acceptance**: Tests written, all FAIL

- [X] **T013** [P] Integration test for dynamic watch registration
  - **Files**: `internal/controller/helmregistry/watches_integration_test.go`
  - **Scenario**: Watch registration with CRD discovery (FR-002)
  - **Action**: Write envtest test for deferred watch activation
  - **Test Cases**:
    - Component registers with watch for non-existent CRD
    - Watch is pending initially
    - Create CRD matching watch GVK
    - Watch automatically registers
    - Reconciliation triggered
  - **Acceptance**: Tests written, all FAIL

- [X] **T014** [P] Integration test for value precedence (component > RHOAI > chart)
  - **Files**: `internal/controller/helmregistry/values_integration_test.go`
  - **Scenario**: Quickstart step 10 - value override validation
  - **Action**: Write test for three-layer value merging
  - **Test Cases**:
    - Chart provides default values
    - RHOAI values.rhoai.yaml overrides chart defaults
    - Component config overrides RHOAI values (FR-005 clarification)
    - Final rendered manifest reflects component config
  - **Acceptance**: Tests written, all FAIL

- [ ] **T015** [P] Integration test for reconciliation failure retry
  - **Files**: `internal/controller/components.platform.opendatahub.io/langfuse_controller_test.go`
  - **Scenario**: Template rendering failure handling (FR-012)
  - **Action**: Write test for reconciliation error handling
  - **Test Cases**:
    - Template rendering fails (invalid syntax)
    - Reconciliation returns error
    - Event emitted
    - Retry with backoff (RequeueAfter)
    - Status condition shows Degraded
  - **Acceptance**: Tests written, all FAIL

### Admission Webhook Tests

- [ ] **T016** [P] Admission webhook test for type validation
  - **Files**: `internal/webhook/components.platform.opendatahub.io/langfuse_validating_test.go`
  - **Scenario**: Type mismatch rejection at admission time (FR-011)
  - **Action**: Write webhook validation tests
  - **Test Cases**:
    - Valid component config accepted
    - String for bool field rejected
    - Invalid enum value rejected
    - Missing required field rejected
    - Type mismatch returns clear error message
  - **Acceptance**: Tests written, all FAIL

## Phase 3.3: Core Implementation (ONLY after tests are failing)

### Registry Implementation

- [X] **T017** Implement HelmManagedComponentRegistry type
  - **Files**: `internal/controller/helmregistry/registry.go`
  - **Entity**: HelmManagedComponentRegistry (data-model.md)
  - **Action**: Define registry struct with components map and chartCache
  - **Acceptance**: Type compiles, T005 test structure passes (registration still fails)
  - **Code**:
    ```go
    type HelmManagedComponentRegistry struct {
        components map[string]*HelmManagedComponent
        mu         sync.RWMutex
    }
    var HelmManagedComponents = &HelmManagedComponentRegistry{
        components: make(map[string]*HelmManagedComponent),
    }
    ```

- [X] **T018** Implement HelmManagedComponent type
  - **Files**: `internal/controller/helmregistry/types.go`
  - **Entity**: HelmManagedComponent (data-model.md)
  - **Action**: Define component struct with Chart, ValuesGenerator, Watches
  - **Acceptance**: Type compiles with all required fields

- [X] **T019** Implement Register() method
  - **Files**: `internal/controller/helmregistry/registry.go`
  - **Contract**: contracts/registry-api.md - Register()
  - **Action**: Implement component registration with chart loading
  - **Logic**:
    - Validate component config (non-empty name, non-nil ValuesGenerator)
    - Check for duplicate name
    - Call LoadChart() to load Helm chart
    - Extract RHOAI values from chart.Files
    - Store component in registry
  - **Acceptance**: T005 contract tests PASS
  - **Dependencies**: Requires T020 (LoadChart)

- [X] **T020** Implement LoadChart() using helm.sh/helm/v3/pkg/chart/loader
  - **Files**: `internal/controller/helmregistry/loader.go`
  - **Entity**: HelmManagedComponent.LoadChart()
  - **Action**: Implement chart loading via LoadArchive
  - **Logic**:
    - Construct chart path: charts/{chartName}-{version}.tgz
    - Call loader.LoadArchive(chartPath)
    - Validate chart structure
    - Return *chart.Chart or error
  - **Acceptance**: T009 unit tests PASS
  - **Code Pattern**:
    ```go
    import "helm.sh/helm/v3/pkg/chart/loader"

    func (c *HelmManagedComponent) LoadChart(chartPath string) error {
        chart, err := loader.LoadArchive(chartPath)
        if err != nil {
            return fmt.Errorf("chart load failed: %w", err)
        }
        c.Chart = chart
        return nil
    }
    ```

- [X] **T021** Implement Render() method
  - **Files**: `internal/controller/helmregistry/registry.go`
  - **Contract**: contracts/registry-api.md - Render()
  - **Action**: Implement template rendering orchestration
  - **Logic**:
    - Get component from registry
    - Call ValuesGenerator to produce component values
    - Call MergeValues() for precedence
    - Call renderTemplates() with Helm engine
    - Return manifest map or error
  - **Acceptance**: T006 contract tests PASS
  - **Dependencies**: Requires T022 (renderTemplates), T023 (MergeValues)

- [X] **T022** Implement renderTemplates() using helm.sh/helm/v3/pkg/engine
  - **Files**: `internal/controller/helmregistry/renderer.go`
  - **Entity**: Template rendering with Helm engine
  - **Action**: Implement ArgoCD-style template-only rendering
  - **Logic**:
    - Create engine.Engine instance
    - Call engine.Render(chart, values)
    - Filter out NOTES.txt, tests/, hooks/ (ArgoCD pattern)
    - Validate rendered YAML
    - Return map[string]string
  - **Acceptance**: T010 unit tests PASS
  - **Code Pattern**:
    ```go
    import "helm.sh/helm/v3/pkg/engine"

    renderer := engine.Engine{}
    manifests, err := renderer.Render(chart, values)
    // Filter and return
    ```

### Value Merging Implementation

- [X] **T023** Implement MergeValues() with precedence logic
  - **Files**: `internal/controller/helmregistry/values.go`
  - **Contract**: contracts/values-api.md - MergeValues()
  - **Action**: Implement three-layer value merging
  - **Logic**:
    - Start with chart.Values (chart defaults)
    - Merge RHOAI values using chartutil.CoalesceTables()
    - Merge component values (wins conflicts)
    - Return final chartutil.Values
  - **Acceptance**: T007 contract tests PASS
  - **Code Pattern**:
    ```go
    import "helm.sh/helm/v3/pkg/chartutil"

    result := c.Chart.Values
    result = chartutil.CoalesceTables(c.RHOAIValues, result)
    result = chartutil.CoalesceTables(componentValues, result)
    return result
    ```

- [X] **T024** Extract RHOAI values from chart.Files during registration
  - **Files**: `internal/controller/helmregistry/loader.go`
  - **Entity**: HelmManagedComponent.RHOAIValues
  - **Action**: Parse values.rhoai.yaml from chart files
  - **Logic**:
    - Iterate chart.Files looking for "values.rhoai.yaml"
    - If found, unmarshal YAML to chartutil.Values
    - Store in component.RHOAIValues
  - **Acceptance**: T014 integration test PASS (RHOAI values applied)

### Dynamic Watch Implementation

- [X] **T025** Implement AddWatches() with CRD discovery
  - **Files**: `internal/controller/helmregistry/watches.go`
  - **Contract**: contracts/watch-api.md - AddWatches()
  - **Action**: Implement immediate and deferred watch registration
  - **Logic**:
    - For each GVK in component.Watches:
      - Check if CRD exists using discovery client
      - If exists: Register watch immediately via controller.Watch()
      - If not: Add to pendingWatches map
    - Register CRD watcher via controller.Watch(&CRD{}, ...)
  - **Acceptance**: T008 contract tests PASS
  - **Dependencies**: Requires T026 (hasPendingWatchForCRD)

- [X] **T026** Implement hasPendingWatchForCRD() predicate
  - **Files**: `internal/controller/helmregistry/watches.go`
  - **Contract**: contracts/watch-api.md - hasPendingWatchForCRD
  - **Action**: Implement CRD matching logic
  - **Logic**:
    - Extract GVK from CRD spec
    - Check if matches any component.Watches
    - Check if watch is pending (not already registered)
    - Return bool
  - **Acceptance**: T013 integration test PASS (deferred watch activates)

- [X] **T027** Implement mapCRDToComponent() for deferred watch activation
  - **Files**: `internal/controller/helmregistry/watches.go`
  - **Contract**: contracts/watch-api.md - mapCRDToComponent
  - **Action**: Implement CRD event handler
  - **Logic**:
    - Cast event object to CRD
    - Call hasPendingWatchForCRD()
    - If true: Register the pending watch
    - Return reconcile.Request to trigger component reconciliation
  - **Acceptance**: T013 integration test PASS (watch registered, reconciliation triggered)

## Phase 3.4: CRD & Controller Integration

### CRD Extension

- [X] **T028** [P] Define Langfuse CRD types (example component)
  - **Files**: `api/components/v1alpha1/langfuse_types.go`
  - **Entity**: ComponentSpec (data-model.md)
  - **Action**: Define Langfuse CR type with kubebuilder markers
  - **Structure**:
    - Langfuse type with Spec and Status
    - LangfuseSpec with ManagementState + Features
    - LangfuseFeatures with bool flags
    - Kubebuilder markers for validation, RBAC, printcolumns
  - **Acceptance**: Type compiles, `make manifests` generates CRD

- [X] **T029** [P] Implement LangfuseValuesFromSpec() generator
  - **Files**: `internal/controller/helmregistry/langfuse_values.go`
  - **Contract**: contracts/values-api.md - ValuesGenerator
  - **Action**: Implement values generation from LangfuseSpec
  - **Logic**:
    - Extract Features fields
    - Map to Helm chart value paths (langfuse.features.*)
    - Return chartutil.Values with nested structure
  - **Acceptance**: T011 unit tests PASS

- [X] **T030** [P] Register Langfuse component in init()
  - **Files**: `internal/controller/helmregistry/langfuse_init.go`
  - **Entity**: Component registration
  - **Action**: Add init() with Register() call
  - **Logic**:
    - Call HelmManagedComponents.Register("langfuse", ...)
    - Pass LangfuseValuesFromSpec as ValuesGenerator
    - Specify Watches: Deployment, Service, ConfigMap
    - Panic on error (fail-fast per FR-008)
  - **Acceptance**: T012 integration test PASS (component registered at startup)

- [X] **T031** Generate CRDs and RBAC manifests
  - **Files**: `config/crd/`, `config/rbac/`
  - **Action**: Run `make manifests` to generate from kubebuilder markers
  - **Acceptance**: CRD YAML created, RBAC rules generated

### Controller Implementation

- [X] **T032** Create Langfuse controller with Helm rendering
  - **Files**: `internal/controller/components/langfuse/langfuse_controller.go`
  - **Entity**: Component controller with reconciliation
  - **Action**: Implement controller.Reconciler for Langfuse
  - **Logic**:
    - Get Langfuse CR from cluster
    - Call HelmManagedComponents.Render("langfuse", spec)
    - Apply rendered manifests using client
    - Update status conditions (Ready, Progressing, Degraded)
    - Handle errors with retry backoff (FR-012)
  - **Acceptance**: Langfuse CR reconciles, resources created
  - **Dependencies**: Requires T021 (Render)

- [X] **T033** Add status condition updates to controller
  - **Files**: `internal/controller/components/langfuse/langfuse_controller.go`
  - **Entity**: ComponentStatus (data-model.md)
  - **Action**: Implement status condition logic
  - **Conditions**:
    - Ready: Reconciliation successful
    - Progressing: Rendering in progress
    - Degraded: Rendering failed (ChartRenderError reason)
    - Available: Resources deployed
  - **Acceptance**: Status conditions visible in `kubectl get langfuse -o yaml`

- [X] **T034** Integrate AddWatches() into controller setup
  - **Files**: `internal/controller/components/langfuse/langfuse_controller.go`
  - **Entity**: Watch registration during controller initialization
  - **Action**: Call AddWatches() in SetupWithManager()
  - **Logic**:
    - Get component from registry
    - Call component.AddWatches(controller, handler)
    - Return error if watch registration fails
  - **Acceptance**: Watches registered for Deployment, Service, ConfigMap
  - **Dependencies**: Requires T025 (AddWatches)

### Admission Webhook

- [X] **T035** Implement type validation webhook for Langfuse
  - **Files**: `internal/controller/components/langfuse/langfuse_webhook.go`
  - **Entity**: Admission validation (FR-011)
  - **Action**: Implement ValidateCreate/ValidateUpdate webhooks
  - **Validation**:
    - Check field types match schema (bool for bool field, etc.)
    - Reject string for bool (FR-011 clarification)
    - Validate enum values for ManagementState
    - Check required fields present
  - **Acceptance**: T016 admission webhook tests PASS

- [X] **T036** Configure webhook certificates and registration
  - **Files**: `config/webhook/manifests.yaml`
  - **Action**: Set up webhook configuration manifests
  - **Acceptance**: Webhook registered in cluster, cert-manager provisions TLS

## Phase 3.5: E2E Tests & Documentation

### E2E Tests

- [ ] **T037** E2E test for component lifecycle (create, update, delete)
  - **Files**: `tests/e2e/helmcomponent_test.go`
  - **Scenario**: Full component lifecycle from quickstart
  - **Action**: Write E2E test using test cluster
  - **Test Flow**:
    - Apply Langfuse CR with managementState: Managed
    - Wait for Ready condition
    - Verify Deployment, Service created
    - Update component config (change feature flag)
    - Verify resources updated with new values
    - Delete CR
    - Verify resources cleaned up
  - **Acceptance**: E2E test PASSES end-to-end

- [ ] **T038** E2E test for value override precedence
  - **Files**: `tests/e2e/helmcomponent_values_test.go`
  - **Scenario**: Quickstart step 10 validation
  - **Action**: Test three-layer value merging
  - **Test Flow**:
    - Create chart with default replicas: 1
    - Add values.rhoai.yaml with replicas: 2
    - Create Langfuse CR (no replicas override)
    - Verify Deployment has 2 replicas (RHOAI wins)
    - Update Langfuse CR with explicit replicas: 3
    - Verify Deployment has 3 replicas (component wins)
  - **Acceptance**: Value precedence correct (component > RHOAI > chart)

- [ ] **T039** E2E test for dynamic watch registration
  - **Files**: `tests/e2e/helmcomponent_watches_test.go`
  - **Scenario**: CRD created after operator startup (FR-002)
  - **Action**: Test deferred watch activation
  - **Test Flow**:
    - Start operator without target CRD
    - Verify watch is pending
    - Create CRD matching component watch
    - Verify watch registered
    - Create CR of new type
    - Verify component reconciles
  - **Acceptance**: Dynamic watch activation works

- [ ] **T040** E2E test for chart load failure (operator startup failure)
  - **Files**: `tests/e2e/helmcomponent_startup_test.go`
  - **Scenario**: FR-008 - Fail operator startup on chart load error
  - **Action**: Test fail-fast behavior
  - **Test Flow**:
    - Remove chart from charts/ directory
    - Attempt operator startup
    - Verify operator fails to start
    - Verify error message mentions missing chart
  - **Acceptance**: Operator startup fails as expected

- [ ] **T041** E2E test for template rendering failure with retry
  - **Files**: `tests/e2e/helmcomponent_render_error_test.go`
  - **Scenario**: FR-012 - Reconciliation retry on render error
  - **Action**: Test error handling and backoff
  - **Test Flow**:
    - Create Langfuse CR with invalid template values
    - Verify reconciliation fails
    - Verify status shows Degraded condition
    - Verify event emitted
    - Verify retry with backoff (RequeueAfter)
  - **Acceptance**: T015 integration test behavior confirmed in E2E

- [ ] **T042** E2E test for multi-component concurrency
  - **Files**: `tests/e2e/helmcomponent_concurrent_test.go`
  - **Scenario**: Multiple Helm components running simultaneously (NFR-003)
  - **Action**: Test concurrent component reconciliation
  - **Test Flow**:
    - Register 3 different Helm components
    - Create CRs for all 3 simultaneously
    - Verify all reconcile independently
    - Verify no resource conflicts
    - Verify all reach Ready state
  - **Acceptance**: Concurrent components work without interference

### Documentation & Polish

- [ ] **T043** [P] Update operator README with Helm component guide
  - **Files**: `README.md`
  - **Action**: Add section on Helm-managed components
  - **Content**:
    - Overview of Helm integration
    - Link to quickstart.md
    - Component developer workflow summary
    - Value precedence documentation
  - **Acceptance**: README has clear Helm component section

- [ ] **T044** [P] Create architecture diagram
  - **Files**: `docs/architecture/helm-components.md`
  - **Action**: Document architecture with diagram
  - **Content**:
    - Component registration flow diagram
    - Value merging precedence diagram
    - Dynamic watch registration sequence diagram
    - Controller reconciliation flowchart
  - **Acceptance**: Architecture documented with visuals

- [ ] **T045** [P] Document troubleshooting guide
  - **Files**: `docs/troubleshooting/helm-components.md`
  - **Action**: Create troubleshooting guide
  - **Content**:
    - Chart not found errors
    - Registration failures
    - Template rendering errors
    - Value merging issues
    - Watch registration problems
  - **Acceptance**: Common issues documented with solutions

- [ ] **T046** [P] Add sample RHOAI values.rhoai.yaml to example chart
  - **Files**: `docs/examples/helm-components/langfuse-values.rhoai.yaml`
  - **Action**: Create example RHOAI override file
  - **Content**:
    - Platform-specific defaults
    - Resource limits/requests
    - ServiceAccount settings
    - Comments explaining purpose
  - **Acceptance**: Example demonstrates value override pattern

## Dependencies

```
Setup (T001-T004)
  └─> Tests (T005-T016) [All tests before implementation]
       └─> Core Implementation (T017-T027)
            ├─> Registry: T017 → T018 → T019 (needs T020) → T021 (needs T022, T023)
            ├─> Loading: T020 → T024
            ├─> Rendering: T022 → T023
            └─> Watches: T025 (needs T026) → T026 → T027
       └─> CRD & Integration (T028-T036)
            ├─> CRD: T028 → T029 → T030 → T031
            └─> Controller: T032 → T033 → T034 (needs T025)
            └─> Webhook: T035 → T036
       └─> E2E & Docs (T037-T046) [Can run after T036 complete]
```

## Parallel Execution Examples

### Phase 3.2: All Tests in Parallel
```bash
# Launch contract tests together (different test files):
T005: Contract test Registry.Register()
T006: Contract test Registry.Render()
T007: Contract test value merging
T008: Contract test AddWatches()

# Launch unit tests together:
T009: Unit test LoadChart()
T010: Unit test template rendering
T011: Unit test ValuesGenerator

# Launch integration tests together:
T012: Integration test startup registration
T013: Integration test dynamic watches
T014: Integration test value precedence
T015: Integration test reconciliation retry
T016: Admission webhook test
```

### Phase 3.3: Core Implementation (Sequential within modules, parallel across)
```bash
# Can run in parallel (different packages):
T017+T018: Registry types (internal/controller/helmregistry/)
T028+T029: CRD types (api/components.platform.opendatahub.io/)

# Sequential within registry package:
T019 (depends on T020)
T021 (depends on T022, T023)
T025 (depends on T026, T027)
```

### Phase 3.5: Documentation in Parallel
```bash
# All docs can run concurrently:
T043: Update README
T044: Architecture diagram
T045: Troubleshooting guide
T046: RHOAI values example
```

## Validation Checklist

**GATE: Verified before marking tasks.md complete**

- [x] All contracts have corresponding tests (T005-T008 cover contracts/)
- [x] All entities have model tasks (T017-T027 cover data-model.md entities)
- [x] All tests come before implementation (Phase 3.2 before 3.3)
- [x] Parallel tasks are truly independent (verified file paths)
- [x] Each task specifies exact file path
- [x] No task modifies same file as another [P] task
- [x] TDD enforced: Tests written and failing before implementation
- [x] All user stories covered by integration/E2E tests

## Notes

- **[P] tasks**: Different files/packages, no dependencies, can run in parallel
- **Test-First**: All Phase 3.2 tests MUST be written and failing before Phase 3.3 implementation
- **Commit Strategy**: Commit after each task completion
- **Ginkgo/Gomega**: Use existing test framework patterns from opendatahub-operator
- **Kubebuilder Markers**: Use existing marker patterns for CRD generation
- **Error Handling**: Follow Go error wrapping conventions (fmt.Errorf("context: %w", err))
- **Status Conditions**: Use metav1.Condition with standard Kubernetes condition types
- **Helm Versions**: Use Helm v3 API (no v2 compatibility needed)

## Task Generation Rules Applied

1. **From Contracts** (contracts/):
   - registry-api.md → T005, T006, T019, T021
   - values-api.md → T007, T023, T024, T029
   - watch-api.md → T008, T025, T026, T027

2. **From Data Model** (data-model.md):
   - HelmManagedComponentRegistry → T017
   - HelmManagedComponent → T018
   - ComponentSpec → T028
   - LoadChart operation → T020
   - Template rendering → T022
   - Value merging → T023

3. **From Quickstart** (quickstart.md):
   - Steps 1-7 → T028-T031 (CRD creation workflow)
   - Step 8 → T012, T037 (operator deployment and lifecycle)
   - Step 9 → T037 (component testing)
   - Step 10 → T014, T038 (value override validation)

4. **Ordering**:
   - Setup (T001-T004) → Tests (T005-T016) → Core (T017-T027) → Integration (T028-T036) → E2E/Docs (T037-T046)
   - Tests before implementation (TDD enforced)
   - Dependencies tracked in dependency graph

## Implementation Notes

**TDD Workflow**:
1. Run Phase 3.2 tasks: All tests written and FAILING
2. Verify tests fail with expected error messages
3. Run Phase 3.3 tasks: Implement to make tests PASS
4. Each implementation task should make specific tests PASS
5. Re-run full test suite after each task

**Performance Targets**:
- Chart loading: <5s total (T020, T012 validate)
- Template rendering: <1s per component (T022, T010 validate)
- Reconciliation: <10s for component updates (T037 validates)

**Constitution Compliance**:
- All tasks follow Test-First Development (Principle IV)
- Controller patterns use controller-runtime (Principle I)
- Component isolation maintained (Principle II)
- Observability built-in via status conditions and events (Principle VI)
