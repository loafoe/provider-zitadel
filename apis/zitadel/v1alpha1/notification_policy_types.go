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

// NotificationPolicyParameters are the configurable fields of a NotificationPolicy.
type NotificationPolicyParameters struct {
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

	// Send a notification when a user changes their password.
	// +optional
	PasswordChange *bool `json:"passwordChange,omitempty"`
}

// NotificationPolicyObservation is what the notification policy of an organization currently is.
type NotificationPolicyObservation struct {
	// OrganizationID is the organization the policy belongs to.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// IsDefault is true while the organization still uses the instance default.
	// +optional
	IsDefault *bool `json:"isDefault,omitempty"`

	// Send a notification when a user changes their password.
	// +optional
	PasswordChange *bool `json:"passwordChange,omitempty"`

	// The organization the policy was last read from.
	// +optional
	Scope *string `json:"scope,omitempty"`

	// The policy this resource overwrote, kept so that deleting it can put the
	// previous value back. Only used for a policy Zitadel cannot reset.
	// +optional
	Restore []byte `json:"restore,omitempty"`
}

// A NotificationPolicy is an organization's notification policy.
//
// controls which events make Zitadel send an email.
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
type NotificationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NotificationPolicySpec   `json:"spec"`
	Status NotificationPolicyStatus `json:"status,omitempty"`
}

// NotificationPolicySpec defines the desired state of a NotificationPolicy.
type NotificationPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              NotificationPolicyParameters `json:"forProvider"`
}

// NotificationPolicyStatus is the observed state of a NotificationPolicy.
type NotificationPolicyStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          NotificationPolicyObservation `json:"atProvider,omitempty"`
}

// NotificationPolicyList contains a list of NotificationPolicy.
//
// +kubebuilder:object:root=true
type NotificationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NotificationPolicy `json:"items"`
}

// NotificationPolicy type metadata.
var (
	NotificationPolicyKind             = reflect.TypeOf(NotificationPolicy{}).Name()
	NotificationPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: NotificationPolicyKind}.String()
	NotificationPolicyKindAPIVersion   = NotificationPolicyKind + "." + SchemeGroupVersion.String()
	NotificationPolicyGroupVersionKind = SchemeGroupVersion.WithKind(NotificationPolicyKind)
)

func init() {
	SchemeBuilder.Register(&NotificationPolicy{}, &NotificationPolicyList{})
}

// SetPolicyScope records the organization the policy belongs to.
func (mg *NotificationPolicy) SetPolicyScope(scope string) {
	mg.Status.AtProvider.Scope = StringPtr(scope)
}

// PolicyScope returns the organization the policy was last read from.
func (mg *NotificationPolicy) PolicyScope() string { return Deref(mg.Status.AtProvider.Scope) }

// SetPolicyRestore records the policy to put back when this resource is deleted.
func (mg *NotificationPolicy) SetPolicyRestore(restore []byte) {
	mg.Status.AtProvider.Restore = restore
}

// PolicyRestore returns the recorded restore point.
func (mg *NotificationPolicy) PolicyRestore() []byte { return mg.Status.AtProvider.Restore }
