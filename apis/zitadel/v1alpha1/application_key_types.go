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

// ApplicationKeySpec defines the desired state of a ApplicationKey.
type ApplicationKeySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ApplicationKeyParameters `json:"forProvider"`
}

// ApplicationKeyParameters is the desired configuration of a hold the key an application authenticates itself with.
type ApplicationKeyParameters struct {
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

	// ExpirationDate controls the following. When the key stops working, as a duration from now such as `8760h`. Empty means it does not expire. Zitadel returns the private half once, at creation, and never again
	//
	// +optional
	ExpirationDate *string `json:"expirationDate,omitempty"`

	// ApplicationID is the ID of the application the key belongs to. Either
	// `applicationID`, `applicationRef` or `applicationSelector` must be set.
	//
	// +optional
	ApplicationID *string `json:"applicationID,omitempty"`

	// ApplicationRef references an OIDCApplication or ApplicationAPI managed by
	// this provider and uses its ID.
	//
	// +optional
	ApplicationRef *xpv1.Reference `json:"applicationRef,omitempty"`

	// ApplicationSelector selects an OIDCApplication or ApplicationAPI managed by
	// this provider and uses its ID.
	//
	// +optional
	ApplicationSelector *xpv1.Selector `json:"applicationSelector,omitempty"`
}

// ApplicationKeyObservation is what Zitadel reports about an application key.
type ApplicationKeyObservation struct {
	ID             string `json:"id,omitempty"`
	ProjectID      string `json:"projectID,omitempty"`
	ApplicationID  string `json:"applicationID,omitempty"`
	CreationDate   string `json:"creationDate,omitempty"`
	ExpirationDate string `json:"expirationDate,omitempty"`
}

// ApplicationKeyStatus reports the observed state of a hold the key an application authenticates itself with.
type ApplicationKeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider ApplicationKeyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// ApplicationKey is a managed resource that hold the key an application authenticates itself with.
//
// The key is written to the connection secret as `key.json`, the same shape as
// a machine key, because Zitadel returns the private half exactly once: at
// creation. There is no way to read it again, so a key whose secret was lost
// can only be replaced.
// +kubebuilder:object:generate=true
type ApplicationKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationKeySpec   `json:"spec"`
	Status ApplicationKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationKeyList contains a list of ApplicationKey.
type ApplicationKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationKey `json:"items"`
}

// ApplicationKey type metadata.
var (
	ApplicationKeyKind             = reflect.TypeOf(ApplicationKey{}).Name()
	ApplicationKeyGroupKind        = schema.GroupKind{Group: Group, Kind: ApplicationKeyKind}.String()
	ApplicationKeyKindAPIVersion   = ApplicationKeyKind + "." + SchemeGroupVersion.String()
	ApplicationKeyGroupVersionKind = SchemeGroupVersion.WithKind(ApplicationKeyKind)
)

func init() {
	SchemeBuilder.Register(&ApplicationKey{}, &ApplicationKeyList{})
}
