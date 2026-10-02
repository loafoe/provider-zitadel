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

// DefaultPrivacyPolicyParameters are the configurable fields of a DefaultPrivacyPolicy.
type DefaultPrivacyPolicyParameters struct {
	// Link to the Terms of Service.
	// +optional
	TOSLink *string `json:"tosLink,omitempty"`

	// Link to the Privacy Policy.
	// +optional
	PrivacyLink *string `json:"privacyLink,omitempty"`

	// Link to the help or manual page.
	// +optional
	HelpLink *string `json:"helpLink,omitempty"`

	// Support email address shown to users.
	// +optional
	SupportEmail *string `json:"supportEmail,omitempty"`

	// Link to documentation, shown in the console.
	// +optional
	DocsLink *string `json:"docsLink,omitempty"`

	// Link to an external resource, shown as a button in the console.
	// +optional
	CustomLink *string `json:"customLink,omitempty"`

	// Text of the custom link's button. Used only with `customLink`.
	// +optional
	CustomLinkText *string `json:"customLinkText,omitempty"`
}

// DefaultPrivacyPolicyObservation is what the instance wide privacy policy of an organization currently is.
type DefaultPrivacyPolicyObservation struct {
	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// Link to the Terms of Service.
	// +optional
	TOSLink *string `json:"tosLink,omitempty"`

	// Link to the Privacy Policy.
	// +optional
	PrivacyLink *string `json:"privacyLink,omitempty"`

	// Link to the help or manual page.
	// +optional
	HelpLink *string `json:"helpLink,omitempty"`

	// Support email address shown to users.
	// +optional
	SupportEmail *string `json:"supportEmail,omitempty"`

	// Link to documentation, shown in the console.
	// +optional
	DocsLink *string `json:"docsLink,omitempty"`

	// Link to an external resource, shown as a button in the console.
	// +optional
	CustomLink *string `json:"customLink,omitempty"`

	// Text of the custom link's button. Used only with `customLink`.
	// +optional
	CustomLinkText *string `json:"customLinkText,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A DefaultPrivacyPolicy is the instance's instance wide privacy policy.
//
// It is namespaced like every other managed resource here, because that
// is what its ProviderConfig reference needs. What it manages is not:
// there is one lockout policy for the whole instance whichever namespace
// the resource lives in.
//
// set the legal and support links an organization inherits.
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
type DefaultPrivacyPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DefaultPrivacyPolicySpec   `json:"spec"`
	Status DefaultPrivacyPolicyStatus `json:"status,omitempty"`
}

// DefaultPrivacyPolicySpec defines the desired state of a DefaultPrivacyPolicy.
type DefaultPrivacyPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              DefaultPrivacyPolicyParameters `json:"forProvider"`
}

// DefaultPrivacyPolicyStatus is the observed state of a DefaultPrivacyPolicy.
type DefaultPrivacyPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          DefaultPrivacyPolicyObservation `json:"atProvider,omitempty"`
}

// DefaultPrivacyPolicyList contains a list of DefaultPrivacyPolicy.
//
// +kubebuilder:object:root=true
type DefaultPrivacyPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DefaultPrivacyPolicy `json:"items"`
}

// DefaultPrivacyPolicy type metadata.
var (
	DefaultPrivacyPolicyKind             = reflect.TypeOf(DefaultPrivacyPolicy{}).Name()
	DefaultPrivacyPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: DefaultPrivacyPolicyKind}.String()
	DefaultPrivacyPolicyKindAPIVersion   = DefaultPrivacyPolicyKind + "." + SchemeGroupVersion.String()
	DefaultPrivacyPolicyGroupVersionKind = SchemeGroupVersion.WithKind(DefaultPrivacyPolicyKind)
)

func init() {
	SchemeBuilder.Register(&DefaultPrivacyPolicy{}, &DefaultPrivacyPolicyList{})
}

// SetPolicyScope does nothing: an instance wide policy has no organization.
func (mg *DefaultPrivacyPolicy) SetPolicyScope(string) {}

// PolicyScope is always empty, which is what tells the harness that this
// policy belongs to the instance rather than to an organization.
func (mg *DefaultPrivacyPolicy) PolicyScope() string { return "" }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *DefaultPrivacyPolicy) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *DefaultPrivacyPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
