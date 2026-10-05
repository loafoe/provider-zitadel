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

package default_domain_policy

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

// Setup adds a controller that reconciles DefaultDomainPolicy managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.DefaultDomainPolicyGroupKind,
		v1alpha1.DefaultDomainPolicyGroupVersionKind,
		&v1alpha1.DefaultDomainPolicy{},
		&v1alpha1.DefaultDomainPolicyList{},
		driver{},
	)
}

// driver is everything specific to the instance wide domain policy. Everything around it is
// shared, because every Zitadel policy behaves the same way.
type driver struct{}

var _ common.PolicyDriver[zitadel.DefaultDomainPolicyInput, zitadel.DefaultDomainPolicy] = driver{}

// Kind names the policy, for errors and events.
func (driver) Kind() string { return "DefaultDomainPolicy" }

// Scope resolves to the empty string, which is how the shared harness tells an
// instance wide policy from an organization's own: there is no organization, and
// so nothing to resolve.
func (driver) Scope(_ context.Context, _ client.Client, _ common.ManagedPolicy) (string, error) {
	return "", nil
}

// Get reads the instance wide domain policy.
func (driver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.DefaultDomainPolicy, bool, error) {
	p, err := c.GetDefaultDomainPolicy(ctx)
	if err != nil {
		return zitadel.DefaultDomainPolicy{}, false, err
	}

	return *p, false, nil
}

// Apply writes the policy.
func (driver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.DefaultDomainPolicyInput) error {
	return c.SetDefaultDomainPolicy(ctx, want)
}

// Resettable reports whether deleting this resource can put the %s
// back the way it was by resetting it.
func (driver) Resettable() bool { return false }

// Reset is only ever reached for a resettable policy, which an instance
// wide one is not: Zitadel has no endpoint that clears it.
func (driver) Reset(ctx context.Context, c *zitadel.Client, _ string) error {
	return errors.New("the instance wide domain policy cannot be reset; deleting the resource restores what it overwrote")
}

// Desired reads the desired policy out of the managed resource.
func (driver) Desired(cr common.ManagedPolicy) zitadel.DefaultDomainPolicyInput {
	fp := cr.(*v1alpha1.DefaultDomainPolicy).Spec.ForProvider

	return zitadel.DefaultDomainPolicyInput{
		UserLoginMustBeDomain:                  common.DerefBool(fp.UserLoginMustBeDomain),
		ValidateOrgDomains:                     common.DerefBool(fp.ValidateOrgDomains),
		SMTPSenderAddressMatchesInstanceDomain: common.DerefBool(fp.SMTPSenderAddressMatchesInstanceDomain),
	}
}

// Report writes the observed policy into the managed resource's status.
func (driver) Report(mg common.ManagedPolicy, observed zitadel.DefaultDomainPolicy) {
	cr := mg.(*v1alpha1.DefaultDomainPolicy)

	cr.Status.AtProvider.IsDefault = common.BoolPtr(false)

	cr.Status.AtProvider.UserLoginMustBeDomain = common.BoolPtr(observed.UserLoginMustBeDomain)
	cr.Status.AtProvider.ValidateOrgDomains = common.BoolPtr(observed.ValidateOrgDomains)
	cr.Status.AtProvider.SMTPSenderAddressMatchesInstanceDomain = common.BoolPtr(observed.SMTPSenderAddressMatchesInstanceDomain)
}

// Equal reports whether the observed policy already matches the desired one.
//
// A field the operator left unset is not compared, so Zitadel's own defaults are
// never fought over. Every field here is optional in the CRD, which is what makes
// that more than a convention: Desired turns an unset field into the zero value,
// so comparing it would report drift on every poll for a manifest that never
// asked for the field to be managed. See common.AllSetMatch.
func (driver) Equal(cr common.ManagedPolicy, want zitadel.DefaultDomainPolicyInput, observed zitadel.DefaultDomainPolicy) bool {
	fp := cr.(*v1alpha1.DefaultDomainPolicy).Spec.ForProvider

	return common.AllSetMatch(
		common.Match("UserLoginMustBeDomain", fp.UserLoginMustBeDomain != nil, want.UserLoginMustBeDomain, observed.UserLoginMustBeDomain),
		common.Match("ValidateOrgDomains", fp.ValidateOrgDomains != nil, want.ValidateOrgDomains, observed.ValidateOrgDomains),
		common.Match("SMTPSenderAddressMatchesInstanceDomain", fp.SMTPSenderAddressMatchesInstanceDomain != nil, want.SMTPSenderAddressMatchesInstanceDomain, observed.SMTPSenderAddressMatchesInstanceDomain),
	)
}
