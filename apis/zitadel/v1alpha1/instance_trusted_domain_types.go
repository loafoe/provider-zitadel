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

// InstanceTrustedDomainSpec defines the desired state of a InstanceTrustedDomain.
type InstanceTrustedDomainSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              InstanceTrustedDomainParameters `json:"forProvider"`
}

// InstanceTrustedDomainParameters is the desired configuration of a Zitadel instance trusted domain.
type InstanceTrustedDomainParameters struct {
	// The domain allowed to ask this Zitadel instance for one of its tokens,
	// which is how an application outside the instance authenticates a user
	// against it.
	//
	// It is required: it is what the entry is called, and therefore what
	// identifies it in Zitadel.
	Domain *string `json:"domain"`
}

// InstanceTrustedDomainObservation is what Zitadel reports about a Zitadel instance trusted domain.
type InstanceTrustedDomainObservation struct {
	Domain string `json:"domain,omitempty"`
}

// InstanceTrustedDomainStatus reports the observed state of a Zitadel instance trusted domain.
type InstanceTrustedDomainStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider InstanceTrustedDomainObservation `json:"atProvider,omitempty"`

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

// InstanceTrustedDomain is a managed resource that let a domain of your own request a token from this instance.
//
// A domain is a name in a list and nothing more: there is no identifier to
// look up, nothing to compare once it is present, and nothing to restore.
// Changing the name therefore leaves the old entry behind, because removing
// something another resource may also be managing is not this resource's call.
// +kubebuilder:object:generate=true
type InstanceTrustedDomain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceTrustedDomainSpec   `json:"spec"`
	Status InstanceTrustedDomainStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InstanceTrustedDomainList contains a list of InstanceTrustedDomain.
type InstanceTrustedDomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanceTrustedDomain `json:"items"`
}

// InstanceTrustedDomain type metadata.
var (
	InstanceTrustedDomainKind             = reflect.TypeOf(InstanceTrustedDomain{}).Name()
	InstanceTrustedDomainGroupKind        = schema.GroupKind{Group: Group, Kind: InstanceTrustedDomainKind}.String()
	InstanceTrustedDomainKindAPIVersion   = InstanceTrustedDomainKind + "." + SchemeGroupVersion.String()
	InstanceTrustedDomainGroupVersionKind = SchemeGroupVersion.WithKind(InstanceTrustedDomainKind)
)

func init() {
	SchemeBuilder.Register(&InstanceTrustedDomain{}, &InstanceTrustedDomainList{})
}

// NamedName returns the name the entry should have, which is also its identity.
func (mg *InstanceTrustedDomain) NamedName() string { return Deref(mg.Spec.ForProvider.Domain) }

// NamedSetScope records the owner the entry was last seen in.
func (mg *InstanceTrustedDomain) NamedSetScope(scope string) { mg.Status.Scope = scope }

// NamedScope returns the recorded owner.
func (mg *InstanceTrustedDomain) NamedScope() string { return mg.Status.Scope }
