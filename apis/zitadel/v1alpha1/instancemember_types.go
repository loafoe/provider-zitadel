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

// InstanceMemberParameters are the configurable fields of a InstanceMember.
type InstanceMemberParameters struct {
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

	// RoleKeys are the instance level roles, such as IAM_OWNER.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	RoleKeys []string `json:"roleKeys"`
}

// InstanceMemberObservation are the observable fields of a InstanceMember.
type InstanceMemberObservation struct {
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

// A InstanceMemberSpec defines the desired state of a InstanceMember.
type InstanceMemberSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              InstanceMemberParameters `json:"forProvider"`
}

// A InstanceMemberStatus represents the observed state of a InstanceMember.
type InstanceMemberStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          InstanceMemberObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="USER",type="string",JSONPath=".status.atProvider.userID"
// +kubebuilder:printcolumn:name="ROLES",type="string",JSONPath=".status.atProvider.roleKeys"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// // An InstanceMember makes a user (or service account) an administrator of the
// Zitadel instance itself, with a set of instance level roles such as
// IAM_OWNER. This is how the first administrator of a new instance is created.
type InstanceMember struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceMemberSpec   `json:"spec"`
	Status InstanceMemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InstanceMemberList contains a list of InstanceMember.
type InstanceMemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanceMember `json:"items"`
}

// InstanceMember type metadata.
var (
	InstanceMemberKind             = reflect.TypeOf(InstanceMember{}).Name()
	InstanceMemberGroupKind        = schema.GroupKind{Group: Group, Kind: InstanceMemberKind}.String()
	InstanceMemberKindAPIVersion   = InstanceMemberKind + "." + SchemeGroupVersion.String()
	InstanceMemberGroupVersionKind = SchemeGroupVersion.WithKind(InstanceMemberKind)
)

func init() {
	SchemeBuilder.Register(&InstanceMember{}, &InstanceMemberList{})
}
