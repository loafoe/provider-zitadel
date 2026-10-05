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
	"sync"
	"testing"

	adminv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	memberv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/member"
	objectv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/object"
	policyv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/policy"
	settingsv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/settings"
	userv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// A fake ZITADEL admin API.
//
// The instance wide policies are the widest thing this provider reads: ten of
// them, each a get and an update, none of which exists without an instance to
// read them from. Testing them through a real gRPC server rather than a stubbed
// client is what proves the request actually carries the fields the manifest
// asked for, which is the only place those fields are written down.
//
// Only the RPCs the tests drive are implemented. Everything else answers
// Unimplemented through the embedded struct, which is also what a ZITADEL
// without that surface does.

// fakeAdmin implements the instance wide policy RPCs.
type fakeAdmin struct {
	adminv1.UnimplementedAdminServiceServer

	mu sync.Mutex

	lockout            *policyv1.LockoutPolicy
	notification       *policyv1.NotificationPolicy
	passwordAge        *policyv1.PasswordAgePolicy
	passwordComplexity *policyv1.PasswordComplexityPolicy
	privacy            *policyv1.PrivacyPolicy
	domain             *policyv1.DomainPolicy
	label              *policyv1.LabelPolicy
	oidcSettings       *settingsv1.OIDCSettings

	// getErr is what every read fails with, nil meaning it reports what is held.
	getErr error

	// updateErr is what every write fails with, nil meaning it succeeds.
	updateErr error

	// instanceMembers is the membership list, and instanceRoles what the
	// instance offers.
	instanceMembers []*memberv1.Member
	instanceRoles   []string

	// restrictions is the last value written, and generators holds one per type:
	// there is one of each, so they are stored rather than overwritten.
	restrictions *adminv1.SetRestrictionsRequest
	generators   map[settingsv1.SecretGeneratorType]*adminv1.UpdateSecretGeneratorRequest

	// writes records the requests the client sent, so a test can check that what
	// the manifest asked for is what ZITADEL was told.
	writes []string
}

func newFakeAdmin() *fakeAdmin {
	return &fakeAdmin{generators: map[settingsv1.SecretGeneratorType]*adminv1.UpdateSecretGeneratorRequest{}}
}

func (f *fakeAdmin) record(call string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.writes = append(f.writes, call)
}

func (f *fakeAdmin) lastWrite() string {
	f.mu.Lock()
	defer f.mu.Unlock()

	if len(f.writes) == 0 {
		return ""
	}

	return f.writes[len(f.writes)-1]
}

func (f *fakeAdmin) readErr() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.getErr
}

func (f *fakeAdmin) writeFailure() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.updateErr
}

func (f *fakeAdmin) GetLockoutPolicy(_ context.Context, _ *adminv1.GetLockoutPolicyRequest) (*adminv1.GetLockoutPolicyResponse, error) {
	f.record("get-lockout")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return &adminv1.GetLockoutPolicyResponse{Policy: f.lockout}, nil
}

func (f *fakeAdmin) UpdateLockoutPolicy(_ context.Context, in *adminv1.UpdateLockoutPolicyRequest) (*adminv1.UpdateLockoutPolicyResponse, error) {
	f.record("update-lockout")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.lockout = &policyv1.LockoutPolicy{
		MaxPasswordAttempts: uint64(in.GetMaxPasswordAttempts()),
		MaxOtpAttempts:      uint64(in.GetMaxOtpAttempts()),
	}

	return &adminv1.UpdateLockoutPolicyResponse{}, nil
}

func (f *fakeAdmin) GetNotificationPolicy(_ context.Context, _ *adminv1.GetNotificationPolicyRequest) (*adminv1.GetNotificationPolicyResponse, error) {
	f.record("get-notification")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return &adminv1.GetNotificationPolicyResponse{Policy: f.notification}, nil
}

func (f *fakeAdmin) UpdateNotificationPolicy(_ context.Context, in *adminv1.UpdateNotificationPolicyRequest) (*adminv1.UpdateNotificationPolicyResponse, error) {
	f.record("update-notification")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.notification = &policyv1.NotificationPolicy{PasswordChange: in.GetPasswordChange()}

	return &adminv1.UpdateNotificationPolicyResponse{}, nil
}

