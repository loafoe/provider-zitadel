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
	"slices"
	"sort"

	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	policyv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/policy"
	"google.golang.org/protobuf/types/known/durationpb"
)

// LoginPolicy is an organization's login and authentication policy. Zitadel
// keeps exactly one per organization, so this resource is a singleton: the
// external name is the organization it belongs to.
type LoginPolicy struct {
	OrgID     string
	IsDefault bool

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
	IDPs          []string
}

// GetLoginPolicy returns the login policy of an organization.
func (c *Client) GetLoginPolicy(ctx context.Context, orgID string) (*LoginPolicy, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.GetLoginPolicy(ctx, &managementv1.GetLoginPolicyRequest{})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get the login policy of organization %s: %w", orgID, err)
	}

	p := resp.GetPolicy()
	if p == nil {
		return nil, ErrNotFound
	}

	return fromProtoLoginPolicy(orgID, p, resp.GetIsDefault()), nil
}

// LoginPolicyInput is the desired login policy. Durations are Go duration
// strings such as "24h"; an empty string leaves the field unset.
type LoginPolicyInput struct {
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

	// SecondFactors and MultiFactors use the API spelling, e.g. "OTP" and
	// "U2FWithVerification", and are translated to the protobuf enums here so
	// that callers never depend on them.
	SecondFactors []string
	MultiFactors  []string
	IDPs          []string
}

// SetLoginPolicy creates the login policy of an organization, or updates it
// when it already exists.
//
// Zitadel splits this in two: AddCustomLoginPolicy takes the factor lists, while
// UpdateCustomLoginPolicy does not. So the second factor, multi factor and IdP
// lists are reconciled by diffing them against the current policy and calling
// the dedicated add/remove endpoints - otherwise changing a factor on an
// existing policy would silently do nothing.
//
//nolint:gocyclo // one branch per policy field; splitting it would obscure the create/update split
func (c *Client) SetLoginPolicy(ctx context.Context, orgID string, in LoginPolicyInput) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	// The API enums are translated once here so the request and the diff below
	// agree on what "the same factor" means.
	secondFactors, err := secondFactorProtos(in.SecondFactors)
	if err != nil {
		return err
	}

	multiFactors, err := multiFactorProtos(in.MultiFactors)
	if err != nil {
		return err
	}

	// Whether the organization has a *custom* policy is what decides create
	// against update. Zitadel answers the read for an organization that has
	// never customised its policy, reporting the instance default, and then
	// rejects the update because there is no custom policy to update.
	current, err := c.GetLoginPolicy(ctx, orgID)
	if err != nil && !IsNotFound(err) {
		return err
	}

	exists := err == nil && !current.IsDefault

	if !exists {
		req := &managementv1.AddCustomLoginPolicyRequest{
			AllowUsernamePassword:  in.AllowUsernamePassword,
			AllowRegister:          in.AllowRegister,
			AllowExternalIdp:       in.AllowExternalIDP,
			ForceMfa:               in.ForceMFA,
			ForceMfaLocalOnly:      in.ForceMFALocalOnly,
			HidePasswordReset:      in.HidePasswordReset,
			IgnoreUnknownUsernames: in.IgnoreUnknownUsernames,
			AllowDomainDiscovery:   in.AllowDomainDiscovery,
			DisableLoginWithEmail:  in.DisableLoginWithEmail,
			DisableLoginWithPhone:  in.DisableLoginWithPhone,
			DefaultRedirectUri:     in.DefaultRedirectURI,
			SecondFactors:          secondFactors,
			MultiFactors:           multiFactors,
		}

		pw, err := passwordlessType(in.PasswordlessType)
		if err != nil {
			return err
		}
		req.PasswordlessType = pw

		for _, d := range in.IDPs {
			req.Idps = append(req.Idps, &managementv1.AddCustomLoginPolicyRequest_IDP{IdpId: d})
		}

		durations, err := loginPolicyDurations(in)
		if err != nil {
			return err
		}
		applyDurationsToAdd(req, durations)

		if _, err := mc.AddCustomLoginPolicy(ctx, req); err != nil {
			return fmt.Errorf("cannot create the login policy of organization %s: %w", orgID, err)
		}

		return nil
	}

	pw, err := passwordlessType(in.PasswordlessType)
	if err != nil {
		return err
	}

	durations, err := loginPolicyDurations(in)
	if err != nil {
		return err
	}

	req := &managementv1.UpdateCustomLoginPolicyRequest{
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
		PasswordlessType:           pw,
		PasswordCheckLifetime:      durations.passwordCheckLifetime,
		ExternalLoginCheckLifetime: durations.externalLoginCheckLifetime,
		MfaInitSkipLifetime:        durations.mfaInitSkipLifetime,
		SecondFactorCheckLifetime:  durations.secondFactorCheckLifetime,
		MultiFactorCheckLifetime:   durations.multiFactorCheckLifetime,
	}

	if _, err := mc.UpdateCustomLoginPolicy(ctx, req); err != nil {
		return fmt.Errorf("cannot update the login policy of organization %s: %w", orgID, err)
	}

	if err := c.reconcileLoginPolicyFactors(ctx, orgID, current, in); err != nil {
		return err
	}

	return nil
}

