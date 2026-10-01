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
	"fmt"
	"time"

	apiv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"
	objectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/object/v2"
	orgv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/org/v2"
	projectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/project/v2"
	userv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user/v2"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Lifecycle states shared by organizations, projects, applications and users.
const (
	// StateActive is an active resource.
	StateActive = "Active"

	// StateInactive is a deactivated resource.
	StateInactive = "Inactive"

	// StateLocked is a locked user.
	StateLocked = "Locked"

	// StateInitial is a user that has never logged in.
	StateInitial = "Initial"
)

// Token formats shared by service accounts and applications.
const (
	// TokenTypeBearer issues opaque bearer tokens.
	TokenTypeBearer = "Bearer"

	// TokenTypeJwt issues signed JWT tokens.
	TokenTypeJwt = "Jwt"
)

// ParseDuration converts a duration string such as "5s" into a protobuf
// duration.
func ParseDuration(d string) (*durationpb.Duration, error) {
	parsed, err := time.ParseDuration(d)
	if err != nil {
		return nil, fmt.Errorf("cannot parse duration %q: %w", d, err)
	}

	if parsed < 0 {
		return nil, fmt.Errorf("duration %q must not be negative", d)
	}

	return durationpb.New(parsed), nil
}

// formatTimestamp renders a protobuf timestamp as an RFC3339 string. Unset or
// invalid timestamps render as an empty string.
func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil || !ts.IsValid() {
		return ""
	}

	return ts.AsTime().Format(time.RFC3339)
}

// formatCreationDate renders the creation timestamp of a Zitadel resource
// details object.
func formatCreationDate(d *objectv2.Details) string {
	if d == nil {
		return ""
	}

	return formatTimestamp(d.GetCreationDate())
}

// formatChangeDate renders the last change timestamp of a Zitadel resource
// details object.
func formatChangeDate(d *objectv2.Details) string {
	if d == nil {
		return ""
	}

	return formatTimestamp(d.GetChangeDate())
}

// UserStateFromProto maps a protobuf user state onto the API representation. A
// deleted or unspecified user maps to the empty string, which the controller
// treats as "no state is managed".
func UserStateFromProto(s userv2.UserState) string {
	switch s {
	case userv2.UserState_USER_STATE_ACTIVE:
		return StateActive
	case userv2.UserState_USER_STATE_INACTIVE:
		return StateInactive
	case userv2.UserState_USER_STATE_LOCKED:
		return StateLocked
	case userv2.UserState_USER_STATE_INITIAL:
		return StateInitial
	case userv2.UserState_USER_STATE_UNSPECIFIED, userv2.UserState_USER_STATE_DELETED:
		return ""
	}

	return ""
}

// UserStateToProto maps an API user state onto the protobuf representation. An
// empty state means "leave it as it is".
func UserStateToProto(s string) (userv2.UserState, error) {
	switch s {
	case "", StateActive:
		return userv2.UserState_USER_STATE_ACTIVE, nil
	case StateInactive:
		return userv2.UserState_USER_STATE_INACTIVE, nil
	case StateLocked:
		return userv2.UserState_USER_STATE_LOCKED, nil
	case StateInitial:
		return userv2.UserState_USER_STATE_INITIAL, nil
	default:
		return userv2.UserState_USER_STATE_UNSPECIFIED, fmt.Errorf("unsupported user state %q", s)
	}
}

// GenderFromProto maps a protobuf gender onto the API representation.
func GenderFromProto(g userv2.Gender) string {
	switch g {
	case userv2.Gender_GENDER_FEMALE:
		return "Female"
	case userv2.Gender_GENDER_MALE:
		return "Male"
	case userv2.Gender_GENDER_DIVERSE:
		return "Diverse"
	case userv2.Gender_GENDER_UNSPECIFIED:
		return "Unspecified"
	}

	return "Unspecified"
}

// GenderToProto maps an API gender onto the protobuf representation.
func GenderToProto(g string) (userv2.Gender, error) {
	switch g {
	case "", "Unspecified":
		return userv2.Gender_GENDER_UNSPECIFIED, nil
	case "Female":
		return userv2.Gender_GENDER_FEMALE, nil
	case "Male":
		return userv2.Gender_GENDER_MALE, nil
	case "Diverse":
		return userv2.Gender_GENDER_DIVERSE, nil
	default:
		return userv2.Gender_GENDER_UNSPECIFIED, fmt.Errorf("unsupported gender %q", g)
	}
}

