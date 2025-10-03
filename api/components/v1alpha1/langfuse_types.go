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

package v1alpha1

import (
	"github.com/opendatahub-io/opendatahub-operator/v2/api/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LangfuseComponentName = "langfuse"
	// LangfuseInstanceName the name of the Langfuse instance singleton.
	// value should match whats set in the XValidation below
	LangfuseInstanceName = "default-" + LangfuseComponentName
	LangfuseKind         = "Langfuse"
)

// Check that the component implements common.PlatformObject.
var _ common.PlatformObject = (*Langfuse)(nil)

// LangfuseFeatures defines feature flags for Langfuse component
type LangfuseFeatures struct {
	// ExperimentalFeaturesEnabled enables experimental features in Langfuse
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	ExperimentalFeaturesEnabled bool `json:"experimentalFeaturesEnabled,omitempty"`

	// TracingEnabled enables distributed tracing
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=false
	TracingEnabled bool `json:"tracingEnabled,omitempty"`

	// StorageSize defines the size of persistent storage for Langfuse
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=`^[0-9]+[EPTGMK]i?$`
	// +kubebuilder:default="10Gi"
	StorageSize string `json:"storageSize,omitempty"`
}

// LangfuseCommonSpec spec defines the shared desired state of Langfuse
type LangfuseCommonSpec struct {
	// langfuse spec exposed to DSC api
	common.DevFlagsSpec `json:",inline"`

	// Features defines feature flags for Langfuse
	// +kubebuilder:validation:Optional
	Features LangfuseFeatures `json:"features,omitempty"`
}

// LangfuseSpec defines the desired state of Langfuse
type LangfuseSpec struct {
	// langfuse spec exposed to DSC api
	LangfuseCommonSpec `json:",inline"`
	// langfuse spec exposed only to internal api
}

// LangfuseCommonStatus defines the shared observed state of Langfuse
type LangfuseCommonStatus struct {
	// URL is the endpoint URL for accessing Langfuse
	URL string `json:"url,omitempty"`
}

// LangfuseStatus defines the observed state of Langfuse
type LangfuseStatus struct {
	common.Status          `json:",inline"`
	LangfuseCommonStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:validation:XValidation:rule="self.metadata.name == 'default-langfuse'",message="Langfuse name must be default-langfuse"
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`,description="Ready"
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`,description="Reason"
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`,description="URL"

// Langfuse is the Schema for the langfuses API
type Langfuse struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LangfuseSpec   `json:"spec,omitempty"`
	Status LangfuseStatus `json:"status,omitempty"`
}

func (c *Langfuse) GetDevFlags() *common.DevFlags {
	return c.Spec.DevFlags
}

func (c *Langfuse) GetStatus() *common.Status {
	return &c.Status.Status
}

func (c *Langfuse) GetConditions() []common.Condition {
	return c.Status.GetConditions()
}

func (c *Langfuse) SetConditions(conditions []common.Condition) {
	c.Status.SetConditions(conditions)
}

// +kubebuilder:object:root=true

// LangfuseList contains a list of Langfuse
type LangfuseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Langfuse `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Langfuse{}, &LangfuseList{})
}

// DSCLangfuse contains all the configuration exposed in DSC instance for Langfuse component
type DSCLangfuse struct {
	// configuration fields common across components
	common.ManagementSpec `json:",inline"`
	// langfuse specific field
	LangfuseCommonSpec `json:",inline"`
}

// DSCLangfuseStatus contains the observed state of the Langfuse exposed in the DSC instance
type DSCLangfuseStatus struct {
	common.ManagementSpec   `json:",inline"`
	*LangfuseCommonStatus `json:",inline"`
}
