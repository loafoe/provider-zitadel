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

// ApplicationSAMLSpec defines the desired state of a ApplicationSAML.
type ApplicationSAMLSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ApplicationSAMLParameters `json:"forProvider"`
}

// ApplicationSAMLParameters is the desired configuration of a Zitadel SAML application.
type ApplicationSAMLParameters struct {
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

	// Name controls the following. The application's name, shown on the login page
	//
	// +optional
	Name *string `json:"name,omitempty"`
	// MetadataXML controls the following. The identity provider's metadata document itself. This is the form that always works, because Zitadel does not have to fetch anything
	//
	// +optional
	MetadataXML *string `json:"metadata,omitempty"`
	// MetadataURL controls the following. Where Zitadel fetches the metadata from. Only worth using when Zitadel can reach that URL: a document given as `metadata` always works
	//
	// +optional
	MetadataURL *string `json:"metadataURL,omitempty"`
	// LoginVersion controls the following. Which login screen the user is sent to: `LoginV1` or `LoginV2`. Empty leaves the instance default
	//
	// +optional
	LoginVersion *string `json:"loginVersion,omitempty"`
}

// ApplicationSAMLObservation is what Zitadel reports about a Zitadel SAML application.
type ApplicationSAMLObservation struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	ProjectID    string `json:"projectID,omitempty"`
	State        string `json:"state,omitempty"`
	MetadataXML  string `json:"metadataXML,omitempty"`
	MetadataURL  string `json:"metadataURL,omitempty"`
	LoginVersion string `json:"loginVersion,omitempty"`
}

// ApplicationSAMLStatus reports the observed state of a Zitadel SAML application.
type ApplicationSAMLStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider ApplicationSAMLObservation `json:"atProvider,omitempty"`

	// WebKeyID is the key that was last found to be active, recorded so that a
	// terminating resource can report on it without following a reference that
	// may since have moved.
	WebKeyID string `json:"webKeyID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// ApplicationSAML is a managed resource that let an application sign users in through SAML.
//
// A SAML application is made and read through the same application API as
// the API and OIDC ones: Zitadel has one application service with three
// types in it, not one service per type. The older SAML-only endpoints still
// exist and are not used.
//
// Zitadel requires one form of the metadata or the other, so exactly one of
// `metadata` and `metadataURL` must be set.
// +kubebuilder:object:generate=true
type ApplicationSAML struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSAMLSpec   `json:"spec"`
	Status ApplicationSAMLStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationSAMLList contains a list of ApplicationSAML.
type ApplicationSAMLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationSAML `json:"items"`
}

// ApplicationSAML type metadata.
var (
	ApplicationSAMLKind             = reflect.TypeOf(ApplicationSAML{}).Name()
	ApplicationSAMLGroupKind        = schema.GroupKind{Group: Group, Kind: ApplicationSAMLKind}.String()
	ApplicationSAMLKindAPIVersion   = ApplicationSAMLKind + "." + SchemeGroupVersion.String()
	ApplicationSAMLGroupVersionKind = SchemeGroupVersion.WithKind(ApplicationSAMLKind)
)

func init() {
	SchemeBuilder.Register(&ApplicationSAML{}, &ApplicationSAMLList{})
}
