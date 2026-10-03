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

// IDPOIDCParameters are the configurable fields of an IDPOIDC.
type IDPOIDCParameters struct {
	// The issuer URL. Zitadel discovers the provider's endpoints from it.
	// +optional
	Issuer *string `json:"issuer,omitempty"`

	// The OAuth client ID issued by the provider.
	// +optional
	ClientID *string `json:"clientID,omitempty"`

	// The OAuth client secret, read from a secret. Zitadel never returns it, so
	// it is applied but never compared.
	// +optional
	ClientSecret *SecretKeySelector `json:"clientSecretSecretRef,omitempty"`

	// Build the user's profile from the ID token instead of calling the userinfo
	// endpoint.
	// +optional
	IsIDTokenMapping *bool `json:"isIDTokenMapping,omitempty"`

	// Use proof key for code exchange, which public clients require.
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
type IDPOIDCObservation = IDPObservation

// An IDPOIDC lets users log in through any OpenID Connect provider.
//
// It is available to every organization that has not set up its own. Deleting
// it removes it from the instance.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type IDPOIDC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IDPOIDCSpec   `json:"spec"`
	Status IDPOIDCStatus `json:"status,omitempty"`
}

// IDPOIDCSpec defines the desired state of an IDPOIDC.
type IDPOIDCSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              IDPOIDCParameters `json:"forProvider"`
}

// IDPOIDCStatus is the observed state of an IDPOIDC.
type IDPOIDCStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          IDPOIDCObservation `json:"atProvider,omitempty"`
}

// IDPOIDCList contains a list of IDPOIDC.
//
// +kubebuilder:object:root=true
type IDPOIDCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IDPOIDC `json:"items"`
}

// IDPOIDC type metadata.
var (
	IDPOIDCKind             = reflect.TypeOf(IDPOIDC{}).Name()
	IDPOIDCGroupKind        = schema.GroupKind{Group: Group, Kind: IDPOIDCKind}.String()
	IDPOIDCKindAPIVersion   = IDPOIDCKind + "." + SchemeGroupVersion.String()
	IDPOIDCGroupVersionKind = SchemeGroupVersion.WithKind(IDPOIDCKind)
)

func init() {
	SchemeBuilder.Register(&IDPOIDC{}, &IDPOIDCList{})
}

// GetIsLinkingAllowed returns the setting every identity provider carries.
func (mg *IDPOIDC) GetIsLinkingAllowed() *bool { return mg.Spec.ForProvider.IsLinkingAllowed }

// GetIsCreationAllowed returns the setting every identity provider carries.
func (mg *IDPOIDC) GetIsCreationAllowed() *bool { return mg.Spec.ForProvider.IsCreationAllowed }

// GetIsAutoCreation returns the setting every identity provider carries.
func (mg *IDPOIDC) GetIsAutoCreation() *bool { return mg.Spec.ForProvider.IsAutoCreation }

// GetIsAutoUpdate returns the setting every identity provider carries.
func (mg *IDPOIDC) GetIsAutoUpdate() *bool { return mg.Spec.ForProvider.IsAutoUpdate }

// GetAutoLinking returns the setting every identity provider carries.
func (mg *IDPOIDC) GetAutoLinking() *IDPAutoLinking { return mg.Spec.ForProvider.AutoLinking }

// GetState returns the setting every identity provider carries.
func (mg *IDPOIDC) GetState() *IDPState { return mg.Spec.ForProvider.State }

// SetIDPOrganization records the organization the provider belongs to, or the
// empty string for an instance wide one.
func (mg *IDPOIDC) SetIDPOrganization(orgID string) { mg.Status.AtProvider.Scope = StringPtr(orgID) }

// IDPOrganization returns the organization the provider was last read from.
func (mg *IDPOIDC) IDPOrganization() string { return Deref(mg.Status.AtProvider.Scope) }