// AccessTokenTypeFromProto maps the protobuf access token type of a service
// account onto the API representation.
func AccessTokenTypeFromProto(t userv2.AccessTokenType) string {
	if t == userv2.AccessTokenType_ACCESS_TOKEN_TYPE_JWT {
		return TokenTypeJwt
	}

	return TokenTypeBearer
}

// AccessTokenTypeToProto maps an API access token type onto the protobuf
// representation used by service accounts.
func AccessTokenTypeToProto(t string) (userv2.AccessTokenType, error) {
	switch t {
	case "", TokenTypeBearer:
		return userv2.AccessTokenType_ACCESS_TOKEN_TYPE_BEARER, nil
	case TokenTypeJwt, "JWT":
		return userv2.AccessTokenType_ACCESS_TOKEN_TYPE_JWT, nil
	default:
		return userv2.AccessTokenType_ACCESS_TOKEN_TYPE_BEARER, fmt.Errorf("unsupported access token type %q", t)
	}
}

// ProjectStateFromProto maps a protobuf project state onto the API
// representation.
func ProjectStateFromProto(s projectv2.ProjectState) string {
	switch s {
	case projectv2.ProjectState_PROJECT_STATE_ACTIVE:
		return StateActive
	case projectv2.ProjectState_PROJECT_STATE_INACTIVE:
		return StateInactive
	case projectv2.ProjectState_PROJECT_STATE_UNSPECIFIED:
		return ""
	}

	return ""
}

// ProjectStateToProto maps an API project state onto the protobuf
// representation.
func ProjectStateToProto(s string) (projectv2.ProjectState, error) {
	switch s {
	case "", StateActive:
		return projectv2.ProjectState_PROJECT_STATE_ACTIVE, nil
	case StateInactive:
		return projectv2.ProjectState_PROJECT_STATE_INACTIVE, nil
	default:
		return projectv2.ProjectState_PROJECT_STATE_UNSPECIFIED, fmt.Errorf("unsupported project state %q", s)
	}
}

// GrantedProjectStateFromProto maps the state of a project grant onto the API
// representation.
func GrantedProjectStateFromProto(s projectv2.GrantedProjectState) string {
	switch s {
	case projectv2.GrantedProjectState_GRANTED_PROJECT_STATE_ACTIVE:
		return StateActive
	case projectv2.GrantedProjectState_GRANTED_PROJECT_STATE_INACTIVE:
		return StateInactive
	case projectv2.GrantedProjectState_GRANTED_PROJECT_STATE_UNSPECIFIED:
		return ""
	}

	return ""
}

// PrivateLabelingSettingFromProto maps the protobuf private labeling setting
// onto the API representation.
func PrivateLabelingSettingFromProto(s projectv2.PrivateLabelingSetting) string {
	switch s {
	case projectv2.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_ENFORCE_PROJECT_RESOURCE_OWNER_POLICY:
		return "EnforceProjectResourceOwnerPolicy"
	case projectv2.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_ALLOW_LOGIN_USER_RESOURCE_OWNER_POLICY:
		return "AllowLoginUserResourceOwnerPolicy"
	case projectv2.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_UNSPECIFIED:
		return ""
	}

	return ""
}

// PrivateLabelingSettingToProto maps an API private labeling setting onto the
// protobuf representation.
func PrivateLabelingSettingToProto(s string) (projectv2.PrivateLabelingSetting, error) {
	switch s {
	case "":
		return projectv2.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_UNSPECIFIED, nil
	case "EnforceProjectResourceOwnerPolicy":
		return projectv2.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_ENFORCE_PROJECT_RESOURCE_OWNER_POLICY, nil
	case "AllowLoginUserResourceOwnerPolicy":
		return projectv2.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_ALLOW_LOGIN_USER_RESOURCE_OWNER_POLICY, nil
	default:
		return projectv2.PrivateLabelingSetting_PRIVATE_LABELING_SETTING_UNSPECIFIED, fmt.Errorf("unsupported private labeling setting %q", s)
	}
}

