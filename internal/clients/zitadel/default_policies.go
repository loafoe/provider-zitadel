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

	adminv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	policyv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/policy"
	settingsapi "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/settings/v2"
)

// Instance wide policies
//
// These are the policies an organization inherits until it customises one, and
// they are what most installations never touch: a sensible default set once, at
// the instance level, rather than per organization.
//
// Two things set them apart from an organization's own policies:
//
//   - there is only one of each, so there is no scope to resolve and nothing to
//     key a resource on;
//   - Zitadel offers no way to reset one. There is no default to go back to -
//     the default *is* this - so an instance policy cannot be unapplied the way
//     an organization's can. The managed resource therefore records the value it
//     overwrote and writes it back when it is deleted.

// passwordlessNotAllowed is the API spelling of the restrictive passwordless
// setting. Zitadel rejects the protobuf zero value rather than reading it as
// "not allowed", so an unset one has to be spelled out.
const passwordlessNotAllowed = "NotAllowed"

// DefaultLockoutPolicy is the instance wide lockout policy.
type DefaultLockoutPolicy struct {
	MaxPasswordAttempts uint32
	MaxOTPAttempts      uint32
}

// DefaultLockoutPolicyInput is the desired instance wide lockout policy.
type DefaultLockoutPolicyInput struct {
	MaxPasswordAttempts uint32
	MaxOTPAttempts      uint32
}

// Input returns the policy in the shape the write API accepts.
func (p DefaultLockoutPolicy) Input() any {
	return DefaultLockoutPolicyInput(p)
}

// GetDefaultLockoutPolicy returns the instance wide lockout policy.
func (c *Client) GetDefaultLockoutPolicy(ctx context.Context) (*DefaultLockoutPolicy, error) {
	resp, err := c.admin.GetLockoutPolicy(ctx, &adminv1.GetLockoutPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the instance lockout policy: %w", err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &DefaultLockoutPolicy{
		MaxPasswordAttempts: boundedUint32(p.GetMaxPasswordAttempts()),
		MaxOTPAttempts:      boundedUint32(p.GetMaxOtpAttempts()),
	}, nil
}

// SetDefaultLockoutPolicy writes the instance wide lockout policy.
//
// There is no add: an instance policy always exists, so writing one is always an
// update.
func (c *Client) SetDefaultLockoutPolicy(ctx context.Context, in DefaultLockoutPolicyInput) error {
	if _, err := c.admin.UpdateLockoutPolicy(ctx, &adminv1.UpdateLockoutPolicyRequest{
		MaxPasswordAttempts: in.MaxPasswordAttempts,
		MaxOtpAttempts:      in.MaxOTPAttempts,
	}); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the instance lockout policy: %w", err)
	}

	return nil
}

// DefaultNotificationPolicy is the instance wide notification policy.
type DefaultNotificationPolicy struct {
	PasswordChange bool
}

// DefaultNotificationPolicyInput is the desired instance wide notification
// policy.
type DefaultNotificationPolicyInput struct {
	PasswordChange bool
}

// Input returns the policy in the shape the write API accepts.
func (p DefaultNotificationPolicy) Input() any {
	return DefaultNotificationPolicyInput(p)
}

// GetDefaultNotificationPolicy returns the instance wide notification policy.
func (c *Client) GetDefaultNotificationPolicy(ctx context.Context) (*DefaultNotificationPolicy, error) {
	resp, err := c.admin.GetNotificationPolicy(ctx, &adminv1.GetNotificationPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the instance notification policy: %w", err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &DefaultNotificationPolicy{PasswordChange: p.GetPasswordChange()}, nil
}

// SetDefaultNotificationPolicy writes the instance wide notification policy.
func (c *Client) SetDefaultNotificationPolicy(ctx context.Context, in DefaultNotificationPolicyInput) error {
	if _, err := c.admin.UpdateNotificationPolicy(ctx, &adminv1.UpdateNotificationPolicyRequest{
		PasswordChange: in.PasswordChange,
	}); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the instance notification policy: %w", err)
	}

	return nil
}

// DefaultPasswordAgePolicy is the instance wide password expiry policy.
type DefaultPasswordAgePolicy struct {
	MaxAgeDays     uint32
	ExpireWarnDays uint32
}

// DefaultPasswordAgePolicyInput is the desired instance wide password expiry
// policy.
type DefaultPasswordAgePolicyInput struct {
	MaxAgeDays     uint32
	ExpireWarnDays uint32
}

