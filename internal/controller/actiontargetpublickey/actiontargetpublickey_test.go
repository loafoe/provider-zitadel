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

package actiontargetpublickey

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// A public key exists to be active or not. The field is optional, and an
// operator who never mentioned it is not asking for it to be flipped - which is
// what keeps the controller from activating or deactivating keys on every poll.

func TestValueOrActive(t *testing.T) {
	yes, no := true, false

	cases := map[string]struct {
		reason string
		in     *bool
		want   bool
	}{
		"Unset": {reason: "An unset flag defaults to active, which is what Zitadel does", in: nil, want: true},
		"True":  {reason: "An explicit true is kept", in: &yes, want: true},
		"False": {reason: "An explicit false is kept", in: &no, want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := valueOrActive(tc.in); got != tc.want {
				t.Errorf("\n%s\nvalueOrActive(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestIsUpToDate(t *testing.T) {
	yes, no := true, false

	cases := map[string]struct {
		reason   string
		in       *bool
		observed bool
		want     bool
	}{
		"Unmentioned": {
			reason:   "A spec that says nothing about activity leaves Zitadel's choice alone",
			in:       nil,
			observed: false,
			want:     true,
		},
		"UnmentionedButActive": {
			reason:   "A spec that says nothing does not fight a key Zitadel has active",
			in:       nil,
			observed: true,
			want:     true,
		},
		"WantActiveAndActive": {
			reason:   "An active key the spec wants active is up to date",
			in:       &yes,
			observed: true,
			want:     true,
		},
		"WantActiveButInactive": {
			reason:   "An inactive key the spec wants active is drift",
			in:       &yes,
			observed: false,
			want:     false,
		},
		"WantInactiveButActive": {
			reason:   "An active key the spec wants inactive is drift",
			in:       &no,
			observed: true,
			want:     false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.ActionTargetPublicKey{Spec: v1alpha1.ActionTargetPublicKeySpec{
				ForProvider: v1alpha1.ActionTargetPublicKeyParameters{Active: tc.in},
			}}

			key := &zitadel.ActionTargetPublicKey{Active: tc.observed}
			if got := isUpToDate(cr, key); got != tc.want {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.ActionTargetPublicKey{}

	updateStatus(cr, "t1", &zitadel.ActionTargetPublicKey{
		KeyID:        "k1",
		PublicKey:    "-----BEGIN PUBLIC KEY-----",
		Active:       true,
		Fingerprint:  "AQID",
		CreationDate: "2026-01-01T00:00:00Z",
	})

	ap := cr.Status.AtProvider

	if ap.TargetID == nil || *ap.TargetID != "t1" {
		t.Errorf("TargetID: want %q, got %v", "t1", ap.TargetID)
	}

	if ap.KeyID != "k1" {
		t.Errorf("KeyID: want %q, got %q", "k1", ap.KeyID)
	}

	if ap.PublicKey == nil || *ap.PublicKey != "-----BEGIN PUBLIC KEY-----" {
		t.Errorf("PublicKey: want it recorded, got %v", ap.PublicKey)
	}

	if ap.Active == nil || !*ap.Active {
		t.Errorf("Active: want true, got %v", ap.Active)
	}

	if ap.Fingerprint != "AQID" {
		t.Errorf("Fingerprint: want %q, got %q", "AQID", ap.Fingerprint)
	}
}
