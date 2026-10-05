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

package loginpolicy

import (
	"context"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// Setup adds a controller that reconciles LoginPolicy managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.LoginPolicyGroupKind,
		v1alpha1.LoginPolicyGroupVersionKind,
		&v1alpha1.LoginPolicy{},
		&v1alpha1.LoginPolicyList{},
		driver{},
	)
}

// driver is everything specific to the login policy. The lifecycle around it -
// when the policy exists, what counts as drift, what deleting it means - is the
// shared policy harness's, because every Zitadel policy behaves that way.
type driver struct{}

var _ common.PolicyDriver[zitadel.LoginPolicyInput, zitadel.LoginPolicy] = driver{}

// Kind names the policy, for errors and events.
func (driver) Kind() string { return "LoginPolicy" }

// Scope resolves the organization the policy belongs to. A login policy is always
// an organization's own, so an unresolved reference is an error rather than a
// request for the instance default.
func (driver) Scope(ctx context.Context, kube client.Client, cr common.ManagedPolicy) (string, error) {
	policy, ok := cr.(*v1alpha1.LoginPolicy)
	if !ok {
		return "", errors.New("managed resource is not a LoginPolicy custom resource")
	}

	fp := policy.Spec.ForProvider

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, kube, policy)
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	// A terminating object acts on the organization it recorded: it exists to
	// reset the policy it set, not to reset one belonging to a replacement.
	orgID, err := common.ResolveOrganizationID(ctx, kube, policy, fp.OrganizationRef,
		fp.OrganizationSelector, fp.OrganizationID, orgDefault,
		common.CurrentIfDeleting(meta.WasDeleted(policy), policy.Status.AtProvider.OrganizationID))
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	if orgID == "" {
		return "", errors.New(common.ErrNoOrganizationID.Error())
	}

	return orgID, nil
}

// Get reads the organization's login policy.
func (driver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.LoginPolicy, bool, error) {
	p, err := c.GetLoginPolicy(ctx, scope)
	if err != nil {
		return zitadel.LoginPolicy{}, false, err
	}

	return *p, p.IsDefault, nil
}

// Apply writes the login policy, adding or updating it as Zitadel requires.
//
// Zitadel answers the read for an organization that has never customised its
// policy, reporting the instance default, and then rejects the update because
// there is no custom policy to update. SetLoginPolicy adds in that case and
// updates otherwise, which is why the two are not separate driver operations.
func (driver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.LoginPolicyInput) error {
	return c.SetLoginPolicy(ctx, scope, want)
}

// Resettable reports that deleting this resource can put the organization back
// on the instance default, which is how Zitadel removes a custom policy.
func (driver) Resettable() bool { return true }

// Reset puts the organization back on the instance default login policy.
func (driver) Reset(ctx context.Context, c *zitadel.Client, scope string) error {
	return c.ResetLoginPolicy(ctx, scope)
}

// Desired reads the desired policy out of the managed resource.
func (driver) Desired(cr common.ManagedPolicy) zitadel.LoginPolicyInput {
	return desired(cr.(*v1alpha1.LoginPolicy))
}

// Report writes the observed policy into the managed resource's status.
func (driver) Report(cr common.ManagedPolicy, observed zitadel.LoginPolicy) {
	updateStatus(cr.(*v1alpha1.LoginPolicy), observed)
}

// Equal reports whether the observed policy already matches the desired one.
// Equal reports whether the observed policy already matches the desired one.
//
// Unlike the other policies, this one defaults an unset field to Zitadel's own
// documented default rather than declining to manage it - see Desired - so the
// two values can be compared directly without consulting the spec.
func (driver) Equal(_ common.ManagedPolicy, want zitadel.LoginPolicyInput, got zitadel.LoginPolicy) bool {
	return samePolicy(want, &got)
}

