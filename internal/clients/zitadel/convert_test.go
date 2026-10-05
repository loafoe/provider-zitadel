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

	apiv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"
	objectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/object/v2"
	orgv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/org/v2"
	projectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/project/v2"
	userv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Values the cases share, named so that a case reads as a statement about
// behaviour rather than about a repeated literal.
const (
	garbageValue = "Nope"
)

// Test values, named so that a case reads as a statement about behaviour
// rather than about a repeated string.
const (
	respCode         = "Code"
	respIdToken      = "IdToken"
	respIdTokenToken = "IdTokenToken"
	unspec           = "Unspecified"
)

// The conversions in convert.go are the boundary between the API the provider
// exposes and the protobuf the Zitadel API speaks, so they are where a
// mismatch silently becomes a resource that never converges. Every mapping is
// covered here in both directions, including the empty and unknown values,
// because those are the ones that decide whether drift is reported.

// stateCase is one row of a bidirectional enum mapping.
type stateCase struct {
	// api is the representation the provider API uses.
	api string
	// proto is the protobuf value it maps to.
	proto int32
	// fromProto is what the reverse mapping yields.
	fromProto string
}

// TestEnumMappings walks a bidirectional mapping: API to protobuf, and
// protobuf back to the API value.
//
// wantProtoErr says whether the API to protobuf direction rejects the value, and
// wantAPIFromProtoErr whether the reverse direction does. They differ for
// values that are spelled differently on the way out and on the way back, such
// as an enum whose empty input is a real value rather than "unspecified".
func TestEnumMappings(t *testing.T) {
	cases := map[string]struct {
		reason string
		// toProto maps the API representation to a protobuf value.
		toProto func(string) (int32, error)
		// fromProto maps a protobuf value back to the API representation.
		fromProto func(int32) string
		rows      []stateCase
	}{
		"UserState": {
			reason:    "A user state round trips, and an unknown one is rejected",
			toProto:   func(s string) (int32, error) { v, err := UserStateToProto(s); return int32(v), err },
			fromProto: func(v int32) string { return UserStateFromProto(userv2.UserState(v)) },
			rows: []stateCase{
				{api: StateActive, proto: int32(userv2.UserState_USER_STATE_ACTIVE), fromProto: StateActive},
				{api: StateInactive, proto: int32(userv2.UserState_USER_STATE_INACTIVE), fromProto: StateInactive},
				{api: StateLocked, proto: int32(userv2.UserState_USER_STATE_LOCKED), fromProto: StateLocked},
				{api: StateInitial, proto: int32(userv2.UserState_USER_STATE_INITIAL), fromProto: StateInitial},
				// An empty state means "leave it as it is" and is sent as active,
				// so it does not come back as empty.
				{api: "", proto: int32(userv2.UserState_USER_STATE_ACTIVE), fromProto: StateActive},
			},
		},
		"Gender": {
			reason:    "A gender round trips, and an unknown one is rejected",
			toProto:   func(s string) (int32, error) { v, err := GenderToProto(s); return int32(v), err },
			fromProto: func(v int32) string { return GenderFromProto(userv2.Gender(v)) },
			rows: []stateCase{
				{api: "Female", proto: int32(userv2.Gender_GENDER_FEMALE), fromProto: "Female"},
				{api: "Male", proto: int32(userv2.Gender_GENDER_MALE), fromProto: "Male"},
				{api: "Diverse", proto: int32(userv2.Gender_GENDER_DIVERSE), fromProto: "Diverse"},
				{api: unspec, proto: int32(userv2.Gender_GENDER_UNSPECIFIED), fromProto: unspec},
				{api: "", proto: int32(userv2.Gender_GENDER_UNSPECIFIED), fromProto: unspec},
			},
		},
		"AccessTokenType": {
			reason: "A token type round trips, and accepts the JWT spelling Zitadel uses",
			toProto: func(s string) (int32, error) {
				v, err := AccessTokenTypeToProto(s)
				return int32(v), err
			},
			fromProto: func(v int32) string { return AccessTokenTypeFromProto(userv2.AccessTokenType(v)) },
			rows: []stateCase{
				{api: TokenTypeBearer, proto: int32(userv2.AccessTokenType_ACCESS_TOKEN_TYPE_BEARER), fromProto: TokenTypeBearer},
				{api: TokenTypeJwt, proto: int32(userv2.AccessTokenType_ACCESS_TOKEN_TYPE_JWT), fromProto: TokenTypeJwt},
				// Zitadel reports the type in both spellings depending on the
				// surface, so both have to be accepted on the way in.
				{api: "JWT", proto: int32(userv2.AccessTokenType_ACCESS_TOKEN_TYPE_JWT), fromProto: TokenTypeJwt},
				{api: "", proto: int32(userv2.AccessTokenType_ACCESS_TOKEN_TYPE_BEARER), fromProto: TokenTypeBearer},
			},
		},
		"ProjectState": {
			reason:    "A project state round trips, and an unknown one is rejected",
			toProto:   func(s string) (int32, error) { v, err := ProjectStateToProto(s); return int32(v), err },
			fromProto: func(v int32) string { return ProjectStateFromProto(projectv2.ProjectState(v)) },
			rows: []stateCase{
				{api: StateActive, proto: int32(projectv2.ProjectState_PROJECT_STATE_ACTIVE), fromProto: StateActive},
				{api: StateInactive, proto: int32(projectv2.ProjectState_PROJECT_STATE_INACTIVE), fromProto: StateInactive},
				{api: "", proto: int32(projectv2.ProjectState_PROJECT_STATE_ACTIVE), fromProto: StateActive},
			},
		},
		"PrivateLabelingSetting": {
			reason: "A private labeling setting round trips",
			toProto: func(s string) (int32, error) {
				v, err := PrivateLabelingSettingToProto(s)
				return int32(v), err
			},
			fromProto: func(v int32) string {
				return PrivateLabelingSettingFromProto(projectv2.PrivateLabelingSetting(v))
			},
			rows: []stateCase{
				{
					api:       "EnforceProjectResourceOwnerPolicy",
					proto:     int32(projectv2.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_ENFORCE_PROJECT_RESOURCE_OWNER_POLICY),
					fromProto: "EnforceProjectResourceOwnerPolicy",
				},
				{
					api:       "AllowLoginUserResourceOwnerPolicy",
					proto:     int32(projectv2.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_ALLOW_LOGIN_USER_RESOURCE_OWNER_POLICY),
					fromProto: "AllowLoginUserResourceOwnerPolicy",
				},
				{
					api:       "",
					proto:     int32(projectv2.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_UNSPECIFIED),
					fromProto: "",
				},
			},
		},
		"OrganizationState": {
			reason:    "An organization state round trips, and a removed one reads as unmanaged",
			toProto:   func(s string) (int32, error) { v, err := OrganizationStateToProto(s); return int32(v), err },
			fromProto: func(v int32) string { return OrganizationStateFromProto(orgv2.OrganizationState(v)) },
			rows: []stateCase{
				{api: StateActive, proto: int32(orgv2.OrganizationState_ORGANIZATION_STATE_ACTIVE), fromProto: StateActive},
				{api: StateInactive, proto: int32(orgv2.OrganizationState_ORGANIZATION_STATE_INACTIVE), fromProto: StateInactive},
				{api: "", proto: int32(orgv2.OrganizationState_ORGANIZATION_STATE_ACTIVE), fromProto: StateActive},
			},
		},
		"OIDCApplicationType": {
			reason:    "An OIDC application type round trips, defaulting to Web",
			toProto:   func(s string) (int32, error) { v, err := OIDCApplicationTypeToProto(s); return int32(v), err },
			fromProto: func(v int32) string { return OIDCApplicationTypeFromProto(apiv2.OIDCApplicationType(v)) },
			rows: []stateCase{
				{api: "Web", proto: int32(apiv2.OIDCApplicationType_OIDC_APP_TYPE_WEB), fromProto: "Web"},
				{api: "", proto: int32(apiv2.OIDCApplicationType_OIDC_APP_TYPE_WEB), fromProto: "Web"},
				{api: "UserAgent", proto: int32(apiv2.OIDCApplicationType_OIDC_APP_TYPE_USER_AGENT), fromProto: "UserAgent"},
				{api: "Native", proto: int32(apiv2.OIDCApplicationType_OIDC_APP_TYPE_NATIVE), fromProto: "Native"},
			},
		},
		"OIDCAuthMethodType": {
			reason:    "An OIDC auth method round trips, defaulting to basic",
			toProto:   func(s string) (int32, error) { v, err := OIDCAuthMethodTypeToProto(s); return int32(v), err },
			fromProto: func(v int32) string { return OIDCAuthMethodTypeFromProto(apiv2.OIDCAuthMethodType(v)) },
			rows: []stateCase{
				{api: AuthMethodBasic, proto: int32(apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC), fromProto: AuthMethodBasic},
				{api: "", proto: int32(apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC), fromProto: AuthMethodBasic},
				{api: "Post", proto: int32(apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_POST), fromProto: "Post"},
				{api: "None", proto: int32(apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_NONE), fromProto: "None"},
				{
					api:       AuthMethodPrivateKeyJwt,
					proto:     int32(apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_PRIVATE_KEY_JWT),
					fromProto: AuthMethodPrivateKeyJwt,
				},
			},
		},
		"OIDCTokenType": {
			reason: "An OIDC token type round trips, accepting the JWT spelling",
			toProto: func(s string) (int32, error) {
				v, err := OIDCTokenTypeToProto(s)
				return int32(v), err
			},
			fromProto: func(v int32) string { return OIDCTokenTypeFromProto(apiv2.OIDCTokenType(v)) },
			rows: []stateCase{
				{api: TokenTypeBearer, proto: int32(apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER), fromProto: TokenTypeBearer},
				{api: "", proto: int32(apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER), fromProto: TokenTypeBearer},
				{api: TokenTypeJwt, proto: int32(apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_JWT), fromProto: TokenTypeJwt},
				{api: "JWT", proto: int32(apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_JWT), fromProto: TokenTypeJwt},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for _, row := range tc.rows {
				t.Run(row.api, func(t *testing.T) {
					got, err := tc.toProto(row.api)
					if err != nil {
						t.Fatalf("\n%s\ntoProto(%q): unexpected error: %v", tc.reason, row.api, err)
					}
					if got != row.proto {
						t.Errorf("\n%s\ntoProto(%q): want %d, got %d", tc.reason, row.api, row.proto, got)
					}

					if back := tc.fromProto(row.proto); back != row.fromProto {
						t.Errorf("\n%s\nfromProto(%d): want %q, got %q", tc.reason, row.proto, row.fromProto, back)
					}
				})
			}
		})
	}
}

