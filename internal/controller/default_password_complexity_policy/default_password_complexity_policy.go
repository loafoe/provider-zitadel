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

package default_password_complexity_policy

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

// Setup adds a controller that reconciles DefaultPasswordComplexityPolicy managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.DefaultPasswordComplexityPolicyGroupKind,
		v1alpha1.DefaultPasswordComplexityPolicyGroupVersionKind,
		&v1alpha1.DefaultPasswordComplexityPolicy{},
		&v1alpha1.DefaultPasswordComplexityPolicyList{},
		driver{},
	)
}

// driver is everything specific to the instance wide password complexity policy. Everything around it is
// shared, because every Zitadel policy behaves the same way.
type driver struct{}

var _ common.PolicyDriver[zitadel.DefaultPasswordComplexityPolicyInput, zitadel.DefaultPasswordComplexityPolicy] = driver{}

// Kind names the policy, for errors and events.
func (driver) Kind() string { return "DefaultPasswordComplexityPolicy" }

// Scope resolves to the empty string, which is how the shared harness tells an
// instance wide policy from an organization's own: there is no organization, and
// so nothing to resolve.
func (driver) Scope(_ context.Context, _ client.Client, _ common.ManagedPolicy) (string, error) {
	return "", nil
}

// Get reads the instance wide password complexity policy.
func (driver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.DefaultPasswordComplexityPolicy, bool, error) {
	p, err := c.GetDefaultPasswordComplexityPolicy(ctx)
	if err != nil {
		return zitadel.DefaultPasswordComplexityPolicy{}, false, err
	}

	return *p, false, nil
}

// Apply writes the policy.
func (driver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.DefaultPasswordComplexityPolicyInput) error {
	return c.SetDefaultPasswordComplexityPolicy(ctx, want)
}

// Resettable reports whether deleting this resource can put the %s
// back the way it was by resetting it.
func (driver) Resettable() bool { return false }

// Reset is only ever reached for a resettable policy, which an instance
// wide one is not: Zitadel has no endpoint that clears it.
func (driver) Reset(ctx context.Context, c *zitadel.Client, _ string) error {
	return errors.New("the instance wide password complexity policy cannot be reset; deleting the resource restores what it overwrote")
}

// Desired reads the desired policy out of the managed resource.
func (driver) Desired(cr common.ManagedPolicy) zitadel.DefaultPasswordComplexityPolicyInput {
	fp := cr.(*v1alpha1.DefaultPasswordComplexityPolicy).Spec.ForProvider

	return zitadel.DefaultPasswordComplexityPolicyInput{
		MinLength:    common.ToUint32(common.DerefInt64(fp.MinLength)),
		HasUppercase: common.DerefBool(fp.HasUppercase),
		HasLowercase: common.DerefBool(fp.HasLowercase),
		HasNumber:    common.DerefBool(fp.HasNumber),
		HasSymbol:    common.DerefBool(fp.HasSymbol),
	}
}

// Report writes the observed policy into the managed resource's status.
func (driver) Report(mg common.ManagedPolicy, observed zitadel.DefaultPasswordComplexityPolicy) {
	cr := mg.(*v1alpha1.DefaultPasswordComplexityPolicy)

	cr.Status.AtProvider.IsDefault = common.BoolPtr(false)

	cr.Status.AtProvider.MinLength = common.Int64Ptr(int64(observed.MinLength))
	cr.Status.AtProvider.HasUppercase = common.BoolPtr(observed.HasUppercase)
	cr.Status.AtProvider.HasLowercase = common.BoolPtr(observed.HasLowercase)
	cr.Status.AtProvider.HasNumber = common.BoolPtr(observed.HasNumber)
	cr.Status.AtProvider.HasSymbol = common.BoolPtr(observed.HasSymbol)
}

// Equal reports whether the observed policy already matches the desired one.
//
// A field the operator left unset is not compared, so Zitadel's own defaults are
// never fought over. Every field here is optional in the CRD, which is what makes
// that more than a convention: Desired turns an unset field into the zero value,
// so comparing it would report drift on every poll for a manifest that never
// asked for the field to be managed. See common.AllSetMatch.
func (driver) Equal(cr common.ManagedPolicy, want zitadel.DefaultPasswordComplexityPolicyInput, observed zitadel.DefaultPasswordComplexityPolicy) bool {
	fp := cr.(*v1alpha1.DefaultPasswordComplexityPolicy).Spec.ForProvider

	return common.AllSetMatch(
		common.Match("HasUppercase", fp.HasUppercase != nil, want.HasUppercase, observed.HasUppercase),
		common.Match("HasLowercase", fp.HasLowercase != nil, want.HasLowercase, observed.HasLowercase),
		common.Match("HasNumber", fp.HasNumber != nil, want.HasNumber, observed.HasNumber),
		common.Match("HasSymbol", fp.HasSymbol != nil, want.HasSymbol, observed.HasSymbol),
	) &&
		common.AllSetMatch(
			common.Match("MinLength", fp.MinLength != nil, want.MinLength, observed.MinLength),
		)
}
