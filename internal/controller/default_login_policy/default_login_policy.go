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

package default_login_policy

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// Setup adds a controller that reconciles DefaultLoginPolicy managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.DefaultLoginPolicyGroupKind,
		v1alpha1.DefaultLoginPolicyGroupVersionKind,
		&v1alpha1.DefaultLoginPolicy{},
		&v1alpha1.DefaultLoginPolicyList{},
		driver{},
	)
}

// driver is everything specific to the instance wide login policy. Everything around it is
// shared, because every Zitadel policy behaves the same way.
type driver struct{}

var _ common.PolicyDriver[zitadel.DefaultLoginPolicyInput, zitadel.DefaultLoginPolicy] = driver{}

// Kind names the policy, for errors and events.
func (driver) Kind() string { return "DefaultLoginPolicy" }

// Scope resolves to the empty string, which is how the shared harness tells an
// instance wide policy from an organization's own: there is no organization, and
// so nothing to resolve.
func (driver) Scope(_ context.Context, _ client.Client, _ common.ManagedPolicy) (string, error) {
	return "", nil
}

// Get reads the instance wide login policy.
func (driver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.DefaultLoginPolicy, bool, error) {
	p, err := c.GetDefaultLoginPolicy(ctx)
	if err != nil {
		return zitadel.DefaultLoginPolicy{}, false, err
	}

	return *p, false, nil
}

// Apply writes the policy.
func (driver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.DefaultLoginPolicyInput) error {
	return c.SetDefaultLoginPolicy(ctx, want)
}

// Resettable reports whether deleting this resource can put the %s
// back the way it was by resetting it.
func (driver) Resettable() bool { return false }

// Reset is only ever reached for a resettable policy, which an instance
// wide one is not: Zitadel has no endpoint that clears it.
func (driver) Reset(ctx context.Context, c *zitadel.Client, _ string) error {
	return errors.New("the instance wide login policy cannot be reset; deleting the resource restores what it overwrote")
}

// Desired reads the desired policy out of the managed resource.
func (driver) Desired(cr common.ManagedPolicy) zitadel.DefaultLoginPolicyInput {
	fp := cr.(*v1alpha1.DefaultLoginPolicy).Spec.ForProvider

	return zitadel.DefaultLoginPolicyInput{
		AllowUsernamePassword:      common.DerefBool(fp.AllowUsernamePassword),
		AllowRegister:              common.DerefBool(fp.AllowRegister),
		AllowExternalIDP:           common.DerefBool(fp.AllowExternalIDP),
		ForceMFA:                   common.DerefBool(fp.ForceMFA),
		ForceMFALocalOnly:          common.DerefBool(fp.ForceMFALocalOnly),
		HidePasswordReset:          common.DerefBool(fp.HidePasswordReset),
		IgnoreUnknownUsernames:     common.DerefBool(fp.IgnoreUnknownUsernames),
		AllowDomainDiscovery:       common.DerefBool(fp.AllowDomainDiscovery),
		DisableLoginWithEmail:      common.DerefBool(fp.DisableLoginWithEmail),
		DisableLoginWithPhone:      common.DerefBool(fp.DisableLoginWithPhone),
		DefaultRedirectURI:         common.Deref(fp.DefaultRedirectURI),
		PasswordlessType:           common.Value(fp.PasswordlessType),
		PasswordCheckLifetime:      common.Deref(fp.PasswordCheckLifetime),
		ExternalLoginCheckLifetime: common.Deref(fp.ExternalLoginCheckLifetime),
		MFAInitSkipLifetime:        common.Deref(fp.MFAInitSkipLifetime),
		SecondFactorCheckLifetime:  common.Deref(fp.SecondFactorCheckLifetime),
		MultiFactorCheckLifetime:   common.Deref(fp.MultiFactorCheckLifetime),
		SecondFactors:              common.EnumNames(fp.SecondFactors),
		MultiFactors:               common.EnumNames(fp.MultiFactors),
	}
}

