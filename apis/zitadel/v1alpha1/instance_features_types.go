/*
Copyright 2025 The Crossplane Authors.

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
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// InstanceFeaturesSpec defines the desired state of a InstanceFeatures.
type InstanceFeaturesSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              InstanceFeaturesParameters `json:"forProvider"`
}

// InstanceFeaturesParameters is the desired configuration of a Zitadel instance feature flags.
type InstanceFeaturesParameters struct {
	// LoginDefaultOrg controls the following. Make the login screen use the default organization's settings when no organization context is set, rather than the organization's own
	//
	// It is optional.
	LoginDefaultOrg *bool `json:"loginDefaultOrg,omitempty"`
	// UserSchema controls the following. Enable the user schema API, which manages a schema of extra data per user. The API is early and may change between Zitadel releases
	//
	// It is optional.
	UserSchema *bool `json:"userSchema,omitempty"`
	// ImprovedPerformance controls the following. Execution paths Zitadel may shortcut in exchange for less work on write. Accepted values are `IMPROVED_PERFORMANCE_PROJECT_GRANT`, `IMPROVED_PERFORMANCE_PROJECT`, `IMPROVED_PERFORMANCE_USER_GRANT` and `IMPROVED_PERFORMANCE_ORG_DOMAIN_VERIFIED`
	//
	// It is optional.
	ImprovedPerformance []string `json:"improvedPerformance,omitempty"`
	// DebugOidcParentError controls the following. Return the underlying error to an OIDC client instead of a generic failure. It can reveal details about the system to anyone who asks, so it is meant for debugging rather than everyday use
	//
	// It is optional.
	DebugOidcParentError *bool `json:"debugOidcParentError,omitempty"`
}

// InstanceFeaturesObservation is what Zitadel reports about a Zitadel instance feature flags.
type InstanceFeaturesObservation struct {
	// Scope is what identifies these settings. It is empty for a
	// singleton and carries the generator type for the keyed kind.
	Scope string `json:"scope,omitempty"`

	LoginDefaultOrg            bool     `json:"loginDefaultOrg,omitempty"`
	UserSchema                 bool     `json:"userSchema,omitempty"`
	ImprovedPerformance        []string `json:"improvedPerformance,omitempty"`
	DebugOidcParentError       bool     `json:"debugOidcParentError,omitempty"`
	LoginDefaultOrgSource      string   `json:"loginDefaultOrgSource,omitempty"`
	UserSchemaSource           string   `json:"userSchemaSource,omitempty"`
	ImprovedPerformanceSource  string   `json:"improvedPerformanceSource,omitempty"`
	DebugOidcParentErrorSource string   `json:"debugOidcParentErrorSource,omitempty"`
}

// InstanceFeaturesStatus reports the observed state of a Zitadel instance feature flags.
type InstanceFeaturesStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider InstanceFeaturesObservation `json:"atProvider,omitempty"`

	// Restore is the value this resource overwrote, kept so that deleting
	// it can put it back.
	Restore []byte `json:"restore,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// InstanceFeatures is a managed resource that turn the instance wide feature flags on or off.
//
// This is a namespaced object managing state that belongs to the whole
// Zitadel instance. Two of them in different namespaces would be asking
// for the same settings, and the second would win.
// +kubebuilder:object:generate=true
type InstanceFeatures struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceFeaturesSpec   `json:"spec"`
	Status InstanceFeaturesStatus `json:"status,omitempty"`
}

// SetPolicyScope records the scope these settings were last read from.
func (mg *InstanceFeatures) SetPolicyScope(scope string) { mg.Status.AtProvider.Scope = scope }

// PolicyScope returns the recorded scope.
func (mg *InstanceFeatures) PolicyScope() string { return mg.Status.AtProvider.Scope }

// SetPolicyRestore records the value to put back when this resource is deleted.
func (mg *InstanceFeatures) SetPolicyRestore(restore []byte) { mg.Status.Restore = restore }

// PolicyRestore returns the recorded restore point.
func (mg *InstanceFeatures) PolicyRestore() []byte { return mg.Status.Restore }

// +kubebuilder:object:root=true

// InstanceFeaturesList contains a list of InstanceFeatures.
type InstanceFeaturesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanceFeatures `json:"items"`
}

// InstanceFeatures type metadata.
var (
	InstanceFeaturesKind             = reflect.TypeOf(InstanceFeatures{}).Name()
	InstanceFeaturesGroupKind        = schema.GroupKind{Group: Group, Kind: InstanceFeaturesKind}.String()
	InstanceFeaturesKindAPIVersion   = InstanceFeaturesKind + "." + SchemeGroupVersion.String()
	InstanceFeaturesGroupVersionKind = SchemeGroupVersion.WithKind(InstanceFeaturesKind)
)

func init() {
	SchemeBuilder.Register(&InstanceFeatures{}, &InstanceFeaturesList{})
}
