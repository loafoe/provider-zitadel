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
	"math"

	adminv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	policyv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/policy"
)

// Organization policies
//
// Every policy here is a singleton of one organization, and every one of them
// follows the same lifecycle:
//
//   - the read answers even when nothing is customised, reporting the instance
//     default, so the response says whether the policy is inherited;
//   - writing is two calls, because Zitadel separates adding a custom policy
//     from updating one, and which to use depends on whether one exists;
//   - there is no delete, so removing the customisation means resetting it to
//     the instance default.
//
// That makes each policy a small, uniform amount of code, and it is why the
// policy types here mirror each other.

// LockoutPolicy is the lockout policy of an organization.
type LockoutPolicy struct {
	OrgID     string
	IsDefault bool

	MaxPasswordAttempts uint32
	MaxOTPAttempts      uint32
}

// LockoutPolicyInput is the desired lockout policy.
type LockoutPolicyInput struct {
	MaxPasswordAttempts uint32
	MaxOTPAttempts      uint32
}

// Input returns the policy in the shape the write API accepts.
func (p LockoutPolicy) Input() any {
	return LockoutPolicyInput{MaxPasswordAttempts: p.MaxPasswordAttempts, MaxOTPAttempts: p.MaxOTPAttempts}
}

// GetLockoutPolicy returns the lockout policy of an organization.
func (c *Client) GetLockoutPolicy(ctx context.Context, orgID string) (*LockoutPolicy, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.GetLockoutPolicy(ctx, &managementv1.GetLockoutPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the lockout policy of organization %s: %w", orgID, err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &LockoutPolicy{
		OrgID:               orgID,
		IsDefault:           p.GetIsDefault(),
		MaxPasswordAttempts: boundedUint32(p.GetMaxPasswordAttempts()),
		MaxOTPAttempts:      boundedUint32(p.GetMaxOtpAttempts()),
	}, nil
}

// SetLockoutPolicy customises or updates the lockout policy of an organization.
func (c *Client) SetLockoutPolicy(ctx context.Context, orgID string, in LockoutPolicyInput) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	current, err := c.GetLockoutPolicy(ctx, orgID)
	if err != nil {
		return err
	}

	if err := setPolicy(current.IsDefault,
		func() error {
			_, aerr := mc.AddCustomLockoutPolicy(ctx, &managementv1.AddCustomLockoutPolicyRequest{
				MaxPasswordAttempts: in.MaxPasswordAttempts,
				MaxOtpAttempts:      in.MaxOTPAttempts,
			})
			return aerr
		},
		func() error {
			_, aerr := mc.UpdateCustomLockoutPolicy(ctx, &managementv1.UpdateCustomLockoutPolicyRequest{
				MaxPasswordAttempts: in.MaxPasswordAttempts,
				MaxOtpAttempts:      in.MaxOTPAttempts,
			})
			return aerr
		},
	); err != nil {
		return fmt.Errorf("cannot set the lockout policy of organization %s: %w", orgID, err)
	}

	return nil
}

// ResetLockoutPolicy puts an organization back on the instance default lockout
// policy.
func (c *Client) ResetLockoutPolicy(ctx context.Context, orgID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.ResetLockoutPolicyToDefault(ctx, &managementv1.ResetLockoutPolicyToDefaultRequest{}); err != nil {
		if IsNotFound(err) || IsNotChanged(err) {
			return nil
		}

		return fmt.Errorf("cannot reset the lockout policy of organization %s: %w", orgID, err)
	}

	return nil
}

// NotificationPolicy is the notification policy of an organization.
type NotificationPolicy struct {
	OrgID     string
	IsDefault bool

	PasswordChange bool
}

// NotificationPolicyInput is the desired notification policy.
type NotificationPolicyInput struct {
	PasswordChange bool
}

// Input returns the policy in the shape the write API accepts.
func (p NotificationPolicy) Input() any {
	return NotificationPolicyInput{PasswordChange: p.PasswordChange}
}

// GetNotificationPolicy returns the notification policy of an organization.
func (c *Client) GetNotificationPolicy(ctx context.Context, orgID string) (*NotificationPolicy, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.GetNotificationPolicy(ctx, &managementv1.GetNotificationPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the notification policy of organization %s: %w", orgID, err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &NotificationPolicy{
		OrgID:          orgID,
		IsDefault:      p.GetIsDefault(),
		PasswordChange: p.GetPasswordChange(),
	}, nil
}

