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

// Package project implements the controller for the Project managed resource.
package project

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
	errNotProject = "managed resource is not a Project custom resource"
	errNoOrgID    = "cannot determine the organization of the project"
)

// Setup adds a controller that reconciles Project managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ProjectGroupKind,
		v1alpha1.ProjectGroupVersionKind,
		&v1alpha1.Project{},
		&v1alpha1.ProjectList{},
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

// Observe fetches the project and reports whether it exists and matches the
// desired state.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProject)
	}

	orgID, err := e.organizationID(ctx, cr)
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

	p, err := e.observe(ctx, cr, orgID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if p == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	updateStatus(cr, p)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate(cr, p),
	}, nil
}

// observe returns the project managed by cr, recovering from a lost external
// name by looking the project up by name within the organization.
func (e *external) observe(ctx context.Context, cr *v1alpha1.Project, orgID string) (*zitadel.Project, error) {
	p, found, err := common.Recover(
		meta.GetExternalName(cr),
		common.Deref(cr.Spec.ForProvider.ID),
		func(id string) (*zitadel.Project, error) { return e.client.GetProject(ctx, id) },
		func() (*zitadel.Project, error) {
			if orgID == "" {
				return nil, nil
			}

			projects, err := e.client.ListProjectsByOrgID(ctx, orgID)
			if err != nil {
				return nil, err
			}

			for _, p := range projects {
				if p.Name == cr.Spec.ForProvider.Name {
					return p, nil
				}
			}

			return nil, nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "cannot look up project in Zitadel")
	}

	if !found {
		return nil, nil
	}

	meta.SetExternalName(cr, p.ProjectID)

	return p, nil
}

// Create creates the project in Zitadel.
//
//nolint:gocyclo // flat translation of the optional forProvider fields
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProject)
	}

	cr.Status.SetConditions(xpv1.Creating())

	orgID, err := e.organizationID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if orgID == "" {
		return managed.ExternalCreation{}, errors.New(errNoOrgID)
	}

	fp := cr.Spec.ForProvider
	in := zitadel.CreateProjectInput{
		OrganizationID: orgID,
		ProjectID:      common.Deref(fp.ID),
		Name:           fp.Name,
	}

	if fp.ProjectRoleAssertion != nil {
		in.ProjectRoleAssertion = *fp.ProjectRoleAssertion
	}
	if fp.AuthorizationRequired != nil {
		in.AuthorizationRequired = *fp.AuthorizationRequired
	}
	if fp.ProjectAccessRequired != nil {
		in.ProjectAccessRequired = *fp.ProjectAccessRequired
	}

	pls, err := zitadel.PrivateLabelingSettingToProto(common.Value(fp.PrivateLabelingSetting))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot set private labeling setting")
	}
	in.PrivateLabelingSetting = pls

	id, err := e.client.CreateProject(ctx, in)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create project in Zitadel")
	}

	meta.SetExternalName(cr, id)

	if fp.State != nil && *fp.State == v1alpha1.ProjectStateInactive {
		if err := e.client.SetProjectState(ctx, id, zitadel.StateActive, zitadel.StateInactive); err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	return managed.ExternalCreation{}, nil
}

// Update reconciles the mutable attributes and the state of the project.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProject)
	}

	orgID, err := e.organizationID(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	current, err := e.observe(ctx, cr, orgID)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if current == nil {
		return managed.ExternalUpdate{}, errors.New("cannot update a project that does not exist")
	}

	fp := cr.Spec.ForProvider
	in := zitadel.UpdateProjectInput{Name: common.StringPtr(fp.Name)}
	in.ProjectRoleAssertion = fp.ProjectRoleAssertion
	in.AuthorizationRequired = fp.AuthorizationRequired
	in.ProjectAccessRequired = fp.ProjectAccessRequired

	if fp.PrivateLabelingSetting != nil {
		pls, err := zitadel.PrivateLabelingSettingToProto(string(*fp.PrivateLabelingSetting))
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "cannot set private labeling setting")
		}
		in.PrivateLabelingSetting = &pls
	}

	if err := e.client.UpdateProject(ctx, current.ProjectID, in); err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.SetProjectState(ctx, current.ProjectID, current.State, common.Value(fp.State)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the project from Zitadel.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotProject)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.OrganizationID, cr.Status.AtProvider.ID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeleteProject(ctx, id); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete project from Zitadel")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect releases the underlying Zitadel client.
func (e *external) Disconnect(ctx context.Context) error {
	return e.client.Close()
}

// organizationID resolves the organization the project belongs to.
func (e *external) organizationID(ctx context.Context, cr *v1alpha1.Project) (string, error) {
	fp := cr.Spec.ForProvider

	// Once the project exists we know its organization, so there is no need to
	// re-resolve references.

	providerDefault, err := common.ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return "", common.Join(common.ErrResolveOrganization, err)
	}

	id, err := common.ResolveOrganizationID(ctx, e.kube, cr, fp.OrganizationRef, fp.OrganizationSelector, fp.OrganizationID, providerDefault,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.OrganizationID))
	if err != nil {
		return "", common.Join(common.ErrResolveOrganization, err)
	}

	return id, nil
}
