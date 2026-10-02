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

// UserMetadataParameters are the configurable fields of a UserMetadata.
type UserMetadataParameters struct {
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

	// Metadata is the complete set of metadata keys of the user. Zitadel
	// replaces the set rather than merging into it, so this resource owns the
	// whole set.
	// +optional
	Metadata MetadataList `json:"metadata,omitempty"`
}

// UserMetadataObservation are the observable fields of a UserMetadata.
type UserMetadataObservation struct {
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

	// Metadata is the metadata set Zitadel currently holds.
	// +optional
	Metadata MetadataList `json:"metadata,omitempty"`
}

// A UserMetadataSpec defines the desired state of a UserMetadata.
type UserMetadataSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              UserMetadataParameters `json:"forProvider"`
}

// A UserMetadataStatus represents the observed state of a UserMetadata.
type UserMetadataStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          UserMetadataObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="USER",type="string",JSONPath=".status.atProvider.userID"
// +kubebuilder:printcolumn:name="KEYS",type="string",JSONPath=".status.atProvider.metadata[*].key"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// // A UserMetadata manages the metadata of a user: the attributes Zitadel's
// Actions read to make decisions, most commonly the identifier a user has in an
// upstream system. The whole set is managed at once because Zitadel replaces
// metadata rather than merging it.
type UserMetadata struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserMetadataSpec   `json:"spec"`
	Status UserMetadataStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserMetadataList contains a list of UserMetadata.
type UserMetadataList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserMetadata `json:"items"`
}

// UserMetadata type metadata.
var (
	UserMetadataKind             = reflect.TypeOf(UserMetadata{}).Name()
	UserMetadataGroupKind        = schema.GroupKind{Group: Group, Kind: UserMetadataKind}.String()
	UserMetadataKindAPIVersion   = UserMetadataKind + "." + SchemeGroupVersion.String()
	UserMetadataGroupVersionKind = SchemeGroupVersion.WithKind(UserMetadataKind)
)

func init() {
	SchemeBuilder.Register(&UserMetadata{}, &UserMetadataList{})
}