// SetNotificationPolicy customises or updates the notification policy of an
// organization.
func (c *Client) SetNotificationPolicy(ctx context.Context, orgID string, in NotificationPolicyInput) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	current, err := c.GetNotificationPolicy(ctx, orgID)
	if err != nil {
		return err
	}

	if err := setPolicy(current.IsDefault,
		func() error {
			_, aerr := mc.AddCustomNotificationPolicy(ctx, &managementv1.AddCustomNotificationPolicyRequest{
				PasswordChange: in.PasswordChange,
			})
			return aerr
		},
		func() error {
			_, aerr := mc.UpdateCustomNotificationPolicy(ctx, &managementv1.UpdateCustomNotificationPolicyRequest{
				PasswordChange: in.PasswordChange,
			})
			return aerr
		},
	); err != nil {
		return fmt.Errorf("cannot set the notification policy of organization %s: %w", orgID, err)
	}

	return nil
}

// ResetNotificationPolicy puts an organization back on the instance default
// notification policy.
func (c *Client) ResetNotificationPolicy(ctx context.Context, orgID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.ResetNotificationPolicyToDefault(ctx, &managementv1.ResetNotificationPolicyToDefaultRequest{}); err != nil {
		if IsNotFound(err) || IsNotChanged(err) {
			return nil
		}

		return fmt.Errorf("cannot reset the notification policy of organization %s: %w", orgID, err)
	}

	return nil
}

// PasswordAgePolicy is the password expiry policy of an organization.
type PasswordAgePolicy struct {
	OrgID     string
	IsDefault bool

	MaxAgeDays     uint32
	ExpireWarnDays uint32
}

// PasswordAgePolicyInput is the desired password expiry policy.
type PasswordAgePolicyInput struct {
	MaxAgeDays     uint32
	ExpireWarnDays uint32
}

// Input returns the policy in the shape the write API accepts.
func (p PasswordAgePolicy) Input() any {
	return PasswordAgePolicyInput{MaxAgeDays: p.MaxAgeDays, ExpireWarnDays: p.ExpireWarnDays}
}

// GetPasswordAgePolicy returns the password expiry policy of an organization.
func (c *Client) GetPasswordAgePolicy(ctx context.Context, orgID string) (*PasswordAgePolicy, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.GetPasswordAgePolicy(ctx, &managementv1.GetPasswordAgePolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the password age policy of organization %s: %w", orgID, err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &PasswordAgePolicy{
		OrgID:          orgID,
		IsDefault:      p.GetIsDefault(),
		MaxAgeDays:     boundedUint32(p.GetMaxAgeDays()),
		ExpireWarnDays: boundedUint32(p.GetExpireWarnDays()),
	}, nil
}

// SetPasswordAgePolicy customises or updates the password expiry policy of an
// organization.
func (c *Client) SetPasswordAgePolicy(ctx context.Context, orgID string, in PasswordAgePolicyInput) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	current, err := c.GetPasswordAgePolicy(ctx, orgID)
	if err != nil {
		return err
	}

	if err := setPolicy(current.IsDefault,
		func() error {
			_, aerr := mc.AddCustomPasswordAgePolicy(ctx, &managementv1.AddCustomPasswordAgePolicyRequest{
				MaxAgeDays:     in.MaxAgeDays,
				ExpireWarnDays: in.ExpireWarnDays,
			})
			return aerr
		},
		func() error {
			_, aerr := mc.UpdateCustomPasswordAgePolicy(ctx, &managementv1.UpdateCustomPasswordAgePolicyRequest{
				MaxAgeDays:     in.MaxAgeDays,
				ExpireWarnDays: in.ExpireWarnDays,
			})
			return aerr
		},
	); err != nil {
		return fmt.Errorf("cannot set the password age policy of organization %s: %w", orgID, err)
	}

	return nil
}

// ResetPasswordAgePolicy puts an organization back on the instance default
// password expiry policy.
func (c *Client) ResetPasswordAgePolicy(ctx context.Context, orgID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.ResetPasswordAgePolicyToDefault(ctx, &managementv1.ResetPasswordAgePolicyToDefaultRequest{}); err != nil {
		if IsNotFound(err) || IsNotChanged(err) {
			return nil
		}

		return fmt.Errorf("cannot reset the password age policy of organization %s: %w", orgID, err)
	}

	return nil
}

