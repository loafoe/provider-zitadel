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

// ProjectGrantParameters are the configurable fields of a ProjectGrant.
type ProjectGrantParameters struct {
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

	// RoleKeys are the roles members of the granted organization may hold on
	// the project.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	RoleKeys []string `json:"roleKeys"`

	// State of the grant. Defaults to Active.
	// +optional
	State *GrantableState `json:"state,omitempty"`
}

// ProjectGrantObservation are the observable fields of a ProjectGrant.
type ProjectGrantObservation struct {
	// OrganizationID is the organization that owns the project. Zitadel grants
	// a project from its owner, so this is what identifies the grant itself.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// ProjectID is the shared project.
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// GrantedOrganizationID is the organization the project is shared with.
	// +optional
	GrantedOrganizationID *string `json:"grantedOrganizationID,omitempty"`

	// GrantedOrganizationName is the name of that organization.
	// +optional
	GrantedOrganizationName *string `json:"grantedOrganizationName,omitempty"`

	// RoleKeys are the roles the grant currently exposes.
	// +optional
	RoleKeys []string `json:"roleKeys,omitempty"`

	// State of the grant.
	// +optional
	State *GrantableState `json:"state,omitempty"`
}

// A ProjectGrantSpec defines the desired state of a ProjectGrant.
type ProjectGrantSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ProjectGrantParameters `json:"forProvider"`
}

// A ProjectGrantStatus represents the observed state of a ProjectGrant.
type ProjectGrantStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ProjectGrantObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="PROJECT",type="string",JSONPath=".status.atProvider.projectID"
// +kubebuilder:printcolumn:name="GRANTED-ORG",type="string",JSONPath=".status.atProvider.grantedOrganizationID"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// // A ProjectGrant shares a project with another organization, together with the
// roles that organization may hold on it. This is the multi-tenancy primitive:
// a platform organization can offer a project to a customer organization
// without transferring ownership.
type ProjectGrant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectGrantSpec   `json:"spec"`
	Status ProjectGrantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectGrantList contains a list of ProjectGrant.
type ProjectGrantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectGrant `json:"items"`
}

// ProjectGrant type metadata.
var (
	ProjectGrantKind             = reflect.TypeOf(ProjectGrant{}).Name()
	ProjectGrantGroupKind        = schema.GroupKind{Group: Group, Kind: ProjectGrantKind}.String()
	ProjectGrantKindAPIVersion   = ProjectGrantKind + "." + SchemeGroupVersion.String()
	ProjectGrantGroupVersionKind = SchemeGroupVersion.WithKind(ProjectGrantKind)
)

func init() {
	SchemeBuilder.Register(&ProjectGrant{}, &ProjectGrantList{})
}