func (f *fakeAdmin) GetPasswordAgePolicy(_ context.Context, _ *adminv1.GetPasswordAgePolicyRequest) (*adminv1.GetPasswordAgePolicyResponse, error) {
	f.record("get-password-age")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return &adminv1.GetPasswordAgePolicyResponse{Policy: f.passwordAge}, nil
}

func (f *fakeAdmin) UpdatePasswordAgePolicy(_ context.Context, in *adminv1.UpdatePasswordAgePolicyRequest) (*adminv1.UpdatePasswordAgePolicyResponse, error) {
	f.record("update-password-age")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.passwordAge = &policyv1.PasswordAgePolicy{
		MaxAgeDays:     uint64(in.GetMaxAgeDays()),
		ExpireWarnDays: uint64(in.GetExpireWarnDays()),
	}

	return &adminv1.UpdatePasswordAgePolicyResponse{}, nil
}

func (f *fakeAdmin) GetPasswordComplexityPolicy(_ context.Context, _ *adminv1.GetPasswordComplexityPolicyRequest) (*adminv1.GetPasswordComplexityPolicyResponse, error) {
	f.record("get-password-complexity")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return &adminv1.GetPasswordComplexityPolicyResponse{Policy: f.passwordComplexity}, nil
}

func (f *fakeAdmin) UpdatePasswordComplexityPolicy(_ context.Context, in *adminv1.UpdatePasswordComplexityPolicyRequest) (*adminv1.UpdatePasswordComplexityPolicyResponse, error) {
	f.record("update-password-complexity")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.passwordComplexity = &policyv1.PasswordComplexityPolicy{
		MinLength:    uint64(in.GetMinLength()),
		HasUppercase: in.GetHasUppercase(),
		HasLowercase: in.GetHasLowercase(),
		HasNumber:    in.GetHasNumber(),
		HasSymbol:    in.GetHasSymbol(),
	}

	return &adminv1.UpdatePasswordComplexityPolicyResponse{}, nil
}

func (f *fakeAdmin) GetPrivacyPolicy(_ context.Context, _ *adminv1.GetPrivacyPolicyRequest) (*adminv1.GetPrivacyPolicyResponse, error) {
	f.record("get-privacy")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return &adminv1.GetPrivacyPolicyResponse{Policy: f.privacy}, nil
}

func (f *fakeAdmin) UpdatePrivacyPolicy(_ context.Context, in *adminv1.UpdatePrivacyPolicyRequest) (*adminv1.UpdatePrivacyPolicyResponse, error) {
	f.record("update-privacy")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.privacy = &policyv1.PrivacyPolicy{
		TosLink:        in.GetTosLink(),
		PrivacyLink:    in.GetPrivacyLink(),
		HelpLink:       in.GetHelpLink(),
		SupportEmail:   in.GetSupportEmail(),
		DocsLink:       in.GetDocsLink(),
		CustomLink:     in.GetCustomLink(),
		CustomLinkText: in.GetCustomLinkText(),
	}

	return &adminv1.UpdatePrivacyPolicyResponse{}, nil
}

func (f *fakeAdmin) GetDomainPolicy(_ context.Context, _ *adminv1.GetDomainPolicyRequest) (*adminv1.GetDomainPolicyResponse, error) {
	f.record("get-domain")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return &adminv1.GetDomainPolicyResponse{Policy: f.domain}, nil
}

func (f *fakeAdmin) UpdateDomainPolicy(_ context.Context, in *adminv1.UpdateDomainPolicyRequest) (*adminv1.UpdateDomainPolicyResponse, error) {
	f.record("update-domain")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.domain = &policyv1.DomainPolicy{
		UserLoginMustBeDomain:                  in.GetUserLoginMustBeDomain(),
		ValidateOrgDomains:                     in.GetValidateOrgDomains(),
		SmtpSenderAddressMatchesInstanceDomain: in.GetSmtpSenderAddressMatchesInstanceDomain(),
	}

	return &adminv1.UpdateDomainPolicyResponse{}, nil
}