// Input returns the policy in the shape the write API accepts.
func (p DefaultPasswordAgePolicy) Input() any {
	return DefaultPasswordAgePolicyInput(p)
}

// GetDefaultPasswordAgePolicy returns the instance wide password expiry policy.
func (c *Client) GetDefaultPasswordAgePolicy(ctx context.Context) (*DefaultPasswordAgePolicy, error) {
	resp, err := c.admin.GetPasswordAgePolicy(ctx, &adminv1.GetPasswordAgePolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the instance password age policy: %w", err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &DefaultPasswordAgePolicy{
		MaxAgeDays:     boundedUint32(p.GetMaxAgeDays()),
		ExpireWarnDays: boundedUint32(p.GetExpireWarnDays()),
	}, nil
}

// SetDefaultPasswordAgePolicy writes the instance wide password expiry policy.
func (c *Client) SetDefaultPasswordAgePolicy(ctx context.Context, in DefaultPasswordAgePolicyInput) error {
	if _, err := c.admin.UpdatePasswordAgePolicy(ctx, &adminv1.UpdatePasswordAgePolicyRequest{
		MaxAgeDays:     in.MaxAgeDays,
		ExpireWarnDays: in.ExpireWarnDays,
	}); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the instance password age policy: %w", err)
	}

	return nil
}

// DefaultPasswordComplexityPolicy is the instance wide password complexity
// policy.
type DefaultPasswordComplexityPolicy struct {
	MinLength    uint32
	HasUppercase bool
	HasLowercase bool
	HasNumber    bool
	HasSymbol    bool
}

// DefaultPasswordComplexityPolicyInput is the desired instance wide password
// complexity policy.
type DefaultPasswordComplexityPolicyInput struct {
	MinLength    uint32
	HasUppercase bool
	HasLowercase bool
	HasNumber    bool
	HasSymbol    bool
}

// Input returns the policy in the shape the write API accepts.
func (p DefaultPasswordComplexityPolicy) Input() any {
	return DefaultPasswordComplexityPolicyInput(p)
}

// GetDefaultPasswordComplexityPolicy returns the instance wide password
// complexity policy.
func (c *Client) GetDefaultPasswordComplexityPolicy(ctx context.Context) (*DefaultPasswordComplexityPolicy, error) {
	resp, err := c.admin.GetPasswordComplexityPolicy(ctx, &adminv1.GetPasswordComplexityPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the instance password complexity policy: %w", err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &DefaultPasswordComplexityPolicy{
		MinLength:    boundedUint32(p.GetMinLength()),
		HasUppercase: p.GetHasUppercase(),
		HasLowercase: p.GetHasLowercase(),
		HasNumber:    p.GetHasNumber(),
		HasSymbol:    p.GetHasSymbol(),
	}, nil
}

// SetDefaultPasswordComplexityPolicy writes the instance wide password
// complexity policy.
func (c *Client) SetDefaultPasswordComplexityPolicy(ctx context.Context, in DefaultPasswordComplexityPolicyInput) error {
	if _, err := c.admin.UpdatePasswordComplexityPolicy(ctx, &adminv1.UpdatePasswordComplexityPolicyRequest{
		MinLength:    in.MinLength,
		HasUppercase: in.HasUppercase,
		HasLowercase: in.HasLowercase,
		HasNumber:    in.HasNumber,
		HasSymbol:    in.HasSymbol,
	}); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the instance password complexity policy: %w", err)
	}

	return nil
}

// DefaultPrivacyPolicy is the instance wide privacy policy: the links and
// contact details Zitadel shows when an organization has set none of its own.
type DefaultPrivacyPolicy struct {
	TOSLink        string
	PrivacyLink    string
	HelpLink       string
	SupportEmail   string
	DocsLink       string
	CustomLink     string
	CustomLinkText string
}

// DefaultPrivacyPolicyInput is the desired instance wide privacy policy.
type DefaultPrivacyPolicyInput struct {
	TOSLink        string
	PrivacyLink    string
	HelpLink       string
	SupportEmail   string
	DocsLink       string
	CustomLink     string
	CustomLinkText string
}

// Input returns the policy in the shape the write API accepts.
func (p DefaultPrivacyPolicy) Input() any {
	return DefaultPrivacyPolicyInput(p)
}

