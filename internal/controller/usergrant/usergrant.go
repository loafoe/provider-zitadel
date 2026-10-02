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

// Package usergrant implements the controller for the UserGrant managed resource - project roles granted to a user.
package usergrant

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

const errNotUserGrant = "managed resource is not a UserGrant custom resource"

// Setup adds a controller that reconciles UserGrant managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.UserGrantGroupKind,
		v1alpha1.UserGrantGroupVersionKind,
		&v1alpha1.UserGrant{},
		&v1alpha1.UserGrantList{},
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

// resolve resolves the organization, project and user of the grant.
func (e *external) resolve(ctx context.Context, cr *v1alpha1.UserGrant) (orgID, projectID, userID string, err error) {
	fp := cr.Spec.ForProvider

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return "", "", "", common.Join(common.ErrNoOrganizationID, err)
	}

	orgID, err = common.ResolveOrganizationID(ctx, e.kube, cr, fp.OrganizationRef, fp.OrganizationSelector, fp.OrganizationID, orgDefault, common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.OrganizationID))
	if err != nil {
		return "", "", "", common.Join(common.ErrNoOrganizationID, err)
	}

	if orgID == "" {
		return "", "", "", errors.New(common.ErrNoOrganizationID.Error())
	}

	projectID, err = common.ResolveProjectID(ctx, e.kube, cr, fp.ProjectRef, fp.ProjectSelector, fp.ProjectID,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.ProjectID))
	if err != nil {
		return "", "", "", common.Join(common.ErrNoProjectID, err)
	}

	userID, err = common.ResolveUserID(ctx, e.kube, cr, fp.UserRef, fp.UserSelector, fp.UserID,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.UserID))
	if err != nil {
		return "", "", "", common.Join(common.ErrNoUserID, err)
	}

	if userID == "" {
		return "", "", "", errors.New(common.ErrNoUserID.Error())
	}

	return orgID, projectID, userID, nil
}

// projectGrantID is the project grant the roles are granted on, or empty when
// the roles are granted on a project of the user's own organization.
func projectGrantID(cr *v1alpha1.UserGrant) string {
	return common.Deref(cr.Spec.ForProvider.ProjectGrantID)
}

// Observe reports whether the grant exists and matches.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.UserGrant)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserGrant)
	}

	orgID, projectID, userID, err := e.resolve(ctx, cr)
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

	grantID := projectGrantID(cr)

	g, err := e.client.FindUserGrant(ctx, orgID, userID, projectID, grantID)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the user grant from Zitadel")
	}

	// Crossplane seeds the external name with the object's own name, so an
	// external name that is simply the object's name is not a Zitadel ID yet.
	if meta.GetExternalName(cr) == "" || meta.GetExternalName(cr) == cr.GetName() {
		meta.SetExternalName(cr, g.ID)
	}

	updateStatus(cr, orgID, g)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  common.EqualStringSlices(cr.Spec.ForProvider.RoleKeys, g.RoleKeys),
		ConnectionDetails: managed.ConnectionDetails{v1alpha1.ConnectionKeyUserGrantID: []byte(g.ID)},
	}, nil
}

// Create grants the roles to the user.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.UserGrant)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUserGrant)
	}

	cr.Status.SetConditions(xpv1.Creating())

	orgID, projectID, userID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.validateRoles(ctx, orgID, projectID, projectGrantID(cr), cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.client.AddUserGrant(ctx, orgID, userID, projectID, projectGrantID(cr), cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalCreation{}, err
	}

	// Zitadel's grant list is served from a projection, so a grant that was
	// just created may not be readable yet. Not being able to read it back is
	// not a failed create: the grant exists and Observe will resolve its ID on
	// the next reconcile. Any other error is real and is reported.
	g, err := e.client.FindUserGrant(ctx, orgID, userID, projectID, projectGrantID(cr))
	if err != nil {
		if !zitadel.IsNotFound(err) {
			return managed.ExternalCreation{}, errors.Wrap(err, "the grant was created but Zitadel could not be read back")
		}

		return managed.ExternalCreation{}, nil
	}

	meta.SetExternalName(cr, g.ID)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
		v1alpha1.ConnectionKeyUserGrantID: []byte(g.ID),
	}}, nil
}

// Update replaces the roles of the grant.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.UserGrant)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUserGrant)
	}

	orgID, projectID, userID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.validateRoles(ctx, orgID, projectID, projectGrantID(cr), cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.UpdateUserGrant(ctx, orgID, userID, meta.GetExternalName(cr), cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete revokes the grant.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.UserGrant)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUserGrant)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.OrganizationID,
		cr.Status.AtProvider.ProjectID, cr.Status.AtProvider.UserID, cr.Status.AtProvider.GrantID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	// The grant's ID comes from the observation rather than the external name:
	// Crossplane fills the external name in with the object's own name before
	// the grant has been read back from Zitadel, so it is not always an ID.
	if err := e.client.RemoveUserGrant(ctx,
		common.Deref(cr.Status.AtProvider.OrganizationID),
		common.Deref(cr.Status.AtProvider.UserID),
		common.Deref(cr.Status.AtProvider.GrantID)); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{}, nil
}

// validateRoles checks the roles against the ones the grant may use: the
// project's own roles when granting inside the user's own organization, or the
// roles the project grant exposes on a shared project.
func (e *external) validateRoles(ctx context.Context, orgID, projectID, grantID string, roles []string) error {
	available, err := e.grantableRoles(ctx, orgID, projectID, grantID)
	if err != nil {
		// Not being able to read the role list must not block the grant: Zitadel
		// is the authority and will reject a role it does not know.
		return nil //nolint:nilerr // see the comment above
	}

	if unknown := zitadel.ValidateRoles(roles, available); len(unknown) > 0 {
		return zitadel.FormatUnknownRoles("project roles", unknown, available)
	}

	return nil
}

// grantableRoles returns the roles a grant on this project or project grant may
// use.
func (e *external) grantableRoles(ctx context.Context, orgID, projectID, grantID string) ([]string, error) {
	if grantID == "" {
		return e.client.ListProjectRoles(ctx, projectID)
	}

	grant, err := e.client.FindProjectGrantByID(ctx, orgID, projectID, grantID)
	if err != nil {
		return nil, err
	}

	return grant.RoleKeys, nil
}

// updateStatus copies the observed grant into the status of cr.
func updateStatus(cr *v1alpha1.UserGrant, orgID string, g *zitadel.UserGrant) {
	cr.Status.AtProvider.GrantID = common.StringPtr(g.ID)
	cr.Status.AtProvider.UserID = common.StringPtr(g.UserID)
	cr.Status.AtProvider.OrganizationID = common.StringPtr(orgID)
	cr.Status.AtProvider.ProjectID = common.StringPtr(g.ProjectID)
	cr.Status.AtProvider.RoleKeys = g.RoleKeys

	if g.ProjectGrantID != "" {
		cr.Status.AtProvider.ProjectGrantID = common.StringPtr(g.ProjectGrantID)
	}
	if g.State != "" {
		state := v1alpha1.MembershipState(g.State)
		cr.Status.AtProvider.State = &state
	}
}
