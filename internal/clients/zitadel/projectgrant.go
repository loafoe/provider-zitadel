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

//nolint:staticcheck // ListProjectGrants is the only place Zitadel reports a grant's ID; see below
package zitadel

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"

	filterv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/filter/v2"
	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	objectv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/object"
	projectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/project/v2"
)

// ProjectGrant shares a project with another organization, along with the
// roles the members of that organization may hold on it. This is how a project
// owned by one organization is offered to another - the SaaS case of a
// platform team handing a billing project to a customer organization.
type ProjectGrant struct {
	ProjectID               string
	GrantedOrganizationID   string
	GrantedOrganizationName string
	RoleKeys                []string
	State                   string

	// GrantID is Zitadel's identifier of the grant. The v2 API does not return
	// it on create, so it is resolved from the v1 management API, which is what
	// ProjectGrantMember needs in order to address the grant.
	GrantID string
}

// FindProjectGrant returns the grant of projectID to grantedOrgID, or
// ErrNotFound.
//
// ownerOrgID is the organization that owns the project. It matters: the v1
// management API lists the grants of the organization in context, so looking a
// grant up in the *granted* organization finds nothing.
func (c *Client) FindProjectGrant(ctx context.Context, ownerOrgID, projectID, grantedOrgID string) (*ProjectGrant, error) {
	resp, err := c.project.ListProjectGrants(ctx, &projectv2.ListProjectGrantsRequest{
		Pagination: &filterv2.PaginationRequest{Limit: 1000},
		Filters: []*projectv2.ProjectGrantSearchFilter{
			{
				Filter: &projectv2.ProjectGrantSearchFilter_GrantedOrganizationIdFilter{
					GrantedOrganizationIdFilter: &filterv2.IDFilter{Id: grantedOrgID},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the grants of project %s: %w", projectID, err)
	}

	for _, g := range resp.GetProjectGrants() {
		if g.GetProjectId() != projectID {
			continue
		}
		if g.GetGrantedOrganizationId() != grantedOrgID {
			continue
		}

		out := &ProjectGrant{
			ProjectID:               g.GetProjectId(),
			GrantedOrganizationID:   g.GetGrantedOrganizationId(),
			GrantedOrganizationName: g.GetGrantedOrganizationName(),
			RoleKeys:                g.GetGrantedRoleKeys(),
			State:                   ProjectGrantStateFromProto(g.GetState()),
		}

		// The v2 API has no grant ID. The management API does, and
		// ProjectGrantMember cannot be addressed without it.
		if id, err := c.projectGrantID(ctx, ownerOrgID, projectID, grantedOrgID); err == nil {
			out.GrantID = id
		}

		return out, nil
	}

	return nil, ErrNotFound
}

// CreateProjectGrant shares a project with an organization.
func (c *Client) CreateProjectGrant(ctx context.Context, projectID, grantedOrgID string, roleKeys []string) error {
	if _, err := c.project.CreateProjectGrant(ctx, &projectv2.CreateProjectGrantRequest{
		ProjectId:             projectID,
		GrantedOrganizationId: grantedOrgID,
		RoleKeys:              roleKeys,
	}); err != nil {
		return fmt.Errorf("cannot grant project %s to organization %s: %w", projectID, grantedOrgID, err)
	}

	return nil
}

// UpdateProjectGrant replaces the roles of an existing project grant.
func (c *Client) UpdateProjectGrant(ctx context.Context, projectID, grantedOrgID string, roleKeys []string) error {
	if _, err := c.project.UpdateProjectGrant(ctx, &projectv2.UpdateProjectGrantRequest{
		ProjectId:             projectID,
		GrantedOrganizationId: grantedOrgID,
		RoleKeys:              roleKeys,
	}); err != nil {
		if isRoleNotFound(err) {
			// Zitadel only lets a grant's roles be narrowed: it rejects any role
			// that is not already on the grant, and reports the whole update as an
			// unknown project role. Saying so is the difference between an
			// operator knowing to recreate the grant and one hunting for a typo in
			// a role key that is perfectly valid.
			return fmt.Errorf("cannot update the grant of project %s to organization %s: "+
				"Zitadel only accepts roles that the grant already has, so a role cannot be added to an existing project grant; "+
				"remove the role from the grant first, or recreate the grant to change its roles: %w", projectID, grantedOrgID, err)
		}

		return fmt.Errorf("cannot update the grant of project %s to organization %s: %w", projectID, grantedOrgID, err)
	}

	return nil
}

// isRoleNotFound reports whether Zitadel rejected an update because it named a
// role that does not exist on the grant.
//
// The same code is also returned for a role key that is not a project role at
// all, which is the case an operator is most likely to hit by mistake.
func isRoleNotFound(err error) bool {
	if err == nil {
		return false
	}

	st, ok := grpcStatus(err)
	if !ok {
		return false
	}

	return st.Code() == codes.FailedPrecondition &&
		strings.Contains(st.Message(), "Errors.Project.Role.NotFound")
}

// SetProjectGrantState activates or deactivates a project grant.
func (c *Client) SetProjectGrantState(ctx context.Context, projectID, grantedOrgID, current, desired string) error {
	if desired == "" || desired == current {
		return nil
	}

	var err error
	switch desired {
	case StateActive:
		_, err = c.project.ActivateProjectGrant(ctx, &projectv2.ActivateProjectGrantRequest{
			ProjectId:             projectID,
			GrantedOrganizationId: grantedOrgID,
		})
	case StateInactive:
		_, err = c.project.DeactivateProjectGrant(ctx, &projectv2.DeactivateProjectGrantRequest{
			ProjectId:             projectID,
			GrantedOrganizationId: grantedOrgID,
		})
	}

	if err != nil {
		return fmt.Errorf("cannot set the state of the grant of project %s to %s: %w", projectID, desired, err)
	}

	return nil
}

// DeleteProjectGrant stops sharing a project with an organization.
func (c *Client) DeleteProjectGrant(ctx context.Context, projectID, grantedOrgID string) error {
	if _, err := c.project.DeleteProjectGrant(ctx, &projectv2.DeleteProjectGrantRequest{
		ProjectId:             projectID,
		GrantedOrganizationId: grantedOrgID,
	}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot remove the grant of project %s to organization %s: %w", projectID, grantedOrgID, err)
	}

	return nil
}

// ProjectGrantExternalName is the external name of a project grant. Zitadel
// has no ID of its own for a grant, so the pair it is addressed by is used.
func ProjectGrantExternalName(projectID, grantedOrgID string) string {
	return projectID + "/" + grantedOrgID
}

// ListProjectGrantMemberRoles returns the role keys a member of a project grant
// may hold.
//
// These are *not* the project's own role keys: Zitadel exposes the roles a
// grant makes available under a PROJECT_GRANT_ prefix, so that a member of a
// granted project cannot be handed a role in the project itself.
func (c *Client) ListProjectGrantMemberRoles(ctx context.Context, ownerOrgID, projectID, grantID string) ([]string, error) {
	mc, err := c.managementClient(ctx, ownerOrgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.ListGrantedProjectRoles(ctx, &managementv1.ListGrantedProjectRolesRequest{
		ProjectId: projectID,
		GrantId:   grantID,
		Query:     &objectv1.ListQuery{Limit: 1000},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the roles of project grant %s: %w", grantID, err)
	}

	out := make([]string, 0, len(resp.GetResult()))
	for _, r := range resp.GetResult() {
		out = append(out, r.GetKey())
	}

	return out, nil
}

// FindProjectGrantByID returns a project grant by Zitadel's own grant ID. The
// grant belongs to ownerOrgID's context for the same reason FindProjectGrant
// does.
func (c *Client) FindProjectGrantByID(ctx context.Context, ownerOrgID, projectID, grantID string) (*ProjectGrant, error) {
	mc, err := c.managementClient(ctx, ownerOrgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.ListProjectGrants(ctx, &managementv1.ListProjectGrantsRequest{
		ProjectId: projectID,
		Query:     &objectv1.ListQuery{Limit: 1000},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the grants of project %s: %w", projectID, err)
	}

	for _, g := range resp.GetResult() {
		if g.GetGrantId() != grantID {
			continue
		}

		return &ProjectGrant{
			ProjectID:               g.GetProjectId(),
			GrantedOrganizationID:   g.GetGrantedOrgId(),
			GrantedOrganizationName: g.GetGrantedOrgName(),
			RoleKeys:                g.GetGrantedRoleKeys(),
			GrantID:                 g.GetGrantId(),
		}, nil
	}

	return nil, ErrNotFound
}

// SplitProjectGrantExternalName splits an external name back into its parts.
func SplitProjectGrantExternalName(name string) (projectID, grantedOrgID string, ok bool) {
	projectID, grantedOrgID, ok = strings.Cut(name, "/")
	if !ok || projectID == "" || grantedOrgID == "" {
		return "", "", false
	}

	return projectID, grantedOrgID, true
}

// projectGrantID resolves the v1 management API's identifier of a project
// grant, which the v2 API does not expose - and which ProjectGrantMember needs
// in order to address the grant at all.
func (c *Client) projectGrantID(ctx context.Context, ownerOrgID, projectID, grantedOrgID string) (string, error) {
	mc, err := c.managementClient(ctx, ownerOrgID)
	if err != nil {
		return "", err
	}

	resp, err := mc.ListProjectGrants(ctx, &managementv1.ListProjectGrantsRequest{
		ProjectId: projectID,
		Query:     &objectv1.ListQuery{Limit: 1000},
	})
	if err != nil {
		return "", err
	}

	for _, g := range resp.GetResult() {
		if g.GetGrantedOrgId() == grantedOrgID {
			return g.GetGrantId(), nil
		}
	}

	return "", ErrNotFound
}

// ProjectGrantStateFromProto maps the protobuf state of a project grant onto
// the API representation.
func ProjectGrantStateFromProto(s projectv2.ProjectGrantState) string {
	switch s {
	case projectv2.ProjectGrantState_PROJECT_GRANT_STATE_ACTIVE:
		return StateActive
	case projectv2.ProjectGrantState_PROJECT_GRANT_STATE_INACTIVE:
		return StateInactive
	case projectv2.ProjectGrantState_PROJECT_GRANT_STATE_UNSPECIFIED:
		return ""
	}

	return ""
}
