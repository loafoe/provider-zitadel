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

// ActionExecutionFunction are the configurable fields of a ActionExecutionFunction.
type ActionExecutionFunctionParameters struct {
	// The ActionTargets to call. Zitadel validates that each one exists before
	// the binding is written, so a typo is reported here rather than at the
	// moment the action would have run.
	// +optional
	TargetIDs []string `json:"targetIDs,omitempty"`

	// The name of the function to run.
	// +optional
	Name *string `json:"name,omitempty"`
}

// ActionExecutionFunctionObservation is what Zitadel currently has.
type ActionExecutionFunctionObservation struct {
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

// ActionExecutionFunction is a function another action calls.
//
// Bind ActionTargets to a function another action calls. Zitadel gives a
// binding no identifier of its own: the condition is the identity, and
// reading one back means listing every binding and matching. Deleting the
// resource clears the binding rather than removing it, because that is
// the only thing Zitadel can do.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type ActionExecutionFunction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionExecutionFunctionSpec   `json:"spec"`
	Status ActionExecutionFunctionStatus `json:"status,omitempty"`
}

// ActionExecutionFunctionSpec defines the desired state of a ActionExecutionFunction.
type ActionExecutionFunctionSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ActionExecutionFunctionParameters `json:"forProvider"`
}

// ActionExecutionFunctionStatus is the observed state of a ActionExecutionFunction.
type ActionExecutionFunctionStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ActionExecutionFunctionObservation `json:"atProvider,omitempty"`
}

// ActionExecutionFunctionList contains a list of ActionExecutionFunction.
//
// +kubebuilder:object:root=true
type ActionExecutionFunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ActionExecutionFunction `json:"items"`
}

// ActionExecutionFunction type metadata.
var (
	ActionExecutionFunctionKind             = reflect.TypeOf(ActionExecutionFunction{}).Name()
	ActionExecutionFunctionGroupKind        = schema.GroupKind{Group: Group, Kind: ActionExecutionFunctionKind}.String()
	ActionExecutionFunctionKindAPIVersion   = ActionExecutionFunctionKind + "." + SchemeGroupVersion.String()
	ActionExecutionFunctionGroupVersionKind = SchemeGroupVersion.WithKind(ActionExecutionFunctionKind)
)

func init() {
	SchemeBuilder.Register(&ActionExecutionFunction{}, &ActionExecutionFunctionList{})
}
