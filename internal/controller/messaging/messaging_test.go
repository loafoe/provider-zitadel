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

package messaging

import (
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/google/go-cmp/cmp"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Values the cases share, named so that a case reads as a statement about
// behaviour rather than about a repeated literal.
const (
	metadataURL = "https://example.com/md"
)

// Values the cases share, named so that a case reads as a statement about
// behaviour rather than about a repeated literal.
const (
	kindSMTP = "smtp"
	samlXML  = "<EntityDescriptor/>"
)

// The messaging kinds share one driver, so the parts that decide what to send
// and what counts as drift live in one place. Each is a small decision with a
// large blast radius: getting the reference handling wrong makes a resource
// point at the wrong organization, and getting the kind filter wrong makes it
// claim a provider it does not own.

func TestRefOrNil(t *testing.T) {
	ref := xpv1.Reference{Name: "customer"}

	cases := map[string]struct {
		reason string
		in     xpv1.Reference
		want   *xpv1.Reference
	}{
		"Unset": {
			reason: "An empty reference becomes nil, so the resolver falls through to the plain identifier",
			in:     xpv1.Reference{},
			want:   nil,
		},
		"Set": {
			reason: "A named reference is passed through",
			in:     ref,
			want:   &ref,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, refOrNil(tc.in)); diff != "" {
				t.Errorf("\n%s\nrefOrNil(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestSelOrNil(t *testing.T) {
	yes := true
	byLabel := xpv1.Selector{MatchLabels: map[string]string{"team": "platform"}}
	byOwner := xpv1.Selector{MatchControllerRef: &yes}

	cases := map[string]struct {
		reason string
		in     xpv1.Selector
		want   *xpv1.Selector
	}{
		"Unset": {
			reason: "A selector with neither labels nor an owner is nil",
			in:     xpv1.Selector{},
			want:   nil,
		},
		"MatchLabels": {
			reason: "A selector matching labels is passed through",
			in:     byLabel,
			want:   &byLabel,
		},
		"MatchControllerRef": {
			reason: "A selector asking for its controller is passed through",
			in:     byOwner,
			want:   &byOwner,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, selOrNil(tc.in)); diff != "" {
				t.Errorf("\n%s\nselOrNil(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

// TestOfKind covers the filter that stops a driver claiming a provider it does
// not own. Zitadel returns the instance wide provider and the organization one
// in a single list, sharing an identifier space, so a driver that took all of
// them would manage both.
func TestOfKind(t *testing.T) {
	providers := []zitadel.MessageProvider{
		{ID: "1", Kind: kindSMTP},
		{ID: "2", Kind: kindSMTP},
		{ID: "3", Kind: "http"},
		{ID: "4", Kind: "twilio"},
	}

	cases := map[string]struct {
		reason string
		kind   string
		want   []zitadel.MessageProvider
	}{
		"SMTP": {
			reason: "Only the matching providers are kept",
			kind:   kindSMTP,
			want:   []zitadel.MessageProvider{{ID: "1", Kind: kindSMTP}, {ID: "2", Kind: kindSMTP}},
		},
		"HTTP": {
			reason: "A single matching provider is kept",
			kind:   "http",
			want:   []zitadel.MessageProvider{{ID: "3", Kind: "http"}},
		},
		"Absent": {
			reason: "An empty list rather than nil, so a caller can range over it",
			kind:   "carrier-pigeon",
			want:   []zitadel.MessageProvider{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := ofKind(providers, tc.kind)

			if diff := cmp.Diff(tc.want, got, cmp.AllowUnexported(zitadel.MessageProvider{})); diff != "" {
				t.Errorf("\n%s\nofKind(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestProviderState(t *testing.T) {
	active := v1alpha1.MessageProviderState("Active")

	cases := map[string]struct {
		reason string
		in     *v1alpha1.MessageProviderState
		want   string
	}{
		"Unset": {
			reason: "A manifest that asks for no state reports none, so Zitadel's own choice is left alone",
			in:     nil,
			want:   "",
		},
		"Set": {
			reason: "A state the manifest asks for is passed on",
			in:     &active,
			want:   "Active",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := providerState(tc.in); got != tc.want {
				t.Errorf("\n%s\nproviderState(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

// TestMetadataMatches covers the SAML metadata comparison.
//
// Zitadel keeps whichever form it was given and reports only that one, so an
// XML document and the URL serving it are not interchangeable. Comparing across
// the two forms would report drift on a resource that is already correct.
func TestMetadataMatches(t *testing.T) {
	cases := map[string]struct {
		reason   string
		want     zitadel.SAMLApplication
		observed zitadel.SAMLApplication
		wantEq   bool
	}{
		"Unset": {
			reason:   "Metadata the spec does not manage is never compared",
			want:     zitadel.SAMLApplication{},
			observed: zitadel.SAMLApplication{MetadataXML: samlXML, MetadataURL: metadataURL},
			wantEq:   true,
		},
		"XMLMatches": {
			reason:   "The same XML matches",
			want:     zitadel.SAMLApplication{MetadataXML: samlXML},
			observed: zitadel.SAMLApplication{MetadataXML: samlXML},
			wantEq:   true,
		},
		"XMLDiffers": {
			reason:   "Different XML is drift",
			want:     zitadel.SAMLApplication{MetadataXML: samlXML},
			observed: zitadel.SAMLApplication{MetadataXML: "<EntityDescriptor version=\"2\"/>"},
			wantEq:   false,
		},
		"URLMatches": {
			reason:   "The same URL matches",
			want:     zitadel.SAMLApplication{MetadataURL: metadataURL},
			observed: zitadel.SAMLApplication{MetadataURL: metadataURL},
			wantEq:   true,
		},
		"URLDiffers": {
			reason:   "A different URL is drift",
			want:     zitadel.SAMLApplication{MetadataURL: metadataURL},
			observed: zitadel.SAMLApplication{MetadataURL: "https://other.example.com/md"},
			wantEq:   false,
		},
		"XMLDesiredURLObserved": {
			reason:   "XML asked for and a URL reported is not a match, because the two forms are not interchangeable",
			want:     zitadel.SAMLApplication{MetadataXML: samlXML},
			observed: zitadel.SAMLApplication{MetadataURL: metadataURL},
			wantEq:   false,
		},
		"URLDesiredXMLObserved": {
			reason:   "A URL asked for and XML reported is not a match either",
			want:     zitadel.SAMLApplication{MetadataURL: metadataURL},
			observed: zitadel.SAMLApplication{MetadataXML: samlXML},
			wantEq:   false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := metadataMatches(tc.want, tc.observed); got != tc.wantEq {
				t.Errorf("\n%s\nmetadataMatches(...): want %v, got %v", tc.reason, tc.wantEq, got)
			}
		})
	}
}
