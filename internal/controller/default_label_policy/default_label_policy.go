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

package default_label_policy

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

// Setup adds a controller that reconciles DefaultLabelPolicy managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.DefaultLabelPolicyGroupKind,
		v1alpha1.DefaultLabelPolicyGroupVersionKind,
		&v1alpha1.DefaultLabelPolicy{},
		&v1alpha1.DefaultLabelPolicyList{},
		driver{},
	)
}

// driver is everything specific to the instance wide branding policy. Everything around it is
// shared, because every Zitadel policy behaves the same way.
type driver struct{}

var _ common.PolicyDriver[zitadel.DefaultLabelPolicyInput, zitadel.DefaultLabelPolicy] = driver{}

// Kind names the policy, for errors and events.
func (driver) Kind() string { return "DefaultLabelPolicy" }

// Scope resolves to the empty string, which is how the shared harness tells an
// instance wide policy from an organization's own: there is no organization, and
// so nothing to resolve.
func (driver) Scope(_ context.Context, _ client.Client, _ common.ManagedPolicy) (string, error) {
	return "", nil
}

// Get reads the instance wide branding policy.
func (driver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.DefaultLabelPolicy, bool, error) {
	p, err := c.GetDefaultLabelPolicy(ctx)
	if err != nil {
		return zitadel.DefaultLabelPolicy{}, false, err
	}

	return *p, false, nil
}

// Apply writes the policy.
func (driver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.DefaultLabelPolicyInput) error {
	return c.SetDefaultLabelPolicy(ctx, want)
}

// Resettable reports whether deleting this resource can put the %s
// back the way it was by resetting it.
func (driver) Resettable() bool { return false }

// Reset is only ever reached for a resettable policy, which an instance
// wide one is not: Zitadel has no endpoint that clears it.
func (driver) Reset(ctx context.Context, c *zitadel.Client, _ string) error {
	return errors.New("the instance wide branding policy cannot be reset; deleting the resource restores what it overwrote")
}

// Desired reads the desired policy out of the managed resource.
func (driver) Desired(cr common.ManagedPolicy) zitadel.DefaultLabelPolicyInput {
	fp := cr.(*v1alpha1.DefaultLabelPolicy).Spec.ForProvider

	return zitadel.DefaultLabelPolicyInput{
		PrimaryColor:        common.Deref(fp.PrimaryColor),
		WarnColor:           common.Deref(fp.WarnColor),
		BackgroundColor:     common.Deref(fp.BackgroundColor),
		FontColor:           common.Deref(fp.FontColor),
		PrimaryColorDark:    common.Deref(fp.PrimaryColorDark),
		WarnColorDark:       common.Deref(fp.WarnColorDark),
		BackgroundColorDark: common.Deref(fp.BackgroundColorDark),
		FontColorDark:       common.Deref(fp.FontColorDark),
		HideLoginNameSuffix: common.DerefBool(fp.HideLoginNameSuffix),
		DisableWatermark:    common.DerefBool(fp.DisableWatermark),
		ThemeMode:           common.Value(fp.ThemeMode),
	}
}

// Report writes the observed policy into the managed resource's status.
func (driver) Report(mg common.ManagedPolicy, observed zitadel.DefaultLabelPolicy) {
	cr := mg.(*v1alpha1.DefaultLabelPolicy)

	cr.Status.AtProvider.IsDefault = common.BoolPtr(false)

	cr.Status.AtProvider.PrimaryColor = common.StringPtr(observed.PrimaryColor)
	cr.Status.AtProvider.WarnColor = common.StringPtr(observed.WarnColor)
	cr.Status.AtProvider.BackgroundColor = common.StringPtr(observed.BackgroundColor)
	cr.Status.AtProvider.FontColor = common.StringPtr(observed.FontColor)
	cr.Status.AtProvider.PrimaryColorDark = common.StringPtr(observed.PrimaryColorDark)
	cr.Status.AtProvider.WarnColorDark = common.StringPtr(observed.WarnColorDark)
	cr.Status.AtProvider.BackgroundColorDark = common.StringPtr(observed.BackgroundColorDark)
	cr.Status.AtProvider.FontColorDark = common.StringPtr(observed.FontColorDark)
	cr.Status.AtProvider.HideLoginNameSuffix = common.BoolPtr(observed.HideLoginNameSuffix)
	cr.Status.AtProvider.DisableWatermark = common.BoolPtr(observed.DisableWatermark)
	if observed.ThemeMode != "" {
		value := v1alpha1.LabelThemeMode(observed.ThemeMode)
		cr.Status.AtProvider.ThemeMode = &value
	}
	cr.Status.AtProvider.LogoURL = common.StringPtr(observed.LogoURL)
	cr.Status.AtProvider.IconURL = common.StringPtr(observed.IconURL)
	cr.Status.AtProvider.LogoDarkURL = common.StringPtr(observed.LogoDarkURL)
	cr.Status.AtProvider.IconDarkURL = common.StringPtr(observed.IconDarkURL)
	cr.Status.AtProvider.FontURL = common.StringPtr(observed.FontURL)
}

// Equal reports whether the observed policy already matches the desired one.
//
// A field the operator left unset is not compared, so Zitadel's own defaults are
// never fought over. Every field here is optional in the CRD, which is what makes
// that more than a convention: Desired turns an unset field into the zero value,
// so comparing it would report drift on every poll for a manifest that never
// asked for the field to be managed. See common.AllSetMatch.
func (driver) Equal(cr common.ManagedPolicy, want zitadel.DefaultLabelPolicyInput, observed zitadel.DefaultLabelPolicy) bool {
	fp := cr.(*v1alpha1.DefaultLabelPolicy).Spec.ForProvider

	return common.AllSetMatch(
		common.Match("HideLoginNameSuffix", fp.HideLoginNameSuffix != nil, want.HideLoginNameSuffix, observed.HideLoginNameSuffix),
		common.Match("DisableWatermark", fp.DisableWatermark != nil, want.DisableWatermark, observed.DisableWatermark),
	) &&
		common.AllSetMatch(
			common.Match("ThemeMode", fp.ThemeMode != nil, want.ThemeMode, observed.ThemeMode),
		) &&
		common.AllSetMatch(
			common.Match("PrimaryColor", fp.PrimaryColor != nil, want.PrimaryColor, observed.PrimaryColor),
			common.Match("WarnColor", fp.WarnColor != nil, want.WarnColor, observed.WarnColor),
			common.Match("BackgroundColor", fp.BackgroundColor != nil, want.BackgroundColor, observed.BackgroundColor),
			common.Match("FontColor", fp.FontColor != nil, want.FontColor, observed.FontColor),
			common.Match("PrimaryColorDark", fp.PrimaryColorDark != nil, want.PrimaryColorDark, observed.PrimaryColorDark),
			common.Match("WarnColorDark", fp.WarnColorDark != nil, want.WarnColorDark, observed.WarnColorDark),
			common.Match("BackgroundColorDark", fp.BackgroundColorDark != nil, want.BackgroundColorDark, observed.BackgroundColorDark),
			common.Match("FontColorDark", fp.FontColorDark != nil, want.FontColorDark, observed.FontColorDark),
		)
}