// desired translates the spec into the client input, applying the documented
// defaults for every field the spec leaves unset. The fields are pointers so
// that "unset" is distinguishable from "false", which matters for a policy
// where false is the restrictive value.
func desired(cr *v1alpha1.LoginPolicy) zitadel.LoginPolicyInput {
	fp := cr.Spec.ForProvider

	in := zitadel.LoginPolicyInput{
		AllowUsernamePassword:      valueOr(fp.AllowUsernamePassword, true),
		AllowRegister:              valueOr(fp.AllowRegister, false),
		AllowExternalIDP:           valueOr(fp.AllowExternalIDP, true),
		ForceMFA:                   valueOr(fp.ForceMFA, false),
		ForceMFALocalOnly:          valueOr(fp.ForceMFALocalOnly, false),
		HidePasswordReset:          valueOr(fp.HidePasswordReset, false),
		IgnoreUnknownUsernames:     valueOr(fp.IgnoreUnknownUsernames, true),
		AllowDomainDiscovery:       valueOr(fp.AllowDomainDiscovery, false),
		DisableLoginWithEmail:      valueOr(fp.DisableLoginWithEmail, false),
		DisableLoginWithPhone:      valueOr(fp.DisableLoginWithPhone, false),
		DefaultRedirectURI:         common.Deref(fp.DefaultRedirectURI),
		PasswordlessType:           common.Value(fp.PasswordlessType),
		PasswordCheckLifetime:      common.Deref(fp.PasswordCheckLifetime),
		ExternalLoginCheckLifetime: common.Deref(fp.ExternalLoginCheckLifetime),
		MFAInitSkipLifetime:        common.Deref(fp.MFAInitSkipLifetime),
		SecondFactorCheckLifetime:  common.Deref(fp.SecondFactorCheckLifetime),
		MultiFactorCheckLifetime:   common.Deref(fp.MultiFactorCheckLifetime),
		IDPs:                       fp.IDPs,
	}

	for _, f := range fp.SecondFactors {
		in.SecondFactors = append(in.SecondFactors, string(f))
	}

	for _, f := range fp.MultiFactors {
		in.MultiFactors = append(in.MultiFactors, string(f))
	}

	// Zitadel reports "NotAllowed" rather than an empty value, so the default is
	// spelled out here or every reconcile would see drift.
	if in.PasswordlessType == "" {
		in.PasswordlessType = string(v1alpha1.PasswordlessTypeNotAllowed)
	}

	return in
}

// valueOr dereferences a bool, falling back to a default.
func valueOr(b *bool, fallback bool) bool {
	if b == nil {
		return fallback
	}

	return *b
}

// samePolicy reports whether an observed login policy already matches the
// desired one.
func samePolicy(want zitadel.LoginPolicyInput, p *zitadel.LoginPolicy) bool {
	for _, f := range []struct {
		name      string
		want, got bool
	}{
		{"allowUsernamePassword", want.AllowUsernamePassword, p.AllowUsernamePassword},
		{"allowRegister", want.AllowRegister, p.AllowRegister},
		{"allowExternalIDP", want.AllowExternalIDP, p.AllowExternalIDP},
		{"forceMFA", want.ForceMFA, p.ForceMFA},
		{"forceMFALocalOnly", want.ForceMFALocalOnly, p.ForceMFALocalOnly},
		{"hidePasswordReset", want.HidePasswordReset, p.HidePasswordReset},
		{"ignoreUnknownUsernames", want.IgnoreUnknownUsernames, p.IgnoreUnknownUsernames},
		{"allowDomainDiscovery", want.AllowDomainDiscovery, p.AllowDomainDiscovery},
		{"disableLoginWithEmail", want.DisableLoginWithEmail, p.DisableLoginWithEmail},
		{"disableLoginWithPhone", want.DisableLoginWithPhone, p.DisableLoginWithPhone},
	} {
		if f.want != f.got {
			return false
		}
	}

	for _, f := range []struct {
		name, want, got string
	}{
		{"defaultRedirectURI", want.DefaultRedirectURI, p.DefaultRedirectURI},
		{"passwordlessType", want.PasswordlessType, p.PasswordlessType},
	} {
		if f.want != f.got {
			return false
		}
	}

	for _, d := range []struct{ want, got string }{
		{want.PasswordCheckLifetime, p.PasswordCheckLifetime},
		{want.ExternalLoginCheckLifetime, p.ExternalLoginCheckLifetime},
		{want.MFAInitSkipLifetime, p.MFAInitSkipLifetime},
		{want.SecondFactorCheckLifetime, p.SecondFactorCheckLifetime},
		{want.MultiFactorCheckLifetime, p.MultiFactorCheckLifetime},
	} {
		if !sameDuration(d.want, d.got) {
			return false
		}
	}

	return sameStringSet(want.IDPs, p.IDPs) &&
		sameStringSet(want.SecondFactors, p.SecondFactors) &&
		sameStringSet(want.MultiFactors, p.MultiFactors)
}

