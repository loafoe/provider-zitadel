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

// DefaultLoginPolicyParameters are the configurable fields of a DefaultLoginPolicy.
type DefaultLoginPolicyParameters struct {
	// Allow login with a username and password.
	// +optional
	AllowUsernamePassword *bool `json:"allowUsernamePassword,omitempty"`

	// Allow users to register themselves.
	// +optional
	AllowRegister *bool `json:"allowRegister,omitempty"`

	// Allow login through an upstream identity provider.
	// +optional
	AllowExternalIDP *bool `json:"allowExternalIDP,omitempty"`

	// Require multi factor authentication for every user.
	// +optional
	ForceMFA *bool `json:"forceMFA,omitempty"`

	// Require MFA also for federated logins.
	// +optional
	ForceMFALocalOnly *bool `json:"forceMFALocalOnly,omitempty"`

	// Hide the password reset option from the login UI.
	// +optional
	HidePasswordReset *bool `json:"hidePasswordReset,omitempty"`

	// Avoid leaking whether an account exists.
	// +optional
	IgnoreUnknownUsernames *bool `json:"ignoreUnknownUsernames,omitempty"`

	// Let a login name without a domain be matched against verified domains.
	// +optional
	AllowDomainDiscovery *bool `json:"allowDomainDiscovery,omitempty"`

	// Disable login with an email address.
	// +optional
	DisableLoginWithEmail *bool `json:"disableLoginWithEmail,omitempty"`

	// Disable login with a phone number.
	// +optional
	DisableLoginWithPhone *bool `json:"disableLoginWithPhone,omitempty"`

	// Where a login lands when no redirect is requested.
	// +optional
	DefaultRedirectURI *string `json:"defaultRedirectURI,omitempty"`

	// Whether passwordless login is allowed. Defaults to `NotAllowed`.
	// +optional
	PasswordlessType *PasswordlessType `json:"passwordlessType,omitempty"`

	// How long a password change is accepted for, such as `24h`.
	// +optional
	PasswordCheckLifetime *string `json:"passwordCheckLifetime,omitempty"`

	// How long an upstream IdP link stays valid.
	// +optional
	ExternalLoginCheckLifetime *string `json:"externalLoginCheckLifetime,omitempty"`

	// How long MFA setup may be skipped.
	// +optional
	MFAInitSkipLifetime *string `json:"mfaInitSkipLifetime,omitempty"`

	// How long a second factor stays valid.
	// +optional
	SecondFactorCheckLifetime *string `json:"secondFactorCheckLifetime,omitempty"`

	// How long a multi factor stays valid.
	// +optional
	MultiFactorCheckLifetime *string `json:"multiFactorCheckLifetime,omitempty"`

	// The second factors users may use.
	// +optional
	SecondFactors []SecondFactor `json:"secondFactors,omitempty"`

	// The multi factors users may use.
	// +optional
	MultiFactors []MultiFactor `json:"multiFactors,omitempty"`
}

