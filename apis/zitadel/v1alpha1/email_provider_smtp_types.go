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

// MessageProviderState is whether Zitadel sends through a provider. A provider
// that is inactive is configured but not offered to anyone.
type MessageProviderState string

const (
	// MessageProviderStateActive means Zitadel sends through this provider.
	MessageProviderStateActive MessageProviderState = "Active"

	// MessageProviderStateInactive means the provider is configured but not
	// used. Zitadel refuses to deactivate the last one it has, so an instance
	// needs another before this can be set.
	MessageProviderStateInactive MessageProviderState = "Inactive"
)

// EmailProviderSMTPSpec defines the desired state of a EmailProviderSMTP.
type EmailProviderSMTPSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              EmailProviderSMTPParameters `json:"forProvider"`
}

// EmailProviderSMTPParameters is the desired configuration of a Zitadel SMTP email provider.
type EmailProviderSMTPParameters struct {
	// Host controls the following. The SMTP server, as `host:port`, such as `smtp.example.com:587`. Zitadel does not connect to it when the provider is added, only when something is sent, so it can be set up ahead of a server that is not reachable yet
	//
	// It is required: Zitadel cannot do without it.
	Host *string `json:"host,omitempty"`
	// User controls the following. The account Zitadel authenticates to the server as
	//
	//
	// +optional
	User *string `json:"user,omitempty"`
	// PasswordSecretRef controls the following. That account's password. Zitadel never returns it, so it is applied and then left alone: an update that carries none leaves the stored password as it is. It is a secret rather than a field so that it is not readable by anyone who can read this object
	//
	//
	// +optional
	PasswordSecretRef *SecretKeySelector `json:"passwordSecretRef,omitempty"`
	// SenderAddress controls the following. The address Zitadel sends from, such as `noreply@example.com`. It has to be a domain the organization owns
	//
	//
	// +optional
	SenderAddress *string `json:"senderAddress,omitempty"`
	// SenderName controls the following. The name shown as the sender of an email Zitadel sends
	//
	//
	// +optional
	SenderName *string `json:"senderName,omitempty"`
	// ReplyToAddress controls the following. Where a reply should go, when that is not the sender address
	//
	//
	// +optional
	ReplyToAddress *string `json:"replyToAddress,omitempty"`
	// TLS controls the following. Connect over TLS. Zitadel calls this `tls` rather than `starttls`: it means the connection is wrapped from the start, which is what port 465 expects
	//
	//
	// +optional
	TLS *bool `json:"tls,omitempty"`
	// Description controls the following. What this provider is for, shown in the Zitadel console
	//
	//
	// +optional
	Description *string `json:"description,omitempty"`

	// State is whether Zitadel sends through this provider. Empty leaves
	// Zitadel's own choice, which is to send through it.
	//
	// +optional
	State *MessageProviderState `json:"state,omitempty"`
}

// EmailProviderSMTPObservation is what Zitadel reports about a Zitadel SMTP email provider.
type EmailProviderSMTPObservation struct {
	ID             string `json:"id,omitempty"`
	State          string `json:"state,omitempty"`
	Kind           string `json:"kind,omitempty"`
	Host           string `json:"host,omitempty"`
	User           string `json:"user,omitempty"`
	SenderAddress  string `json:"senderAddress,omitempty"`
	SenderName     string `json:"senderName,omitempty"`
	ReplyToAddress string `json:"replyToAddress,omitempty"`
	TLS            bool   `json:"tls,omitempty"`
	Description    string `json:"description,omitempty"`
}

// EmailProviderSMTPStatus reports the observed state of a Zitadel SMTP email provider.
type EmailProviderSMTPStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider EmailProviderSMTPObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// EmailProviderSMTP is a managed resource that have Zitadel send email through an SMTP server.
//
// This is a namespaced object managing state that belongs to the whole
// Zitadel instance. Zitadel sends through one provider of each kind, so
// two of these of the same kind in different namespaces would be asking
// for the same thing and the second would win.
// +kubebuilder:object:generate=true
type EmailProviderSMTP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EmailProviderSMTPSpec   `json:"spec"`
	Status EmailProviderSMTPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EmailProviderSMTPList contains a list of EmailProviderSMTP.
type EmailProviderSMTPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EmailProviderSMTP `json:"items"`
}

// EmailProviderSMTP type metadata.
var (
	EmailProviderSMTPKind             = reflect.TypeOf(EmailProviderSMTP{}).Name()
	EmailProviderSMTPGroupKind        = schema.GroupKind{Group: Group, Kind: EmailProviderSMTPKind}.String()
	EmailProviderSMTPKindAPIVersion   = EmailProviderSMTPKind + "." + SchemeGroupVersion.String()
	EmailProviderSMTPGroupVersionKind = SchemeGroupVersion.WithKind(EmailProviderSMTPKind)
)

func init() {
	SchemeBuilder.Register(&EmailProviderSMTP{}, &EmailProviderSMTPList{})
}
