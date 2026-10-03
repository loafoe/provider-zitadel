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

// Package idp reconciles the twenty three Zitadel identity provider kinds.
//
// They are one package rather than twenty three because they are one resource
// with a different field set: the same lifecycle, the same two levels, and the
// same limitation about what Zitadel reads back. A driver therefore says three
// things - which managed resource it is, which Zitadel provider type it selects,
// and which fields it carries - and the lifecycle lives in
// internal/controller/common/idp.go.
//
// Two facts about Zitadel shape everything here:
//
//   - It has no generic "add a provider" call. Each provider type has its own,
//     so the type name is what selects the call.
//   - It reports a provider's name, state and auto-register flag, and its
//     configuration only for the OIDC and JWT kinds. Every other setting is
//     therefore applied once at creation and is neither compared nor changeable.
package idp

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// base is what every identity provider driver shares.
type base struct {
	kind     string
	provider string
	// org reports whether this kind belongs to an organization.
	org bool
	// observable reports whether Zitadel reads this kind of provider back.
	observable bool
}

// Kind names the managed resource, for errors and events.
func (b base) Kind() string { return b.kind }

// Provider names the Zitadel provider type, which is what selects the call.
func (b base) Provider() string { return b.provider }

// OrganizationScoped reports whether this provider belongs to an organization.
func (b base) OrganizationScoped() bool { return b.org }

// Observable reports whether Zitadel reads this kind of provider back.
//
// Only an OIDC and a JWT provider are; see the driver's interface for what that
// means for the other twenty one.
func (b base) Observable() bool { return b.observable }

// Report writes what Zitadel reported into the status.
//
// The same for every kind, because the fields Zitadel reports are the same for
// every kind: a name, a state, whether it registers users automatically, and a
// configuration for an OIDC or a JWT provider and nothing for any other.
func (base) Report(out *v1alpha1.IDPObservation, observed zitadel.IdentityProvider) {
	out.ID = observed.ID
	out.Name = common.StringPtr(observed.Name)
	out.State = common.StringPtr(observed.State)
	out.IsAutoCreation = common.BoolPtr(observed.AutoRegister)

	if o := observed.OIDC; o != nil {
		out.Issuer = common.StringPtr(o.Issuer)
		out.ClientID = common.StringPtr(o.ClientID)
		out.Scopes = o.Scopes
	}

	if j := observed.JWT; j != nil {
		out.Issuer = common.StringPtr(j.Issuer)
		out.JWTEndpoint = common.StringPtr(j.JWTEndpoint)
		out.KeysEndpoint = common.StringPtr(j.KeysEndpoint)
		out.HeaderName = common.StringPtr(j.HeaderName)
	}
}

// Equal reports whether the observed provider already matches the desired one.
//
// Only what Zitadel reports back is compared. A field it does not return cannot
// be drifted into existence by asking again, so comparing one would report drift
// that no amount of reconciling could ever settle - and the same rule already
// applies to a user's initial password and an action's script.
//
// The state is deliberately not compared: activating and deactivating are
// separate calls from editing a provider, so they are handled by their own step
// rather than here.
//
//nolint:gocyclo // One arm per readable kind, and an arm for none.
func (base) Equal(want zitadel.ProviderInput, observed zitadel.IdentityProvider) bool {
	if want.Name != observed.Name {
		return false
	}

	if want.Options.IsAutoCreation != observed.AutoRegister {
		return false
	}

	if o := observed.OIDC; o != nil {
		if want.OIDC == nil {
			return false
		}

		return want.OIDC.Issuer == o.Issuer &&
			want.OIDC.ClientID == o.ClientID &&
			common.EqualStringSlices(want.OIDC.Scopes, o.Scopes)
	}

	if j := observed.JWT; j != nil {
		if want.JWT == nil {
			return false
		}

		return want.JWT.JWTEndpoint == j.JWTEndpoint &&
			want.JWT.Issuer == j.Issuer &&
			want.JWT.KeysEndpoint == j.KeysEndpoint &&
			want.JWT.HeaderName == j.HeaderName
	}

	return true
}

