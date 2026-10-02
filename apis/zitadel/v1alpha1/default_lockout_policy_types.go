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

// DefaultLockoutPolicyParameters are the configurable fields of a DefaultLockoutPolicy.
type DefaultLockoutPolicyParameters struct {
	// Maximum password check attempts before the account is locked. Attempts
	// reset as soon as the password is checked correctly.
	// +optional
	MaxPasswordAttempts *int64 `json:"maxPasswordAttempts,omitempty"`

	// Maximum OTP check attempts before the account is locked. Attempts reset as
	// soon as an OTP is checked correctly.
	// +optional
	MaxOTPAttempts *int64 `json:"maxOTPAttempts,omitempty"`
}

// DefaultLockoutPolicyObservation is what the instance wide lockout policy of an organization currently is.
type DefaultLockoutPolicyObservation struct {
	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// Maximum password check attempts before the account is locked. Attempts
	// reset as soon as the password is checked correctly.
	// +optional
	MaxPasswordAttempts *int64 `json:"maxPasswordAttempts,omitempty"`

	// Maximum OTP check attempts before the account is locked. Attempts reset as
	// soon as an OTP is checked correctly.
	// +optional
	MaxOTPAttempts *int64 `json:"maxOTPAttempts,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A DefaultLockoutPolicy is the instance's instance wide lockout policy.
//
// It is namespaced like every other managed resource here, because that
// is what its ProviderConfig reference needs. What it manages is not:
// there is one lockout policy for the whole instance whichever namespace
// the resource lives in.
//
// lock out a user of any organization after too many failed attempts.
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
type DefaultLockoutPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DefaultLockoutPolicySpec   `json:"spec"`
	Status DefaultLockoutPolicyStatus `json:"status,omitempty"`
}

// DefaultLockoutPolicySpec defines the desired state of a DefaultLockoutPolicy.
type DefaultLockoutPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              DefaultLockoutPolicyParameters `json:"forProvider"`
}

// DefaultLockoutPolicyStatus is the observed state of a DefaultLockoutPolicy.
type DefaultLockoutPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DefaultLockoutPolicyObservation `json:"atProvider,omitempty"`
}

// DefaultLockoutPolicyList contains a list of DefaultLockoutPolicy.
//
// +kubebuilder:object:root=true
type DefaultLockoutPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefaultLockoutPolicy `json:"items"`
}

// DefaultLockoutPolicy type metadata.
var (
	DefaultLockoutPolicyKind             = reflect.TypeOf(DefaultLockoutPolicy{}).Name()
	DefaultLockoutPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: DefaultLockoutPolicyKind}.String()
	DefaultLockoutPolicyKindAPIVersion   = DefaultLockoutPolicyKind + "." + SchemeGroupVersion.String()
	DefaultLockoutPolicyGroupVersionKind = SchemeGroupVersion.WithKind(DefaultLockoutPolicyKind)
)

func init() {
	SchemeBuilder.Register(&DefaultLockoutPolicy{}, &DefaultLockoutPolicyList{})
}

// SetPolicyScope does nothing: an instance wide policy has no organization.
func (mg *DefaultLockoutPolicy) SetPolicyScope(string) {}

// PolicyScope is always empty, which is what tells the harness that this
// policy belongs to the instance rather than to an organization.
func (mg *DefaultLockoutPolicy) PolicyScope() string { return "" }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *DefaultLockoutPolicy) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *DefaultLockoutPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
