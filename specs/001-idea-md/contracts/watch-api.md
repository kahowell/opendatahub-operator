# Contract: Dynamic Watch Registration API

**Package**: `internal/controller/helmregistry`

## AddWatches

**Purpose**: Register resource watches for a Helm component with dynamic CRD discovery

**Signature**:
```go
func (c *HelmManagedComponent) AddWatches(
    controller controller.Controller,
    handler handler.EventHandler,
) error
```

**Request Contract**:
- controller: controller-runtime Controller to add watches to
- handler: Event handler for watch events (typically enqueue owner requests)

**Processing Steps**:
1. For each GVK in c.Watches:
   - Check if CRD exists in cluster (via discovery client)
   - If exists: Register watch immediately
   - If not exists: Add to pending watches list
2. Register CRD watch to detect new CRD creations
3. On CRD creation event:
   - Check if CRD matches pending watch
   - If match: Register the pending watch

**Immediate Watch Registration**:
```go
// For existing CRDs
for _, gvk := range c.Watches {
    if crdExists(gvk) {
        err := controller.Watch(
            source.Kind(cache, gvkToObject(gvk)),
            handler,
            predicate.ResourceVersionChangedPredicate{},
        )
        if err != nil {
            return fmt.Errorf("failed to watch %s: %w", gvk, err)
        }
    }
}
```

**Deferred Watch Registration** (FR-002 clarification):
```go
// Watch for CRD creations
err := controller.Watch(
    source.Kind(cache, &apiextensionsv1.CustomResourceDefinition{}),
    handler.EnqueueRequestsFromMapFunc(c.mapCRDToComponent),
    predicate.Funcs{
        CreateFunc: func(e event.CreateEvent) bool {
            crd := e.Object.(*apiextensionsv1.CustomResourceDefinition)
            return c.hasPendingWatchForCRD(crd)
        },
    },
)
```

**Postconditions Success**:
- All watches for existing CRDs registered
- CRD creation watcher active
- Returns nil error

**Postconditions Failure**:
- Watch registration fails → Returns error
- Controller startup continues (doesn't abort operator)
- Error logged, retry on next reconciliation

**Error Cases**:
```go
ErrWatchRegistration    // controller.Watch() failed
ErrInvalidGVK           // GVK format invalid
ErrDiscoveryFailed      // Failed to query API server for CRD existence
```

## hasPendingWatchForCRD

**Purpose**: Check if a CRD matches any pending component watches

**Signature**:
```go
func (c *HelmManagedComponent) hasPendingWatchForCRD(
    crd *apiextensionsv1.CustomResourceDefinition,
) bool
```

**Logic**:
```go
crdGVK := schema.GroupVersionKind{
    Group:   crd.Spec.Group,
    Version: crd.Spec.Versions[0].Name, // Served version
    Kind:    crd.Spec.Names.Kind,
}

for _, watchGVK := range c.Watches {
    if !c.isWatchRegistered(watchGVK) && crdGVK == watchGVK {
        return true
    }
}
return false
```

**Returns**:
- true: CRD matches a pending (not yet registered) watch
- false: CRD doesn't match OR watch already registered

## mapCRDToComponent

**Purpose**: Map CRD creation event to component reconciliation request

**Signature**:
```go
func (c *HelmManagedComponent) mapCRDToComponent(
    ctx context.Context,
    obj client.Object,
) []reconcile.Request
```

**Processing**:
1. Cast obj to CRD
2. Check if CRD matches pending watch
3. Register the watch for new CRD
4. Return reconciliation request to update component

**Example**:
```go
crd := obj.(*apiextensionsv1.CustomResourceDefinition)
if !c.hasPendingWatchForCRD(crd) {
    return nil
}

// Register the watch now that CRD exists
gvk := extractGVK(crd)
err := c.registerWatch(gvk, controller, handler)
if err != nil {
    log.Error(err, "failed to register deferred watch", "gvk", gvk)
    return nil
}

// Trigger component reconciliation to apply resources for new CRD
return []reconcile.Request{
    {NamespacedName: types.NamespacedName{Name: c.ChartName}},
}
```

## Watch Predicate Filtering

**Purpose**: Reduce unnecessary reconciliations

**Predicate Types**:
```go
// Only reconcile on resource version changes (not metadata-only)
predicate.ResourceVersionChangedPredicate{}

// Custom predicate for component-specific filtering
predicate.Funcs{
    UpdateFunc: func(e event.UpdateEvent) bool {
        // Only reconcile if relevant fields changed
        return hasRelevantChanges(e.ObjectOld, e.ObjectNew)
    },
}
```

**Example Component-Specific Predicate**:
```go
// Only reconcile Langfuse on spec changes, not status
predicate.Funcs{
    UpdateFunc: func(e event.UpdateEvent) bool {
        oldLangfuse := e.ObjectOld.(*langfusev1.Langfuse)
        newLangfuse := e.ObjectNew.(*langfusev1.Langfuse)
        return !equality.Semantic.DeepEqual(oldLangfuse.Spec, newLangfuse.Spec)
    },
}
```

## Watch Lifecycle

```
[Component Registered with GVK List]
    ↓
[AddWatches Called During Controller Setup]
    ↓
For each GVK:
    ├─ CRD exists → Register watch immediately → [Watch Active]
    └─ CRD missing → Add to pending → [Watch Pending]
    ↓
[CRD Watch Active for Discovery]
    ↓
[New CRD Created Matching Pending Watch]
    ↓
[mapCRDToComponent Triggered]
    ↓
[Register Deferred Watch] → [Watch Active]
    ↓
[Component Reconciliation Triggered]
```

## Error Handling

**Watch Registration Failure**:
- Log error with details (GVK, reason)
- Continue with other watches (don't fail all)
- Retry on next controller restart
- Component status shows warning condition

**CRD Discovery Failure**:
- Log error
- Assume CRD doesn't exist (optimistic)
- Retry via CRD watcher

**Example Error Handling**:
```go
for _, gvk := range c.Watches {
    if err := registerWatchIfCRDExists(gvk); err != nil {
        log.Error(err, "failed to register watch, will retry on CRD event",
            "gvk", gvk,
            "component", c.ChartName)
        // Don't fail - watch will be registered when CRD appears
        continue
    }
}
```

## Thread Safety

**Concurrent Access**:
- AddWatches called once during controller setup (single-threaded)
- mapCRDToComponent may be called concurrently (from multiple CRD events)
- Use sync.RWMutex to protect pending watches map

**Synchronization**:
```go
type HelmManagedComponent struct {
    // ...
    pendingWatches map[schema.GroupVersionKind]bool
    watchesMutex   sync.RWMutex
}

func (c *HelmManagedComponent) hasPendingWatchForCRD(crd *apiextensionsv1.CustomResourceDefinition) bool {
    c.watchesMutex.RLock()
    defer c.watchesMutex.RUnlock()

    // Check pendingWatches map...
}
```

## Performance Considerations

**Watch Count**:
- O(components × resources per component)
- Typical: 3-5 resource types per component
- Shared informer caching reduces API server load

**CRD Discovery Overhead**:
- One-time cost at controller startup
- Uses discovery client (cached)
- Minimal latency (<100ms typical)

**Event Filtering**:
- Predicates reduce reconciliation load
- Filter at watch level (before enqueuing)
- Use ResourceVersionChanged to skip metadata-only updates