// SettingsEqual compares the settings Zitadel reads back, which is what decides
// whether the settings of a kind can be updated in place.
//
//nolint:gocyclo // One arm per readable kind, and an arm for none.
func (base) SettingsEqual(want zitadel.IDPSettings, got zitadel.IDPSettings) bool {
	switch {
	case want.OIDC != nil || got.OIDC != nil:
		if want.OIDC == nil || got.OIDC == nil {
			return false
		}

		return want.OIDC.Issuer == got.OIDC.Issuer &&
			want.OIDC.ClientID == got.OIDC.ClientID &&
			common.EqualStringSlices(want.OIDC.Scopes, got.OIDC.Scopes)

	case want.JWT != nil || got.JWT != nil:
		if want.JWT == nil || got.JWT == nil {
			return false
		}

		return want.JWT.JWTEndpoint == got.JWT.JWTEndpoint &&
			want.JWT.Issuer == got.JWT.Issuer &&
			want.JWT.KeysEndpoint == got.JWT.KeysEndpoint &&
			want.JWT.HeaderName == got.JWT.HeaderName

	default:
		// Neither side has readable settings, which is every kind but these two.
		return true
	}
}

// The provider types, and the fields each carries beyond the common ones.
//
// Every provider takes a name, scopes, and the linking and creation rules; they
// are added by options() below, so a driver only carries what is its own.

// oidcDriver is a generic OpenID Connect provider.
type oidcDriver struct{ base }

// Observation returns the status this kind writes into.
func (oidcDriver) Observation(cr *v1alpha1.IDPOIDC) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

// Settings returns the settings Zitadel reads back, which for an OIDC provider
// it does.
func (oidcDriver) Settings(ctx context.Context, kube client.Client, cr *v1alpha1.IDPOIDC) zitadel.IDPSettings {
	fp := cr.Spec.ForProvider

	return zitadel.IDPSettings{OIDC: &zitadel.OIDCSettings{
		Issuer:       common.Deref(fp.Issuer),
		ClientID:     common.Deref(fp.ClientID),
		ClientSecret: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
		Scopes:       fp.Scopes,
	}}
}

// Desired reads the desired provider out of the managed resource.
func (oidcDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPOIDC) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		Issuer:           common.Deref(fp.Issuer),
		ClientID:         common.Deref(fp.ClientID),
		ClientSecret:     common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
		IsIDTokenMapping: common.DerefBool(fp.IsIDTokenMapping),
		UsePKCE:          common.DerefBool(fp.UsePKCE),
	}

	in = withCommon(in, cr, fp.Name, fp.Scopes)

	return in
}

// orgOIDCDriver is an organization's OpenID Connect provider.
type orgOIDCDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgOIDCDriver) Observation(cr *v1alpha1.OrgIDPOIDC) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgOIDCDriver) Settings(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPOIDC) zitadel.IDPSettings {
	fp := cr.Spec.ForProvider

	return zitadel.IDPSettings{OIDC: &zitadel.OIDCSettings{
		Issuer:       common.Deref(fp.Issuer),
		ClientID:     common.Deref(fp.ClientID),
		ClientSecret: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
		Scopes:       fp.Scopes,
	}}
}

func (orgOIDCDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPOIDC) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		Issuer:           common.Deref(fp.Issuer),
		ClientID:         common.Deref(fp.ClientID),
		ClientSecret:     common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
		IsIDTokenMapping: common.DerefBool(fp.IsIDTokenMapping),
		UsePKCE:          common.DerefBool(fp.UsePKCE),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// oauthDriver is a generic OAuth 2.0 provider, whose endpoints are given
// explicitly rather than discovered.
type oauthDriver struct{ base }

// Observation returns the status this kind writes into.
func (oauthDriver) Observation(cr *v1alpha1.IDPOAuth) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (oauthDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPOAuth) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		AuthorizationEndpoint: common.Deref(fp.AuthorizationEndpoint),
		TokenEndpoint:         common.Deref(fp.TokenEndpoint),
		UserEndpoint:          common.Deref(fp.UserEndpoint),
		ClientID:              common.Deref(fp.ClientID),
		ClientSecret:          common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
		IDAttribute:           common.Deref(fp.IDAttribute),
		UsePKCE:               common.DerefBool(fp.UsePKCE),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// orgOAuthDriver is an organization's generic OAuth provider.
type orgOAuthDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgOAuthDriver) Observation(cr *v1alpha1.OrgIDPOAuth) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgOAuthDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPOAuth) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		AuthorizationEndpoint: common.Deref(fp.AuthorizationEndpoint),
		TokenEndpoint:         common.Deref(fp.TokenEndpoint),
		UserEndpoint:          common.Deref(fp.UserEndpoint),
		ClientID:              common.Deref(fp.ClientID),
		ClientSecret:          common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
		IDAttribute:           common.Deref(fp.IDAttribute),
		UsePKCE:               common.DerefBool(fp.UsePKCE),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// jwtDriver is a provider that authenticates a machine by a JWT it presents,
// rather than a person logging in.
//
// It exists only at the organization level: Zitadel lets an organization accept
// one, not an instance offer it to all of them.
type jwtDriver struct{ base }

// Observation returns the status this kind writes into.
func (jwtDriver) Observation(cr *v1alpha1.OrgIDPJWT) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

// Settings returns the settings Zitadel reads back for a JWT provider.
func (jwtDriver) Settings(_ context.Context, _ client.Client, cr *v1alpha1.OrgIDPJWT) zitadel.IDPSettings {
	fp := cr.Spec.ForProvider

	return zitadel.IDPSettings{JWT: &zitadel.JWTSettings{
		JWTEndpoint:  common.Deref(fp.JWTEndpoint),
		Issuer:       common.Deref(fp.Issuer),
		KeysEndpoint: common.Deref(fp.KeysEndpoint),
		HeaderName:   common.Deref(fp.HeaderName),
		Audience:     common.Deref(fp.Audience),
	}}
}

func (jwtDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPJWT) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		JWTEndpoint:  common.Deref(fp.JWTEndpoint),
		KeysEndpoint: common.Deref(fp.KeysEndpoint),
		Issuer:       common.Deref(fp.Issuer),
		HeaderName:   common.Deref(fp.HeaderName),
		Audience:     common.Deref(fp.Audience),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// appleDriver lets users sign in with Apple.
type appleDriver struct{ base }

// Observation returns the status this kind writes into.
func (appleDriver) Observation(cr *v1alpha1.IDPApple) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (appleDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPApple) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:   common.Deref(fp.ClientID),
		TeamID:     common.Deref(fp.TeamID),
		KeyID:      common.Deref(fp.KeyID),
		PrivateKey: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.PrivateKey),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// orgAppleDriver is an organization's Apple provider.
type orgAppleDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgAppleDriver) Observation(cr *v1alpha1.OrgIDPApple) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgAppleDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPApple) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:   common.Deref(fp.ClientID),
		TeamID:     common.Deref(fp.TeamID),
		KeyID:      common.Deref(fp.KeyID),
		PrivateKey: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.PrivateKey),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// azureDriver lets users sign in with Microsoft Entra ID.
type azureDriver struct{ base }

// Observation returns the status this kind writes into.
func (azureDriver) Observation(cr *v1alpha1.IDPAzureAD) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (azureDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPAzureAD) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:      common.Deref(fp.ClientID),
		ClientSecret:  common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
		Tenant:        common.Value(fp.Tenant),
		EmailVerified: common.DerefBool(fp.EmailVerified),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// orgAzureDriver is an organization's Entra ID provider.
type orgAzureDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgAzureDriver) Observation(cr *v1alpha1.OrgIDPAzureAD) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgAzureDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPAzureAD) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:      common.Deref(fp.ClientID),
		ClientSecret:  common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
		Tenant:        common.Value(fp.Tenant),
		EmailVerified: common.DerefBool(fp.EmailVerified),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// githubDriver lets users sign in with GitHub.
type githubDriver struct{ base }

// Observation returns the status this kind writes into.
func (githubDriver) Observation(cr *v1alpha1.IDPGitHub) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (githubDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPGitHub) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:     common.Deref(fp.ClientID),
		ClientSecret: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// orgGitHubDriver is an organization's GitHub provider.
type orgGitHubDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgGitHubDriver) Observation(cr *v1alpha1.OrgIDPGitHub) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgGitHubDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPGitHub) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:     common.Deref(fp.ClientID),
		ClientSecret: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// githubESDriver lets users sign in with a self hosted GitHub Enterprise Server.
type githubESDriver struct{ base }

// Observation returns the status this kind writes into.
func (githubESDriver) Observation(cr *v1alpha1.IDPGitHubEnterpriseServer) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (githubESDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPGitHubEnterpriseServer) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:              common.Deref(fp.ClientID),
		ClientSecret:          common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
		AuthorizationEndpoint: common.Deref(fp.AuthorizationEndpoint),
		TokenEndpoint:         common.Deref(fp.TokenEndpoint),
		UserEndpoint:          common.Deref(fp.UserEndpoint),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// orgGitHubESDriver is an organization's GitHub Enterprise Server provider.
type orgGitHubESDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgGitHubESDriver) Observation(cr *v1alpha1.OrgIDPGitHubEnterpriseServer) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgGitHubESDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPGitHubEnterpriseServer) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:              common.Deref(fp.ClientID),
		ClientSecret:          common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
		AuthorizationEndpoint: common.Deref(fp.AuthorizationEndpoint),
		TokenEndpoint:         common.Deref(fp.TokenEndpoint),
		UserEndpoint:          common.Deref(fp.UserEndpoint),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// gitlabDriver lets users sign in with gitlab.com.
type gitlabDriver struct{ base }

// Observation returns the status this kind writes into.
func (gitlabDriver) Observation(cr *v1alpha1.IDPGitLab) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (gitlabDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPGitLab) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:     common.Deref(fp.ClientID),
		ClientSecret: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// orgGitLabDriver is an organization's GitLab provider.
type orgGitLabDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgGitLabDriver) Observation(cr *v1alpha1.OrgIDPGitLab) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgGitLabDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPGitLab) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:     common.Deref(fp.ClientID),
		ClientSecret: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// gitlabSHDriver lets users sign in with a self hosted GitLab.
type gitlabSHDriver struct{ base }

// Observation returns the status this kind writes into.
func (gitlabSHDriver) Observation(cr *v1alpha1.IDPGitLabSelfHosted) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (gitlabSHDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPGitLabSelfHosted) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		Issuer:       common.Deref(fp.Issuer),
		ClientID:     common.Deref(fp.ClientID),
		ClientSecret: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// orgGitLabSHDriver is an organization's self hosted GitLab provider.
type orgGitLabSHDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgGitLabSHDriver) Observation(cr *v1alpha1.OrgIDPGitLabSelfHosted) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgGitLabSHDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPGitLabSelfHosted) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		Issuer:       common.Deref(fp.Issuer),
		ClientID:     common.Deref(fp.ClientID),
		ClientSecret: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// googleDriver lets users sign in with Google.
type googleDriver struct{ base }

// Observation returns the status this kind writes into.
func (googleDriver) Observation(cr *v1alpha1.IDPGoogle) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (googleDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPGoogle) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:     common.Deref(fp.ClientID),
		ClientSecret: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// orgGoogleDriver is an organization's Google provider.
type orgGoogleDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgGoogleDriver) Observation(cr *v1alpha1.OrgIDPGoogle) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgGoogleDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPGoogle) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		ClientID:     common.Deref(fp.ClientID),
		ClientSecret: common.SecretValue(ctx, kube, cr.GetNamespace(), fp.ClientSecret),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// ldapDriver lets users sign in against a directory server.
type ldapDriver struct{ base }

// Observation returns the status this kind writes into.
func (ldapDriver) Observation(cr *v1alpha1.IDPLDAP) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (ldapDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPLDAP) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		Servers:           fp.Servers,
		StartTLS:          common.DerefBool(fp.StartTLS),
		BaseDN:            common.Deref(fp.BaseDN),
		BindDN:            common.Deref(fp.BindDN),
		BindPassword:      common.SecretValue(ctx, kube, cr.GetNamespace(), fp.BindPassword),
		UserBase:          common.Deref(fp.UserBase),
		UserObjectClasses: fp.UserObjectClasses,
		UserFilters:       fp.UserFilters,
		IDAttribute:       common.Deref(fp.IDAttribute),
		RootCA:            common.SecretValue(ctx, kube, cr.GetNamespace(), fp.RootCA),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// orgLDAPDriver is an organization's directory provider.
type orgLDAPDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgLDAPDriver) Observation(cr *v1alpha1.OrgIDPLDAP) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgLDAPDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPLDAP) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		Servers:           fp.Servers,
		StartTLS:          common.DerefBool(fp.StartTLS),
		BaseDN:            common.Deref(fp.BaseDN),
		BindDN:            common.Deref(fp.BindDN),
		BindPassword:      common.SecretValue(ctx, kube, cr.GetNamespace(), fp.BindPassword),
		UserBase:          common.Deref(fp.UserBase),
		UserObjectClasses: fp.UserObjectClasses,
		UserFilters:       fp.UserFilters,
		IDAttribute:       common.Deref(fp.IDAttribute),
		RootCA:            common.SecretValue(ctx, kube, cr.GetNamespace(), fp.RootCA),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// samlDriver lets users sign in with a SAML 2.0 identity provider.
type samlDriver struct{ base }

// Observation returns the status this kind writes into.
func (samlDriver) Observation(cr *v1alpha1.IDPSAML) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (samlDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.IDPSAML) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		Metadata:               common.Deref(fp.Metadata),
		Binding:                common.Value(fp.Binding),
		WithSignedRequest:      common.DerefBool(fp.WithSignedRequest),
		NameIDFormat:           common.Value(fp.NameIDFormat),
		SignatureAlgorithm:     common.Value(fp.SignatureAlgorithm),
		FederatedLogoutEnabled: common.DerefBool(fp.FederatedLogoutEnabled),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// orgSAMLDriver is an organization's SAML provider.
type orgSAMLDriver struct{ base }

// Observation returns the status this kind writes into.
func (orgSAMLDriver) Observation(cr *v1alpha1.OrgIDPSAML) *v1alpha1.IDPObservation {
	return &cr.Status.AtProvider
}

func (orgSAMLDriver) Desired(ctx context.Context, kube client.Client, cr *v1alpha1.OrgIDPSAML) zitadel.ProviderInput {
	fp := cr.Spec.ForProvider

	in := zitadel.ProviderInput{
		Metadata:               common.Deref(fp.Metadata),
		Binding:                common.Value(fp.Binding),
		WithSignedRequest:      common.DerefBool(fp.WithSignedRequest),
		NameIDFormat:           common.Value(fp.NameIDFormat),
		SignatureAlgorithm:     common.Value(fp.SignatureAlgorithm),
		FederatedLogoutEnabled: common.DerefBool(fp.FederatedLogoutEnabled),
	}

	return withCommon(in, cr, fp.Name, fp.Scopes)
}

// Setup adds a controller for every identity provider kind.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return setups(mgr, o)
}

// The fields every provider has, applied after the kind's own.
func withCommon(in zitadel.ProviderInput, cr commonFields, name *string, scopes []string) zitadel.ProviderInput {
	in.Name = common.Deref(name)
	in.Scopes = scopes

	// An unset state is left empty, which is what tells the harness to keep
	// Zitadel's default of active rather than fight over it.
	if s := cr.GetState(); s != nil {
		in.State = string(*s)
	}

	in.Options = zitadel.IDPOptions{
		IsLinkingAllowed:  common.DerefBool(cr.GetIsLinkingAllowed()),
		IsCreationAllowed: common.DerefBool(cr.GetIsCreationAllowed()),
		IsAutoCreation:    common.DerefBool(cr.GetIsAutoCreation()),
		IsAutoUpdate:      common.DerefBool(cr.GetIsAutoUpdate()),
		AutoLinking:       common.Value(cr.GetAutoLinking()),
	}

	return in
}

