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

// OrgMemberParameters are the configurable fields of a OrgMember.
type OrgMemberParameters struct {
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

	// UserID is the ID of the user or service account. Either `userID`,
	// `userRef` or `userSelector` must be set.
	// +optional
	UserID *string `json:"userID,omitempty"`

	// UserRef references a HumanUser or ServiceAccount managed by this
	// provider.
	// +optional
	UserRef *xpv1.Reference `json:"userRef,omitempty"`

	// UserSelector selects a HumanUser or ServiceAccount managed by this
	// provider.
	// +optional
	UserSelector *xpv1.Selector `json:"userSelector,omitempty"`

	// RoleKeys are the organization level roles, such as ORG_OWNER. Zitadel
	// configures which keys exist per instance, so they are validated against
	// the instance instead of a fixed list.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	RoleKeys []string `json:"roleKeys"`
}

// OrgMemberObservation are the observable fields of a OrgMember.
type OrgMemberObservation struct {
	// UserID is the ID of the user or service account. Either `userID`,
	// `userRef` or `userSelector` must be set.
	// +optional
	UserID *string `json:"userID,omitempty"`

	// UserRef references a HumanUser or ServiceAccount managed by this
	// provider.
	// +optional
	UserRef *xpv1.Reference `json:"userRef,omitempty"`

	// UserSelector selects a HumanUser or ServiceAccount managed by this
	// provider.
	// +optional
	UserSelector *xpv1.Selector `json:"userSelector,omitempty"`

	// OrganizationID is the organization the user is a member of.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// RoleKeys are the roles the membership currently holds.
	// +optional
	RoleKeys []string `json:"roleKeys,omitempty"`

	// DisplayName of the member.
	// +optional
	DisplayName *string `json:"displayName,omitempty"`

	// Email of the member.
	// +optional
	Email *string `json:"email,omitempty"`

	// UserType of the member, e.g. human or machine.
	// +optional
	UserType *string `json:"userType,omitempty"`
}

// A OrgMemberSpec defines the desired state of a OrgMember.
type OrgMemberSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OrgMemberParameters `json:"forProvider"`
}

// A OrgMemberStatus represents the observed state of a OrgMember.
type OrgMemberStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrgMemberObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="USER",type="string",JSONPath=".status.atProvider.userID"
// +kubebuilder:printcolumn:name="ORGANIZATION",type="string",JSONPath=".status.atProvider.organizationID"
// +kubebuilder:printcolumn:name="ROLES",type="string",JSONPath=".status.atProvider.roleKeys"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// // An OrgMember makes a user (or service account) a member of an organization
// with a set of organization level roles, such as ORG_OWNER or
// ORG_USER_MANAGER. Zitadel separates this from project roles: an OrgMember
// administers the organization itself.
type OrgMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrgMemberSpec   `json:"spec"`
	Status OrgMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrgMemberList contains a list of OrgMember.
type OrgMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrgMember `json:"items"`
}

// OrgMember type metadata.
var (
	OrgMemberKind             = reflect.TypeOf(OrgMember{}).Name()
	OrgMemberGroupKind        = schema.GroupKind{Group: Group, Kind: OrgMemberKind}.String()
	OrgMemberKindAPIVersion   = OrgMemberKind + "." + SchemeGroupVersion.String()
	OrgMemberGroupVersionKind = SchemeGroupVersion.WithKind(OrgMemberKind)
)

func init() {
	SchemeBuilder.Register(&OrgMember{}, &OrgMemberList{})
}
