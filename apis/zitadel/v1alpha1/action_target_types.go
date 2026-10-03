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

// ActionTarget are the configurable fields of a ActionTarget.
type ActionTargetParameters struct {
	// Name of the target, shown in the console.
	// +optional
	Name *string `json:"name,omitempty"`

	// How the target is called. One of `Webhook`, `Call` or `Async`.
	// +optional
	Type *ActionTargetType `json:"type,omitempty"`

	// Where the payload is sent, as an absolute URL.
	// +optional
	Endpoint *string `json:"endpoint,omitempty"`

	// How long Zitadel waits for the target, such as `10s`.
	// +optional
	Timeout *string `json:"timeout,omitempty"`

	// How the payload is encoded. One of `Json`, `Jwt` or `Jwe`. `Jwe` needs an
	// active ActionTargetPublicKey on the target.
	// +optional
	PayloadType *ActionPayloadType `json:"payloadType,omitempty"`
}

// ActionTargetObservation is what Zitadel currently has.
type ActionTargetObservation struct {
	// Zitadel's identifier of the target.
	// +optional
	ID string `json:"id,omitempty"`

	// Name of the target.
	// +optional
	Name *string `json:"name,omitempty"`

	// How the target is called.
	// +optional
	Type *string `json:"type,omitempty"`

	// Where the payload is sent.
	// +optional
	Endpoint *string `json:"endpoint,omitempty"`

	// How long Zitadel waits for the target.
	// +optional
	Timeout *string `json:"timeout,omitempty"`

	// How the payload is encoded.
	// +optional
	PayloadType *string `json:"payloadType,omitempty"`

	// The key Zitadel signs this target's payloads with. Generated once, at
	// creation, and never changed.
	// +optional
	SigningKey string `json:"signingKey,omitempty"`
}

// ActionTarget is somewhere an action's payload is sent.
//
// Somewhere an action's payload is sent: a webhook, a call back into
// Zitadel, or an asynchronous request. Zitadel generates a signing key
// for each target, which is published to the connection secret so that
// the receiving side can verify what it gets.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".status.atProvider.type"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".status.atProvider.endpoint"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type ActionTarget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionTargetSpec   `json:"spec"`
	Status ActionTargetStatus `json:"status,omitempty"`
}

// ActionTargetSpec defines the desired state of a ActionTarget.
type ActionTargetSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ActionTargetParameters `json:"forProvider"`
}

// ActionTargetStatus is the observed state of a ActionTarget.
type ActionTargetStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ActionTargetObservation `json:"atProvider,omitempty"`
}

// ActionTargetList contains a list of ActionTarget.
//
// +kubebuilder:object:root=true
type ActionTargetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActionTarget `json:"items"`
}

// ActionTarget type metadata.
var (
	ActionTargetKind             = reflect.TypeOf(ActionTarget{}).Name()
	ActionTargetGroupKind        = schema.GroupKind{Group: Group, Kind: ActionTargetKind}.String()
	ActionTargetKindAPIVersion   = ActionTargetKind + "." + SchemeGroupVersion.String()
	ActionTargetGroupVersionKind = SchemeGroupVersion.WithKind(ActionTargetKind)
)

func init() {
	SchemeBuilder.Register(&ActionTarget{}, &ActionTargetList{})
}
