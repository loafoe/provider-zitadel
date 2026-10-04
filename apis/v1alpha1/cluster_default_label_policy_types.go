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

	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	managed "github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
)

// ClusterDefaultLabelPolicySpec is the desired state of a ClusterDefaultLabelPolicy.
//
// It carries the same parameters as the namespaced DefaultLabelPolicy, so the two kinds
// cannot describe a different thing from one another.
type ClusterDefaultLabelPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              managed.DefaultLabelPolicyParameters `json:"forProvider"`
}

// ClusterDefaultLabelPolicyStatus is the observed state of a ClusterDefaultLabelPolicy.
//
// It is the namespaced status under another name, for the same reason: one
// definition of what a field means.
type ClusterDefaultLabelPolicyStatus = managed.DefaultLabelPolicyStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// ClusterDefaultLabelPolicy is the cluster scoped form of DefaultLabelPolicy.
//
// It manages the instance wide default for the labelPolicy of any organization.
//
// It is the instance default, so it exists once however it is scoped.
//
// The namespaced kind is kept and behaves identically. Use whichever suits the
// cluster: a namespaced one to keep instance configuration beside the team or
// environment that owns it, this one when it should be unambiguous.
//
// It references a ClusterProviderConfig, because a cluster scoped object cannot
// name a secret in a namespace.
// +kubebuilder:object:generate=true
type ClusterDefaultLabelPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterDefaultLabelPolicySpec   `json:"spec"`
	Status ClusterDefaultLabelPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterDefaultLabelPolicyList contains a list of ClusterDefaultLabelPolicy.
type ClusterDefaultLabelPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterDefaultLabelPolicy `json:"items"`
}

// ClusterDefaultLabelPolicy type metadata.
var (
	ClusterDefaultLabelPolicyKind             = reflect.TypeOf(ClusterDefaultLabelPolicy{}).Name()
	ClusterDefaultLabelPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterDefaultLabelPolicyKind}.String()
	ClusterDefaultLabelPolicyKindAPIVersion   = ClusterDefaultLabelPolicyKind + "." + SchemeGroupVersion.String()
	ClusterDefaultLabelPolicyGroupVersionKind = SchemeGroupVersion.WithKind(ClusterDefaultLabelPolicyKind)
)

func init() {
	SchemeBuilder.Register(&ClusterDefaultLabelPolicy{}, &ClusterDefaultLabelPolicyList{})
}

// AsNamespaced returns this resource as the namespaced DefaultLabelPolicy it mirrors.
//
// The controller that already reconciles DefaultLabelPolicy is reused unchanged: this is
// the same object, the same fields and the same external resource, so it is
// handed over as the kind it knows. Nothing is copied that could disagree -
// there is only one definition of the spec, the observation and the status.
func (mg *ClusterDefaultLabelPolicy) AsNamespaced() *managed.DefaultLabelPolicy {
	return &managed.DefaultLabelPolicy{
		TypeMeta:   mg.TypeMeta,
		ObjectMeta: *mg.ObjectMeta.DeepCopy(),
		Spec: managed.DefaultLabelPolicySpec{
			ManagedResourceSpec: *mg.Spec.ManagedResourceSpec.DeepCopy(),
			ForProvider:         mg.Spec.ForProvider,
		},
		Status: mg.Status,
	}
}

// AdoptNamespaced copies back what the namespaced controller recorded.
//
// The external name and the status are what a reconcile produces, so they are
// taken from the resource it was given. The spec is left alone: it is what the
// operator wrote, and it is the same value either way.
func (mg *ClusterDefaultLabelPolicy) AdoptNamespaced(src *managed.DefaultLabelPolicy) {
	mg.ObjectMeta = *src.ObjectMeta.DeepCopy()
	mg.Status = src.Status
}