func sameDuration(a, b string) bool {
	if a == b {
		return true
	}

	da, err := parseLifetime(a)
	if err != nil {
		return false
	}

	db, err := parseLifetime(b)
	if err != nil {
		return false
	}

	return da == db
}

// parseLifetime reads a lifetime, treating an unset value as zero.
//
// Zitadel reports a lifetime that was never configured as "0s", while an unset
// field in the spec is empty. Both spellings of "never" have to compare equal or
// every reconcile would see drift.
func parseLifetime(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	return time.ParseDuration(s)
}

// sameStringSet compares two string sets ignoring order.
func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	seen := make(map[string]int, len(a))
	for _, v := range a {
		seen[v]++
	}

	for _, v := range b {
		seen[v]--
		if seen[v] < 0 {
			return false
		}
	}

	return true
}

// updateStatus copies the observed policy into the status of cr.
func updateStatus(cr *v1alpha1.LoginPolicy, p zitadel.LoginPolicy) {
	cr.Status.AtProvider.IsDefault = common.BoolPtr(p.IsDefault)
	cr.Status.AtProvider.AllowUsernamePassword = common.BoolPtr(p.AllowUsernamePassword)
	cr.Status.AtProvider.AllowRegister = common.BoolPtr(p.AllowRegister)
	cr.Status.AtProvider.AllowExternalIDP = common.BoolPtr(p.AllowExternalIDP)
	cr.Status.AtProvider.ForceMFA = common.BoolPtr(p.ForceMFA)
	cr.Status.AtProvider.ForceMFALocalOnly = common.BoolPtr(p.ForceMFALocalOnly)
	cr.Status.AtProvider.HidePasswordReset = common.BoolPtr(p.HidePasswordReset)
	cr.Status.AtProvider.IgnoreUnknownUsernames = common.BoolPtr(p.IgnoreUnknownUsernames)
	cr.Status.AtProvider.AllowDomainDiscovery = common.BoolPtr(p.AllowDomainDiscovery)
	cr.Status.AtProvider.DisableLoginWithEmail = common.BoolPtr(p.DisableLoginWithEmail)
	cr.Status.AtProvider.DisableLoginWithPhone = common.BoolPtr(p.DisableLoginWithPhone)
	cr.Status.AtProvider.DefaultRedirectURI = common.StringPtr(p.DefaultRedirectURI)
	cr.Status.AtProvider.PasswordCheckLifetime = common.StringPtr(p.PasswordCheckLifetime)
	cr.Status.AtProvider.ExternalLoginCheckLifetime = common.StringPtr(p.ExternalLoginCheckLifetime)
	cr.Status.AtProvider.MFAInitSkipLifetime = common.StringPtr(p.MFAInitSkipLifetime)
	cr.Status.AtProvider.SecondFactorCheckLifetime = common.StringPtr(p.SecondFactorCheckLifetime)
	cr.Status.AtProvider.MultiFactorCheckLifetime = common.StringPtr(p.MultiFactorCheckLifetime)
	cr.Status.AtProvider.IDPs = p.IDPs

	if p.PasswordlessType != "" {
		pl := v1alpha1.PasswordlessType(p.PasswordlessType)
		cr.Status.AtProvider.PasswordlessType = &pl
	}

	cr.Status.AtProvider.SecondFactors = nil
	for _, f := range p.SecondFactors {
		cr.Status.AtProvider.SecondFactors = append(cr.Status.AtProvider.SecondFactors, v1alpha1.SecondFactor(f))
	}

	cr.Status.AtProvider.MultiFactors = nil
	for _, f := range p.MultiFactors {
		cr.Status.AtProvider.MultiFactors = append(cr.Status.AtProvider.MultiFactors, v1alpha1.MultiFactor(f))
	}
}