// ResetLoginPolicy restores the instance default login policy for an
// organization. Zitadel has no delete for a custom login policy, so deleting
// this resource means reverting to the default rather than removing anything.
func (c *Client) ResetLoginPolicy(ctx context.Context, orgID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.ResetLoginPolicyToDefault(ctx, &managementv1.ResetLoginPolicyToDefaultRequest{}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot reset the login policy of organization %s: %w", orgID, err)
	}

	return nil
}

// reconcileLoginPolicyFactors applies the difference between the desired and the
// current second factor, multi factor and IdP lists.
//
//nolint:gocyclo // add and remove for three factor lists
func (c *Client) reconcileLoginPolicyFactors(ctx context.Context, orgID string, current *LoginPolicy, in LoginPolicyInput) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	wantSecond := normalize(in.SecondFactors)

	for _, name := range difference(wantSecond, current.SecondFactors) {
		t, err := SecondFactorToProto(name)
		if err != nil {
			return err
		}
		if _, err := mc.AddSecondFactorToLoginPolicy(ctx, &managementv1.AddSecondFactorToLoginPolicyRequest{Type: t}); err != nil {
			return fmt.Errorf("cannot add the second factor %s to the login policy of organization %s: %w", name, orgID, err)
		}
	}
	for _, name := range difference(current.SecondFactors, wantSecond) {
		t, err := SecondFactorToProto(name)
		if err != nil {
			return err
		}
		if _, err := mc.RemoveSecondFactorFromLoginPolicy(ctx, &managementv1.RemoveSecondFactorFromLoginPolicyRequest{Type: t}); err != nil {
			return fmt.Errorf("cannot remove the second factor %s from the login policy of organization %s: %w", name, orgID, err)
		}
	}

	wantMulti := normalize(in.MultiFactors)

	for _, name := range difference(wantMulti, current.MultiFactors) {
		t, err := MultiFactorToProto(name)
		if err != nil {
			return err
		}
		if _, err := mc.AddMultiFactorToLoginPolicy(ctx, &managementv1.AddMultiFactorToLoginPolicyRequest{Type: t}); err != nil {
			return fmt.Errorf("cannot add the multi factor %s to the login policy of organization %s: %w", name, orgID, err)
		}
	}
	for _, name := range difference(current.MultiFactors, wantMulti) {
		t, err := MultiFactorToProto(name)
		if err != nil {
			return err
		}
		if _, err := mc.RemoveMultiFactorFromLoginPolicy(ctx, &managementv1.RemoveMultiFactorFromLoginPolicyRequest{Type: t}); err != nil {
			return fmt.Errorf("cannot remove the multi factor %s from the login policy of organization %s: %w", name, orgID, err)
		}
	}

	for _, name := range difference(in.IDPs, current.IDPs) {
		if _, err := mc.AddIDPToLoginPolicy(ctx, &managementv1.AddIDPToLoginPolicyRequest{IdpId: name}); err != nil {
			return fmt.Errorf("cannot add IdP %s to the login policy of organization %s: %w", name, orgID, err)
		}
	}
	for _, name := range difference(current.IDPs, in.IDPs) {
		if _, err := mc.RemoveIDPFromLoginPolicy(ctx, &managementv1.RemoveIDPFromLoginPolicyRequest{IdpId: name}); err != nil {
			return fmt.Errorf("cannot remove IdP %s from the login policy of organization %s: %w", name, orgID, err)
		}
	}

	return nil
}

// difference returns the entries of a that are not in b.
func difference(a, b []string) []string {
	var out []string
	for _, v := range a {
		if !slices.Contains(b, v) {
			out = append(out, v)
		}
	}

	return out
}

