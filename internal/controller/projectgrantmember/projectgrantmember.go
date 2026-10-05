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

// Package projectgrantmember implements the controller for the ProjectGrantMember managed resource - a user's roles on a shared project.
package projectgrantmember

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

const errNotProjectGrantMember = "managed resource is not a ProjectGrantMember custom resource"

// Setup adds a controller that reconciles ProjectGrantMember managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ProjectGrantMemberGroupKind,
		v1alpha1.ProjectGrantMemberGroupVersionKind,
		&v1alpha1.ProjectGrantMember{},
		&v1alpha1.ProjectGrantMemberList{},
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

// Disconnect releases this reconcile's claim on the Zitadel client. The
// client itself is usually shared and outlives the reconcile, so this is a
// no-op unless the client is not owned by a cache.
func (e *external) Disconnect(ctx context.Context) error {
	e.client.Release()

	return nil
}

// resolve resolves the organization, project, grant and user of the membership.
//
//nolint:gocyclo // four references must resolve before the grant can be looked up
func (e *external) resolve(ctx context.Context, cr *v1alpha1.ProjectGrantMember) (orgID, projectID, grantID, userID string, err error) {
	//nolint:gocyclo // four references must resolve before the grant can be looked up
	fp := cr.Spec.ForProvider

	projectID, err = common.ResolveProjectID(ctx, e.kube, cr, fp.ProjectRef, fp.ProjectSelector, fp.ProjectID,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.ProjectID))
	if err != nil {
		return "", "", "", "", common.Join(common.ErrNoProjectID, err)
	}

	// The grant ID is Zitadel's, and the v2 project API does not return it, so
	// the grant is looked up by the organization it was granted to.
	grantedOrgID, err := common.ResolveOrganizationID(ctx, e.kube, cr, fp.GrantedOrganizationRef,
		fp.GrantedOrganizationSelector, fp.GrantedOrganizationID, "",
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.GrantedOrganizationID))
	if err != nil {
		return "", "", "", "", common.Join(common.ErrNoOrganizationID, err)
	}

	if grantedOrgID == "" {
		return "", "", "", "", errors.New("grantedOrganizationID, grantedOrganizationRef or grantedOrganizationSelector must be set")
	}

	// Members of a project grant live in the organization the project was
	// granted to, and the v1 management API is scoped to that organization.
	orgID = grantedOrgID

	// The grant itself is owned by the project's organization, which the v2 API
	// reports; the v1 management API only lists the grants of the organization
	// in context, so looking the grant up in the granted organization would find
	// nothing.
	ownerOrgID, err := e.projectOwnerOrgID(ctx, projectID)
	if err != nil {
		return "", "", "", "", err
	}

	grant, err := e.client.FindProjectGrant(ctx, ownerOrgID, projectID, grantedOrgID)
	if err != nil {
		return "", "", "", "", errors.Wrap(err, "cannot find the project grant the membership belongs to")
	}

	if grant.GrantID == "" {
		return "", "", "", "", errors.New("Zitadel did not report an ID for the project grant, so members cannot be managed on it")
	}

	userID, err = common.ResolveUserID(ctx, e.kube, cr, fp.UserRef, fp.UserSelector, fp.UserID,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.UserID))
	if err != nil {
		return "", "", "", "", common.Join(common.ErrNoUserID, err)
	}

	if userID == "" {
		return "", "", "", "", errors.New(common.ErrNoUserID.Error())
	}

	return orgID, projectID, grant.GrantID, userID, nil
}

// Observe reports whether the user is a member of the grant and whether the
// roles match.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ProjectGrantMember)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProjectGrantMember)
	}

	orgID, projectID, grantID, userID, err := e.resolve(ctx, cr)
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
		meta.SetExternalName(cr, grantID+"/"+userID)
	}

	m, err := e.client.GetProjectGrantMember(ctx, orgID, projectID, grantID, userID)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the project grant membership from Zitadel")
	}

	updateStatus(cr, orgID, projectID, grantID, m)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: common.EqualStringSlices(cr.Spec.ForProvider.RoleKeys, m.Roles),
	}, nil
}

// Create adds the user to the grant with the desired roles.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ProjectGrantMember)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProjectGrantMember)
	}

	cr.Status.SetConditions(xpv1.Creating())

	orgID, projectID, grantID, userID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.validateRoles(ctx, projectID, grantID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.client.AddProjectGrantMember(ctx, orgID, projectID, grantID, userID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, grantID+"/"+userID)

	return managed.ExternalCreation{}, nil
}

// Update replaces the roles of the membership.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ProjectGrantMember)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProjectGrantMember)
	}

	orgID, projectID, grantID, userID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.validateRoles(ctx, projectID, grantID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.UpdateProjectGrantMember(ctx, orgID, projectID, grantID, userID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the user from the grant.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ProjectGrantMember)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotProjectGrantMember)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.ProjectID, cr.Status.AtProvider.UserID, cr.Status.AtProvider.GrantID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	// The membership is managed inside the organization the project was granted
	// to, and every identifier needed to address it is already observed.
	if err := e.client.RemoveProjectGrantMember(ctx,
		common.Deref(cr.Status.AtProvider.GrantedOrganizationID),
		common.Deref(cr.Status.AtProvider.ProjectID),
		common.Deref(cr.Status.AtProvider.GrantID),
		common.Deref(cr.Status.AtProvider.UserID)); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{}, nil
}

// projectOwnerOrgID returns the organization that owns a project. Project grants
// belong to the owning organization, not to the one they are granted to.
func (e *external) projectOwnerOrgID(ctx context.Context, projectID string) (string, error) {
	p, err := e.client.GetProject(ctx, projectID)
	if err != nil {
		return "", errors.Wrap(err, "cannot get the project of the grant")
	}

	return p.OrganizationID, nil
}

// validateRoles checks the roles against the ones the project grant makes
// available to its members.
func (e *external) validateRoles(ctx context.Context, projectID, grantID string, roles []string) error {
	ownerOrgID, err := e.projectOwnerOrgID(ctx, projectID)
	if err != nil {
		return nil //nolint:nilerr // Zitadel is the authority on which roles exist
	}

	available, err := e.client.ListProjectGrantMemberRoles(ctx, ownerOrgID, projectID, grantID)
	if err != nil {
		return nil //nolint:nilerr // Zitadel is the authority on which roles exist
	}

	if unknown := zitadel.ValidateRoles(roles, available); len(unknown) > 0 {
		return zitadel.FormatUnknownRoles(
			"project grant roles. A member of a granted project is not given a role in the project itself, so these are the grant's own PROJECT_GRANT_ roles",
			unknown, available,
		)
	}

	return nil
}

// updateStatus copies the observed membership into the status of cr.
func updateStatus(cr *v1alpha1.ProjectGrantMember, orgID, projectID, grantID string, m *zitadel.Membership) {
	cr.Status.AtProvider.UserID = common.StringPtr(m.UserID)
	cr.Status.AtProvider.OrganizationID = common.StringPtr(orgID)
	cr.Status.AtProvider.ProjectID = common.StringPtr(projectID)
	cr.Status.AtProvider.GrantID = common.StringPtr(grantID)
	cr.Status.AtProvider.RoleKeys = m.Roles

	if m.DisplayName != "" {
		cr.Status.AtProvider.DisplayName = common.StringPtr(m.DisplayName)
	}
	if m.Email != "" {
		cr.Status.AtProvider.Email = common.StringPtr(m.Email)
	}
}