func (f *fakeAdmin) GetLabelPolicy(_ context.Context, _ *adminv1.GetLabelPolicyRequest) (*adminv1.GetLabelPolicyResponse, error) {
	f.record("get-label")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return &adminv1.GetLabelPolicyResponse{Policy: f.label}, nil
}

func (f *fakeAdmin) UpdateLabelPolicy(_ context.Context, in *adminv1.UpdateLabelPolicyRequest) (*adminv1.UpdateLabelPolicyResponse, error) {
	f.record("update-label")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.label = &policyv1.LabelPolicy{
		PrimaryColor:        in.GetPrimaryColor(),
		WarnColor:           in.GetWarnColor(),
		BackgroundColor:     in.GetBackgroundColor(),
		FontColor:           in.GetFontColor(),
		PrimaryColorDark:    in.GetPrimaryColorDark(),
		WarnColorDark:       in.GetWarnColorDark(),
		BackgroundColorDark: in.GetBackgroundColorDark(),
		FontColorDark:       in.GetFontColorDark(),
		HideLoginNameSuffix: in.GetHideLoginNameSuffix(),
		DisableWatermark:    in.GetDisableWatermark(),
		ThemeMode:           in.GetThemeMode(),
	}

	return &adminv1.UpdateLabelPolicyResponse{}, nil
}

func (f *fakeAdmin) GetOIDCSettings(_ context.Context, _ *adminv1.GetOIDCSettingsRequest) (*adminv1.GetOIDCSettingsResponse, error) {
	f.record("get-oidc-settings")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return &adminv1.GetOIDCSettingsResponse{Settings: f.oidcSettings}, nil
}

func (f *fakeAdmin) UpdateOIDCSettings(_ context.Context, in *adminv1.UpdateOIDCSettingsRequest) (*adminv1.UpdateOIDCSettingsResponse, error) {
	f.record("update-oidc-settings")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.oidcSettings = &settingsv1.OIDCSettings{
		AccessTokenLifetime:        in.GetAccessTokenLifetime(),
		IdTokenLifetime:            in.GetIdTokenLifetime(),
		RefreshTokenExpiration:     in.GetRefreshTokenExpiration(),
		RefreshTokenIdleExpiration: in.GetRefreshTokenIdleExpiration(),
	}

	return &adminv1.UpdateOIDCSettingsResponse{}, nil
}

// newAdminClient starts a fake serving the admin API and returns a client for it.
func newAdminClient(t *testing.T) (*Client, *fakeAdmin) {
	t.Helper()

	a := newFakeAdmin()

	fake := newFakeZitadel(t, func(s *grpc.Server) {
		adminv1.RegisterAdminServiceServer(s, a)
	})

	return fake.client(t), a
}

// notFound is what a ZITADEL without the surface answers with.
var notFound = status.Error(codes.NotFound, "not found")

// noChanges is what ZITADEL answers when a write would change nothing.
var noChanges = status.Error(codes.FailedPrecondition, "No changes")

// The registration restrictions and the secret generators, which are also
// instance wide and also have no v2 equivalent.

func (f *fakeAdmin) GetRestrictions(_ context.Context, _ *adminv1.GetRestrictionsRequest) (*adminv1.GetRestrictionsResponse, error) {
	f.record("get-restrictions")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.restrictions == nil {
		return &adminv1.GetRestrictionsResponse{}, nil
	}

	return &adminv1.GetRestrictionsResponse{
		DisallowPublicOrgRegistration: f.restrictions.GetDisallowPublicOrgRegistration(),
		AllowedLanguages:              f.restrictions.GetAllowedLanguages().GetList(),
	}, nil
}

func (f *fakeAdmin) SetRestrictions(_ context.Context, in *adminv1.SetRestrictionsRequest) (*adminv1.SetRestrictionsResponse, error) {
	f.record("set-restrictions")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.restrictions = in

	return &adminv1.SetRestrictionsResponse{}, nil
}

