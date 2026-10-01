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
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/google/go-cmp/cmp"
	apiv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

const (
	testAppName = "billing-web"
)

func TestResolveDefaults(t *testing.T) {
	cases := map[string]struct {
		reason string
		fp     v1alpha1.OIDCApplicationParameters
		want   defaults
	}{
		"Empty": {
			reason: "Unset fields fall back to the Zitadel defaults",
			want: defaults{
				ResponseTypes:   []apiv2.OIDCResponseType{apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE},
				GrantTypes:      []apiv2.OIDCGrantType{apiv2.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE, apiv2.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN},
				ApplicationType: apiv2.OIDCApplicationType_OIDC_APP_TYPE_WEB,
				AuthMethodType:  apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC,
				Version:         apiv2.OIDCVersion_OIDC_VERSION_1_0,
				AccessTokenType: apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER,
			},
		},
		"Explicit": {
			reason: "Explicit values win over the defaults",
			fp: v1alpha1.OIDCApplicationParameters{
				ResponseTypes:   []v1alpha1.OIDCResponseType{v1alpha1.OIDCResponseTypeCode},
				GrantTypes:      []v1alpha1.OIDCGrantType{v1alpha1.OIDCGrantTypeImplicit},
				ApplicationType: ptr(v1alpha1.OIDCApplicationTypeNative),
				AuthMethodType:  ptr(v1alpha1.OIDCAuthMethodTypeNone),
				AccessTokenType: ptr(v1alpha1.OIDCTokenTypeJwt),
			},
			want: defaults{
				ResponseTypes:   []apiv2.OIDCResponseType{apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE},
				GrantTypes:      []apiv2.OIDCGrantType{apiv2.OIDCGrantType_OIDC_GRANT_TYPE_IMPLICIT},
				ApplicationType: apiv2.OIDCApplicationType_OIDC_APP_TYPE_NATIVE,
				AuthMethodType:  apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_NONE,
				Version:         apiv2.OIDCVersion_OIDC_VERSION_1_0,
				AccessTokenType: apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_JWT,
			},
		},
		"InvalidFallsBackToDefault": {
			reason: "An invalid value falls back to the default rather than panicking",
			fp: v1alpha1.OIDCApplicationParameters{
				ApplicationType: ptr(v1alpha1.OIDCApplicationType("Bogus")),
			},
			want: defaults{
				ResponseTypes:   []apiv2.OIDCResponseType{apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE},
				GrantTypes:      []apiv2.OIDCGrantType{apiv2.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE, apiv2.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN},
				ApplicationType: apiv2.OIDCApplicationType_OIDC_APP_TYPE_WEB,
				AuthMethodType:  apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC,
				Version:         apiv2.OIDCVersion_OIDC_VERSION_1_0,
				AccessTokenType: apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := resolveDefaults(tc.fp)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\n%s\nresolveDefaults(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestIsUpToDate(t *testing.T) {
	redirectURIs := []string{"https://example.com/callback"}

	remote := zitadel.OIDCApplication{
		Name:                 testAppName,
		RedirectURIs:         redirectURIs,
		ApplicationType:      apiv2.OIDCApplicationType_OIDC_APP_TYPE_WEB,
		AuthMethodType:       apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC,
		AccessTokenType:      apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER,
		ResponseTypes:        []apiv2.OIDCResponseType{apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE},
		GrantTypes:           []apiv2.OIDCGrantType{apiv2.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE, apiv2.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN},
		State:                "Active",
		AdditionalOrigins:    []string{"https://example.com"},
		BackChannelLogoutURI: "https://example.com/backchannel",
		ClockSkew:            "300s",
	}

	base := v1alpha1.OIDCApplicationParameters{
		Name:                 testAppName,
		RedirectURIs:         redirectURIs,
		AdditionalOrigins:    []string{"https://example.com"},
		BackChannelLogoutURI: ptr("https://example.com/backchannel"),
		ClockSkew:            ptr("5m"),
	}

	cases := map[string]struct {
		reason string
		fp     v1alpha1.OIDCApplicationParameters
		want   bool
	}{
		"Unchanged": {reason: "An application matching the desired state is up to date", fp: base, want: true},
		"Renamed": {
			reason: "A differing name is drift",
			fp:     withName(base, "billing-web-v2"),
			want:   false,
		},
		"RedirectURIDrift": {
			reason: "Differing redirect URIs are drift",
			fp:     withRedirects(base, []string{"https://evil.example.com/callback"}),
			want:   false,
		},
		"ResponseTypeDrift": {
			reason: "Differing response types are drift",
			fp:     withResponseTypes(base, []v1alpha1.OIDCResponseType{v1alpha1.OIDCResponseTypeIdToken}),
			want:   false,
		},
		"GrantTypeDrift": {
			reason: "Differing grant types are drift",
			fp:     withGrantTypes(base, []v1alpha1.OIDCGrantType{v1alpha1.OIDCGrantTypeAuthorizationCode}),
			want:   false,
		},
		"ApplicationTypeDrift": {
			reason: "Differing application types are drift",
			fp:     withAppType(base, v1alpha1.OIDCApplicationTypeNative),
			want:   false,
		},
		"StateDrift": {
			reason: "Differing states are drift",
			fp:     withState(base, v1alpha1.ApplicationStateInactive),
			want:   false,
		},
		"ClockSkewEquivalent": {
			reason: "An equivalent but differently formatted clock skew is not drift",
			fp:     withClockSkew(base, "300s"),
			want:   true,
		},
		"ClockSkewDrift": {
			reason: "A differing clock skew is drift",
			fp:     withClockSkew(base, "10s"),
			want:   false,
		},
		"BackChannelDrift": {
			reason: "A differing back channel logout URI is drift",
			fp:     withBackChannel(base, "https://example.com/other"),
			want:   false,
		},
		"AdditionalOriginsDrift": {
			reason: "Differing additional origins are drift",
			fp:     withOrigins(base, []string{"https://other.example.com"}),
			want:   false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.OIDCApplication{Spec: v1alpha1.OIDCApplicationSpec{ForProvider: tc.fp}}
			got, _ := isUpToDate(cr, &remote)
			if got != tc.want {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestIsUpToDateRegenerateSecret(t *testing.T) {
	cr := &v1alpha1.OIDCApplication{
		Spec: v1alpha1.OIDCApplicationSpec{
			ForProvider: v1alpha1.OIDCApplicationParameters{
				Name:                   testAppName,
				RedirectURIs:           []string{"https://example.com/callback"},
				RegenerateClientSecret: ptr(true),
			},
		},
	}

	remote := zitadel.OIDCApplication{
		Name:            testAppName,
		RedirectURIs:    []string{"https://example.com/callback"},
		ApplicationType: apiv2.OIDCApplicationType_OIDC_APP_TYPE_WEB,
		AuthMethodType:  apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC,
		AccessTokenType: apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER,
	}

	upToDate, regenerate := isUpToDate(cr, &remote)
	if !upToDate {
		t.Errorf("isUpToDate(...): want true, got false")
	}
	if !regenerate {
		t.Errorf("isUpToDate(...): want regenerate true, got false")
	}
}

func TestConnectionDetails(t *testing.T) {
	details := connectionDetails(&zitadel.OIDCApplication{ClientID: "123@project"})
	want := managed.ConnectionDetails{v1alpha1.ConnectionKeyClientID: []byte("123@project")}

	if diff := cmp.Diff(want, details); diff != "" {
		t.Errorf("connectionDetails(...): -want, +got:\n%s", diff)
	}
}

func TestEqualDuration(t *testing.T) {
	cases := map[string]struct {
		reason  string
		desired string
		actual  string
		want    bool
	}{
		"Identical": {
			reason:  "Identical durations are equal",
			desired: "5s", actual: "5s", want: true,
		},
		"Equivalent": {
			reason:  "Equivalent durations written differently are equal",
			desired: "300s", actual: "5m", want: true,
		},
		"Different": {
			reason:  "Different durations are not equal",
			desired: "5s", actual: "10s", want: false,
		},
		"GarbageDesired": {
			reason:  "A garbage desired duration is not equal",
			desired: "soon", actual: "5s", want: false,
		},
		"GarbageActual": {
			reason:  "A garbage actual duration is not equal",
			desired: "5s", actual: "later", want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := equalDuration(tc.desired, tc.actual); got != tc.want {
				t.Errorf("\n%s\nequalDuration(%q, %q): want %v, got %v", tc.reason, tc.desired, tc.actual, tc.want, got)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.OIDCApplication{}

	updateStatus(cr, &zitadel.OIDCApplication{
		ApplicationID:   "a1",
		ProjectID:       "p1",
		Name:            testAppName,
		State:           "Active",
		ClientID:        "123@p1",
		AllowedOrigins:  []string{"https://example.com"},
		NonCompliant:    true,
		ResponseTypes:   []apiv2.OIDCResponseType{apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE},
		GrantTypes:      []apiv2.OIDCGrantType{apiv2.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN},
		ApplicationType: apiv2.OIDCApplicationType_OIDC_APP_TYPE_NATIVE,
		AuthMethodType:  apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_POST,
		AccessTokenType: apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_JWT,
		CreationDate:    "2024-01-02T03:04:05Z",
	})

	if got := common.Deref(cr.Status.AtProvider.ClientID); got != "123@p1" {
		t.Errorf("clientID: want %q, got %q", "123@p1", got)
	}
	if cr.Status.AtProvider.ApplicationType == nil || *cr.Status.AtProvider.ApplicationType != v1alpha1.OIDCApplicationTypeNative {
		t.Errorf("applicationType: want %q, got %v", v1alpha1.OIDCApplicationTypeNative, cr.Status.AtProvider.ApplicationType)
	}
	if cr.Status.AtProvider.AuthMethodType == nil || *cr.Status.AtProvider.AuthMethodType != v1alpha1.OIDCAuthMethodTypePost {
		t.Errorf("authMethodType: want %q, got %v", v1alpha1.OIDCAuthMethodTypePost, cr.Status.AtProvider.AuthMethodType)
	}
	if cr.Status.AtProvider.AccessTokenType == nil || *cr.Status.AtProvider.AccessTokenType != v1alpha1.OIDCTokenTypeJwt {
		t.Errorf("accessTokenType: want %q, got %v", v1alpha1.OIDCTokenTypeJwt, cr.Status.AtProvider.AccessTokenType)
	}
	if diff := cmp.Diff([]v1alpha1.OIDCGrantType{v1alpha1.OIDCGrantTypeRefreshToken}, cr.Status.AtProvider.GrantTypes); diff != "" {
		t.Errorf("grantTypes: -want, +got:\n%s", diff)
	}
	if !common.Bool(cr.Status.AtProvider.NonCompliant) {
		t.Error("nonCompliant: want true, got false")
	}
}

func ptr[T any](v T) *T { return &v }

func withName(fp v1alpha1.OIDCApplicationParameters, v string) v1alpha1.OIDCApplicationParameters {
	fp.Name = v
	return fp
}

func withRedirects(fp v1alpha1.OIDCApplicationParameters, v []string) v1alpha1.OIDCApplicationParameters {
	fp.RedirectURIs = v
	return fp
}

func withOrigins(fp v1alpha1.OIDCApplicationParameters, v []string) v1alpha1.OIDCApplicationParameters {
	fp.AdditionalOrigins = v
	return fp
}

func withResponseTypes(fp v1alpha1.OIDCApplicationParameters, v []v1alpha1.OIDCResponseType) v1alpha1.OIDCApplicationParameters {
	fp.ResponseTypes = v
	return fp
}

func withGrantTypes(fp v1alpha1.OIDCApplicationParameters, v []v1alpha1.OIDCGrantType) v1alpha1.OIDCApplicationParameters {
	fp.GrantTypes = v
	return fp
}

func withAppType(fp v1alpha1.OIDCApplicationParameters, v v1alpha1.OIDCApplicationType) v1alpha1.OIDCApplicationParameters {
	fp.ApplicationType = &v
	return fp
}

func withState(fp v1alpha1.OIDCApplicationParameters, v v1alpha1.ApplicationState) v1alpha1.OIDCApplicationParameters {
	fp.State = &v
	return fp
}

func withClockSkew(fp v1alpha1.OIDCApplicationParameters, v string) v1alpha1.OIDCApplicationParameters {
	fp.ClockSkew = &v
	return fp
}

func withBackChannel(fp v1alpha1.OIDCApplicationParameters, v string) v1alpha1.OIDCApplicationParameters {
	fp.BackChannelLogoutURI = &v
	return fp
}
