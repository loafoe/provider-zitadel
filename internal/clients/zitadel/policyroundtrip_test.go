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
	"context"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	memberv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/member"
	policyv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/policy"
	userv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// A fake management API, so the policy layer can be driven end to end.
//
// The seven organization policies all share one shape: a read, a write that is
// either an add or an update depending on what the read said, and a reset. That
// shape is where a managed policy gets stuck if it is wrong, so it is worth
// exercising against a server that can be made to fail in each of the ways
// Zitadel actually fails, rather than only through the pure helpers.

// fakeManagement implements just the policy RPCs the client uses. Everything
// else answers Unimplemented, which is what the embedded struct provides.
type fakeManagement struct {
	managementv1.UnimplementedManagementServiceServer

	mu sync.Mutex

	// policies is what the fake reports a policy to be. isDefault is what tells
	// the client to add rather than to update.
	policies  map[string]*policyv1.LockoutPolicy
	isDefault map[string]bool

	// recorded is the last write the client made, so a test can check which
	// call it chose.
	recorded []string

	// The write calls each fail with their own error, nil meaning they succeed.
	addErr    error
	updateErr error
	removeErr error
	resetErr  error

	// getErr is what the read fails with, nil meaning it reports the policy.
	getErr error

	// members is the organization membership list, orgRoles what the organization
	// offers, and grants the project grants of the organization.
	members      []*memberv1.Member
	orgRoles     []string
	grants       []*userv1.UserGrant
	grantMembers []*memberv1.Member
}

func newFakeManagement() *fakeManagement {
	return &fakeManagement{
		policies:  map[string]*policyv1.LockoutPolicy{},
		isDefault: map[string]bool{},
	}
}

// --- Memberships and grants, which share the fake with the policies ---

func (f *fakeManagement) ListOrgMemberRoles(_ context.Context, _ *managementv1.ListOrgMemberRolesRequest) (*managementv1.ListOrgMemberRolesResponse, error) {
	f.record("list-org-roles")

	if f.getErr != nil {
		return nil, f.getErr
	}

	return &managementv1.ListOrgMemberRolesResponse{Result: f.orgRoles}, nil
}

func (f *fakeManagement) ListOrgMembers(_ context.Context, _ *managementv1.ListOrgMembersRequest) (*managementv1.ListOrgMembersResponse, error) {
	f.record("list-org-members")

	if f.getErr != nil {
		return nil, f.getErr
	}

	return &managementv1.ListOrgMembersResponse{Result: f.members}, nil
}

func (f *fakeManagement) AddOrgMember(_ context.Context, in *managementv1.AddOrgMemberRequest) (*managementv1.AddOrgMemberResponse, error) {
	f.record("add-org-member")

	if f.addErr != nil {
		return nil, f.addErr
	}

	f.members = upsertMember(f.members, in.GetUserId(), in.GetRoles())

	return &managementv1.AddOrgMemberResponse{}, nil
}

func (f *fakeManagement) UpdateOrgMember(_ context.Context, in *managementv1.UpdateOrgMemberRequest) (*managementv1.UpdateOrgMemberResponse, error) {
	f.record("update-org-member")

	if f.updateErr != nil {
		return nil, f.updateErr
	}

	f.members = upsertMember(f.members, in.GetUserId(), in.GetRoles())

	return &managementv1.UpdateOrgMemberResponse{}, nil
}

func (f *fakeManagement) RemoveOrgMember(_ context.Context, in *managementv1.RemoveOrgMemberRequest) (*managementv1.RemoveOrgMemberResponse, error) {
	f.record("remove-org-member")

	if f.removeErr != nil {
		return nil, f.removeErr
	}

	f.members = dropMember(f.members, in.GetUserId())

	return &managementv1.RemoveOrgMemberResponse{}, nil
}

func (f *fakeManagement) ListUserGrants(_ context.Context, _ *managementv1.ListUserGrantRequest) (*managementv1.ListUserGrantResponse, error) {
	f.record("list-user-grants")

	if f.getErr != nil {
		return nil, f.getErr
	}

	return &managementv1.ListUserGrantResponse{Result: f.grants}, nil
}