// TestToProtoRejectsUnknown covers the rejection path of every mapping that can
// fail. A value the provider does not understand has to be an error rather than
// a silent fallback, otherwise a typo in a manifest turns into permanent drift
// that nothing ever reports.
func TestToProtoRejectsUnknown(t *testing.T) {
	cases := map[string]struct {
		reason  string
		fn      func(string) error
		garbage string
	}{
		"UserState": {reason: "An unknown user state is rejected", fn: func(s string) error { _, err := UserStateToProto(s); return err }, garbage: garbageValue},
		"Gender":    {reason: "An unknown gender is rejected", fn: func(s string) error { _, err := GenderToProto(s); return err }, garbage: garbageValue},
		"AccessTokenType": {
			reason:  "An unknown access token type is rejected",
			fn:      func(s string) error { _, err := AccessTokenTypeToProto(s); return err },
			garbage: garbageValue,
		},
		"ProjectState": {
			reason:  "An unknown project state is rejected",
			fn:      func(s string) error { _, err := ProjectStateToProto(s); return err },
			garbage: StateLocked,
		},
		"PrivateLabelingSetting": {
			reason:  "An unknown private labeling setting is rejected",
			fn:      func(s string) error { _, err := PrivateLabelingSettingToProto(s); return err },
			garbage: garbageValue,
		},
		"OrganizationState": {
			reason:  "An unknown organization state is rejected",
			fn:      func(s string) error { _, err := OrganizationStateToProto(s); return err },
			garbage: "Removed",
		},
		"OIDCApplicationType": {
			reason:  "An unknown OIDC application type is rejected",
			fn:      func(s string) error { _, err := OIDCApplicationTypeToProto(s); return err },
			garbage: garbageValue,
		},
		"OIDCAuthMethodType": {
			reason:  "An unknown OIDC auth method is rejected",
			fn:      func(s string) error { _, err := OIDCAuthMethodTypeToProto(s); return err },
			garbage: garbageValue,
		},
		"OIDCResponseType": {
			reason:  "An unknown OIDC response type is rejected, including the empty one",
			fn:      func(s string) error { _, err := OIDCResponseTypeToProto(s); return err },
			garbage: garbageValue,
		},
		"OIDCResponseTypeEmpty": {
			reason:  "An empty OIDC response type is rejected rather than defaulted",
			fn:      func(s string) error { _, err := OIDCResponseTypeToProto(s); return err },
			garbage: "",
		},
		"OIDCGrantType": {
			reason:  "An unknown OIDC grant type is rejected",
			fn:      func(s string) error { _, err := OIDCGrantTypeToProto(s); return err },
			garbage: garbageValue,
		},
		"OIDCTokenType": {
			reason:  "An unknown OIDC token type is rejected",
			fn:      func(s string) error { _, err := OIDCTokenTypeToProto(s); return err },
			garbage: garbageValue,
		},
		"OIDCVersion": {
			reason:  "An unknown OIDC version is rejected",
			fn:      func(s string) error { _, err := OIDCVersionToProto(s); return err },
			garbage: "2.0",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if err := tc.fn(tc.garbage); err == nil {
				t.Errorf("\n%s\ntoProto(%q): want an error, got nil", tc.reason, tc.garbage)
			}
		})
	}
}