// PasswordComplexityPolicy is the password complexity policy of an organization.
type PasswordComplexityPolicy struct {
	OrgID     string
	IsDefault bool

	MinLength    uint32
	HasUppercase bool
	HasLowercase bool
	HasNumber    bool
	HasSymbol    bool
}

// PasswordComplexityPolicyInput is the desired password complexity policy.
type PasswordComplexityPolicyInput struct {
	MinLength    uint32
	HasUppercase bool
	HasLowercase bool
	HasNumber    bool
	HasSymbol    bool
}

// Input returns the policy in the shape the write API accepts.
func (p PasswordComplexityPolicy) Input() any {
	return PasswordComplexityPolicyInput{
		MinLength:    p.MinLength,
		HasUppercase: p.HasUppercase,
		HasLowercase: p.HasLowercase,
		HasNumber:    p.HasNumber,
		HasSymbol:    p.HasSymbol,
	}
}

// GetPasswordComplexityPolicy returns the password complexity policy of an
// organization.
func (c *Client) GetPasswordComplexityPolicy(ctx context.Context, orgID string) (*PasswordComplexityPolicy, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.GetPasswordComplexityPolicy(ctx, &managementv1.GetPasswordComplexityPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the password complexity policy of organization %s: %w", orgID, err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &PasswordComplexityPolicy{
		OrgID:        orgID,
		IsDefault:    p.GetIsDefault(),
		MinLength:    boundedUint32(p.GetMinLength()),
		HasUppercase: p.GetHasUppercase(),
		HasLowercase: p.GetHasLowercase(),
		HasNumber:    p.GetHasNumber(),
		HasSymbol:    p.GetHasSymbol(),
	}, nil
}

// SetPasswordComplexityPolicy customises or updates the password complexity
// policy of an organization.
func (c *Client) SetPasswordComplexityPolicy(ctx context.Context, orgID string, in PasswordComplexityPolicyInput) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	current, err := c.GetPasswordComplexityPolicy(ctx, orgID)
	if err != nil {
		return err
	}

	if err := setPolicy(current.IsDefault,
		func() error {
			_, aerr := mc.AddCustomPasswordComplexityPolicy(ctx, &managementv1.AddCustomPasswordComplexityPolicyRequest{
				MinLength:    uint64(in.MinLength),
				HasUppercase: in.HasUppercase,
				HasLowercase: in.HasLowercase,
				HasNumber:    in.HasNumber,
				HasSymbol:    in.HasSymbol,
			})
			return aerr
		},
		func() error {
			_, aerr := mc.UpdateCustomPasswordComplexityPolicy(ctx, &managementv1.UpdateCustomPasswordComplexityPolicyRequest{
				MinLength:    uint64(in.MinLength),
				HasUppercase: in.HasUppercase,
				HasLowercase: in.HasLowercase,
				HasNumber:    in.HasNumber,
				HasSymbol:    in.HasSymbol,
			})
			return aerr
		},
	); err != nil {
		return fmt.Errorf("cannot set the password complexity policy of organization %s: %w", orgID, err)
	}

	return nil
}

// ResetPasswordComplexityPolicy puts an organization back on the instance
// default password complexity policy.
func (c *Client) ResetPasswordComplexityPolicy(ctx context.Context, orgID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.ResetPasswordComplexityPolicyToDefault(ctx, &managementv1.ResetPasswordComplexityPolicyToDefaultRequest{}); err != nil {
		if IsNotFound(err) || IsNotChanged(err) {
			return nil
		}

		return fmt.Errorf("cannot reset the password complexity policy of organization %s: %w", orgID, err)
	}

	return nil
}

// PrivacyPolicy is the privacy policy of an organization: the links and contact
// details shown on Zitadel's own pages.
type PrivacyPolicy struct {
	OrgID     string
	IsDefault bool

	TOSLink        string
	PrivacyLink    string
	HelpLink       string
	SupportEmail   string
	DocsLink       string
	CustomLink     string
	CustomLinkText string
}

// PrivacyPolicyInput is the desired privacy policy.
type PrivacyPolicyInput struct {
	TOSLink        string
	PrivacyLink    string
	HelpLink       string
	SupportEmail   string
	DocsLink       string
	CustomLink     string
	CustomLinkText string
}

// Input returns the policy in the shape the write API accepts.
func (p PrivacyPolicy) Input() any {
	return PrivacyPolicyInput{
		TOSLink:        p.TOSLink,
		PrivacyLink:    p.PrivacyLink,
		HelpLink:       p.HelpLink,
		SupportEmail:   p.SupportEmail,
		DocsLink:       p.DocsLink,
		CustomLink:     p.CustomLink,
		CustomLinkText: p.CustomLinkText,
	}
}

