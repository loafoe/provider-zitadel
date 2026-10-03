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

// TriggerActions are the configurable fields of a TriggerActions.
type TriggerActionsParameters struct {
	// The ID of the organization whose login flow is customised. Either
	// `organizationID`, `organizationRef` or `organizationSelector` must be set.
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

	// Which login flow the trigger belongs to.
	// +optional
	FlowType *TriggerFlowType `json:"flowType,omitempty"`

	// Where in that flow the actions run.
	// +optional
	TriggerType *TriggerType `json:"triggerType,omitempty"`

	// The Actions to run at that point.
	// +optional
	ActionIDs []string `json:"actionIDs,omitempty"`
}

// TriggerActionsObservation is what Zitadel currently has.
type TriggerActionsObservation struct {
	// The organization whose login flow is customised.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// The Actions the trigger currently runs.
	// +optional
	ActionIDs []string `json:"actionIDs,omitempty"`
}

// TriggerActions is the actions to run at a point in a login flow.
//
// Bind Actions to a point in a login flow, such as after a user has
// authenticated. Zitadel has no call for listing these, so they are read
// back out of the flow and the wanted trigger picked from it. Deleting
// the resource clears the trigger rather than removing it.
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
type TriggerActions struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TriggerActionsSpec   `json:"spec"`
	Status TriggerActionsStatus `json:"status,omitempty"`
}

// TriggerActionsSpec defines the desired state of a TriggerActions.
type TriggerActionsSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              TriggerActionsParameters `json:"forProvider"`
}

// TriggerActionsStatus is the observed state of a TriggerActions.
type TriggerActionsStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          TriggerActionsObservation `json:"atProvider,omitempty"`
}

// TriggerActionsList contains a list of TriggerActions.
//
// +kubebuilder:object:root=true
type TriggerActionsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TriggerActions `json:"items"`
}

// TriggerActions type metadata.
var (
	TriggerActionsKind             = reflect.TypeOf(TriggerActions{}).Name()
	TriggerActionsGroupKind        = schema.GroupKind{Group: Group, Kind: TriggerActionsKind}.String()
	TriggerActionsKindAPIVersion   = TriggerActionsKind + "." + SchemeGroupVersion.String()
	TriggerActionsGroupVersionKind = SchemeGroupVersion.WithKind(TriggerActionsKind)
)

func init() {
	SchemeBuilder.Register(&TriggerActions{}, &TriggerActionsList{})
}
