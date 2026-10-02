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

// DefaultDomainPolicyParameters are the configurable fields of a DefaultDomainPolicy.
type DefaultDomainPolicyParameters struct {
	// Require login names to be qualified with a domain, such as
	// `alice@example.com` rather than `alice`.
	// +optional
	UserLoginMustBeDomain *bool `json:"userLoginMustBeDomain,omitempty"`

	// Verify that an organization's domains resolve before accepting them.
	// Zitadel rejects the change until they do.
	// +optional
	ValidateOrgDomains *bool `json:"validateOrgDomains,omitempty"`

	// Require the SMTP sender address to match the instance domain.
	// +optional
	SMTPSenderAddressMatchesInstanceDomain *bool `json:"smtpSenderAddressMatchesInstanceDomain,omitempty"`
}

// DefaultDomainPolicyObservation is what the instance wide domain policy of an organization currently is.
type DefaultDomainPolicyObservation struct {
	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// Require login names to be qualified with a domain, such as
	// `alice@example.com` rather than `alice`.
	// +optional
	UserLoginMustBeDomain *bool `json:"userLoginMustBeDomain,omitempty"`

	// Verify that an organization's domains resolve before accepting them.
	// Zitadel rejects the change until they do.
	// +optional
	ValidateOrgDomains *bool `json:"validateOrgDomains,omitempty"`

	// Require the SMTP sender address to match the instance domain.
	// +optional
	SMTPSenderAddressMatchesInstanceDomain *bool `json:"smtpSenderAddressMatchesInstanceDomain,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A DefaultDomainPolicy is the instance's instance wide domain policy.
//
// It is namespaced like every other managed resource here, because that
// is what its ProviderConfig reference needs. What it manages is not:
// there is one lockout policy for the whole instance whichever namespace
// the resource lives in.
//
// control how login names relate to domains, instance wide.
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
type DefaultDomainPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DefaultDomainPolicySpec   `json:"spec"`
	Status DefaultDomainPolicyStatus `json:"status,omitempty"`
}

// DefaultDomainPolicySpec defines the desired state of a DefaultDomainPolicy.
type DefaultDomainPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              DefaultDomainPolicyParameters `json:"forProvider"`
}

// DefaultDomainPolicyStatus is the observed state of a DefaultDomainPolicy.
type DefaultDomainPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DefaultDomainPolicyObservation `json:"atProvider,omitempty"`
}

// DefaultDomainPolicyList contains a list of DefaultDomainPolicy.
//
// +kubebuilder:object:root=true
type DefaultDomainPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefaultDomainPolicy `json:"items"`
}

// DefaultDomainPolicy type metadata.
var (
	DefaultDomainPolicyKind             = reflect.TypeOf(DefaultDomainPolicy{}).Name()
	DefaultDomainPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: DefaultDomainPolicyKind}.String()
	DefaultDomainPolicyKindAPIVersion   = DefaultDomainPolicyKind + "." + SchemeGroupVersion.String()
	DefaultDomainPolicyGroupVersionKind = SchemeGroupVersion.WithKind(DefaultDomainPolicyKind)
)

func init() {
	SchemeBuilder.Register(&DefaultDomainPolicy{}, &DefaultDomainPolicyList{})
}

// SetPolicyScope does nothing: an instance wide policy has no organization.
func (mg *DefaultDomainPolicy) SetPolicyScope(string) {}

// PolicyScope is always empty, which is what tells the harness that this
// policy belongs to the instance rather than to an organization.
func (mg *DefaultDomainPolicy) PolicyScope() string { return "" }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *DefaultDomainPolicy) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *DefaultDomainPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
