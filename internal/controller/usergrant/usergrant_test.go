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

package usergrant

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// A user grant can grant roles either on a project the user's own organization
// owns or on a project shared with it. Which one decides where the grant is
// looked up, so an unset field has to mean "the user's own project" rather than
// "no grant".

func TestProjectGrantID(t *testing.T) {
	id := "pg-1"

	cases := map[string]struct {
		reason string
		in     *string
		want   string
	}{
		"Unset": {
			reason: "No grant named means the roles are granted on a project of the user's own organization",
			in:     nil,
			want:   "",
		},
		"Set": {
			reason: "A named grant is used",
			in:     &id,
			want:   "pg-1",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.UserGrant{Spec: v1alpha1.UserGrantSpec{
				ForProvider: v1alpha1.UserGrantParameters{ProjectGrantID: tc.in},
			}}

			if got := projectGrantID(cr); got != tc.want {
				t.Errorf("\n%s\nprojectGrantID(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.UserGrant{}

	updateStatus(cr, "org-1", &zitadel.UserGrant{
		ID:             "ug-1",
		UserID:         "u1",
		ProjectID:      "p1",
		ProjectGrantID: "pg-1",
		RoleKeys:       []string{"reader"},
		State:          "Active",
	})

	ap := cr.Status.AtProvider

	for name, tc := range map[string]struct{ got, want *string }{
		"GrantID":        {ap.GrantID, ptr("ug-1")},
		"UserID":         {ap.UserID, ptr("u1")},
		"OrganizationID": {ap.OrganizationID, ptr("org-1")},
		"ProjectID":      {ap.ProjectID, ptr("p1")},
		"ProjectGrantID": {ap.ProjectGrantID, ptr("pg-1")},
	} {
		if tc.got == nil || *tc.got != *tc.want {
			t.Errorf("%s: want %q, got %v", name, *tc.want, tc.got)
		}
	}

	if ap.State == nil || string(*ap.State) != "Active" {
		t.Errorf("State: want %q, got %v", "Active", ap.State)
	}

	if len(ap.RoleKeys) != 1 {
		t.Errorf("RoleKeys: want 1, got %d", len(ap.RoleKeys))
	}
}

// TestUpdateStatusOnAnOwnProject covers a grant on a project of the user's own
// organization, which has no project grant behind it.
func TestUpdateStatusOnAnOwnProject(t *testing.T) {
	cr := &v1alpha1.UserGrant{}

	updateStatus(cr, "org-1", &zitadel.UserGrant{
		ID:        "ug-1",
		UserID:    "u1",
		ProjectID: "p1",
		RoleKeys:  []string{"reader"},
	})

	if ap := cr.Status.AtProvider; ap.ProjectGrantID != nil {
		t.Errorf("ProjectGrantID: want it left unset for a project's own grant, got %v", *ap.ProjectGrantID)
	}

	if ap := cr.Status.AtProvider; ap.State != nil {
		t.Errorf("State: want it left unset when Zitadel reported none, got %v", *ap.State)
	}
}

func ptr(s string) *string { return &s }
