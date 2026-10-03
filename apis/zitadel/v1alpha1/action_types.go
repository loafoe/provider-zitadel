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

// Action are the configurable fields of a Action.
type ActionParameters struct {
	// The ID of the organization the action belongs to. Either `organizationID`,
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

	// Name of the action, shown in the console.
	// +optional
	Name *string `json:"name,omitempty"`

	// The JavaScript the action runs. It is sent to Zitadel as written and is
	// never read back, so a change here is applied but never drift detected.
	// +optional
	Script *string `json:"script,omitempty"`

	// How long the action may run before Zitadel gives up, such as `10s`.
	// +optional
	Timeout *string `json:"timeout,omitempty"`

	// Carry on with the next action when this one fails.
	// +optional
	AllowedToFail *bool `json:"allowedToFail,omitempty"`

	// Whether the action runs. One of `Active` or `Inactive`. An inactive action
	// keeps its script and its bindings. Defaults to `Active`.
	// +optional
	State *ActionState `json:"state,omitempty"`
}

// ActionObservation is what Zitadel currently has.
type ActionObservation struct {
	// The organization the action belongs to.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// Zitadel's identifier of the action.
	// +optional
	ID string `json:"id,omitempty"`

	// Name of the action.
	// +optional
	Name *string `json:"name,omitempty"`

	// How long the action may run.
	// +optional
	Timeout *string `json:"timeout,omitempty"`

	// Whether the next action runs when this one fails.
	// +optional
	AllowedToFail *bool `json:"allowedToFail,omitempty"`

	// Whether the action runs. One of `Active` or `Inactive`.
	// +optional
	State *string `json:"state,omitempty"`
}

// Action is a JavaScript snippet Zitadel runs during a login.
//
// A JavaScript snippet Zitadel runs at a point in a login flow. The
// script itself is managed through Zitadel's v1 management API, because
// the v2 action service manages where actions are sent rather than what
// they do.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Organization",type="string",JSONPath=".status.atProvider.organizationID"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type Action struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionSpec   `json:"spec"`
	Status ActionStatus `json:"status,omitempty"`
}

// ActionSpec defines the desired state of a Action.
type ActionSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              ActionParameters `json:"forProvider"`
}

// ActionStatus is the observed state of a Action.
type ActionStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          ActionObservation `json:"atProvider,omitempty"`
}

// ActionList contains a list of Action.
//
// +kubebuilder:object:root=true
type ActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Action `json:"items"`
}

// Action type metadata.
var (
	ActionKind             = reflect.TypeOf(Action{}).Name()
	ActionGroupKind        = schema.GroupKind{Group: Group, Kind: ActionKind}.String()
	ActionKindAPIVersion   = ActionKind + "." + SchemeGroupVersion.String()
	ActionGroupVersionKind = SchemeGroupVersion.WithKind(ActionKind)
)

func init() {
	SchemeBuilder.Register(&Action{}, &ActionList{})
}
