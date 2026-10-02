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

// DefaultLabelPolicyParameters are the configurable fields of a DefaultLabelPolicy.
type DefaultLabelPolicyParameters struct {
	// Primary colour, as a hex value such as `#5282C1`.
	// +optional
	PrimaryColor *string `json:"primaryColor,omitempty"`

	// Colour used for warnings, as a hex value.
	// +optional
	WarnColor *string `json:"warnColor,omitempty"`

	// Page background colour, as a hex value.
	// +optional
	BackgroundColor *string `json:"backgroundColor,omitempty"`

	// Text colour, as a hex value.
	// +optional
	FontColor *string `json:"fontColor,omitempty"`

	// Primary colour of the dark theme, as a hex value.
	// +optional
	PrimaryColorDark *string `json:"primaryColorDark,omitempty"`

	// Warning colour of the dark theme, as a hex value.
	// +optional
	WarnColorDark *string `json:"warnColorDark,omitempty"`

	// Page background colour of the dark theme, as a hex value.
	// +optional
	BackgroundColorDark *string `json:"backgroundColorDark,omitempty"`

	// Text colour of the dark theme, as a hex value.
	// +optional
	FontColorDark *string `json:"fontColorDark,omitempty"`

	// Hide the organization suffix on the login form. Takes effect when the
	// `urn:zitadel:iam:org:domain:primary:{domainname}` scope is requested.
	// +optional
	HideLoginNameSuffix *bool `json:"hideLoginNameSuffix,omitempty"`

	// Hide the Zitadel watermark.
	// +optional
	DisableWatermark *bool `json:"disableWatermark,omitempty"`

	// Which theme the login pages use. One of `Auto`, `Light` or `Dark`. Defaults
	// to `Auto`.
	// +optional
	ThemeMode *LabelThemeMode `json:"themeMode,omitempty"`

	// Where Zitadel serves the logo from. Reported only: assets are uploaded
	// outside the API, so this provider neither sets nor removes them.
	// +optional
	LogoURL *string `json:"logoURL,omitempty"`

	// Where Zitadel serves the favicon from. Reported only, as with `logoURL`.
	// +optional
	IconURL *string `json:"iconURL,omitempty"`

	// Where Zitadel serves the dark theme logo from. Reported only, as with
	// `logoURL`.
	// +optional
	LogoDarkURL *string `json:"logoDarkURL,omitempty"`

	// Where Zitadel serves the dark theme favicon from. Reported only, as with
	// `logoURL`.
	// +optional
	IconDarkURL *string `json:"iconDarkURL,omitempty"`

	// Where Zitadel serves the custom font from. Reported only, as with
	// `logoURL`.
	// +optional
	FontURL *string `json:"fontURL,omitempty"`
}

// DefaultLabelPolicyObservation is what the instance wide branding policy of an organization currently is.
type DefaultLabelPolicyObservation struct {
	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// Primary colour, as a hex value such as `#5282C1`.
	// +optional
	PrimaryColor *string `json:"primaryColor,omitempty"`

	// Colour used for warnings, as a hex value.
	// +optional
	WarnColor *string `json:"warnColor,omitempty"`

	// Page background colour, as a hex value.
	// +optional
	BackgroundColor *string `json:"backgroundColor,omitempty"`

	// Text colour, as a hex value.
	// +optional
	FontColor *string `json:"fontColor,omitempty"`

	// Primary colour of the dark theme, as a hex value.
	// +optional
	PrimaryColorDark *string `json:"primaryColorDark,omitempty"`

	// Warning colour of the dark theme, as a hex value.
	// +optional
	WarnColorDark *string `json:"warnColorDark,omitempty"`

	// Page background colour of the dark theme, as a hex value.
	// +optional
	BackgroundColorDark *string `json:"backgroundColorDark,omitempty"`

	// Text colour of the dark theme, as a hex value.
	// +optional
	FontColorDark *string `json:"fontColorDark,omitempty"`

	// Hide the organization suffix on the login form. Takes effect when the
	// `urn:zitadel:iam:org:domain:primary:{domainname}` scope is requested.
	// +optional
	HideLoginNameSuffix *bool `json:"hideLoginNameSuffix,omitempty"`

	// Hide the Zitadel watermark.
	// +optional
	DisableWatermark *bool `json:"disableWatermark,omitempty"`

	// Which theme the login pages use. One of `Auto`, `Light` or `Dark`. Defaults
	// to `Auto`.
	// +optional
	ThemeMode *LabelThemeMode `json:"themeMode,omitempty"`

	// Where Zitadel serves the logo from. Reported only: assets are uploaded
	// outside the API, so this provider neither sets nor removes them.
	// +optional
	LogoURL *string `json:"logoURL,omitempty"`

	// Where Zitadel serves the favicon from. Reported only, as with `logoURL`.
	// +optional
	IconURL *string `json:"iconURL,omitempty"`

	// Where Zitadel serves the dark theme logo from. Reported only, as with
	// `logoURL`.
	// +optional
	LogoDarkURL *string `json:"logoDarkURL,omitempty"`

	// Where Zitadel serves the dark theme favicon from. Reported only, as with
	// `logoURL`.
	// +optional
	IconDarkURL *string `json:"iconDarkURL,omitempty"`

	// Where Zitadel serves the custom font from. Reported only, as with
	// `logoURL`.
	// +optional
	FontURL *string `json:"fontURL,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A DefaultLabelPolicy is the instance's instance wide branding policy.
//
// It is namespaced like every other managed resource here, because that
// is what its ProviderConfig reference needs. What it manages is not:
// there is one lockout policy for the whole instance whichever namespace
// the resource lives in.
//
// set the colours and theme an organization inherits for its login pages.
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
type DefaultLabelPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DefaultLabelPolicySpec   `json:"spec"`
	Status DefaultLabelPolicyStatus `json:"status,omitempty"`
}

// DefaultLabelPolicySpec defines the desired state of a DefaultLabelPolicy.
type DefaultLabelPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              DefaultLabelPolicyParameters `json:"forProvider"`
}

// DefaultLabelPolicyStatus is the observed state of a DefaultLabelPolicy.
type DefaultLabelPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DefaultLabelPolicyObservation `json:"atProvider,omitempty"`
}

// DefaultLabelPolicyList contains a list of DefaultLabelPolicy.
//
// +kubebuilder:object:root=true
type DefaultLabelPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefaultLabelPolicy `json:"items"`
}

// DefaultLabelPolicy type metadata.
var (
	DefaultLabelPolicyKind             = reflect.TypeOf(DefaultLabelPolicy{}).Name()
	DefaultLabelPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: DefaultLabelPolicyKind}.String()
	DefaultLabelPolicyKindAPIVersion   = DefaultLabelPolicyKind + "." + SchemeGroupVersion.String()
	DefaultLabelPolicyGroupVersionKind = SchemeGroupVersion.WithKind(DefaultLabelPolicyKind)
)

func init() {
	SchemeBuilder.Register(&DefaultLabelPolicy{}, &DefaultLabelPolicyList{})
}

// SetPolicyScope does nothing: an instance wide policy has no organization.
func (mg *DefaultLabelPolicy) SetPolicyScope(string) {}

// PolicyScope is always empty, which is what tells the harness that this
// policy belongs to the instance rather than to an organization.
func (mg *DefaultLabelPolicy) PolicyScope() string { return "" }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *DefaultLabelPolicy) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *DefaultLabelPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
