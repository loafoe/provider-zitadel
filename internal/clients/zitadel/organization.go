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

	objectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/object/v2"
	orgv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/org/v2"
)

// Organization describes a Zitadel organization.
type Organization struct {
	ID            string
	Name          string
	PrimaryDomain string
	State         string
	CreationDate  string
	ChangeDate    string
}

// GetOrganization returns the organization with the supplied ID. It returns
// ErrNotFound when the organization does not exist.
func (c *Client) GetOrganization(ctx context.Context, organizationID string) (*Organization, error) {
	if organizationID == "" {
		return nil, fmt.Errorf("%w: empty organization ID", ErrNotFound)
	}

	resp, err := c.org.ListOrganizations(ctx, &orgv2.ListOrganizationsRequest{
		Query: &objectv2.ListQuery{Offset: 0, Limit: 1},
		Queries: []*orgv2.SearchQuery{
			{
				Query: &orgv2.SearchQuery_IdQuery{
					IdQuery: &orgv2.OrganizationIDQuery{Id: organizationID},
				},
			},
		},
	})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get organization %s: %w", organizationID, err)
	}

	if len(resp.GetResult()) == 0 {
		return nil, ErrNotFound
	}

	return fromProtoOrganization(resp.GetResult()[0]), nil
}

// ListOrganizations returns all organizations the caller has permission to
// read.
func (c *Client) ListOrganizations(ctx context.Context) ([]*Organization, error) {
	resp, err := c.org.ListOrganizations(ctx, &orgv2.ListOrganizationsRequest{
		Query: &objectv2.ListQuery{Offset: 0, Limit: 1000},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list organizations: %w", err)
	}

	out := make([]*Organization, 0, len(resp.GetResult()))
	for _, o := range resp.GetResult() {
		out = append(out, fromProtoOrganization(o))
	}

	return out, nil
}

// FindOrganizationByName returns the organization with the given name, or nil
// when no such organization exists.
func (c *Client) FindOrganizationByName(ctx context.Context, name string) (*Organization, error) {
	resp, err := c.org.ListOrganizations(ctx, &orgv2.ListOrganizationsRequest{
		Query: &objectv2.ListQuery{Offset: 0, Limit: 1000},
		Queries: []*orgv2.SearchQuery{
			{
				Query: &orgv2.SearchQuery_NameQuery{
					NameQuery: &orgv2.OrganizationNameQuery{Name: name},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list organizations named %q: %w", name, err)
	}

	for _, o := range resp.GetResult() {
		if o.GetName() == name {
			return fromProtoOrganization(o), nil
		}
	}

	return nil, nil
}

// fromProtoOrganization maps a protobuf organization onto the client
// representation.
func fromProtoOrganization(o *orgv2.Organization) *Organization {
	return &Organization{
		ID:            o.GetId(),
		Name:          o.GetName(),
		PrimaryDomain: o.GetPrimaryDomain(),
		State:         OrganizationStateFromProto(o.GetState()),
		CreationDate:  formatCreationDate(o.GetDetails()),
		ChangeDate:    formatChangeDate(o.GetDetails()),
	}
}

// CreateOrganization creates an organization and returns its ID.
func (c *Client) CreateOrganization(ctx context.Context, name, organizationID string) (string, error) {
	req := &orgv2.AddOrganizationRequest{Name: name}
	if organizationID != "" {
		id := organizationID
		req.OrganizationId = &id
	}

	resp, err := c.org.AddOrganization(ctx, req)
	if err != nil {
		return "", fmt.Errorf("cannot create organization %q: %w", name, err)
	}

	return resp.GetOrganizationId(), nil
}

// UpdateOrganization renames an organization.
func (c *Client) UpdateOrganization(ctx context.Context, organizationID, name string) error {
	if name == "" {
		return nil
	}

	if _, err := c.org.UpdateOrganization(ctx, &orgv2.UpdateOrganizationRequest{
		OrganizationId: organizationID,
		Name:           name,
	}); err != nil {
		return fmt.Errorf("cannot update organization %s: %w", organizationID, err)
	}

	return nil
}

// SetOrganizationState activates or deactivates an organization. No-op when the
// state already matches.
func (c *Client) SetOrganizationState(ctx context.Context, organizationID, current, desired string) error {
	if desired == "" || desired == current {
		return nil
	}

	switch desired {
	case StateActive:
		if _, err := c.org.ActivateOrganization(ctx, &orgv2.ActivateOrganizationRequest{
			OrganizationId: organizationID,
		}); err != nil {
			return fmt.Errorf("cannot activate organization %s: %w", organizationID, err)
		}
	case StateInactive:
		if _, err := c.org.DeactivateOrganization(ctx, &orgv2.DeactivateOrganizationRequest{
			OrganizationId: organizationID,
		}); err != nil {
			return fmt.Errorf("cannot deactivate organization %s: %w", organizationID, err)
		}
	}

	return nil
}

// DeleteOrganization removes an organization.
func (c *Client) DeleteOrganization(ctx context.Context, organizationID string) error {
	if _, err := c.org.DeleteOrganization(ctx, &orgv2.DeleteOrganizationRequest{
		OrganizationId: organizationID,
	}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot delete organization %s: %w", organizationID, err)
	}

	return nil
}
