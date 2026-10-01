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

package oidcapplication

import (
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/pkg/errors"
	apiv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// defaults holds the effective values of the optional OIDC fields. It mirrors
// the defaults applied by the Zitadel API so that drift detection compares
// like with like.
type defaults struct {
	ResponseTypes   []apiv2.OIDCResponseType
	GrantTypes      []apiv2.OIDCGrantType
	ApplicationType apiv2.OIDCApplicationType
	AuthMethodType  apiv2.OIDCAuthMethodType
	Version         apiv2.OIDCVersion
	AccessTokenType apiv2.OIDCTokenType
}

// resolveDefaults computes the effective values of the optional OIDC fields of
// fp. Values that fail validation are left at their defaults here; the
// validation errors are surfaced when the resource is created or updated.
//
//nolint:gocyclo // flat translation of the optional forProvider fields
func resolveDefaults(fp v1alpha1.OIDCApplicationParameters) defaults {
	d := defaults{
		ResponseTypes: []apiv2.OIDCResponseType{apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE},
		GrantTypes: []apiv2.OIDCGrantType{
			apiv2.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE,
			apiv2.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN,
		},
		ApplicationType: apiv2.OIDCApplicationType_OIDC_APP_TYPE_WEB,
		AuthMethodType:  apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC,
		Version:         apiv2.OIDCVersion_OIDC_VERSION_1_0,
		AccessTokenType: apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER,
	}

	if len(fp.ResponseTypes) > 0 {
		if v, err := responseTypes(fp.ResponseTypes); err == nil {
			d.ResponseTypes = v
		}
	}

	if len(fp.GrantTypes) > 0 {
		if v, err := grantTypes(fp.GrantTypes); err == nil {
			d.GrantTypes = v
		}
	}

	if fp.ApplicationType != nil {
		if v, err := zitadel.OIDCApplicationTypeToProto(string(*fp.ApplicationType)); err == nil {
			d.ApplicationType = v
		}
	}

	if fp.AuthMethodType != nil {
		if v, err := zitadel.OIDCAuthMethodTypeToProto(string(*fp.AuthMethodType)); err == nil {
			d.AuthMethodType = v
		}
	}

	if fp.AccessTokenType != nil {
		if v, err := zitadel.OIDCTokenTypeToProto(string(*fp.AccessTokenType)); err == nil {
			d.AccessTokenType = v
		}
	}

	if fp.Version != nil {
		if v, err := zitadel.OIDCVersionToProto(string(*fp.Version)); err == nil {
			d.Version = v
		}
	}

	return d
}

// responseTypes converts the API response types into their protobuf
// representation.
func responseTypes(in []v1alpha1.OIDCResponseType) ([]apiv2.OIDCResponseType, error) {
	out := make([]apiv2.OIDCResponseType, 0, len(in))
	for _, t := range in {
		v, err := zitadel.OIDCResponseTypeToProto(string(t))
		if err != nil {
			return nil, err
		}

		out = append(out, v)
	}

	return out, nil
}

// grantTypes converts the API grant types into their protobuf representation.
func grantTypes(in []v1alpha1.OIDCGrantType) ([]apiv2.OIDCGrantType, error) {
	out := make([]apiv2.OIDCGrantType, 0, len(in))
	for _, t := range in {
		v, err := zitadel.OIDCGrantTypeToProto(string(t))
		if err != nil {
			return nil, err
		}

		out = append(out, v)
	}

	return out, nil
}

// responseTypesFromProto converts protobuf response types back into the API
// representation.
func responseTypesFromProto(in []apiv2.OIDCResponseType) []v1alpha1.OIDCResponseType {
	out := make([]v1alpha1.OIDCResponseType, 0, len(in))
	for _, t := range in {
		if v := zitadel.OIDCResponseTypeFromProto(t); v != "" {
			out = append(out, v1alpha1.OIDCResponseType(v))
		}
	}

	return out
}

// grantTypesFromProto converts protobuf grant types back into the API
// representation.
func grantTypesFromProto(in []apiv2.OIDCGrantType) []v1alpha1.OIDCGrantType {
	out := make([]v1alpha1.OIDCGrantType, 0, len(in))
	for _, t := range in {
		if v := zitadel.OIDCGrantTypeFromProto(t); v != "" {
			out = append(out, v1alpha1.OIDCGrantType(v))
		}
	}

	return out
}

// equalResponseTypes compares desired response types against the ones Zitadel
// reported.
func equalResponseTypes(desired []v1alpha1.OIDCResponseType, actual []apiv2.OIDCResponseType) bool {
	want := responseTypesFromProto(actual)
	if len(want) == 0 && len(desired) == 0 {
		return true
	}

	if len(want) != len(desired) {
		return false
	}

	for i := range desired {
		if desired[i] != want[i] {
			return false
		}
	}

	return true
}

// equalGrantTypes compares desired grant types against the ones Zitadel
// reported.
func equalGrantTypes(desired []v1alpha1.OIDCGrantType, actual []apiv2.OIDCGrantType) bool {
	want := grantTypesFromProto(actual)
	if len(want) == 0 && len(desired) == 0 {
		return true
	}

	if len(want) != len(desired) {
		return false
	}

	for i := range desired {
		if desired[i] != want[i] {
			return false
		}
	}

	return true
}

// createInput builds the client input used to create the application.
func createInput(cr *v1alpha1.OIDCApplication, projectID string) (zitadel.CreateOIDCApplicationInput, error) {
	fp := cr.Spec.ForProvider
	d := resolveDefaults(fp)

	in := zitadel.CreateOIDCApplicationInput{
		ProjectID:              projectID,
		ApplicationID:          common.Deref(fp.ID),
		Name:                   fp.Name,
		RedirectURIs:           fp.RedirectURIs,
		PostLogoutRedirectURIs: fp.PostLogoutRedirectURIs,
		ResponseTypes:          d.ResponseTypes,
		GrantTypes:             d.GrantTypes,
		ApplicationType:        d.ApplicationType,
		AuthMethodType:         d.AuthMethodType,
		Version:                d.Version,
		AccessTokenType:        d.AccessTokenType,
		AdditionalOrigins:      fp.AdditionalOrigins,
		BackChannelLogoutURI:   common.Deref(fp.BackChannelLogoutURI),
	}

	if fp.DevelopmentMode != nil {
		in.DevelopmentMode = *fp.DevelopmentMode
	}
	if fp.AccessTokenRoleAssertion != nil {
		in.AccessTokenRoleAssertion = *fp.AccessTokenRoleAssertion
	}
	if fp.IDTokenRoleAssertion != nil {
		in.IDTokenRoleAssertion = *fp.IDTokenRoleAssertion
	}
	if fp.IDTokenUserinfoAssertion != nil {
		in.IDTokenUserinfoAssertion = *fp.IDTokenUserinfoAssertion
	}
	if fp.SkipNativeAppSuccessPage != nil {
		in.SkipNativeAppSuccessPage = *fp.SkipNativeAppSuccessPage
	}
	if fp.ClockSkew != nil && *fp.ClockSkew != "" {
		skew, err := zitadel.ParseDuration(*fp.ClockSkew)
		if err != nil {
			return zitadel.CreateOIDCApplicationInput{}, err
		}
		in.ClockSkew = skew
	}

	return in, nil
}

// updateInput builds the client input used to update the application. It uses
// the v1 management API shape, which is the only Zitadel API that reliably
// applies OIDC configuration changes (see internal/clients/zitadel/management.go).
//
//nolint:gocyclo // flat translation of the optional forProvider fields
func updateInput(cr *v1alpha1.OIDCApplication, projectID string) (zitadel.UpdateOIDCAppInput, error) {
	fp := cr.Spec.ForProvider

	in := zitadel.UpdateOIDCAppInput{
		ProjectID:              projectID,
		Name:                   fp.Name,
		RedirectURIs:           fp.RedirectURIs,
		PostLogoutRedirectURIs: fp.PostLogoutRedirectURIs,
		AdditionalOrigins:      fp.AdditionalOrigins,
		BackChannelLogoutURI:   common.Deref(fp.BackChannelLogoutURI),
	}

	if len(fp.ResponseTypes) > 0 {
		for _, t := range fp.ResponseTypes {
			v, err := zitadel.ResponseTypeToV1(string(t))
			if err != nil {
				return zitadel.UpdateOIDCAppInput{}, err
			}
			in.ResponseTypes = append(in.ResponseTypes, v)
		}
	}

	if len(fp.GrantTypes) > 0 {
		for _, t := range fp.GrantTypes {
			v, err := zitadel.GrantTypeToV1(string(t))
			if err != nil {
				return zitadel.UpdateOIDCAppInput{}, err
			}
			in.GrantTypes = append(in.GrantTypes, v)
		}
	}

	appType, err := zitadel.ApplicationTypeToV1(common.Value(fp.ApplicationType))
	if err != nil {
		return zitadel.UpdateOIDCAppInput{}, errors.Wrap(err, "cannot set the application type")
	}
	in.ApplicationType = appType

	authType, err := zitadel.AuthMethodTypeToV1(common.Value(fp.AuthMethodType))
	if err != nil {
		return zitadel.UpdateOIDCAppInput{}, errors.Wrap(err, "cannot set the auth method type")
	}
	in.AuthMethodType = authType

	tokenType, err := zitadel.TokenTypeToV1(common.Value(fp.AccessTokenType))
	if err != nil {
		return zitadel.UpdateOIDCAppInput{}, errors.Wrap(err, "cannot set the access token type")
	}
	in.AccessTokenType = tokenType

	if fp.DevelopmentMode != nil {
		in.DevelopmentMode = *fp.DevelopmentMode
	}
	if fp.AccessTokenRoleAssertion != nil {
		in.AccessTokenRoleAssertion = *fp.AccessTokenRoleAssertion
	}
	if fp.IDTokenRoleAssertion != nil {
		in.IDTokenRoleAssertion = *fp.IDTokenRoleAssertion
	}
	if fp.IDTokenUserinfoAssertion != nil {
		in.IDTokenUserinfoAssertion = *fp.IDTokenUserinfoAssertion
	}
	if fp.SkipNativeAppSuccessPage != nil {
		in.SkipNativeAppSuccessPage = *fp.SkipNativeAppSuccessPage
	}

	if fp.ClockSkew != nil && *fp.ClockSkew != "" {
		skew, err := zitadel.ParseDuration(*fp.ClockSkew)
		if err != nil {
			return zitadel.UpdateOIDCAppInput{}, err
		}
		in.ClockSkew = skew
	}

	return in, nil
}

// updateStatus copies the observed state of app into the status of cr.
func updateStatus(cr *v1alpha1.OIDCApplication, app *zitadel.OIDCApplication) {
	cr.Status.AtProvider.ID = common.StringPtr(app.ApplicationID)
	cr.Status.AtProvider.ProjectID = common.StringPtr(app.ProjectID)
	cr.Status.AtProvider.Name = common.StringPtr(app.Name)
	cr.Status.AtProvider.ClientID = common.StringPtr(app.ClientID)
	cr.Status.AtProvider.RedirectURIs = app.RedirectURIs
	cr.Status.AtProvider.PostLogoutRedirectURIs = app.PostLogoutRedirectURIs
	cr.Status.AtProvider.ResponseTypes = responseTypesFromProto(app.ResponseTypes)
	cr.Status.AtProvider.GrantTypes = grantTypesFromProto(app.GrantTypes)
	cr.Status.AtProvider.AllowedOrigins = app.AllowedOrigins
	cr.Status.AtProvider.NonCompliant = common.BoolPtr(app.NonCompliant)

	if app.State != "" {
		state := v1alpha1.ApplicationState(app.State)
		cr.Status.AtProvider.State = &state
	}

	appType := v1alpha1.OIDCApplicationType(zitadel.OIDCApplicationTypeFromProto(app.ApplicationType))
	cr.Status.AtProvider.ApplicationType = &appType

	authType := v1alpha1.OIDCAuthMethodType(zitadel.OIDCAuthMethodTypeFromProto(app.AuthMethodType))
	cr.Status.AtProvider.AuthMethodType = &authType

	tokenType := v1alpha1.OIDCTokenType(zitadel.OIDCTokenTypeFromProto(app.AccessTokenType))
	cr.Status.AtProvider.AccessTokenType = &tokenType

	cr.Status.AtProvider.CreationDate = common.ParseTime(app.CreationDate)
	cr.Status.AtProvider.ChangeDate = common.ParseTime(app.ChangeDate)
}

// connectionDetails returns the connection details published for the
// application. The client secret is not part of it: Zitadel only returns the
// secret at creation or regeneration time, so it is written to the connection
// secret by Create and Update respectively.
func connectionDetails(app *zitadel.OIDCApplication) managed.ConnectionDetails {
	return managed.ConnectionDetails{
		v1alpha1.ConnectionKeyClientID: []byte(app.ClientID),
	}
}

// isUpToDate reports whether the remote application matches the desired state
// described by cr. The second return value signals that the resource is in
// sync but that a client secret regeneration was requested.
func isUpToDate(cr *v1alpha1.OIDCApplication, app *zitadel.OIDCApplication) (upToDate, regenerate bool) { //nolint:gocyclo // a flat list of field comparisons
	fp := cr.Spec.ForProvider
	d := resolveDefaults(fp)

	upToDate = true

	if fp.Name != app.Name {
		upToDate = false
	}

	if !common.EqualStringSlices(fp.RedirectURIs, app.RedirectURIs) {
		upToDate = false
	}

	if !common.EqualStringSlices(fp.PostLogoutRedirectURIs, app.PostLogoutRedirectURIs) {
		upToDate = false
	}

	if len(fp.ResponseTypes) > 0 && !equalResponseTypes(fp.ResponseTypes, app.ResponseTypes) {
		upToDate = false
	}

	if len(fp.GrantTypes) > 0 && !equalGrantTypes(fp.GrantTypes, app.GrantTypes) {
		upToDate = false
	}

	if d.ApplicationType != app.ApplicationType {
		upToDate = false
	}

	if d.AuthMethodType != app.AuthMethodType {
		upToDate = false
	}

	if d.AccessTokenType != app.AccessTokenType {
		upToDate = false
	}

	if fp.DevelopmentMode != nil && *fp.DevelopmentMode != app.DevelopmentMode {
		upToDate = false
	}

	if fp.AccessTokenRoleAssertion != nil && *fp.AccessTokenRoleAssertion != app.AccessTokenRoleAssertion {
		upToDate = false
	}

	if fp.IDTokenRoleAssertion != nil && *fp.IDTokenRoleAssertion != app.IDTokenRoleAssertion {
		upToDate = false
	}

	if fp.IDTokenUserinfoAssertion != nil && *fp.IDTokenUserinfoAssertion != app.IDTokenUserinfoAssertion {
		upToDate = false
	}

	if fp.SkipNativeAppSuccessPage != nil && *fp.SkipNativeAppSuccessPage != app.SkipNativeAppSuccessPage {
		upToDate = false
	}

	if !common.EqualStringSlices(fp.AdditionalOrigins, app.AdditionalOrigins) {
		upToDate = false
	}

	if fp.BackChannelLogoutURI != nil && *fp.BackChannelLogoutURI != app.BackChannelLogoutURI {
		upToDate = false
	}

	if fp.ClockSkew != nil && !equalDuration(*fp.ClockSkew, app.ClockSkew) {
		upToDate = false
	}

	if fp.State != nil && string(*fp.State) != app.State {
		upToDate = false
	}

	return upToDate, fp.RegenerateClientSecret != nil && *fp.RegenerateClientSecret
}

// equalDuration compares two duration strings, tolerating the difference
// between "300s" and "5m" that Zitadel's normalisation may introduce.
func equalDuration(desired, actual string) bool {
	if desired == actual {
		return true
	}

	d, err := time.ParseDuration(desired)
	if err != nil {
		return false
	}

	a, err := time.ParseDuration(actual)
	if err != nil {
		return false
	}

	return d == a
}
