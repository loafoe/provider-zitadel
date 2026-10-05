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

package projectgrantmember

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
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
		Roles:       []string{"admin", "reader"},
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.ProjectGrantMember{}
	updateStatus(cr, "org-1", "p1", "g1", full())

	ap := cr.Status.AtProvider
	if ap.UserID == nil || *ap.UserID != "u1" {
		t.Errorf("UserID: want %q, got %v", "u1", ap.UserID)
	}

	for name, tc := range map[string]struct {
		got  *string
		want string
	}{
		"OrganizationID": {ap.OrganizationID, "org-1"},
		"ProjectID":      {ap.ProjectID, "p1"},
		"GrantID":        {ap.GrantID, "g1"},
	} {
		if tc.got == nil || *tc.got != tc.want {
			t.Errorf("%s: want %q, got %v", name, tc.want, tc.got)
		}
	}

	if diff := len(ap.RoleKeys); diff != 2 {
		t.Errorf("RoleKeys: want 2, got %d", diff)
	}
}

// TestUpdateStatusLeavesAbsentFieldsAlone covers the reason the optional fields
// are guarded: a later observation that simply does not carry a field must not
// erase what an earlier one recorded.
func TestUpdateStatusLeavesAbsentFieldsAlone(t *testing.T) {
	cr := &v1alpha1.ProjectGrantMember{}

	// Record a complete observation first.
	updateStatus(cr, "org-1", "p1", "g1", full())

	// Then an observation carrying only the required fields.
	updateStatus(cr, "org-1", "p1", "g1", &zitadel.Membership{UserID: "u1"})

	ap := cr.Status.AtProvider
	if ap.DisplayName == nil || *ap.DisplayName != "Ada Lovelace" {
		t.Errorf("DisplayName: a later observation erased %v, want it left at %q", ap.DisplayName, "Ada Lovelace")
	}

	if ap.Email == nil || *ap.Email != "ada@example.com" {
		t.Errorf("Email: a later observation erased %v, want it left at %q", ap.Email, "ada@example.com")
	}
}
