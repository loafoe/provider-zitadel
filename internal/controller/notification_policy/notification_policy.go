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

package notification_policy

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

// Setup adds a controller that reconciles NotificationPolicy managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.NotificationPolicyGroupKind,
		v1alpha1.NotificationPolicyGroupVersionKind,
		&v1alpha1.NotificationPolicy{},
		&v1alpha1.NotificationPolicyList{},
		driver{},
	)
}

// driver is everything specific to the notification policy. Everything around it is
// shared, because every Zitadel policy behaves the same way.
type driver struct{}

var _ common.PolicyDriver[zitadel.NotificationPolicyInput, zitadel.NotificationPolicy] = driver{}

// Kind names the policy, for errors and events.
func (driver) Kind() string { return "NotificationPolicy" }

// Scope resolves the organization the policy belongs to. A policy is always
// an organization's own, so an unresolved reference is an error rather than a
// request for the instance default.
func (driver) Scope(ctx context.Context, kube client.Client, cr common.ManagedPolicy) (string, error) {
	policy, ok := cr.(*v1alpha1.NotificationPolicy)
	if !ok {
		return "", errors.New("managed resource is not a NotificationPolicy custom resource")
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

// Get reads the organization's notification policy.
func (driver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.NotificationPolicy, bool, error) {
	p, err := c.GetNotificationPolicy(ctx, scope)
	if err != nil {
		return zitadel.NotificationPolicy{}, false, err
	}

	return *p, p.IsDefault, nil
}

// Apply writes the policy, adding or updating it as Zitadel requires.
//
// Zitadel separates adding a custom policy from updating one, and which to use
// depends on whether the organization has one. The client works that out, so a
// driver does not have to.
func (driver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.NotificationPolicyInput) error {
	return c.SetNotificationPolicy(ctx, scope, want)
}

// Reset puts the organization back on the instance default notification policy.
// Resettable reports that deleting this resource can put the organization
// back on the instance default, which is how Zitadel removes a custom policy.
func (driver) Resettable() bool { return true }

func (driver) Reset(ctx context.Context, c *zitadel.Client, scope string) error {
	return c.ResetNotificationPolicy(ctx, scope)
}

// Desired reads the desired policy out of the managed resource.
func (driver) Desired(cr common.ManagedPolicy) zitadel.NotificationPolicyInput {
	fp := cr.(*v1alpha1.NotificationPolicy).Spec.ForProvider

	return zitadel.NotificationPolicyInput{
		PasswordChange: common.DerefBool(fp.PasswordChange),
	}
}

// Report writes the observed policy into the managed resource's status.
func (driver) Report(mg common.ManagedPolicy, observed zitadel.NotificationPolicy) {
	cr := mg.(*v1alpha1.NotificationPolicy)

	cr.Status.AtProvider.OrganizationID = common.StringPtr(observed.OrgID)
	cr.Status.AtProvider.IsDefault = common.BoolPtr(observed.IsDefault)

	cr.Status.AtProvider.PasswordChange = common.BoolPtr(observed.PasswordChange)
}

// Equal reports whether the observed policy already matches the desired one.
//
// A field the operator left unset is not compared, so Zitadel's own defaults are
// never fought over.
func (driver) Equal(want zitadel.NotificationPolicyInput, observed zitadel.NotificationPolicy) bool {
	for _, f := range []struct {
		name string
		want bool
		got  bool
	}{
		{"PasswordChange", want.PasswordChange, observed.PasswordChange},
	} {
		if f.want != f.got {
			return false
		}
	}

	return true
}
