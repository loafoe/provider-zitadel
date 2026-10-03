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

// ProjectMemberSpec defines the desired state of a ProjectMember.
type ProjectMemberSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ProjectMemberParameters `json:"forProvider"`
}

// ProjectMemberParameters is the desired configuration of a grant a user roles on a project.
type ProjectMemberParameters struct {
	// OrganizationID is the ID of the organization the resource belongs to.
	// Either `organizationID`, `organizationRef` or `organizationSelector` must
	// be set.
	//
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// OrganizationRef references an Organization managed by this provider and
	// uses its ID.
	//
	// +optional
	OrganizationRef *xpv1.Reference `json:"organizationRef,omitempty"`

	// OrganizationSelector selects an Organization managed by this provider and
	// uses its ID.
	//
	// +optional
	OrganizationSelector *xpv1.Selector `json:"organizationSelector,omitempty"`

	// ProjectID is the ID of the project the resource belongs to. Either
	// `projectID`, `projectRef` or `projectSelector` must be set.
	//
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// ProjectRef references a Project managed by this provider and uses its ID.
	//
	// +optional
	ProjectRef *xpv1.Reference `json:"projectRef,omitempty"`

	// ProjectSelector selects a Project managed by this provider and uses its ID.
	//
	// +optional
	ProjectSelector *xpv1.Selector `json:"projectSelector,omitempty"`

	// UserID is the ID of the user the grant applies to. Either `userID`,
	// `userRef` or `userSelector` must be set.
	//
	// +optional
	UserID *string `json:"userID,omitempty"`

	// UserRef references a HumanUser or ServiceAccount managed by this provider
	// and uses its ID.
	//
	// +optional
	UserRef *xpv1.Reference `json:"userRef,omitempty"`

	// UserSelector selects a HumanUser or ServiceAccount managed by this provider
	// and uses its ID.
	//
	// +optional
	UserSelector *xpv1.Selector `json:"userSelector,omitempty"`

	// Roles controls the following. The roles the user holds on the project, such as `PROJECT_OWNER`. A project may define roles of its own, so the keys are not fixed. Zitadel refuses a member with no role at all, so at least one is required: to take every role away, delete this resource instead
	//
	// +optional
	// +kubebuilder:validation:MinItems=1
	Roles []string `json:"roles,omitempty"`
}

// ProjectMemberObservation is what Zitadel reports about a project member.
type ProjectMemberObservation struct {
	UserID      string   `json:"userID,omitempty"`
	Roles       []string `json:"roles,omitempty"`
	UserName    string   `json:"userName,omitempty"`
	DisplayName string   `json:"displayName,omitempty"`
	ProjectName string   `json:"projectName,omitempty"`
}

// ProjectMemberStatus reports the observed state of a grant a user roles on a project.
type ProjectMemberStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider ProjectMemberObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="ROLES",type="string",JSONPath=".spec.forProvider.roles"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// ProjectMember is a managed resource that grant a user roles on a project.
//
// Deleting the resource takes the user off the project. An empty role list is
// not the same as deleting: it leaves the user a member who can see the
// project and hold nothing on it.
// +kubebuilder:object:generate=true
type ProjectMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectMemberSpec   `json:"spec"`
	Status ProjectMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectMemberList contains a list of ProjectMember.
type ProjectMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectMember `json:"items"`
}

// ProjectMember type metadata.
var (
	ProjectMemberKind             = reflect.TypeOf(ProjectMember{}).Name()
	ProjectMemberGroupKind        = schema.GroupKind{Group: Group, Kind: ProjectMemberKind}.String()
	ProjectMemberKindAPIVersion   = ProjectMemberKind + "." + SchemeGroupVersion.String()
	ProjectMemberGroupVersionKind = SchemeGroupVersion.WithKind(ProjectMemberKind)
)

func init() {
	SchemeBuilder.Register(&ProjectMember{}, &ProjectMemberList{})
}
