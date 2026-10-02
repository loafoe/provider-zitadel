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

// PasswordAgePolicyParameters are the configurable fields of a PasswordAgePolicy.
type PasswordAgePolicyParameters struct {
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

	// Days after which a password expires. Zitadel treats 0 as "never".
	// +optional
	MaxAgeDays *int64 `json:"maxAgeDays,omitempty"`

	// Days before expiry at which the user is warned. Setting this without
	// `maxAgeDays` has no effect.
	// +optional
	ExpireWarnDays *int64 `json:"expireWarnDays,omitempty"`
}

// PasswordAgePolicyObservation is what the password expiry policy of an organization currently is.
type PasswordAgePolicyObservation struct {
	// OrganizationID is the organization the policy belongs to.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// Days after which a password expires. Zitadel treats 0 as "never".
	// +optional
	MaxAgeDays *int64 `json:"maxAgeDays,omitempty"`

	// Days before expiry at which the user is warned. Setting this without
	// `maxAgeDays` has no effect.
	// +optional
	ExpireWarnDays *int64 `json:"expireWarnDays,omitempty"`

	// The organization the policy was last read from.
	// +optional
	Scope *string `json:"scope,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A PasswordAgePolicy is an organization's password expiry policy.
//
// controls when a password has to be changed.
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
type PasswordAgePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PasswordAgePolicySpec   `json:"spec"`
	Status PasswordAgePolicyStatus `json:"status,omitempty"`
}

// PasswordAgePolicySpec defines the desired state of a PasswordAgePolicy.
type PasswordAgePolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              PasswordAgePolicyParameters `json:"forProvider"`
}

// PasswordAgePolicyStatus is the observed state of a PasswordAgePolicy.
type PasswordAgePolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PasswordAgePolicyObservation `json:"atProvider,omitempty"`
}

// PasswordAgePolicyList contains a list of PasswordAgePolicy.
//
// +kubebuilder:object:root=true
type PasswordAgePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PasswordAgePolicy `json:"items"`
}

// PasswordAgePolicy type metadata.
var (
	PasswordAgePolicyKind             = reflect.TypeOf(PasswordAgePolicy{}).Name()
	PasswordAgePolicyGroupKind        = schema.GroupKind{Group: Group, Kind: PasswordAgePolicyKind}.String()
	PasswordAgePolicyKindAPIVersion   = PasswordAgePolicyKind + "." + SchemeGroupVersion.String()
	PasswordAgePolicyGroupVersionKind = SchemeGroupVersion.WithKind(PasswordAgePolicyKind)
)

func init() {
	SchemeBuilder.Register(&PasswordAgePolicy{}, &PasswordAgePolicyList{})
}

// SetPolicyScope records the organization the policy belongs to.
func (mg *PasswordAgePolicy) SetPolicyScope(scope string) {
	mg.Status.AtProvider.Scope = StringPtr(scope)
}

// PolicyScope returns the organization the policy was last read from.
func (mg *PasswordAgePolicy) PolicyScope() string { return Deref(mg.Status.AtProvider.Scope) }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *PasswordAgePolicy) SetPolicyRestore(restore []byte) { mg.Status.AtProvider.Restore = restore }

// PolicyRestore returns the recorded restore point.
func (mg *PasswordAgePolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
