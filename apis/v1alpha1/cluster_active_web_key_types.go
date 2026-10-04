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

// ClusterActiveWebKeySpec is the desired state of a ClusterActiveWebKey.
//
// It carries the same parameters as the namespaced ActiveWebKey, so the two kinds
// cannot describe a different thing from one another.
type ClusterActiveWebKeySpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              managed.ActiveWebKeyParameters `json:"forProvider"`
}

// ClusterActiveWebKeyStatus is the observed state of a ClusterActiveWebKey.
//
// It is the namespaced status under another name, for the same reason: one
// definition of what a field means.
type ClusterActiveWebKeyStatus = managed.ActiveWebKeyStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// ClusterActiveWebKey is the cluster scoped form of ActiveWebKey.
//
// It manages which of Zitadel's signing keys is the active one.
//
// There is one active key on an instance, so a namespaced copy in two
// namespaces would be activating different keys at once.
//
// The namespaced kind is kept and behaves identically. Use whichever suits the
// cluster: a namespaced one to keep instance configuration beside the team or
// environment that owns it, this one when it should be unambiguous.
//
// It references a ClusterProviderConfig, because a cluster scoped object cannot
// name a secret in a namespace.
// +kubebuilder:object:generate=true
type ClusterActiveWebKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterActiveWebKeySpec   `json:"spec"`
	Status ClusterActiveWebKeyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterActiveWebKeyList contains a list of ClusterActiveWebKey.
type ClusterActiveWebKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterActiveWebKey `json:"items"`
}

// ClusterActiveWebKey type metadata.
var (
	ClusterActiveWebKeyKind             = reflect.TypeOf(ClusterActiveWebKey{}).Name()
	ClusterActiveWebKeyGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterActiveWebKeyKind}.String()
	ClusterActiveWebKeyKindAPIVersion   = ClusterActiveWebKeyKind + "." + SchemeGroupVersion.String()
	ClusterActiveWebKeyGroupVersionKind = SchemeGroupVersion.WithKind(ClusterActiveWebKeyKind)
)

func init() {
	SchemeBuilder.Register(&ClusterActiveWebKey{}, &ClusterActiveWebKeyList{})
}

// AsNamespaced returns this resource as the namespaced ActiveWebKey it mirrors.
//
// The controller that already reconciles ActiveWebKey is reused unchanged: this is
// the same object, the same fields and the same external resource, so it is
// handed over as the kind it knows. Nothing is copied that could disagree -
// there is only one definition of the spec, the observation and the status.
func (mg *ClusterActiveWebKey) AsNamespaced() *managed.ActiveWebKey {
	return &managed.ActiveWebKey{
		TypeMeta:   mg.TypeMeta,
		ObjectMeta: *mg.ObjectMeta.DeepCopy(),
		Spec: managed.ActiveWebKeySpec{
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
func (mg *ClusterActiveWebKey) AdoptNamespaced(src *managed.ActiveWebKey) {
	mg.ObjectMeta = *src.ObjectMeta.DeepCopy()
	mg.Status = src.Status
}
