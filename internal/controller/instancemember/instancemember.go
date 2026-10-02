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

// Package instancemember implements the controller for the InstanceMember managed resource - a user's instance level roles.
package instancemember

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

const errNotInstanceMember = "managed resource is not a InstanceMember custom resource"

// Setup adds a controller that reconciles InstanceMember managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.InstanceMemberGroupKind,
		v1alpha1.InstanceMemberGroupVersionKind,
		&v1alpha1.InstanceMember{},
		&v1alpha1.InstanceMemberList{},
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
func (e *external) Disconnect(ctx context.Context) error {
	return e.client.Close()
}

// Observe reports whether the user is an instance member and whether the roles
// match.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.InstanceMember)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotInstanceMember)
	}

	userID, err := e.userID(ctx, cr)
	if err != nil {
		if meta.WasDeleted(cr) {
			// Nothing resolves while an object is terminating, so report the
			// external resource as gone. That lets the reconciler run Delete,
			// which lets the finalizer go, instead of retrying an observation
			// that can never succeed.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	if meta.GetExternalName(cr) == "" {
		meta.SetExternalName(cr, userID)
	}

	m, err := e.client.GetIAMMember(ctx, userID)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the instance membership from Zitadel")
	}

	updateStatus(cr, m)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: common.EqualStringSlices(cr.Spec.ForProvider.RoleKeys, m.Roles),
	}, nil
}

// Create adds the user to the instance with the desired roles.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.InstanceMember)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotInstanceMember)
	}

	cr.Status.SetConditions(xpv1.Creating())

	userID, err := e.userID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.validateRoles(ctx, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.client.AddIAMMember(ctx, userID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, userID)

	return managed.ExternalCreation{}, nil
}

// Update replaces the roles of the membership.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.InstanceMember)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotInstanceMember)
	}

	userID, err := e.userID(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.validateRoles(ctx, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.UpdateIAMMember(ctx, userID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the user from the instance.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.InstanceMember)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotInstanceMember)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.UserID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.RemoveIAMMember(ctx, common.Deref(cr.Status.AtProvider.UserID)); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{}, nil
}

// userID resolves the user of the membership. There is no organization: an
// InstanceMember belongs to the instance itself.
func (e *external) userID(ctx context.Context, cr *v1alpha1.InstanceMember) (string, error) {
	fp := cr.Spec.ForProvider

	id, err := common.ResolveUserID(ctx, e.kube, cr, fp.UserRef, fp.UserSelector, fp.UserID, common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.UserID))
	if err != nil {
		return "", common.Join(common.ErrNoUserID, err)
	}

	if id == "" {
		return "", errors.New(common.ErrNoUserID.Error())
	}

	return id, nil
}

// validateRoles checks the roles against the ones this instance offers, so a
// typo is reported together with the valid values.
func (e *external) validateRoles(ctx context.Context, roles []string) error {
	available, err := e.client.ListIAMMemberRoles(ctx)
	if err != nil {
		return err
	}

	if unknown := zitadel.ValidateRoles(roles, available); len(unknown) > 0 {
		return zitadel.FormatUnknownRoles("instance roles", unknown, available)
	}

	return nil
}

// updateStatus copies the observed membership into the status of cr.
func updateStatus(cr *v1alpha1.InstanceMember, m *zitadel.Membership) {
	cr.Status.AtProvider.UserID = common.StringPtr(m.UserID)
	cr.Status.AtProvider.RoleKeys = m.Roles

	if m.DisplayName != "" {
		cr.Status.AtProvider.DisplayName = common.StringPtr(m.DisplayName)
	}
	if m.Email != "" {
		cr.Status.AtProvider.Email = common.StringPtr(m.Email)
	}
	if m.UserType != "" {
		cr.Status.AtProvider.UserType = common.StringPtr(m.UserType)
	}
}
