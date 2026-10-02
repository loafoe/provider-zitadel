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

// DefaultOIDCSettingsParameters are the configurable fields of a DefaultOIDCSettings.
type DefaultOIDCSettingsParameters struct {
	// How long an access token stays valid, such as `1h`.
	// +optional
	AccessTokenLifetime *string `json:"accessTokenLifetime,omitempty"`

	// How long an ID token stays valid.
	// +optional
	IDTokenLifetime *string `json:"idTokenLifetime,omitempty"`

	// How long a refresh token stays valid in total.
	// +optional
	RefreshTokenExpiration *string `json:"refreshTokenExpiration,omitempty"`

	// How long a refresh token stays valid without being used.
	// +optional
	RefreshTokenIdleExpiration *string `json:"refreshTokenIdleExpiration,omitempty"`
}

// DefaultOIDCSettingsObservation is what the instance wide OIDC token lifetimes of an organization currently is.
type DefaultOIDCSettingsObservation struct {
	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// How long an access token stays valid, such as `1h`.
	// +optional
	AccessTokenLifetime *string `json:"accessTokenLifetime,omitempty"`

	// How long an ID token stays valid.
	// +optional
	IDTokenLifetime *string `json:"idTokenLifetime,omitempty"`

	// How long a refresh token stays valid in total.
	// +optional
	RefreshTokenExpiration *string `json:"refreshTokenExpiration,omitempty"`

	// How long a refresh token stays valid without being used.
	// +optional
	RefreshTokenIdleExpiration *string `json:"refreshTokenIdleExpiration,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A DefaultOIDCSettings is the instance's instance wide OIDC token lifetimes.
//
// It is namespaced like every other managed resource here, because that
// is what its ProviderConfig reference needs. What it manages is not:
// there is one lockout policy for the whole instance whichever namespace
// the resource lives in.
//
// set how long the tokens Zitadel issues stay valid.
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
type DefaultOIDCSettings struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DefaultOIDCSettingsSpec   `json:"spec"`
	Status DefaultOIDCSettingsStatus `json:"status,omitempty"`
}

// DefaultOIDCSettingsSpec defines the desired state of a DefaultOIDCSettings.
type DefaultOIDCSettingsSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              DefaultOIDCSettingsParameters `json:"forProvider"`
}

// DefaultOIDCSettingsStatus is the observed state of a DefaultOIDCSettings.
type DefaultOIDCSettingsStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DefaultOIDCSettingsObservation `json:"atProvider,omitempty"`
}

// DefaultOIDCSettingsList contains a list of DefaultOIDCSettings.
//
// +kubebuilder:object:root=true
type DefaultOIDCSettingsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefaultOIDCSettings `json:"items"`
}

// DefaultOIDCSettings type metadata.
var (
	DefaultOIDCSettingsKind             = reflect.TypeOf(DefaultOIDCSettings{}).Name()
	DefaultOIDCSettingsGroupKind        = schema.GroupKind{Group: Group, Kind: DefaultOIDCSettingsKind}.String()
	DefaultOIDCSettingsKindAPIVersion   = DefaultOIDCSettingsKind + "." + SchemeGroupVersion.String()
	DefaultOIDCSettingsGroupVersionKind = SchemeGroupVersion.WithKind(DefaultOIDCSettingsKind)
)

func init() {
	SchemeBuilder.Register(&DefaultOIDCSettings{}, &DefaultOIDCSettingsList{})
}

// SetPolicyScope does nothing: an instance wide policy has no organization.
func (mg *DefaultOIDCSettings) SetPolicyScope(string) {}

// PolicyScope is always empty, which is what tells the harness that this
// policy belongs to the instance rather than to an organization.
func (mg *DefaultOIDCSettings) PolicyScope() string { return "" }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *DefaultOIDCSettings) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *DefaultOIDCSettings) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