// GetPrivacyPolicy returns the privacy policy of an organization.
func (c *Client) GetPrivacyPolicy(ctx context.Context, orgID string) (*PrivacyPolicy, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.GetPrivacyPolicy(ctx, &managementv1.GetPrivacyPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the privacy policy of organization %s: %w", orgID, err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &PrivacyPolicy{
		OrgID:          orgID,
		IsDefault:      p.GetIsDefault(),
		TOSLink:        p.GetTosLink(),
		PrivacyLink:    p.GetPrivacyLink(),
		HelpLink:       p.GetHelpLink(),
		SupportEmail:   p.GetSupportEmail(),
		DocsLink:       p.GetDocsLink(),
		CustomLink:     p.GetCustomLink(),
		CustomLinkText: p.GetCustomLinkText(),
	}, nil
}

// SetPrivacyPolicy customises or updates the privacy policy of an organization.
func (c *Client) SetPrivacyPolicy(ctx context.Context, orgID string, in PrivacyPolicyInput) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	current, err := c.GetPrivacyPolicy(ctx, orgID)
	if err != nil {
		return err
	}

	if err := setPolicy(current.IsDefault,
		func() error {
			_, aerr := mc.AddCustomPrivacyPolicy(ctx, &managementv1.AddCustomPrivacyPolicyRequest{
				TosLink:        in.TOSLink,
				PrivacyLink:    in.PrivacyLink,
				HelpLink:       in.HelpLink,
				SupportEmail:   in.SupportEmail,
				DocsLink:       in.DocsLink,
				CustomLink:     in.CustomLink,
				CustomLinkText: in.CustomLinkText,
			})
			return aerr
		},
		func() error {
			_, aerr := mc.UpdateCustomPrivacyPolicy(ctx, &managementv1.UpdateCustomPrivacyPolicyRequest{
				TosLink:        in.TOSLink,
				PrivacyLink:    in.PrivacyLink,
				HelpLink:       in.HelpLink,
				SupportEmail:   in.SupportEmail,
				DocsLink:       in.DocsLink,
				CustomLink:     in.CustomLink,
				CustomLinkText: in.CustomLinkText,
			})
			return aerr
		},
	); err != nil {
		return fmt.Errorf("cannot set the privacy policy of organization %s: %w", orgID, err)
	}

	return nil
}

// ResetPrivacyPolicy puts an organization back on the instance default privacy
// policy.
func (c *Client) ResetPrivacyPolicy(ctx context.Context, orgID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.ResetPrivacyPolicyToDefault(ctx, &managementv1.ResetPrivacyPolicyToDefaultRequest{}); err != nil {
		if IsNotFound(err) || IsNotChanged(err) {
			return nil
		}

		return fmt.Errorf("cannot reset the privacy policy of organization %s: %w", orgID, err)
	}

	return nil
}

// DomainPolicy controls how login names relate to an organization's domains.
type DomainPolicy struct {
	OrgID     string
	IsDefault bool

	UserLoginMustBeDomain                  bool
	ValidateOrgDomains                     bool
	SMTPSenderAddressMatchesInstanceDomain bool
}

// DomainPolicyInput is the desired domain policy.
type DomainPolicyInput struct {
	UserLoginMustBeDomain                  bool
	ValidateOrgDomains                     bool
	SMTPSenderAddressMatchesInstanceDomain bool
}

// Input returns the policy in the shape the write API accepts.
func (p DomainPolicy) Input() any {
	return DomainPolicyInput{
		UserLoginMustBeDomain:                  p.UserLoginMustBeDomain,
		ValidateOrgDomains:                     p.ValidateOrgDomains,
		SMTPSenderAddressMatchesInstanceDomain: p.SMTPSenderAddressMatchesInstanceDomain,
	}
}

// GetDomainPolicy returns the domain policy of an organization.
//
// The domain policy is reached through the admin API rather than the management
// API, even for an organization's own policy: the management API only reads it.
func (c *Client) GetDomainPolicy(ctx context.Context, orgID string) (*DomainPolicy, error) {
	ac, err := c.adminClientForOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := ac.GetDomainPolicy(ctx, &adminv1.GetDomainPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the domain policy of organization %s: %w", orgID, err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return &DomainPolicy{
		OrgID:                                  orgID,
		IsDefault:                              p.GetIsDefault(),
		UserLoginMustBeDomain:                  p.GetUserLoginMustBeDomain(),
		ValidateOrgDomains:                     p.GetValidateOrgDomains(),
		SMTPSenderAddressMatchesInstanceDomain: p.GetSmtpSenderAddressMatchesInstanceDomain(),
	}, nil
}

