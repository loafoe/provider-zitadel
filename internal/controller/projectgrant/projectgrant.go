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

// Package projectgrant implements the controller for the ProjectGrant managed resource - a project shared with another organization.
package projectgrant

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

const errNotProjectGrant = "managed resource is not a ProjectGrant custom resource"

// Setup adds a controller that reconciles ProjectGrant managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ProjectGrantGroupKind,
		v1alpha1.ProjectGrantGroupVersionKind,
		&v1alpha1.ProjectGrant{},
		&v1alpha1.ProjectGrantList{},
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

// resolve resolves the owning organization, the project and the organization
// the project is granted to.
func (e *external) resolve(ctx context.Context, cr *v1alpha1.ProjectGrant) (ownerOrgID, projectID, grantedOrgID string, err error) {
	fp := cr.Spec.ForProvider

	ownerDefault, err := common.ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return "", "", "", common.Join(common.ErrNoOrganizationID, err)
	}

	ownerOrgID, err = common.ResolveOrganizationID(ctx, e.kube, cr, fp.OrganizationRef,
		fp.OrganizationSelector, fp.OrganizationID, ownerDefault,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.OrganizationID))
	if err != nil {
		return "", "", "", common.Join(common.ErrNoOrganizationID, err)
	}

	projectID, err = common.ResolveProjectID(ctx, e.kube, cr, fp.ProjectRef, fp.ProjectSelector, fp.ProjectID, common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.ProjectID))
	if err != nil {
		return "", "", "", common.Join(common.ErrNoProjectID, err)
	}

	grantedOrgID, err = common.ResolveOrganizationID(ctx, e.kube, cr, fp.GrantedOrganizationRef,
		fp.GrantedOrganizationSelector, fp.GrantedOrganizationID, "",
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.GrantedOrganizationID))
	if err != nil {
		return "", "", "", common.Join(common.ErrNoOrganizationID, err)
	}

	if grantedOrgID == "" {
		return "", "", "", errors.New("grantedOrganizationID, grantedOrganizationRef or grantedOrganizationSelector must be set")
	}

	if projectID == "" {
		return "", "", "", errors.New(common.ErrNoProjectID.Error())
	}

	return ownerOrgID, projectID, grantedOrgID, nil
}

// Observe reports whether the project is granted to the organization and whether
// the roles match.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ProjectGrant)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProjectGrant)
	}

	ownerOrgID, projectID, grantedOrgID, err := e.resolve(ctx, cr)
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

	// Zitadel gives a project grant no ID of its own, so the pair it exists for
	// is the external name.
	if meta.GetExternalName(cr) == "" {
		meta.SetExternalName(cr, zitadel.ProjectGrantExternalName(projectID, grantedOrgID))
	}

	g, err := e.client.FindProjectGrant(ctx, ownerOrgID, projectID, grantedOrgID)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the project grant from Zitadel")
	}

	updateStatus(cr, ownerOrgID, g)
	cr.Status.SetConditions(xpv1.Available())

	upToDate := common.EqualStringSlices(cr.Spec.ForProvider.RoleKeys, g.RoleKeys)
	if fp := cr.Spec.ForProvider.State; fp != nil {
		upToDate = upToDate && string(*fp) == g.State
	}

	details := managed.ConnectionDetails{}
	if g.GrantID != "" {
		details[v1alpha1.ConnectionKeyProjectGrantID] = []byte(g.GrantID)
	}

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  upToDate,
		ConnectionDetails: details,
	}, nil
}

// Create shares the project with the organization.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ProjectGrant)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProjectGrant)
	}

	cr.Status.SetConditions(xpv1.Creating())

	_, projectID, grantedOrgID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.client.CreateProjectGrant(ctx, projectID, grantedOrgID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, zitadel.ProjectGrantExternalName(projectID, grantedOrgID))

	if state := cr.Spec.ForProvider.State; state != nil && *state == v1alpha1.GrantableStateInactive {
		if err := e.client.SetProjectGrantState(ctx, projectID, grantedOrgID, zitadel.StateActive, zitadel.StateInactive); err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	return managed.ExternalCreation{}, nil
}

// Update replaces the roles and the state of the grant.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ProjectGrant)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProjectGrant)
	}

	ownerOrgID, projectID, grantedOrgID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.UpdateProjectGrant(ctx, projectID, grantedOrgID, cr.Spec.ForProvider.RoleKeys); err != nil {
		return managed.ExternalUpdate{}, err
	}

	current, err := e.client.FindProjectGrant(ctx, ownerOrgID, projectID, grantedOrgID)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.SetProjectGrantState(ctx, projectID, grantedOrgID, current.State, common.Value(cr.Spec.ForProvider.State)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete stops sharing the project with the organization.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ProjectGrant)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotProjectGrant)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.ProjectID, cr.Status.AtProvider.GrantedOrganizationID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeleteProjectGrant(ctx,
		common.Deref(cr.Status.AtProvider.ProjectID),
		common.Deref(cr.Status.AtProvider.GrantedOrganizationID)); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{}, nil
}

// updateStatus copies the observed grant into the status of cr.
func updateStatus(cr *v1alpha1.ProjectGrant, ownerOrgID string, g *zitadel.ProjectGrant) {
	if ownerOrgID != "" {
		cr.Status.AtProvider.OrganizationID = common.StringPtr(ownerOrgID)
	}
	cr.Status.AtProvider.ProjectID = common.StringPtr(g.ProjectID)
	cr.Status.AtProvider.GrantedOrganizationID = common.StringPtr(g.GrantedOrganizationID)
	cr.Status.AtProvider.RoleKeys = g.RoleKeys

	if g.GrantedOrganizationName != "" {
		cr.Status.AtProvider.GrantedOrganizationName = common.StringPtr(g.GrantedOrganizationName)
	}
	if g.State != "" {
		state := v1alpha1.GrantableState(g.State)
		cr.Status.AtProvider.State = &state
	}
}
