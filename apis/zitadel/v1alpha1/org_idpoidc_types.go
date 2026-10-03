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

// OrgIDPOIDCParameters are the configurable fields of an OrgIDPOIDC.
type OrgIDPOIDCParameters struct {
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

	// The ID of the organization the provider belongs to. Either
	// `organizationID`, `organizationRef` or `organizationSelector` must be set.
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
type OrgIDPOIDCObservation = IDPObservation

// An OrgIDPOIDC lets users log in through any OpenID Connect provider.
//
// It belongs to one organization. Deleting it removes the provider from that
// organization; Zitadel refuses to remove one that is still offered on a
// login page, so it has to be unbound from the login policy first.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Organization",type="string",JSONPath=".status.atProvider.organizationID"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type OrgIDPOIDC struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrgIDPOIDCSpec   `json:"spec"`
	Status OrgIDPOIDCStatus `json:"status,omitempty"`
}

// OrgIDPOIDCSpec defines the desired state of an OrgIDPOIDC.
type OrgIDPOIDCSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OrgIDPOIDCParameters `json:"forProvider"`
}

// OrgIDPOIDCStatus is the observed state of an OrgIDPOIDC.
type OrgIDPOIDCStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrgIDPOIDCObservation `json:"atProvider,omitempty"`
}

// OrgIDPOIDCList contains a list of OrgIDPOIDC.
//
// +kubebuilder:object:root=true
type OrgIDPOIDCList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrgIDPOIDC `json:"items"`
}

// OrgIDPOIDC type metadata.
var (
	OrgIDPOIDCKind             = reflect.TypeOf(OrgIDPOIDC{}).Name()
	OrgIDPOIDCGroupKind        = schema.GroupKind{Group: Group, Kind: OrgIDPOIDCKind}.String()
	OrgIDPOIDCKindAPIVersion   = OrgIDPOIDCKind + "." + SchemeGroupVersion.String()
	OrgIDPOIDCGroupVersionKind = SchemeGroupVersion.WithKind(OrgIDPOIDCKind)
)

func init() {
	SchemeBuilder.Register(&OrgIDPOIDC{}, &OrgIDPOIDCList{})
}

// GetIsLinkingAllowed returns the setting every identity provider carries.
func (mg *OrgIDPOIDC) GetIsLinkingAllowed() *bool { return mg.Spec.ForProvider.IsLinkingAllowed }

// GetIsCreationAllowed returns the setting every identity provider carries.
func (mg *OrgIDPOIDC) GetIsCreationAllowed() *bool { return mg.Spec.ForProvider.IsCreationAllowed }

// GetIsAutoCreation returns the setting every identity provider carries.
func (mg *OrgIDPOIDC) GetIsAutoCreation() *bool { return mg.Spec.ForProvider.IsAutoCreation }

// GetIsAutoUpdate returns the setting every identity provider carries.
func (mg *OrgIDPOIDC) GetIsAutoUpdate() *bool { return mg.Spec.ForProvider.IsAutoUpdate }

// GetAutoLinking returns the setting every identity provider carries.
func (mg *OrgIDPOIDC) GetAutoLinking() *IDPAutoLinking { return mg.Spec.ForProvider.AutoLinking }

// GetState returns the setting every identity provider carries.
func (mg *OrgIDPOIDC) GetState() *IDPState { return mg.Spec.ForProvider.State }

// SetIDPOrganization records the organization the provider belongs to, or the
// empty string for an instance wide one.
func (mg *OrgIDPOIDC) SetIDPOrganization(orgID string) { mg.Status.AtProvider.Scope = StringPtr(orgID) }

// IDPOrganization returns the organization the provider was last read from.
func (mg *OrgIDPOIDC) IDPOrganization() string { return Deref(mg.Status.AtProvider.Scope) }

// GetOrganizationRef returns the reference the provider's organization is
// resolved from.
func (mg *OrgIDPOIDC) GetOrganizationRef() *xpv1.Reference {
	return mg.Spec.ForProvider.OrganizationRef
}

// GetOrganizationSelector returns the selector the provider's organization
// is resolved from.
func (mg *OrgIDPOIDC) GetOrganizationSelector() *xpv1.Selector {
	return mg.Spec.ForProvider.OrganizationSelector
}

// GetOrganizationID returns the pinned organization ID, if any.
func (mg *OrgIDPOIDC) GetOrganizationID() *string { return mg.Spec.ForProvider.OrganizationID }

// GetObservedOrganization returns the organization the provider was last read
// from, which is what a terminating object acts on.
func (mg *OrgIDPOIDC) GetObservedOrganization() *string { return mg.Status.AtProvider.OrganizationID }
