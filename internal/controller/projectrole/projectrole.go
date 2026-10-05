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

// Package projectrole implements the controller for the ProjectRole managed
// resource.
package projectrole

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

const (
	errNotProjectRole = "managed resource is not a ProjectRole custom resource"
	errNoProjectID    = "cannot determine the project of the role"
	errResolveProject = "cannot resolve the project of the role"
)

// Setup adds a controller that reconciles ProjectRole managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ProjectRoleGroupKind,
		v1alpha1.ProjectRoleGroupVersionKind,
		&v1alpha1.ProjectRole{},
		&v1alpha1.ProjectRoleList{},
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

// Observe fetches the project role and reports whether it exists and matches
// the desired state.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ProjectRole)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProjectRole)
	}

	projectID, err := e.projectID(ctx, cr)
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

	if projectID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// A project role has no ID of its own: it is addressed by the
	// project<key> pair, so that pair is the external name.
	externalName := meta.GetExternalName(cr)
	if externalName == "" {
		meta.SetExternalName(cr, externalNameFor(projectID, cr.Spec.ForProvider.Key))
	}

	r, err := e.client.GetProjectRole(ctx, projectID, cr.Spec.ForProvider.Key)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get project role from Zitadel")
	}

	_ = externalName

	updateStatus(cr, r)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate(cr, r),
	}, nil
}

// Create adds the role to the project.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ProjectRole)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProjectRole)
	}

	cr.Status.SetConditions(xpv1.Creating())

	projectID, err := e.projectID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if projectID == "" {
		return managed.ExternalCreation{}, errors.New(errNoProjectID)
	}

	fp := cr.Spec.ForProvider
	if _, err := e.client.AddProjectRole(ctx, projectID, fp.Key, fp.DisplayName, common.Deref(fp.Group)); err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, externalNameFor(projectID, fp.Key))

	return managed.ExternalCreation{}, nil
}

// Update reconciles the display name and the group of the role.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ProjectRole)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProjectRole)
	}

	projectID, err := e.projectID(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if projectID == "" {
		return managed.ExternalUpdate{}, errors.New(errNoProjectID)
	}

	fp := cr.Spec.ForProvider
	if err := e.client.UpdateProjectRole(ctx, projectID, fp.Key, fp.DisplayName, common.Deref(fp.Group)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the role from the project.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ProjectRole)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotProjectRole)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.ProjectID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	if meta.GetExternalName(cr) == "" {
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.RemoveProjectRole(ctx,
		common.Deref(cr.Status.AtProvider.ProjectID), cr.Spec.ForProvider.Key); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot remove project role from Zitadel")
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

// projectID resolves the project the role belongs to.
func (e *external) projectID(ctx context.Context, cr *v1alpha1.ProjectRole) (string, error) {
	fp := cr.Spec.ForProvider

	id, err := common.ResolveProjectID(ctx, e.kube, cr, fp.ProjectRef, fp.ProjectSelector, fp.ProjectID,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.ProjectID))
	if err != nil {
		return "", common.Join(common.ErrResolveProject, err)
	}

	return id, nil
}

// externalNameFor builds the external name of a project role, which is the
// "<projectID>/<roleKey>" pair identifying it.
func externalNameFor(projectID, key string) string {
	return projectID + "/" + key
}

// updateStatus copies the observed state of r into the status of cr.
func updateStatus(cr *v1alpha1.ProjectRole, r *zitadel.ProjectRole) {
	cr.Status.AtProvider.Key = common.StringPtr(r.Key)
	cr.Status.AtProvider.DisplayName = common.StringPtr(r.DisplayName)
	cr.Status.AtProvider.Group = common.StringPtr(r.Group)
	cr.Status.AtProvider.CreationDate = common.ParseTime(r.CreationDate)
	cr.Status.AtProvider.ChangeDate = common.ParseTime(r.ChangeDate)
}

// isUpToDate reports whether the remote role matches the desired state
// described by cr.
func isUpToDate(cr *v1alpha1.ProjectRole, r *zitadel.ProjectRole) bool {
	fp := cr.Spec.ForProvider

	if fp.Key != r.Key {
		return false
	}

	if fp.DisplayName != r.DisplayName {
		return false
	}

	if fp.Group != nil && *fp.Group != r.Group {
		return false
	}

	return true
}