// OrganizationStateFromProto maps a protobuf organization state onto the API
// representation.
func OrganizationStateFromProto(s orgv2.OrganizationState) string {
	switch s {
	case orgv2.OrganizationState_ORGANIZATION_STATE_ACTIVE:
		return StateActive
	case orgv2.OrganizationState_ORGANIZATION_STATE_INACTIVE:
		return StateInactive
	case orgv2.OrganizationState_ORGANIZATION_STATE_UNSPECIFIED, orgv2.OrganizationState_ORGANIZATION_STATE_REMOVED:
		return ""
	}

	return ""
}

// OrganizationStateToProto maps an API organization state onto the protobuf
// representation.
func OrganizationStateToProto(s string) (orgv2.OrganizationState, error) {
	switch s {
	case "", StateActive:
		return orgv2.OrganizationState_ORGANIZATION_STATE_ACTIVE, nil
	case StateInactive:
		return orgv2.OrganizationState_ORGANIZATION_STATE_INACTIVE, nil
	default:
		return orgv2.OrganizationState_ORGANIZATION_STATE_UNSPECIFIED, fmt.Errorf("unsupported organization state %q", s)
	}
}

// ApplicationStateFromProto maps a protobuf application state onto the API
// representation.
func ApplicationStateFromProto(s apiv2.ApplicationState) string {
	switch s {
	case apiv2.ApplicationState_APPLICATION_STATE_ACTIVE:
		return StateActive
	case apiv2.ApplicationState_APPLICATION_STATE_INACTIVE:
		return StateInactive
	case apiv2.ApplicationState_APPLICATION_STATE_REMOVED:
		return "Removed"
	case apiv2.ApplicationState_APPLICATION_STATE_UNSPECIFIED:
		return ""
	}

	return ""
}

// OIDCApplicationTypeToProto maps an API OIDC application type onto the
// protobuf representation.
func OIDCApplicationTypeToProto(t string) (apiv2.OIDCApplicationType, error) {
	switch t {
	case "", "Web":
		return apiv2.OIDCApplicationType_OIDC_APP_TYPE_WEB, nil
	case "UserAgent":
		return apiv2.OIDCApplicationType_OIDC_APP_TYPE_USER_AGENT, nil
	case "Native":
		return apiv2.OIDCApplicationType_OIDC_APP_TYPE_NATIVE, nil
	default:
		return apiv2.OIDCApplicationType_OIDC_APP_TYPE_WEB, fmt.Errorf("unsupported OIDC application type %q", t)
	}
}

// OIDCApplicationTypeFromProto maps the protobuf OIDC application type onto the
// API representation.
func OIDCApplicationTypeFromProto(t apiv2.OIDCApplicationType) string {
	switch t {
	case apiv2.OIDCApplicationType_OIDC_APP_TYPE_USER_AGENT:
		return "UserAgent"
	case apiv2.OIDCApplicationType_OIDC_APP_TYPE_NATIVE:
		return "Native"
	case apiv2.OIDCApplicationType_OIDC_APP_TYPE_WEB:
		return "Web"
	}

	return "Web"
}

// OIDCAuthMethodTypeToProto maps an API OIDC auth method type onto the protobuf
// representation.
func OIDCAuthMethodTypeToProto(t string) (apiv2.OIDCAuthMethodType, error) {
	switch t {
	case "", "Basic":
		return apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC, nil
	case "Post":
		return apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_POST, nil
	case "None":
		return apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_NONE, nil
	case "PrivateKeyJwt":
		return apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_PRIVATE_KEY_JWT, nil
	default:
		return apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC, fmt.Errorf("unsupported OIDC auth method type %q", t)
	}
}

// OIDCAuthMethodTypeFromProto maps the protobuf OIDC auth method type onto the
// API representation.
func OIDCAuthMethodTypeFromProto(t apiv2.OIDCAuthMethodType) string {
	switch t {
	case apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_POST:
		return "Post"
	case apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_NONE:
		return "None"
	case apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_PRIVATE_KEY_JWT:
		return "PrivateKeyJwt"
	case apiv2.OIDCAuthMethodType_OIDC_AUTH_METHOD_TYPE_BASIC:
		return "Basic"
	}

	return "Basic"
}