// GetDefaultPrivacyPolicy returns the instance wide privacy policy.
func (c *Client) GetDefaultPrivacyPolicy(ctx context.Context) (*DefaultPrivacyPolicy, error) {
	resp, err := c.admin.GetPrivacyPolicy(ctx, &adminv1.GetPrivacyPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the instance privacy policy: %w", err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &DefaultPrivacyPolicy{
		TOSLink:        p.GetTosLink(),
		PrivacyLink:    p.GetPrivacyLink(),
		HelpLink:       p.GetHelpLink(),
		SupportEmail:   p.GetSupportEmail(),
		DocsLink:       p.GetDocsLink(),
		CustomLink:     p.GetCustomLink(),
		CustomLinkText: p.GetCustomLinkText(),
	}, nil
}

// SetDefaultPrivacyPolicy writes the instance wide privacy policy.
func (c *Client) SetDefaultPrivacyPolicy(ctx context.Context, in DefaultPrivacyPolicyInput) error {
	if _, err := c.admin.UpdatePrivacyPolicy(ctx, &adminv1.UpdatePrivacyPolicyRequest{
		TosLink:        in.TOSLink,
		PrivacyLink:    in.PrivacyLink,
		HelpLink:       in.HelpLink,
		SupportEmail:   in.SupportEmail,
		DocsLink:       in.DocsLink,
		CustomLink:     in.CustomLink,
		CustomLinkText: in.CustomLinkText,
	}); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the instance privacy policy: %w", err)
	}

	return nil
}

// DefaultDomainPolicy is the instance wide domain policy.
type DefaultDomainPolicy struct {
	UserLoginMustBeDomain                  bool
	ValidateOrgDomains                     bool
	SMTPSenderAddressMatchesInstanceDomain bool
}

// DefaultDomainPolicyInput is the desired instance wide domain policy.
type DefaultDomainPolicyInput struct {
	UserLoginMustBeDomain                  bool
	ValidateOrgDomains                     bool
	SMTPSenderAddressMatchesInstanceDomain bool
}

// Input returns the policy in the shape the write API accepts.
func (p DefaultDomainPolicy) Input() any {
	return DefaultDomainPolicyInput(p)
}

// GetDefaultDomainPolicy returns the instance wide domain policy.
func (c *Client) GetDefaultDomainPolicy(ctx context.Context) (*DefaultDomainPolicy, error) {
	resp, err := c.admin.GetDomainPolicy(ctx, &adminv1.GetDomainPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the instance domain policy: %w", err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &DefaultDomainPolicy{
		UserLoginMustBeDomain:                  p.GetUserLoginMustBeDomain(),
		ValidateOrgDomains:                     p.GetValidateOrgDomains(),
		SMTPSenderAddressMatchesInstanceDomain: p.GetSmtpSenderAddressMatchesInstanceDomain(),
	}, nil
}

// SetDefaultDomainPolicy writes the instance wide domain policy.
func (c *Client) SetDefaultDomainPolicy(ctx context.Context, in DefaultDomainPolicyInput) error {
	if _, err := c.admin.UpdateDomainPolicy(ctx, &adminv1.UpdateDomainPolicyRequest{
		UserLoginMustBeDomain:                  in.UserLoginMustBeDomain,
		ValidateOrgDomains:                     in.ValidateOrgDomains,
		SmtpSenderAddressMatchesInstanceDomain: in.SMTPSenderAddressMatchesInstanceDomain,
	}); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the instance domain policy: %w", err)
	}

	return nil
}

// DefaultLabelPolicy is the instance wide branding policy.
type DefaultLabelPolicy struct {
	PrimaryColor        string
	WarnColor           string
	BackgroundColor     string
	FontColor           string
	PrimaryColorDark    string
	WarnColorDark       string
	BackgroundColorDark string
	FontColorDark       string
	HideLoginNameSuffix bool
	DisableWatermark    bool
	ThemeMode           string

	// AssetURLs are where Zitadel serves the branding assets from. They are
	// reported rather than managed: Zitadel uploads assets over a plain HTTP
	// endpoint outside its API, so this provider neither sets nor removes them.
	LogoURL     string
	IconURL     string
	LogoDarkURL string
	IconDarkURL string
	FontURL     string
}

