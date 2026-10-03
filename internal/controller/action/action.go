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

package action

import (
	"context"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// Setup adds a controller that reconciles Action managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ActionGroupKind,
		v1alpha1.ActionGroupVersionKind,
		&v1alpha1.Action{},
		&v1alpha1.ActionList{},
		newExternal,
	)
}

type external struct {
	kube   client.Client
	client *zitadel.Client
}

func newExternal(_ context.Context, kube client.Client, _ resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
	return &external{kube: kube, client: zc}, nil
}

// Disconnect releases the underlying Zitadel client.
func (e *external) Disconnect(_ context.Context) error {
	return e.client.Close()
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Action)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotAction)
	}

	orgID, err := e.organizationID(ctx, cr)
	if err != nil {
		if meta.WasDeleted(cr) {
			// Nothing resolves while an object is terminating, so report the
			// action as gone and let the finalizer go.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	a, err := e.client.GetAction(ctx, orgID, id)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the action from Zitadel")
	}

	updateStatus(cr, orgID, a)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists: true,
		// Zitadel never returns an action's script, so there is nothing to
		// compare it against and a script change is applied but not detected.
		ResourceUpToDate: isUpToDate(cr, a),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Action)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotAction)
	}

	cr.Status.SetConditions(xpv1.Creating())

	orgID, err := e.organizationID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	id, err := e.client.CreateAction(ctx, orgID, desired(cr))
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, id)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Action)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotAction)
	}

	orgID, err := e.organizationID(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	id := meta.GetExternalName(cr)

	if err := e.client.UpdateAction(ctx, orgID, id, desired(cr)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	// Activating and deactivating are separate calls from editing an action, so
	// the state is applied on its own.
	if fp := cr.Spec.ForProvider.State; fp != nil {
		if err := e.client.SetActionState(ctx, orgID, id, string(*fp)); err != nil {
			return managed.ExternalUpdate{}, err
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Action)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotAction)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	// Deleting acts on the organization the action was created in, not on
	// whichever one the reference points at now.
	orgID := common.Deref(cr.Status.AtProvider.OrganizationID)
	if orgID == "" {
		resolved, err := e.organizationID(ctx, cr)
		switch {
		case err == nil:
			orgID = resolved

		case common.IsReferenceGone(err):
			// The organization is gone, and with it the action.

		default:
			return managed.ExternalDelete{}, err
		}
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	return managed.ExternalDelete{}, e.client.DeleteAction(ctx, orgID, id)
}

const errNotAction = "managed resource is not an Action custom resource"

// organizationID resolves the organization the action belongs to.
func (e *external) organizationID(ctx context.Context, cr *v1alpha1.Action) (string, error) {
	fp := cr.Spec.ForProvider

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	orgID, err := common.ResolveOrganizationID(ctx, e.kube, cr, fp.OrganizationRef,
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

// desired translates the spec into the client input, applying the documented
// defaults for every field the spec leaves unset.
func desired(cr *v1alpha1.Action) zitadel.ActionInput {
	fp := cr.Spec.ForProvider

	return zitadel.ActionInput{
		Name:          common.Deref(fp.Name),
		Script:        common.Deref(fp.Script),
		Timeout:       valueOrTimeout(fp.Timeout, "10s"),
		AllowedToFail: common.DerefBool(fp.AllowedToFail),
	}
}

func isUpToDate(cr *v1alpha1.Action, a *zitadel.Action) bool {
	fp := cr.Spec.ForProvider

	if fp.Name != nil && common.Deref(fp.Name) != a.Name {
		return false
	}

	if fp.Timeout != nil && common.Deref(fp.Timeout) != a.Timeout {
		return false
	}

	if fp.AllowedToFail != nil && common.DerefBool(fp.AllowedToFail) != a.AllowedToFail {
		return false
	}

	// A state the spec does not mention is not compared, so an action someone
	// deactivated in the console is left as they left it.
	if fp.State != nil {
		return string(*fp.State) == a.State
	}

	return true
}

func updateStatus(cr *v1alpha1.Action, orgID string, a *zitadel.Action) {
	cr.Status.AtProvider.OrganizationID = common.StringPtr(orgID)
	cr.Status.AtProvider.ID = a.ID
	cr.Status.AtProvider.Name = common.StringPtr(a.Name)
	cr.Status.AtProvider.Timeout = common.StringPtr(a.Timeout)
	cr.Status.AtProvider.AllowedToFail = common.BoolPtr(a.AllowedToFail)
	cr.Status.AtProvider.State = common.StringPtr(a.State)
}

func valueOrTimeout(t *string, fallback string) string {
	if t == nil || *t == "" {
		return fallback
	}

	return *t
}
