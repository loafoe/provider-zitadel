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

// ActionTargetPublicKey are the configurable fields of a ActionTargetPublicKey.
type ActionTargetPublicKeyParameters struct {
	// The ID of the ActionTarget the key belongs to. Either `targetID`,
	// `targetRef` or `targetSelector` must be set.
	// +optional
	TargetID *string `json:"targetID,omitempty"`

	// TargetRef references an ActionTarget managed by this provider and uses its
	// ID.
	// +optional
	TargetRef *xpv1.Reference `json:"targetRef,omitempty"`

	// TargetSelector selects an ActionTarget managed by this provider and uses
	// its ID.
	// +optional
	TargetSelector *xpv1.Selector `json:"targetSelector,omitempty"`

	// The public key in PEM form, RSA or EC. It cannot be changed in place:
	// Zitadel ties a key to the payload encryption of the target, so rotating one
	// means adding a new key and removing the old.
	// +optional
	PublicKey *string `json:"publicKey,omitempty"`

	// When the key stops being accepted, as an RFC3339 timestamp.
	// +optional
	ExpirationDate *metav1.Time `json:"expirationDate,omitempty"`

	// Whether the key is used to encrypt. Only one key on a target encrypts at a
	// time, so adding a key and activating it is how a rotation is done. Defaults
	// to true.
	// +optional
	Active *bool `json:"active,omitempty"`
}

// ActionTargetPublicKeyObservation is what Zitadel currently has.
type ActionTargetPublicKeyObservation struct {
	// The ActionTarget the key belongs to.
	// +optional
	TargetID *string `json:"targetID,omitempty"`

	// Zitadel's identifier of the key, used as `kid` in the payload header.
	// +optional
	KeyID string `json:"keyID,omitempty"`

	// The public key in PEM form.
	// +optional
	PublicKey *string `json:"publicKey,omitempty"`

	// Whether the key is used to encrypt.
	// +optional
	Active *bool `json:"active,omitempty"`

	// Zitadel's fingerprint of the key.
	// +optional
	Fingerprint string `json:"fingerprint,omitempty"`

	// When the key was added, as an RFC3339 timestamp.
	// +optional
	CreationDate string `json:"creationDate,omitempty"`
}

// ActionTargetPublicKey is a key a target's payloads are encrypted with.
//
// A key that a target's payloads are encrypted with. It belongs to an
// ActionTarget, and is what makes the `Jwe` payload type possible.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Active",type="boolean",JSONPath=".status.atProvider.active"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type ActionTargetPublicKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionTargetPublicKeySpec   `json:"spec"`
	Status ActionTargetPublicKeyStatus `json:"status,omitempty"`
}

// ActionTargetPublicKeySpec defines the desired state of a ActionTargetPublicKey.
type ActionTargetPublicKeySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ActionTargetPublicKeyParameters `json:"forProvider"`
}

// ActionTargetPublicKeyStatus is the observed state of a ActionTargetPublicKey.
type ActionTargetPublicKeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ActionTargetPublicKeyObservation `json:"atProvider,omitempty"`
}

// ActionTargetPublicKeyList contains a list of ActionTargetPublicKey.
//
// +kubebuilder:object:root=true
type ActionTargetPublicKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActionTargetPublicKey `json:"items"`
}

// ActionTargetPublicKey type metadata.
var (
	ActionTargetPublicKeyKind             = reflect.TypeOf(ActionTargetPublicKey{}).Name()
	ActionTargetPublicKeyGroupKind        = schema.GroupKind{Group: Group, Kind: ActionTargetPublicKeyKind}.String()
	ActionTargetPublicKeyKindAPIVersion   = ActionTargetPublicKeyKind + "." + SchemeGroupVersion.String()
	ActionTargetPublicKeyGroupVersionKind = SchemeGroupVersion.WithKind(ActionTargetPublicKeyKind)
)

func init() {
	SchemeBuilder.Register(&ActionTargetPublicKey{}, &ActionTargetPublicKeyList{})
}
