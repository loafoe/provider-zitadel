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

// DefaultPasswordComplexityPolicyParameters are the configurable fields of a DefaultPasswordComplexityPolicy.
type DefaultPasswordComplexityPolicyParameters struct {
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

// DefaultPasswordComplexityPolicyObservation is what the instance wide password complexity policy of an organization currently is.
type DefaultPasswordComplexityPolicyObservation struct {
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

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A DefaultPasswordComplexityPolicy is the instance's instance wide password complexity policy.
//
// It is namespaced like every other managed resource here, because that
// is what its ProviderConfig reference needs. What it manages is not:
// there is one lockout policy for the whole instance whichever namespace
// the resource lives in.
//
// set what a new password must contain, instance wide.
//
// It is a singleton: there is one of each for the whole instance. Zitadel
// offers no way to reset an instance wide policy - there is no default
// beneath it - so deleting this resource writes back the value it
// overwrote rather than removing anything.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Default",type="boolean",JSONPath=".status.atProvider.isDefault"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type DefaultPasswordComplexityPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DefaultPasswordComplexityPolicySpec   `json:"spec"`
	Status DefaultPasswordComplexityPolicyStatus `json:"status,omitempty"`
}

// DefaultPasswordComplexityPolicySpec defines the desired state of a DefaultPasswordComplexityPolicy.
type DefaultPasswordComplexityPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              DefaultPasswordComplexityPolicyParameters `json:"forProvider"`
}

// DefaultPasswordComplexityPolicyStatus is the observed state of a DefaultPasswordComplexityPolicy.
type DefaultPasswordComplexityPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DefaultPasswordComplexityPolicyObservation `json:"atProvider,omitempty"`
}

// DefaultPasswordComplexityPolicyList contains a list of DefaultPasswordComplexityPolicy.
//
// +kubebuilder:object:root=true
type DefaultPasswordComplexityPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefaultPasswordComplexityPolicy `json:"items"`
}

// DefaultPasswordComplexityPolicy type metadata.
var (
	DefaultPasswordComplexityPolicyKind             = reflect.TypeOf(DefaultPasswordComplexityPolicy{}).Name()
	DefaultPasswordComplexityPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: DefaultPasswordComplexityPolicyKind}.String()
	DefaultPasswordComplexityPolicyKindAPIVersion   = DefaultPasswordComplexityPolicyKind + "." + SchemeGroupVersion.String()
	DefaultPasswordComplexityPolicyGroupVersionKind = SchemeGroupVersion.WithKind(DefaultPasswordComplexityPolicyKind)
)

func init() {
	SchemeBuilder.Register(&DefaultPasswordComplexityPolicy{}, &DefaultPasswordComplexityPolicyList{})
}

// SetPolicyScope does nothing: an instance wide policy has no organization.
func (mg *DefaultPasswordComplexityPolicy) SetPolicyScope(string) {}

// PolicyScope is always empty, which is what tells the harness that this
// policy belongs to the instance rather than to an organization.
func (mg *DefaultPasswordComplexityPolicy) PolicyScope() string { return "" }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *DefaultPasswordComplexityPolicy) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *DefaultPasswordComplexityPolicy) PolicyRestore() []byte {
	return mg.Status.AtProvider.Restore
}
