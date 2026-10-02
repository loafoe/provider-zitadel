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

	apiv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"
	filterv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/filter/v2"
)

// APIApplication is a Zitadel API application: a client used to authenticate
// service to service calls against the Zitadel API, or to authenticate a
// machine user through the Auth API.
type APIApplication struct {
	ApplicationID  string
	ProjectID      string
	Name           string
	State          string
	ClientID       string
	AuthMethodType string
	CreationDate   string
	ChangeDate     string
}

// CreateAPIApplicationInput describes an API application to create.
type CreateAPIApplicationInput struct {
	ProjectID string
	ID        string
	Name      string

	AuthMethodType apiv2.APIAuthMethodType
}

// CreateAPIApplication creates an API application and returns it together with
// the client secret, which Zitadel only reports once.
func (c *Client) CreateAPIApplication(ctx context.Context, in CreateAPIApplicationInput) (*APIApplication, string, error) {
	req := &apiv2.CreateApplicationRequest{
		ProjectId: in.ProjectID,
		Name:      in.Name,
		ApplicationType: &apiv2.CreateApplicationRequest_ApiConfiguration{
			ApiConfiguration: &apiv2.CreateAPIApplicationRequest{AuthMethodType: in.AuthMethodType},
		},
	}
	if in.ID != "" {
		req.ApplicationId = in.ID
	}

	resp, err := c.application.CreateApplication(ctx, req)
	if err != nil {
		return nil, "", fmt.Errorf("cannot create API application in project %s: %w", in.ProjectID, err)
	}

	out := &APIApplication{
		ApplicationID: resp.GetApplicationId(),
		ProjectID:     in.ProjectID,
		Name:          in.Name,
		State:         StateActive,
		CreationDate:  formatTimestamp(resp.GetCreationDate()),
	}
	if created, ok := resp.GetApplicationType().(*apiv2.CreateApplicationResponse_ApiConfiguration); ok {
		out.ClientID = created.ApiConfiguration.GetClientId()
		out.AuthMethodType = APIAuthMethodFromProto(in.AuthMethodType)
		return out, created.ApiConfiguration.GetClientSecret(), nil
	}

	return out, "", nil
}

// GetAPIApplication returns an API application by ID.
func (c *Client) GetAPIApplication(ctx context.Context, applicationID string) (*APIApplication, error) {
	resp, err := c.application.GetApplication(ctx, &apiv2.GetApplicationRequest{ApplicationId: applicationID})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get API application %s: %w", applicationID, err)
	}

	app := resp.GetApplication()
	if app == nil {
		return nil, ErrNotFound
	}

	cfg, ok := app.GetConfiguration().(*apiv2.Application_ApiConfiguration)
	if !ok {
		return nil, fmt.Errorf("application %s is not an API application", applicationID)
	}

	return &APIApplication{
		ApplicationID:  app.GetApplicationId(),
		ProjectID:      app.GetProjectId(),
		Name:           app.GetName(),
		State:          ApplicationStateFromProto(app.GetState()),
		ClientID:       cfg.ApiConfiguration.GetClientId(),
		AuthMethodType: APIAuthMethodFromProto(cfg.ApiConfiguration.GetAuthMethodType()),
		CreationDate:   formatTimestamp(app.GetCreationDate()),
		ChangeDate:     formatTimestamp(app.GetChangeDate()),
	}, nil
}

// FindAPIApplicationByName returns an API application of a project by name.
func (c *Client) FindAPIApplicationByName(ctx context.Context, projectID, name string) (*APIApplication, error) {
	resp, err := c.application.ListApplications(ctx, &apiv2.ListApplicationsRequest{
		Pagination: &filterv2.PaginationRequest{Limit: 1000},
		Filters: []*apiv2.ApplicationSearchFilter{
			{
				Filter: &apiv2.ApplicationSearchFilter_ProjectIdFilter{
					ProjectIdFilter: &apiv2.ProjectIDFilter{ProjectId: projectID},
				},
			},
			{
				Filter: &apiv2.ApplicationSearchFilter_NameFilter{
					NameFilter: &apiv2.ApplicationNameFilter{Name: name},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the applications of project %s: %w", projectID, err)
	}

	for _, a := range resp.GetApplications() {
		if a.GetName() != name {
			continue
		}
		if _, ok := a.GetConfiguration().(*apiv2.Application_ApiConfiguration); !ok {
			continue
		}
		got, err := c.GetAPIApplication(ctx, a.GetApplicationId())
		if err != nil {
			return nil, err
		}
		return got, nil
	}

	return nil, nil
}

// UpdateAPIApplication renames an application and changes its auth method type.
func (c *Client) UpdateAPIApplication(ctx context.Context, applicationID, projectID, name string, authMethod apiv2.APIAuthMethodType) error {
	req := &apiv2.UpdateApplicationRequest{
		ApplicationId: applicationID,
		ProjectId:     projectID,
		Name:          name,
		ApplicationType: &apiv2.UpdateApplicationRequest_ApiConfiguration{
			ApiConfiguration: &apiv2.UpdateAPIApplicationConfigurationRequest{AuthMethodType: authMethod},
		},
	}

	if _, err := c.application.UpdateApplication(ctx, req); err != nil {
		return fmt.Errorf("cannot update API application %s: %w", applicationID, err)
	}

	return nil
}

// SetAPIApplicationState activates or deactivates an application.
func (c *Client) SetAPIApplicationState(ctx context.Context, applicationID, current, desired string) error {
	if desired == "" || desired == current {
		return nil
	}

	var err error
	switch desired {
	case StateActive:
		_, err = c.application.ReactivateApplication(ctx, &apiv2.ReactivateApplicationRequest{ApplicationId: applicationID})
	case StateInactive:
		_, err = c.application.DeactivateApplication(ctx, &apiv2.DeactivateApplicationRequest{ApplicationId: applicationID})
	}

	if err != nil {
		return fmt.Errorf("cannot set the state of application %s to %s: %w", applicationID, desired, err)
	}

	return nil
}

// DeleteAPIApplication removes an application.
func (c *Client) DeleteAPIApplication(ctx context.Context, applicationID, projectID string) error {
	// Zitadel validates both fields, so the project is not optional even though
	// the application identifies itself.
	if _, err := c.application.DeleteApplication(ctx, &apiv2.DeleteApplicationRequest{
		ApplicationId: applicationID,
		ProjectId:     projectID,
	}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot delete API application %s: %w", applicationID, err)
	}

	return nil
}

// APIAuthMethodFromProto maps an API application auth method onto the API
// representation.
func APIAuthMethodFromProto(m apiv2.APIAuthMethodType) string {
	switch m {
	case apiv2.APIAuthMethodType_API_AUTH_METHOD_TYPE_PRIVATE_KEY_JWT:
		return AuthMethodPrivateKeyJwt
	case apiv2.APIAuthMethodType_API_AUTH_METHOD_TYPE_BASIC:
		return AuthMethodBasic
	}

	return ""
}

// APIAuthMethodToProto maps an API application auth method onto the protobuf
// representation.
func APIAuthMethodToProto(m string) (apiv2.APIAuthMethodType, error) {
	switch m {
	case "", AuthMethodBasic:
		return apiv2.APIAuthMethodType_API_AUTH_METHOD_TYPE_BASIC, nil
	case AuthMethodPrivateKeyJwt:
		return apiv2.APIAuthMethodType_API_AUTH_METHOD_TYPE_PRIVATE_KEY_JWT, nil
	default:
		return apiv2.APIAuthMethodType_API_AUTH_METHOD_TYPE_BASIC, fmt.Errorf("unsupported API auth method type %q", m)
	}
}
