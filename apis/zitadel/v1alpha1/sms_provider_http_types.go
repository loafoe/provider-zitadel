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

// SMSProviderHTTPSpec defines the desired state of a SMSProviderHTTP.
type SMSProviderHTTPSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              SMSProviderHTTPParameters `json:"forProvider"`
}

// SMSProviderHTTPParameters is the desired configuration of a Zitadel HTTP SMS provider.
type SMSProviderHTTPParameters struct {
	// Endpoint controls the following. Where Zitadel posts the message. It receives the same payload Zitadel would hand an SMS gateway, so the endpoint has to understand it
	//
	// It is required: Zitadel cannot do without it.
	Endpoint *string `json:"endpoint,omitempty"`
	// Description controls the following. What this provider is for, shown in the Zitadel console
	//
	//
	// +optional
	Description *string `json:"description,omitempty"`

	// State is whether Zitadel sends through this provider. Empty leaves
	// Zitadel's own choice, which is to send through it.
	//
	// +optional
	State *MessageProviderState `json:"state,omitempty"`
}

// SMSProviderHTTPObservation is what Zitadel reports about a Zitadel HTTP SMS provider.
type SMSProviderHTTPObservation struct {
	ID          string `json:"id,omitempty"`
	State       string `json:"state,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Endpoint    string `json:"endpoint,omitempty"`
	Description string `json:"description,omitempty"`
}

// SMSProviderHTTPStatus reports the observed state of a Zitadel HTTP SMS provider.
type SMSProviderHTTPStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider SMSProviderHTTPObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// SMSProviderHTTP is a managed resource that have Zitadel send SMS by posting it to an endpoint.
//
// This is a namespaced object managing state that belongs to the whole
// Zitadel instance. Zitadel sends through one provider of each kind, so
// two of these of the same kind in different namespaces would be asking
// for the same thing and the second would win.
// +kubebuilder:object:generate=true
type SMSProviderHTTP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SMSProviderHTTPSpec   `json:"spec"`
	Status SMSProviderHTTPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SMSProviderHTTPList contains a list of SMSProviderHTTP.
type SMSProviderHTTPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SMSProviderHTTP `json:"items"`
}

// SMSProviderHTTP type metadata.
var (
	SMSProviderHTTPKind             = reflect.TypeOf(SMSProviderHTTP{}).Name()
	SMSProviderHTTPGroupKind        = schema.GroupKind{Group: Group, Kind: SMSProviderHTTPKind}.String()
	SMSProviderHTTPKindAPIVersion   = SMSProviderHTTPKind + "." + SchemeGroupVersion.String()
	SMSProviderHTTPGroupVersionKind = SchemeGroupVersion.WithKind(SMSProviderHTTPKind)
)

func init() {
	SchemeBuilder.Register(&SMSProviderHTTP{}, &SMSProviderHTTPList{})
}
