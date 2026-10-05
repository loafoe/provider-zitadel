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

package orgmember

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Test values, named so that a case reads as a statement about behaviour
// rather than about a repeated string.
const (
	roleReader = "reader"
	roleAdmin  = "admin"
)

// A membership copies what Zitadel reports into its status, and only that. The
// optional fields are written when Zitadel has something to say and left alone
// when it does not, so that an observation missing a field cannot erase a value
// a previous one recorded.

// full is an observation with everything reported.
func full() *zitadel.Membership {
	return &zitadel.Membership{
		UserID:      "u1",
		DisplayName: "Ada Lovelace",
		Email:       "ada@example.com",
		UserType:    "human",
		Roles:       []string{roleAdmin, roleReader},
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.OrgMember{}
	updateStatus(cr, "org-1", full())

	ap := cr.Status.AtProvider
	if ap.UserID == nil || *ap.UserID != "u1" {
		t.Errorf("UserID: want %q, got %v", "u1", ap.UserID)
	}

	if ap.OrganizationID == nil || *ap.OrganizationID != "org-1" {
		t.Errorf("OrganizationID: want %q, got %v", "org-1", ap.OrganizationID)
	}

	if diff := len(ap.RoleKeys); diff != 2 {
		t.Errorf("RoleKeys: want 2, got %d", diff)
	}

	if ap.DisplayName == nil || *ap.DisplayName != "Ada Lovelace" {
		t.Errorf("DisplayName: want %q, got %v", "Ada Lovelace", ap.DisplayName)
	}
}

// TestUpdateStatusLeavesAbsentFieldsAlone covers the reason the optional fields
// are guarded: a later observation that simply does not carry a field must not
// erase what an earlier one recorded.
func TestUpdateStatusLeavesAbsentFieldsAlone(t *testing.T) {
	cr := &v1alpha1.OrgMember{}

	// Record a complete observation first.
	updateStatus(cr, "org-1", full())

	// Then an observation carrying only the required fields.
	updateStatus(cr, "org-1", &zitadel.Membership{UserID: "u1"})

	ap := cr.Status.AtProvider
	if ap.DisplayName == nil || *ap.DisplayName != "Ada Lovelace" {
		t.Errorf("DisplayName: a later observation erased %v, want it left at %q", ap.DisplayName, "Ada Lovelace")
	}

	if ap.Email == nil || *ap.Email != "ada@example.com" {
		t.Errorf("Email: a later observation erased %v, want it left at %q", ap.Email, "ada@example.com")
	}

	if ap.UserType == nil || *ap.UserType != "human" {
		t.Errorf("UserType: a later observation erased %v, want it left at %q", ap.UserType, "human")
	}
}

func TestIsUpToDate(t *testing.T) {
	cases := map[string]struct {
		reason string
		want   []string
		got    []string
		up     bool
	}{
		"Matching": {
			reason: "The membership holds exactly the roles the spec asks for",
			want:   []string{roleAdmin, roleReader},
			got:    []string{roleAdmin, roleReader},
			up:     true,
		},
		"Reordered": {
			// Order is significant by design, as it is for every role list in
			// this provider: Zitadel reports them in a stable order and the spec
			// is expected to match it.
			reason: "The same roles in a different order are drift",
			want:   []string{roleAdmin, roleReader},
			got:    []string{roleReader, roleAdmin},
			up:     false,
		},
		"ExtraRole": {
			reason: "A role the spec does not ask for is drift",
			want:   []string{roleReader},
			got:    []string{roleReader, roleAdmin},
			up:     false,
		},
		"MissingRole": {
			reason: "A role the spec asks for but Zitadel does not hold is drift",
			want:   []string{roleReader, roleAdmin},
			got:    []string{roleReader},
			up:     false,
		},
		"Empty": {
			reason: "A membership with no roles matches a spec that asks for none",
			want:   nil,
			got:    nil,
			up:     true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.OrgMember{Spec: v1alpha1.OrgMemberSpec{
				ForProvider: v1alpha1.OrgMemberParameters{RoleKeys: tc.want},
			}}

			if got := isUpToDate(cr, &zitadel.Membership{Roles: tc.got}); got != tc.up {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.up, got)
			}
		})
	}
}
