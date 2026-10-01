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
	"google.golang.org/protobuf/types/known/durationpb"
)

// CreateOIDCApplicationInput describes an OIDC application that should be
// created.
type CreateOIDCApplicationInput struct {
	// ProjectID is the project the application belongs to.
	ProjectID string

	// ApplicationID optionally pins the application ID.
	ApplicationID string

	// Name is the display name of the application.
	Name string

	RedirectURIs             []string
	PostLogoutRedirectURIs   []string
	ResponseTypes            []apiv2.OIDCResponseType
	GrantTypes               []apiv2.OIDCGrantType
	ApplicationType          apiv2.OIDCApplicationType
	AuthMethodType           apiv2.OIDCAuthMethodType
	Version                  apiv2.OIDCVersion
	DevelopmentMode          bool
	AccessTokenType          apiv2.OIDCTokenType
	AccessTokenRoleAssertion bool
	IDTokenRoleAssertion     bool
	IDTokenUserinfoAssertion bool
	ClockSkew                *durationpb.Duration
	AdditionalOrigins        []string
	SkipNativeAppSuccessPage bool
	BackChannelLogoutURI     string
}

// CreateOIDCApplicationResult is returned after creating an OIDC application.
type CreateOIDCApplicationResult struct {
	// ApplicationID is the ID of the created application.
	ApplicationID string

	// ClientID is the OAuth2 client ID of the application.
	ClientID string

	// ClientSecret is the client secret. Zitadel only returns it once.
	ClientSecret string

	// NonCompliant is true when the configuration does not comply with the
	// OIDC specification.
	NonCompliant bool

	// ComplianceProblems lists the reasons for a non compliant configuration.
	ComplianceProblems []string
}

// OIDCApplication describes an OIDC application.
type OIDCApplication struct {
	ApplicationID            string
	ProjectID                string
	Name                     string
	State                    string
	ClientID                 string
	RedirectURIs             []string
	PostLogoutRedirectURIs   []string
	ResponseTypes            []apiv2.OIDCResponseType
	GrantTypes               []apiv2.OIDCGrantType
	ApplicationType          apiv2.OIDCApplicationType
	AuthMethodType           apiv2.OIDCAuthMethodType
	AccessTokenType          apiv2.OIDCTokenType
	Version                  apiv2.OIDCVersion
	DevelopmentMode          bool
	AccessTokenRoleAssertion bool
	IDTokenRoleAssertion     bool
	IDTokenUserinfoAssertion bool
	ClockSkew                string
	AdditionalOrigins        []string
	AllowedOrigins           []string
	SkipNativeAppSuccessPage bool
	BackChannelLogoutURI     string
	NonCompliant             bool
	CreationDate             string
	ChangeDate               string
}

// GetOIDCApplication returns the OIDC application with the supplied ID. It
// returns ErrNotFound when the application does not exist.
func (c *Client) GetOIDCApplication(ctx context.Context, applicationID string) (*OIDCApplication, error) {
	if applicationID == "" {
		return nil, fmt.Errorf("%w: empty application ID", ErrNotFound)
	}

	resp, err := c.application.GetApplication(ctx, &apiv2.GetApplicationRequest{ApplicationId: applicationID})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get application %s: %w", applicationID, err)
	}

	app := resp.GetApplication()
	if app == nil {
		return nil, ErrNotFound
	}

	out := &OIDCApplication{
		ApplicationID: app.GetApplicationId(),
		ProjectID:     app.GetProjectId(),
		Name:          app.GetName(),
		State:         ApplicationStateFromProto(app.GetState()),
		CreationDate:  formatTimestamp(app.GetCreationDate()),
		ChangeDate:    formatTimestamp(app.GetChangeDate()),
	}

	cfg, ok := app.GetConfiguration().(*apiv2.Application_OidcConfiguration)
	if !ok {
		return nil, fmt.Errorf("application %s is not an OIDC application", applicationID)
	}

	o := cfg.OidcConfiguration
	out.ClientID = o.GetClientId()
	out.RedirectURIs = o.GetRedirectUris()
	out.PostLogoutRedirectURIs = o.GetPostLogoutRedirectUris()
	out.ResponseTypes = o.GetResponseTypes()
	out.GrantTypes = o.GetGrantTypes()
	out.ApplicationType = o.GetApplicationType()
	out.AuthMethodType = o.GetAuthMethodType()
	out.Version = o.GetVersion()
	out.AccessTokenType = o.GetAccessTokenType()
	out.DevelopmentMode = o.GetDevelopmentMode()
	out.AccessTokenRoleAssertion = o.GetAccessTokenRoleAssertion()
	out.IDTokenRoleAssertion = o.GetIdTokenRoleAssertion()
	out.IDTokenUserinfoAssertion = o.GetIdTokenUserinfoAssertion()
	out.AdditionalOrigins = o.GetAdditionalOrigins()
	out.AllowedOrigins = o.GetAllowedOrigins()
	out.SkipNativeAppSuccessPage = o.GetSkipNativeAppSuccessPage()
	out.BackChannelLogoutURI = o.GetBackChannelLogoutUri()
	out.NonCompliant = o.GetNonCompliant()

	if skew := o.GetClockSkew(); skew != nil {
		out.ClockSkew = skew.AsDuration().String()
	}

	return out, nil
}