// OIDCResponseTypeToProto maps an API OIDC response type onto the protobuf
// representation.
func OIDCResponseTypeToProto(t string) (apiv2.OIDCResponseType, error) {
	switch t {
	case "Code":
		return apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE, nil
	case "IdToken":
		return apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_ID_TOKEN, nil
	case "IdTokenToken":
		return apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_ID_TOKEN_TOKEN, nil
	default:
		return apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_UNSPECIFIED, fmt.Errorf("unsupported OIDC response type %q", t)
	}
}

// OIDCResponseTypeFromProto maps the protobuf OIDC response type onto the API
// representation.
func OIDCResponseTypeFromProto(t apiv2.OIDCResponseType) string {
	switch t {
	case apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_CODE:
		return "Code"
	case apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_ID_TOKEN:
		return "IdToken"
	case apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_ID_TOKEN_TOKEN:
		return "IdTokenToken"
	case apiv2.OIDCResponseType_OIDC_RESPONSE_TYPE_UNSPECIFIED:
		return ""
	}

	return ""
}

// OIDCGrantTypeToProto maps an API OIDC grant type onto the protobuf
// representation.
func OIDCGrantTypeToProto(t string) (apiv2.OIDCGrantType, error) {
	switch t {
	case "AuthorizationCode":
		return apiv2.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE, nil
	case "Implicit":
		return apiv2.OIDCGrantType_OIDC_GRANT_TYPE_IMPLICIT, nil
	case "RefreshToken":
		return apiv2.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN, nil
	case "DeviceCode":
		return apiv2.OIDCGrantType_OIDC_GRANT_TYPE_DEVICE_CODE, nil
	case "TokenExchange":
		return apiv2.OIDCGrantType_OIDC_GRANT_TYPE_TOKEN_EXCHANGE, nil
	default:
		return apiv2.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE, fmt.Errorf("unsupported OIDC grant type %q", t)
	}
}

// OIDCGrantTypeFromProto maps the protobuf OIDC grant type onto the API
// representation.
func OIDCGrantTypeFromProto(t apiv2.OIDCGrantType) string {
	switch t {
	case apiv2.OIDCGrantType_OIDC_GRANT_TYPE_AUTHORIZATION_CODE:
		return "AuthorizationCode"
	case apiv2.OIDCGrantType_OIDC_GRANT_TYPE_IMPLICIT:
		return "Implicit"
	case apiv2.OIDCGrantType_OIDC_GRANT_TYPE_REFRESH_TOKEN:
		return "RefreshToken"
	case apiv2.OIDCGrantType_OIDC_GRANT_TYPE_DEVICE_CODE:
		return "DeviceCode"
	case apiv2.OIDCGrantType_OIDC_GRANT_TYPE_TOKEN_EXCHANGE:
		return "TokenExchange"
	}

	return ""
}

// OIDCTokenTypeToProto maps an API OIDC token type onto the protobuf
// representation.
func OIDCTokenTypeToProto(t string) (apiv2.OIDCTokenType, error) {
	switch t {
	case "", TokenTypeBearer:
		return apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER, nil
	case TokenTypeJwt, "JWT":
		return apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_JWT, nil
	default:
		return apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_BEARER, fmt.Errorf("unsupported OIDC token type %q", t)
	}
}

// OIDCTokenTypeFromProto maps the protobuf OIDC token type onto the API
// representation.
func OIDCTokenTypeFromProto(t apiv2.OIDCTokenType) string {
	if t == apiv2.OIDCTokenType_OIDC_TOKEN_TYPE_JWT {
		return TokenTypeJwt
	}

	return TokenTypeBearer
}

// OIDCVersionToProto maps an API OIDC version onto the protobuf
// representation.
func OIDCVersionToProto(v string) (apiv2.OIDCVersion, error) {
	switch v {
	case "", "1.0":
		return apiv2.OIDCVersion_OIDC_VERSION_1_0, nil
	default:
		return apiv2.OIDCVersion_OIDC_VERSION_1_0, fmt.Errorf("unsupported OIDC version %q", v)
	}
}
