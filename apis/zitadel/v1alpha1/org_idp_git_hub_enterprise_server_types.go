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

// OrgIDPGitHubEnterpriseServerParameters are the configurable fields of an OrgIDPGitHubEnterpriseServer.
type OrgIDPGitHubEnterpriseServerParameters struct {
	// The OAuth app's client ID.
	// +optional
	ClientID *string `json:"clientID,omitempty"`

	// The OAuth app's client secret, read from a secret.
	// +optional
	ClientSecret *SecretKeySelector `json:"clientSecretSecretRef,omitempty"`

	// Where the browser is sent to log in.
	// +optional
	AuthorizationEndpoint *string `json:"authorizationEndpoint,omitempty"`

	// Where the code is exchanged for a token.
	// +optional
	TokenEndpoint *string `json:"tokenEndpoint,omitempty"`

	// Where the token is exchanged for the user's profile.
	// +optional
	UserEndpoint *string `json:"userEndpoint,omitempty"`

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
type OrgIDPGitHubEnterpriseServerObservation = IDPObservation

// An OrgIDPGitHubEnterpriseServer lets users sign in with a self hosted GitHub Enterprise Server.
//
// It belongs to one organization. Deleting it removes the provider from that
// organization; Zitadel refuses to remove one that is still offered on a
// login page, so it has to be unbound from the login policy first.
//
// Zitadel does not report this kind's settings back, so they are applied once
// at creation and are not drift detected: changing one means deleting the
// provider and making it again.
//
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Organization",type="string",JSONPath=".status.atProvider.organizationID"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type OrgIDPGitHubEnterpriseServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrgIDPGitHubEnterpriseServerSpec   `json:"spec"`
	Status OrgIDPGitHubEnterpriseServerStatus `json:"status,omitempty"`
}

// OrgIDPGitHubEnterpriseServerSpec defines the desired state of an OrgIDPGitHubEnterpriseServer.
type OrgIDPGitHubEnterpriseServerSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OrgIDPGitHubEnterpriseServerParameters `json:"forProvider"`
}

// OrgIDPGitHubEnterpriseServerStatus is the observed state of an OrgIDPGitHubEnterpriseServer.
type OrgIDPGitHubEnterpriseServerStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrgIDPGitHubEnterpriseServerObservation `json:"atProvider,omitempty"`
}

// OrgIDPGitHubEnterpriseServerList contains a list of OrgIDPGitHubEnterpriseServer.
//
// +kubebuilder:object:root=true
type OrgIDPGitHubEnterpriseServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrgIDPGitHubEnterpriseServer `json:"items"`
}

// OrgIDPGitHubEnterpriseServer type metadata.
var (
	OrgIDPGitHubEnterpriseServerKind             = reflect.TypeOf(OrgIDPGitHubEnterpriseServer{}).Name()
	OrgIDPGitHubEnterpriseServerGroupKind        = schema.GroupKind{Group: Group, Kind: OrgIDPGitHubEnterpriseServerKind}.String()
	OrgIDPGitHubEnterpriseServerKindAPIVersion   = OrgIDPGitHubEnterpriseServerKind + "." + SchemeGroupVersion.String()
	OrgIDPGitHubEnterpriseServerGroupVersionKind = SchemeGroupVersion.WithKind(OrgIDPGitHubEnterpriseServerKind)
)

func init() {
	SchemeBuilder.Register(&OrgIDPGitHubEnterpriseServer{}, &OrgIDPGitHubEnterpriseServerList{})
}

// GetIsLinkingAllowed returns the setting every identity provider carries.
func (mg *OrgIDPGitHubEnterpriseServer) GetIsLinkingAllowed() *bool {
	return mg.Spec.ForProvider.IsLinkingAllowed
}

// GetIsCreationAllowed returns the setting every identity provider carries.
func (mg *OrgIDPGitHubEnterpriseServer) GetIsCreationAllowed() *bool {
	return mg.Spec.ForProvider.IsCreationAllowed
}

// GetIsAutoCreation returns the setting every identity provider carries.
func (mg *OrgIDPGitHubEnterpriseServer) GetIsAutoCreation() *bool {
	return mg.Spec.ForProvider.IsAutoCreation
}

// GetIsAutoUpdate returns the setting every identity provider carries.
func (mg *OrgIDPGitHubEnterpriseServer) GetIsAutoUpdate() *bool {
	return mg.Spec.ForProvider.IsAutoUpdate
}

// GetAutoLinking returns the setting every identity provider carries.
func (mg *OrgIDPGitHubEnterpriseServer) GetAutoLinking() *IDPAutoLinking {
	return mg.Spec.ForProvider.AutoLinking
}

// GetState returns the setting every identity provider carries.
func (mg *OrgIDPGitHubEnterpriseServer) GetState() *IDPState { return mg.Spec.ForProvider.State }

// SetIDPOrganization records the organization the provider belongs to, or the
// empty string for an instance wide one.
func (mg *OrgIDPGitHubEnterpriseServer) SetIDPOrganization(orgID string) {
	mg.Status.AtProvider.Scope = StringPtr(orgID)
}

// IDPOrganization returns the organization the provider was last read from.
func (mg *OrgIDPGitHubEnterpriseServer) IDPOrganization() string {
	return Deref(mg.Status.AtProvider.Scope)
}

// GetOrganizationRef returns the reference the provider's organization is
// resolved from.
func (mg *OrgIDPGitHubEnterpriseServer) GetOrganizationRef() *xpv1.Reference {
	return mg.Spec.ForProvider.OrganizationRef
}

// GetOrganizationSelector returns the selector the provider's organization
// is resolved from.
func (mg *OrgIDPGitHubEnterpriseServer) GetOrganizationSelector() *xpv1.Selector {
	return mg.Spec.ForProvider.OrganizationSelector
}

// GetOrganizationID returns the pinned organization ID, if any.
func (mg *OrgIDPGitHubEnterpriseServer) GetOrganizationID() *string {
	return mg.Spec.ForProvider.OrganizationID
}

// GetObservedOrganization returns the organization the provider was last read
// from, which is what a terminating object acts on.
func (mg *OrgIDPGitHubEnterpriseServer) GetObservedOrganization() *string {
	return mg.Status.AtProvider.OrganizationID
}
