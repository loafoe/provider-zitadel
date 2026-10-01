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

// HumanUserParameters are the configurable fields of a HumanUser.
type HumanUserParameters struct {
	// UserName is the unique username of the user within the organization. If
	// omitted Zitadel defaults it to the email address.
	// +optional
	UserName *string `json:"userName,omitempty"`

	// GivenName is the first name of the user.
	// +kubebuilder:validation:Required
	GivenName string `json:"givenName"`

	// FamilyName is the last name of the user.
	// +kubebuilder:validation:Required
	FamilyName string `json:"familyName"`

	// NickName of the user.
	// +optional
	NickName *string `json:"nickName,omitempty"`

	// DisplayName of the user. Defaults to "<givenName> <familyName>".
	// +optional
	DisplayName *string `json:"displayName,omitempty"`

	// PreferredLanguage of the user, as an ISO 639-1 code, e.g. "en".
	// +optional
	PreferredLanguage *string `json:"preferredLanguage,omitempty"`

	// Gender of the user.
	// +optional
	Gender *Gender `json:"gender,omitempty"`

	// Email is the email address of the user.
	// +kubebuilder:validation:Required
	Email string `json:"email"`

	// EmailVerification determines how Zitadel handles the email address.
	// Defaults to SendCode.
	// +optional
	EmailVerification *EmailVerificationType `json:"emailVerification,omitempty"`

	// Phone of the user in E.164 format.
	// +optional
	Phone *string `json:"phone,omitempty"`

	// PhoneVerification determines how Zitadel handles the phone number.
	// +optional
	PhoneVerification *PhoneVerificationType `json:"phoneVerification,omitempty"`

	// PasswordSecretRef references the secret key holding the initial password
	// of the user. Zitadel never returns the password again, so changing it
	// does not update the remote user. Only used on creation.
	// +optional
	PasswordSecretRef *xpv1.LocalSecretKeySelector `json:"passwordSecretRef,omitempty"`

	// PasswordChangeRequired forces the user to change the initial password on
	// the first login. Only used on creation.
	// +optional
	PasswordChangeRequired *bool `json:"passwordChangeRequired,omitempty"`

	// IDPLinks are identity provider links added at creation time. Useful for
	// migration scenarios.
	// +optional
	IDPLinks []IDPLink `json:"idpLinks,omitempty"`

	// Metadata entries set on the user. Only used on creation.
	// +optional
	Metadata MetadataList `json:"metadata,omitempty"`

	// OrganizationID is the ID of the organization owning the user. Either
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

	// ID allows setting a custom user ID. If omitted Zitadel generates one. It
	// cannot be changed after creation.
	// +optional
	ID *string `json:"id,omitempty"`

	// State of the user. Defaults to Active.
	// +optional
	State *UserState `json:"state,omitempty"`
}

// HumanUserObservation are the observable fields of a HumanUser.
type HumanUserObservation struct {
	// ID is the Zitadel user ID.
	// +optional
	ID *string `json:"id,omitempty"`

	// UserName of the user.
	// +optional
	UserName *string `json:"userName,omitempty"`

	// PreferredLoginName is the primary login name of the user, e.g.
	// "john.doe@example.com".
	// +optional
	PreferredLoginName *string `json:"preferredLoginName,omitempty"`

	// LoginNames are all login names of the user.
	// +optional
	LoginNames []string `json:"loginNames,omitempty"`

	// OrganizationID is the ID of the owning organization.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// State of the user.
	// +optional
	State *UserState `json:"state,omitempty"`

	// Email of the user.
	// +optional
	Email *string `json:"email,omitempty"`

	// EmailVerified reports whether the email address is verified.
	// +optional
	EmailVerified *bool `json:"emailVerified,omitempty"`

	// Phone of the user.
	// +optional
	Phone *string `json:"phone,omitempty"`

	// PhoneVerified reports whether the phone number is verified.
	// +optional
	PhoneVerified *bool `json:"phoneVerified,omitempty"`

	// DisplayName of the user.
	// +optional
	DisplayName *string `json:"displayName,omitempty"`

	// NickName of the user.
	// +optional
	NickName *string `json:"nickName,omitempty"`

	// PasswordChangeRequired reports whether the user must change its password
	// on the next login.
	// +optional
	PasswordChangeRequired *bool `json:"passwordChangeRequired,omitempty"`

	// CreationDate is the timestamp the user was created at.
	// +optional
	CreationDate *metav1.Time `json:"creationDate,omitempty"`

	// ChangeDate is the timestamp the user was last modified at.
	// +optional
	ChangeDate *metav1.Time `json:"changeDate,omitempty"`
}

// A HumanUserSpec defines the desired state of a HumanUser.
type HumanUserSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              HumanUserParameters `json:"forProvider"`
}

// A HumanUserStatus represents the observed state of a HumanUser.
type HumanUserStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          HumanUserObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".status.atProvider.userName"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// A HumanUser is an interactive (human) Zitadel user.
type HumanUser struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HumanUserSpec   `json:"spec"`
	Status HumanUserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HumanUserList contains a list of HumanUser.
type HumanUserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HumanUser `json:"items"`
}

// HumanUser type metadata.
var (
	HumanUserKind             = reflect.TypeOf(HumanUser{}).Name()
	HumanUserGroupKind        = schema.GroupKind{Group: Group, Kind: HumanUserKind}.String()
	HumanUserKindAPIVersion   = HumanUserKind + "." + SchemeGroupVersion.String()
	HumanUserGroupVersionKind = SchemeGroupVersion.WithKind(HumanUserKind)
)

func init() {
	SchemeBuilder.Register(&HumanUser{}, &HumanUserList{})
}
