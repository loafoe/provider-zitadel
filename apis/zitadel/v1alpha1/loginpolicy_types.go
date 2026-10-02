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

// LoginPolicyParameters are the configurable fields of a LoginPolicy.
type LoginPolicyParameters struct {
	// OrganizationID is the ID of the organization. Either `organizationID`,
	// `organizationRef` or `organizationSelector` must be set.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// OrganizationRef references an Organization managed by this provider.
	// +optional
	OrganizationRef *xpv1.Reference `json:"organizationRef,omitempty"`

	// OrganizationSelector selects an Organization managed by this provider.
	// +optional
	OrganizationSelector *xpv1.Selector `json:"organizationSelector,omitempty"`

	// AllowUsernamePassword allows login with a username and password.
	// +optional
	AllowUsernamePassword *bool `json:"allowUsernamePassword,omitempty"`

	// AllowRegister allows users to register themselves.
	// +optional
	AllowRegister *bool `json:"allowRegister,omitempty"`

	// AllowExternalIDP allows login through an upstream identity provider.
	// +optional
	AllowExternalIDP *bool `json:"allowExternalIDP,omitempty"`

	// ForceMFA requires multi factor authentication for every user.
	// +optional
	ForceMFA *bool `json:"forceMFA,omitempty"`

	// ForceMFALocalOnly requires MFA also for federated logins.
	// +optional
	ForceMFALocalOnly *bool `json:"forceMFALocalOnly,omitempty"`

	// HidePasswordReset hides the password reset option from the login UI.
	// +optional
	HidePasswordReset *bool `json:"hidePasswordReset,omitempty"`

	// IgnoreUnknownUsernames avoids leaking whether an account exists by
	// answering the same way for an unknown and a wrong username.
	// +optional
	IgnoreUnknownUsernames *bool `json:"ignoreUnknownUsernames,omitempty"`

	// AllowDomainDiscovery lets a login name without a domain be matched
	// against verified domains.
	// +optional
	AllowDomainDiscovery *bool `json:"allowDomainDiscovery,omitempty"`

	// DisableLoginWithEmail disables login with an email address.
	// +optional
	DisableLoginWithEmail *bool `json:"disableLoginWithEmail,omitempty"`

	// DisableLoginWithPhone disables login with a phone number.
	// +optional
	DisableLoginWithPhone *bool `json:"disableLoginWithPhone,omitempty"`

	// DefaultRedirectURI is where a login lands when no redirect is requested.
	// +optional
	DefaultRedirectURI *string `json:"defaultRedirectURI,omitempty"`

	// PasswordlessType selects whether passwordless login is allowed.
	// +optional
	PasswordlessType *PasswordlessType `json:"passwordlessType,omitempty"`

	// PasswordCheckLifetime is how long a password change is accepted for,
	// e.g. "8760h".
	// +optional
	PasswordCheckLifetime *string `json:"passwordCheckLifetime,omitempty"`

	// ExternalLoginCheckLifetime is how long an upstream IdP link stays valid,
	// e.g. "8760h".
	// +optional
	ExternalLoginCheckLifetime *string `json:"externalLoginCheckLifetime,omitempty"`

	// MFAInitSkipLifetime is how long MFA setup may be skipped, e.g. "720h".
	// +optional
	MFAInitSkipLifetime *string `json:"mfaInitSkipLifetime,omitempty"`

	// SecondFactorCheckLifetime is how long a second factor stays valid, e.g.
	// "8760h".
	// +optional
	SecondFactorCheckLifetime *string `json:"secondFactorCheckLifetime,omitempty"`

	// MultiFactorCheckLifetime is how long a multi factor stays valid, e.g.
	// "8760h".
	// +optional
	MultiFactorCheckLifetime *string `json:"multiFactorCheckLifetime,omitempty"`

	// SecondFactors are the second factors users may use.
	// +optional
	SecondFactors []SecondFactor `json:"secondFactors,omitempty"`

	// MultiFactors are the multi factors users may use.
	// +optional
	MultiFactors []MultiFactor `json:"multiFactors,omitempty"`

	// IDPs are the IDs of the identity providers offered on the login page.
	// +optional
	IDPs []string `json:"idps,omitempty"`
}

