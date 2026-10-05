package projectgrant

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Test values, named so that a case reads as a statement about behaviour
// rather than about a repeated string.
const (
	projID     = "p1"
	granteeOrg = "org-2"
)

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.ProjectGrant{}

	updateStatus(cr, "org-1", &zitadel.ProjectGrant{
		ProjectID:               projID,
		GrantedOrganizationID:   granteeOrg,
		RoleKeys:                []string{"reader"},
		GrantedOrganizationName: "Customer",
		State:                   "Active",
	})

	ap := cr.Status.AtProvider

	for name, tc := range map[string]struct {
		got  *string
		want string
	}{
		"OrganizationID":          {ap.OrganizationID, "org-1"},
		"ProjectID":               {ap.ProjectID, projID},
		"GrantedOrganizationID":   {ap.GrantedOrganizationID, granteeOrg},
		"GrantedOrganizationName": {ap.GrantedOrganizationName, "Customer"},
	} {
		if tc.got == nil || *tc.got != tc.want {
			t.Errorf("%s: want %q, got %v", name, tc.want, tc.got)
		}
	}

	if ap.State == nil || string(*ap.State) != "Active" {
		t.Errorf("State: want %q, got %v", "Active", ap.State)
	}

	if len(ap.RoleKeys) != 1 {
		t.Errorf("RoleKeys: want 1, got %d", len(ap.RoleKeys))
	}
}

// TestUpdateStatusWithoutAnOwner covers a grant whose owning organization could
// not be resolved: the rest of the observation is still worth recording.
func TestUpdateStatusWithoutAnOwner(t *testing.T) {
	cr := &v1alpha1.ProjectGrant{}

	updateStatus(cr, "", &zitadel.ProjectGrant{
		ProjectID:             projID,
		GrantedOrganizationID: granteeOrg,
	})

	if cr.Status.AtProvider.OrganizationID != nil {
		t.Errorf("OrganizationID: want it left unset when the owner is unknown, got %v", cr.Status.AtProvider.OrganizationID)
	}

	if ap := cr.Status.AtProvider; ap.ProjectID == nil || *ap.ProjectID != projID {
		t.Errorf("ProjectID: want %q, got %v", projID, ap.ProjectID)
	}
}

// TestUpdateStatusLeavesAbsentFieldsAlone checks that an observation without a
// state or a name does not erase what an earlier one recorded.
func TestUpdateStatusLeavesAbsentFieldsAlone(t *testing.T) {
	cr := &v1alpha1.ProjectGrant{}

	updateStatus(cr, "org-1", &zitadel.ProjectGrant{
		ProjectID:               projID,
		GrantedOrganizationID:   granteeOrg,
		GrantedOrganizationName: "Customer",
		State:                   "Active",
	})

	updateStatus(cr, "org-1", &zitadel.ProjectGrant{
		ProjectID:             projID,
		GrantedOrganizationID: granteeOrg,
	})

	ap := cr.Status.AtProvider

	if ap.GrantedOrganizationName == nil || *ap.GrantedOrganizationName != "Customer" {
		t.Errorf("GrantedOrganizationName: a later observation erased %v", ap.GrantedOrganizationName)
	}

	if ap.State == nil || string(*ap.State) != "Active" {
		t.Errorf("State: a later observation erased %v", ap.State)
	}
}
