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

// ProjectGrantMemberParameters are the configurable fields of a ProjectGrantMember.
type ProjectGrantMemberParameters struct {
	// ProjectID is the ID of the project. Either `projectID`, `projectRef` or
	// `projectSelector` must be set.
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// ProjectRef references a Project managed by this provider.
	// +optional
	ProjectRef *xpv1.Reference `json:"projectRef,omitempty"`

	// ProjectSelector selects a Project managed by this provider.
	// +optional
	ProjectSelector *xpv1.Selector `json:"projectSelector,omitempty"`

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

	// GrantedOrganizationID is the ID of the organization. Either
	// `grantedOrganizationID`, `grantedOrganizationRef` or
	// `grantedOrganizationSelector` must be set.
	// +optional
	GrantedOrganizationID *string `json:"grantedOrganizationID,omitempty"`

	// GrantedOrganizationRef references an Organization managed by this
	// provider.
	// +optional
	GrantedOrganizationRef *xpv1.Reference `json:"grantedOrganizationRef,omitempty"`

	// GrantedOrganizationSelector selects an Organization managed by this
	// provider.
	// +optional
	GrantedOrganizationSelector *xpv1.Selector `json:"grantedOrganizationSelector,omitempty"`

	// RoleKeys are the roles to give the user on the granted project.
	//
	// These are the grant's own roles, which Zitadel prefixes with
	// PROJECT_GRANT_ - for example PROJECT_GRANT_OWNER - not the project's
	// role keys. A member of a granted project is deliberately not given a role
	// in the project itself, which is what the grant is for.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	RoleKeys []string `json:"roleKeys"`
}

// ProjectGrantMemberObservation are the observable fields of a ProjectGrantMember.
type ProjectGrantMemberObservation struct {
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

	// OrganizationID is the organization the member belongs to.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// ProjectID is the shared project.
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// GrantID is Zitadel's identifier of the project grant.
	// +optional
	GrantID *string `json:"grantID,omitempty"`

	// GrantedOrganizationID is the organization the project was granted to.
	// +optional
	GrantedOrganizationID *string `json:"grantedOrganizationID,omitempty"`

	// RoleKeys are the roles the membership currently holds.
	// +optional
	RoleKeys []string `json:"roleKeys,omitempty"`

	// DisplayName of the member.
	// +optional
	DisplayName *string `json:"displayName,omitempty"`

	// Email of the member.
	// +optional
	Email *string `json:"email,omitempty"`
}

// A ProjectGrantMemberSpec defines the desired state of a ProjectGrantMember.
type ProjectGrantMemberSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ProjectGrantMemberParameters `json:"forProvider"`
}

// A ProjectGrantMemberStatus represents the observed state of a ProjectGrantMember.
type ProjectGrantMemberStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ProjectGrantMemberObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="USER",type="string",JSONPath=".status.atProvider.userID"
// +kubebuilder:printcolumn:name="PROJECT",type="string",JSONPath=".status.atProvider.projectID"
// +kubebuilder:printcolumn:name="ROLES",type="string",JSONPath=".status.atProvider.roleKeys"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// // A ProjectGrantMember gives a user (or service account) of the granted
// organization specific roles on a shared project. It is the member
// counterpart of a ProjectGrant, and its roles must come from that grant.
type ProjectGrantMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectGrantMemberSpec   `json:"spec"`
	Status ProjectGrantMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectGrantMemberList contains a list of ProjectGrantMember.
type ProjectGrantMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectGrantMember `json:"items"`
}

// ProjectGrantMember type metadata.
var (
	ProjectGrantMemberKind             = reflect.TypeOf(ProjectGrantMember{}).Name()
	ProjectGrantMemberGroupKind        = schema.GroupKind{Group: Group, Kind: ProjectGrantMemberKind}.String()
	ProjectGrantMemberKindAPIVersion   = ProjectGrantMemberKind + "." + SchemeGroupVersion.String()
	ProjectGrantMemberGroupVersionKind = SchemeGroupVersion.WithKind(ProjectGrantMemberKind)
)

func init() {
	SchemeBuilder.Register(&ProjectGrantMember{}, &ProjectGrantMemberList{})
}
