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
	"regexp"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	componentsv1alpha1 "github.com/opendatahub-io/opendatahub-operator/v2/api/components/v1alpha1"
)

// +kubebuilder:webhook:path=/validate-components-platform-opendatahub-io-v1alpha1-langfuse,mutating=false,failurePolicy=fail,sideEffects=None,groups=components.platform.opendatahub.io,resources=langfuses,verbs=create;update,versions=v1alpha1,name=vlangfuse.kb.io,admissionReviewVersions=v1

// LangfuseValidator implements admission webhook validation for Langfuse
type LangfuseValidator struct{}

// ValidateCreate implements webhook.Validator
func (v *LangfuseValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	langfuse, ok := obj.(*componentsv1alpha1.Langfuse)
	if !ok {
		return nil, fmt.Errorf("expected Langfuse but got %T", obj)
	}

	return v.validateLangfuse(langfuse)
}

// ValidateUpdate implements webhook.Validator
func (v *LangfuseValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	langfuse, ok := newObj.(*componentsv1alpha1.Langfuse)
	if !ok {
		return nil, fmt.Errorf("expected Langfuse but got %T", newObj)
	}

	return v.validateLangfuse(langfuse)
}

// ValidateDelete implements webhook.Validator
func (v *LangfuseValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// No validation needed for deletion
	return nil, nil
}

// validateLangfuse performs comprehensive validation on Langfuse CR
// This implements FR-011: Type validation and field constraints
func (v *LangfuseValidator) validateLangfuse(langfuse *componentsv1alpha1.Langfuse) (admission.Warnings, error) {
	var warnings admission.Warnings

	// Validate instance name (must be default-langfuse)
	if langfuse.Name != componentsv1alpha1.LangfuseInstanceName {
		return nil, fmt.Errorf("Langfuse name must be %s, got %s", componentsv1alpha1.LangfuseInstanceName, langfuse.Name)
	}

	// Validate storage size format if specified
	if langfuse.Spec.Features.StorageSize != "" {
		if err := validateStorageSize(langfuse.Spec.Features.StorageSize); err != nil {
			return nil, fmt.Errorf("invalid storageSize: %w", err)
		}
	}

	// Warn if experimental features are enabled
	if langfuse.Spec.Features.ExperimentalFeaturesEnabled {
		warnings = append(warnings, "Experimental features are enabled - not recommended for production")
	}

	// Validate DevFlags manifests if present
	if langfuse.Spec.DevFlags != nil && len(langfuse.Spec.DevFlags.Manifests) > 0 {
		warnings = append(warnings, "DevFlags manifests are set - ensure manifests are from trusted sources")
		for i, manifest := range langfuse.Spec.DevFlags.Manifests {
			if manifest.URI == "" {
				return nil, fmt.Errorf("devFlags.manifests[%d].uri cannot be empty", i)
			}
		}
	}

	return warnings, nil
}

// validateStorageSize validates Kubernetes resource quantity format
// Pattern: numeric value followed by unit (Ei, Pi, Ti, Gi, Mi, Ki, or binary equivalents)
func validateStorageSize(size string) error {
	// Kubernetes quantity pattern: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	pattern := `^[0-9]+(\.[0-9]+)?([EPTGMK]i?)?$`
	matched, err := regexp.MatchString(pattern, size)
	if err != nil {
		return fmt.Errorf("regex error: %w", err)
	}
	if !matched {
		return fmt.Errorf("must be valid Kubernetes quantity (e.g., 10Gi, 100Mi), got %s", size)
	}
	return nil
}

// SetupWebhookWithManager registers the webhook with the manager
func (v *LangfuseValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&componentsv1alpha1.Langfuse{}).
		WithValidator(v).
		Complete()
}

// Ensure LangfuseValidator implements webhook.Validator
var _ webhook.CustomValidator = &LangfuseValidator{}
