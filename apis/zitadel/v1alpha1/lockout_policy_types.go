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

// LockoutPolicyParameters are the configurable fields of a LockoutPolicy.
type LockoutPolicyParameters struct {
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

	// Maximum password check attempts before the account is locked. Attempts
	// reset as soon as the password is checked correctly.
	// +optional
	MaxPasswordAttempts *int64 `json:"maxPasswordAttempts,omitempty"`

	// Maximum OTP check attempts before the account is locked. Attempts reset as
	// soon as an OTP is checked correctly.
	// +optional
	MaxOTPAttempts *int64 `json:"maxOTPAttempts,omitempty"`
}

// LockoutPolicyObservation is what the lockout policy of an organization currently is.
type LockoutPolicyObservation struct {
	// OrganizationID is the organization the policy belongs to.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

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

	// The organization the policy was last read from.
	// +optional
	Scope *string `json:"scope,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A LockoutPolicy is an organization's lockout policy.
//
// controls how many failed attempts lock a user out.
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
type LockoutPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LockoutPolicySpec   `json:"spec"`
	Status LockoutPolicyStatus `json:"status,omitempty"`
}

// LockoutPolicySpec defines the desired state of a LockoutPolicy.
type LockoutPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              LockoutPolicyParameters `json:"forProvider"`
}

// LockoutPolicyStatus is the observed state of a LockoutPolicy.
type LockoutPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          LockoutPolicyObservation `json:"atProvider,omitempty"`
}

// LockoutPolicyList contains a list of LockoutPolicy.
//
// +kubebuilder:object:root=true
type LockoutPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LockoutPolicy `json:"items"`
}

// LockoutPolicy type metadata.
var (
	LockoutPolicyKind             = reflect.TypeOf(LockoutPolicy{}).Name()
	LockoutPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: LockoutPolicyKind}.String()
	LockoutPolicyKindAPIVersion   = LockoutPolicyKind + "." + SchemeGroupVersion.String()
	LockoutPolicyGroupVersionKind = SchemeGroupVersion.WithKind(LockoutPolicyKind)
)

func init() {
	SchemeBuilder.Register(&LockoutPolicy{}, &LockoutPolicyList{})
}

// SetPolicyScope records the organization the policy belongs to.
func (mg *LockoutPolicy) SetPolicyScope(scope string) { mg.Status.AtProvider.Scope = StringPtr(scope) }

// PolicyScope returns the organization the policy was last read from.
func (mg *LockoutPolicy) PolicyScope() string { return Deref(mg.Status.AtProvider.Scope) }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *LockoutPolicy) SetPolicyRestore(restore []byte) { mg.Status.AtProvider.Restore = restore }

// PolicyRestore returns the recorded restore point.
func (mg *LockoutPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
