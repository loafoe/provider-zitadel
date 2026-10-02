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

// UserGrantParameters are the configurable fields of a UserGrant.
type UserGrantParameters struct {
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

	// ProjectGrantID is the ID of the project grant the roles are granted on.
	// Leave empty to grant roles on a project of the user's own organization.
	// +optional
	ProjectGrantID *string `json:"projectGrantID,omitempty"`

	// RoleKeys are the project roles to grant.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	RoleKeys []string `json:"roleKeys"`

	// State of the grant. Defaults to Active.
	// +optional
	State *MembershipState `json:"state,omitempty"`
}

// UserGrantObservation are the observable fields of a UserGrant.
type UserGrantObservation struct {
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

	// GrantID is the ID of the grant in Zitadel.
	// +optional
	GrantID *string `json:"grantID,omitempty"`

	// OrganizationID is the organization the grant belongs to.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// ProjectID is the project the roles are granted on.
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// ProjectGrantID is the project grant the roles are granted on.
	// +optional
	ProjectGrantID *string `json:"projectGrantID,omitempty"`

	// RoleKeys are the roles the grant currently holds.
	// +optional
	RoleKeys []string `json:"roleKeys,omitempty"`

	// State of the grant.
	// +optional
	State *MembershipState `json:"state,omitempty"`
}

// A UserGrantSpec defines the desired state of a UserGrant.
type UserGrantSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              UserGrantParameters `json:"forProvider"`
}

// A UserGrantStatus represents the observed state of a UserGrant.
type UserGrantStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserGrantObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="USER",type="string",JSONPath=".status.atProvider.userID"
// +kubebuilder:printcolumn:name="PROJECT",type="string",JSONPath=".status.atProvider.projectID"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// // A UserGrant gives a user (or service account) a set of project roles. This is
// the primitive that makes a project usable: without it a user can exist but
// can do nothing in the project. Roles may be granted on a project of the
// user's own organization, or on a project granted to it with a ProjectGrant.
type UserGrant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserGrantSpec   `json:"spec"`
	Status UserGrantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserGrantList contains a list of UserGrant.
type UserGrantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserGrant `json:"items"`
}

// UserGrant type metadata.
var (
	UserGrantKind             = reflect.TypeOf(UserGrant{}).Name()
	UserGrantGroupKind        = schema.GroupKind{Group: Group, Kind: UserGrantKind}.String()
	UserGrantKindAPIVersion   = UserGrantKind + "." + SchemeGroupVersion.String()
	UserGrantGroupVersionKind = SchemeGroupVersion.WithKind(UserGrantKind)
)

func init() {
	SchemeBuilder.Register(&UserGrant{}, &UserGrantList{})
}
