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

// ServiceAccountParameters are the configurable fields of a ServiceAccount.
type ServiceAccountParameters struct {
	// Name is the display name of the service account. If both `name` and
	// `userName` are omitted Zitadel uses the user ID as the name.
	// +optional
	Name *string `json:"name,omitempty"`

	// UserName is the unique username of the service account within the
	// organization. If omitted Zitadel defaults it to the user ID.
	// +optional
	UserName *string `json:"userName,omitempty"`

	// Description of the service account.
	// +optional
	Description *string `json:"description,omitempty"`

	// AccessTokenType of the tokens issued for this service account. Defaults
	// to Bearer.
	// +optional
	AccessTokenType *AccessTokenType `json:"accessTokenType,omitempty"`

	// Metadata entries set on the service account. Only used on creation.
	// +optional
	Metadata MetadataList `json:"metadata,omitempty"`

	// OrganizationID is the ID of the organization owning the service account.
	// Either `organizationID`, `organizationRef` or `organizationSelector` must
	// be set.
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

	// ID allows setting a custom user ID. If omitted Zitadel generates one. It
	// cannot be changed after creation.
	// +optional
	ID *string `json:"id,omitempty"`

	// State of the service account. Defaults to Active.
	// +optional
	State *UserState `json:"state,omitempty"`
}

// ServiceAccountObservation are the observable fields of a ServiceAccount.
type ServiceAccountObservation struct {
	// ID is the Zitadel user ID of the service account (machine user).
	// +optional
	ID *string `json:"id,omitempty"`

	// UserName of the service account.
	// +optional
	UserName *string `json:"userName,omitempty"`

	// PreferredLoginName of the service account.
	// +optional
	PreferredLoginName *string `json:"preferredLoginName,omitempty"`

	// Name is the display name of the service account.
	// +optional
	Name *string `json:"name,omitempty"`

	// Description of the service account.
	// +optional
	Description *string `json:"description,omitempty"`

	// OrganizationID is the ID of the owning organization.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// State of the service account.
	// +optional
	State *UserState `json:"state,omitempty"`

	// AccessTokenType of the issued tokens.
	// +optional
	AccessTokenType *AccessTokenType `json:"accessTokenType,omitempty"`

	// HasSecret reports whether a client secret is set on the service account.
	// +optional
	HasSecret *bool `json:"hasSecret,omitempty"`

	// CreationDate is the timestamp the service account was created at.
	// +optional
	CreationDate *metav1.Time `json:"creationDate,omitempty"`

	// ChangeDate is the timestamp the service account was last modified at.
	// +optional
	ChangeDate *metav1.Time `json:"changeDate,omitempty"`
}

// A ServiceAccountSpec defines the desired state of a ServiceAccount.
type ServiceAccountSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ServiceAccountParameters `json:"forProvider"`
}

// A ServiceAccountStatus represents the observed state of a ServiceAccount.
type ServiceAccountStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ServiceAccountObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="USERNAME",type="string",JSONPath=".status.atProvider.userName"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.atProvider.id"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// A ServiceAccount is a Zitadel machine user. Service accounts are used both
// to authenticate workloads against the Zitadel API (see the ProviderConfig)
// and to grant access to Zitadel projects.
type ServiceAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServiceAccountSpec   `json:"spec"`
	Status ServiceAccountStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ServiceAccountList contains a list of ServiceAccount.
type ServiceAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ServiceAccount `json:"items"`
}

// ServiceAccount type metadata.
var (
	ServiceAccountKind             = reflect.TypeOf(ServiceAccount{}).Name()
	ServiceAccountGroupKind        = schema.GroupKind{Group: Group, Kind: ServiceAccountKind}.String()
	ServiceAccountKindAPIVersion   = ServiceAccountKind + "." + SchemeGroupVersion.String()
	ServiceAccountGroupVersionKind = SchemeGroupVersion.WithKind(ServiceAccountKind)
)

func init() {
	SchemeBuilder.Register(&ServiceAccount{}, &ServiceAccountList{})
}