type policyDurations struct {
	passwordCheckLifetime      *durationpb.Duration
	externalLoginCheckLifetime *durationpb.Duration
	mfaInitSkipLifetime        *durationpb.Duration
	secondFactorCheckLifetime  *durationpb.Duration
	multiFactorCheckLifetime   *durationpb.Duration
}

func loginPolicyDurations(in LoginPolicyInput) (policyDurations, error) {
	var out policyDurations

	for _, d := range []struct {
		in    string
		field string
		out   **durationpb.Duration
	}{
		{in.PasswordCheckLifetime, "passwordCheckLifetime", &out.passwordCheckLifetime},
		{in.ExternalLoginCheckLifetime, "externalLoginCheckLifetime", &out.externalLoginCheckLifetime},
		{in.MFAInitSkipLifetime, "mfaInitSkipLifetime", &out.mfaInitSkipLifetime},
		{in.SecondFactorCheckLifetime, "secondFactorCheckLifetime", &out.secondFactorCheckLifetime},
		{in.MultiFactorCheckLifetime, "multiFactorCheckLifetime", &out.multiFactorCheckLifetime},
	} {
		if d.in == "" {
			continue
		}
		v, err := ParseDuration(d.in)
		if err != nil {
			return out, fmt.Errorf("%s: %w", d.field, err)
		}
		*d.out = v
	}

	return out, nil
}

func applyDurationsToAdd(req *managementv1.AddCustomLoginPolicyRequest, d policyDurations) {
	req.PasswordCheckLifetime = d.passwordCheckLifetime
	req.ExternalLoginCheckLifetime = d.externalLoginCheckLifetime
	req.MfaInitSkipLifetime = d.mfaInitSkipLifetime
	req.SecondFactorCheckLifetime = d.secondFactorCheckLifetime
	req.MultiFactorCheckLifetime = d.multiFactorCheckLifetime
}

// normalize sorts a set of names so that comparison ignores order.
func normalize(in []string) []string {
	out := make([]string, 0, len(in))
	out = append(out, in...)
	sort.Strings(out)

	return out
}

func secondFactorProtos(in []string) ([]policyv1.SecondFactorType, error) {
	out := make([]policyv1.SecondFactorType, 0, len(in))
	for _, name := range in {
		v, err := SecondFactorToProto(name)
		if err != nil {
			return nil, err
		}
		if v != policyv1.SecondFactorType_SECOND_FACTOR_TYPE_UNSPECIFIED {
			out = append(out, v)
		}
	}

	return out, nil
}

func multiFactorProtos(in []string) ([]policyv1.MultiFactorType, error) {
	out := make([]policyv1.MultiFactorType, 0, len(in))
	for _, name := range in {
		v, err := MultiFactorToProto(name)
		if err != nil {
			return nil, err
		}
		if v != policyv1.MultiFactorType_MULTI_FACTOR_TYPE_UNSPECIFIED {
			out = append(out, v)
		}
	}

	return out, nil
}

func passwordlessType(v string) (policyv1.PasswordlessType, error) {
	switch v {
	case "", "NotAllowed":
		return policyv1.PasswordlessType_PASSWORDLESS_TYPE_NOT_ALLOWED, nil
	case "Allowed":
		return policyv1.PasswordlessType_PASSWORDLESS_TYPE_ALLOWED, nil
	default:
		return policyv1.PasswordlessType_PASSWORDLESS_TYPE_NOT_ALLOWED,
			fmt.Errorf("unsupported passwordless type %q", v)
	}
}

// The factor names below are the API representation of Zitadel's factor enums,
// which are otherwise spelled as long protobuf constants.

// SecondFactorToProto maps an API second factor name onto its protobuf enum.
func SecondFactorToProto(name string) (policyv1.SecondFactorType, error) {
	switch name {
	case "":
		return policyv1.SecondFactorType_SECOND_FACTOR_TYPE_UNSPECIFIED, nil
	case "OTP":
		return policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP, nil
	case "U2F":
		return policyv1.SecondFactorType_SECOND_FACTOR_TYPE_U2F, nil
	case "OTPEmail":
		return policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP_EMAIL, nil
	case "OTPSMS":
		return policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP_SMS, nil
	case "RecoveryCodes":
		return policyv1.SecondFactorType_SECOND_FACTOR_TYPE_RECOVERY_CODES, nil
	default:
		return policyv1.SecondFactorType_SECOND_FACTOR_TYPE_UNSPECIFIED,
			fmt.Errorf("unsupported second factor %q", name)
	}
}