// DefaultLabelPolicyInput is the desired instance wide branding policy.
type DefaultLabelPolicyInput struct {
	PrimaryColor        string
	WarnColor           string
	BackgroundColor     string
	FontColor           string
	PrimaryColorDark    string
	WarnColorDark       string
	BackgroundColorDark string
	FontColorDark       string
	HideLoginNameSuffix bool
	DisableWatermark    bool
	ThemeMode           string
}

// Input returns the policy in the shape the write API accepts.
func (p DefaultLabelPolicy) Input() any {
	return DefaultLabelPolicyInput{
		PrimaryColor:        p.PrimaryColor,
		WarnColor:           p.WarnColor,
		BackgroundColor:     p.BackgroundColor,
		FontColor:           p.FontColor,
		PrimaryColorDark:    p.PrimaryColorDark,
		WarnColorDark:       p.WarnColorDark,
		BackgroundColorDark: p.BackgroundColorDark,
		FontColorDark:       p.FontColorDark,
		HideLoginNameSuffix: p.HideLoginNameSuffix,
		DisableWatermark:    p.DisableWatermark,
		ThemeMode:           p.ThemeMode,
	}
}

// GetDefaultLabelPolicy returns the instance wide branding policy.
func (c *Client) GetDefaultLabelPolicy(ctx context.Context) (*DefaultLabelPolicy, error) {
	resp, err := c.admin.GetLabelPolicy(ctx, &adminv1.GetLabelPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the instance label policy: %w", err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &DefaultLabelPolicy{
		PrimaryColor:        p.GetPrimaryColor(),
		WarnColor:           p.GetWarnColor(),
		BackgroundColor:     p.GetBackgroundColor(),
		FontColor:           p.GetFontColor(),
		PrimaryColorDark:    p.GetPrimaryColorDark(),
		WarnColorDark:       p.GetWarnColorDark(),
		BackgroundColorDark: p.GetBackgroundColorDark(),
		FontColorDark:       p.GetFontColorDark(),
		HideLoginNameSuffix: p.GetHideLoginNameSuffix(),
		DisableWatermark:    p.GetDisableWatermark(),
		ThemeMode:           LabelThemeModeFromProto(p.GetThemeMode()),
		LogoURL:             p.GetLogoUrl(),
		IconURL:             p.GetIconUrl(),
		LogoDarkURL:         p.GetLogoUrlDark(),
		IconDarkURL:         p.GetIconUrlDark(),
		FontURL:             p.GetFontUrl(),
	}, nil
}

// SetDefaultLabelPolicy writes the instance wide branding policy.
func (c *Client) SetDefaultLabelPolicy(ctx context.Context, in DefaultLabelPolicyInput) error {
	mode, err := LabelThemeModeToProto(in.ThemeMode)
	if err != nil {
		return err
	}

	if _, err := c.admin.UpdateLabelPolicy(ctx, &adminv1.UpdateLabelPolicyRequest{
		PrimaryColor:        in.PrimaryColor,
		HideLoginNameSuffix: in.HideLoginNameSuffix,
		WarnColor:           in.WarnColor,
		BackgroundColor:     in.BackgroundColor,
		FontColor:           in.FontColor,
		PrimaryColorDark:    in.PrimaryColorDark,
		BackgroundColorDark: in.BackgroundColorDark,
		WarnColorDark:       in.WarnColorDark,
		FontColorDark:       in.FontColorDark,
		DisableWatermark:    in.DisableWatermark,
		ThemeMode:           mode,
	}); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the instance label policy: %w", err)
	}

	return nil
}

// DefaultLoginPolicy is the instance wide login policy.
type DefaultLoginPolicy struct {
	AllowUsernamePassword  bool
	AllowRegister          bool
	AllowExternalIDP       bool
	ForceMFA               bool
	ForceMFALocalOnly      bool
	HidePasswordReset      bool
	IgnoreUnknownUsernames bool
	AllowDomainDiscovery   bool
	DisableLoginWithEmail  bool
	DisableLoginWithPhone  bool
	DefaultRedirectURI     string
	PasswordlessType       string

	PasswordCheckLifetime      string
	ExternalLoginCheckLifetime string
	MFAInitSkipLifetime        string
	SecondFactorCheckLifetime  string
	MultiFactorCheckLifetime   string

	SecondFactors []string
	MultiFactors  []string
}

