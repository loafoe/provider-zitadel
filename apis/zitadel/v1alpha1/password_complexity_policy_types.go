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

// PasswordComplexityPolicyParameters are the configurable fields of a PasswordComplexityPolicy.
type PasswordComplexityPolicyParameters struct {
	// The ID of the organization the policy belongs to. Either `organizationID`,
	// `organizationRef` or `organizationSelector` must be set.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// OrganizationRef references an Organization managed by this provider and
	// uses its ID.
	// +optional
	OrganizationRef *xpv1.Reference `json:"organizationRef,omitempty"`

	// OrganizationSelector selects an Organization managed by this provider and
	// uses its ID.
	// +optional
	OrganizationSelector *xpv1.Selector `json:"organizationSelector,omitempty"`

	// Minimum password length.
	// +optional
	MinLength *int64 `json:"minLength,omitempty"`

	// Require an uppercase letter.
	// +optional
	HasUppercase *bool `json:"hasUppercase,omitempty"`

	// Require a lowercase letter.
	// +optional
	HasLowercase *bool `json:"hasLowercase,omitempty"`

	// Require a number.
	// +optional
	HasNumber *bool `json:"hasNumber,omitempty"`

	// Require a symbol, such as `$`.
	// +optional
	HasSymbol *bool `json:"hasSymbol,omitempty"`
}

// PasswordComplexityPolicyObservation is what the password complexity policy of an organization currently is.
type PasswordComplexityPolicyObservation struct {
	// OrganizationID is the organization the policy belongs to.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// Minimum password length.
	// +optional
	MinLength *int64 `json:"minLength,omitempty"`

	// Require an uppercase letter.
	// +optional
	HasUppercase *bool `json:"hasUppercase,omitempty"`

	// Require a lowercase letter.
	// +optional
	HasLowercase *bool `json:"hasLowercase,omitempty"`

	// Require a number.
	// +optional
	HasNumber *bool `json:"hasNumber,omitempty"`

	// Require a symbol, such as `$`.
	// +optional
	HasSymbol *bool `json:"hasSymbol,omitempty"`

	// The organization the policy was last read from.
	// +optional
	Scope *string `json:"scope,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A PasswordComplexityPolicy is an organization's password complexity policy.
//
// controls what a new password must contain.
//
// It is a singleton: one resource describes the policy of one
// organization, and deleting it puts the organization back on the
// instance default rather than removing anything.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Organization",type="string",JSONPath=".status.atProvider.organizationID"
// +kubebuilder:printcolumn:name="Default",type="boolean",JSONPath=".status.atProvider.isDefault"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type PasswordComplexityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PasswordComplexityPolicySpec   `json:"spec"`
	Status PasswordComplexityPolicyStatus `json:"status,omitempty"`
}

// PasswordComplexityPolicySpec defines the desired state of a PasswordComplexityPolicy.
type PasswordComplexityPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              PasswordComplexityPolicyParameters `json:"forProvider"`
}

// PasswordComplexityPolicyStatus is the observed state of a PasswordComplexityPolicy.
type PasswordComplexityPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PasswordComplexityPolicyObservation `json:"atProvider,omitempty"`
}

// PasswordComplexityPolicyList contains a list of PasswordComplexityPolicy.
//
// +kubebuilder:object:root=true
type PasswordComplexityPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PasswordComplexityPolicy `json:"items"`
}

// PasswordComplexityPolicy type metadata.
var (
	PasswordComplexityPolicyKind             = reflect.TypeOf(PasswordComplexityPolicy{}).Name()
	PasswordComplexityPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: PasswordComplexityPolicyKind}.String()
	PasswordComplexityPolicyKindAPIVersion   = PasswordComplexityPolicyKind + "." + SchemeGroupVersion.String()
	PasswordComplexityPolicyGroupVersionKind = SchemeGroupVersion.WithKind(PasswordComplexityPolicyKind)
)

func init() {
	SchemeBuilder.Register(&PasswordComplexityPolicy{}, &PasswordComplexityPolicyList{})
}

// SetPolicyScope records the organization the policy belongs to.
func (mg *PasswordComplexityPolicy) SetPolicyScope(scope string) {
	mg.Status.AtProvider.Scope = StringPtr(scope)
}

// PolicyScope returns the organization the policy was last read from.
func (mg *PasswordComplexityPolicy) PolicyScope() string { return Deref(mg.Status.AtProvider.Scope) }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *PasswordComplexityPolicy) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *PasswordComplexityPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
