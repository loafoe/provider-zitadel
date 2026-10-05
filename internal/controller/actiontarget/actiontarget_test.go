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

package actiontarget

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Values the cases share, named so that a case reads as a statement about
// behaviour rather than about a repeated literal.
const (
	defaultTimeout = "10s"
)

// Test values, named so that a case reads as a statement about behaviour
// rather than about a repeated string.
const (
	targetName = "notify"
)

// An action target is where an action is sent. Every field but the endpoint is
// optional, and the defaults matter: a target with no explicit type has to be
// sent as a webhook, not as an empty string Zitadel would reject.

func ptr[T any](v T) *T { return &v }

func TestValueOrType(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     *v1alpha1.ActionTargetType
		want   v1alpha1.ActionTargetType
	}{
		"Unset":   {reason: "An unset type defaults to a webhook", in: nil, want: v1alpha1.ActionTargetTypeWebhook},
		"Empty":   {reason: "An empty type defaults to a webhook", in: ptr(v1alpha1.ActionTargetType("")), want: v1alpha1.ActionTargetTypeWebhook},
		"Webhook": {reason: "A webhook is kept", in: ptr(v1alpha1.ActionTargetTypeWebhook), want: v1alpha1.ActionTargetTypeWebhook},
		"Event":   {reason: "Another type is kept", in: ptr(v1alpha1.ActionTargetTypeCall), want: v1alpha1.ActionTargetTypeCall},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := valueOrType(tc.in); got != tc.want {
				t.Errorf("\n%s\nvalueOrType(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

func TestValueOrPayload(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     *v1alpha1.ActionPayloadType
		want   v1alpha1.ActionPayloadType
	}{
		"Unset": {reason: "An unset payload type defaults to JSON", in: nil, want: v1alpha1.ActionPayloadTypeJson},
		"Empty": {reason: "An empty payload type defaults to JSON", in: ptr(v1alpha1.ActionPayloadType("")), want: v1alpha1.ActionPayloadTypeJson},
		"JSON":  {reason: "JSON is kept", in: ptr(v1alpha1.ActionPayloadTypeJson), want: v1alpha1.ActionPayloadTypeJson},
		"Other": {
			reason: "Another payload type is kept",
			in:     ptr(v1alpha1.ActionPayloadTypeJwt),
			want:   v1alpha1.ActionPayloadTypeJwt,
		},
		"EmptyValue": {reason: "An empty value is replaced rather than sent as nothing", in: ptr(v1alpha1.ActionPayloadType("")), want: v1alpha1.ActionPayloadTypeJson},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := valueOrPayload(tc.in); got != tc.want {
				t.Errorf("\n%s\nvalueOrPayload(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

func TestValueOrTimeout(t *testing.T) {
	cases := map[string]struct {
		reason   string
		in       *string
		fallback string
		want     string
	}{
		"Unset": {reason: "An unset timeout falls back", in: nil, fallback: defaultTimeout, want: defaultTimeout},
		"Empty": {reason: "An empty timeout falls back", in: ptr(""), fallback: defaultTimeout, want: defaultTimeout},
		"Set":   {reason: "A timeout the operator gave is used", in: ptr("5s"), fallback: defaultTimeout, want: "5s"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := valueOrTimeout(tc.in, tc.fallback); got != tc.want {
				t.Errorf("\n%s\nvalueOrTimeout(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

func TestDesired(t *testing.T) {
	cases := map[string]struct {
		reason string
		fp     v1alpha1.ActionTargetParameters
		want   zitadel.ActionTarget
	}{
		"Defaults": {
			reason: "An empty spec gets the documented defaults",
			fp:     v1alpha1.ActionTargetParameters{},
			want:   zitadel.ActionTarget{Type: "Webhook", Timeout: defaultTimeout, PayloadType: "Json"},
		},
		"EverythingSet": {
			reason: "Every field the operator gave is carried through",
			fp: v1alpha1.ActionTargetParameters{
				Name:        ptr(targetName),
				Type:        ptr(v1alpha1.ActionTargetTypeCall),
				Endpoint:    ptr("https://example.com/hook"),
				Timeout:     ptr("5s"),
				PayloadType: ptr(v1alpha1.ActionPayloadTypeJwe),
			},
			want: zitadel.ActionTarget{
				Name:        targetName,
				Type:        "Call",
				Endpoint:    "https://example.com/hook",
				Timeout:     "5s",
				PayloadType: "Jwe",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.ActionTarget{Spec: v1alpha1.ActionTargetSpec{ForProvider: tc.fp}}

			if diff := cmp.Diff(tc.want, desired(cr)); diff != "" {
				t.Errorf("\n%s\ndesired(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestIsUpToDate(t *testing.T) {
	observed := &zitadel.ActionTarget{
		ID:          "t1",
		Name:        targetName,
		Type:        "Webhook",
		Endpoint:    "https://example.com/hook",
		Timeout:     defaultTimeout,
		PayloadType: "Json",
	}

	cases := map[string]struct {
		reason string
		fp     v1alpha1.ActionTargetParameters
		want   bool
	}{
		"Unspecified": {
			reason: "A spec that mentions nothing is up to date",
			fp:     v1alpha1.ActionTargetParameters{},
			want:   true,
		},
		"Matching": {
			reason: "Every field the spec set matches",
			fp: v1alpha1.ActionTargetParameters{
				Name:        ptr(targetName),
				Type:        ptr(v1alpha1.ActionTargetTypeWebhook),
				Endpoint:    ptr("https://example.com/hook"),
				Timeout:     ptr(defaultTimeout),
				PayloadType: ptr(v1alpha1.ActionPayloadTypeJson),
			},
			want: true,
		},
		"NameDrift": {
			reason: "A differing name is drift",
			fp:     v1alpha1.ActionTargetParameters{Name: ptr("other")},
			want:   false,
		},
		"TypeDrift": {
			reason: "A differing type is drift",
			fp:     v1alpha1.ActionTargetParameters{Type: ptr(v1alpha1.ActionTargetTypeCall)},
			want:   false,
		},
		"EndpointDrift": {
			reason: "A differing endpoint is drift",
			fp:     v1alpha1.ActionTargetParameters{Endpoint: ptr("https://other.example.com")},
			want:   false,
		},
		"TimeoutDrift": {
			reason: "A differing timeout is drift",
			fp:     v1alpha1.ActionTargetParameters{Timeout: ptr("5s")},
			want:   false,
		},
		"PayloadDrift": {
			reason: "A differing payload type is drift",
			fp:     v1alpha1.ActionTargetParameters{PayloadType: ptr(v1alpha1.ActionPayloadTypeJwe)},
			want:   false,
		},
		"DefaultedTimeout": {
			// The spec left the timeout out, so Zitadel's own value is not
			// drift even though desired() would send 10s.
			reason: "A field the spec left unset is not compared",
			fp:     v1alpha1.ActionTargetParameters{Name: ptr(targetName)},
			want:   true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.ActionTarget{Spec: v1alpha1.ActionTargetSpec{ForProvider: tc.fp}}

			if got := isUpToDate(cr, observed); got != tc.want {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.ActionTarget{}

	updateStatus(cr, &zitadel.ActionTarget{
		ID:          "t1",
		Name:        targetName,
		Type:        "Webhook",
		Endpoint:    "https://example.com/hook",
		Timeout:     defaultTimeout,
		PayloadType: "Json",
		SigningKey:  "signing-key",
	})

	ap := cr.Status.AtProvider

	if ap.ID != "t1" {
		t.Errorf("ID: want %q, got %q", "t1", ap.ID)
	}

	for name, tc := range map[string]struct{ got, want *string }{
		"Name":        {ap.Name, strPtr(targetName)},
		"Type":        {ap.Type, strPtr("Webhook")},
		"Endpoint":    {ap.Endpoint, strPtr("https://example.com/hook")},
		"Timeout":     {ap.Timeout, strPtr(defaultTimeout)},
		"PayloadType": {ap.PayloadType, strPtr("Json")},
	} {
		if diff := cmp.Diff(tc.want, tc.got); diff != "" {
			t.Errorf("%s: -want, +got:\n%s", name, diff)
		}
	}

	if ap.SigningKey != "signing-key" {
		t.Errorf("SigningKey: want %q, got %q", "signing-key", ap.SigningKey)
	}
}

func strPtr(s string) *string { return &s }