func (f *fakeManagement) AddUserGrant(_ context.Context, in *managementv1.AddUserGrantRequest) (*managementv1.AddUserGrantResponse, error) {
	f.record("add-user-grant")

	if f.addErr != nil {
		return nil, f.addErr
	}

	f.grants = append(f.grants, &userv1.UserGrant{
		Id:             "g-" + in.GetUserId(),
		UserId:         in.GetUserId(),
		ProjectId:      in.GetProjectId(),
		ProjectGrantId: in.GetProjectGrantId(),
		RoleKeys:       in.GetRoleKeys(),
		State:          userv1.UserGrantState_USER_GRANT_STATE_ACTIVE,
	})

	return &managementv1.AddUserGrantResponse{}, nil
}

func (f *fakeManagement) UpdateUserGrant(_ context.Context, in *managementv1.UpdateUserGrantRequest) (*managementv1.UpdateUserGrantResponse, error) {
	f.record("update-user-grant")

	if f.updateErr != nil {
		return nil, f.updateErr
	}

	for _, g := range f.grants {
		if g.GetId() == in.GetGrantId() {
			g.RoleKeys = in.GetRoleKeys()

			return &managementv1.UpdateUserGrantResponse{}, nil
		}
	}

	return nil, status.Error(codes.NotFound, "no such grant")
}

func (f *fakeManagement) RemoveUserGrant(_ context.Context, in *managementv1.RemoveUserGrantRequest) (*managementv1.RemoveUserGrantResponse, error) {
	f.record("remove-user-grant")

	if f.removeErr != nil {
		return nil, f.removeErr
	}

	return &managementv1.RemoveUserGrantResponse{}, nil
}

func (f *fakeManagement) ListProjectGrantMembers(_ context.Context, _ *managementv1.ListProjectGrantMembersRequest) (*managementv1.ListProjectGrantMembersResponse, error) {
	f.record("list-project-grant-members")

	if f.getErr != nil {
		return nil, f.getErr
	}

	return &managementv1.ListProjectGrantMembersResponse{Result: f.grantMembers}, nil
}

func (f *fakeManagement) AddProjectGrantMember(_ context.Context, in *managementv1.AddProjectGrantMemberRequest) (*managementv1.AddProjectGrantMemberResponse, error) {
	f.record("add-project-grant-member")

	if f.addErr != nil {
		return nil, f.addErr
	}

	f.grantMembers = upsertMember(f.grantMembers, in.GetUserId(), in.GetRoles())

	return &managementv1.AddProjectGrantMemberResponse{}, nil
}

func (f *fakeManagement) UpdateProjectGrantMember(_ context.Context, in *managementv1.UpdateProjectGrantMemberRequest) (*managementv1.UpdateProjectGrantMemberResponse, error) {
	f.record("update-project-grant-member")

	if f.updateErr != nil {
		return nil, f.updateErr
	}

	f.grantMembers = upsertMember(f.grantMembers, in.GetUserId(), in.GetRoles())

	return &managementv1.UpdateProjectGrantMemberResponse{}, nil
}

func (f *fakeManagement) RemoveProjectGrantMember(_ context.Context, _ *managementv1.RemoveProjectGrantMemberRequest) (*managementv1.RemoveProjectGrantMemberResponse, error) {
	f.record("remove-project-grant-member")

	if f.removeErr != nil {
		return nil, f.removeErr
	}

	return &managementv1.RemoveProjectGrantMemberResponse{}, nil
}

func (f *fakeManagement) record(call string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.recorded = append(f.recorded, call)
}

func (f *fakeManagement) lastCall() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.recorded) == 0 {
		return ""
	}

	return f.recorded[len(f.recorded)-1]
}

func (f *fakeManagement) get(orgID string) (*policyv1.LockoutPolicy, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.getErr != nil {
		return nil, false, f.getErr
	}

	return f.policies[orgID], f.isDefault[orgID], nil
}

func (f *fakeManagement) set(orgID string, p *policyv1.LockoutPolicy) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.policies[orgID] = p
	f.isDefault[orgID] = false
}

func (f *fakeManagement) GetLockoutPolicy(_ context.Context, _ *managementv1.GetLockoutPolicyRequest) (*managementv1.GetLockoutPolicyResponse, error) {
	f.record("get")

	p, _, err := f.get("lockout")
	if err != nil {
		return nil, err
	}

	if p == nil {
		return nil, status.Error(codes.NotFound, "no lockout policy")
	}

	return &managementv1.GetLockoutPolicyResponse{Policy: p}, nil
}