// LoginPolicyObservation are the observable fields of a LoginPolicy.
type LoginPolicyObservation struct {
	// OrganizationID is the organization the policy belongs to.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// IsDefault is true while the organization still uses the instance default
	// policy.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// AllowUsernamePassword allows login with a username and password.
	// +optional
	AllowUsernamePassword *bool `json:"allowUsernamePassword,omitempty"`

	// AllowRegister allows users to register themselves.
	// +optional
	AllowRegister *bool `json:"allowRegister,omitempty"`

	// AllowExternalIDP allows login through an upstream identity provider.
	// +optional
	AllowExternalIDP *bool `json:"allowExternalIDP,omitempty"`

	// ForceMFA requires multi factor authentication for every user.
	// +optional
	ForceMFA *bool `json:"forceMFA,omitempty"`

	// ForceMFALocalOnly requires MFA also for federated logins.
	// +optional
	ForceMFALocalOnly *bool `json:"forceMFALocalOnly,omitempty"`

	// HidePasswordReset hides the password reset option from the login UI.
	// +optional
	HidePasswordReset *bool `json:"hidePasswordReset,omitempty"`

	// IgnoreUnknownUsernames avoids leaking whether an account exists.
	// +optional
	IgnoreUnknownUsernames *bool `json:"ignoreUnknownUsernames,omitempty"`

	// AllowDomainDiscovery lets a login name without a domain be matched
	// against verified domains.
	// +optional
	AllowDomainDiscovery *bool `json:"allowDomainDiscovery,omitempty"`

	// DisableLoginWithEmail disables login with an email address.
	// +optional
	DisableLoginWithEmail *bool `json:"disableLoginWithEmail,omitempty"`

	// DisableLoginWithPhone disables login with a phone number.
	// +optional
	DisableLoginWithPhone *bool `json:"disableLoginWithPhone,omitempty"`

	// DefaultRedirectURI is where a login lands when no redirect is requested.
	// +optional
	DefaultRedirectURI *string `json:"defaultRedirectURI,omitempty"`

	// PasswordlessType selects whether passwordless login is allowed.
	// +optional
	PasswordlessType *PasswordlessType `json:"passwordlessType,omitempty"`

	// PasswordCheckLifetime is how long a password change is accepted for.
	// +optional
	PasswordCheckLifetime *string `json:"passwordCheckLifetime,omitempty"`

	// ExternalLoginCheckLifetime is how long an upstream IdP link stays valid.
	// +optional
	ExternalLoginCheckLifetime *string `json:"externalLoginCheckLifetime,omitempty"`

	// MFAInitSkipLifetime is how long MFA setup may be skipped.
	// +optional
	MFAInitSkipLifetime *string `json:"mfaInitSkipLifetime,omitempty"`

	// SecondFactorCheckLifetime is how long a second factor stays valid.
	// +optional
	SecondFactorCheckLifetime *string `json:"secondFactorCheckLifetime,omitempty"`

	// MultiFactorCheckLifetime is how long a multi factor stays valid.
	// +optional
	MultiFactorCheckLifetime *string `json:"multiFactorCheckLifetime,omitempty"`

	// SecondFactors are the second factors users may use.
	// +optional
	SecondFactors []SecondFactor `json:"secondFactors,omitempty"`

	// MultiFactors are the multi factors users may use.
	// +optional
	MultiFactors []MultiFactor `json:"multiFactors,omitempty"`

	// IDPs are the identity providers offered on the login page.
	// +optional
	IDPs []string `json:"idps,omitempty"`

	// Scope is the organization the policy was last read from.
	// +optional
	Scope *string `json:"scope,omitempty"`

	// Restore is the policy this resource overwrote, kept so that deleting it can
	// put the previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A LoginPolicySpec defines the desired state of a LoginPolicy.
type LoginPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              LoginPolicyParameters `json:"forProvider"`
}

// A LoginPolicyStatus represents the observed state of a LoginPolicy.
type LoginPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          LoginPolicyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="ORGANIZATION",type="string",JSONPath=".status.atProvider.organizationID"
// +kubebuilder:printcolumn:name="DEFAULT",type="boolean",JSONPath=".status.atProvider.isDefault"
// +kubebuilder:printcolumn:name="REGISTER",type="boolean",JSONPath=".status.atProvider.allowRegister"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// // A LoginPolicy is an organization's login and authentication policy: whether
// self registration is allowed, whether MFA is forced, how long a password
// change is accepted for. Zitadel keeps exactly one per organization, so this
// resource is a singleton and deleting it reverts the organization to the
// instance default rather than removing a policy.
type LoginPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoginPolicySpec   `json:"spec"`
	Status LoginPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LoginPolicyList contains a list of LoginPolicy.
type LoginPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoginPolicy `json:"items"`
}

// LoginPolicy type metadata.
var (
	LoginPolicyKind             = reflect.TypeOf(LoginPolicy{}).Name()
	LoginPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: LoginPolicyKind}.String()
	LoginPolicyKindAPIVersion   = LoginPolicyKind + "." + SchemeGroupVersion.String()
	LoginPolicyGroupVersionKind = SchemeGroupVersion.WithKind(LoginPolicyKind)
)

func init() {
	SchemeBuilder.Register(&LoginPolicy{}, &LoginPolicyList{})
}

// SetPolicyScope records the organization the policy belongs to.
func (mg *LoginPolicy) SetPolicyScope(scope string) { mg.Status.AtProvider.Scope = StringPtr(scope) }

// PolicyScope returns the organization the policy was last read from.
func (mg *LoginPolicy) PolicyScope() string { return Deref(mg.Status.AtProvider.Scope) }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *LoginPolicy) SetPolicyRestore(restore []byte) { mg.Status.AtProvider.Restore = restore }

// PolicyRestore returns the recorded restore point.
func (mg *LoginPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