// DefaultLoginPolicyInput is the desired instance wide login policy. Lifetimes
// are Go duration strings such as "24h".
type DefaultLoginPolicyInput struct {
	AllowUsernamePassword  bool
	AllowRegister          bool
	AllowExternalIDP       bool
	ForceMFA               bool
	ForceMFALocalOnly      bool
	HidePasswordReset      bool
	IgnoreUnknownUsernames bool
	AllowDomainDiscovery   bool
	DisableLoginWithEmail  bool
	DisableLoginWithPhone  bool
	DefaultRedirectURI     string
	PasswordlessType       string

	PasswordCheckLifetime      string
	ExternalLoginCheckLifetime string
	MFAInitSkipLifetime        string
	SecondFactorCheckLifetime  string
	MultiFactorCheckLifetime   string

	// SecondFactors and MultiFactors use the API spelling, e.g. "OTP", and are
	// translated to the protobuf enums here.
	SecondFactors []string
	MultiFactors  []string
}

// Input returns the policy in the shape the write API accepts.
func (p DefaultLoginPolicy) Input() any {
	in := DefaultLoginPolicyInput(p)

	// An unset passwordless type would otherwise be written as the protobuf zero
	// value, which Zitadel rejects rather than reading as "not allowed".
	if in.PasswordlessType == "" {
		in.PasswordlessType = passwordlessNotAllowed
	}

	return in
}

// GetDefaultLoginPolicy returns the instance wide login policy.
func (c *Client) GetDefaultLoginPolicy(ctx context.Context) (*DefaultLoginPolicy, error) {
	resp, err := c.admin.GetLoginPolicy(ctx, &adminv1.GetLoginPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the instance login policy: %w", err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	out := &DefaultLoginPolicy{
		AllowUsernamePassword:      p.GetAllowUsernamePassword(),
		AllowRegister:              p.GetAllowRegister(),
		AllowExternalIDP:           p.GetAllowExternalIdp(),
		ForceMFA:                   p.GetForceMfa(),
		ForceMFALocalOnly:          p.GetForceMfaLocalOnly(),
		HidePasswordReset:          p.GetHidePasswordReset(),
		IgnoreUnknownUsernames:     p.GetIgnoreUnknownUsernames(),
		AllowDomainDiscovery:       p.GetAllowDomainDiscovery(),
		DisableLoginWithEmail:      p.GetDisableLoginWithEmail(),
		DisableLoginWithPhone:      p.GetDisableLoginWithPhone(),
		DefaultRedirectURI:         p.GetDefaultRedirectUri(),
		PasswordlessType:           passwordlessTypeFromProto(p.GetPasswordlessType()),
		PasswordCheckLifetime:      durationString(p.GetPasswordCheckLifetime()),
		ExternalLoginCheckLifetime: durationString(p.GetExternalLoginCheckLifetime()),
		MFAInitSkipLifetime:        durationString(p.GetMfaInitSkipLifetime()),
		SecondFactorCheckLifetime:  durationString(p.GetSecondFactorCheckLifetime()),
		MultiFactorCheckLifetime:   durationString(p.GetMultiFactorCheckLifetime()),
	}

	secondFactors, err := SecondFactorNames(p.GetSecondFactors())
	if err != nil {
		return nil, err
	}

	multiFactors, err := MultiFactorNames(p.GetMultiFactors())
	if err != nil {
		return nil, err
	}

	out.SecondFactors = secondFactors
	out.MultiFactors = multiFactors

	return out, nil
}

