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
	"fmt"
	"time"

	filterv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/filter/v2"
	projectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/project/v2"
)

// rfc3339 is the timestamp layout used for all timestamps returned by the
// client.
const rfc3339 = time.RFC3339

// CreateProjectInput describes a project that should be created.
type CreateProjectInput struct {
	// OrganizationID is the organization owning the project.
	OrganizationID string

	// ProjectID optionally pins the project ID.
	ProjectID string

	// Name is the display name of the project.
	Name string

	ProjectRoleAssertion   bool
	AuthorizationRequired  bool
	ProjectAccessRequired  bool
	PrivateLabelingSetting projectv2.PrivateLabelingSetting
}

// Project describes a Zitadel project.
type Project struct {
	ProjectID              string
	OrganizationID         string
	Name                   string
	State                  string
	ProjectRoleAssertion   bool
	AuthorizationRequired  bool
	ProjectAccessRequired  bool
	PrivateLabelingSetting string
	GrantedOrganizationID  string
	GrantedState           string
	CreationDate           string
	ChangeDate             string
}

// GetProject returns the project with the supplied ID. It returns ErrNotFound
// when the project does not exist.
func (c *Client) GetProject(ctx context.Context, projectID string) (*Project, error) {
	if projectID == "" {
		return nil, fmt.Errorf("%w: empty project ID", ErrNotFound)
	}

	resp, err := c.project.GetProject(ctx, &projectv2.GetProjectRequest{ProjectId: projectID})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get project %s: %w", projectID, err)
	}

	p := resp.GetProject()
	if p == nil {
		return nil, ErrNotFound
	}

	out := &Project{
		ProjectID:              p.GetProjectId(),
		OrganizationID:         p.GetOrganizationId(),
		Name:                   p.GetName(),
		State:                  ProjectStateFromProto(p.GetState()),
		ProjectRoleAssertion:   p.GetProjectRoleAssertion(),
		AuthorizationRequired:  p.GetAuthorizationRequired(),
		ProjectAccessRequired:  p.GetProjectAccessRequired(),
		PrivateLabelingSetting: PrivateLabelingSettingFromProto(p.GetPrivateLabelingSetting()),
	}
	if p.GrantedOrganizationId != nil {
		out.GrantedOrganizationID = p.GetGrantedOrganizationId()
	}
	out.GrantedState = GrantedProjectStateFromProto(p.GetGrantedState())
	if ts := p.GetCreationDate(); ts.IsValid() {
		out.CreationDate = ts.AsTime().Format(rfc3339)
	}
	if ts := p.GetChangeDate(); ts.IsValid() {
		out.ChangeDate = ts.AsTime().Format(rfc3339)
	}

	return out, nil
}

// ListProjectsByOrgID returns all projects of an organization. It is used to
// recover from a lost external name annotation.
func (c *Client) ListProjectsByOrgID(ctx context.Context, organizationID string) ([]*Project, error) {
	req := &projectv2.ListProjectsRequest{
		Pagination: &filterv2.PaginationRequest{Offset: 0, Limit: 1000},
		Filters: []*projectv2.ProjectSearchFilter{
			{
				Filter: &projectv2.ProjectSearchFilter_OrganizationIdFilter{
					OrganizationIdFilter: &projectv2.ProjectOrganizationIDFilter{
						OrganizationId: organizationID,
					},
				},
			},
		},
	}

	resp, err := c.project.ListProjects(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cannot list projects of organization %s: %w", organizationID, err)
	}

	out := make([]*Project, 0, len(resp.GetProjects()))
	for _, p := range resp.GetProjects() {
		project := &Project{
			ProjectID:              p.GetProjectId(),
			OrganizationID:         p.GetOrganizationId(),
			Name:                   p.GetName(),
			State:                  ProjectStateFromProto(p.GetState()),
			ProjectRoleAssertion:   p.GetProjectRoleAssertion(),
			AuthorizationRequired:  p.GetAuthorizationRequired(),
			ProjectAccessRequired:  p.GetProjectAccessRequired(),
			PrivateLabelingSetting: PrivateLabelingSettingFromProto(p.GetPrivateLabelingSetting()),
		}
		if ts := p.GetCreationDate(); ts.IsValid() {
			project.CreationDate = ts.AsTime().Format(rfc3339)
		}
		if ts := p.GetChangeDate(); ts.IsValid() {
			project.ChangeDate = ts.AsTime().Format(rfc3339)
		}
		out = append(out, project)
	}

	return out, nil
}

// CreateProject creates a project and returns its ID.
func (c *Client) CreateProject(ctx context.Context, in CreateProjectInput) (string, error) {
	req := &projectv2.CreateProjectRequest{
		OrganizationId:         in.OrganizationID,
		Name:                   in.Name,
		ProjectRoleAssertion:   in.ProjectRoleAssertion,
		AuthorizationRequired:  in.AuthorizationRequired,
		ProjectAccessRequired:  in.ProjectAccessRequired,
		PrivateLabelingSetting: in.PrivateLabelingSetting,
	}
	if in.ProjectID != "" {
		id := in.ProjectID
		req.ProjectId = &id
	}

	resp, err := c.project.CreateProject(ctx, req)
	if err != nil {
		return "", fmt.Errorf("cannot create project in organization %s: %w", in.OrganizationID, err)
	}

	return resp.GetProjectId(), nil
}

// UpdateProjectInput holds the mutable attributes of a project. Nil fields are
// left untouched.
type UpdateProjectInput struct {
	Name                   *string
	ProjectRoleAssertion   *bool
	AuthorizationRequired  *bool
	ProjectAccessRequired  *bool
	PrivateLabelingSetting *projectv2.PrivateLabelingSetting
}

