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

// Package orgmember implements the controller for the OrgMember managed
// resource - a user's organization level roles.
package orgmember

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

const errNotOrgMember = "managed resource is not an OrgMember custom resource"

// Setup adds a controller that reconciles OrgMember managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.OrgMemberGroupKind,
		v1alpha1.OrgMemberGroupVersionKind,
		&v1alpha1.OrgMember{},
		&v1alpha1.OrgMemberList{},
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

// Observe reports whether the user is a member of the organization and whether
// the roles match.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.OrgMember)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrgMember)
	}

	orgID, userID, err := e.resolve(ctx, cr)
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

	// Zitadel gives memberships no ID, so they are addressed by the pair they
	// exist for and the external name is that pair.
	if meta.GetExternalName(cr) == "" {
		meta.SetExternalName(cr, orgID+"/"+userID)
	}

	m, err := e.client.GetOrgMember(ctx, orgID, userID)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the organization membership from Zitadel")
	}

	updateStatus(cr, orgID, m)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate(cr, m),
	}, nil
}

// Create adds the user to the organization with the desired roles.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.OrgMember)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrgMember)
	}

	cr.Status.SetConditions(xpv1.Creating())

	orgID, userID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.validateRoles(ctx, orgID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.client.AddOrgMember(ctx, orgID, userID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, orgID+"/"+userID)

	return managed.ExternalCreation{}, nil
}

// Update replaces the roles of the membership.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.OrgMember)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrgMember)
	}

	orgID, userID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.validateRoles(ctx, orgID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.UpdateOrgMember(ctx, orgID, userID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the user from the organization.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.OrgMember)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrgMember)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.OrganizationID, cr.Status.AtProvider.UserID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.RemoveOrgMember(ctx,
		common.Deref(cr.Status.AtProvider.OrganizationID),
		common.Deref(cr.Status.AtProvider.UserID)); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect releases this reconcile's claim on the Zitadel client. The
// client itself is usually shared and outlives the reconcile, so this is a
// no-op unless the client is not owned by a cache.
func (e *external) Disconnect(ctx context.Context) error {
	e.client.Release()

	return nil
}

// resolve resolves the organization and the user of the membership.
func (e *external) resolve(ctx context.Context, cr *v1alpha1.OrgMember) (string, string, error) {
	fp := cr.Spec.ForProvider

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return "", "", common.Join(common.ErrNoOrganizationID, err)
	}

	orgID, err := common.ResolveOrganizationID(ctx, e.kube, cr, fp.OrganizationRef, fp.OrganizationSelector, fp.OrganizationID, orgDefault, common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.OrganizationID))
	if err != nil {
		return "", "", common.Join(common.ErrNoOrganizationID, err)
	}

	if orgID == "" {
		return "", "", errors.New(common.ErrNoOrganizationID.Error())
	}

	userID, err := common.ResolveUserID(ctx, e.kube, cr, fp.UserRef, fp.UserSelector, fp.UserID,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.UserID))
	if err != nil {
		return "", "", common.Join(common.ErrNoUserID, err)
	}

	if userID == "" {
		return "", "", errors.New(common.ErrNoUserID.Error())
	}

	return orgID, userID, nil
}

// validateRoles checks the roles against the ones this instance offers, so a
// typo is reported together with the valid values instead of failing deep
// inside Zitadel with an opaque error.
func (e *external) validateRoles(ctx context.Context, orgID string, roles []string) error {
	available, err := e.client.ListOrgMemberRoles(ctx, orgID)
	if err != nil {
		return err
	}

	if unknown := zitadel.ValidateRoles(roles, available); len(unknown) > 0 {
		return zitadel.FormatUnknownRoles("organization roles", unknown, available)
	}

	return nil
}

// isUpToDate reports whether the membership holds exactly the desired roles.
func isUpToDate(cr *v1alpha1.OrgMember, m *zitadel.Membership) bool {
	return common.EqualStringSlices(cr.Spec.ForProvider.RoleKeys, m.Roles)
}

// updateStatus copies the observed membership into the status of cr.
func updateStatus(cr *v1alpha1.OrgMember, orgID string, m *zitadel.Membership) {
	cr.Status.AtProvider.UserID = common.StringPtr(m.UserID)
	cr.Status.AtProvider.OrganizationID = common.StringPtr(orgID)
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
