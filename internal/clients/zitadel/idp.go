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
	"errors"
	"fmt"

	adminv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	idpv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/idp"
	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
)

// Identity providers
//
// Zitadel has two levels of identity provider, and the v1 APIs give each a
// parallel set of calls: the admin API manages the ones available to every
// organization, and the management API the ones an organization owns. The shape
// is the same at both, so one type and one set of functions cover both and a
// scope decides which API is spoken to.
//
// The per provider settings differ, which is why a provider is named rather than
// inferred: Zitadel has no call that takes a generic provider, so each one has
// its own request type and its own field set.
//
// What Zitadel reads back is the real limitation. An IdP is returned with its
// name, styling, state and auto-register flag, and its configuration only for
// the OIDC and JWT kinds. Every other provider's settings are therefore applied
// at creation and are not drift detected - the same rule that already applies to
// a user's initial password and an action's script.

// IDPOptions are the linking and creation rules that apply to every provider.
type IDPOptions struct {
	IsLinkingAllowed  bool
	IsCreationAllowed bool
	IsAutoCreation    bool
	IsAutoUpdate      bool
	AutoLinking       string
}

// IdentityProvider is an identity provider as Zitadel reports it.
type IdentityProvider struct {
	ID   string
	Name string
	// State is `Active` or `Inactive`.
	State string

	// AutoRegister is whether a user is created without asking on first login.
	// It is the one provider option Zitadel reports; the rest - linking,
	// creation and auto-update - it does not, so they are applied once and are
	// never compared.
	AutoRegister bool

	// OIDC is set when Zitadel returned the configuration of an OIDC provider,
	// and JWT when it returned one of a JWT provider.
	OIDC *OIDCSettings
	JWT  *JWTSettings
}

// IDPSettings are the settings of an identity provider that Zitadel can read
// back, and so can be updated in place.
//
// Only an OIDC or a JWT provider has one of these. For every other kind the
// settings are applied once, at creation.
type IDPSettings struct {
	OIDC *OIDCSettings
	JWT  *JWTSettings
}

// OIDCSettings are the settings of an OIDC provider, as read back by Zitadel.
type OIDCSettings struct {
	Issuer   string
	ClientID string
	Scopes   []string

	// ClientSecret is write only: Zitadel never returns it, so it is applied but
	// never compared.
	ClientSecret string
}

// JWTSettings are the settings Zitadel reads back for a JWT provider.
type JWTSettings struct {
	JWTEndpoint  string
	Issuer       string
	KeysEndpoint string
	HeaderName   string
	Audience     string

	// ClientSecret is write only, like the OIDC one.
	ClientSecret string
}

// idpScope says which of the two levels of provider a call is about.
type idpScope struct {
	// orgID is empty for an instance wide provider.
	orgID string
}

func (s idpScope) org() bool { return s.orgID != "" }

func (s idpScope) String() string {
	if s.org() {
		return "organization " + s.orgID
	}

	return "the instance"
}

// GetIdentityProvider returns one identity provider.
//
// The by-ID read is tried first, because it is precise. For a kind whose
// configuration Zitadel does not expose it answers "Identity Provider
// Configuration doesn't exist" even though the provider is there - the same
// read-side gap that makes those settings unreadable in the first place. So a
// not-found is retried against the list, which reports every provider
// regardless of its kind.
func (c *Client) GetIdentityProvider(ctx context.Context, orgID, idpID string) (*IdentityProvider, error) {
	p, err := c.getIdentityProviderByID(ctx, orgID, idpID)
	if err == nil {
		return p, nil
	}

	if !IsNotFound(err) {
		return nil, err
	}

	all, lerr := c.ListIdentityProviders(ctx, orgID)
	if lerr != nil {
		// The provider is reported as gone only when the list agrees.
		return nil, err
	}

	for i := range all {
		if all[i].ID == idpID {
			return &all[i], nil
		}
	}

	return nil, ErrNotFound
}

func (c *Client) getIdentityProviderByID(ctx context.Context, orgID, idpID string) (*IdentityProvider, error) {
	var p *idpv1.IDP

	if orgID != "" {
		mc, err := c.managementClient(ctx, orgID)
		if err != nil {
			return nil, err
		}

		resp, err := mc.GetOrgIDPByID(ctx, &managementv1.GetOrgIDPByIDRequest{Id: idpID})
		if err != nil {
			return nil, wrapIDPError(err, idpScope{orgID}, "get %s", idpID)
		}

		p = resp.GetIdp()
	} else {
		resp, err := c.admin.GetIDPByID(ctx, &adminv1.GetIDPByIDRequest{Id: idpID})
		if err != nil {
			return nil, wrapIDPError(err, idpScope{}, "get %s", idpID)
		}

		p = resp.GetIdp()
	}

	if p == nil {
		return nil, ErrNotFound
	}

	return fromProtoIDP(p), nil
}

