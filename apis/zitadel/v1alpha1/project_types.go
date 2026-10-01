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

// ProjectState is the lifecycle state of a Project.
// +kubebuilder:validation:Enum=Active;Inactive
type ProjectState string

const (
	ProjectStateActive   ProjectState = "Active"
	ProjectStateInactive ProjectState = "Inactive"
)

// PrivateLabelingSetting controls the branding of the login for a project.
// +kubebuilder:validation:Enum=EnforceProjectResourceOwnerPolicy;AllowLoginUserResourceOwnerPolicy
type PrivateLabelingSetting string

const (
	PrivateLabelingSettingEnforceProjectResourceOwnerPolicy PrivateLabelingSetting = "EnforceProjectResourceOwnerPolicy"
	PrivateLabelingSettingAllowLoginUserResourceOwnerPolicy PrivateLabelingSetting = "AllowLoginUserResourceOwnerPolicy"
)

// ProjectParameters are the configurable fields of a Project.
type ProjectParameters struct {
	// Name is the display name of the project.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// OrganizationID is the ID of the organization owning the project.
	// Either `organizationID`, `organizationRef` or `organizationSelector`
	// must be set. When only `organizationRef`/`organizationSelector` is used
	// the provider resolves the referenced Organization to its ID.
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

	// ID allows setting a custom project ID. If omitted Zitadel generates one.
	// It cannot be changed after creation.
	// +optional
	ID *string `json:"id,omitempty"`

	// ProjectRoleAssertion makes Zitadel add the project roles to the tokens
	// issued for this project.
	// +optional
	ProjectRoleAssertion *bool `json:"projectRoleAssertion,omitempty"`

	// AuthorizationRequired requires that every user of the project is
	// authorized through a project grant.
	// +optional
	AuthorizationRequired *bool `json:"authorizationRequired,omitempty"`

	// ProjectAccessRequired requires an explicit project grant before a user
	// may access the project at all.
	// +optional
	ProjectAccessRequired *bool `json:"projectAccessRequired,omitempty"`

	// PrivateLabelingSetting controls the branding of the login.
	// +optional
	PrivateLabelingSetting *PrivateLabelingSetting `json:"privateLabelingSetting,omitempty"`

	// State of the project. Defaults to Active. Setting it to Inactive
	// deactivates the project, setting it back to Active reactivates it.
	// +optional
	State *ProjectState `json:"state,omitempty"`
}

// ProjectObservation are the observable fields of a Project.
type ProjectObservation struct {
	// ID is the Zitadel project ID.
	// +optional
	ID *string `json:"id,omitempty"`

	// OrganizationID is the ID of the owning organization.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// Name is the display name of the project.
	// +optional
	Name *string `json:"name,omitempty"`

	// State of the project.
	// +optional
	State *ProjectState `json:"state,omitempty"`

	// ProjectRoleAssertion is reported by Zitadel.
	// +optional
	ProjectRoleAssertion *bool `json:"projectRoleAssertion,omitempty"`

	// AuthorizationRequired is reported by Zitadel.
	// +optional
	AuthorizationRequired *bool `json:"authorizationRequired,omitempty"`

	// ProjectAccessRequired is reported by Zitadel.
	// +optional
	ProjectAccessRequired *bool `json:"projectAccessRequired,omitempty"`

	// PrivateLabelingSetting is reported by Zitadel.
	// +optional
	PrivateLabelingSetting *PrivateLabelingSetting `json:"privateLabelingSetting,omitempty"`

	// GrantedOrganizationID is set when the project is shared with another
	// organization through a project grant.
	// +optional
	GrantedOrganizationID *string `json:"grantedOrganizationID,omitempty"`

	// GrantedState is the state of the project grant.
	// +optional
	GrantedState *string `json:"grantedState,omitempty"`

	// CreationDate is the timestamp the project was created at.
	// +optional
	CreationDate *metav1.Time `json:"creationDate,omitempty"`

	// ChangeDate is the timestamp the project was last modified at.
	// +optional
	ChangeDate *metav1.Time `json:"changeDate,omitempty"`
}

// A ProjectSpec defines the desired state of a Project.
type ProjectSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ProjectParameters `json:"forProvider"`
}

// A ProjectStatus represents the observed state of a Project.
type ProjectStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ProjectObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// A Project is a Zitadel project. Applications, project roles and grants all
// live inside a project.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

// Project type metadata.
var (
	ProjectKind             = reflect.TypeOf(Project{}).Name()
	ProjectGroupKind        = schema.GroupKind{Group: Group, Kind: ProjectKind}.String()
	ProjectKindAPIVersion   = ProjectKind + "." + SchemeGroupVersion.String()
	ProjectGroupVersionKind = SchemeGroupVersion.WithKind(ProjectKind)
)

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
