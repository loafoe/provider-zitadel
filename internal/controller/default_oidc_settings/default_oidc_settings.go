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

package default_oidc_settings

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

// Setup adds a controller that reconciles DefaultOIDCSettings managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.DefaultOIDCSettingsGroupKind,
		v1alpha1.DefaultOIDCSettingsGroupVersionKind,
		&v1alpha1.DefaultOIDCSettings{},
		&v1alpha1.DefaultOIDCSettingsList{},
		driver{},
	)
}

// driver is everything specific to the instance wide OIDC token lifetimes. Everything around it is
// shared, because every Zitadel policy behaves the same way.
type driver struct{}

var _ common.PolicyDriver[zitadel.DefaultOIDCSettingsInput, zitadel.DefaultOIDCSettings] = driver{}

// Kind names the policy, for errors and events.
func (driver) Kind() string { return "DefaultOIDCSettings" }

// Scope resolves to the empty string, which is how the shared harness tells an
// instance wide policy from an organization's own: there is no organization, and
// so nothing to resolve.
func (driver) Scope(_ context.Context, _ client.Client, _ common.ManagedPolicy) (string, error) {
	return "", nil
}

// Get reads the instance wide OIDC token lifetimes.
func (driver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.DefaultOIDCSettings, bool, error) {
	p, err := c.GetDefaultOIDCSettings(ctx)
	if err != nil {
		return zitadel.DefaultOIDCSettings{}, false, err
	}

	return *p, false, nil
}

// Apply writes the policy.
func (driver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.DefaultOIDCSettingsInput) error {
	return c.SetDefaultOIDCSettings(ctx, want)
}

// Resettable reports whether deleting this resource can put the %s
// back the way it was by resetting it.
func (driver) Resettable() bool { return false }

// Reset is only ever reached for a resettable policy, which an instance
// wide one is not: Zitadel has no endpoint that clears it.
func (driver) Reset(ctx context.Context, c *zitadel.Client, _ string) error {
	return errors.New("the instance wide OIDC token lifetimes cannot be reset; deleting the resource restores what it overwrote")
}

// Desired reads the desired policy out of the managed resource.
func (driver) Desired(cr common.ManagedPolicy) zitadel.DefaultOIDCSettingsInput {
	fp := cr.(*v1alpha1.DefaultOIDCSettings).Spec.ForProvider

	return zitadel.DefaultOIDCSettingsInput{
		AccessTokenLifetime:        common.Deref(fp.AccessTokenLifetime),
		IDTokenLifetime:            common.Deref(fp.IDTokenLifetime),
		RefreshTokenExpiration:     common.Deref(fp.RefreshTokenExpiration),
		RefreshTokenIdleExpiration: common.Deref(fp.RefreshTokenIdleExpiration),
	}
}

// Report writes the observed policy into the managed resource's status.
func (driver) Report(mg common.ManagedPolicy, observed zitadel.DefaultOIDCSettings) {
	cr := mg.(*v1alpha1.DefaultOIDCSettings)

	cr.Status.AtProvider.IsDefault = common.BoolPtr(false)

	cr.Status.AtProvider.AccessTokenLifetime = common.StringPtr(observed.AccessTokenLifetime)
	cr.Status.AtProvider.IDTokenLifetime = common.StringPtr(observed.IDTokenLifetime)
	cr.Status.AtProvider.RefreshTokenExpiration = common.StringPtr(observed.RefreshTokenExpiration)
	cr.Status.AtProvider.RefreshTokenIdleExpiration = common.StringPtr(observed.RefreshTokenIdleExpiration)
}

// Equal reports whether the observed policy already matches the desired one.
//
// A field the operator left unset is not compared, so Zitadel's own defaults are
// never fought over.
func (driver) Equal(want zitadel.DefaultOIDCSettingsInput, observed zitadel.DefaultOIDCSettings) bool {
	for _, f := range []struct {
		name string
		want string
		got  string
	}{
		{"AccessTokenLifetime", want.AccessTokenLifetime, observed.AccessTokenLifetime},
		{"IDTokenLifetime", want.IDTokenLifetime, observed.IDTokenLifetime},
		{"RefreshTokenExpiration", want.RefreshTokenExpiration, observed.RefreshTokenExpiration},
		{"RefreshTokenIdleExpiration", want.RefreshTokenIdleExpiration, observed.RefreshTokenIdleExpiration},
	} {
		if f.want != f.got {
			return false
		}
	}

	return true
}
