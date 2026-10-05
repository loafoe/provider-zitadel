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
	"testing"

	appv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/app"
	projectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/project/v2"
)

// A collection of the small pure mappings that sit between this provider's API
// and Zitadel's. None of them needs an instance, and all of them are places
// where a mistake is invisible until a resource fails to settle.

// TestProjectGrantExternalName covers the name a project grant is recorded under,
// and the parsing of it back.
//
// TestSplitProjectGrantExternalName covers the rejections, which matter because a
// half-parsed name would point at the wrong project or organization.
func TestSplitProjectGrantExternalName(t *testing.T) {
	cases := map[string]struct {
		reason      string
		in          string
		wantProject string
		wantGranted string
		wantOK      bool
	}{
		"Both": {
			reason:      "A well formed name parses",
			in:          "p1/org-2",
			wantProject: "p1", wantGranted: "org-2", wantOK: true,
		},
		"NoSeparator": {
			reason: "A name with no separator is rejected rather than half parsed",
			in:     "p1org-2",
		},
		"NoProject": {
			reason: "A name with no project is rejected",
			in:     "/org-2",
		},
		"NoGrantedOrg": {
			reason: "A name with no granted organization is rejected",
			in:     "p1/",
		},
		"Empty": {
			reason: "An empty name is rejected",
			in:     "",
		},
		"ExtraSeparators": {
			reason:      "Only the first separator splits, so a name with more is not mistaken for a pair",
			in:          "p1/org/2",
			wantProject: "p1", wantGranted: "org/2", wantOK: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			project, granted, ok := SplitProjectGrantExternalName(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("\n%s\nSplitProjectGrantExternalName(%q): want ok=%v, got %v", tc.reason, tc.in, tc.wantOK, ok)
			}

			if !tc.wantOK {
				if project != "" || granted != "" {
					t.Errorf("\n%s\na rejected name must yield nothing, got %q/%q", tc.reason, project, granted)
				}

				return
			}

			if project != tc.wantProject || granted != tc.wantGranted {
				t.Errorf("\n%s\nSplitProjectGrantExternalName(%q): want %q/%q, got %q/%q",
					tc.reason, tc.in, tc.wantProject, tc.wantGranted, project, granted)
			}
		})
	}
}

func TestProjectGrantStateFromProto(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     projectv2.ProjectGrantState
		want   string
	}{
		"Active":      {reason: "An active grant reads as active", in: projectv2.ProjectGrantState_PROJECT_GRANT_STATE_ACTIVE, want: StateActive},
		"Inactive":    {reason: "An inactive grant reads as inactive", in: projectv2.ProjectGrantState_PROJECT_GRANT_STATE_INACTIVE, want: StateInactive},
		"Unspecified": {reason: "An unspecified grant reads as unmanaged", in: projectv2.ProjectGrantState_PROJECT_GRANT_STATE_UNSPECIFIED, want: ""},
		"Unknown":     {reason: "An unknown state reads as unmanaged", in: projectv2.ProjectGrantState(99), want: ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ProjectGrantStateFromProto(tc.in); got != tc.want {
				t.Errorf("\n%s\nProjectGrantStateFromProto(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

// TestV1OIDCmappers covers the translations into ZITADEL's v1 application API,
// which is the code path its own Terraform provider uses for OIDC configuration.
//
// Each is checked with a value it accepts, an empty input, and a value it must
// reject: a rejected value becomes an error the operator sees rather than a
// setting that silently does not apply.
func TestV1OIDCmappers(t *testing.T) {
	cases := map[string]struct {
		reason string
		fn     func(string) (int32, error)
		rows   map[string]int32
		// rejects is the input the mapper must refuse.
		rejects []string
	}{
		"ResponseType": {
			reason: "Response types map to their v1 equivalents",
			fn: func(s string) (int32, error) {
				v, err := ResponseTypeToV1(s)
				return int32(v), err
			},
			rows: map[string]int32{
				"Code":         int32(appv1.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE),
				"IdToken":      int32(appv1.OIDCResponseType_OIDC_RESPONSE_TYPE_ID_TOKEN),
				"IdTokenToken": int32(appv1.OIDCResponseType_OIDC_RESPONSE_TYPE_ID_TOKEN_TOKEN),
			},
			rejects: []string{"", "Nope"},
		},
		"GrantType": {
			reason: "Grant types map to their v1 equivalents",
			fn: func(s string) (int32, error) {
				v, err := GrantTypeToV1(s)
				return int32(v), err
			},
			rows: map[string]int32{
				"AuthorizationCode": int32(appv1.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE),
				"Implicit":          int32(appv1.OIDCGrantType_OIDC_GRANT_TYPE_IMPLICIT),
				"RefreshToken":      int32(appv1.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN),
				"DeviceCode":        int32(appv1.OIDCGrantType_OIDC_GRANT_TYPE_DEVICE_CODE),
				"TokenExchange":     int32(appv1.OIDCGrantType_OIDC_GRANT_TYPE_TOKEN_EXCHANGE),
			},
			rejects: []string{"", "Nope"},
		},
		"ApplicationType": {
			reason: "Application types map, defaulting to Web",
			fn: func(s string) (int32, error) {
				v, err := ApplicationTypeToV1(s)
				return int32(v), err
			},
			rows: map[string]int32{
				"":          int32(appv1.OIDCAppType_OIDC_APP_TYPE_WEB),
				"Web":       int32(appv1.OIDCAppType_OIDC_APP_TYPE_WEB),
				"UserAgent": int32(appv1.OIDCAppType_OIDC_APP_TYPE_USER_AGENT),
				"Native":    int32(appv1.OIDCAppType_OIDC_APP_TYPE_NATIVE),
			},
			rejects: []string{"Nope"},
		},
		"AuthMethodType": {
			reason: "Auth methods map, defaulting to basic",
			fn: func(s string) (int32, error) {
				v, err := AuthMethodTypeToV1(s)
				return int32(v), err
			},
			rows: map[string]int32{
				"":                      int32(appv1.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC),
				AuthMethodBasic:         int32(appv1.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC),
				"Post":                  int32(appv1.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_POST),
				"None":                  int32(appv1.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_NONE),
				AuthMethodPrivateKeyJwt: int32(appv1.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_PRIVATE_KEY_JWT),
			},
			rejects: []string{"Nope"},
		},
		"TokenType": {
			reason: "Token types map, accepting both spellings of JWT",
			fn: func(s string) (int32, error) {
				v, err := TokenTypeToV1(s)
				return int32(v), err
			},
			rows: map[string]int32{
				"":              int32(appv1.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER),
				TokenTypeBearer: int32(appv1.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER),
				TokenTypeJwt:    int32(appv1.OIDCTokenType_OIDC_TOKEN_TYPE_JWT),
				"JWT":           int32(appv1.OIDCTokenType_OIDC_TOKEN_TYPE_JWT),
			},
			rejects: []string{"Nope"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for in, want := range tc.rows {
				got, err := tc.fn(in)
				if err != nil {
					t.Fatalf("\n%s\n%q: unexpected error: %v", tc.reason, in, err)
				}

				if got != want {
					t.Errorf("\n%s\n%q: want %d, got %d", tc.reason, in, want, got)
				}
			}

			for _, in := range tc.rejects {
				if _, err := tc.fn(in); err == nil {
					t.Errorf("\n%s\n%q: want the value to be rejected, got no error", tc.reason, in)
				}
			}
		})
	}
}
