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

	"github.com/zitadel/zitadel-go/v3/pkg/client/management"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel"
	appv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/app"
	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	"google.golang.org/protobuf/types/known/durationpb"
)

// zitadelOption and withOrgID are thin aliases so the management client reads
// like the rest of the package.
type zitadelOption = zitadel.Option

// withOrgID sets the organization context of a connection.
var withOrgID = zitadel.WithOrgID

// Why the v1 management API?
//
// The v2 application API creates and reads OIDC applications fine, but as of
// Zitadel 4.15 its UpdateApplication call reports `FailedPrecondition: No
// changes` for OIDC configuration changes and the new configuration is never
// persisted. The v1 management API is the code path the official Zitadel
// Terraform provider uses for `application_oidc`, and it applies the change
// correctly. The v1 API is organization scoped, so every call needs an explicit
// organization context.

// UpdateOIDCAppInput holds the OIDC configuration of an application, in the
// shape of the v1 management API.
type UpdateOIDCAppInput struct {
	// ProjectID is the project the application belongs to.
	ProjectID string

	// Name is the display name of the application. An empty name leaves the
	// current name untouched.
	Name string

	RedirectURIs             []string
	PostLogoutRedirectURIs   []string
	ResponseTypes            []appv1.OIDCResponseType
	GrantTypes               []appv1.OIDCGrantType
	ApplicationType          appv1.OIDCAppType
	AuthMethodType           appv1.OIDCAuthMethodType
	AccessTokenType          appv1.OIDCTokenType
	DevelopmentMode          bool
	AccessTokenRoleAssertion bool
	IDTokenRoleAssertion     bool
	IDTokenUserinfoAssertion bool
	ClockSkew                *durationpb.Duration
	AdditionalOrigins        []string
	SkipNativeAppSuccessPage bool
	BackChannelLogoutURI     string
}

// UpdateOIDCApplicationInOrg applies the OIDC configuration of an application
// through the v1 management API. orgID is the organization owning the project
// of the application.
func (c *Client) UpdateOIDCApplicationInOrg(ctx context.Context, orgID, applicationID string, in UpdateOIDCAppInput) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if in.Name != "" {
		//nolint:staticcheck // the v2 equivalent is broken, see the file comment
		if _, err := mc.UpdateApp(ctx, &managementv1.UpdateAppRequest{
			ProjectId: in.ProjectID,
			AppId:     applicationID,
			Name:      in.Name,
		}); err != nil && !IsNotFound(err) && !IsNoChanges(err) {
			return fmt.Errorf("cannot rename application %s: %w", applicationID, err)
		}
	}

	//nolint:staticcheck // the v2 equivalent is broken, see the file comment
	_, err = mc.UpdateOIDCAppConfig(ctx, &managementv1.UpdateOIDCAppConfigRequest{
		ProjectId:                in.ProjectID,
		AppId:                    applicationID,
		RedirectUris:             in.RedirectURIs,
		PostLogoutRedirectUris:   in.PostLogoutRedirectURIs,
		ResponseTypes:            in.ResponseTypes,
		GrantTypes:               in.GrantTypes,
		AppType:                  in.ApplicationType,
		AuthMethodType:           in.AuthMethodType,
		AccessTokenType:          in.AccessTokenType,
		DevMode:                  in.DevelopmentMode,
		AccessTokenRoleAssertion: in.AccessTokenRoleAssertion,
		IdTokenRoleAssertion:     in.IDTokenRoleAssertion,
		IdTokenUserinfoAssertion: in.IDTokenUserinfoAssertion,
		ClockSkew:                in.ClockSkew,
		AdditionalOrigins:        in.AdditionalOrigins,
		SkipNativeAppSuccessPage: in.SkipNativeAppSuccessPage,
		BackChannelLogoutUri:     in.BackChannelLogoutURI,
	})
	switch {
	case err == nil, IsNoChanges(err):
		// Zitadel rejects an update that changes nothing with a precondition
		// failure. That is exactly the state the caller asked for, so it is
		// reported as success.
		return nil

	case IsNotFound(err):
		return ErrNotFound

	default:
		return fmt.Errorf("cannot update the OIDC configuration of application %s: %w", applicationID, err)
	}
}

