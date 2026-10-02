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

// PrivacyPolicyParameters are the configurable fields of a PrivacyPolicy.
type PrivacyPolicyParameters struct {
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

// PrivacyPolicyObservation is what the privacy policy of an organization currently is.
type PrivacyPolicyObservation struct {
	// OrganizationID is the organization the policy belongs to.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

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

	// The organization the policy was last read from.
	// +optional
	Scope *string `json:"scope,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A PrivacyPolicy is an organization's privacy policy.
//
// sets the legal and support links Zitadel shows.
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
type PrivacyPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrivacyPolicySpec   `json:"spec"`
	Status PrivacyPolicyStatus `json:"status,omitempty"`
}

// PrivacyPolicySpec defines the desired state of a PrivacyPolicy.
type PrivacyPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              PrivacyPolicyParameters `json:"forProvider"`
}

// PrivacyPolicyStatus is the observed state of a PrivacyPolicy.
type PrivacyPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PrivacyPolicyObservation `json:"atProvider,omitempty"`
}

// PrivacyPolicyList contains a list of PrivacyPolicy.
//
// +kubebuilder:object:root=true
type PrivacyPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrivacyPolicy `json:"items"`
}

// PrivacyPolicy type metadata.
var (
	PrivacyPolicyKind             = reflect.TypeOf(PrivacyPolicy{}).Name()
	PrivacyPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: PrivacyPolicyKind}.String()
	PrivacyPolicyKindAPIVersion   = PrivacyPolicyKind + "." + SchemeGroupVersion.String()
	PrivacyPolicyGroupVersionKind = SchemeGroupVersion.WithKind(PrivacyPolicyKind)
)

func init() {
	SchemeBuilder.Register(&PrivacyPolicy{}, &PrivacyPolicyList{})
}

// SetPolicyScope records the organization the policy belongs to.
func (mg *PrivacyPolicy) SetPolicyScope(scope string) { mg.Status.AtProvider.Scope = StringPtr(scope) }

// PolicyScope returns the organization the policy was last read from.
func (mg *PrivacyPolicy) PolicyScope() string { return Deref(mg.Status.AtProvider.Scope) }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *PrivacyPolicy) SetPolicyRestore(restore []byte) { mg.Status.AtProvider.Restore = restore }

// PolicyRestore returns the recorded restore point.
func (mg *PrivacyPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
