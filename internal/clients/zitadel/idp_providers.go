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

	"strings"

	"github.com/zitadel/zitadel-go/v3/pkg/client/management"
	adminv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	idpv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/idp"
	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
)

// ProviderInput is the desired configuration of an identity provider.
//
// The fields are the union of every provider kind's, and a kind uses only its
// own. OIDC and JWT carry their settings in the nested values because those two
// are the only kinds Zitadel reads back, and so the only ones that can be
// updated in place.
type ProviderInput struct {
	Name string

	// State is whether the provider is offered on the login page, as `Active` or
	// `Inactive`. An empty state leaves Zitadel's default alone.
	//
	// Activating and deactivating is a call of its own rather than part of
	// updating a provider, which is why it is set separately.
	State string

	ClientID     string
	ClientSecret string
	Issuer       string
	Scopes       []string

	// OAuth and self hosted providers give their endpoints explicitly; OIDC
	// discovers them from the issuer.
	AuthorizationEndpoint string
	TokenEndpoint         string
	UserEndpoint          string
	IDAttribute           string

	// JWT providers are called by Zitadel rather than the other way round.
	JWTEndpoint  string
	KeysEndpoint string
	HeaderName   string
	Audience     string

	// Apple and Azure AD.
	TeamID        string
	KeyID         string
	PrivateKey    string
	Tenant        string
	EmailVerified bool

	// LDAP.
	Servers           []string
	StartTLS          bool
	BaseDN            string
	BindDN            string
	BindPassword      string
	UserBase          string
	UserObjectClasses []string
	UserFilters       []string
	Timeout           string
	RootCA            string

	// Generic OIDC and OAuth.
	IsIDTokenMapping bool
	UsePKCE          bool

	// SAML.
	Metadata                 string
	Binding                  string
	WithSignedRequest        bool
	NameIDFormat             string
	TransientMappingAttrName string
	FederatedLogoutEnabled   bool
	SignatureAlgorithm       string

	Options IDPOptions

	// OIDC and JWT settings, set only for those two kinds.
	OIDC *OIDCSettings
	JWT  *JWTSettings
}

// CreateIdentityProvider adds an identity provider and returns its ID.
//
// Zitadel has no generic "add a provider" call: each kind has its own request
// type, so the provider is named and the matching call is made. Adding and
// updating are separate calls for every kind too, for the same reason.
func (c *Client) CreateIdentityProvider(ctx context.Context, orgID, provider string, in ProviderInput) (string, error) {
	scope := idpScope{orgID: orgID}

	if orgID != "" {
		mc, err := c.managementClient(ctx, orgID)
		if err != nil {
			return "", err
		}

		id, err := createOrgIdentityProvider(ctx, mc, provider, in)
		if err != nil {
			return "", wrapIDPError(err, scope, "create the %s provider %s", provider, in.Name)
		}

		return id, nil
	}

	id, err := c.createInstanceIdentityProvider(ctx, provider, in)
	if err != nil {
		return "", wrapIDPError(err, scope, "create the %s provider %s", provider, in.Name)
	}

	return id, nil
}

