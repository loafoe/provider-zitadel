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

// ActionExecutionResponse are the configurable fields of a ActionExecutionResponse.
type ActionExecutionResponseParameters struct {
	// The ActionTargets to call. Zitadel validates that each one exists before
	// the binding is written, so a typo is reported here rather than at the
	// moment the action would have run.
	// +optional
	TargetIDs []string `json:"targetIDs,omitempty"`

	// The gRPC method whose response triggers the action.
	// +optional
	Method *string `json:"method,omitempty"`

	// The gRPC service whose responses trigger the action. Exactly one of
	// `method`, `service` or `all` must be set.
	// +optional
	Service *string `json:"service,omitempty"`

	// Run on every response Zitadel is about to return. Exactly one of `method`,
	// `service` or `all` must be set.
	// +optional
	All *bool `json:"all,omitempty"`
}

// ActionExecutionResponseObservation is what Zitadel currently has.
type ActionExecutionResponseObservation struct {
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

// ActionExecutionResponse is a response Zitadel is about to return.
//
// Bind ActionTargets to a gRPC response Zitadel is about to return.
// Zitadel gives a binding no identifier of its own: the condition is the
// identity, and reading one back means listing every binding and
// matching. Deleting the resource clears the binding rather than removing
// it, because that is the only thing Zitadel can do.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type ActionExecutionResponse struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionExecutionResponseSpec   `json:"spec"`
	Status ActionExecutionResponseStatus `json:"status,omitempty"`
}

// ActionExecutionResponseSpec defines the desired state of a ActionExecutionResponse.
type ActionExecutionResponseSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ActionExecutionResponseParameters `json:"forProvider"`
}

// ActionExecutionResponseStatus is the observed state of a ActionExecutionResponse.
type ActionExecutionResponseStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ActionExecutionResponseObservation `json:"atProvider,omitempty"`
}

// ActionExecutionResponseList contains a list of ActionExecutionResponse.
//
// +kubebuilder:object:root=true
type ActionExecutionResponseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActionExecutionResponse `json:"items"`
}

// ActionExecutionResponse type metadata.
var (
	ActionExecutionResponseKind             = reflect.TypeOf(ActionExecutionResponse{}).Name()
	ActionExecutionResponseGroupKind        = schema.GroupKind{Group: Group, Kind: ActionExecutionResponseKind}.String()
	ActionExecutionResponseKindAPIVersion   = ActionExecutionResponseKind + "." + SchemeGroupVersion.String()
	ActionExecutionResponseGroupVersionKind = SchemeGroupVersion.WithKind(ActionExecutionResponseKind)
)

func init() {
	SchemeBuilder.Register(&ActionExecutionResponse{}, &ActionExecutionResponseList{})
}
