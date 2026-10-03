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

// IDPAzureADParameters are the configurable fields of an IDPAzureAD.
type IDPAzureADParameters struct {
	// The application ID of the Entra app registration.
	// +optional
	ClientID *string `json:"clientID,omitempty"`

	// The client secret, read from a secret.
	// +optional
	ClientSecret *SecretKeySelector `json:"clientSecretSecretRef,omitempty"`

	// Which Entra tenants may log in: `Common` for any organization, `Consumers`
	// for personal accounts, or an exact tenant ID. Defaults to `Common`.
	// +optional
	Tenant *AzureADTenant `json:"tenant,omitempty"`

	// Treat the address Entra reports as verified.
	// +optional
	EmailVerified *bool `json:"emailVerified,omitempty"`

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
type IDPAzureADObservation = IDPObservation

// An IDPAzureAD lets users sign in with Microsoft Entra ID.
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
type IDPAzureAD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IDPAzureADSpec   `json:"spec"`
	Status IDPAzureADStatus `json:"status,omitempty"`
}

// IDPAzureADSpec defines the desired state of an IDPAzureAD.
type IDPAzureADSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              IDPAzureADParameters `json:"forProvider"`
}

// IDPAzureADStatus is the observed state of an IDPAzureAD.
type IDPAzureADStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          IDPAzureADObservation `json:"atProvider,omitempty"`
}

// IDPAzureADList contains a list of IDPAzureAD.
//
// +kubebuilder:object:root=true
type IDPAzureADList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IDPAzureAD `json:"items"`
}

// IDPAzureAD type metadata.
var (
	IDPAzureADKind             = reflect.TypeOf(IDPAzureAD{}).Name()
	IDPAzureADGroupKind        = schema.GroupKind{Group: Group, Kind: IDPAzureADKind}.String()
	IDPAzureADKindAPIVersion   = IDPAzureADKind + "." + SchemeGroupVersion.String()
	IDPAzureADGroupVersionKind = SchemeGroupVersion.WithKind(IDPAzureADKind)
)

func init() {
	SchemeBuilder.Register(&IDPAzureAD{}, &IDPAzureADList{})
}

// GetIsLinkingAllowed returns the setting every identity provider carries.
func (mg *IDPAzureAD) GetIsLinkingAllowed() *bool { return mg.Spec.ForProvider.IsLinkingAllowed }

// GetIsCreationAllowed returns the setting every identity provider carries.
func (mg *IDPAzureAD) GetIsCreationAllowed() *bool { return mg.Spec.ForProvider.IsCreationAllowed }

// GetIsAutoCreation returns the setting every identity provider carries.
func (mg *IDPAzureAD) GetIsAutoCreation() *bool { return mg.Spec.ForProvider.IsAutoCreation }

// GetIsAutoUpdate returns the setting every identity provider carries.
func (mg *IDPAzureAD) GetIsAutoUpdate() *bool { return mg.Spec.ForProvider.IsAutoUpdate }

// GetAutoLinking returns the setting every identity provider carries.
func (mg *IDPAzureAD) GetAutoLinking() *IDPAutoLinking { return mg.Spec.ForProvider.AutoLinking }

// GetState returns the setting every identity provider carries.
func (mg *IDPAzureAD) GetState() *IDPState { return mg.Spec.ForProvider.State }

// SetIDPOrganization records the organization the provider belongs to, or the
// empty string for an instance wide one.
func (mg *IDPAzureAD) SetIDPOrganization(orgID string) { mg.Status.AtProvider.Scope = StringPtr(orgID) }

// IDPOrganization returns the organization the provider was last read from.
func (mg *IDPAzureAD) IDPOrganization() string { return Deref(mg.Status.AtProvider.Scope) }
