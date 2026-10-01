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

// ProjectRoleParameters are the configurable fields of a ProjectRole.
type ProjectRoleParameters struct {
	// Key is the unique key of the role within the project. It cannot be
	// changed after creation.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Key string `json:"key"`

	// DisplayName is the human readable name of the role.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`

	// Group is the group the role is shown in. Defaults to an empty group.
	// +optional
	Group *string `json:"group,omitempty"`

	// ProjectID is the ID of the project the role belongs to. Either
	// `projectID`, `projectRef` or `projectSelector` must be set.
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// ProjectRef references a Project managed by this provider and uses its ID.
	// +optional
	ProjectRef *xpv1.Reference `json:"projectRef,omitempty"`

	// ProjectSelector selects a Project managed by this provider and uses its
	// ID.
	// +optional
	ProjectSelector *xpv1.Selector `json:"projectSelector,omitempty"`
}

// ProjectRoleObservation are the observable fields of a ProjectRole.
type ProjectRoleObservation struct {
	// Key of the role.
	// +optional
	Key *string `json:"key,omitempty"`

	// ProjectID is the ID of the owning project.
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// DisplayName of the role.
	// +optional
	DisplayName *string `json:"displayName,omitempty"`

	// Group of the role.
	// +optional
	Group *string `json:"group,omitempty"`

	// CreationDate is the timestamp the role was created at.
	// +optional
	CreationDate *metav1.Time `json:"creationDate,omitempty"`

	// ChangeDate is the timestamp the role was last modified at.
	// +optional
	ChangeDate *metav1.Time `json:"changeDate,omitempty"`
}

// A ProjectRoleSpec defines the desired state of a ProjectRole.
type ProjectRoleSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ProjectRoleParameters `json:"forProvider"`
}

// A ProjectRoleStatus represents the observed state of a ProjectRole.
type ProjectRoleStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ProjectRoleObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="KEY",type="string",JSONPath=".status.atProvider.key"
// +kubebuilder:printcolumn:name="PROJECT-ID",type="string",JSONPath=".status.atProvider.projectID"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// A ProjectRole is a role that can be granted to users of a Zitadel project.
type ProjectRole struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectRoleSpec   `json:"spec"`
	Status ProjectRoleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectRoleList contains a list of ProjectRole.
type ProjectRoleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProjectRole `json:"items"`
}

// ProjectRole type metadata.
var (
	ProjectRoleKind             = reflect.TypeOf(ProjectRole{}).Name()
	ProjectRoleGroupKind        = schema.GroupKind{Group: Group, Kind: ProjectRoleKind}.String()
	ProjectRoleKindAPIVersion   = ProjectRoleKind + "." + SchemeGroupVersion.String()
	ProjectRoleGroupVersionKind = SchemeGroupVersion.WithKind(ProjectRoleKind)
)

func init() {
	SchemeBuilder.Register(&ProjectRole{}, &ProjectRoleList{})
}
