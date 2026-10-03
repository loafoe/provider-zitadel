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

// SMSProviderTwilioSpec defines the desired state of a SMSProviderTwilio.
type SMSProviderTwilioSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              SMSProviderTwilioParameters `json:"forProvider"`
}

// SMSProviderTwilioParameters is the desired configuration of a Zitadel Twilio SMS provider.
type SMSProviderTwilioParameters struct {
	// SID controls the following. The Twilio account SID
	//
	// It is required: Zitadel cannot do without it.
	SID *string `json:"sid,omitempty"`
	// TokenSecretRef controls the following. The Twilio auth token. Zitadel never returns it and sets it through a call of its own, so it is applied and then left alone. It is a secret rather than a field so that it is not readable by anyone who can read this object
	//
	//
	// +optional
	TokenSecretRef *SecretKeySelector `json:"tokenSecretRef,omitempty"`
	// SenderNumber controls the following. The number messages are sent from, in E.164 form such as `+441234567890`
	//
	//
	// +optional
	SenderNumber *string `json:"senderNumber,omitempty"`
	// VerifyServiceSID controls the following. The SID of a Twilio Verify service, when messages go through one rather than through the account directly
	//
	//
	// +optional
	VerifyServiceSID *string `json:"verifyServiceSID,omitempty"`
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

// SMSProviderTwilioObservation is what Zitadel reports about a Zitadel Twilio SMS provider.
type SMSProviderTwilioObservation struct {
	ID               string `json:"id,omitempty"`
	State            string `json:"state,omitempty"`
	Kind             string `json:"kind,omitempty"`
	SID              string `json:"sid,omitempty"`
	SenderNumber     string `json:"senderNumber,omitempty"`
	VerifyServiceSID string `json:"verifyServiceSID,omitempty"`
	Description      string `json:"description,omitempty"`
}

// SMSProviderTwilioStatus reports the observed state of a Zitadel Twilio SMS provider.
type SMSProviderTwilioStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider SMSProviderTwilioObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// SMSProviderTwilio is a managed resource that have Zitadel send SMS through Twilio.
//
// This is a namespaced object managing state that belongs to the whole
// Zitadel instance. Zitadel sends through one provider of each kind, so
// two of these of the same kind in different namespaces would be asking
// for the same thing and the second would win.
// +kubebuilder:object:generate=true
type SMSProviderTwilio struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SMSProviderTwilioSpec   `json:"spec"`
	Status SMSProviderTwilioStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SMSProviderTwilioList contains a list of SMSProviderTwilio.
type SMSProviderTwilioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SMSProviderTwilio `json:"items"`
}

// SMSProviderTwilio type metadata.
var (
	SMSProviderTwilioKind             = reflect.TypeOf(SMSProviderTwilio{}).Name()
	SMSProviderTwilioGroupKind        = schema.GroupKind{Group: Group, Kind: SMSProviderTwilioKind}.String()
	SMSProviderTwilioKindAPIVersion   = SMSProviderTwilioKind + "." + SchemeGroupVersion.String()
	SMSProviderTwilioGroupVersionKind = SchemeGroupVersion.WithKind(SMSProviderTwilioKind)
)

func init() {
	SchemeBuilder.Register(&SMSProviderTwilio{}, &SMSProviderTwilioList{})
}
