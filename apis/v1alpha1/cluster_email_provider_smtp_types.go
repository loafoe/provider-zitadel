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

// ClusterEmailProviderSMTPSpec is the desired state of a ClusterEmailProviderSMTP.
//
// It carries the same parameters as the namespaced EmailProviderSMTP, so the two kinds
// cannot describe a different thing from one another.
type ClusterEmailProviderSMTPSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              managed.EmailProviderSMTPParameters `json:"forProvider"`
}

// ClusterEmailProviderSMTPStatus is the observed state of a ClusterEmailProviderSMTP.
//
// It is the namespaced status under another name, for the same reason: one
// definition of what a field means.
type ClusterEmailProviderSMTPStatus = managed.EmailProviderSMTPStatus

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// ClusterEmailProviderSMTP is the cluster scoped form of EmailProviderSMTP.
//
// It manages the SMTP server Zitadel sends email through.
//
// Zitadel sends through one of these, so two namespaced copies would
// be fighting over the same provider.
//
// The namespaced kind is kept and behaves identically. Use whichever suits the
// cluster: a namespaced one to keep instance configuration beside the team or
// environment that owns it, this one when it should be unambiguous.
//
// It references a ClusterProviderConfig, because a cluster scoped object cannot
// name a secret in a namespace.
// +kubebuilder:object:generate=true
type ClusterEmailProviderSMTP struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterEmailProviderSMTPSpec   `json:"spec"`
	Status ClusterEmailProviderSMTPStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterEmailProviderSMTPList contains a list of ClusterEmailProviderSMTP.
type ClusterEmailProviderSMTPList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterEmailProviderSMTP `json:"items"`
}

// ClusterEmailProviderSMTP type metadata.
var (
	ClusterEmailProviderSMTPKind             = reflect.TypeOf(ClusterEmailProviderSMTP{}).Name()
	ClusterEmailProviderSMTPGroupKind        = schema.GroupKind{Group: Group, Kind: ClusterEmailProviderSMTPKind}.String()
	ClusterEmailProviderSMTPKindAPIVersion   = ClusterEmailProviderSMTPKind + "." + SchemeGroupVersion.String()
	ClusterEmailProviderSMTPGroupVersionKind = SchemeGroupVersion.WithKind(ClusterEmailProviderSMTPKind)
)

func init() {
	SchemeBuilder.Register(&ClusterEmailProviderSMTP{}, &ClusterEmailProviderSMTPList{})
}

// AsNamespaced returns this resource as the namespaced EmailProviderSMTP it mirrors.
//
// The controller that already reconciles EmailProviderSMTP is reused unchanged: this is
// the same object, the same fields and the same external resource, so it is
// handed over as the kind it knows. Nothing is copied that could disagree -
// there is only one definition of the spec, the observation and the status.
func (mg *ClusterEmailProviderSMTP) AsNamespaced() *managed.EmailProviderSMTP {
	return &managed.EmailProviderSMTP{
		TypeMeta:   mg.TypeMeta,
		ObjectMeta: *mg.ObjectMeta.DeepCopy(),
		Spec: managed.EmailProviderSMTPSpec{
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
func (mg *ClusterEmailProviderSMTP) AdoptNamespaced(src *managed.EmailProviderSMTP) {
	mg.ObjectMeta = *src.ObjectMeta.DeepCopy()
	mg.Status = src.Status
}
