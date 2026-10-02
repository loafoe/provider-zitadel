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

//nolint:staticcheck // every call here is a v1 management/admin API call; see the note above
package zitadel

import (
	"context"
	"fmt"
	"slices"
	"strings"

	adminv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	memberv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/member"
	objectv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/object"
	userv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user"
)

// Why these resources use the v1 management and admin APIs
//
// Memberships and grants are the one area where Zitadel has no v2 API, even
// though everything around them has moved on. The Terraform provider uses the
// same calls, so this mirrors it rather than inventing a private path.
//
// The SDK marks every call in this file deprecated in favour of the
// authorization v2 and internal_permission v2 services. Neither is usable
// here: those client packages ship no service methods in this SDK version, and
// internal_permission is Zitadel's own console-facing API, which an ordinary
// service account is not permitted to call. Hence the file-scoped directive
// below.

// MembershipState is the lifecycle state of a membership or grant.
type MembershipState string

const (
	// MembershipActive is an active membership.
	MembershipActive MembershipState = "Active"
	// MembershipInactive is a deactivated membership.
	MembershipInactive MembershipState = "Inactive"
)

// Membership is a user's membership of an organization, with the roles it
// holds there.
type Membership struct {
	UserID      string
	Roles       []string
	State       MembershipState
	DisplayName string
	Email       string
	UserType    string
	UserName    string
}

// UserGrant is a set of project roles granted to a user, either in the user's
// own organization or on a project granted to that organization.
type UserGrant struct {
	ID             string
	UserID         string
	OrgID          string
	ProjectID      string
	ProjectGrantID string
	RoleKeys       []string
	State          MembershipState
}

// ValidateRoles reports the subset of roles that this Zitadel instance does not
// recognise for the given prefix.
//
// Role keys are not a fixed enumeration: an instance owner maps them from the
// instance's role mapping, which is why the provider asks the instance rather
// than hardcoding a list. Getting this wrong produces a confusing "role not
// found" deep inside Zitadel, so the provider checks first and names the roles
// that are valid.
func ValidateRoles(requested, available []string) []string {
	var unknown []string

	for _, r := range requested {
		if !slices.Contains(available, r) {
			unknown = append(unknown, r)
		}
	}

	return unknown
}

// FormatUnknownRolesLabel names the kind of role being validated, so the error
// message reads correctly wherever it is used.
const FormatUnknownRolesLabel = "roles"

// FormatUnknownRoles renders an actionable error for roles an instance does not
// recognise.
func FormatUnknownRoles(kind string, unknown, available []string) error {
	return fmt.Errorf(
		"zitadel does not recognise %s %v. Roles are configured per instance. Available in this instance: %v",
		kind, unknown, available,
	)
}

// ListOrgMemberRoles returns the organization level roles this instance offers.
func (c *Client) ListOrgMemberRoles(ctx context.Context, orgID string) ([]string, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.ListOrgMemberRoles(ctx, &managementv1.ListOrgMemberRolesRequest{})
	if err != nil {
		return nil, fmt.Errorf("cannot list the organization roles of organization %s: %w", orgID, err)
	}

	return resp.GetResult(), nil
}

// ListIAMMemberRoles returns the instance level roles this instance offers.
func (c *Client) ListIAMMemberRoles(ctx context.Context) ([]string, error) {
	resp, err := c.admin.ListIAMMemberRoles(ctx, &adminv1.ListIAMMemberRolesRequest{})
	if err != nil {
		return nil, fmt.Errorf("cannot list the instance roles: %w", err)
	}

	return resp.GetRoles(), nil
}

