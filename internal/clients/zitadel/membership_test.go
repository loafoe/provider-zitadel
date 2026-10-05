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

package zitadel

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	adminv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	userv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Memberships and grants are the one area with no v2 API, so every call here
// goes through the v1 services. That makes them worth testing in their own
// right: the shape of a request is written out by hand, and the read is a filter
// over a list because ZITADEL has no lookup by anything but a query.

// TestValidateRoles covers the check that turns a confusing failure inside
// ZITADEL into a message that names the problem.
//
// TestOrgMembership covers the organization membership lifecycle end to end:
// read, add, update and remove, all against the management API.
func TestOrgMembership(t *testing.T) {
	t.Run("AddThenRead", func(t *testing.T) {
		zc, m := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddOrgMember(ctx, "org-1", "u-1", []string{"admin"}); err != nil {
			t.Fatalf("AddOrgMember(...): unexpected error: %v", err)
		}

		got, err := zc.GetOrgMember(ctx, "org-1", "u-1")
		if err != nil {
			t.Fatalf("GetOrgMember(...): unexpected error: %v", err)
		}

		if got.UserID != "u-1" {
			t.Errorf("UserID: want %q, got %q", "u-1", got.UserID)
		}

		if diff := cmp.Diff([]string{"admin"}, got.Roles); diff != "" {
			t.Errorf("Roles: -want, +got:\n%s", diff)
		}

		if got.State != MembershipActive {
			t.Errorf("State: want %q, got %q", MembershipActive, got.State)
		}

		// A membership of a real user carries the details a controller shows.
		if got.UserType == "" {
			t.Error("UserType: want the type of a user that has details")
		}

		_ = m
	})

	t.Run("UpdateReplacesTheRoles", func(t *testing.T) {
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddOrgMember(ctx, "org-1", "u-1", []string{"admin"}); err != nil {
			t.Fatalf("AddOrgMember(...): unexpected error: %v", err)
		}

		if err := zc.UpdateOrgMember(ctx, "org-1", "u-1", []string{"reader"}); err != nil {
			t.Fatalf("UpdateOrgMember(...): unexpected error: %v", err)
		}

		got, err := zc.GetOrgMember(ctx, "org-1", "u-1")
		if err != nil {
			t.Fatalf("GetOrgMember(...): unexpected error: %v", err)
		}

		// The roles are replaced rather than added to, or an update would
		// silently accumulate roles.
		if diff := cmp.Diff([]string{"reader"}, got.Roles); diff != "" {
			t.Errorf("Roles: -want, +got:\n%s", diff)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddOrgMember(ctx, "org-1", "u-1", []string{"admin"}); err != nil {
			t.Fatalf("AddOrgMember(...): unexpected error: %v", err)
		}

		if err := zc.RemoveOrgMember(ctx, "org-1", "u-1"); err != nil {
			t.Fatalf("RemoveOrgMember(...): unexpected error: %v", err)
		}

		if _, err := zc.GetOrgMember(ctx, "org-1", "u-1"); !IsNotFound(err) {
			t.Errorf("after a removal the membership must be gone, got %v", err)
		}
	})

	t.Run("RemoveAlreadyGoneIsSuccess", func(t *testing.T) {
		// Deleting a resource whose membership is already gone has to succeed,
		// or the finalizer waits for something that will never come back.
		zc, m := newMembershipClient(t)
		m.removeErr = notFound

		if err := zc.RemoveOrgMember(testContext(t), "org-1", "u-1"); err != nil {
			t.Errorf("RemoveOrgMember(...): a membership that is already gone must not be an error: %v", err)
		}
	})

	t.Run("RemoveFailureIsReported", func(t *testing.T) {
		zc, m := newMembershipClient(t)
		m.removeErr = status.Error(codes.PermissionDenied, "nope")

		if err := zc.RemoveOrgMember(testContext(t), "org-1", "u-1"); err == nil {
			t.Error("a permission failure was swallowed")
		}
	})

	t.Run("Missing", func(t *testing.T) {
		zc, _ := newMembershipClient(t)

		_, err := zc.GetOrgMember(testContext(t), "org-1", "absent")
		if !IsNotFound(err) {
			t.Errorf("a user who is not a member must report not found, got %v", err)
		}
	})

	t.Run("ListRoles", func(t *testing.T) {
		zc, m := newMembershipClient(t)
		m.orgRoles = []string{"admin", "reader"}

		got, err := zc.ListOrgMemberRoles(testContext(t), "org-1")
		if err != nil {
			t.Fatalf("ListOrgMemberRoles(...): unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{"admin", "reader"}, got); diff != "" {
			t.Errorf("ListOrgMemberRoles(...): -want, +got:\n%s", diff)
		}
	})
}

// TestInstanceMembership covers the same lifecycle at the instance level, which
// goes through the admin API rather than the management one.
func TestInstanceMembership(t *testing.T) {
	t.Run("AddThenRead", func(t *testing.T) {
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddIAMMember(ctx, "u-1", []string{"INSTANCE_OWNER"}); err != nil {
			t.Fatalf("AddIAMMember(...): unexpected error: %v", err)
		}

		got, err := zc.GetIAMMember(ctx, "u-1")
		if err != nil {
			t.Fatalf("GetIAMMember(...): unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{"INSTANCE_OWNER"}, got.Roles); diff != "" {
			t.Errorf("Roles: -want, +got:\n%s", diff)
		}

		if got.State != MembershipActive {
			t.Errorf("State: want %q, got %q", MembershipActive, got.State)
		}
	})

	t.Run("Update", func(t *testing.T) {
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddIAMMember(ctx, "u-1", []string{"INSTANCE_OWNER"}); err != nil {
			t.Fatalf("AddIAMMember(...): unexpected error: %v", err)
		}

		if err := zc.UpdateIAMMember(ctx, "u-1", []string{"INSTANCE_USER_MANAGER"}); err != nil {
			t.Fatalf("UpdateIAMMember(...): unexpected error: %v", err)
		}

		got, err := zc.GetIAMMember(ctx, "u-1")
		if err != nil {
			t.Fatalf("GetIAMMember(...): unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{"INSTANCE_USER_MANAGER"}, got.Roles); diff != "" {
			t.Errorf("Roles: -want, +got:\n%s", diff)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddIAMMember(ctx, "u-1", []string{"INSTANCE_OWNER"}); err != nil {
			t.Fatalf("AddIAMMember(...): unexpected error: %v", err)
		}

		if err := zc.RemoveIAMMember(ctx, "u-1"); err != nil {
			t.Fatalf("RemoveIAMMember(...): unexpected error: %v", err)
		}

		if _, err := zc.GetIAMMember(ctx, "u-1"); !IsNotFound(err) {
			t.Errorf("after a removal the membership must be gone, got %v", err)
		}
	})

	t.Run("RemoveAlreadyGoneIsSuccess", func(t *testing.T) {
		zc, _, a := newMembershipClientWithAdmin(t)
		a.updateErr = notFound

		if err := zc.RemoveIAMMember(testContext(t), "u-1"); err != nil {
			t.Errorf("RemoveIAMMember(...): a membership that is already gone must not be an error: %v", err)
		}
	})

	t.Run("Missing", func(t *testing.T) {
		zc, _ := newMembershipClient(t)

		if _, err := zc.GetIAMMember(testContext(t), "absent"); !IsNotFound(err) {
			t.Errorf("a user who is not an instance member must report not found, got %v", err)
		}
	})

	t.Run("ListRoles", func(t *testing.T) {
		zc, _, a := newMembershipClientWithAdmin(t)
		a.instanceRoles = []string{"INSTANCE_OWNER", "INSTANCE_USER_MANAGER"}

		got, err := zc.ListIAMMemberRoles(testContext(t))
		if err != nil {
			t.Fatalf("ListIAMMemberRoles(...): unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{"INSTANCE_OWNER", "INSTANCE_USER_MANAGER"}, got); diff != "" {
			t.Errorf("ListIAMMemberRoles(...): -want, +got:\n%s", diff)
		}
	})
}

// TestUserGrants covers the project grants, where the read is a filter over the
// organization's grants because ZITADEL has no lookup by project.
func TestUserGrants(t *testing.T) {
	t.Run("AddThenFind", func(t *testing.T) {
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddUserGrant(ctx, "org-1", "u-1", "p-1", "", []string{"reader"}); err != nil {
			t.Fatalf("AddUserGrant(...): unexpected error: %v", err)
		}

		got, err := zc.FindUserGrant(ctx, "org-1", "u-1", "p-1", "")
		if err != nil {
			t.Fatalf("FindUserGrant(...): unexpected error: %v", err)
		}

		if got.ProjectID != "p-1" {
			t.Errorf("ProjectID: want %q, got %q", "p-1", got.ProjectID)
		}

		if diff := cmp.Diff([]string{"reader"}, got.RoleKeys); diff != "" {
			t.Errorf("RoleKeys: -want, +got:\n%s", diff)
		}

		if got.State != MembershipActive {
			t.Errorf("State: want %q, got %q", MembershipActive, got.State)
		}
	})

	t.Run("Update", func(t *testing.T) {
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddUserGrant(ctx, "org-1", "u-1", "p-1", "", []string{"reader"}); err != nil {
			t.Fatalf("AddUserGrant(...): unexpected error: %v", err)
		}

		got, err := zc.FindUserGrant(ctx, "org-1", "u-1", "p-1", "")
		if err != nil {
			t.Fatalf("FindUserGrant(...): unexpected error: %v", err)
		}

		if err := zc.UpdateUserGrant(ctx, "org-1", "u-1", got.ID, []string{"writer"}); err != nil {
			t.Fatalf("UpdateUserGrant(...): unexpected error: %v", err)
		}

		got, err = zc.FindUserGrant(ctx, "org-1", "u-1", "p-1", "")
		if err != nil {
			t.Fatalf("FindUserGrant(...): unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{"writer"}, got.RoleKeys); diff != "" {
			t.Errorf("RoleKeys: -want, +got:\n%s", diff)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddUserGrant(ctx, "org-1", "u-1", "p-1", "", []string{"reader"}); err != nil {
			t.Fatalf("AddUserGrant(...): unexpected error: %v", err)
		}

		got, err := zc.FindUserGrant(ctx, "org-1", "u-1", "p-1", "")
		if err != nil {
			t.Fatalf("FindUserGrant(...): unexpected error: %v", err)
		}

		if err := zc.RemoveUserGrant(ctx, "org-1", "u-1", got.ID); err != nil {
			t.Fatalf("RemoveUserGrant(...): unexpected error: %v", err)
		}
	})

	t.Run("RemoveAlreadyGoneIsSuccess", func(t *testing.T) {
		zc, m := newMembershipClient(t)
		m.removeErr = notFound

		if err := zc.RemoveUserGrant(testContext(t), "org-1", "u-1", "g-1"); err != nil {
			t.Errorf("RemoveUserGrant(...): a grant that is already gone must not be an error: %v", err)
		}
	})

	t.Run("OtherProjectIsNotFound", func(t *testing.T) {
		// The read filters a list, so a grant on a different project has to be
		// skipped rather than returned.
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddUserGrant(ctx, "org-1", "u-1", "p-1", "", []string{"reader"}); err != nil {
			t.Fatalf("AddUserGrant(...): unexpected error: %v", err)
		}

		if _, err := zc.FindUserGrant(ctx, "org-1", "u-1", "p-other", ""); !IsNotFound(err) {
			t.Errorf("a grant on another project must not be found, got %v", err)
		}
	})

	t.Run("OtherProjectGrantIsNotFound", func(t *testing.T) {
		// The same project reached through a different project grant is a
		// different grant, and the filter has to tell them apart.
		zc, m := newMembershipClient(t)
		m.grants = []*userv1.UserGrant{
			{Id: "g-1", UserId: "u-1", ProjectId: "p-1", ProjectGrantId: "pg-a", State: userv1.UserGrantState_USER_GRANT_STATE_ACTIVE},
			{Id: "g-2", UserId: "u-1", ProjectId: "p-1", ProjectGrantId: "pg-b", State: userv1.UserGrantState_USER_GRANT_STATE_INACTIVE},
		}

		got, err := zc.FindUserGrant(testContext(t), "org-1", "u-1", "p-1", "pg-b")
		if err != nil {
			t.Fatalf("FindUserGrant(...): unexpected error: %v", err)
		}

		if got.ID != "g-2" {
			t.Errorf("ID: want %q, got %q", "g-2", got.ID)
		}

		// An inactive grant is reported as such rather than as active.
		if got.State != MembershipInactive {
			t.Errorf("State: want %q for an inactive grant, got %q", MembershipInactive, got.State)
		}
	})
}

// TestProjectGrantMembers covers the third membership kind, which is scoped to
// a project grant rather than an organization.
func TestProjectGrantMembers(t *testing.T) {
	t.Run("AddThenRead", func(t *testing.T) {
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddProjectGrantMember(ctx, "org-1", "p-1", "pg-1", "u-1", []string{"reader"}); err != nil {
			t.Fatalf("AddProjectGrantMember(...): unexpected error: %v", err)
		}

		got, err := zc.GetProjectGrantMember(ctx, "org-1", "p-1", "pg-1", "u-1")
		if err != nil {
			t.Fatalf("GetProjectGrantMember(...): unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{"reader"}, got.Roles); diff != "" {
			t.Errorf("Roles: -want, +got:\n%s", diff)
		}
	})

	t.Run("Update", func(t *testing.T) {
		zc, _ := newMembershipClient(t)
		ctx := testContext(t)

		if err := zc.AddProjectGrantMember(ctx, "org-1", "p-1", "pg-1", "u-1", []string{"reader"}); err != nil {
			t.Fatalf("AddProjectGrantMember(...): unexpected error: %v", err)
		}

		if err := zc.UpdateProjectGrantMember(ctx, "org-1", "p-1", "pg-1", "u-1", []string{"writer"}); err != nil {
			t.Fatalf("UpdateProjectGrantMember(...): unexpected error: %v", err)
		}

		got, err := zc.GetProjectGrantMember(ctx, "org-1", "p-1", "pg-1", "u-1")
		if err != nil {
			t.Fatalf("GetProjectGrantMember(...): unexpected error: %v", err)
		}

		if diff := cmp.Diff([]string{"writer"}, got.Roles); diff != "" {
			t.Errorf("Roles: -want, +got:\n%s", diff)
		}
	})

	t.Run("RemoveAlreadyGoneIsSuccess", func(t *testing.T) {
		zc, m := newMembershipClient(t)
		m.removeErr = notFound

		if err := zc.RemoveProjectGrantMember(testContext(t), "org-1", "p-1", "pg-1", "u-1"); err != nil {
			t.Errorf("RemoveProjectGrantMember(...): a membership that is already gone must not be an error: %v", err)
		}
	})

	t.Run("Missing", func(t *testing.T) {
		zc, _ := newMembershipClient(t)

		if _, err := zc.GetProjectGrantMember(testContext(t), "org-1", "p-1", "pg-1", "absent"); !IsNotFound(err) {
			t.Errorf("a user who is not a member must report not found, got %v", err)
		}
	})
}

// newMembershipClient starts a fake serving both the admin and the management
// API, and returns a client for it alongside the two fakes.
func newMembershipClient(t *testing.T) (*Client, *fakeManagement) {
	t.Helper()

	zc, m, _ := newMembershipClientWithAdmin(t)

	return zc, m
}

// newMembershipClientWithAdmin returns both fakes, for the instance level calls
// that go through the admin API rather than the management one.
func newMembershipClientWithAdmin(t *testing.T) (*Client, *fakeManagement, *fakeAdmin) {
	t.Helper()

	m := newFakeManagement()
	a := newFakeAdmin()

	fake := newFakeZitadel(t, func(s *grpc.Server) {
		managementv1.RegisterManagementServiceServer(s, m)
		adminv1.RegisterAdminServiceServer(s, a)
	})

	return fake.client(t), m, a
}
