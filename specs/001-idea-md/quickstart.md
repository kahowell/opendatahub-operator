# Quickstart: Adding a Helm-Managed Component

**Audience**: Component developers adding new Helm charts to OpenDataHub operator
**Time**: ~30 minutes
**Prerequisites**: Go 1.24+, kubectl, Helm v3, access to Kubernetes cluster

## Goal
Add a new Helm-managed component (example: Langfuse) to the OpenDataHub operator with minimal code changes.

## Steps

### 1. Add Helm Chart Dependency (2 min)

Edit `Chart.yaml` in repository root:

```yaml
apiVersion: v2
name: opendatahub-operator
version: 2.x.x
dependencies:
  # ... existing dependencies ...

  # Add new component
  - name: langfuse
    version: "1.0.0"
    repository: "https://langfuse.github.io/langfuse-helm"
    condition: components.langfuse.enabled
```

Update chart dependencies:
```bash
helm dependency update
```

**Validation**:
```bash
ls charts/  # Should see langfuse-1.0.0.tgz
```

### 2. Define Component CRD Type (10 min)

Create `api/components.platform.opendatahub.io/v1alpha1/langfuse_types.go`:

```go
package v1alpha1

import (
    "github.com/opendatahub-io/opendatahub-operator/v2/pkg/controller/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=langfuses,scope=Namespaced
type Langfuse struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   LangfuseSpec   `json:"spec,omitempty"`
    Status types.Status   `json:"status,omitempty"`
}

type LangfuseSpec struct {
    // Common management fields
    ManagementState types.ManagementState `json:"managementState,omitempty"`

    // Component-specific features
    Features LangfuseFeatures `json:"features,omitempty"`
}

type LangfuseFeatures struct {
    // +helmvalue:path=langfuse.features.experimentalEnabled
    // +kubebuilder:default=false
    ExperimentalFeaturesEnabled bool `json:"experimentalFeaturesEnabled,omitempty"`

    // +helmvalue:path=langfuse.persistence.size
    // +kubebuilder:validation:Pattern=`^\d+(Mi|Gi)$`
    StorageSize string `json:"storageSize,omitempty"`
}

// +kubebuilder:object:root=true
type LangfuseList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []Langfuse `json:"items"`
}

func init() {
    SchemeBuilder.Register(&Langfuse{}, &LangfuseList{})
}
```

Update `api/components.platform.opendatahub.io/v1alpha1/groupversion_info.go` if needed.

### 3. Implement Values Generator (10 min)

In same file, add values generator function:

```go
import (
    "helm.sh/helm/v3/pkg/chartutil"
)

// LangfuseValuesFromSpec generates Helm values from Langfuse component spec
func LangfuseValuesFromSpec(spec LangfuseSpec) (chartutil.Values, error) {
    values := chartutil.Values{}

    // Map features to Helm chart value paths
    values["langfuse"] = map[string]interface{}{
        "features": map[string]interface{}{
            "experimentalFeaturesEnabled": spec.Features.ExperimentalFeaturesEnabled,
        },
    }

    // Conditional values
    if spec.Features.StorageSize != "" {
        langfuseMap := values["langfuse"].(map[string]interface{})
        langfuseMap["persistence"] = map[string]interface{}{
            "size": spec.Features.StorageSize,
        }
    }

    return values, nil
}
```

### 4. Register Component (5 min)

Add registration in init() function (same file):

```go
import (
    "github.com/opendatahub-io/opendatahub-operator/v2/internal/controller/helmregistry"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

func init() {
    SchemeBuilder.Register(&Langfuse{}, &LangfuseList{})

    // Register with Helm component registry
    err := helmregistry.HelmManagedComponents.Register("langfuse", helmregistry.ComponentConfig{
        ChartName: "langfuse",
        ValuesGenerator: func(spec interface{}) (chartutil.Values, error) {
            langfuseSpec := spec.(LangfuseSpec)
            return LangfuseValuesFromSpec(langfuseSpec)
        },
        Watches: []schema.GroupVersionKind{
            {Group: "apps", Version: "v1", Kind: "Deployment"},
            {Group: "", Version: "v1", Kind: "Service"},
            {Group: "", Version: "v1", Kind: "ConfigMap"},
        },
    })
    if err != nil {
        panic(fmt.Sprintf("failed to register langfuse component: %v", err))
    }
}
```

### 5. Generate CRDs and RBAC (2 min)

```bash
make manifests
```

**Validation**:
```bash
ls config/crd/bases/ | grep langfuse  # Should see langfuse CRD
ls config/rbac/ | grep langfuse        # Should see langfuse RBAC
```

