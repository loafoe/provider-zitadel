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

// ApplicationAPIParameters are the configurable fields of a ApplicationAPI.
type ApplicationAPIParameters struct {
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

	// Name is the display name of the application.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// ID allows setting a custom application ID.
	// +optional
	ID *string `json:"id,omitempty"`

	// AuthMethodType is how clients authenticate: Basic sends the client
	// secret, PrivateKeyJwt uses a signed JWT. Defaults to Basic.
	// +optional
	// +kubebuilder:default=Basic
	AuthMethodType *APIAuthMethodType `json:"authMethodType,omitempty"`

	// GeneratePrivateKey asks the provider to generate an RSA key pair for the
	// application and write it to the connection secret, so a workload
	// authenticating with PrivateKeyJwt has a key to sign with.
	// +optional
	GeneratePrivateKey *bool `json:"generatePrivateKey,omitempty"`

	// State of the application. Defaults to Active.
	// +optional
	State *ApplicationState `json:"state,omitempty"`
}

// ApplicationAPIObservation are the observable fields of a ApplicationAPI.
type ApplicationAPIObservation struct {
	// ApplicationID is the Zitadel application ID.
	// +optional
	ApplicationID *string `json:"applicationID,omitempty"`

	// ProjectID is the project the application belongs to.
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// Name is the display name of the application.
	// +optional
	Name *string `json:"name,omitempty"`

	// State of the application.
	// +optional
	State *ApplicationState `json:"state,omitempty"`

	// ClientID is the client ID of the application.
	// +optional
	ClientID *string `json:"clientID,omitempty"`

	// AuthMethodType of the application.
	// +optional
	AuthMethodType *APIAuthMethodType `json:"authMethodType,omitempty"`

	// CreationDate is when the application was created.
	// +optional
	CreationDate *metav1.Time `json:"creationDate,omitempty"`

	// ChangeDate is when the application was last modified.
	// +optional
	ChangeDate *metav1.Time `json:"changeDate,omitempty"`
}

// A ApplicationAPISpec defines the desired state of a ApplicationAPI.
type ApplicationAPISpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ApplicationAPIParameters `json:"forProvider"`
}

// A ApplicationAPIStatus represents the observed state of a ApplicationAPI.
type ApplicationAPIStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ApplicationAPIObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="PROJECT",type="string",JSONPath=".status.atProvider.projectID"
// +kubebuilder:printcolumn:name="CLIENT-ID",type="string",JSONPath=".status.atProvider.clientID"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// // An ApplicationAPI is a Zitadel API application: a client that authenticates
// service to service calls against the Zitadel API or the Auth API. Use it when
// a workload needs a client identity of its own, as opposed to a service
// account used for a single workload.
type ApplicationAPI struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationAPISpec   `json:"spec"`
	Status ApplicationAPIStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationAPIList contains a list of ApplicationAPI.
type ApplicationAPIList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationAPI `json:"items"`
}

// ApplicationAPI type metadata.
var (
	ApplicationAPIKind             = reflect.TypeOf(ApplicationAPI{}).Name()
	ApplicationAPIGroupKind        = schema.GroupKind{Group: Group, Kind: ApplicationAPIKind}.String()
	ApplicationAPIKindAPIVersion   = ApplicationAPIKind + "." + SchemeGroupVersion.String()
	ApplicationAPIGroupVersionKind = SchemeGroupVersion.WithKind(ApplicationAPIKind)
)

func init() {
	SchemeBuilder.Register(&ApplicationAPI{}, &ApplicationAPIList{})
}

// GetApplicationID returns the Zitadel identifier of the application.
//
// A key belongs to an application of either kind, so a reference to one is
// resolved through this rather than reaching into each kind's status.
func (mg *ApplicationAPI) GetApplicationID() string { return Deref(mg.Status.AtProvider.ApplicationID) }