// SetDomainPolicy customises or updates the domain policy of an organization.
func (c *Client) SetDomainPolicy(ctx context.Context, orgID string, in DomainPolicyInput) error {
	ac, err := c.adminClientForOrg(ctx, orgID)
	if err != nil {
		return err
	}

	current, err := c.GetDomainPolicy(ctx, orgID)
	if err != nil {
		return err
	}

	if err := setPolicy(current.IsDefault,
		func() error {
			_, aerr := ac.AddCustomDomainPolicy(ctx, &adminv1.AddCustomDomainPolicyRequest{
				OrgId:                                  orgID,
				UserLoginMustBeDomain:                  in.UserLoginMustBeDomain,
				ValidateOrgDomains:                     in.ValidateOrgDomains,
				SmtpSenderAddressMatchesInstanceDomain: in.SMTPSenderAddressMatchesInstanceDomain,
			})
			return aerr
		},
		func() error {
			_, aerr := ac.UpdateCustomDomainPolicy(ctx, &adminv1.UpdateCustomDomainPolicyRequest{
				OrgId:                                  orgID,
				UserLoginMustBeDomain:                  in.UserLoginMustBeDomain,
				ValidateOrgDomains:                     in.ValidateOrgDomains,
				SmtpSenderAddressMatchesInstanceDomain: in.SMTPSenderAddressMatchesInstanceDomain,
			})
			return aerr
		},
	); err != nil {
		return fmt.Errorf("cannot set the domain policy of organization %s: %w", orgID, err)
	}

	return nil
}

