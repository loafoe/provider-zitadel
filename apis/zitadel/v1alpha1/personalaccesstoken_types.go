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

// PersonalAccessTokenParameters are the configurable fields of a
// PersonalAccessToken.
type PersonalAccessTokenParameters struct {
	// UserID is the ID of the machine user the token belongs to. Either `userID`,
	// `userRef` or `userSelector` must be set.
	// +optional
	UserID *string `json:"userID,omitempty"`

	// UserRef references a ServiceAccount managed by this provider and uses its
	// ID. Zitadel issues personal access tokens for machine users only, so a
	// HumanUser is rejected.
	// +optional
	UserRef *xpv1.Reference `json:"userRef,omitempty"`

	// UserSelector selects a ServiceAccount managed by this provider and uses
	// its ID. Zitadel issues personal access tokens for machine users only, so a
	// HumanUser is rejected.
	// +optional
	UserSelector *xpv1.Selector `json:"userSelector,omitempty"`

	// ExpirationDate of the token, as an RFC3339 timestamp. Zitadel always
	// requires an expiration date; when it is omitted the provider uses one
	// year from the time the token is created.
	// +optional
	ExpirationDate *metav1.Time `json:"expirationDate,omitempty"`
}

// PersonalAccessTokenObservation are the observable fields of a
// PersonalAccessToken.
type PersonalAccessTokenObservation struct {
	// TokenID is the ID of the personal access token.
	// +optional
	TokenID *string `json:"tokenID,omitempty"`

	// UserID is the ID of the user the token belongs to.
	// +optional
	UserID *string `json:"userID,omitempty"`

	// OrganizationID is the ID of the organization owning the user.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// ExpirationDate of the token.
	// +optional
	ExpirationDate *metav1.Time `json:"expirationDate,omitempty"`

	// CreationDate is the timestamp the token was created at.
	// +optional
	CreationDate *metav1.Time `json:"creationDate,omitempty"`

	// ChangeDate is the timestamp the token was last modified at.
	// +optional
	ChangeDate *metav1.Time `json:"changeDate,omitempty"`
}

// A PersonalAccessTokenSpec defines the desired state of a PersonalAccessToken.
type PersonalAccessTokenSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              PersonalAccessTokenParameters `json:"forProvider"`
}

// A PersonalAccessTokenStatus represents the observed state of a
// PersonalAccessToken.
type PersonalAccessTokenStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PersonalAccessTokenObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="TOKEN-ID",type="string",JSONPath=".status.atProvider.tokenID"
// +kubebuilder:printcolumn:name="USER-ID",type="string",JSONPath=".status.atProvider.userID"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// A PersonalAccessToken issues a Personal Access Token (PAT) for a user. The
// token itself is only returned once and is written to the connection secret.
type PersonalAccessToken struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PersonalAccessTokenSpec   `json:"spec"`
	Status PersonalAccessTokenStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PersonalAccessTokenList contains a list of PersonalAccessToken.
type PersonalAccessTokenList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PersonalAccessToken `json:"items"`
}

// PersonalAccessToken type metadata.
var (
	PersonalAccessTokenKind             = reflect.TypeOf(PersonalAccessToken{}).Name()
	PersonalAccessTokenGroupKind        = schema.GroupKind{Group: Group, Kind: PersonalAccessTokenKind}.String()
	PersonalAccessTokenKindAPIVersion   = PersonalAccessTokenKind + "." + SchemeGroupVersion.String()
	PersonalAccessTokenGroupVersionKind = SchemeGroupVersion.WithKind(PersonalAccessTokenKind)
)

func init() {
	SchemeBuilder.Register(&PersonalAccessToken{}, &PersonalAccessTokenList{})
}
