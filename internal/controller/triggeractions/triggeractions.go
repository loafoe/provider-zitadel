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

package triggeractions

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

// trigger identifies one point in one login flow. It is the identity of a
// TriggerActions resource, and what Zitadel addresses it by.
type trigger struct {
	OrgID       string
	FlowType    string
	TriggerType string
}

// Setup adds a controller that reconciles TriggerActions managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupBindingController(mgr, o,
		v1alpha1.TriggerActionsGroupKind,
		v1alpha1.TriggerActionsGroupVersionKind,
		&v1alpha1.TriggerActions{}, &v1alpha1.TriggerActionsList{},
		driver{kube: mgr.GetClient()})
}

// driver is everything specific to a login flow trigger.
//
// It is the same shape as an action execution - a condition, a list of things
// to call, and no identity of its own - which is why it runs on the shared
// binding harness.
type driver struct {
	kube client.Client
}

var _ common.BindingDriver[*v1alpha1.TriggerActions, trigger] = driver{}

// Kind names the binding, for errors and events.
func (driver) Kind() string { return "TriggerActions" }

// Condition returns the trigger this resource binds on, and the identity derived
// from it.
func (d driver) Condition(cr *v1alpha1.TriggerActions) (trigger, string, error) {
	fp := cr.Spec.ForProvider

	if fp.FlowType == nil || *fp.FlowType == "" {
		return trigger{}, "", errors.New("flowType must be set")
	}

	if fp.TriggerType == nil || *fp.TriggerType == "" {
		return trigger{}, "", errors.New("triggerType must be set")
	}

	orgID, err := d.organizationID(cr)
	if err != nil {
		return trigger{}, "", err
	}

	flow, point := string(*fp.FlowType), string(*fp.TriggerType)

	return trigger{OrgID: orgID, FlowType: flow, TriggerType: point},
		common.DeriveTriggerID(flow, point), nil
}

// organizationID resolves the organization whose login flow is customised.
//
// A terminating resource acts on the organization it recorded: it exists to
// clear the trigger it set, not one belonging to a replacement.
func (d driver) organizationID(cr *v1alpha1.TriggerActions) (string, error) {
	fp := cr.Spec.ForProvider

	if meta.WasDeleted(cr) {
		if orgID := common.Deref(cr.Status.AtProvider.OrganizationID); orgID != "" {
			return orgID, nil
		}
	}

	ctx := context.Background()

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, d.kube, cr)
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	orgID, err := common.ResolveOrganizationID(ctx, d.kube, cr, fp.OrganizationRef,
		fp.OrganizationSelector, fp.OrganizationID, orgDefault,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.OrganizationID))
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	if orgID == "" {
		return "", errors.New(common.ErrNoOrganizationID.Error())
	}

	return orgID, nil
}

// Get reads the actions bound to the trigger.
//
// Zitadel has no call for listing them, so they are read out of the flow and
// the wanted trigger picked from it. A flow with no such trigger has nothing
// bound, which is not an error.
func (driver) Get(ctx context.Context, c *zitadel.Client, t trigger) ([]string, bool, error) {
	actions, err := c.GetTriggerActions(ctx, t.OrgID, t.FlowType, t.TriggerType)
	if err != nil {
		return nil, false, err
	}

	return actions, len(actions) > 0, nil
}

// Set writes the trigger's whole action list, which is the only thing Zitadel
// can do with one.
func (driver) Set(ctx context.Context, c *zitadel.Client, t trigger, actionIDs []string) error {
	return c.SetTriggerActions(ctx, t.OrgID, t.FlowType, t.TriggerType, actionIDs)
}

// Targets reads the desired actions out of the managed resource.
func (driver) Targets(cr *v1alpha1.TriggerActions) []string { return cr.Spec.ForProvider.ActionIDs }

// Report writes the observed binding into the managed resource's status.
func (driver) Report(cr *v1alpha1.TriggerActions, actionIDs []string) {
	cr.Status.AtProvider.ActionIDs = actionIDs
}

// Equal compares the desired actions with the observed ones, as sets: which
// actions run, not what order Zitadel lists them in.
func (driver) Equal(want, got []string) bool { return common.EqualStringSlices(want, got) }