// ResetDomainPolicy puts an organization back on the instance default domain
// policy.
func (c *Client) ResetDomainPolicy(ctx context.Context, orgID string) error {
	ac, err := c.adminClientForOrg(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := ac.ResetCustomDomainPolicyToDefault(ctx, &adminv1.ResetCustomDomainPolicyToDefaultRequest{OrgId: orgID}); err != nil {
		if IsNotFound(err) || IsNotChanged(err) {
			return nil
		}

		return fmt.Errorf("cannot reset the domain policy of organization %s: %w", orgID, err)
	}

	return nil
}

// LabelPolicy is the branding policy of an organization.
type LabelPolicy struct {
	OrgID     string
	IsDefault bool

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

	// AssetHashes are the Zitadel file object hashes of the branding assets. A
	// hash identifies an asset that was uploaded to Zitadel's file storage
	// beforehand; this provider cannot create one.
	LogoHash     string
	IconHash     string
	LogoDarkHash string
	IconDarkHash string
	FontHash     string

	// The URLs Zitadel serves the assets from, once they are set.
	LogoURL     string
	IconURL     string
	LogoDarkURL string
	IconDarkURL string
	FontURL     string
}

// LabelPolicyInput is the desired branding policy.
type LabelPolicyInput struct {
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
func (p LabelPolicy) Input() any {
	return LabelPolicyInput{
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

// GetLabelPolicy returns the branding policy of an organization.
func (c *Client) GetLabelPolicy(ctx context.Context, orgID string) (*LabelPolicy, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// The policy is read through the preview endpoint rather than GetLabelPolicy.
	//
	// They do not agree: a write is reflected immediately by the preview, which
	// is what Zitadel itself renders the login pages from, while GetLabelPolicy
	// keeps answering with the instance default's values and its is_default
	// flag. Reading the preview is what makes drift detection see the policy
	// that is actually in effect - and reading the other one makes an
	// organization look customised when it is not, and the reverse after a
	// change, which is a reconcile loop with no end.
	resp, err := mc.GetPreviewLabelPolicy(ctx, &managementv1.GetPreviewLabelPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the label policy of organization %s: %w", orgID, err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	out := &LabelPolicy{
		OrgID:               orgID,
		IsDefault:           resp.GetIsDefault(),
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
	}

	return out, nil
}

// SetLabelPolicy customises or updates the branding policy of an organization.
//
// Zitadel's policy endpoints carry only the colours and flags. Setting or
// clearing an asset is a separate call, so the assets are reconciled here
// against what is currently set.
func (c *Client) SetLabelPolicy(ctx context.Context, orgID string, in LabelPolicyInput) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	current, err := c.GetLabelPolicy(ctx, orgID)
	if err != nil {
		return err
	}

	mode, err := LabelThemeModeToProto(in.ThemeMode)
	if err != nil {
		return err
	}

	if err := setPolicy(current.IsDefault,
		func() error {
			_, aerr := mc.AddCustomLabelPolicy(ctx, &managementv1.AddCustomLabelPolicyRequest{
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
			})
			return aerr
		},
		func() error {
			_, aerr := mc.UpdateCustomLabelPolicy(ctx, &managementv1.UpdateCustomLabelPolicyRequest{
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
			})
			return aerr
		},
	); err != nil {
		return fmt.Errorf("cannot set the label policy of organization %s: %w", orgID, err)
	}

	return nil
}

// ResetLabelPolicy puts an organization back on the instance default branding
// policy.
func (c *Client) ResetLabelPolicy(ctx context.Context, orgID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.ResetLabelPolicyToDefault(ctx, &managementv1.ResetLabelPolicyToDefaultRequest{}); err != nil {
		if IsNotFound(err) || IsNotChanged(err) {
			return nil
		}

		return fmt.Errorf("cannot reset the label policy of organization %s: %w", orgID, err)
	}

	return nil
}

// LabelThemeModeToProto maps the API spelling of a label policy theme mode onto
// the protobuf enum, so that callers never depend on the generated enum.
func LabelThemeModeToProto(mode string) (policyv1.ThemeMode, error) {
	switch mode {
	case "", "Auto":
		return policyv1.ThemeMode_THEME_MODE_AUTO, nil

	case "Light":
		return policyv1.ThemeMode_THEME_MODE_LIGHT, nil

	case "Dark":
		return policyv1.ThemeMode_THEME_MODE_DARK, nil

	default:
		return policyv1.ThemeMode_THEME_MODE_UNSPECIFIED, fmt.Errorf("unknown label policy theme mode %q: must be Auto, Light or Dark", mode)
	}
}

// boundedUint32 narrows a count Zitadel reports as a uint64 into the uint32 its
// write API accepts.
//
// Every such field is an attempt count or a number of days, which no real
// deployment comes anywhere near uint32. Saturating rather than wrapping is what
// makes a nonsensical value from the server visible as an extreme setting rather
// than as a small plausible one.
func boundedUint32(v uint64) uint32 {
	if v > math.MaxUint32 {
		return math.MaxUint32
	}

	return uint32(v) //nolint:gosec // Bounded above by the check on the line before.
}

// setPolicy applies an organization policy, choosing between Zitadel's add and
// update calls.
//
// Which one applies depends on whether the organization already holds a custom
// policy, and the read that answers that is not dependable: an organization can
// be reported as still on the instance default while already holding a custom
// policy, and an add then fails with "already exists". So the read picks the
// call, and the failure of that call corrects it - which makes writing a policy
// correct whichever way the read went, instead of depending on it.
func setPolicy(inherited bool, add, update func() error) error {
	if inherited {
		err := add()
		if err != nil && IsAlreadyExists(err) {
			err = update()
		}

		return ignoreNoChange(err)
	}

	err := update()
	if err != nil && IsNotFound(err) {
		err = add()
	}

	return ignoreNoChange(err)
}

// ignoreNoChange swallows Zitadel's refusal to apply a write that changes
// nothing, which is the desired end state rather than a failure.
func ignoreNoChange(err error) error {
	if err == nil || IsNotChanged(err) {
		return nil
	}

	return err
}

// LabelThemeModeFromProto maps a label policy theme mode back onto the API
// spelling, so that a value read from Zitadel compares equal to the one an
// operator wrote.
//
// The generated enum's own String method is not used: it returns the protobuf
// constant name, THEME_MODE_LIGHT, rather than the API's Light.
func LabelThemeModeFromProto(mode policyv1.ThemeMode) string {
	switch mode {
	case policyv1.ThemeMode_THEME_MODE_LIGHT:
		return "Light"

	case policyv1.ThemeMode_THEME_MODE_DARK:
		return "Dark"

	// An unset or unrecognised mode follows the visitor's system preference,
	// which is what Zitadel does with it.
	case policyv1.ThemeMode_THEME_MODE_AUTO, policyv1.ThemeMode_THEME_MODE_UNSPECIFIED:
		return "Auto"

	default:
		return "Auto"
	}
}