func (f *fakeManagement) AddCustomLockoutPolicy(_ context.Context, in *managementv1.AddCustomLockoutPolicyRequest) (*managementv1.AddCustomLockoutPolicyResponse, error) {
	f.record("add")

	if f.addErr != nil {
		return nil, f.addErr
	}

	f.set("lockout", &policyv1.LockoutPolicy{
		IsDefault:           false,
		MaxPasswordAttempts: uint64(in.GetMaxPasswordAttempts()),
		MaxOtpAttempts:      uint64(in.GetMaxOtpAttempts()),
	})

	return &managementv1.AddCustomLockoutPolicyResponse{}, nil
}

func (f *fakeManagement) UpdateCustomLockoutPolicy(_ context.Context, in *managementv1.UpdateCustomLockoutPolicyRequest) (*managementv1.UpdateCustomLockoutPolicyResponse, error) {
	f.record("update")

	if f.updateErr != nil {
		return nil, f.updateErr
	}

	f.set("lockout", &policyv1.LockoutPolicy{
		IsDefault:           false,
		MaxPasswordAttempts: uint64(in.GetMaxPasswordAttempts()),
		MaxOtpAttempts:      uint64(in.GetMaxOtpAttempts()),
	})

	return &managementv1.UpdateCustomLockoutPolicyResponse{}, nil
}

func (f *fakeManagement) ResetLockoutPolicyToDefault(_ context.Context, _ *managementv1.ResetLockoutPolicyToDefaultRequest) (*managementv1.ResetLockoutPolicyToDefaultResponse, error) {
	f.record("reset")

	if f.resetErr != nil {
		return nil, f.resetErr
	}

	f.mu.Lock()
	delete(f.policies, "lockout")
	f.mu.Unlock()

	return &managementv1.ResetLockoutPolicyToDefaultResponse{}, nil
}

// newPolicyClient starts a fake serving the management API and returns a client
// pointed at it.
func newPolicyClient(t *testing.T) (*Client, *fakeManagement) {
	t.Helper()

	m := newFakeManagement()

	fake := newFakeZitadel(t, func(s *grpc.Server) {
		managementv1.RegisterManagementServiceServer(s, m)
	})

	return fake.client(t), m
}

