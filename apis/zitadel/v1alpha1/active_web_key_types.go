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

// ActiveWebKeySpec defines the desired state of a ActiveWebKey.
type ActiveWebKeySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ActiveWebKeyParameters `json:"forProvider"`
}

// ActiveWebKeyParameters is the desired configuration of a Zitadel active signing key.
type ActiveWebKeyParameters struct {
	// WebKeyID controls the following. The signing key to activate, as an ID or a reference to a WebKey managed by this provider
	//
	// +optional
	WebKeyID *string `json:"webKeyID,omitempty"`
	// WebKeyRef controls the following. References a WebKey managed by this provider and uses its key
	//
	// +optional
	WebKeyRef xpv1.Reference `json:"webKeyRef,omitempty"`
	// WebKeySelector controls the following. Selects a WebKey managed by this provider and uses its key
	//
	// +optional
	WebKeySelector xpv1.Selector `json:"webKeySelector,omitempty"`
}

// ActiveWebKeyObservation is what Zitadel reports about a Zitadel active signing key.
type ActiveWebKeyObservation struct {
	ID        string `json:"id,omitempty"`
	Algorithm string `json:"algorithm,omitempty"`
	State     string `json:"state,omitempty"`
}

// ActiveWebKeyStatus reports the observed state of a Zitadel active signing key.
type ActiveWebKeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider ActiveWebKeyObservation `json:"atProvider,omitempty"`

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

// ActiveWebKey is a managed resource that point at the signing key Zitadel signs tokens with.
//
// Zitadel signs its tokens with one of its keys, and this is what decides
// which. There is only ever one active key: activating another makes this
// stale and the former one inactive, so two of these in different namespaces
// would be fighting over the same setting.
//
// Zitadel will not remove a key that is active, so a key this resource
// points at has to be activated away before it can be deleted.
// +kubebuilder:object:generate=true
type ActiveWebKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActiveWebKeySpec   `json:"spec"`
	Status ActiveWebKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ActiveWebKeyList contains a list of ActiveWebKey.
type ActiveWebKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActiveWebKey `json:"items"`
}

// ActiveWebKey type metadata.
var (
	ActiveWebKeyKind             = reflect.TypeOf(ActiveWebKey{}).Name()
	ActiveWebKeyGroupKind        = schema.GroupKind{Group: Group, Kind: ActiveWebKeyKind}.String()
	ActiveWebKeyKindAPIVersion   = ActiveWebKeyKind + "." + SchemeGroupVersion.String()
	ActiveWebKeyGroupVersionKind = SchemeGroupVersion.WithKind(ActiveWebKeyKind)
)

func init() {
	SchemeBuilder.Register(&ActiveWebKey{}, &ActiveWebKeyList{})
}