// Report writes the observed policy into the managed resource's status.
func (driver) Report(mg common.ManagedPolicy, observed zitadel.DefaultLoginPolicy) {
	cr := mg.(*v1alpha1.DefaultLoginPolicy)

	cr.Status.AtProvider.IsDefault = common.BoolPtr(false)

	cr.Status.AtProvider.AllowUsernamePassword = common.BoolPtr(observed.AllowUsernamePassword)
	cr.Status.AtProvider.AllowRegister = common.BoolPtr(observed.AllowRegister)
	cr.Status.AtProvider.AllowExternalIDP = common.BoolPtr(observed.AllowExternalIDP)
	cr.Status.AtProvider.ForceMFA = common.BoolPtr(observed.ForceMFA)
	cr.Status.AtProvider.ForceMFALocalOnly = common.BoolPtr(observed.ForceMFALocalOnly)
	cr.Status.AtProvider.HidePasswordReset = common.BoolPtr(observed.HidePasswordReset)
	cr.Status.AtProvider.IgnoreUnknownUsernames = common.BoolPtr(observed.IgnoreUnknownUsernames)
	cr.Status.AtProvider.AllowDomainDiscovery = common.BoolPtr(observed.AllowDomainDiscovery)
	cr.Status.AtProvider.DisableLoginWithEmail = common.BoolPtr(observed.DisableLoginWithEmail)
	cr.Status.AtProvider.DisableLoginWithPhone = common.BoolPtr(observed.DisableLoginWithPhone)
	cr.Status.AtProvider.DefaultRedirectURI = common.StringPtr(observed.DefaultRedirectURI)
	if observed.PasswordlessType != "" {
		value := v1alpha1.PasswordlessType(observed.PasswordlessType)
		cr.Status.AtProvider.PasswordlessType = &value
	}
	cr.Status.AtProvider.PasswordCheckLifetime = common.StringPtr(observed.PasswordCheckLifetime)
	cr.Status.AtProvider.ExternalLoginCheckLifetime = common.StringPtr(observed.ExternalLoginCheckLifetime)
	cr.Status.AtProvider.MFAInitSkipLifetime = common.StringPtr(observed.MFAInitSkipLifetime)
	cr.Status.AtProvider.SecondFactorCheckLifetime = common.StringPtr(observed.SecondFactorCheckLifetime)
	cr.Status.AtProvider.MultiFactorCheckLifetime = common.StringPtr(observed.MultiFactorCheckLifetime)
	cr.Status.AtProvider.SecondFactors = common.EnumValues[v1alpha1.SecondFactor](observed.SecondFactors)
	cr.Status.AtProvider.MultiFactors = common.EnumValues[v1alpha1.MultiFactor](observed.MultiFactors)
}

// Equal reports whether the observed policy already matches the desired one.
//
// A field the operator left unset is not compared, so Zitadel's own defaults are
// never fought over. Every field here is optional in the CRD, which is what makes
// that more than a convention: Desired turns an unset field into the zero value,
// so comparing it would report drift on every poll for a manifest that never
// asked for the field to be managed. See common.AllSetMatch.
func (driver) Equal(cr common.ManagedPolicy, want zitadel.DefaultLoginPolicyInput, observed zitadel.DefaultLoginPolicy) bool {
	fp := cr.(*v1alpha1.DefaultLoginPolicy).Spec.ForProvider

	return common.AllSetMatch(
		common.Match("AllowUsernamePassword", fp.AllowUsernamePassword != nil, want.AllowUsernamePassword, observed.AllowUsernamePassword),
		common.Match("AllowRegister", fp.AllowRegister != nil, want.AllowRegister, observed.AllowRegister),
		common.Match("AllowExternalIDP", fp.AllowExternalIDP != nil, want.AllowExternalIDP, observed.AllowExternalIDP),
		common.Match("ForceMFA", fp.ForceMFA != nil, want.ForceMFA, observed.ForceMFA),
		common.Match("ForceMFALocalOnly", fp.ForceMFALocalOnly != nil, want.ForceMFALocalOnly, observed.ForceMFALocalOnly),
		common.Match("HidePasswordReset", fp.HidePasswordReset != nil, want.HidePasswordReset, observed.HidePasswordReset),
		common.Match("IgnoreUnknownUsernames", fp.IgnoreUnknownUsernames != nil, want.IgnoreUnknownUsernames, observed.IgnoreUnknownUsernames),
		common.Match("AllowDomainDiscovery", fp.AllowDomainDiscovery != nil, want.AllowDomainDiscovery, observed.AllowDomainDiscovery),
		common.Match("DisableLoginWithEmail", fp.DisableLoginWithEmail != nil, want.DisableLoginWithEmail, observed.DisableLoginWithEmail),
		common.Match("DisableLoginWithPhone", fp.DisableLoginWithPhone != nil, want.DisableLoginWithPhone, observed.DisableLoginWithPhone),
	) &&
		common.AllSetMatch(
			common.Match("PasswordlessType", fp.PasswordlessType != nil, want.PasswordlessType, observed.PasswordlessType),
			common.Match("DefaultRedirectURI", fp.DefaultRedirectURI != nil, want.DefaultRedirectURI, observed.DefaultRedirectURI),
			common.Match("PasswordCheckLifetime", fp.PasswordCheckLifetime != nil, want.PasswordCheckLifetime, observed.PasswordCheckLifetime),
			common.Match("ExternalLoginCheckLifetime", fp.ExternalLoginCheckLifetime != nil, want.ExternalLoginCheckLifetime, observed.ExternalLoginCheckLifetime),
			common.Match("MFAInitSkipLifetime", fp.MFAInitSkipLifetime != nil, want.MFAInitSkipLifetime, observed.MFAInitSkipLifetime),
			common.Match("SecondFactorCheckLifetime", fp.SecondFactorCheckLifetime != nil, want.SecondFactorCheckLifetime, observed.SecondFactorCheckLifetime),
			common.Match("MultiFactorCheckLifetime", fp.MultiFactorCheckLifetime != nil, want.MultiFactorCheckLifetime, observed.MultiFactorCheckLifetime),
		) &&
		common.AllSetMatchLists(
			common.MatchList("SecondFactors", fp.SecondFactors != nil, want.SecondFactors, observed.SecondFactors),
			common.MatchList("MultiFactors", fp.MultiFactors != nil, want.MultiFactors, observed.MultiFactors),
		)
}
