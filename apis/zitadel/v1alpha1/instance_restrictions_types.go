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

// InstanceRestrictionsSpec defines the desired state of a InstanceRestrictions.
type InstanceRestrictionsSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              InstanceRestrictionsParameters `json:"forProvider"`
}

// InstanceRestrictionsParameters is the desired configuration of a Zitadel instance registration restrictions.
type InstanceRestrictionsParameters struct {
	// DisallowPublicOrgRegistration controls the following. Stop anyone registering their own organization. Off by default, so turning it off lets anyone sign up an organization again
	//
	// It is optional.
	DisallowPublicOrgRegistration *bool `json:"disallowPublicOrgRegistration,omitempty"`
	// AllowedLanguages controls the following. The languages the login screen offers, such as `en` and `de`. Empty means every language Zitadel supports
	//
	// It is optional.
	AllowedLanguages []string `json:"allowedLanguages,omitempty"`
}

// InstanceRestrictionsObservation is what Zitadel reports about a Zitadel instance registration restrictions.
type InstanceRestrictionsObservation struct {
	// Scope is what identifies these settings. It is empty for a
	// singleton and carries the generator type for the keyed kind.
	Scope string `json:"scope,omitempty"`

	DisallowPublicOrgRegistration bool     `json:"disallowPublicOrgRegistration,omitempty"`
	AllowedLanguages              []string `json:"allowedLanguages,omitempty"`
}

// InstanceRestrictionsStatus reports the observed state of a Zitadel instance registration restrictions.
type InstanceRestrictionsStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider InstanceRestrictionsObservation `json:"atProvider,omitempty"`

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

// InstanceRestrictions is a managed resource that limit what anyone may register on this instance.
//
// This is a namespaced object managing state that belongs to the whole
// Zitadel instance. Two of them in different namespaces would be asking
// for the same settings, and the second would win.
// +kubebuilder:object:generate=true
type InstanceRestrictions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceRestrictionsSpec   `json:"spec"`
	Status InstanceRestrictionsStatus `json:"status,omitempty"`
}

// SetPolicyScope records the scope these settings were last read from.
func (mg *InstanceRestrictions) SetPolicyScope(scope string) { mg.Status.AtProvider.Scope = scope }

// PolicyScope returns the recorded scope.
func (mg *InstanceRestrictions) PolicyScope() string { return mg.Status.AtProvider.Scope }

// SetPolicyRestore records the value to put back when this resource is deleted.
func (mg *InstanceRestrictions) SetPolicyRestore(restore []byte) { mg.Status.Restore = restore }

// PolicyRestore returns the recorded restore point.
func (mg *InstanceRestrictions) PolicyRestore() []byte { return mg.Status.Restore }

// +kubebuilder:object:root=true

// InstanceRestrictionsList contains a list of InstanceRestrictions.
type InstanceRestrictionsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanceRestrictions `json:"items"`
}

// InstanceRestrictions type metadata.
var (
	InstanceRestrictionsKind             = reflect.TypeOf(InstanceRestrictions{}).Name()
	InstanceRestrictionsGroupKind        = schema.GroupKind{Group: Group, Kind: InstanceRestrictionsKind}.String()
	InstanceRestrictionsKindAPIVersion   = InstanceRestrictionsKind + "." + SchemeGroupVersion.String()
	InstanceRestrictionsGroupVersionKind = SchemeGroupVersion.WithKind(InstanceRestrictionsKind)
)

func init() {
	SchemeBuilder.Register(&InstanceRestrictions{}, &InstanceRestrictionsList{})
}
