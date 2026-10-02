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

// DefaultPasswordAgePolicyParameters are the configurable fields of a DefaultPasswordAgePolicy.
type DefaultPasswordAgePolicyParameters struct {
	// Days after which a password expires. Zitadel treats 0 as "never".
	// +optional
	MaxAgeDays *int64 `json:"maxAgeDays,omitempty"`

	// Days before expiry at which the user is warned. Setting this without
	// `maxAgeDays` has no effect.
	// +optional
	ExpireWarnDays *int64 `json:"expireWarnDays,omitempty"`
}

// DefaultPasswordAgePolicyObservation is what the instance wide password expiry policy of an organization currently is.
type DefaultPasswordAgePolicyObservation struct {
	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// Days after which a password expires. Zitadel treats 0 as "never".
	// +optional
	MaxAgeDays *int64 `json:"maxAgeDays,omitempty"`

	// Days before expiry at which the user is warned. Setting this without
	// `maxAgeDays` has no effect.
	// +optional
	ExpireWarnDays *int64 `json:"expireWarnDays,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A DefaultPasswordAgePolicy is the instance's instance wide password expiry policy.
//
// It is namespaced like every other managed resource here, because that
// is what its ProviderConfig reference needs. What it manages is not:
// there is one lockout policy for the whole instance whichever namespace
// the resource lives in.
//
// make passwords expire everywhere in the instance.
//
// It is a singleton: there is one of each for the whole instance. Zitadel
// offers no way to reset an instance wide policy - there is no default
// beneath it - so deleting this resource writes back the value it
// overwrote rather than removing anything.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Default",type="boolean",JSONPath=".status.atProvider.isDefault"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type DefaultPasswordAgePolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DefaultPasswordAgePolicySpec   `json:"spec"`
	Status DefaultPasswordAgePolicyStatus `json:"status,omitempty"`
}

// DefaultPasswordAgePolicySpec defines the desired state of a DefaultPasswordAgePolicy.
type DefaultPasswordAgePolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              DefaultPasswordAgePolicyParameters `json:"forProvider"`
}

// DefaultPasswordAgePolicyStatus is the observed state of a DefaultPasswordAgePolicy.
type DefaultPasswordAgePolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DefaultPasswordAgePolicyObservation `json:"atProvider,omitempty"`
}

// DefaultPasswordAgePolicyList contains a list of DefaultPasswordAgePolicy.
//
// +kubebuilder:object:root=true
type DefaultPasswordAgePolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefaultPasswordAgePolicy `json:"items"`
}

// DefaultPasswordAgePolicy type metadata.
var (
	DefaultPasswordAgePolicyKind             = reflect.TypeOf(DefaultPasswordAgePolicy{}).Name()
	DefaultPasswordAgePolicyGroupKind        = schema.GroupKind{Group: Group, Kind: DefaultPasswordAgePolicyKind}.String()
	DefaultPasswordAgePolicyKindAPIVersion   = DefaultPasswordAgePolicyKind + "." + SchemeGroupVersion.String()
	DefaultPasswordAgePolicyGroupVersionKind = SchemeGroupVersion.WithKind(DefaultPasswordAgePolicyKind)
)

func init() {
	SchemeBuilder.Register(&DefaultPasswordAgePolicy{}, &DefaultPasswordAgePolicyList{})
}

// SetPolicyScope does nothing: an instance wide policy has no organization.
func (mg *DefaultPasswordAgePolicy) SetPolicyScope(string) {}

// PolicyScope is always empty, which is what tells the harness that this
// policy belongs to the instance rather than to an organization.
func (mg *DefaultPasswordAgePolicy) PolicyScope() string { return "" }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *DefaultPasswordAgePolicy) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *DefaultPasswordAgePolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