// managementClient returns a v1 management API client scoped to orgID,
// creating it on first use. Clients are cached per organization because the
// connection is what carries the organization context.
func (c *Client) managementClient(ctx context.Context, orgID string) (*management.Client, error) {
	c.managementMu.Lock()
	defer c.managementMu.Unlock()

	if mc, ok := c.management[orgID]; ok {
		return mc, nil
	}

	opts := append([]zitadelOption{}, c.options...)
	opts = append(opts, withOrgID(orgID))

	mc, err := management.NewClient(ctx, c.issuer, c.api, defaultScopes(), opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create the Zitadel management client for organization %s: %w", orgID, err)
	}

	if c.management == nil {
		c.management = map[string]*management.Client{}
	}

	c.management[orgID] = mc

	return mc, nil
}

// CloseManagement releases the cached management API clients.
func (c *Client) CloseManagement() {
	c.managementMu.Lock()
	defer c.managementMu.Unlock()

	for orgID, mc := range c.management {
		_ = mc.Connection.Close()
		delete(c.management, orgID)
	}
}

// ResponseTypeToV1 converts an OIDC response type into its v1 representation.
func ResponseTypeToV1(t string) (appv1.OIDCResponseType, error) {
	switch t {
	case "Code":
		return appv1.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE, nil
	case "IdToken":
		return appv1.OIDCResponseType_OIDC_RESPONSE_TYPE_ID_TOKEN, nil
	case "IdTokenToken":
		return appv1.OIDCResponseType_OIDC_RESPONSE_TYPE_ID_TOKEN_TOKEN, nil
	default:
		return appv1.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE, fmt.Errorf("unsupported OIDC response type %q", t)
	}
}

// GrantTypeToV1 converts an OIDC grant type into its v1 representation.
func GrantTypeToV1(t string) (appv1.OIDCGrantType, error) {
	switch t {
	case "AuthorizationCode":
		return appv1.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE, nil
	case "Implicit":
		return appv1.OIDCGrantType_OIDC_GRANT_TYPE_IMPLICIT, nil
	case "RefreshToken":
		return appv1.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN, nil
	case "DeviceCode":
		return appv1.OIDCGrantType_OIDC_GRANT_TYPE_DEVICE_CODE, nil
	case "TokenExchange":
		return appv1.OIDCGrantType_OIDC_GRANT_TYPE_TOKEN_EXCHANGE, nil
	default:
		return appv1.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE, fmt.Errorf("unsupported OIDC grant type %q", t)
	}
}

// ApplicationTypeToV1 converts an OIDC application type into its v1
// representation.
func ApplicationTypeToV1(t string) (appv1.OIDCAppType, error) {
	switch t {
	case "", "Web":
		return appv1.OIDCAppType_OIDC_APP_TYPE_WEB, nil
	case "UserAgent":
		return appv1.OIDCAppType_OIDC_APP_TYPE_USER_AGENT, nil
	case "Native":
		return appv1.OIDCAppType_OIDC_APP_TYPE_NATIVE, nil
	default:
		return appv1.OIDCAppType_OIDC_APP_TYPE_WEB, fmt.Errorf("unsupported OIDC application type %q", t)
	}
}

// AuthMethodTypeToV1 converts an OIDC auth method type into its v1
// representation.
func AuthMethodTypeToV1(t string) (appv1.OIDCAuthMethodType, error) {
	switch t {
	case "", "Basic":
		return appv1.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC, nil
	case "Post":
		return appv1.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_POST, nil
	case "None":
		return appv1.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_NONE, nil
	case "PrivateKeyJwt":
		return appv1.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_PRIVATE_KEY_JWT, nil
	default:
		return appv1.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC, fmt.Errorf("unsupported OIDC auth method type %q", t)
	}
}

// TokenTypeToV1 converts an OIDC token type into its v1 representation.
func TokenTypeToV1(t string) (appv1.OIDCTokenType, error) {
	switch t {
	case "", TokenTypeBearer:
		return appv1.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER, nil
	case TokenTypeJwt, "JWT":
		return appv1.OIDCTokenType_OIDC_TOKEN_TYPE_JWT, nil
	default:
		return appv1.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER, fmt.Errorf("unsupported OIDC token type %q", t)
	}
}
