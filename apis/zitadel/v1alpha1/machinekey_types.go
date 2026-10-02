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

// MachineKeyParameters are the configurable fields of a MachineKey.
type MachineKeyParameters struct {
	// ServiceAccountID is the ID of the service account. Either
	// `serviceAccountID`, `serviceAccountRef` or `serviceAccountSelector`
	// must be set.
	// +optional
	ServiceAccountID *string `json:"serviceAccountID,omitempty"`

	// ServiceAccountRef references a ServiceAccount managed by this provider.
	// +optional
	ServiceAccountRef *xpv1.Reference `json:"serviceAccountRef,omitempty"`

	// ServiceAccountSelector selects a ServiceAccount managed by this provider.
	// +optional
	ServiceAccountSelector *xpv1.Selector `json:"serviceAccountSelector,omitempty"`

	// ExpirationDate is when the key stops working. Defaults to one year from
	// creation, because Zitadel requires an expiration date.
	// +optional
	ExpirationDate *metav1.Time `json:"expirationDate,omitempty"`

	// KeyBits is the RSA modulus size of the generated key. Defaults to 2048.
	// Changing it creates a new key.
	// +optional
	// +kubebuilder:validation:Enum=2048;3072;4096
	// +kubebuilder:default=2048
	KeyBits *int `json:"keyBits,omitempty"`
}

// MachineKeyObservation are the observable fields of a MachineKey.
type MachineKeyObservation struct {
	// KeyID is the ID of the machine key in Zitadel.
	// +optional
	KeyID *string `json:"keyID,omitempty"`

	// ServiceAccountID is the service account the key belongs to.
	// +optional
	ServiceAccountID *string `json:"serviceAccountID,omitempty"`

	// OrganizationID is the organization of the service account.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// ExpirationDate is when the key stops working.
	// +optional
	ExpirationDate *metav1.Time `json:"expirationDate,omitempty"`

	// CreationDate is when the key was created.
	// +optional
	CreationDate *metav1.Time `json:"creationDate,omitempty"`
}

// A MachineKeySpec defines the desired state of a MachineKey.
type MachineKeySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              MachineKeyParameters `json:"forProvider"`
}

// A MachineKeyStatus represents the observed state of a MachineKey.
type MachineKeyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          MachineKeyObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="KEY-ID",type="string",JSONPath=".status.atProvider.keyID"
// +kubebuilder:printcolumn:name="SERVICE-ACCOUNT",type="string",JSONPath=".status.atProvider.serviceAccountID"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// // A MachineKey is an RSA key a Zitadel service account authenticates with. The
// provider generates the key pair, registers the public half with Zitadel and
// writes the key.json document to the connection secret, so a workload's
// credentials can be created and rotated without anything leaving the cluster.
type MachineKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineKeySpec   `json:"spec"`
	Status MachineKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MachineKeyList contains a list of MachineKey.
type MachineKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MachineKey `json:"items"`
}

// MachineKey type metadata.
var (
	MachineKeyKind             = reflect.TypeOf(MachineKey{}).Name()
	MachineKeyGroupKind        = schema.GroupKind{Group: Group, Kind: MachineKeyKind}.String()
	MachineKeyKindAPIVersion   = MachineKeyKind + "." + SchemeGroupVersion.String()
	MachineKeyGroupVersionKind = SchemeGroupVersion.WithKind(MachineKeyKind)
)

func init() {
	SchemeBuilder.Register(&MachineKey{}, &MachineKeyList{})
}
