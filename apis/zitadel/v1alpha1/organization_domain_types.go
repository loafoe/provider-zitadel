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

// OrganizationDomainSpec defines the desired state of a OrganizationDomain.
type OrganizationDomainSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OrganizationDomainParameters `json:"forProvider"`
}

// OrganizationDomainParameters is the desired configuration of a Zitadel organization domain.
type OrganizationDomainParameters struct {
	// OrganizationID is the ID of the organization the domain belongs to.
	// Either `organizationID`, `organizationRef` or `organizationSelector`
	// must be set.
	//
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// OrganizationRef references an Organization managed by this provider and
	// uses its ID.
	//
	// +optional
	OrganizationRef *xpv1.Reference `json:"organizationRef,omitempty"`

	// OrganizationSelector selects an Organization managed by this provider and
	// uses its ID.
	//
	// +optional
	OrganizationSelector *xpv1.Selector `json:"organizationSelector,omitempty"`

	// The domain the organization owns. Zitadel adds it unverified; it
	// becomes verified once the organization has proved it owns the domain,
	// which `verify` starts but does not finish.
	//
	// It is required: it is what the entry is called, and therefore what
	// identifies it in Zitadel.
	Domain *string `json:"domain"`

	// ValidationType controls the following. How the domain is proved to belong to the organization: `DOMAIN_VALIDATION_TYPE_DNS` or `DOMAIN_VALIDATION_TYPE_HTTP`. `DNS` asks for a TXT record, `HTTP` for a file at a URL Zitadel gives. Only asked for when `verify` is set
	//
	// +optional
	ValidationType *string `json:"validationType,omitempty"`
	// Verify controls the following. Ask Zitadel to check that the organization has proved it owns the domain. Proving it is something the organization does afterwards, so this only starts the check: the token or URL it reports is in the status, and the domain becomes verified once Zitadel sees the proof
	//
	// +optional
	Verify *bool `json:"verify,omitempty"`
}

// OrganizationDomainObservation is what Zitadel reports about a Zitadel organization domain.
type OrganizationDomainObservation struct {
	Domain          string `json:"domain,omitempty"`
	IsVerified      bool   `json:"isVerified,omitempty"`
	IsPrimary       bool   `json:"isPrimary,omitempty"`
	ValidationType  string `json:"validationType,omitempty"`
	ValidationToken string `json:"validationToken,omitempty"`
	ValidationURL   string `json:"validationURL,omitempty"`
}

// OrganizationDomainStatus reports the observed state of a Zitadel organization domain.
type OrganizationDomainStatus struct {
	xpv1.ResourceStatus `json:",inline"`

	// AtProvider is the observed state.
	AtProvider OrganizationDomainObservation `json:"atProvider,omitempty"`

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

// OrganizationDomain is a managed resource that let an organization own a domain and sign users in with it.
//
// A domain is a name in a list and nothing more: there is no identifier to
// look up, nothing to compare once it is present, and nothing to restore.
// Changing the name therefore leaves the old entry behind, because removing
// something another resource may also be managing is not this resource's call.
// +kubebuilder:object:generate=true
type OrganizationDomain struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OrganizationDomainSpec   `json:"spec"`
	Status OrganizationDomainStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OrganizationDomainList contains a list of OrganizationDomain.
type OrganizationDomainList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OrganizationDomain `json:"items"`
}

// OrganizationDomain type metadata.
var (
	OrganizationDomainKind             = reflect.TypeOf(OrganizationDomain{}).Name()
	OrganizationDomainGroupKind        = schema.GroupKind{Group: Group, Kind: OrganizationDomainKind}.String()
	OrganizationDomainKindAPIVersion   = OrganizationDomainKind + "." + SchemeGroupVersion.String()
	OrganizationDomainGroupVersionKind = SchemeGroupVersion.WithKind(OrganizationDomainKind)
)

func init() {
	SchemeBuilder.Register(&OrganizationDomain{}, &OrganizationDomainList{})
}

// NamedName returns the name the entry should have, which is also its identity.
func (mg *OrganizationDomain) NamedName() string { return Deref(mg.Spec.ForProvider.Domain) }

// NamedSetScope records the owner the entry was last seen in.
func (mg *OrganizationDomain) NamedSetScope(scope string) { mg.Status.Scope = scope }

// NamedScope returns the recorded owner.
func (mg *OrganizationDomain) NamedScope() string { return mg.Status.Scope }