// TestResponseAndGrantTypes lists every response and grant type the OIDC
// settings can carry. These are sets rather than enums with a default, so each
// one is checked individually: a response type that silently becomes
// unspecified would drop a grant the manifest asked for.
func TestResponseAndGrantTypes(t *testing.T) {
	responseCases := map[string]struct {
		reason string
		api    string
		proto  apiv2.OIDCResponseType
	}{
		respCode: {reason: "Code maps to code", api: respCode, proto: apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE},
		respIdToken: {
			reason: "IdToken maps to id token", api: respIdToken,
			proto: apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_ID_TOKEN,
		},
		respIdTokenToken: {
			reason: "IdTokenToken maps to both", api: respIdTokenToken,
			proto: apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_ID_TOKEN_TOKEN,
		},
		unspec: {
			// Unlike the enums with a default, this one has no empty case: the
			// list is optional and the CRD restricts its entries to the three
			// above, so an unset response type cannot reach this function.
			reason: "Unspecified renders as the empty API value",
			proto:  apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_UNSPECIFIED,
		},
	}

	for name, tc := range responseCases {
		t.Run("Response/"+name, func(t *testing.T) {
			if tc.api != "" {
				got, err := OIDCResponseTypeToProto(tc.api)
				if err != nil {
					t.Fatalf("\n%s\nOIDCResponseTypeToProto(%q): unexpected error: %v", tc.reason, tc.api, err)
				}

				if got != tc.proto {
					t.Errorf("\n%s\nOIDCResponseTypeToProto(%q): want %v, got %v", tc.reason, tc.api, tc.proto, got)
				}
			}

			if back := OIDCResponseTypeFromProto(tc.proto); back != tc.api {
				t.Errorf("\n%s\nround trip: want %q, got %q", tc.reason, tc.api, back)
			}
		})
	}

	grantCases := map[string]string{
		"AuthorizationCode": "AuthorizationCode",
		"Implicit":          "Implicit",
		"RefreshToken":      "RefreshToken",
		"DeviceCode":        "DeviceCode",
		"TokenExchange":     "TokenExchange",
	}

	for api := range grantCases {
		t.Run("Grant/"+api, func(t *testing.T) {
			got, err := OIDCGrantTypeToProto(api)
			if err != nil {
				t.Fatalf("OIDCGrantTypeToProto(%q): unexpected error: %v", api, err)
			}

			if back := OIDCGrantTypeFromProto(got); back != api {
				t.Errorf("round trip: want %q, got %q", api, back)
			}
		})
	}
}