// SecondFactorNames renders the second factor enums as their API spelling.
func SecondFactorNames(in []policyv1.SecondFactorType) ([]string, error) {
	out := make([]string, 0, len(in))
	for _, v := range in {
		switch v {
		case policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP:
			out = append(out, "OTP")
		case policyv1.SecondFactorType_SECOND_FACTOR_TYPE_U2F:
			out = append(out, "U2F")
		case policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP_EMAIL:
			out = append(out, "OTPEmail")
		case policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP_SMS:
			out = append(out, "OTPSMS")
		case policyv1.SecondFactorType_SECOND_FACTOR_TYPE_RECOVERY_CODES:
			out = append(out, "RecoveryCodes")
		case policyv1.SecondFactorType_SECOND_FACTOR_TYPE_UNSPECIFIED:
		default:
			return nil, fmt.Errorf("unsupported second factor %q", v.String())
		}
	}

	return out, nil
}

// MultiFactorToProto maps an API multi factor name onto its protobuf enum.
func MultiFactorToProto(name string) (policyv1.MultiFactorType, error) {
	switch name {
	case "", "NotAllowed":
		return policyv1.MultiFactorType_MULTI_FACTOR_TYPE_UNSPECIFIED, nil
	case "U2FWithVerification":
		return policyv1.MultiFactorType_MULTI_FACTOR_TYPE_U2F_WITH_VERIFICATION, nil
	default:
		return policyv1.MultiFactorType_MULTI_FACTOR_TYPE_UNSPECIFIED,
			fmt.Errorf("unsupported multi factor %q", name)
	}
}

// MultiFactorNames renders the multi factor enums as their API spelling.
func MultiFactorNames(in []policyv1.MultiFactorType) ([]string, error) {
	out := make([]string, 0, len(in))
	for _, v := range in {
		switch v {
		case policyv1.MultiFactorType_MULTI_FACTOR_TYPE_U2F_WITH_VERIFICATION:
			out = append(out, "U2FWithVerification")
		case policyv1.MultiFactorType_MULTI_FACTOR_TYPE_UNSPECIFIED:
		default:
			return nil, fmt.Errorf("unsupported multi factor %q", v.String())
		}
	}

	return out, nil
}

func passwordlessTypeFromProto(v policyv1.PasswordlessType) string {
	switch v {
	case policyv1.PasswordlessType_PASSWORDLESS_TYPE_ALLOWED:
		return "Allowed"
	case policyv1.PasswordlessType_PASSWORDLESS_TYPE_NOT_ALLOWED:
		return "NotAllowed"
	}

	return ""
}

func fromProtoLoginPolicy(orgID string, p *policyv1.LoginPolicy, isDefault bool) *LoginPolicy {
	// Zitadel only ever reports factors it knows, so an unrecognised value is
	// surfaced verbatim rather than failing the whole observation.
	secondFactors, err := SecondFactorNames(p.GetSecondFactors())
	if err != nil {
		for _, f := range p.GetSecondFactors() {
			secondFactors = append(secondFactors, f.String())
		}
	}

	multiFactors, err := MultiFactorNames(p.GetMultiFactors())
	if err != nil {
		for _, f := range p.GetMultiFactors() {
			multiFactors = append(multiFactors, f.String())
		}
	}

	out := &LoginPolicy{
		OrgID:                      orgID,
		IsDefault:                  isDefault,
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
		SecondFactors:              secondFactors,
		MultiFactors:               multiFactors,
	}

	for _, idp := range p.GetIdps() {
		out.IDPs = append(out.IDPs, idp.GetIdpId())
	}

	return out
}

func durationString(d *durationpb.Duration) string {
	if d == nil {
		return ""
	}

	return d.AsDuration().String()
}

// durationProto parses a Go duration string for the Zitadel API.
//
// An empty string means "leave it alone" and is passed on as an absent duration,
// which is what keeps an unset lifetime from being written as an explicit zero.
//
// Anything unparseable - including a negative duration, which Zitadel would
// reject - is dropped rather than sent, the same way a missing field is. The
// parsing goes through ParseDuration so that this and the exported helper cannot
// disagree about what counts as a duration.
func durationProto(s string) *durationpb.Duration {
	if s == "" {
		return nil
	}

	d, err := ParseDuration(s)
	if err != nil {
		return nil
	}

	return d
}