// SetDefaultLoginPolicy writes the instance wide login policy.
//
// As with an organization's own login policy, the write endpoint does not accept
// the factor lists, so they are reconciled against the current policy by calling
// the dedicated add and remove endpoints. Zitadel offers no reset for an
// instance wide policy, so this only ever adds and removes.
func (c *Client) SetDefaultLoginPolicy(ctx context.Context, in DefaultLoginPolicyInput) error {
	if in.PasswordlessType == "" {
		in.PasswordlessType = passwordlessNotAllowed
	}

	ptype, err := passwordlessType(in.PasswordlessType)
	if err != nil {
		return err
	}

	if _, err := c.admin.UpdateLoginPolicy(ctx, &adminv1.UpdateLoginPolicyRequest{
		AllowUsernamePassword:      in.AllowUsernamePassword,
		AllowRegister:              in.AllowRegister,
		AllowExternalIdp:           in.AllowExternalIDP,
		ForceMfa:                   in.ForceMFA,
		ForceMfaLocalOnly:          in.ForceMFALocalOnly,
		HidePasswordReset:          in.HidePasswordReset,
		IgnoreUnknownUsernames:     in.IgnoreUnknownUsernames,
		AllowDomainDiscovery:       in.AllowDomainDiscovery,
		DisableLoginWithEmail:      in.DisableLoginWithEmail,
		DisableLoginWithPhone:      in.DisableLoginWithPhone,
		DefaultRedirectUri:         in.DefaultRedirectURI,
		PasswordlessType:           ptype,
		PasswordCheckLifetime:      durationProto(in.PasswordCheckLifetime),
		ExternalLoginCheckLifetime: durationProto(in.ExternalLoginCheckLifetime),
		MfaInitSkipLifetime:        durationProto(in.MFAInitSkipLifetime),
		SecondFactorCheckLifetime:  durationProto(in.SecondFactorCheckLifetime),
		MultiFactorCheckLifetime:   durationProto(in.MultiFactorCheckLifetime),
	}); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the instance login policy: %w", err)
	}

	return c.reconcileDefaultLoginPolicyFactors(ctx, in)
}

// DefaultOIDCSettings are the instance wide token lifetimes.
type DefaultOIDCSettings struct {
	AccessTokenLifetime        string
	IDTokenLifetime            string
	RefreshTokenExpiration     string
	RefreshTokenIdleExpiration string
}

// DefaultOIDCSettingsInput is the desired instance wide OIDC settings.
type DefaultOIDCSettingsInput struct {
	AccessTokenLifetime        string
	IDTokenLifetime            string
	RefreshTokenExpiration     string
	RefreshTokenIdleExpiration string
}

// Input returns the settings in the shape the write API accepts.
func (s DefaultOIDCSettings) Input() any {
	return DefaultOIDCSettingsInput(s)
}

// GetDefaultOIDCSettings returns the instance wide OIDC settings: how long the
// tokens Zitadel issues stay valid.
func (c *Client) GetDefaultOIDCSettings(ctx context.Context) (*DefaultOIDCSettings, error) {
	resp, err := c.admin.GetOIDCSettings(ctx, &adminv1.GetOIDCSettingsRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the instance OIDC settings: %w", err)
	}

	o := resp.GetSettings()

	return &DefaultOIDCSettings{
		AccessTokenLifetime:        durationString(o.GetAccessTokenLifetime()),
		IDTokenLifetime:            durationString(o.GetIdTokenLifetime()),
		RefreshTokenExpiration:     durationString(o.GetRefreshTokenExpiration()),
		RefreshTokenIdleExpiration: durationString(o.GetRefreshTokenIdleExpiration()),
	}, nil
}

// SetDefaultOIDCSettings writes the instance wide OIDC settings.
func (c *Client) SetDefaultOIDCSettings(ctx context.Context, in DefaultOIDCSettingsInput) error {
	if _, err := c.admin.UpdateOIDCSettings(ctx, &adminv1.UpdateOIDCSettingsRequest{
		AccessTokenLifetime:        durationProto(in.AccessTokenLifetime),
		IdTokenLifetime:            durationProto(in.IDTokenLifetime),
		RefreshTokenExpiration:     durationProto(in.RefreshTokenExpiration),
		RefreshTokenIdleExpiration: durationProto(in.RefreshTokenIdleExpiration),
	}); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the instance OIDC settings: %w", err)
	}

	return nil
}

// DefaultSecuritySettings are the instance wide security settings.
type DefaultSecuritySettings struct {
	EnableImpersonation bool
	EmbeddedIframe      bool
	AllowedOrigins      []string
}

// DefaultSecuritySettingsInput is the desired instance wide security settings.
type DefaultSecuritySettingsInput struct {
	EnableImpersonation bool
	EmbeddedIframe      bool
	AllowedOrigins      []string
}

// Input returns the settings in the shape the write API accepts.
func (s DefaultSecuritySettings) Input() any {
	return DefaultSecuritySettingsInput(s)
}

