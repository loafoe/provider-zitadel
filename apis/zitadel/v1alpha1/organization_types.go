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

// OrganizationState is the lifecycle state of an Organization.
// +kubebuilder:validation:Enum=Active;Inactive
type OrganizationState string

const (
	OrganizationStateActive   OrganizationState = "Active"
	OrganizationStateInactive OrganizationState = "Inactive"
)

// OrganizationParameters are the configurable fields of an Organization.
type OrganizationParameters struct {
	// Name is the display name of the organization.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// ID allows setting a custom organization ID. If omitted Zitadel generates
	// one. It cannot be changed after creation.
	// +optional
	ID *string `json:"id,omitempty"`

	// PrimaryDomain is the primary (verified) domain of the organization. It is
	// only reported, not managed, by this provider.
	// +optional
	PrimaryDomain *string `json:"primaryDomain,omitempty"`

	// State of the organization. Defaults to Active.
	// +optional
	State *OrganizationState `json:"state,omitempty"`
}

// OrganizationObservation are the observable fields of an Organization.
type OrganizationObservation struct {
	// ID is the Zitadel organization ID.
	// +optional
	ID *string `json:"id,omitempty"`

	// Name is the display name of the organization.
	// +optional
	Name *string `json:"name,omitempty"`

	// PrimaryDomain is the primary domain of the organization.
	// +optional
	PrimaryDomain *string `json:"primaryDomain,omitempty"`

	// State of the organization.
	// +optional
	State *OrganizationState `json:"state,omitempty"`

	// CreationDate is the timestamp the organization was created at.
	// +optional
	CreationDate *metav1.Time `json:"creationDate,omitempty"`

	// ChangeDate is the timestamp the organization was last modified at.
	// +optional
	ChangeDate *metav1.Time `json:"changeDate,omitempty"`
}

// An OrganizationSpec defines the desired state of an Organization.
type OrganizationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OrganizationParameters `json:"forProvider"`
}

// An OrganizationStatus represents the observed state of an Organization.
type OrganizationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrganizationObservation `json:"atProvider,omitempty"`
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
// An Organization is a Zitadel organization (tenant). Users, projects and
// applications always belong to an organization.
type Organization struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationSpec   `json:"spec"`
	Status OrganizationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationList contains a list of Organization.
type OrganizationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Organization `json:"items"`
}

// Organization type metadata.
var (
	OrganizationKind             = reflect.TypeOf(Organization{}).Name()
	OrganizationGroupKind        = schema.GroupKind{Group: Group, Kind: OrganizationKind}.String()
	OrganizationKindAPIVersion   = OrganizationKind + "." + SchemeGroupVersion.String()
	OrganizationGroupVersionKind = SchemeGroupVersion.WithKind(OrganizationKind)
)

func init() {
	SchemeBuilder.Register(&Organization{}, &OrganizationList{})
}
