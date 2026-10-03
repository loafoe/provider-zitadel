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

// OrgIDPGitLabParameters are the configurable fields of an OrgIDPGitLab.
type OrgIDPGitLabParameters struct {
	// The OAuth app's client ID.
	// +optional
	ClientID *string `json:"clientID,omitempty"`

	// The OAuth app's client secret, read from a secret.
	// +optional
	ClientSecret *SecretKeySelector `json:"clientSecretSecretRef,omitempty"`

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
type OrgIDPGitLabObservation = IDPObservation

// An OrgIDPGitLab lets users sign in with gitlab.com.
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
type OrgIDPGitLab struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrgIDPGitLabSpec   `json:"spec"`
	Status OrgIDPGitLabStatus `json:"status,omitempty"`
}

// OrgIDPGitLabSpec defines the desired state of an OrgIDPGitLab.
type OrgIDPGitLabSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OrgIDPGitLabParameters `json:"forProvider"`
}

// OrgIDPGitLabStatus is the observed state of an OrgIDPGitLab.
type OrgIDPGitLabStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrgIDPGitLabObservation `json:"atProvider,omitempty"`
}

// OrgIDPGitLabList contains a list of OrgIDPGitLab.
//
// +kubebuilder:object:root=true
type OrgIDPGitLabList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrgIDPGitLab `json:"items"`
}

// OrgIDPGitLab type metadata.
var (
	OrgIDPGitLabKind             = reflect.TypeOf(OrgIDPGitLab{}).Name()
	OrgIDPGitLabGroupKind        = schema.GroupKind{Group: Group, Kind: OrgIDPGitLabKind}.String()
	OrgIDPGitLabKindAPIVersion   = OrgIDPGitLabKind + "." + SchemeGroupVersion.String()
	OrgIDPGitLabGroupVersionKind = SchemeGroupVersion.WithKind(OrgIDPGitLabKind)
)

func init() {
	SchemeBuilder.Register(&OrgIDPGitLab{}, &OrgIDPGitLabList{})
}

// GetIsLinkingAllowed returns the setting every identity provider carries.
func (mg *OrgIDPGitLab) GetIsLinkingAllowed() *bool { return mg.Spec.ForProvider.IsLinkingAllowed }

// GetIsCreationAllowed returns the setting every identity provider carries.
func (mg *OrgIDPGitLab) GetIsCreationAllowed() *bool { return mg.Spec.ForProvider.IsCreationAllowed }

// GetIsAutoCreation returns the setting every identity provider carries.
func (mg *OrgIDPGitLab) GetIsAutoCreation() *bool { return mg.Spec.ForProvider.IsAutoCreation }

// GetIsAutoUpdate returns the setting every identity provider carries.
func (mg *OrgIDPGitLab) GetIsAutoUpdate() *bool { return mg.Spec.ForProvider.IsAutoUpdate }

// GetAutoLinking returns the setting every identity provider carries.
func (mg *OrgIDPGitLab) GetAutoLinking() *IDPAutoLinking { return mg.Spec.ForProvider.AutoLinking }

// GetState returns the setting every identity provider carries.
func (mg *OrgIDPGitLab) GetState() *IDPState { return mg.Spec.ForProvider.State }

// SetIDPOrganization records the organization the provider belongs to, or the
// empty string for an instance wide one.
func (mg *OrgIDPGitLab) SetIDPOrganization(orgID string) {
	mg.Status.AtProvider.Scope = StringPtr(orgID)
}

// IDPOrganization returns the organization the provider was last read from.
func (mg *OrgIDPGitLab) IDPOrganization() string { return Deref(mg.Status.AtProvider.Scope) }

// GetOrganizationRef returns the reference the provider's organization is
// resolved from.
func (mg *OrgIDPGitLab) GetOrganizationRef() *xpv1.Reference {
	return mg.Spec.ForProvider.OrganizationRef
}

// GetOrganizationSelector returns the selector the provider's organization
// is resolved from.
func (mg *OrgIDPGitLab) GetOrganizationSelector() *xpv1.Selector {
	return mg.Spec.ForProvider.OrganizationSelector
}

// GetOrganizationID returns the pinned organization ID, if any.
func (mg *OrgIDPGitLab) GetOrganizationID() *string { return mg.Spec.ForProvider.OrganizationID }

// GetObservedOrganization returns the organization the provider was last read
// from, which is what a terminating object acts on.
func (mg *OrgIDPGitLab) GetObservedOrganization() *string { return mg.Status.AtProvider.OrganizationID }
