
# Implementation Plan: Helm-Managed Component Registry

**Branch**: `001-idea-md` | **Date**: 2025-10-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/home/khowell/github/rhoai/opendatahub-operator/specs/001-idea-md/spec.md`

## Execution Flow (/plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code or `AGENTS.md` for opencode).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary
Create a Helm-managed component registry system for the OpenDataHub operator that enables component developers to add new Helm-based components to DataScienceCluster with minimal code changes. The system will load Helm charts as Chart.yaml dependencies, render templates using helm.sh/helm/v3/pkg/chart/loader's LoadArchive function, merge RHOAI-specific value overrides with component configuration (component values take precedence), and automatically create controllers with dynamic resource watches. The integration follows an ArgoCD-style approach supporting only template rendering without advanced Helm features.

## Technical Context
**Language/Version**: Go 1.24.4
**Primary Dependencies**:
- controller-runtime v0.20.4 (Kubernetes controller framework)
- helm.sh/helm/v3/pkg/chart/loader (Helm chart loading)
- k8s.io/client-go v0.32.4 (Kubernetes client)
- sigs.k8s.io/kustomize/api v0.20.1 (RHOAI value merging)

**Storage**: Kubernetes etcd (via CRD persistence)
**Testing**:
- Ginkgo v2.23.4 + Gomega v1.36.3 (BDD-style testing)
- envtest (controller unit tests)
- e2e tests in tests/e2e/

**Target Platform**: Kubernetes 1.32+ (OpenShift compatible)
**Project Type**: Single Kubernetes operator (Go monorepo)
**Performance Goals**:
- Chart loading: <5s at operator startup
- Reconciliation: <10s for component updates
- Watch registration: Dynamic (non-blocking)

**Constraints**:
- ArgoCD-style rendering (template-only, no hooks/tests/advanced features)
- Fail-fast on chart load errors at startup
- Admission-time validation for type mismatches
- Component values override RHOAI defaults

**Scale/Scope**:
- Multiple concurrent Helm-managed components per cluster
- Minimal per-component code overhead (reusable design)
- Integration with existing DataScienceCluster controller pattern

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Kubernetes-Native Design ✅
- Helm chart rendering integrated into reconciliation loops
- Idempotent template rendering (deterministic for same inputs per NFR-002)
- Status conditions will follow Kubernetes conventions
- Uses controller-runtime for watches and event handling
- All changes through declarative DataScienceCluster CR

### II. Component Integration Architecture ✅
- Each Helm component gets dedicated controller instance (FR-003)
- ManagementState integration via existing DataScienceCluster patterns
- Component failures isolated (FR-012: fail reconciliation, retry with backoff)
- No cross-component dependencies in reconciliation

### III. Manifest-First Deployment ⚠️ DEVIATION JUSTIFIED
- **Deviation**: Helm charts loaded as Chart.yaml dependencies instead of get_all_manifests.sh
- **Justification**: Helm charts are industry-standard for component packaging; LoadArchive provides manifest extraction
- **Alignment**: Rendered templates are Kubernetes manifests, same as current pattern
- **No hard-coded resources**: Manifests come from Helm chart templates
- **Kustomize compatibility**: RHOAI values.rhoai.yaml provides overlay mechanism

### IV. Test-First Development ✅
- Will follow TDD with Ginkgo/Gomega (existing test framework)
- Unit tests for chart loading, value merging, rendering logic
- Integration tests for dynamic watch registration
- Contract tests for value type validation

### V. API Stability & Versioning ✅
- Extends existing DataScienceCluster Components structure (FR-010)
- New component types added with backward compatibility
- OpenAPI schema validation for component configurations
- Admission webhook for type validation (FR-011)

### VI. Observability & Debugging ✅
- Structured logging via controller-runtime logger
- Events emitted on chart load failures (FR-008) and render errors (FR-012)
- Status conditions for component state
- Helm rendering errors surfaced in status

### VII. Resource Efficiency ✅
- Leader election via existing operator infrastructure
- Chart metadata caching to reduce rendering overhead
- Dynamic watch registration (FR-002) instead of upfront watches for all possible types
- Admission validation rejects invalid configs before reconciliation

## Project Structure

### Documentation (this feature)
```
specs/[###-feature]/
├── plan.md              # This file (/plan command output)
├── research.md          # Phase 0 output (/plan command)
├── data-model.md        # Phase 1 output (/plan command)
├── quickstart.md        # Phase 1 output (/plan command)
├── contracts/           # Phase 1 output (/plan command)
└── tasks.md             # Phase 2 output (/tasks command - NOT created by /plan)
```

### Source Code (repository root)
```
api/
├── components.platform.opendatahub.io/v1alpha1/
│   └── [New Helm component CRD types]
└── datasciencecluster/v1/
    └── [Extended Components structure]

internal/controller/
├── components.platform.opendatahub.io/
│   └── [Helm component controllers]
├── datasciencecluster/
│   └── [Extended DSC controller]
└── helmregistry/
    ├── registry.go            # HelmManagedComponentRegistry
    ├── loader.go              # Chart loading via LoadArchive
    ├── renderer.go            # Template rendering
    ├── values.go              # Value merging logic
    └── watches.go             # Dynamic watch registration

pkg/
├── helmchart/
│   ├── chart.go               # Chart metadata structures
│   └── values.go              # Value path mapping
└── controller/
    └── actions/
        └── render/
            └── helm/          # Helm rendering action

config/
├── crd/                       # Generated CRDs
├── rbac/                      # Component-specific RBAC
└── samples/                   # Example Helm component configs

Chart.yaml                     # Operator Helm dependencies
charts/                        # Packaged Helm chart dependencies

tests/
├── e2e/
│   └── helmcomponent_test.go
└── integration/
    └── helmregistry_test.go
```

**Structure Decision**: Single Go operator monorepo following existing OpenDataHub operator patterns. New code in `internal/controller/helmregistry/` package and `api/components.platform.opendatahub.io/` for Helm component types. Extends existing DataScienceCluster controller rather than creating separate projects.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:
   ```
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Generate API contracts** from functional requirements:
   - For each user action → endpoint
   - Use standard REST/GraphQL patterns
   - Output OpenAPI/GraphQL schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - One test file per endpoint
   - Assert request/response schemas
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/bash/update-agent-context.sh claude`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /tasks command will do - DO NOT execute during /plan*

**Task Generation Strategy**:
1. **Foundation Tasks** (Parallel):
   - Create helmregistry package structure
   - Define HelmManagedComponentRegistry type
   - Create test suite setup (Ginkgo/Gomega)
   - Add Helm SDK dependencies to go.mod

2. **Core Registry Tasks** (Sequential):
   - Implement Register() with chart loading (LoadArchive)
   - Write unit tests for Register() (TDD)
   - Implement Render() with template rendering
   - Write unit tests for Render() (TDD)
   - Implement value merging logic (precedence)
   - Write unit tests for value merging (TDD)

3. **Dynamic Watch Tasks** (Sequential):
   - Implement AddWatches() with CRD discovery
   - Write unit tests for watch registration (TDD)
   - Implement CRD watch mapper
   - Write integration tests for dynamic watches (envtest)

4. **CRD Extension Tasks** (Parallel per component):
   - Define Langfuse CRD type (example component)
   - Implement LangfuseValuesFromSpec generator
   - Write unit tests for values generation
   - Create sample CR (config/samples/)

5. **Controller Integration Tasks** (Sequential):
   - Create component controller factory
   - Integrate Render() into reconciliation loop
   - Add status condition updates
   - Write controller reconciliation tests

6. **Admission Webhook Tasks** (Parallel):
   - Implement type validation webhook
   - Add admission validation tests
   - Configure webhook certificates

7. **E2E Test Tasks** (Sequential):
   - Write component lifecycle test (create/update/delete)
   - Write value override test (component > RHOAI > chart)
   - Write dynamic watch test (CRD created after startup)
   - Write multi-component concurrency test

8. **Documentation Tasks** (Parallel):
   - Update operator README with Helm component guide
   - Create architecture diagram
   - Document RHOAI value override pattern
   - Add troubleshooting guide

**Ordering Strategy**:
- TDD: Tests written before implementation for each task
- Dependencies: Foundation → Core → Watches → CRD → Controller → Validation → E2E
- Parallelization: Mark [P] for independent tasks (different packages/files)
- Incremental: Each task produces working, tested code

**Task Template**:
```markdown
## Task N: [Description]

**Type**: [Implementation|Test|Documentation]
**Dependencies**: Task N-1, Task N-2
**Parallel**: [Yes|No] - Can run independently of other tasks
**Files**: pkg/helmregistry/registry.go, pkg/helmregistry/registry_test.go

**Acceptance Criteria**:
- [ ] Function implemented
- [ ] Unit tests pass
- [ ] Integration tests pass (if applicable)
- [ ] Code follows Go conventions
- [ ] RBAC markers added (if needed)
```

**Estimated Output**: 35-40 tasks

**Breakdown**:
- Foundation: 4 tasks
- Core Registry: 12 tasks (6 implementation + 6 test)
- Dynamic Watches: 8 tasks (4 implementation + 4 test)
- CRD Extension: 8 tasks (per component)
- Controller Integration: 8 tasks
- Admission Webhook: 6 tasks
- E2E Tests: 8 tasks
- Documentation: 4 tasks

**IMPORTANT**: This phase is executed by the /tasks command, NOT by /plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Manifest-First Deployment (Chart.yaml vs get_all_manifests.sh) | Helm charts are industry-standard component packaging with built-in versioning, dependencies, and value management | get_all_manifests.sh doesn't support: (1) declarative dependency management, (2) value templating, (3) chart versioning, (4) upstream component updates without manifest copying |


## Progress Tracking
*This checklist is updated during execution flow*

**Phase Status**:
- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command - describe approach only)
- [x] Phase 3: Tasks generated (/tasks command) - 46 tasks created
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS (1 justified deviation)
- [x] Post-Design Constitution Check: PASS (design aligns with constitutional principles)
- [x] All NEEDS CLARIFICATION resolved
- [x] Complexity deviations documented

**Phase 1 Artifacts Created**:
- [x] research.md - Technical research and decisions
- [x] data-model.md - Entity definitions and relationships
- [x] contracts/registry-api.md - Registry interface contracts
- [x] contracts/values-api.md - Value merging contracts
- [x] contracts/watch-api.md - Watch registration contracts
- [x] quickstart.md - Component developer quickstart guide
- [x] CLAUDE.md - Agent context updated

---
*Based on Constitution v1.0.0 - See `.specify/memory/constitution.md`*
