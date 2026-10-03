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

// InstanceCustomDomainSpec defines the desired state of a InstanceCustomDomain.
type InstanceCustomDomainSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              InstanceCustomDomainParameters `json:"forProvider"`
}

// InstanceCustomDomainParameters is the desired configuration of a Zitadel instance custom domain.
type InstanceCustomDomainParameters struct {
	// The domain this instance answers on. Zitadel answers on one generated
	// domain of its own as well, which is not managed here: it is the one
	// Zitadel made, and removing it is not possible.
	//
	// It is required: it is what the entry is called, and therefore what
	// identifies it in Zitadel.
	Domain *string `json:"domain"`
}

// InstanceCustomDomainObservation is what Zitadel reports about a Zitadel instance custom domain.
type InstanceCustomDomainObservation struct {
	Domain    string `json:"domain,omitempty"`
	IsPrimary bool   `json:"isPrimary,omitempty"`
}

// InstanceCustomDomainStatus reports the observed state of a Zitadel instance custom domain.
type InstanceCustomDomainStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider InstanceCustomDomainObservation `json:"atProvider,omitempty"`

	// Scope is the owner the entry was last seen in, recorded so that a
	// terminating resource removes it from where it actually put it.
	Scope string `json:"scope,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={zitadel}
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="DOMAIN",type="string",JSONPath=".spec.forProvider.domain"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"

// InstanceCustomDomain is a managed resource that make this Zitadel instance answer on a domain of its own.
//
// A domain is a name in a list and nothing more: there is no identifier to
// look up, nothing to compare once it is present, and nothing to restore.
// Changing the name therefore leaves the old entry behind, because removing
// something another resource may also be managing is not this resource's call.
// +kubebuilder:object:generate=true
type InstanceCustomDomain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceCustomDomainSpec   `json:"spec"`
	Status InstanceCustomDomainStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InstanceCustomDomainList contains a list of InstanceCustomDomain.
type InstanceCustomDomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanceCustomDomain `json:"items"`
}

// InstanceCustomDomain type metadata.
var (
	InstanceCustomDomainKind             = reflect.TypeOf(InstanceCustomDomain{}).Name()
	InstanceCustomDomainGroupKind        = schema.GroupKind{Group: Group, Kind: InstanceCustomDomainKind}.String()
	InstanceCustomDomainKindAPIVersion   = InstanceCustomDomainKind + "." + SchemeGroupVersion.String()
	InstanceCustomDomainGroupVersionKind = SchemeGroupVersion.WithKind(InstanceCustomDomainKind)
)

func init() {
	SchemeBuilder.Register(&InstanceCustomDomain{}, &InstanceCustomDomainList{})
}

// NamedName returns the name the entry should have, which is also its identity.
func (mg *InstanceCustomDomain) NamedName() string { return Deref(mg.Spec.ForProvider.Domain) }

// NamedSetScope records the owner the entry was last seen in.
func (mg *InstanceCustomDomain) NamedSetScope(scope string) { mg.Status.Scope = scope }

// NamedScope returns the recorded owner.
func (mg *InstanceCustomDomain) NamedScope() string { return mg.Status.Scope }
