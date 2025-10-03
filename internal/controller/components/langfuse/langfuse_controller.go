/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package langfuse

import (
	"context"
	"fmt"

	operatorv1 "github.com/openshift/api/operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	componentsv1alpha1 "github.com/opendatahub-io/opendatahub-operator/v2/api/components/v1alpha1"
	"github.com/opendatahub-io/opendatahub-operator/v2/api/common"
	"github.com/opendatahub-io/opendatahub-operator/v2/internal/controller/helmregistry"
)

// LangfuseReconciler reconciles a Langfuse object using Helm chart rendering
type LangfuseReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Component *helmregistry.HelmManagedComponent
}

// +kubebuilder:rbac:groups=components.platform.opendatahub.io,resources=langfuses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=components.platform.opendatahub.io,resources=langfuses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=components.platform.opendatahub.io,resources=langfuses/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile implements the Helm-based reconciliation logic for Langfuse
func (r *LangfuseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch Langfuse instance
	var langfuse componentsv1alpha1.Langfuse
	if err := r.Get(ctx, req.NamespacedName, &langfuse); err != nil {
		if errors.IsNotFound(err) {
			// Resource deleted, nothing to do
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Convert Langfuse CR to DSCLangfuse for values generation
	dscLangfuse := &componentsv1alpha1.DSCLangfuse{
		ManagementSpec:      common.ManagementSpec{ManagementState: operatorv1.Managed},
		LangfuseCommonSpec:  langfuse.Spec.LangfuseCommonSpec,
	}

	// Render Helm templates using registry
	manifests, err := helmregistry.HelmManagedComponents.Render("langfuse", dscLangfuse)
	if err != nil {
		logger.Error(err, "Failed to render Helm templates")
		// Update status condition (T033)
		r.updateStatusCondition(ctx, &langfuse, common.Condition{
			Type:    "Ready",
			Status:  metav1.ConditionFalse,
			Reason:  "ChartRenderError",
			Message: fmt.Sprintf("Failed to render Helm templates: %v", err),
		})
		return ctrl.Result{}, err
	}

	// Apply rendered manifests to cluster
	for name, manifestYAML := range manifests {
		if err := r.applyManifest(ctx, name, manifestYAML, &langfuse); err != nil {
			logger.Error(err, "Failed to apply manifest", "name", name)
			r.updateStatusCondition(ctx, &langfuse, common.Condition{
				Type:    "Ready",
				Status:  metav1.ConditionFalse,
				Reason:  "ManifestApplyError",
				Message: fmt.Sprintf("Failed to apply manifest %s: %v", name, err),
			})
			return ctrl.Result{}, err
		}
	}

	// Update status to Ready
	r.updateStatusCondition(ctx, &langfuse, common.Condition{
		Type:    "Ready",
		Status:  metav1.ConditionTrue,
		Reason:  "ResourcesApplied",
		Message: "All Helm manifests successfully applied",
	})

	logger.Info("Successfully reconciled Langfuse", "manifests", len(manifests))
	return ctrl.Result{}, nil
}

// applyManifest applies a single rendered manifest to the cluster
func (r *LangfuseReconciler) applyManifest(ctx context.Context, name, manifestYAML string, owner *componentsv1alpha1.Langfuse) error {
	// Parse YAML to unstructured object
	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(manifestYAML), obj); err != nil {
		return fmt.Errorf("failed to parse manifest %s: %w", name, err)
	}

	// Set owner reference for garbage collection
	if err := ctrl.SetControllerReference(owner, obj, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Apply manifest (create or update)
	if err := r.Patch(ctx, obj, client.Apply, client.ForceOwnership, client.FieldOwner("langfuse-controller")); err != nil {
		return fmt.Errorf("failed to apply manifest: %w", err)
	}

	return nil
}

// updateStatusCondition updates the status condition for Langfuse CR
// This implements T033: Add status condition updates
func (r *LangfuseReconciler) updateStatusCondition(ctx context.Context, langfuse *componentsv1alpha1.Langfuse, condition common.Condition) {
	// Update condition timestamp
	condition.LastTransitionTime = metav1.Now()

	// Find existing condition or append new one
	conditions := langfuse.Status.GetConditions()
	found := false
	for i, c := range conditions {
		if c.Type == condition.Type {
			conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		conditions = append(conditions, condition)
	}

	langfuse.Status.SetConditions(conditions)

	// Update status in cluster
	if err := r.Status().Update(ctx, langfuse); err != nil {
		log.FromContext(ctx).Error(err, "Failed to update status condition")
	}
}

// SetupWithManager sets up the controller with the Manager
// This implements T034: Integrate AddWatches() into controller setup
func (r *LangfuseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Get Langfuse component from registry
	component, exists := helmregistry.HelmManagedComponents.GetComponent("langfuse")
	if !exists {
		return fmt.Errorf("langfuse component not registered")
	}
	r.Component = component

	// Build controller
	ctrl, err := ctrl.NewControllerManagedBy(mgr).
		For(&componentsv1alpha1.Langfuse{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Build(r)
	if err != nil {
		return err
	}

	// Add dynamic watches using helmregistry (T034)
	eventHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		// Map any watched resource back to Langfuse singleton
		return []reconcile.Request{
			{NamespacedName: client.ObjectKey{Name: componentsv1alpha1.LangfuseInstanceName}},
		}
	})

	if err := component.AddWatches(ctrl.(controller.Controller), eventHandler); err != nil {
		return fmt.Errorf("failed to add dynamic watches: %w", err)
	}

	return nil
}

// GVK helpers for dynamic watches
var (
	DeploymentGVK = schema.GroupVersionKind{
		Group:   "apps",
		Version: "v1",
		Kind:    "Deployment",
	}
	ServiceGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "Service",
	}
	ConfigMapGVK = schema.GroupVersionKind{
		Group:   "",
		Version: "v1",
		Kind:    "ConfigMap",
	}
)
