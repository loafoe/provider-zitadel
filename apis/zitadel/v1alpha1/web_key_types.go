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

// WebKeySpec defines the desired state of a WebKey.
type WebKeySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              WebKeyParameters `json:"forProvider"`
}

// WebKeyParameters is the desired configuration of a hold one of Zitadel's own signing keys.
type WebKeyParameters struct {

	// Algorithm controls the following. `rsa`, `ecdsa` or `ed25519`. Zitadel generates the key; the private half never leaves it, so there is no material to supply or read back
	//
	// +optional
	Algorithm *string `json:"algorithm,omitempty"`
	// RSABits controls the following. The key size for an RSA key: `RSA_BITS_2048`, `RSA_BITS_3072` or `RSA_BITS_4096`. Zitadel's default is 2048. Ignored for the other algorithms
	//
	// +optional
	RSABits *string `json:"rsaBits,omitempty"`
	// RSAHasher controls the following. The signing algorithm for an RSA key: `RSA_HASHER_SHA256`, `RSA_HASHER_SHA384` or `RSA_HASHER_SHA512`. Zitadel's default is SHA256. Ignored for the other algorithms
	//
	// +optional
	RSAHasher *string `json:"rsaHasher,omitempty"`
	// ECDSACurve controls the following. The curve for an ECDSA key: `ECDSA_CURVE_P256`, `ECDSA_CURVE_P384` or `ECDSA_CURVE_P512`. Zitadel documents P-256 as the default but rejects an unset curve rather than choosing one, so an empty field means P-256. Ignored for the other algorithms
	//
	// +optional
	ECDSACurve *string `json:"ecdsaCurve,omitempty"`
}

// WebKeyObservation is what Zitadel reports about a signing key.
type WebKeyObservation struct {
	ID           string `json:"id,omitempty"`
	Algorithm    string `json:"algorithm,omitempty"`
	State        string `json:"state,omitempty"`
	CreationDate string `json:"creationDate,omitempty"`
	ChangeDate   string `json:"changeDate,omitempty"`
}

// WebKeyStatus reports the observed state of a hold one of Zitadel's own signing keys.
type WebKeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider WebKeyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// WebKey is a managed resource that hold one of Zitadel's own signing keys.
//
// Zitadel signs its tokens with one of these, and an application verifies
// them with the active one. The private half never leaves Zitadel, so a key
// this provider manages is an identity and a state and nothing more:
//
//   - Zitadel creates it inactive, so it verifies nothing until activated.
//     ActivatedWebKey is what does that.
//   - The key type cannot be changed, only replaced, so changing the algorithm
//     is reported as needing a new key rather than silently ignored.
//   - Zitadel refuses to remove the active key, so it has to be activated
//     away first.
//
// +kubebuilder:object:generate=true
type WebKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebKeySpec   `json:"spec"`
	Status WebKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WebKeyList contains a list of WebKey.
type WebKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebKey `json:"items"`
}

// WebKey type metadata.
var (
	WebKeyKind             = reflect.TypeOf(WebKey{}).Name()
	WebKeyGroupKind        = schema.GroupKind{Group: Group, Kind: WebKeyKind}.String()
	WebKeyKindAPIVersion   = WebKeyKind + "." + SchemeGroupVersion.String()
	WebKeyGroupVersionKind = SchemeGroupVersion.WithKind(WebKeyKind)
)

func init() {
	SchemeBuilder.Register(&WebKey{}, &WebKeyList{})
}