// TestGetLockoutPolicy covers the read, including the two ways it can report
// that there is nothing to read.
func TestGetLockoutPolicy(t *testing.T) {
	zc, m := newPolicyClient(t)
	ctx := testContext(t)

	cases := map[string]struct {
		reason   string
		setup    func(*fakeManagement)
		want     *LockoutPolicy
		wantErr  bool
		wantNil  bool
		wantCode codes.Code
	}{
		"Custom": {
			reason: "A custom policy is reported with its values and flagged as not a default",
			setup: func(m *fakeManagement) {
				m.policies["lockout"] = &policyv1.LockoutPolicy{
					IsDefault:           false,
					MaxPasswordAttempts: 7,
					MaxOtpAttempts:      3,
				}
			},
			want: &LockoutPolicy{OrgID: "org-1", IsDefault: false, MaxPasswordAttempts: 7, MaxOTPAttempts: 3},
		},
		"Default": {
			reason: "A policy still on the instance default is flagged, which is what makes the write add",
			setup: func(m *fakeManagement) {
				m.policies["lockout"] = &policyv1.LockoutPolicy{IsDefault: true, MaxPasswordAttempts: 5, MaxOtpAttempts: 5}
				m.isDefault["lockout"] = true
			},
			want: &LockoutPolicy{OrgID: "org-1", IsDefault: true, MaxPasswordAttempts: 5, MaxOTPAttempts: 5},
		},
		"NotFound": {
			reason:   "A missing policy becomes the sentinel the controller turns into a Create",
			setup:    func(*fakeManagement) {},
			wantNil:  true,
			wantCode: codes.NotFound,
		},
		"EmptyResponse": {
			reason: "A response with no policy at all is also a missing policy",
			setup: func(m *fakeManagement) {
				// Present, so the read succeeds, but carrying nothing.
				m.policies["lockout"] = nil
				m.getErr = status.Error(codes.NotFound, "no lockout policy")
			},
			wantNil:  true,
			wantCode: codes.NotFound,
		},
		"Failure": {
			reason:   "An unrelated failure is reported as itself",
			setup:    func(m *fakeManagement) { m.getErr = status.Error(codes.PermissionDenied, "nope") },
			wantNil:  true,
			wantCode: codes.PermissionDenied,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			m.mu.Lock()
			m.policies = map[string]*policyv1.LockoutPolicy{}
			m.isDefault = map[string]bool{}
			m.getErr = nil
			m.mu.Unlock()

			tc.setup(m)

			got, err := zc.GetLockoutPolicy(ctx, "org-1")
			if tc.wantNil {
				if !IsNotFound(err) && !statusHasCode(err, tc.wantCode) {
					t.Errorf("\n%s\nGetLockoutPolicy(...): want a not-found or %v error, got %v", tc.reason, tc.wantCode, err)
				}

				return
			}

			if err != nil {
				t.Fatalf("\n%s\nGetLockoutPolicy(...): unexpected error: %v", tc.reason, err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\n%s\nGetLockoutPolicy(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

// TestSetLockoutPolicy covers the write against a server, which is where the
// add-or-update choice and its fallback actually matter.
func TestSetLockoutPolicy(t *testing.T) {
	cases := map[string]struct {
		reason string
		// setup arranges what the fake reports and how it fails.
		setup func(*fakeManagement)
		// wantCall is the write the client is expected to have ended on.
		wantCall string
		wantErr  bool
	}{
		"OrganizationOnDefaultGetsAnAdd": {
			reason: "An organization still on the instance default is given an add",
			setup: func(m *fakeManagement) {
				m.policies["lockout"] = &policyv1.LockoutPolicy{IsDefault: true, MaxPasswordAttempts: 5, MaxOtpAttempts: 5}
				m.isDefault["lockout"] = true
			},
			wantCall: "add",
		},
		"OrganizationWithACustomPolicyGetsAnUpdate": {
			reason: "An organization already holding a custom policy is given an update",
			setup: func(m *fakeManagement) {
				m.policies["lockout"] = &policyv1.LockoutPolicy{IsDefault: false, MaxPasswordAttempts: 5, MaxOtpAttempts: 5}
			},
			wantCall: "update",
		},
		"StaleDefaultFallsBackToUpdate": {
			reason: "An organization reported as default but already holding a policy falls back to an update",
			// This is the case that makes a policy succeed rather than get
			// stuck: the read is wrong, the add is refused, and the write has to
			// correct itself.
			setup: func(m *fakeManagement) {
				m.policies["lockout"] = &policyv1.LockoutPolicy{IsDefault: false, MaxPasswordAttempts: 5, MaxOtpAttempts: 5}
				m.isDefault["lockout"] = true
				m.addErr = status.Error(codes.AlreadyExists, "already exists")
			},
			wantCall: "update",
		},
		"MissingPolicyFallsBackToAdd": {
			reason: "An organization with no policy at all is added when the update is refused",
			setup: func(m *fakeManagement) {
				m.policies["lockout"] = &policyv1.LockoutPolicy{IsDefault: false, MaxPasswordAttempts: 5, MaxOtpAttempts: 5}
				m.updateErr = status.Error(codes.NotFound, "no policy")
			},
			wantCall: "add",
		},
		"NoChangeIsSuccess": {
			reason: "A write Zitadel refuses because it changes nothing is success",
			setup: func(m *fakeManagement) {
				m.policies["lockout"] = &policyv1.LockoutPolicy{IsDefault: false, MaxPasswordAttempts: 5, MaxOtpAttempts: 5}
				m.updateErr = status.Error(codes.FailedPrecondition, "No changes")
			},
			wantCall: "update",
		},
		"UnrelatedFailureIsReported": {
			reason: "A failure nothing can be done about is reported rather than retried the other way",
			setup: func(m *fakeManagement) {
				m.policies["lockout"] = &policyv1.LockoutPolicy{IsDefault: true, MaxPasswordAttempts: 5, MaxOtpAttempts: 5}
				m.isDefault["lockout"] = true
				m.addErr = status.Error(codes.PermissionDenied, "nope")
			},
			wantCall: "add",
			wantErr:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			zc, m := newPolicyClient(t)

			m.mu.Lock()
			m.policies = map[string]*policyv1.LockoutPolicy{}
			m.isDefault = map[string]bool{}
			m.recorded = nil
			m.addErr = nil
			m.updateErr = nil
			m.mu.Unlock()

			tc.setup(m)

			err := zc.SetLockoutPolicy(context.Background(), "org-1", LockoutPolicyInput{
				MaxPasswordAttempts: 9,
				MaxOTPAttempts:      4,
			})
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\nSetLockoutPolicy(...): wantErr %v, got %v", tc.reason, tc.wantErr, err)
			}

			if got := m.lastCall(); got != tc.wantCall {
				t.Errorf("\n%s\nSetLockoutPolicy(...): want the write to end on %q, got %q", tc.reason, tc.wantCall, got)
			}
		})
	}
}

// TestSetLockoutPolicyWritesTheValues checks that the values the caller asked
// for are the ones Zitadel ends up holding, which is the whole point of the
// write and is not visible from the call log alone.
func TestSetLockoutPolicyWritesTheValues(t *testing.T) {
	zc, m := newPolicyClient(t)

	m.mu.Lock()
	m.policies["lockout"] = &policyv1.LockoutPolicy{IsDefault: true}
	m.isDefault["lockout"] = true
	m.mu.Unlock()

	want := LockoutPolicyInput{MaxPasswordAttempts: 9, MaxOTPAttempts: 4}
	if err := zc.SetLockoutPolicy(context.Background(), "org-1", want); err != nil {
		t.Fatalf("SetLockoutPolicy(...): unexpected error: %v", err)
	}

	got, err := zc.GetLockoutPolicy(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("GetLockoutPolicy(...): unexpected error: %v", err)
	}

	wantPolicy := &LockoutPolicy{
		OrgID:               "org-1",
		IsDefault:           false,
		MaxPasswordAttempts: want.MaxPasswordAttempts,
		MaxOTPAttempts:      want.MaxOTPAttempts,
	}

	if diff := cmp.Diff(wantPolicy, got); diff != "" {
		t.Errorf("SetLockoutPolicy(...) then GetLockoutPolicy(...): -want, +got:\n%s", diff)
	}
}

// TestResetLockoutPolicy covers the reset, and the two failures that mean the
// organization is already on the default.
func TestResetLockoutPolicy(t *testing.T) {
	cases := map[string]struct {
		reason   string
		resetErr error
		wantErr  bool
	}{
		"Success": {
			reason: "A reset an organization can have is performed",
		},
		"AlreadyReset": {
			reason:   "An organization already on the default is left alone",
			resetErr: status.Error(codes.NotFound, "no policy"),
		},
		"NothingToChange": {
			reason:   "A reset that would change nothing is left alone",
			resetErr: status.Error(codes.FailedPrecondition, "No changes"),
		},
		"Failure": {
			reason:   "A failure nothing can be done about is reported",
			resetErr: status.Error(codes.PermissionDenied, "nope"),
			wantErr:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			zc, m := newPolicyClient(t)

			m.mu.Lock()
			m.policies["lockout"] = &policyv1.LockoutPolicy{IsDefault: false, MaxPasswordAttempts: 1, MaxOtpAttempts: 1}
			m.resetErr = tc.resetErr
			m.mu.Unlock()

			err := zc.ResetLockoutPolicy(context.Background(), "org-1")
			if (err != nil) != tc.wantErr {
				t.Errorf("\n%s\nResetLockoutPolicy(...): wantErr %v, got %v", tc.reason, tc.wantErr, err)
			}
		})
	}
}

// TestPolicyManagementClientIsCachedPerOrganization checks that the per
// organization management clients are built once each.
//
// They carry the organization as a connection interceptor, so a client built for
// one organization cannot serve another: a cache that handed the wrong one back
// would quietly write to the wrong organization, which is the kind of bug that
// only shows up in production.
func TestPolicyManagementClientIsCachedPerOrganization(t *testing.T) {
	zc, _ := newPolicyClient(t)

	first, err := zc.managementClient(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("managementClient(...): unexpected error: %v", err)
	}

	again, err := zc.managementClient(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("managementClient(...): unexpected error: %v", err)
	}

	if first != again {
		t.Error("two calls for one organization built two clients")
	}

	other, err := zc.managementClient(context.Background(), "org-2")
	if err != nil {
		t.Fatalf("managementClient(...): unexpected error: %v", err)
	}

	if other == first {
		t.Error("two organizations were given the same client, so one would write to the other")
	}

	// Three configurations, and the shared connection on top of them.
	if got := len(zc.management); got != 2 {
		t.Errorf("want two cached management clients, got %d", got)
	}
}

// statusHasCode reports whether err carries the given gRPC code.
func statusHasCode(err error, code codes.Code) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	return st.Code() == code
}
