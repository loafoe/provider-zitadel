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

// Package oidcapplication implements the controller for the OIDCApplication
// managed resource - an OAuth2/OIDC client registered with Zitadel.
package oidcapplication

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
	errNotApplication = "managed resource is not an OIDCApplication custom resource"
	errNoProjectID    = "cannot determine the project of the application"
	errResolveProject = "cannot resolve the project of the application"
)

// Setup adds a controller that reconciles OIDCApplication managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.OIDCApplicationGroupKind,
		v1alpha1.OIDCApplicationGroupVersionKind,
		&v1alpha1.OIDCApplication{},
		&v1alpha1.OIDCApplicationList{},
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

// Observe fetches the application and reports whether it exists and matches
// the desired state.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.OIDCApplication)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotApplication)
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

	app, err := e.observe(ctx, cr, projectID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if app == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	updateStatus(cr, app)
	cr.Status.SetConditions(xpv1.Available())

	upToDate, regenerate := isUpToDate(cr, app)

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  upToDate && !regenerate,
		ConnectionDetails: connectionDetails(app),
	}, nil
}

// observe returns the application managed by cr, recovering from a lost
// external name by looking the application up by name within the project.
func (e *external) observe(ctx context.Context, cr *v1alpha1.OIDCApplication, projectID string) (*zitadel.OIDCApplication, error) {
	app, found, err := common.Recover(
		meta.GetExternalName(cr),
		common.Deref(cr.Spec.ForProvider.ID),
		func(id string) (*zitadel.OIDCApplication, error) { return e.client.GetOIDCApplication(ctx, id) },
		func() (*zitadel.OIDCApplication, error) {
			if projectID == "" {
				return nil, nil
			}

			apps, err := e.client.ListOIDCApplicationsByProjectID(ctx, projectID)
			if err != nil {
				return nil, err
			}

			for _, a := range apps {
				if a.Name == cr.Spec.ForProvider.Name {
					return a, nil
				}
			}

			return nil, nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "cannot look up application in Zitadel")
	}

	if !found {
		return nil, nil
	}

	meta.SetExternalName(cr, app.ApplicationID)

	return app, nil
}

// Create creates the OIDC application in Zitadel and publishes the client
// credentials to the connection secret.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.OIDCApplication)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotApplication)
	}

	cr.Status.SetConditions(xpv1.Creating())

	projectID, err := e.projectID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if projectID == "" {
		return managed.ExternalCreation{}, errors.New(errNoProjectID)
	}

	in, err := createInput(cr, projectID)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	res, err := e.client.CreateOIDCApplication(ctx, in)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create application in Zitadel")
	}

	meta.SetExternalName(cr, res.ApplicationID)

	if cr.Spec.ForProvider.State != nil && *cr.Spec.ForProvider.State == v1alpha1.ApplicationStateInactive {
		if err := e.client.SetApplicationState(ctx, res.ApplicationID, zitadel.StateActive, zitadel.StateInactive); err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
		v1alpha1.ConnectionKeyClientID:     []byte(res.ClientID),
		v1alpha1.ConnectionKeyClientSecret: []byte(res.ClientSecret),
	}}, nil
}

// Update reconciles the mutable attributes, the state and - when requested -
// the client secret of the application.
//
//nolint:gocyclo // a handful of independent reconciliations
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.OIDCApplication)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotApplication)
	}

	projectID, err := e.projectID(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	current, err := e.observe(ctx, cr, projectID)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if current == nil {
		return managed.ExternalUpdate{}, errors.New("cannot update an application that does not exist")
	}

	fp := cr.Spec.ForProvider

	in, err := updateInput(cr, current.ProjectID)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	// The v1 management API needs the organization of the project, which is
	// the only place the v2 API stops being able to apply the change.
	project, err := e.client.GetProject(ctx, current.ProjectID)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot get the project of the application from Zitadel")
	}

	if err := e.client.UpdateOIDCApplicationInOrg(ctx, project.OrganizationID, current.ApplicationID, in); err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.SetApplicationState(ctx, current.ApplicationID, current.State, common.Value(fp.State)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	details := managed.ConnectionDetails{}
	if fp.RegenerateClientSecret != nil && *fp.RegenerateClientSecret {
		secret, err := e.client.GenerateClientSecret(ctx, current.ApplicationID)
		if err != nil {
			return managed.ExternalUpdate{}, err
		}
		details[v1alpha1.ConnectionKeyClientSecret] = []byte(secret)
	}

	return managed.ExternalUpdate{ConnectionDetails: details}, nil
}

// Delete removes the application from Zitadel.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.OIDCApplication)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotApplication)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.ProjectID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeleteOIDCApplication(ctx, id, common.Deref(cr.Status.AtProvider.ProjectID)); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete application from Zitadel")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect releases the underlying Zitadel client.
func (e *external) Disconnect(ctx context.Context) error {
	return e.client.Close()
}

// projectID resolves the project the application belongs to.
func (e *external) projectID(ctx context.Context, cr *v1alpha1.OIDCApplication) (string, error) {
	fp := cr.Spec.ForProvider

	id, err := common.ResolveProjectID(ctx, e.kube, cr, fp.ProjectRef, fp.ProjectSelector, fp.ProjectID,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.ProjectID))
	if err != nil {
		return "", common.Join(common.ErrResolveProject, err)
	}

	return id, nil
}