// ListIdentityProviders returns every identity provider of a scope.
func (c *Client) ListIdentityProviders(ctx context.Context, orgID string) ([]IdentityProvider, error) {
	var raw []*idpv1.IDP

	if orgID != "" {
		mc, err := c.managementClient(ctx, orgID)
		if err != nil {
			return nil, err
		}

		resp, err := mc.ListOrgIDPs(ctx, &managementv1.ListOrgIDPsRequest{})
		if err != nil {
			return nil, wrapIDPError(err, idpScope{orgID}, "list")
		}

		raw = resp.GetResult()
	} else {
		resp, err := c.admin.ListIDPs(ctx, &adminv1.ListIDPsRequest{})
		if err != nil {
			return nil, wrapIDPError(err, idpScope{}, "list")
		}

		raw = resp.GetResult()
	}

	out := make([]IdentityProvider, 0, len(raw))
	for _, p := range raw {
		if p == nil {
			continue
		}

		out = append(out, *fromProtoIDP(p))
	}

	return out, nil
}

// SetIdentityProviderState activates or deactivates an identity provider.
//
// A deactivated provider keeps its settings and its user links but is not offered
// on the login page, which is what scaling one back should look like.
func (c *Client) SetIdentityProviderState(ctx context.Context, orgID, idpID, state string) error {
	var err error

	if orgID != "" {
		mc, merr := c.managementClient(ctx, orgID)
		if merr != nil {
			return merr
		}

		if state == "Active" {
			_, err = mc.ReactivateOrgIDP(ctx, &managementv1.ReactivateOrgIDPRequest{IdpId: idpID})
		} else {
			_, err = mc.DeactivateOrgIDP(ctx, &managementv1.DeactivateOrgIDPRequest{IdpId: idpID})
		}
	} else {
		if state == "Active" {
			_, err = c.admin.ReactivateIDP(ctx, &adminv1.ReactivateIDPRequest{IdpId: idpID})
		} else {
			_, err = c.admin.DeactivateIDP(ctx, &adminv1.DeactivateIDPRequest{IdpId: idpID})
		}
	}

	if err != nil && !IsNotFound(err) && !IsNotChanged(err) && !IsAlreadyExists(err) {
		return wrapIDPError(err, idpScope{orgID}, "set %s to %s", idpID, state)
	}

	return nil
}

// UpdateIdentityProviderSettings changes the settings Zitadel reads back.
//
// Only an OIDC or JWT provider can be updated this way: every other kind's
// settings are write once, at creation, and Zitadel reports no configuration for
// them at all.
func (c *Client) UpdateIdentityProviderSettings(ctx context.Context, orgID, idpID string, in IDPSettings) error {
	if orgID != "" {
		return c.updateOrgIdentityProviderSettings(ctx, orgID, idpID, in)
	}

	return c.updateInstanceIdentityProviderSettings(ctx, idpID, in)
}

// The settings are updated through a different API at each level, and only an
// OIDC or a JWT provider has settings Zitadel reads back at all, so each level
// and each kind is a small explicit step.

func (c *Client) updateInstanceIdentityProviderSettings(ctx context.Context, idpID string, in IDPSettings) error {
	scope := idpScope{}

	switch {
	case in.OIDC != nil:
		if _, err := c.admin.UpdateIDPOIDCConfig(ctx, &adminv1.UpdateIDPOIDCConfigRequest{
			IdpId:        idpID,
			ClientId:     in.OIDC.ClientID,
			ClientSecret: in.OIDC.ClientSecret,
			Issuer:       in.OIDC.Issuer,
			Scopes:       in.OIDC.Scopes,
		}); err != nil && !IsNotChanged(err) {
			return wrapIDPError(err, scope, "update the OIDC settings of %s", idpID)
		}

	case in.JWT != nil:
		if _, err := c.admin.UpdateIDPJWTConfig(ctx, &adminv1.UpdateIDPJWTConfigRequest{
			IdpId:        idpID,
			JwtEndpoint:  in.JWT.JWTEndpoint,
			Issuer:       in.JWT.Issuer,
			KeysEndpoint: in.JWT.KeysEndpoint,
			HeaderName:   in.JWT.HeaderName,
		}); err != nil && !IsNotChanged(err) {
			return wrapIDPError(err, scope, "update the JWT settings of %s", idpID)
		}

	default:
		return errNoReadableIDPSettings
	}

	return nil
}

