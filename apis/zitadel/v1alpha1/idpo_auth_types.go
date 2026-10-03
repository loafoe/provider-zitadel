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

// IDPOAuthParameters are the configurable fields of an IDPOAuth.
type IDPOAuthParameters struct {
	// Where the browser is sent to log in.
	// +optional
	AuthorizationEndpoint *string `json:"authorizationEndpoint,omitempty"`

	// Where the code is exchanged for a token.
	// +optional
	TokenEndpoint *string `json:"tokenEndpoint,omitempty"`

	// Where the token is exchanged for the user's profile.
	// +optional
	UserEndpoint *string `json:"userEndpoint,omitempty"`

	// The OAuth client ID issued by the provider.
	// +optional
	ClientID *string `json:"clientID,omitempty"`

	// The OAuth client secret, read from a secret.
	// +optional
	ClientSecret *SecretKeySelector `json:"clientSecretSecretRef,omitempty"`

	// The claim in the profile that identifies the user.
	// +optional
	IDAttribute *string `json:"idAttribute,omitempty"`

	// Use proof key for code exchange.
	// +optional
	UsePKCE *bool `json:"usePKCE,omitempty"`

	// Name of the provider, as it appears on the login page.
	// +optional
	Name *string `json:"name,omitempty"`

	// Whether the provider is offered on the login page. One of `Active` or
	// `Inactive`. An inactive provider keeps its settings and its user links.
	// Defaults to `Active`.
	// +optional
	State *IDPState `json:"state,omitempty"`

	// The OAuth scopes requested from the provider, such as `openid email
	// profile`.
	// +optional
	Scopes []string `json:"scopes,omitempty"`

	// Let an existing Zitadel account be linked to a new external one that has
	// the same email address.
	// +optional
	IsLinkingAllowed *bool `json:"isLinkingAllowed,omitempty"`

	// Let a user be created on first login through this provider.
	// +optional
	IsCreationAllowed *bool `json:"isCreationAllowed,omitempty"`

	// Create the user without asking, when the provider says the email address is
	// verified. Needs `isCreationAllowed`.
	// +optional
	IsAutoCreation *bool `json:"isAutoCreation,omitempty"`

	// Refresh a linked user's profile on every login, rather than only when they
	// log in to an existing account.
	// +optional
	IsAutoUpdate *bool `json:"isAutoUpdate,omitempty"`

	// How an account is matched without asking: by `Email` or by `Username`.
	// Needs `isLinkingAllowed`.
	// +optional
	AutoLinking *IDPAutoLinking `json:"autoLinking,omitempty"`
}

// Every identity provider reports the same things, and reports a
// configuration only for an OIDC or a JWT one, so all twenty three kinds
// share one observation type rather than each declaring an identical copy.
type IDPOAuthObservation = IDPObservation

// An IDPOAuth lets users log in through a generic OAuth 2.0 provider.
//
// It is available to every organization that has not set up its own. Deleting
// it removes it from the instance.
//
// Zitadel does not report this kind's settings back, so they are applied once
// at creation and are not drift detected: changing one means deleting the
// provider and making it again.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type IDPOAuth struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IDPOAuthSpec   `json:"spec"`
	Status IDPOAuthStatus `json:"status,omitempty"`
}

// IDPOAuthSpec defines the desired state of an IDPOAuth.
type IDPOAuthSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              IDPOAuthParameters `json:"forProvider"`
}

// IDPOAuthStatus is the observed state of an IDPOAuth.
type IDPOAuthStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          IDPOAuthObservation `json:"atProvider,omitempty"`
}

// IDPOAuthList contains a list of IDPOAuth.
//
// +kubebuilder:object:root=true
type IDPOAuthList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IDPOAuth `json:"items"`
}

// IDPOAuth type metadata.
var (
	IDPOAuthKind             = reflect.TypeOf(IDPOAuth{}).Name()
	IDPOAuthGroupKind        = schema.GroupKind{Group: Group, Kind: IDPOAuthKind}.String()
	IDPOAuthKindAPIVersion   = IDPOAuthKind + "." + SchemeGroupVersion.String()
	IDPOAuthGroupVersionKind = SchemeGroupVersion.WithKind(IDPOAuthKind)
)

func init() {
	SchemeBuilder.Register(&IDPOAuth{}, &IDPOAuthList{})
}

// GetIsLinkingAllowed returns the setting every identity provider carries.
func (mg *IDPOAuth) GetIsLinkingAllowed() *bool { return mg.Spec.ForProvider.IsLinkingAllowed }

// GetIsCreationAllowed returns the setting every identity provider carries.
func (mg *IDPOAuth) GetIsCreationAllowed() *bool { return mg.Spec.ForProvider.IsCreationAllowed }

// GetIsAutoCreation returns the setting every identity provider carries.
func (mg *IDPOAuth) GetIsAutoCreation() *bool { return mg.Spec.ForProvider.IsAutoCreation }

// GetIsAutoUpdate returns the setting every identity provider carries.
func (mg *IDPOAuth) GetIsAutoUpdate() *bool { return mg.Spec.ForProvider.IsAutoUpdate }

// GetAutoLinking returns the setting every identity provider carries.
func (mg *IDPOAuth) GetAutoLinking() *IDPAutoLinking { return mg.Spec.ForProvider.AutoLinking }

// GetState returns the setting every identity provider carries.
func (mg *IDPOAuth) GetState() *IDPState { return mg.Spec.ForProvider.State }

// SetIDPOrganization records the organization the provider belongs to, or the
// empty string for an instance wide one.
func (mg *IDPOAuth) SetIDPOrganization(orgID string) { mg.Status.AtProvider.Scope = StringPtr(orgID) }

// IDPOrganization returns the organization the provider was last read from.
func (mg *IDPOAuth) IDPOrganization() string { return Deref(mg.Status.AtProvider.Scope) }