### 6. Add to DataScienceCluster (Optional, 1 min)

If component should be part of DataScienceCluster, edit `api/datasciencecluster/v1/datasciencecluster_types.go`:

```go
type Components struct {
    // ... existing components ...

    // Langfuse component configuration
    Langfuse DSCLangfuse `json:"langfuse,omitempty"`
}

type DSCLangfuse struct {
    common.ManagementSpec `json:",inline"`
    LangfuseFeatures      `json:"features,omitempty"`
}
```

Re-generate manifests:
```bash
make manifests
```

### 7. Create Sample CR (2 min)

Create `config/samples/components.platform.opendatahub.io_v1alpha1_langfuse.yaml`:

```yaml
apiVersion: components.platform.opendatahub.io/v1alpha1
kind: Langfuse
metadata:
  name: langfuse-sample
spec:
  managementState: Managed
  features:
    experimentalFeaturesEnabled: true
    storageSize: "10Gi"
```

### 8. Build and Deploy Operator (3 min)

```bash
# Build operator image
make docker-build IMG=quay.io/your-org/opendatahub-operator:dev

# Push image
make docker-push IMG=quay.io/your-org/opendatahub-operator:dev

# Deploy to cluster
make deploy IMG=quay.io/your-org/opendatahub-operator:dev
```

### 9. Test Component (5 min)

```bash
# Apply sample CR
kubectl apply -f config/samples/components.platform.opendatahub.io_v1alpha1_langfuse.yaml

# Watch reconciliation
kubectl get langfuse langfuse-sample -o yaml

# Check rendered resources
kubectl get deployments,services,configmaps -l app.kubernetes.io/name=langfuse

# Check component status
kubectl get langfuse langfuse-sample -o jsonpath='{.status.conditions}'
```

**Expected Results**:
- Langfuse Deployment created
- Services and ConfigMaps from Helm chart applied
- Status shows Ready condition

### 10. Validate Value Merging (Optional, 2 min)

Create `charts/langfuse/values.rhoai.yaml` for platform overrides:

```yaml
langfuse:
  replicas: 2  # RHOAI default: 2 replicas
  image:
    pullPolicy: IfNotPresent
  resources:
    limits:
      memory: 512Mi
    requests:
      memory: 256Mi
```

Rebuild chart dependencies:
```bash
helm dependency update
```

Apply updated operator and verify:
```bash
# Check that Deployment has 2 replicas (from RHOAI override)
kubectl get deployment -l app.kubernetes.io/name=langfuse -o jsonpath='{.items[0].spec.replicas}'
# Expected: 2

# But user can override via component spec:
# spec.features.replicas: 3 would override RHOAI value
```

## Validation Checklist

- [ ] Chart dependency added to Chart.yaml
- [ ] Chart downloaded to charts/ directory
- [ ] Component CRD type defined
- [ ] ValuesGenerator function implemented
- [ ] Component registered in init()
- [ ] CRDs and RBAC generated
- [ ] Sample CR created
- [ ] Operator builds successfully
- [ ] Component reconciles successfully
- [ ] Resources created in cluster
- [ ] Status conditions updated
- [ ] Value precedence correct (component > RHOAI > chart)

## Code Change Summary

**Files Modified**: 2-3 files
**Lines of Code**: ~100 lines total
- CRD type definition: ~50 lines
- ValuesGenerator: ~30 lines
- Registration: ~10 lines
- Chart.yaml: ~5 lines

**Time Investment**: ~30 minutes (excluding chart development)

## Common Issues

### Chart Not Found
```
Error: chart langfuse not found in charts/
```
**Fix**: Run `helm dependency update` to download chart

### Registration Panic
```
panic: failed to register langfuse component: chart load failed
```
**Fix**: Check Chart.yaml dependency version matches available chart

### CRD Generation Fails
```
Error: no matches for kind "Langfuse" in version "components.platform.opendatahub.io/v1alpha1"
```
**Fix**: Run `make manifests` and `make install` to install CRDs

### Values Not Applied
```
Deployment still using chart defaults instead of component config
```
**Fix**: Check ValuesGenerator function maps fields to correct Helm value paths

### Watch Not Registered
```
Component doesn't reconcile when watched resources change
```
**Fix**: Verify GVK in Watches list matches actual resource types rendered by chart

## Next Steps

- Add unit tests for ValuesGenerator
- Add integration tests for component controller
- Document component-specific configuration options
- Create upgrade path documentation
- Add monitoring and alerts for component