// GetDefaultSecuritySettings returns the instance wide security settings.
//
// These come from the v2 settings service rather than the v1 admin API, which is
// the only Zitadel surface that reads and writes them.
func (c *Client) GetDefaultSecuritySettings(ctx context.Context) (*DefaultSecuritySettings, error) {
	resp, err := c.settings.GetSecuritySettings(ctx, &settingsapi.GetSecuritySettingsRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the instance security settings: %w", err)
	}

	s := resp.GetSettings()
	if s == nil {
		return nil, ErrNotFound
	}

	out := &DefaultSecuritySettings{EnableImpersonation: s.GetEnableImpersonation()}

	if f := s.GetEmbeddedIframe(); f != nil {
		out.EmbeddedIframe = f.GetEnabled()
		out.AllowedOrigins = f.GetAllowedOrigins()
	}

	return out, nil
}

// SetDefaultSecuritySettings writes the instance wide security settings.
func (c *Client) SetDefaultSecuritySettings(ctx context.Context, in DefaultSecuritySettingsInput) error {
	req := &settingsapi.SetSecuritySettingsRequest{
		EnableImpersonation: in.EnableImpersonation,
		EmbeddedIframe: &settingsapi.EmbeddedIframeSettings{
			Enabled:        in.EmbeddedIframe,
			AllowedOrigins: in.AllowedOrigins,
		},
	}

	if _, err := c.settings.SetSecuritySettings(ctx, req); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the instance security settings: %w", err)
	}

	return nil
}

// reconcileDefaultLoginPolicyFactors applies the difference between the desired
// and the current second and multi factor lists.
//
// Zitadel has no call that writes a login policy's factor lists, so they are
// added and removed one at a time. Doing that by hand means comparing the lists
// in two directions and getting one of them backwards, which shows up as a
// factor being added when it was already there or removed when it was not.
func (c *Client) reconcileDefaultLoginPolicyFactors(ctx context.Context, in DefaultLoginPolicyInput) error {
	current, err := c.GetDefaultLoginPolicy(ctx)
	if err != nil {
		return err
	}

	if err := reconcileFactorList(ctx, "second factor", in.SecondFactors, current.SecondFactors,
		SecondFactorToProto,
		func(ctx context.Context, t policyv1.SecondFactorType) error {
			_, err := c.admin.AddSecondFactorToLoginPolicy(ctx, &adminv1.AddSecondFactorToLoginPolicyRequest{Type: t})
			return err
		},
		func(ctx context.Context, t policyv1.SecondFactorType) error {
			_, err := c.admin.RemoveSecondFactorFromLoginPolicy(ctx, &adminv1.RemoveSecondFactorFromLoginPolicyRequest{Type: t})
			return err
		},
	); err != nil {
		return err
	}

	return reconcileFactorList(ctx, "multi factor", in.MultiFactors, current.MultiFactors,
		MultiFactorToProto,
		func(ctx context.Context, t policyv1.MultiFactorType) error {
			_, err := c.admin.AddMultiFactorToLoginPolicy(ctx, &adminv1.AddMultiFactorToLoginPolicyRequest{Type: t})
			return err
		},
		func(ctx context.Context, t policyv1.MultiFactorType) error {
			_, err := c.admin.RemoveMultiFactorFromLoginPolicy(ctx, &adminv1.RemoveMultiFactorFromLoginPolicyRequest{Type: t})
			return err
		},
	)
}

// reconcileFactorList adds the factors that are wanted but missing and removes
// the ones that are present but no longer wanted.
func reconcileFactorList[T any](ctx context.Context, kind string, want, have []string,
	toProto func(string) (T, error),
	add func(context.Context, T) error,
	remove func(context.Context, T) error,
) error {
	wantList := normalize(want)

	for _, name := range difference(wantList, have) {
		t, err := toProto(name)
		if err != nil {
			return err
		}

		if err := add(ctx, t); err != nil {
			// The read that decides what to add is not always accurate: Zitadel
			// can report a factor as absent when the instance already has it, and
			// refuse the add as already existing. Having it is what was wanted,
			// so that is not a failure.
			if !IsAlreadyExists(err) {
				return fmt.Errorf("cannot add the %s %s: %w", kind, name, err)
			}
		}
	}

	for _, name := range difference(have, wantList) {
		t, err := toProto(name)
		if err != nil {
			return err
		}

		if err := remove(ctx, t); err != nil {
			// The same disagreement in the other direction: a factor reported as
			// present that the instance has already dropped is already gone.
			if !IsNotFound(err) {
				return fmt.Errorf("cannot remove the %s %s: %w", kind, name, err)
			}
		}
	}

	return nil
}
