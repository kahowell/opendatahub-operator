<!--
Sync Impact Report:
Version: 0.1.0 → 1.0.0 (Initial constitution ratification)
Modified Principles: N/A (initial creation)
Added Sections: All core principles, Kubernetes standards, Quality gates, Governance
Removed Sections: None
Templates Requiring Updates:
  ✅ plan-template.md - Constitution Check section reference added
  ✅ spec-template.md - Aligned with testability requirements
  ✅ tasks-template.md - TDD enforcement aligned
Follow-up TODOs: None
-->

# OpenDataHub Operator Constitution

## Core Principles

### I. Kubernetes-Native Design
The operator MUST follow Kubernetes controller patterns and best practices:
- All features implemented as reconciliation loops responding to CR changes
- Idempotent operations that can safely retry on failure
- Status conditions following Kubernetes conventions (Ready, Progressing, Degraded)
- Use controller-runtime patterns for watches, caching, and event handling
- No direct state mutations - all changes through declarative CRs

**Rationale**: Ensures predictable behavior in distributed systems and aligns with operator framework expectations.

### II. Component Integration Architecture
Components MUST be integrated as loosely-coupled, independently manageable units:
- Each component has dedicated controller with isolated reconciliation logic
- Component state managed through ManagementState (Managed, Removed, Unmanaged)
- Manifest-based deployment using versioned component repositories
- No cross-component dependencies in reconciliation logic
- Component failures MUST NOT cascade to other components

**Rationale**: Allows independent component lifecycle management and reduces blast radius of failures.

### III. Manifest-First Deployment (NON-NEGOTIABLE)
All component deployments MUST use externalized manifests from source repositories:
- Manifests fetched from component repositories via `get_all_manifests.sh`
- No hard-coded Kubernetes resources in operator code
- Manifest overlays applied via Kustomize for customization
- DevFlags.manifests enables testing with custom manifest sources
- Operator code orchestrates manifest application, not resource templates

**Rationale**: Separates component versioning from operator versioning, enables independent component updates.

### IV. Test-First Development
TDD MUST be enforced for all controller logic and API changes:
- Unit tests using envtest for controller reconciliation logic
- Integration tests (e2e) for multi-component scenarios
- Contract tests for CRD schema validation
- Tests written → approved → failing → then implement
- Red-Green-Refactor cycle strictly followed
- No PR merge without corresponding test coverage

**Rationale**: Kubernetes operators are complex; tests prevent regressions in reconciliation logic.

### V. API Stability & Versioning
CRD APIs MUST maintain backward compatibility and follow Kubernetes versioning:
- Semantic versioning: MAJOR (breaking), MINOR (additive), PATCH (bug fixes)
- API version progression: v1alpha1 → v1beta1 → v1 with conversion webhooks
- Deprecation warnings required 2 releases before removal
- OpenAPI schema validation for all CRD fields
- API documentation auto-generated and committed with changes

**Rationale**: Users depend on stable APIs; breaking changes require migration paths.

### VI. Observability & Debugging
Comprehensive observability MUST be built into all controllers:
- Structured logging using controller-runtime logger (JSON format in production)
- Prometheus metrics for reconciliation performance and error rates
- Status conditions expose controller state to kubectl/UI
- Event recording for significant state transitions
- PPROF endpoint for runtime profiling (disabled by default)

**Rationale**: Operators run in production; debugging requires structured observability.

### VII. Resource Efficiency
Operator MUST minimize cluster resource consumption:
- Leader election for high availability without duplicate work
- Informer caching to reduce API server load
- Rate limiting for reconciliation retries
- Resource limits defined for operator deployment
- Webhook validation to reject invalid CRs before reconciliation

**Rationale**: Operators are cluster infrastructure; resource waste impacts all workloads.

## Kubernetes Standards

### Controller Implementation
- Use controller-runtime builder pattern for all controllers
- Implement reconcile.Reconciler interface with explicit error handling
- Return ctrl.Result with RequeueAfter for rate-limited retries
- Use predicates to filter unnecessary reconciliation triggers
- Owner references for garbage collection of managed resources

### CRD Design
- All fields must have OpenAPI schema validation and descriptions
- Use kubebuilder markers for code generation (validation, RBAC, status)
- Default values specified via kubebuilder:default marker
- PrinterColumns defined for kubectl get output readability
- Subresources (status, scale) enabled where appropriate

### Security
- RBAC roles auto-generated from kubebuilder markers
- Principle of least privilege for service account permissions
- No cluster-admin permissions required for normal operation
- Webhook TLS certificates managed via cert-manager or OLM
- Sensitive data stored in Secrets, never in ConfigMaps or CRs

## Quality Gates

### Pre-Merge Requirements (NON-NEGOTIABLE)
- [ ] All unit tests pass (`make unit-test`)
- [ ] E2E tests pass for affected components
- [ ] API documentation updated (`make api-docs`)
- [ ] CRD changes include conversion logic if multi-version
- [ ] No linter errors (`golangci-lint`)
- [ ] Code coverage maintained or improved
- [ ] Commit follows conventional commit format

### Release Requirements
- [ ] Prometheus alert unit tests pass (`make test-alerts`)
- [ ] Upgrade tests validate migration from previous version
- [ ] Bundle validation passes for OLM packaging
- [ ] All CRD versions have upgrade paths documented
- [ ] Release notes document breaking changes and deprecations

## Governance

### Amendment Process
- Constitution changes require documented justification and team review
- Breaking changes to principles require MAJOR version bump
- Additive principles or clarifications require MINOR version bump
- Typos and non-semantic fixes require PATCH version bump
- All amendments must update dependent templates (plan, spec, tasks)

### Compliance Review
- All PRs MUST verify alignment with constitutional principles
- Component additions MUST follow Component Integration Architecture (II)
- API changes MUST follow API Stability & Versioning (V)
- Complexity additions MUST be justified in PR description
- Deviations require explicit approval with documented rationale

### Development Guidance
Runtime development guidance for agent-specific workflows is maintained in repository root:
- `CLAUDE.md` for Claude Code agent-specific instructions
- `.github/copilot-instructions.md` for GitHub Copilot guidance
- Project-agnostic constitutional principles remain in this file
- Agent files auto-updated via `.specify/scripts/bash/update-agent-context.sh`

**Version**: 1.0.0 | **Ratified**: 2025-10-02 | **Last Amended**: 2025-10-02
