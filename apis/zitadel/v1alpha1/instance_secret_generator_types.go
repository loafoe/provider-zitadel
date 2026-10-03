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

// InstanceSecretGeneratorSpec defines the desired state of a InstanceSecretGenerator.
type InstanceSecretGeneratorSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              InstanceSecretGeneratorParameters `json:"forProvider"`
}

// InstanceSecretGeneratorParameters is the desired configuration of a Zitadel instance secret generator.
type InstanceSecretGeneratorParameters struct {
	// GeneratorType controls the following. Which of Zitadel's codes this shapes, such as `SECRET_GENERATOR_TYPE_INIT_CODE` or `SECRET_GENERATOR_TYPE_APP_SECRET`. Each type is a generator of its own, so two of these resources may manage two different types at once
	//
	// It is required.
	GeneratorType *string `json:"generatorType,omitempty"`
	// Length controls the following. How many characters the code has. Zitadel refuses a length below four
	//
	// It is optional.
	Length *int64 `json:"length,omitempty"`
	// Expiry controls the following. How long the code stays usable, as a Go duration such as `5m` or `1h`. Zitadel refuses an expiry that is longer than the code's own lifetime
	//
	// It is optional.
	Expiry *string `json:"expiry,omitempty"`
	// IncludeLowerLetters controls the following. Let the code contain lower case letters
	//
	// It is optional.
	IncludeLowerLetters *bool `json:"includeLowerLetters,omitempty"`
	// IncludeUpperLetters controls the following. Let the code contain upper case letters
	//
	// It is optional.
	IncludeUpperLetters *bool `json:"includeUpperLetters,omitempty"`
	// IncludeDigits controls the following. Let the code contain digits
	//
	// It is optional.
	IncludeDigits *bool `json:"includeDigits,omitempty"`
	// IncludeSymbols controls the following. Let the code contain symbols
	//
	// It is optional.
	IncludeSymbols *bool `json:"includeSymbols,omitempty"`
}

// InstanceSecretGeneratorObservation is what Zitadel reports about a Zitadel instance secret generator.
type InstanceSecretGeneratorObservation struct {
	// Scope is what identifies these settings. It is empty for a
	// singleton and carries the generator type for the keyed kind.
	Scope string `json:"scope,omitempty"`

	GeneratorType       string `json:"generatorType,omitempty"`
	Length              int64  `json:"length,omitempty"`
	Expiry              string `json:"expiry,omitempty"`
	IncludeLowerLetters bool   `json:"includeLowerLetters,omitempty"`
	IncludeUpperLetters bool   `json:"includeUpperLetters,omitempty"`
	IncludeDigits       bool   `json:"includeDigits,omitempty"`
	IncludeSymbols      bool   `json:"includeSymbols,omitempty"`
}

// InstanceSecretGeneratorStatus reports the observed state of a Zitadel instance secret generator.
type InstanceSecretGeneratorStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider InstanceSecretGeneratorObservation `json:"atProvider,omitempty"`

	// Restore is the value this resource overwrote, kept so that deleting
	// it can put it back.
	Restore []byte `json:"restore,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// InstanceSecretGenerator is a managed resource that shape one of the codes Zitadel sends out.
//
// This is a namespaced object managing state that belongs to the whole
// Zitadel instance. Two of them in different namespaces would be asking
// for the same settings, and the second would win.
// +kubebuilder:object:generate=true
type InstanceSecretGenerator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceSecretGeneratorSpec   `json:"spec"`
	Status InstanceSecretGeneratorStatus `json:"status,omitempty"`
}

// SetPolicyScope records the scope these settings were last read from.
func (mg *InstanceSecretGenerator) SetPolicyScope(scope string) { mg.Status.AtProvider.Scope = scope }

// PolicyScope returns the recorded scope.
func (mg *InstanceSecretGenerator) PolicyScope() string { return mg.Status.AtProvider.Scope }

// SetPolicyRestore records the value to put back when this resource is deleted.
func (mg *InstanceSecretGenerator) SetPolicyRestore(restore []byte) { mg.Status.Restore = restore }

// PolicyRestore returns the recorded restore point.
func (mg *InstanceSecretGenerator) PolicyRestore() []byte { return mg.Status.Restore }

// +kubebuilder:object:root=true

// InstanceSecretGeneratorList contains a list of InstanceSecretGenerator.
type InstanceSecretGeneratorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanceSecretGenerator `json:"items"`
}

// InstanceSecretGenerator type metadata.
var (
	InstanceSecretGeneratorKind             = reflect.TypeOf(InstanceSecretGenerator{}).Name()
	InstanceSecretGeneratorGroupKind        = schema.GroupKind{Group: Group, Kind: InstanceSecretGeneratorKind}.String()
	InstanceSecretGeneratorKindAPIVersion   = InstanceSecretGeneratorKind + "." + SchemeGroupVersion.String()
	InstanceSecretGeneratorGroupVersionKind = SchemeGroupVersion.WithKind(InstanceSecretGeneratorKind)
)

func init() {
	SchemeBuilder.Register(&InstanceSecretGenerator{}, &InstanceSecretGeneratorList{})
}