// TestOIDCVersion covers the only version Zitadel supports, and the rejection
// of anything else.
func TestOIDCVersion(t *testing.T) {
	for _, v := range []string{"", "1.0"} {
		got, err := OIDCVersionToProto(v)
		if err != nil {
			t.Fatalf("OIDCVersionToProto(%q): unexpected error: %v", v, err)
		}

		if got != apiv2.OIDCVersion_OIDC_VERSION_1_0 {
			t.Errorf("OIDCVersionToProto(%q): want 1.0, got %v", v, got)
		}
	}
}

// TestGrantedProjectState covers the grant states, which are a separate enum
// from the project's own state and are what a project grant reports.
func TestGrantedProjectState(t *testing.T) {
	cases := map[string]struct {
		reason  string
		proto   projectv2.GrantedProjectState
		want    string
		wantNil bool
	}{
		"Active":   {reason: "An active grant reads as active", proto: projectv2.GrantedProjectState_GRANTED_PROJECT_STATE_ACTIVE, want: StateActive},
		"Inactive": {reason: "An inactive grant reads as inactive", proto: projectv2.GrantedProjectState_GRANTED_PROJECT_STATE_INACTIVE, want: StateInactive},
		unspec:     {reason: "An unspecified grant reads as unmanaged", proto: projectv2.GrantedProjectState_GRANTED_PROJECT_STATE_UNSPECIFIED, want: ""},
		"Unknown":  {reason: "An unknown grant state reads as unmanaged", proto: projectv2.GrantedProjectState(99), want: "", wantNil: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := GrantedProjectStateFromProto(tc.proto); got != tc.want {
				t.Errorf("\n%s\nGrantedProjectStateFromProto(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

// TestApplicationState covers the application states, including the removed
// state that organizations and projects do not have.
func TestApplicationState(t *testing.T) {
	cases := map[string]struct {
		reason string
		proto  apiv2.ApplicationState
		want   string
	}{
		"Active":   {reason: "An active application reads as active", proto: apiv2.ApplicationState_APPLICATION_STATE_ACTIVE, want: StateActive},
		"Inactive": {reason: "An inactive application reads as inactive", proto: apiv2.ApplicationState_APPLICATION_STATE_INACTIVE, want: StateInactive},
		"Removed":  {reason: "A removed application is reported as removed", proto: apiv2.ApplicationState_APPLICATION_STATE_REMOVED, want: "Removed"},
		unspec:     {reason: "An unspecified application reads as unmanaged", proto: apiv2.ApplicationState_APPLICATION_STATE_UNSPECIFIED, want: ""},
		"Unknown":  {reason: "An unknown state reads as unmanaged", proto: apiv2.ApplicationState(99), want: ""},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ApplicationStateFromProto(tc.proto); got != tc.want {
				t.Errorf("\n%s\nApplicationStateFromProto(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

// TestFormatDates covers the rendering of the timestamps Zitadel reports on a
// resource's details. An unset or invalid timestamp has to render as empty
// rather than as the zero time, because it is compared against a value the
// controller writes.
func TestFormatDates(t *testing.T) {
	valid := timestamppb.New(timestamppb.Now().AsTime())

	cases := map[string]struct {
		reason    string
		details   *objectv2.Details
		wantEmpty bool
	}{
		"Nil": {
			reason:    "A resource with no details renders no date",
			details:   nil,
			wantEmpty: true,
		},
		"Unset": {
			reason:    "A resource with an unset date renders no date",
			details:   &objectv2.Details{},
			wantEmpty: true,
		},
		"Invalid": {
			reason:    "An out of range date renders no date rather than a nonsense one",
			details:   &objectv2.Details{CreationDate: &timestamppb.Timestamp{Seconds: 1 << 62}},
			wantEmpty: true,
		},
		"Set": {
			reason:  "A valid date renders as RFC3339",
			details: &objectv2.Details{CreationDate: valid, ChangeDate: valid},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := formatCreationDate(tc.details)
			if tc.wantEmpty && got != "" {
				t.Errorf("\n%s\nformatCreationDate(...): want empty, got %q", tc.reason, got)
			}

			if !tc.wantEmpty && got == "" {
				t.Errorf("\n%s\nformatCreationDate(...): want a date, got empty", tc.reason)
			}

			if got := formatChangeDate(tc.details); tc.wantEmpty != (got == "") {
				t.Errorf("\n%s\nformatChangeDate(...): wantEmpty %v, got %q", tc.reason, tc.wantEmpty, got)
			}
		})
	}
}