// GetOrgMember returns a user's membership of an organization.
func (c *Client) GetOrgMember(ctx context.Context, orgID, userID string) (*Membership, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.ListOrgMembers(ctx, &managementv1.ListOrgMembersRequest{
		Query: &objectv1.ListQuery{Limit: 1000},
		Queries: []*memberv1.SearchQuery{
			{
				Query: &memberv1.SearchQuery_UserIdQuery{
					UserIdQuery: &memberv1.UserIDQuery{UserId: userID},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the members of organization %s: %w", orgID, err)
	}

	for _, m := range resp.GetResult() {
		if m.GetUserId() == userID {
			return fromProtoMember(m), nil
		}
	}

	return nil, ErrNotFound
}

// AddOrgMember adds a user to an organization with the given roles.
func (c *Client) AddOrgMember(ctx context.Context, orgID, userID string, roles []string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.AddOrgMember(ctx, &managementv1.AddOrgMemberRequest{
		UserId: userID,
		Roles:  roles,
	}); err != nil {
		return fmt.Errorf("cannot add user %s to organization %s: %w", userID, orgID, err)
	}

	return nil
}

// UpdateOrgMember replaces the roles of a user's organization membership.
func (c *Client) UpdateOrgMember(ctx context.Context, orgID, userID string, roles []string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.UpdateOrgMember(ctx, &managementv1.UpdateOrgMemberRequest{
		UserId: userID,
		Roles:  roles,
	}); err != nil {
		return fmt.Errorf("cannot update the membership of user %s in organization %s: %w", userID, orgID, err)
	}

	return nil
}

// RemoveOrgMember removes a user from an organization.
func (c *Client) RemoveOrgMember(ctx context.Context, orgID, userID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.RemoveOrgMember(ctx, &managementv1.RemoveOrgMemberRequest{UserId: userID}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot remove user %s from organization %s: %w", userID, orgID, err)
	}

	return nil
}

// GetIAMMember returns a user's membership of the instance.
func (c *Client) GetIAMMember(ctx context.Context, userID string) (*Membership, error) {
	resp, err := c.admin.ListIAMMembers(ctx, &adminv1.ListIAMMembersRequest{
		Query: &objectv1.ListQuery{Limit: 1000},
		Queries: []*memberv1.SearchQuery{
			{
				Query: &memberv1.SearchQuery_UserIdQuery{
					UserIdQuery: &memberv1.UserIDQuery{UserId: userID},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the members of the instance: %w", err)
	}

	for _, m := range resp.GetResult() {
		if m.GetUserId() == userID {
			return fromProtoMember(m), nil
		}
	}

	return nil, ErrNotFound
}

// AddIAMMember adds a user to the instance with the given roles.
func (c *Client) AddIAMMember(ctx context.Context, userID string, roles []string) error {
	if _, err := c.admin.AddIAMMember(ctx, &adminv1.AddIAMMemberRequest{
		UserId: userID,
		Roles:  roles,
	}); err != nil {
		return fmt.Errorf("cannot add user %s to the instance: %w", userID, err)
	}

	return nil
}

// UpdateIAMMember replaces the roles of a user's instance membership.
func (c *Client) UpdateIAMMember(ctx context.Context, userID string, roles []string) error {
	if _, err := c.admin.UpdateIAMMember(ctx, &adminv1.UpdateIAMMemberRequest{
		UserId: userID,
		Roles:  roles,
	}); err != nil {
		return fmt.Errorf("cannot update the instance membership of user %s: %w", userID, err)
	}

	return nil
}

// RemoveIAMMember removes a user from the instance.
func (c *Client) RemoveIAMMember(ctx context.Context, userID string) error {
	if _, err := c.admin.RemoveIAMMember(ctx, &adminv1.RemoveIAMMemberRequest{UserId: userID}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot remove user %s from the instance: %w", userID, err)
	}

	return nil
}

// FindUserGrant returns the grant of a user on a project, optionally restricted
// to a project grant. Zitadel has no lookup by project, so the grants of the
// organization are listed and filtered here.
func (c *Client) FindUserGrant(ctx context.Context, orgID, userID, projectID, projectGrantID string) (*UserGrant, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.ListUserGrants(ctx, &managementv1.ListUserGrantRequest{
		Query: &objectv1.ListQuery{Limit: 1000},
		Queries: []*userv1.UserGrantQuery{
			{
				Query: &userv1.UserGrantQuery_UserIdQuery{
					UserIdQuery: &userv1.UserGrantUserIDQuery{UserId: userID},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the user grants of organization %s: %w", orgID, err)
	}

	for _, g := range resp.GetResult() {
		if g.GetProjectId() != projectID {
			continue
		}
		if g.GetProjectGrantId() != projectGrantID {
			continue
		}
		return fromProtoUserGrant(g), nil
	}

	return nil, ErrNotFound
}

// AddUserGrant grants project roles to a user in an organization.
func (c *Client) AddUserGrant(ctx context.Context, orgID, userID, projectID, projectGrantID string, roles []string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.AddUserGrant(ctx, &managementv1.AddUserGrantRequest{
		UserId:         userID,
		ProjectId:      projectID,
		ProjectGrantId: projectGrantID,
		RoleKeys:       roles,
	}); err != nil {
		return fmt.Errorf("cannot grant roles %v to user %s on project %s: %w", roles, userID, projectID, err)
	}

	return nil
}

// UpdateUserGrant replaces the roles of a user grant.
func (c *Client) UpdateUserGrant(ctx context.Context, orgID, userID, grantID string, roles []string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.UpdateUserGrant(ctx, &managementv1.UpdateUserGrantRequest{
		UserId:   userID,
		GrantId:  grantID,
		RoleKeys: roles,
	}); err != nil {
		return fmt.Errorf("cannot update the roles of user grant %s: %w", grantID, err)
	}

	return nil
}

// RemoveUserGrant revokes a user grant.
func (c *Client) RemoveUserGrant(ctx context.Context, orgID, userID, grantID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.RemoveUserGrant(ctx, &managementv1.RemoveUserGrantRequest{
		UserId:  userID,
		GrantId: grantID,
	}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot revoke user grant %s: %w", grantID, err)
	}

	return nil
}

// GetProjectGrantMember returns a user's membership of a project grant.
func (c *Client) GetProjectGrantMember(ctx context.Context, orgID, projectID, grantID, userID string) (*Membership, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.ListProjectGrantMembers(ctx, &managementv1.ListProjectGrantMembersRequest{
		ProjectId: projectID,
		GrantId:   grantID,
		Query:     &objectv1.ListQuery{Limit: 1000},
		Queries: []*memberv1.SearchQuery{
			{
				Query: &memberv1.SearchQuery_UserIdQuery{
					UserIdQuery: &memberv1.UserIDQuery{UserId: userID},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the members of project grant %s: %w", grantID, err)
	}

	for _, m := range resp.GetResult() {
		if m.GetUserId() == userID {
			return fromProtoMember(m), nil
		}
	}

	return nil, ErrNotFound
}

// AddProjectGrantMember adds a user to a project grant with the given roles.
func (c *Client) AddProjectGrantMember(ctx context.Context, orgID, projectID, grantID, userID string, roles []string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.AddProjectGrantMember(ctx, &managementv1.AddProjectGrantMemberRequest{
		ProjectId: projectID,
		GrantId:   grantID,
		UserId:    userID,
		Roles:     roles,
	}); err != nil {
		return fmt.Errorf("cannot add user %s to project grant %s with roles %v: %w", userID, grantID, roles, err)
	}

	return nil
}

// UpdateProjectGrantMember replaces the roles of a project grant member.
func (c *Client) UpdateProjectGrantMember(ctx context.Context, orgID, projectID, grantID, userID string, roles []string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.UpdateProjectGrantMember(ctx, &managementv1.UpdateProjectGrantMemberRequest{
		ProjectId: projectID,
		GrantId:   grantID,
		UserId:    userID,
		Roles:     roles,
	}); err != nil {
		return fmt.Errorf("cannot update the roles of user %s on project grant %s: %w", userID, grantID, err)
	}

	return nil
}

// RemoveProjectGrantMember removes a user from a project grant.
func (c *Client) RemoveProjectGrantMember(ctx context.Context, orgID, projectID, grantID, userID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.RemoveProjectGrantMember(ctx, &managementv1.RemoveProjectGrantMemberRequest{
		ProjectId: projectID,
		GrantId:   grantID,
		UserId:    userID,
	}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot remove user %s from project grant %s: %w", userID, grantID, err)
	}

	return nil
}

func fromProtoMember(m *memberv1.Member) *Membership {
	out := &Membership{
		UserID:      m.GetUserId(),
		Roles:       m.GetRoles(),
		State:       MembershipActive,
		DisplayName: m.GetDisplayName(),
		Email:       m.GetEmail(),
		UserName:    m.GetPreferredLoginName(),
	}

	if m.GetDetails() != nil {
		out.UserType = strings.ToLower(m.GetUserType().String())
	}

	return out
}

func fromProtoUserGrant(g *userv1.UserGrant) *UserGrant {
	out := &UserGrant{
		ID:             g.GetId(),
		UserID:         g.GetUserId(),
		OrgID:          g.GetOrgId(),
		ProjectID:      g.GetProjectId(),
		ProjectGrantID: g.GetProjectGrantId(),
		RoleKeys:       g.GetRoleKeys(),
		State:          MembershipActive,
	}

	if g.GetState() == userv1.UserGrantState_USER_GRANT_STATE_INACTIVE {
		out.State = MembershipInactive
	}

	return out
}