// DefaultLoginPolicyObservation is what the instance wide login policy of an organization currently is.
type DefaultLoginPolicyObservation struct {
	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// Allow login with a username and password.
	// +optional
	AllowUsernamePassword *bool `json:"allowUsernamePassword,omitempty"`

	// Allow users to register themselves.
	// +optional
	AllowRegister *bool `json:"allowRegister,omitempty"`

	// Allow login through an upstream identity provider.
	// +optional
	AllowExternalIDP *bool `json:"allowExternalIDP,omitempty"`

	// Require multi factor authentication for every user.
	// +optional
	ForceMFA *bool `json:"forceMFA,omitempty"`

	// Require MFA also for federated logins.
	// +optional
	ForceMFALocalOnly *bool `json:"forceMFALocalOnly,omitempty"`

	// Hide the password reset option from the login UI.
	// +optional
	HidePasswordReset *bool `json:"hidePasswordReset,omitempty"`

	// Avoid leaking whether an account exists.
	// +optional
	IgnoreUnknownUsernames *bool `json:"ignoreUnknownUsernames,omitempty"`

	// Let a login name without a domain be matched against verified domains.
	// +optional
	AllowDomainDiscovery *bool `json:"allowDomainDiscovery,omitempty"`

	// Disable login with an email address.
	// +optional
	DisableLoginWithEmail *bool `json:"disableLoginWithEmail,omitempty"`

	// Disable login with a phone number.
	// +optional
	DisableLoginWithPhone *bool `json:"disableLoginWithPhone,omitempty"`

	// Where a login lands when no redirect is requested.
	// +optional
	DefaultRedirectURI *string `json:"defaultRedirectURI,omitempty"`

	// Whether passwordless login is allowed. Defaults to `NotAllowed`.
	// +optional
	PasswordlessType *PasswordlessType `json:"passwordlessType,omitempty"`

	// How long a password change is accepted for, such as `24h`.
	// +optional
	PasswordCheckLifetime *string `json:"passwordCheckLifetime,omitempty"`

	// How long an upstream IdP link stays valid.
	// +optional
	ExternalLoginCheckLifetime *string `json:"externalLoginCheckLifetime,omitempty"`

	// How long MFA setup may be skipped.
	// +optional
	MFAInitSkipLifetime *string `json:"mfaInitSkipLifetime,omitempty"`

	// How long a second factor stays valid.
	// +optional
	SecondFactorCheckLifetime *string `json:"secondFactorCheckLifetime,omitempty"`

	// How long a multi factor stays valid.
	// +optional
	MultiFactorCheckLifetime *string `json:"multiFactorCheckLifetime,omitempty"`

	// The second factors users may use.
	// +optional
	SecondFactors []SecondFactor `json:"secondFactors,omitempty"`

	// The multi factors users may use.
	// +optional
	MultiFactors []MultiFactor `json:"multiFactors,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A DefaultLoginPolicy is the instance's instance wide login policy.
//
// It is namespaced like every other managed resource here, because that
// is what its ProviderConfig reference needs. What it manages is not:
// there is one lockout policy for the whole instance whichever namespace
// the resource lives in.
//
// set what an organization allows until it customises its own login
// policy.
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
type DefaultLoginPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DefaultLoginPolicySpec   `json:"spec"`
	Status DefaultLoginPolicyStatus `json:"status,omitempty"`
}

// DefaultLoginPolicySpec defines the desired state of a DefaultLoginPolicy.
type DefaultLoginPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              DefaultLoginPolicyParameters `json:"forProvider"`
}

// DefaultLoginPolicyStatus is the observed state of a DefaultLoginPolicy.
type DefaultLoginPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DefaultLoginPolicyObservation `json:"atProvider,omitempty"`
}

// DefaultLoginPolicyList contains a list of DefaultLoginPolicy.
//
// +kubebuilder:object:root=true
type DefaultLoginPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefaultLoginPolicy `json:"items"`
}

// DefaultLoginPolicy type metadata.
var (
	DefaultLoginPolicyKind             = reflect.TypeOf(DefaultLoginPolicy{}).Name()
	DefaultLoginPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: DefaultLoginPolicyKind}.String()
	DefaultLoginPolicyKindAPIVersion   = DefaultLoginPolicyKind + "." + SchemeGroupVersion.String()
	DefaultLoginPolicyGroupVersionKind = SchemeGroupVersion.WithKind(DefaultLoginPolicyKind)
)

func init() {
	SchemeBuilder.Register(&DefaultLoginPolicy{}, &DefaultLoginPolicyList{})
}

// SetPolicyScope does nothing: an instance wide policy has no organization.
func (mg *DefaultLoginPolicy) SetPolicyScope(string) {}

// PolicyScope is always empty, which is what tells the harness that this
// policy belongs to the instance rather than to an organization.
func (mg *DefaultLoginPolicy) PolicyScope() string { return "" }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *DefaultLoginPolicy) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *DefaultLoginPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