// UpdateProject updates the mutable attributes of a project. An input without
// any field set is a no-op.
func (c *Client) UpdateProject(ctx context.Context, projectID string, in UpdateProjectInput) error {
	req := &projectv2.UpdateProjectRequest{
		ProjectId:              projectID,
		Name:                   in.Name,
		ProjectRoleAssertion:   in.ProjectRoleAssertion,
		AuthorizationRequired:  in.AuthorizationRequired,
		ProjectAccessRequired:  in.ProjectAccessRequired,
		PrivateLabelingSetting: in.PrivateLabelingSetting,
	}

	if req.Name == nil && req.ProjectRoleAssertion == nil && req.AuthorizationRequired == nil &&
		req.ProjectAccessRequired == nil && req.PrivateLabelingSetting == nil {
		return nil
	}

	if _, err := c.project.UpdateProject(ctx, req); err != nil {
		return fmt.Errorf("cannot update project %s: %w", projectID, err)
	}

	return nil
}

// SetProjectState activates or deactivates a project. No-op when the state
// already matches.
func (c *Client) SetProjectState(ctx context.Context, projectID, current, desired string) error {
	if desired == "" || desired == current {
		return nil
	}

	switch desired {
	case StateActive:
		if _, err := c.project.ActivateProject(ctx, &projectv2.ActivateProjectRequest{ProjectId: projectID}); err != nil {
			return fmt.Errorf("cannot activate project %s: %w", projectID, err)
		}
	case StateInactive:
		if _, err := c.project.DeactivateProject(ctx, &projectv2.DeactivateProjectRequest{ProjectId: projectID}); err != nil {
			return fmt.Errorf("cannot deactivate project %s: %w", projectID, err)
		}
	}

	return nil
}

// DeleteProject removes a project.
func (c *Client) DeleteProject(ctx context.Context, projectID string) error {
	if _, err := c.project.DeleteProject(ctx, &projectv2.DeleteProjectRequest{ProjectId: projectID}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot delete project %s: %w", projectID, err)
	}

	return nil
}

// ProjectRole describes a role of a project.
type ProjectRole struct {
	Key          string
	DisplayName  string
	Group        string
	CreationDate string
	ChangeDate   string
}

// GetProjectRole returns a single project role. It returns ErrNotFound when the
// role does not exist.
func (c *Client) GetProjectRole(ctx context.Context, projectID, key string) (*ProjectRole, error) {
	resp, err := c.project.ListProjectRoles(ctx, &projectv2.ListProjectRolesRequest{
		ProjectId:  projectID,
		Pagination: &filterv2.PaginationRequest{Offset: 0, Limit: 1000},
	})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot list project roles of project %s: %w", projectID, err)
	}

	for _, r := range resp.GetProjectRoles() {
		if r.GetKey() == key {
			return fromProtoProjectRole(r), nil
		}
	}

	return nil, ErrNotFound
}

// AddProjectRole creates a role in a project and returns its key.
func (c *Client) AddProjectRole(ctx context.Context, projectID, key, displayName, group string) (string, error) {
	req := &projectv2.AddProjectRoleRequest{
		ProjectId:   projectID,
		RoleKey:     key,
		DisplayName: displayName,
	}
	if group != "" {
		g := group
		req.Group = &g
	}

	resp, err := c.project.AddProjectRole(ctx, req)
	if err != nil {
		return "", fmt.Errorf("cannot add project role %s to project %s: %w", key, projectID, err)
	}
	_ = resp

	return key, nil
}

// UpdateProjectRole updates the display name and group of a project role.
func (c *Client) UpdateProjectRole(ctx context.Context, projectID, key, displayName, group string) error {
	req := &projectv2.UpdateProjectRoleRequest{
		ProjectId:   projectID,
		RoleKey:     key,
		DisplayName: &displayName,
		Group:       &group,
	}

	if _, err := c.project.UpdateProjectRole(ctx, req); err != nil {
		return fmt.Errorf("cannot update project role %s of project %s: %w", key, projectID, err)
	}

	return nil
}

// RemoveProjectRole deletes a role from a project.
func (c *Client) RemoveProjectRole(ctx context.Context, projectID, key string) error {
	if _, err := c.project.RemoveProjectRole(ctx, &projectv2.RemoveProjectRoleRequest{
		ProjectId: projectID,
		RoleKey:   key,
	}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot remove project role %s from project %s: %w", key, projectID, err)
	}

	return nil
}

func fromProtoProjectRole(r *projectv2.ProjectRole) *ProjectRole {
	out := &ProjectRole{
		Key:         r.GetKey(),
		DisplayName: r.GetDisplayName(),
		Group:       r.GetGroup(),
	}
	if ts := r.GetCreationDate(); ts.IsValid() {
		out.CreationDate = ts.AsTime().Format(rfc3339)
	}
	if ts := r.GetChangeDate(); ts.IsValid() {
		out.ChangeDate = ts.AsTime().Format(rfc3339)
	}

	return out
}

// ListProjectRoles returns the roles a project defines. A user grant on a
// project of its own organization may only use these.
func (c *Client) ListProjectRoles(ctx context.Context, projectID string) ([]string, error) {
	resp, err := c.project.ListProjectRoles(ctx, &projectv2.ListProjectRolesRequest{
		ProjectId:  projectID,
		Pagination: &filterv2.PaginationRequest{Limit: 1000},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the roles of project %s: %w", projectID, err)
	}

	out := make([]string, 0, len(resp.GetProjectRoles()))
	for _, r := range resp.GetProjectRoles() {
		out = append(out, r.GetKey())
	}

	return out, nil
}