func (c *Client) updateOrgIdentityProviderSettings(ctx context.Context, orgID, idpID string, in IDPSettings) error {
	scope := idpScope{orgID: orgID}

	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	switch {
	case in.OIDC != nil:
		if _, err := mc.UpdateOrgIDPOIDCConfig(ctx, &managementv1.UpdateOrgIDPOIDCConfigRequest{
			IdpId:        idpID,
			ClientId:     in.OIDC.ClientID,
			ClientSecret: in.OIDC.ClientSecret,
			Issuer:       in.OIDC.Issuer,
			Scopes:       in.OIDC.Scopes,
		}); err != nil && !IsNotChanged(err) {
			return wrapIDPError(err, scope, "update the OIDC settings of %s", idpID)
		}

	case in.JWT != nil:
		if _, err := mc.UpdateOrgIDPJWTConfig(ctx, &managementv1.UpdateOrgIDPJWTConfigRequest{
			IdpId:        idpID,
			JwtEndpoint:  in.JWT.JWTEndpoint,
			Issuer:       in.JWT.Issuer,
			KeysEndpoint: in.JWT.KeysEndpoint,
			HeaderName:   in.JWT.HeaderName,
		}); err != nil && !IsNotChanged(err) {
			return wrapIDPError(err, scope, "update the JWT settings of %s", idpID)
		}

	default:
		return errNoReadableIDPSettings
	}

	return nil
}

// errNoReadableIDPSettings is what an operator sees when they change a setting
// Zitadel cannot read back, and so cannot change in place.
var errNoReadableIDPSettings = errors.New(
	"zitadel does not read back the settings of this kind of identity provider, so they cannot be updated; delete and recreate it to change them")

// RemoveIdentityProvider deletes an identity provider.
//
// Zitadel refuses to remove one that is still offered on a login page, so the
// provider has to be unbound first - the same rule that applies to an action
// that something still calls.
func (c *Client) RemoveIdentityProvider(ctx context.Context, orgID, idpID string) error {
	var err error

	if orgID != "" {
		mc, merr := c.managementClient(ctx, orgID)
		if merr != nil {
			return merr
		}

		_, err = mc.RemoveOrgIDP(ctx, &managementv1.RemoveOrgIDPRequest{IdpId: idpID})
	} else {
		_, err = c.admin.RemoveIDP(ctx, &adminv1.RemoveIDPRequest{IdpId: idpID})
	}

	if err != nil {
		if IsNotFound(err) {
			return nil
		}

		return wrapIDPError(err, idpScope{orgID}, "remove %s", idpID)
	}

	return nil
}

func fromProtoIDP(p *idpv1.IDP) *IdentityProvider {
	out := &IdentityProvider{
		ID:           p.GetId(),
		Name:         p.GetName(),
		State:        idpStateFromProto(p.GetState()),
		AutoRegister: p.GetAutoRegister(),
	}

	// Zitadel returns the configuration of an OIDC or a JWT provider and
	// nothing at all for any other kind.
	if o := p.GetOidcConfig(); o != nil {
		out.OIDC = &OIDCSettings{
			Issuer:   o.GetIssuer(),
			ClientID: o.GetClientId(),
			Scopes:   o.GetScopes(),
		}
	}

	if j := p.GetJwtConfig(); j != nil {
		out.JWT = &JWTSettings{
			JWTEndpoint:  j.GetJwtEndpoint(),
			Issuer:       j.GetIssuer(),
			KeysEndpoint: j.GetKeysEndpoint(),
			HeaderName:   j.GetHeaderName(),
			Audience:     j.GetAudience(),
		}
	}

	return out
}

func idpStateFromProto(s idpv1.IDPState) string {
	if s == idpv1.IDPState_IDP_STATE_ACTIVE {
		return "Active"
	}

	return "Inactive"
}

func toProtoIDPOptions(o IDPOptions) idpv1.Options {
	linking := idpv1.AutoLinkingOption_AUTO_LINKING_OPTION_UNSPECIFIED

	switch o.AutoLinking {
	case "Email":
		linking = idpv1.AutoLinkingOption_AUTO_LINKING_OPTION_EMAIL

	case "Username":
		linking = idpv1.AutoLinkingOption_AUTO_LINKING_OPTION_USERNAME
	}

	return idpv1.Options{
		IsLinkingAllowed:  o.IsLinkingAllowed,
		IsCreationAllowed: o.IsCreationAllowed,
		IsAutoCreation:    o.IsAutoCreation,
		IsAutoUpdate:      o.IsAutoUpdate,
		AutoLinking:       linking,
	}
}

// wrapIDPError says which level of provider a failure is about, because "cannot
// update the identity provider" is useless when an instance holds several.
func wrapIDPError(err error, scope idpScope, format string, args ...any) error {
	return fmt.Errorf("cannot %s of %s: %w", fmt.Sprintf(format, args...), scope, err)
}