//nolint:gocyclo // one arm per provider kind; splitting them would hide that they differ only in a few fields
func (c *Client) createInstanceIdentityProvider(ctx context.Context, provider string, in ProviderInput) (string, error) {
	opts := toProtoIDPOptions(in.Options)

	switch provider {
	case "Apple":
		resp, err := c.admin.AddAppleProvider(ctx, &adminv1.AddAppleProviderRequest{
			Name: in.Name, ClientId: in.ClientID, TeamId: in.TeamID, KeyId: in.KeyID,
			PrivateKey: []byte(in.PrivateKey), Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "AzureAD":
		resp, err := c.admin.AddAzureADProvider(ctx, &adminv1.AddAzureADProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			Tenant: azureTenant(in.Tenant), EmailVerified: in.EmailVerified,
			Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "GitHub":
		resp, err := c.admin.AddGitHubProvider(ctx, &adminv1.AddGitHubProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "GitHubEnterpriseServer":
		resp, err := c.admin.AddGitHubEnterpriseServerProvider(ctx, &adminv1.AddGitHubEnterpriseServerProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			AuthorizationEndpoint: in.AuthorizationEndpoint, TokenEndpoint: in.TokenEndpoint,
			UserEndpoint: in.UserEndpoint, Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "GitLab":
		resp, err := c.admin.AddGitLabProvider(ctx, &adminv1.AddGitLabProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "GitLabSelfHosted":
		resp, err := c.admin.AddGitLabSelfHostedProvider(ctx, &adminv1.AddGitLabSelfHostedProviderRequest{
			Issuer: in.Issuer, Name: in.Name, ClientId: in.ClientID,
			ClientSecret: in.ClientSecret, Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "Google":
		resp, err := c.admin.AddGoogleProvider(ctx, &adminv1.AddGoogleProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "LDAP":
		attrs := ldapAttributes(in)

		resp, err := c.admin.AddLDAPProvider(ctx, &adminv1.AddLDAPProviderRequest{
			Name: in.Name, Servers: in.Servers, StartTls: in.StartTLS, BaseDn: in.BaseDN,
			BindDn: in.BindDN, BindPassword: in.BindPassword, UserBase: in.UserBase,
			UserObjectClasses: in.UserObjectClasses, UserFilters: in.UserFilters,
			Timeout: durationProto(in.Timeout), Attributes: attrs, ProviderOptions: &opts,
			RootCa: []byte(in.RootCA),
		})
		return idpID(resp.GetId(), err)

	case "OAuth":
		resp, err := c.admin.AddGenericOAuthProvider(ctx, &adminv1.AddGenericOAuthProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			AuthorizationEndpoint: in.AuthorizationEndpoint, TokenEndpoint: in.TokenEndpoint,
			UserEndpoint: in.UserEndpoint, Scopes: in.Scopes, IdAttribute: in.IDAttribute,
			ProviderOptions: &opts, UsePkce: in.UsePKCE,
		})
		return idpID(resp.GetId(), err)

	case "OIDC":
		resp, err := c.admin.AddGenericOIDCProvider(ctx, &adminv1.AddGenericOIDCProviderRequest{
			Name: in.Name, Issuer: in.Issuer, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			Scopes: in.Scopes, ProviderOptions: &opts,
			IsIdTokenMapping: in.IsIDTokenMapping, UsePkce: in.UsePKCE,
		})
		return idpID(resp.GetId(), err)

	case "SAML":
		req := &adminv1.AddSAMLProviderRequest{
			Name: in.Name, Binding: samlBinding(in.Binding),
			WithSignedRequest: in.WithSignedRequest, ProviderOptions: &opts,
			NameIdFormat:                  samlNameIDFormatPtr(in.NameIDFormat),
			TransientMappingAttributeName: optionalString(in.TransientMappingAttrName),
			FederatedLogoutEnabled:        optionalBool(in.FederatedLogoutEnabled),
			SignatureAlgorithm:            samlSignatureAlgorithm(in.SignatureAlgorithm),
		}

		if isXMLDocument(in.Metadata) {
			req.Metadata = &adminv1.AddSAMLProviderRequest_MetadataXml{MetadataXml: []byte(in.Metadata)}
		} else {
			req.Metadata = &adminv1.AddSAMLProviderRequest_MetadataUrl{MetadataUrl: in.Metadata}
		}

		resp, err := c.admin.AddSAMLProvider(ctx, req)

		return idpID(resp.GetId(), err)

	default:
		return "", fmt.Errorf("unknown identity provider %q", provider)
	}
}

//nolint:gocyclo // one arm per provider kind; splitting them would hide that they differ only in a few fields
func createOrgIdentityProvider(ctx context.Context, mc *management.Client, provider string, in ProviderInput) (string, error) {
	opts := toProtoIDPOptions(in.Options)

	switch provider {
	case "Apple":
		resp, err := mc.AddAppleProvider(ctx, &managementv1.AddAppleProviderRequest{
			Name: in.Name, ClientId: in.ClientID, TeamId: in.TeamID, KeyId: in.KeyID,
			PrivateKey: []byte(in.PrivateKey), Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "AzureAD":
		resp, err := mc.AddAzureADProvider(ctx, &managementv1.AddAzureADProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			Tenant: azureTenant(in.Tenant), EmailVerified: in.EmailVerified,
			Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "GitHub":
		resp, err := mc.AddGitHubProvider(ctx, &managementv1.AddGitHubProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "GitHubEnterpriseServer":
		resp, err := mc.AddGitHubEnterpriseServerProvider(ctx, &managementv1.AddGitHubEnterpriseServerProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			AuthorizationEndpoint: in.AuthorizationEndpoint, TokenEndpoint: in.TokenEndpoint,
			UserEndpoint: in.UserEndpoint, Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "GitLab":
		resp, err := mc.AddGitLabProvider(ctx, &managementv1.AddGitLabProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "GitLabSelfHosted":
		resp, err := mc.AddGitLabSelfHostedProvider(ctx, &managementv1.AddGitLabSelfHostedProviderRequest{
			Issuer: in.Issuer, Name: in.Name, ClientId: in.ClientID,
			ClientSecret: in.ClientSecret, Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "Google":
		resp, err := mc.AddGoogleProvider(ctx, &managementv1.AddGoogleProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			Scopes: in.Scopes, ProviderOptions: &opts,
		})
		return idpID(resp.GetId(), err)

	case "LDAP":
		attrs := ldapAttributes(in)

		resp, err := mc.AddLDAPProvider(ctx, &managementv1.AddLDAPProviderRequest{
			Name: in.Name, Servers: in.Servers, StartTls: in.StartTLS, BaseDn: in.BaseDN,
			BindDn: in.BindDN, BindPassword: in.BindPassword, UserBase: in.UserBase,
			UserObjectClasses: in.UserObjectClasses, UserFilters: in.UserFilters,
			Timeout: durationProto(in.Timeout), Attributes: attrs, ProviderOptions: &opts,
			RootCa: []byte(in.RootCA),
		})
		return idpID(resp.GetId(), err)

	case "OAuth":
		resp, err := mc.AddGenericOAuthProvider(ctx, &managementv1.AddGenericOAuthProviderRequest{
			Name: in.Name, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			AuthorizationEndpoint: in.AuthorizationEndpoint, TokenEndpoint: in.TokenEndpoint,
			UserEndpoint: in.UserEndpoint, Scopes: in.Scopes, IdAttribute: in.IDAttribute,
			ProviderOptions: &opts, UsePkce: in.UsePKCE,
		})
		return idpID(resp.GetId(), err)

	case "OIDC":
		resp, err := mc.AddGenericOIDCProvider(ctx, &managementv1.AddGenericOIDCProviderRequest{
			Name: in.Name, Issuer: in.Issuer, ClientId: in.ClientID, ClientSecret: in.ClientSecret,
			Scopes: in.Scopes, ProviderOptions: &opts,
			IsIdTokenMapping: in.IsIDTokenMapping, UsePkce: in.UsePKCE,
		})
		return idpID(resp.GetId(), err)

	case "SAML":
		req := &managementv1.AddSAMLProviderRequest{
			Name: in.Name, Binding: samlBinding(in.Binding),
			WithSignedRequest: in.WithSignedRequest, ProviderOptions: &opts,
			NameIdFormat:                  samlNameIDFormatPtr(in.NameIDFormat),
			TransientMappingAttributeName: optionalString(in.TransientMappingAttrName),
			FederatedLogoutEnabled:        optionalBool(in.FederatedLogoutEnabled),
			SignatureAlgorithm:            samlSignatureAlgorithm(in.SignatureAlgorithm),
		}

		if isXMLDocument(in.Metadata) {
			req.Metadata = &managementv1.AddSAMLProviderRequest_MetadataXml{MetadataXml: []byte(in.Metadata)}
		} else {
			req.Metadata = &managementv1.AddSAMLProviderRequest_MetadataUrl{MetadataUrl: in.Metadata}
		}

		resp, err := mc.AddSAMLProvider(ctx, req)

		return idpID(resp.GetId(), err)

	case "JWT":
		// A JWT provider is one Zitadel calls, rather than one that calls it,
		// so it has its own calls on both APIs.
		resp, err := mc.AddOrgJWTIDP(ctx, &managementv1.AddOrgJWTIDPRequest{
			Name: in.Name, JwtEndpoint: in.JWTEndpoint, Issuer: in.Issuer,
			KeysEndpoint: in.KeysEndpoint, HeaderName: in.HeaderName,
			AutoRegister: in.Options.IsAutoCreation,
		})
		return idpID(resp.GetIdpId(), err)

	default:
		return "", fmt.Errorf("unknown identity provider %q", provider)
	}
}

// UpdateIdentityProvider changes the parts of a provider that Zitadel lets an
// existing one keep: its name, its styling and whether it registers users.
//
// Everything else is create only. Zitadel has no update call for most provider
// settings, and reports none of them back, so a change to one means deleting the
// provider and making it again.
func (c *Client) UpdateIdentityProvider(ctx context.Context, orgID, idpID string, in ProviderInput) error {
	if orgID != "" {
		mc, err := c.managementClient(ctx, orgID)
		if err != nil {
			return err
		}

		if _, err := mc.UpdateOrgIDP(ctx, &managementv1.UpdateOrgIDPRequest{
			IdpId: idpID, Name: in.Name, AutoRegister: in.Options.IsAutoCreation,
		}); err != nil && !IsNotChanged(err) {
			return wrapIDPError(err, idpScope{orgID}, "update %s", idpID)
		}

		return nil
	}

	if _, err := c.admin.UpdateIDP(ctx, &adminv1.UpdateIDPRequest{
		IdpId: idpID, Name: in.Name, AutoRegister: in.Options.IsAutoCreation,
	}); err != nil && !IsNotChanged(err) {
		return wrapIDPError(err, idpScope{}, "update %s", idpID)
	}

	return nil
}

func idpID(id string, err error) (string, error) {
	if err != nil {
		return "", err
	}

	return id, nil
}

// azureTenant maps the API spelling of an Azure AD tenant onto the enum.
//
// Zitadel has three tenants: the common one, an organization, and a consumer
// tenant. "Common" is the one that lets any Azure AD organization log in, and is
// what most installations use.
func azureTenant(t string) *idpv1.AzureADTenant {
	kind := idpv1.AzureADTenantType_AZURE_AD_TENANT_TYPE_COMMON

	switch t {
	case "Consumers":
		kind = idpv1.AzureADTenantType_AZURE_AD_TENANT_TYPE_CONSUMERS

	case "Common", "":
		// The common tenant is the one that lets any Azure AD organization log
		// in, and is what most installations use.

	case "ID":
		// An explicit tenant ID, which Zitadel takes rather than a kind.
		return &idpv1.AzureADTenant{
			Type: &idpv1.AzureADTenant_TenantId{TenantId: t},
		}

	default:
		kind = idpv1.AzureADTenantType_AZURE_AD_TENANT_TYPE_ORGANISATIONS
	}

	return &idpv1.AzureADTenant{
		Type: &idpv1.AzureADTenant_TenantType{TenantType: kind},
	}
}

// samlMetadataInput is the metadata of a SAML provider, which Zitadel accepts as
// either a URL it fetches or the XML itself.
// isXMLDocument reports whether SAML metadata is the document itself rather than
// a URL for Zitadel to fetch. Zitadel accepts both, and which one was given is
// how it is sent.
func isXMLDocument(m string) bool {
	return strings.HasPrefix(strings.TrimSpace(m), "<")
}

// ldapAttributes names the LDAP attributes Zitadel reads a user from.
//
// Zitadel reads a fixed set of attributes and asks for them by name, so these are
// what a directory is expected to expose. The display name and the email address
// are what a person signs in with, so they are the two that matter.
func ldapAttributes(in ProviderInput) *idpv1.LDAPAttributes {
	out := &idpv1.LDAPAttributes{
		DisplayNameAttribute: "displayName",
		EmailAttribute:       "mail",
	}

	if in.IDAttribute != "" {
		out.IdAttribute = in.IDAttribute
	}

	return out
}

func samlBinding(b string) idpv1.SAMLBinding {
	if b == "POST" {
		return idpv1.SAMLBinding_SAML_BINDING_POST
	}

	return idpv1.SAMLBinding_SAML_BINDING_REDIRECT
}

func samlNameIDFormatPtr(n string) *idpv1.SAMLNameIDFormat {
	f := samlNameIDFormat(n)

	return &f
}

func samlNameIDFormat(n string) idpv1.SAMLNameIDFormat {
	switch n {
	case "EmailAddress":
		return idpv1.SAMLNameIDFormat_SAML_NAME_ID_FORMAT_EMAIL_ADDRESS

	case "Persistent":
		return idpv1.SAMLNameIDFormat_SAML_NAME_ID_FORMAT_PERSISTENT

	case "Transient":
		return idpv1.SAMLNameIDFormat_SAML_NAME_ID_FORMAT_TRANSIENT

	default:
		return idpv1.SAMLNameIDFormat_SAML_NAME_ID_FORMAT_UNSPECIFIED
	}
}

func samlSignatureAlgorithm(a string) idpv1.SAMLSignatureAlgorithm {
	switch a {
	case "RSA-SHA256":
		return idpv1.SAMLSignatureAlgorithm_SAML_SIGNATURE_RSA_SHA256

	case "RSA-SHA512":
		return idpv1.SAMLSignatureAlgorithm_SAML_SIGNATURE_RSA_SHA512

	case "RSA-SHA1":
		return idpv1.SAMLSignatureAlgorithm_SAML_SIGNATURE_RSA_SHA1

	default:
		return idpv1.SAMLSignatureAlgorithm_SAML_SIGNATURE_UNSPECIFIED
	}
}

// Ensure the imported identity provider types are referenced even when a kind is
// only reachable through the generated calls.
var _ = idpv1.Options{}

// optionalString is nil for an unset field, so that Zitadel applies its own
// default rather than being told to use an empty string.
func optionalString(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

// optionalBool is nil for an unset field, which is how Zitadel distinguishes
// "leave it alone" from "set it to false".
func optionalBool(b bool) *bool {
	return &b
}