// ListOIDCApplicationsByProjectID returns all applications of a project. It is
// used to recover from a lost external name annotation.
func (c *Client) ListOIDCApplicationsByProjectID(ctx context.Context, projectID string) ([]*OIDCApplication, error) {
	resp, err := c.application.ListApplications(ctx, &apiv2.ListApplicationsRequest{
		Pagination: &filterv2.PaginationRequest{Offset: 0, Limit: 1000},
		Filters: []*apiv2.ApplicationSearchFilter{
			{
				Filter: &apiv2.ApplicationSearchFilter_ProjectIdFilter{
					ProjectIdFilter: &apiv2.ProjectIDFilter{ProjectId: projectID},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list applications of project %s: %w", projectID, err)
	}

	out := make([]*OIDCApplication, 0, len(resp.GetApplications()))
	for _, app := range resp.GetApplications() {
		a := &OIDCApplication{
			ApplicationID: app.GetApplicationId(),
			ProjectID:     app.GetProjectId(),
			Name:          app.GetName(),
			State:         ApplicationStateFromProto(app.GetState()),
			CreationDate:  formatTimestamp(app.GetCreationDate()),
			ChangeDate:    formatTimestamp(app.GetChangeDate()),
		}
		if cfg, ok := app.GetConfiguration().(*apiv2.Application_OidcConfiguration); ok {
			o := cfg.OidcConfiguration
			a.ClientID = o.GetClientId()
			a.RedirectURIs = o.GetRedirectUris()
			a.PostLogoutRedirectURIs = o.GetPostLogoutRedirectUris()
			a.ResponseTypes = o.GetResponseTypes()
			a.GrantTypes = o.GetGrantTypes()
			a.ApplicationType = o.GetApplicationType()
			a.AuthMethodType = o.GetAuthMethodType()
			a.Version = o.GetVersion()
			a.AccessTokenType = o.GetAccessTokenType()
			a.DevelopmentMode = o.GetDevelopmentMode()
			a.AccessTokenRoleAssertion = o.GetAccessTokenRoleAssertion()
			a.IDTokenRoleAssertion = o.GetIdTokenRoleAssertion()
			a.IDTokenUserinfoAssertion = o.GetIdTokenUserinfoAssertion()
			a.AdditionalOrigins = o.GetAdditionalOrigins()
			a.AllowedOrigins = o.GetAllowedOrigins()
			a.SkipNativeAppSuccessPage = o.GetSkipNativeAppSuccessPage()
			a.BackChannelLogoutURI = o.GetBackChannelLogoutUri()
			a.NonCompliant = o.GetNonCompliant()
			if skew := o.GetClockSkew(); skew != nil {
				a.ClockSkew = skew.AsDuration().String()
			}
		}
		out = append(out, a)
	}

	return out, nil
}

// CreateOIDCApplication creates an OIDC application.
func (c *Client) CreateOIDCApplication(ctx context.Context, in CreateOIDCApplicationInput) (*CreateOIDCApplicationResult, error) { //nolint:gocyclo // flat request construction
	oidc := &apiv2.CreateOIDCApplicationRequest{
		RedirectUris:             in.RedirectURIs,
		PostLogoutRedirectUris:   in.PostLogoutRedirectURIs,
		ResponseTypes:            in.ResponseTypes,
		GrantTypes:               in.GrantTypes,
		ApplicationType:          in.ApplicationType,
		AuthMethodType:           in.AuthMethodType,
		Version:                  in.Version,
		DevelopmentMode:          in.DevelopmentMode,
		AccessTokenType:          in.AccessTokenType,
		AccessTokenRoleAssertion: in.AccessTokenRoleAssertion,
		IdTokenRoleAssertion:     in.IDTokenRoleAssertion,
		IdTokenUserinfoAssertion: in.IDTokenUserinfoAssertion,
		ClockSkew:                in.ClockSkew,
		AdditionalOrigins:        in.AdditionalOrigins,
		SkipNativeAppSuccessPage: in.SkipNativeAppSuccessPage,
		BackChannelLogoutUri:     in.BackChannelLogoutURI,
	}

	req := &apiv2.CreateApplicationRequest{
		ProjectId: in.ProjectID,
		Name:      in.Name,
		ApplicationType: &apiv2.CreateApplicationRequest_OidcConfiguration{
			OidcConfiguration: oidc,
		},
	}
	if in.ApplicationID != "" {
		req.ApplicationId = in.ApplicationID
	}

	resp, err := c.application.CreateApplication(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cannot create application in project %s: %w", in.ProjectID, err)
	}

	out := &CreateOIDCApplicationResult{ApplicationID: resp.GetApplicationId()}

	if created, ok := resp.GetApplicationType().(*apiv2.CreateApplicationResponse_OidcConfiguration); ok {
		r := created.OidcConfiguration
		out.ClientID = r.GetClientId()
		out.ClientSecret = r.GetClientSecret()
		out.NonCompliant = r.GetNonCompliant()
		for _, p := range r.GetComplianceProblems() {
			out.ComplianceProblems = append(out.ComplianceProblems, p.GetLocalizedMessage())
		}
	}

	return out, nil
}

// UpdateOIDCApplicationInput holds the mutable attributes of an OIDC
// application. Nil fields are left untouched.
type UpdateOIDCApplicationInput struct {
	ProjectID                string
	Name                     string
	RedirectURIs             []string
	PostLogoutRedirectURIs   []string
	ResponseTypes            []apiv2.OIDCResponseType
	GrantTypes               []apiv2.OIDCGrantType
	ApplicationType          *apiv2.OIDCApplicationType
	AuthMethodType           *apiv2.OIDCAuthMethodType
	Version                  *apiv2.OIDCVersion
	DevelopmentMode          *bool
	AccessTokenType          *apiv2.OIDCTokenType
	AccessTokenRoleAssertion *bool
	IDTokenRoleAssertion     *bool
	IDTokenUserinfoAssertion *bool
	ClockSkew                *durationpb.Duration
	AdditionalOrigins        []string
	SkipNativeAppSuccessPage *bool
	BackChannelLogoutURI     *string
}

// UpdateOIDCApplication updates the mutable attributes of an OIDC application.
func (c *Client) UpdateOIDCApplication(ctx context.Context, applicationID string, in UpdateOIDCApplicationInput) error {
	oidc := &apiv2.UpdateOIDCApplicationConfigurationRequest{
		RedirectUris:             in.RedirectURIs,
		PostLogoutRedirectUris:   in.PostLogoutRedirectURIs,
		ResponseTypes:            in.ResponseTypes,
		GrantTypes:               in.GrantTypes,
		ApplicationType:          in.ApplicationType,
		AuthMethodType:           in.AuthMethodType,
		Version:                  in.Version,
		DevelopmentMode:          in.DevelopmentMode,
		AccessTokenType:          in.AccessTokenType,
		AccessTokenRoleAssertion: in.AccessTokenRoleAssertion,
		IdTokenRoleAssertion:     in.IDTokenRoleAssertion,
		IdTokenUserinfoAssertion: in.IDTokenUserinfoAssertion,
		ClockSkew:                in.ClockSkew,
		AdditionalOrigins:        in.AdditionalOrigins,
		SkipNativeAppSuccessPage: in.SkipNativeAppSuccessPage,
		BackChannelLogoutUri:     in.BackChannelLogoutURI,
	}

	req := &apiv2.UpdateApplicationRequest{
		ApplicationId: applicationID,
		ProjectId:     in.ProjectID,
		ApplicationType: &apiv2.UpdateApplicationRequest_OidcConfiguration{
			OidcConfiguration: oidc,
		},
	}
	if in.Name != "" {
		req.Name = in.Name
	}

	if _, err := c.application.UpdateApplication(ctx, req); err != nil {
		return fmt.Errorf("cannot update application %s: %w", applicationID, err)
	}

	return nil
}

// SetApplicationState activates or deactivates an application. No-op when the
// state already matches.
func (c *Client) SetApplicationState(ctx context.Context, applicationID, current, desired string) error {
	if desired == "" || desired == current {
		return nil
	}

	switch desired {
	case StateActive:
		if _, err := c.application.ReactivateApplication(ctx, &apiv2.ReactivateApplicationRequest{
			ApplicationId: applicationID,
		}); err != nil {
			return fmt.Errorf("cannot reactivate application %s: %w", applicationID, err)
		}
	case StateInactive:
		if _, err := c.application.DeactivateApplication(ctx, &apiv2.DeactivateApplicationRequest{
			ApplicationId: applicationID,
		}); err != nil {
			return fmt.Errorf("cannot deactivate application %s: %w", applicationID, err)
		}
	}

	return nil
}

// GenerateClientSecret issues a new client secret for an OIDC application and
// returns it. The previous secret stops working immediately.
func (c *Client) GenerateClientSecret(ctx context.Context, applicationID string) (string, error) {
	resp, err := c.application.GenerateClientSecret(ctx, &apiv2.GenerateClientSecretRequest{
		ApplicationId: applicationID,
	})
	if err != nil {
		return "", fmt.Errorf("cannot generate client secret for application %s: %w", applicationID, err)
	}

	return resp.GetClientSecret(), nil
}

// DeleteOIDCApplication removes an application.
func (c *Client) DeleteOIDCApplication(ctx context.Context, applicationID string) error {
	if _, err := c.application.DeleteApplication(ctx, &apiv2.DeleteApplicationRequest{
		ApplicationId: applicationID,
	}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot delete application %s: %w", applicationID, err)
	}

	return nil
}