// keep the error package referenced by the drivers that validate a value.
var _ = errors.New

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (oauthDriver) Settings(context.Context, client.Client, *v1alpha1.IDPOAuth) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (appleDriver) Settings(context.Context, client.Client, *v1alpha1.IDPApple) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (azureDriver) Settings(context.Context, client.Client, *v1alpha1.IDPAzureAD) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (githubDriver) Settings(context.Context, client.Client, *v1alpha1.IDPGitHub) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (githubESDriver) Settings(context.Context, client.Client, *v1alpha1.IDPGitHubEnterpriseServer) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (gitlabDriver) Settings(context.Context, client.Client, *v1alpha1.IDPGitLab) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (gitlabSHDriver) Settings(context.Context, client.Client, *v1alpha1.IDPGitLabSelfHosted) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (googleDriver) Settings(context.Context, client.Client, *v1alpha1.IDPGoogle) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (ldapDriver) Settings(context.Context, client.Client, *v1alpha1.IDPLDAP) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (samlDriver) Settings(context.Context, client.Client, *v1alpha1.IDPSAML) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (orgOAuthDriver) Settings(context.Context, client.Client, *v1alpha1.OrgIDPOAuth) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (orgAppleDriver) Settings(context.Context, client.Client, *v1alpha1.OrgIDPApple) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (orgAzureDriver) Settings(context.Context, client.Client, *v1alpha1.OrgIDPAzureAD) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (orgGitHubDriver) Settings(context.Context, client.Client, *v1alpha1.OrgIDPGitHub) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (orgGitHubESDriver) Settings(context.Context, client.Client, *v1alpha1.OrgIDPGitHubEnterpriseServer) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (orgGitLabDriver) Settings(context.Context, client.Client, *v1alpha1.OrgIDPGitLab) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (orgGitLabSHDriver) Settings(context.Context, client.Client, *v1alpha1.OrgIDPGitLabSelfHosted) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (orgGoogleDriver) Settings(context.Context, client.Client, *v1alpha1.OrgIDPGoogle) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (orgLDAPDriver) Settings(context.Context, client.Client, *v1alpha1.OrgIDPLDAP) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Settings returns no readable settings.
//
// Zitadel reports no configuration for this kind of provider, so there is
// nothing it can be asked to update: its settings are applied once, at creation.
func (orgSAMLDriver) Settings(context.Context, client.Client, *v1alpha1.OrgIDPSAML) zitadel.IDPSettings {
	return zitadel.IDPSettings{}
}

// Each driver is checked against the harness interface here, so a method that
// goes missing is named once rather than at twenty three call sites.

var _ common.IDPDriver[*v1alpha1.IDPOIDC] = oidcDriver{}

var _ common.IDPDriver[*v1alpha1.IDPOAuth] = oauthDriver{}

var _ common.IDPDriver[*v1alpha1.IDPApple] = appleDriver{}

var _ common.IDPDriver[*v1alpha1.IDPAzureAD] = azureDriver{}

var _ common.IDPDriver[*v1alpha1.IDPGitHub] = githubDriver{}

var _ common.IDPDriver[*v1alpha1.IDPGitHubEnterpriseServer] = githubESDriver{}

var _ common.IDPDriver[*v1alpha1.IDPGitLab] = gitlabDriver{}

var _ common.IDPDriver[*v1alpha1.IDPGitLabSelfHosted] = gitlabSHDriver{}

var _ common.IDPDriver[*v1alpha1.IDPGoogle] = googleDriver{}

var _ common.IDPDriver[*v1alpha1.IDPLDAP] = ldapDriver{}

var _ common.IDPDriver[*v1alpha1.IDPSAML] = samlDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPOIDC] = orgOIDCDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPOAuth] = orgOAuthDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPJWT] = jwtDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPApple] = orgAppleDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPAzureAD] = orgAzureDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPGitHub] = orgGitHubDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPGitHubEnterpriseServer] = orgGitHubESDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPGitLab] = orgGitLabDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPGitLabSelfHosted] = orgGitLabSHDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPGoogle] = orgGoogleDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPLDAP] = orgLDAPDriver{}

var _ common.IDPDriver[*v1alpha1.OrgIDPSAML] = orgSAMLDriver{}
