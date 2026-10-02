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

package label_policy

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// Setup adds a controller that reconciles LabelPolicy managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.LabelPolicyGroupKind,
		v1alpha1.LabelPolicyGroupVersionKind,
		&v1alpha1.LabelPolicy{},
		&v1alpha1.LabelPolicyList{},
		driver{},
	)
}

// driver is everything specific to the branding policy. Everything around it is
// shared, because every Zitadel policy behaves the same way.
type driver struct{}

var _ common.PolicyDriver[zitadel.LabelPolicyInput, zitadel.LabelPolicy] = driver{}

// Kind names the policy, for errors and events.
func (driver) Kind() string { return "LabelPolicy" }

// Scope resolves the organization the policy belongs to. A policy is always
// an organization's own, so an unresolved reference is an error rather than a
// request for the instance default.
func (driver) Scope(ctx context.Context, kube client.Client, cr common.ManagedPolicy) (string, error) {
	policy, ok := cr.(*v1alpha1.LabelPolicy)
	if !ok {
		return "", errors.New("managed resource is not a LabelPolicy custom resource")
	}

	fp := policy.Spec.ForProvider

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, kube, policy)
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	// A terminating object acts on the organization it recorded: it exists to
	// reset the policy it set, not one belonging to a replacement.
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

// Get reads the branding policy.
func (driver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.LabelPolicy, bool, error) {
	p, err := c.GetLabelPolicy(ctx, scope)
	if err != nil {
		return zitadel.LabelPolicy{}, false, err
	}

	return *p, false, nil
}

// Apply writes the policy.
func (driver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.LabelPolicyInput) error {
	return c.SetLabelPolicy(ctx, scope, want)
}

// Resettable reports whether deleting this resource can put the %s
// back the way it was by resetting it.
func (driver) Resettable() bool { return true }

// Reset is only ever reached for a resettable policy, which an instance
// wide one is not: Zitadel has no endpoint that clears it.
func (driver) Reset(ctx context.Context, c *zitadel.Client, _ string) error {
	return errors.New("the branding policy cannot be reset; deleting the resource restores what it overwrote")
}

// Desired reads the desired policy out of the managed resource.
func (driver) Desired(cr common.ManagedPolicy) zitadel.LabelPolicyInput {
	fp := cr.(*v1alpha1.LabelPolicy).Spec.ForProvider

	return zitadel.LabelPolicyInput{
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
func (driver) Report(mg common.ManagedPolicy, observed zitadel.LabelPolicy) {
	cr := mg.(*v1alpha1.LabelPolicy)

	cr.Status.AtProvider.OrganizationID = common.StringPtr(observed.OrgID)
	cr.Status.AtProvider.IsDefault = common.BoolPtr(observed.IsDefault)

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
// never fought over.
func (driver) Equal(want zitadel.LabelPolicyInput, observed zitadel.LabelPolicy) bool {
	for _, f := range []struct {
		name string
		want bool
		got  bool
	}{
		{"HideLoginNameSuffix", want.HideLoginNameSuffix, observed.HideLoginNameSuffix},
		{"DisableWatermark", want.DisableWatermark, observed.DisableWatermark},
	} {
		if f.want != f.got {
			return false
		}
	}

	for _, f := range []struct {
		name string
		want string
		got  string
	}{
		{"ThemeMode", want.ThemeMode, observed.ThemeMode},
	} {
		if f.want != f.got {
			return false
		}
	}

	for _, f := range []struct {
		name string
		want string
		got  string
	}{
		{"PrimaryColor", want.PrimaryColor, observed.PrimaryColor},
		{"WarnColor", want.WarnColor, observed.WarnColor},
		{"BackgroundColor", want.BackgroundColor, observed.BackgroundColor},
		{"FontColor", want.FontColor, observed.FontColor},
		{"PrimaryColorDark", want.PrimaryColorDark, observed.PrimaryColorDark},
		{"WarnColorDark", want.WarnColorDark, observed.WarnColorDark},
		{"BackgroundColorDark", want.BackgroundColorDark, observed.BackgroundColorDark},
		{"FontColorDark", want.FontColorDark, observed.FontColorDark},
	} {
		if f.want != f.got {
			return false
		}
	}

	return true
}
