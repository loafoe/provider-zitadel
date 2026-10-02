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

// OrganizationMetadataParameters are the configurable fields of a OrganizationMetadata.
type OrganizationMetadataParameters struct {
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

	// Metadata is the complete set of metadata keys of the organization.
	// +optional
	Metadata MetadataList `json:"metadata,omitempty"`
}

// OrganizationMetadataObservation are the observable fields of a OrganizationMetadata.
type OrganizationMetadataObservation struct {
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

	// Metadata is the metadata set Zitadel currently holds.
	// +optional
	Metadata MetadataList `json:"metadata,omitempty"`
}

// A OrganizationMetadataSpec defines the desired state of a OrganizationMetadata.
type OrganizationMetadataSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OrganizationMetadataParameters `json:"forProvider"`
}

// A OrganizationMetadataStatus represents the observed state of a OrganizationMetadata.
type OrganizationMetadataStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OrganizationMetadataObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="ORGANIZATION",type="string",JSONPath=".status.atProvider.organizationID"
// +kubebuilder:printcolumn:name="KEYS",type="string",JSONPath=".status.atProvider.metadata[*].key"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// // An OrganizationMetadata manages the metadata of an organization.
type OrganizationMetadata struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationMetadataSpec   `json:"spec"`
	Status OrganizationMetadataStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationMetadataList contains a list of OrganizationMetadata.
type OrganizationMetadataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationMetadata `json:"items"`
}

// OrganizationMetadata type metadata.
var (
	OrganizationMetadataKind             = reflect.TypeOf(OrganizationMetadata{}).Name()
	OrganizationMetadataGroupKind        = schema.GroupKind{Group: Group, Kind: OrganizationMetadataKind}.String()
	OrganizationMetadataKindAPIVersion   = OrganizationMetadataKind + "." + SchemeGroupVersion.String()
	OrganizationMetadataGroupVersionKind = SchemeGroupVersion.WithKind(OrganizationMetadataKind)
)

func init() {
	SchemeBuilder.Register(&OrganizationMetadata{}, &OrganizationMetadataList{})
}
