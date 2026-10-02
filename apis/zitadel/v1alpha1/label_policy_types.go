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

// LabelPolicyParameters are the configurable fields of a LabelPolicy.
type LabelPolicyParameters struct {
	// The ID of the organization the policy belongs to. Either `organizationID`,
	// `organizationRef` or `organizationSelector` must be set.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// OrganizationRef references an Organization managed by this provider and
	// uses its ID.
	// +optional
	OrganizationRef *xpv1.Reference `json:"organizationRef,omitempty"`

	// OrganizationSelector selects an Organization managed by this provider and
	// uses its ID.
	// +optional
	OrganizationSelector *xpv1.Selector `json:"organizationSelector,omitempty"`

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

// LabelPolicyObservation is what the branding policy of an organization currently is.
type LabelPolicyObservation struct {
	// OrganizationID is the organization the policy belongs to.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

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

	// The organization the policy was last read from.
	// +optional
	Scope *string `json:"scope,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A LabelPolicy is an organization's branding policy.
//
// sets the colours, logo and watermark of an organization's login pages.
//
// It is a singleton: one resource describes the policy of one
// organization, and deleting it puts the organization back on the
// instance default rather than removing anything.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Organization",type="string",JSONPath=".status.atProvider.organizationID"
// +kubebuilder:printcolumn:name="Default",type="boolean",JSONPath=".status.atProvider.isDefault"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type LabelPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LabelPolicySpec   `json:"spec"`
	Status LabelPolicyStatus `json:"status,omitempty"`
}

// LabelPolicySpec defines the desired state of a LabelPolicy.
type LabelPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              LabelPolicyParameters `json:"forProvider"`
}

// LabelPolicyStatus is the observed state of a LabelPolicy.
type LabelPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          LabelPolicyObservation `json:"atProvider,omitempty"`
}

// LabelPolicyList contains a list of LabelPolicy.
//
// +kubebuilder:object:root=true
type LabelPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LabelPolicy `json:"items"`
}

// LabelPolicy type metadata.
var (
	LabelPolicyKind             = reflect.TypeOf(LabelPolicy{}).Name()
	LabelPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: LabelPolicyKind}.String()
	LabelPolicyKindAPIVersion   = LabelPolicyKind + "." + SchemeGroupVersion.String()
	LabelPolicyGroupVersionKind = SchemeGroupVersion.WithKind(LabelPolicyKind)
)

func init() {
	SchemeBuilder.Register(&LabelPolicy{}, &LabelPolicyList{})
}

// SetPolicyScope records the organization the policy belongs to.
func (mg *LabelPolicy) SetPolicyScope(scope string) { mg.Status.AtProvider.Scope = StringPtr(scope) }

// PolicyScope returns the organization the policy was last read from.
func (mg *LabelPolicy) PolicyScope() string { return Deref(mg.Status.AtProvider.Scope) }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *LabelPolicy) SetPolicyRestore(restore []byte) { mg.Status.AtProvider.Restore = restore }

// PolicyRestore returns the recorded restore point.
func (mg *LabelPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }

// LabelThemeMode is which theme an organization's login pages use.
//
// +kubebuilder:validation:Enum=Auto;Light;Dark
type LabelThemeMode string

const (
	// LabelThemeModeAuto follows the visitor's own system preference.
	LabelThemeModeAuto LabelThemeMode = "Auto"

	// LabelThemeModeLight always uses the light theme.
	LabelThemeModeLight LabelThemeMode = "Light"

	// LabelThemeModeDark always uses the dark theme.
	LabelThemeModeDark LabelThemeMode = "Dark"
)
