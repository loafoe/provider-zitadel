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

package default_lockout_policy

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

// Setup adds a controller that reconciles DefaultLockoutPolicy managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.DefaultLockoutPolicyGroupKind,
		v1alpha1.DefaultLockoutPolicyGroupVersionKind,
		&v1alpha1.DefaultLockoutPolicy{},
		&v1alpha1.DefaultLockoutPolicyList{},
		driver{},
	)
}

// driver is everything specific to the instance wide lockout policy. Everything around it is
// shared, because every Zitadel policy behaves the same way.
type driver struct{}

var _ common.PolicyDriver[zitadel.DefaultLockoutPolicyInput, zitadel.DefaultLockoutPolicy] = driver{}

// Kind names the policy, for errors and events.
func (driver) Kind() string { return "DefaultLockoutPolicy" }

// Scope resolves to the empty string, which is how the shared harness tells an
// instance wide policy from an organization's own: there is no organization, and
// so nothing to resolve.
func (driver) Scope(_ context.Context, _ client.Client, _ common.ManagedPolicy) (string, error) {
	return "", nil
}

// Get reads the instance wide lockout policy.
func (driver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.DefaultLockoutPolicy, bool, error) {
	p, err := c.GetDefaultLockoutPolicy(ctx)
	if err != nil {
		return zitadel.DefaultLockoutPolicy{}, false, err
	}

	return *p, false, nil
}

// Apply writes the policy.
func (driver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.DefaultLockoutPolicyInput) error {
	return c.SetDefaultLockoutPolicy(ctx, want)
}

// Resettable reports whether deleting this resource can put the %s
// back the way it was by resetting it.
func (driver) Resettable() bool { return false }

// Reset is only ever reached for a resettable policy, which an instance
// wide one is not: Zitadel has no endpoint that clears it.
func (driver) Reset(ctx context.Context, c *zitadel.Client, _ string) error {
	return errors.New("the instance wide lockout policy cannot be reset; deleting the resource restores what it overwrote")
}

// Desired reads the desired policy out of the managed resource.
func (driver) Desired(cr common.ManagedPolicy) zitadel.DefaultLockoutPolicyInput {
	fp := cr.(*v1alpha1.DefaultLockoutPolicy).Spec.ForProvider

	return zitadel.DefaultLockoutPolicyInput{
		MaxPasswordAttempts: common.ToUint32(common.DerefInt64(fp.MaxPasswordAttempts)),
		MaxOTPAttempts:      common.ToUint32(common.DerefInt64(fp.MaxOTPAttempts)),
	}
}

// Report writes the observed policy into the managed resource's status.
func (driver) Report(mg common.ManagedPolicy, observed zitadel.DefaultLockoutPolicy) {
	cr := mg.(*v1alpha1.DefaultLockoutPolicy)

	cr.Status.AtProvider.IsDefault = common.BoolPtr(false)

	cr.Status.AtProvider.MaxPasswordAttempts = common.Int64Ptr(int64(observed.MaxPasswordAttempts))
	cr.Status.AtProvider.MaxOTPAttempts = common.Int64Ptr(int64(observed.MaxOTPAttempts))
}

// Equal reports whether the observed policy already matches the desired one.
//
// A field the operator left unset is not compared, so Zitadel's own defaults are
// never fought over.
func (driver) Equal(want zitadel.DefaultLockoutPolicyInput, observed zitadel.DefaultLockoutPolicy) bool {
	for _, f := range []struct {
		name string
		want uint32
		got  uint32
	}{
		{"MaxPasswordAttempts", want.MaxPasswordAttempts, observed.MaxPasswordAttempts},
		{"MaxOTPAttempts", want.MaxOTPAttempts, observed.MaxOTPAttempts},
	} {
		if f.want != f.got {
			return false
		}
	}

	return true
}
