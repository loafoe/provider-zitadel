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

// ActionExecutionEvent are the configurable fields of a ActionExecutionEvent.
type ActionExecutionEventParameters struct {
	// The ActionTargets to call. Zitadel validates that each one exists before
	// the binding is written, so a typo is reported here rather than at the
	// moment the action would have run.
	// +optional
	TargetIDs []string `json:"targetIDs,omitempty"`

	// The event to run on, such as `user.human.created`.
	// +optional
	Event *string `json:"event,omitempty"`

	// The group of events to run on, such as `user`. Exactly one of `event`,
	// `group` or `all` must be set.
	// +optional
	Group *string `json:"group,omitempty"`

	// Run on every event Zitadel emits. Exactly one of `event`, `group` or `all`
	// must be set.
	// +optional
	All *bool `json:"all,omitempty"`
}

// ActionExecutionEventObservation is what Zitadel currently has.
type ActionExecutionEventObservation struct {
	// The ActionTargets this binding currently calls.
	// +optional
	TargetIDs []string `json:"targetIDs,omitempty"`

	// When the binding was created, as an RFC3339 timestamp.
	// +optional
	CreationDate string `json:"creationDate,omitempty"`

	// When the binding last changed, as an RFC3339 timestamp.
	// +optional
	ChangeDate string `json:"changeDate,omitempty"`
}

// ActionExecutionEvent is an event Zitadel emits.
//
// Bind ActionTargets to a Zitadel event. Zitadel gives a binding no
// identifier of its own: the condition is the identity, and reading one
// back means listing every binding and matching. Deleting the resource
// clears the binding rather than removing it, because that is the only
// thing Zitadel can do.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type ActionExecutionEvent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionExecutionEventSpec   `json:"spec"`
	Status ActionExecutionEventStatus `json:"status,omitempty"`
}

// ActionExecutionEventSpec defines the desired state of a ActionExecutionEvent.
type ActionExecutionEventSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ActionExecutionEventParameters `json:"forProvider"`
}

// ActionExecutionEventStatus is the observed state of a ActionExecutionEvent.
type ActionExecutionEventStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ActionExecutionEventObservation `json:"atProvider,omitempty"`
}

// ActionExecutionEventList contains a list of ActionExecutionEvent.
//
// +kubebuilder:object:root=true
type ActionExecutionEventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActionExecutionEvent `json:"items"`
}

// ActionExecutionEvent type metadata.
var (
	ActionExecutionEventKind             = reflect.TypeOf(ActionExecutionEvent{}).Name()
	ActionExecutionEventGroupKind        = schema.GroupKind{Group: Group, Kind: ActionExecutionEventKind}.String()
	ActionExecutionEventKindAPIVersion   = ActionExecutionEventKind + "." + SchemeGroupVersion.String()
	ActionExecutionEventGroupVersionKind = SchemeGroupVersion.WithKind(ActionExecutionEventKind)
)

func init() {
	SchemeBuilder.Register(&ActionExecutionEvent{}, &ActionExecutionEventList{})
}
