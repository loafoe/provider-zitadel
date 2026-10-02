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

// DefaultSecuritySettingsParameters are the configurable fields of a DefaultSecuritySettings.
type DefaultSecuritySettingsParameters struct {
	// Allow users to impersonate other users. Zitadel requires an organization
	// role for impersonation regardless of this setting.
	// +optional
	EnableImpersonation *bool `json:"enableImpersonation,omitempty"`

	// Allow Zitadel's login pages to be embedded in an iframe.
	// +optional
	EmbeddedIframe *bool `json:"embeddedIframe,omitempty"`

	// The origins allowed to embed Zitadel, as origins with a scheme, such as
	// `https://app.example.com`. Needs `embeddedIframe`.
	// +optional
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`
}

// DefaultSecuritySettingsObservation is what the instance wide security settings of an organization currently is.
type DefaultSecuritySettingsObservation struct {
	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// Allow users to impersonate other users. Zitadel requires an organization
	// role for impersonation regardless of this setting.
	// +optional
	EnableImpersonation *bool `json:"enableImpersonation,omitempty"`

	// Allow Zitadel's login pages to be embedded in an iframe.
	// +optional
	EmbeddedIframe *bool `json:"embeddedIframe,omitempty"`

	// The origins allowed to embed Zitadel, as origins with a scheme, such as
	// `https://app.example.com`. Needs `embeddedIframe`.
	// +optional
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A DefaultSecuritySettings is the instance's instance wide security settings.
//
// It is namespaced like every other managed resource here, because that
// is what its ProviderConfig reference needs. What it manages is not:
// there is one lockout policy for the whole instance whichever namespace
// the resource lives in.
//
// control impersonation and whether Zitadel may be embedded in a frame.
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
type DefaultSecuritySettings struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DefaultSecuritySettingsSpec   `json:"spec"`
	Status DefaultSecuritySettingsStatus `json:"status,omitempty"`
}

// DefaultSecuritySettingsSpec defines the desired state of a DefaultSecuritySettings.
type DefaultSecuritySettingsSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              DefaultSecuritySettingsParameters `json:"forProvider"`
}

// DefaultSecuritySettingsStatus is the observed state of a DefaultSecuritySettings.
type DefaultSecuritySettingsStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DefaultSecuritySettingsObservation `json:"atProvider,omitempty"`
}

// DefaultSecuritySettingsList contains a list of DefaultSecuritySettings.
//
// +kubebuilder:object:root=true
type DefaultSecuritySettingsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefaultSecuritySettings `json:"items"`
}

// DefaultSecuritySettings type metadata.
var (
	DefaultSecuritySettingsKind             = reflect.TypeOf(DefaultSecuritySettings{}).Name()
	DefaultSecuritySettingsGroupKind        = schema.GroupKind{Group: Group, Kind: DefaultSecuritySettingsKind}.String()
	DefaultSecuritySettingsKindAPIVersion   = DefaultSecuritySettingsKind + "." + SchemeGroupVersion.String()
	DefaultSecuritySettingsGroupVersionKind = SchemeGroupVersion.WithKind(DefaultSecuritySettingsKind)
)

func init() {
	SchemeBuilder.Register(&DefaultSecuritySettings{}, &DefaultSecuritySettingsList{})
}

// SetPolicyScope does nothing: an instance wide policy has no organization.
func (mg *DefaultSecuritySettings) SetPolicyScope(string) {}

// PolicyScope is always empty, which is what tells the harness that this
// policy belongs to the instance rather than to an organization.
func (mg *DefaultSecuritySettings) PolicyScope() string { return "" }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *DefaultSecuritySettings) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *DefaultSecuritySettings) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