func (f *fakeAdmin) GetSecretGenerator(_ context.Context, in *adminv1.GetSecretGeneratorRequest) (*adminv1.GetSecretGeneratorResponse, error) { //nolint:revive // the generated signature is what it is
	f.record("get-secret-generator")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	stored := f.generator(in.GetGeneratorType())
	if stored == nil {
		return &adminv1.GetSecretGeneratorResponse{}, nil
	}

	return &adminv1.GetSecretGeneratorResponse{SecretGenerator: &settingsv1.SecretGenerator{
		GeneratorType:       stored.GetGeneratorType(),
		Length:              stored.GetLength(),
		Expiry:              stored.GetExpiry(),
		IncludeLowerLetters: stored.GetIncludeLowerLetters(),
		IncludeUpperLetters: stored.GetIncludeUpperLetters(),
		IncludeDigits:       stored.GetIncludeDigits(),
		IncludeSymbols:      stored.GetIncludeSymbols(),
	}}, nil
}

func (f *fakeAdmin) UpdateSecretGenerator(_ context.Context, in *adminv1.UpdateSecretGeneratorRequest) (*adminv1.UpdateSecretGeneratorResponse, error) { //nolint:revive // the generated signature is what it is
	f.record("update-secret-generator")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.generators[in.GetGeneratorType()] = in

	return &adminv1.UpdateSecretGeneratorResponse{}, nil
}

// generator is the stored generator of a type, or nil when none was written.
func (f *fakeAdmin) generator(t settingsv1.SecretGeneratorType) *adminv1.UpdateSecretGeneratorRequest {
	return f.generators[t]
}

// The instance level memberships, which are the only memberships ZITADEL has no
// v2 API for.

func (f *fakeAdmin) ListIAMMemberRoles(_ context.Context, _ *adminv1.ListIAMMemberRolesRequest) (*adminv1.ListIAMMemberRolesResponse, error) {
	f.record("list-instance-roles")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return &adminv1.ListIAMMemberRolesResponse{Roles: f.instanceRoles}, nil
}

func (f *fakeAdmin) ListIAMMembers(_ context.Context, _ *adminv1.ListIAMMembersRequest) (*adminv1.ListIAMMembersResponse, error) {
	f.record("list-instance-members")

	if err := f.readErr(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return &adminv1.ListIAMMembersResponse{Result: f.instanceMembers}, nil
}

func (f *fakeAdmin) AddIAMMember(_ context.Context, in *adminv1.AddIAMMemberRequest) (*adminv1.AddIAMMemberResponse, error) {
	f.record("add-instance-member")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.instanceMembers = upsertMember(f.instanceMembers, in.GetUserId(), in.GetRoles())

	return &adminv1.AddIAMMemberResponse{}, nil
}

func (f *fakeAdmin) UpdateIAMMember(_ context.Context, in *adminv1.UpdateIAMMemberRequest) (*adminv1.UpdateIAMMemberResponse, error) {
	f.record("update-instance-member")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.instanceMembers = upsertMember(f.instanceMembers, in.GetUserId(), in.GetRoles())

	return &adminv1.UpdateIAMMemberResponse{}, nil
}

func (f *fakeAdmin) RemoveIAMMember(_ context.Context, in *adminv1.RemoveIAMMemberRequest) (*adminv1.RemoveIAMMemberResponse, error) {
	f.record("remove-instance-member")

	if err := f.writeFailure(); err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.instanceMembers = dropMember(f.instanceMembers, in.GetUserId())

	return &adminv1.RemoveIAMMemberResponse{}, nil
}

// upsertMember is how the fake holds a membership: the roles of a user are
// replaced rather than appended, so a test can see what the client wrote.
func upsertMember(members []*memberv1.Member, userID string, roles []string) []*memberv1.Member {
	out := dropMember(members, userID)

	// The details are what the client uses to decide a membership has a user
	// type at all, so the fake sets them the way a real instance would.
	return append(out, &memberv1.Member{
		UserId:             userID,
		Roles:              roles,
		Details:            &objectv1.ObjectDetails{},
		UserType:           userv1.Type_TYPE_HUMAN,
		PreferredLoginName: userID + "@example.com",
		Email:              userID + "@example.com",
		DisplayName:        "Member " + userID,
	})
}

// dropMember removes a user from a membership list.
func dropMember(members []*memberv1.Member, userID string) []*memberv1.Member {
	out := make([]*memberv1.Member, 0, len(members))
	for _, m := range members {
		if m.GetUserId() != userID {
			out = append(out, m)
		}
	}

	return out
}
