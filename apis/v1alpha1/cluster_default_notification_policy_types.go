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

// ClusterDefaultNotificationPolicySpec is the desired state of a ClusterDefaultNotificationPolicy.
//
// It carries the same parameters as the namespaced DefaultNotificationPolicy, so the two kinds
// cannot describe a different thing from one another.
type ClusterDefaultNotificationPolicySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              managed.DefaultNotificationPolicyParameters `json:"forProvider"`
}

// ClusterDefaultNotificationPolicyStatus is the observed state of a ClusterDefaultNotificationPolicy.
//
// It is the namespaced status under another name, for the same reason: one
// definition of what a field means.
type ClusterDefaultNotificationPolicyStatus = managed.DefaultNotificationPolicyStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// ClusterDefaultNotificationPolicy is the cluster scoped form of DefaultNotificationPolicy.
//
// It manages the instance wide default for the notificationPolicy of any organization.
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
type ClusterDefaultNotificationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterDefaultNotificationPolicySpec   `json:"spec"`
	Status ClusterDefaultNotificationPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterDefaultNotificationPolicyList contains a list of ClusterDefaultNotificationPolicy.
type ClusterDefaultNotificationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterDefaultNotificationPolicy `json:"items"`
}

// ClusterDefaultNotificationPolicy type metadata.
var (
	ClusterDefaultNotificationPolicyKind             = reflect.TypeOf(ClusterDefaultNotificationPolicy{}).Name()
	ClusterDefaultNotificationPolicyGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterDefaultNotificationPolicyKind}.String()
	ClusterDefaultNotificationPolicyKindAPIVersion   = ClusterDefaultNotificationPolicyKind + "." + SchemeGroupVersion.String()
	ClusterDefaultNotificationPolicyGroupVersionKind = SchemeGroupVersion.WithKind(ClusterDefaultNotificationPolicyKind)
)

func init() {
	SchemeBuilder.Register(&ClusterDefaultNotificationPolicy{}, &ClusterDefaultNotificationPolicyList{})
}

// AsNamespaced returns this resource as the namespaced DefaultNotificationPolicy it mirrors.
//
// The controller that already reconciles DefaultNotificationPolicy is reused unchanged: this is
// the same object, the same fields and the same external resource, so it is
// handed over as the kind it knows. Nothing is copied that could disagree -
// there is only one definition of the spec, the observation and the status.
func (mg *ClusterDefaultNotificationPolicy) AsNamespaced() *managed.DefaultNotificationPolicy {
	return &managed.DefaultNotificationPolicy{
		TypeMeta:   mg.TypeMeta,
		ObjectMeta: *mg.ObjectMeta.DeepCopy(),
		Spec: managed.DefaultNotificationPolicySpec{
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
func (mg *ClusterDefaultNotificationPolicy) AdoptNamespaced(src *managed.DefaultNotificationPolicy) {
	mg.ObjectMeta = *src.ObjectMeta.DeepCopy()
	mg.Status = src.Status
}
