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

// Package organization implements the controller for the Organization managed
// resource.
package organization

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

const errNotOrganization = "managed resource is not an Organization custom resource"

// Setup adds a controller that reconciles Organization managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.OrganizationGroupKind,
		v1alpha1.OrganizationGroupVersionKind,
		&v1alpha1.Organization{},
		&v1alpha1.OrganizationList{},
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

// Observe fetches the organization and reports whether it exists and matches
// the desired state.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Organization)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganization)
	}

	org, err := e.observe(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if org == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	updateStatus(cr, org)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate(cr, org),
	}, nil
}

// observe returns the organization managed by cr, recovering from a lost
// external name by looking the organization up by name.
func (e *external) observe(ctx context.Context, cr *v1alpha1.Organization) (*zitadel.Organization, error) {
	org, found, err := common.Recover(
		meta.GetExternalName(cr),
		common.Deref(cr.Spec.ForProvider.ID),
		func(id string) (*zitadel.Organization, error) { return e.client.GetOrganization(ctx, id) },
		func() (*zitadel.Organization, error) {
			return e.client.FindOrganizationByName(ctx, cr.Spec.ForProvider.Name)
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "cannot look up organization in Zitadel")
	}

	if !found {
		return nil, nil
	}

	meta.SetExternalName(cr, org.ID)

	return org, nil
}

// Create creates the organization in Zitadel.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Organization)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganization)
	}

	cr.Status.SetConditions(xpv1.Creating())

	id, err := e.client.CreateOrganization(ctx, cr.Spec.ForProvider.Name, common.Deref(cr.Spec.ForProvider.ID))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create organization in Zitadel")
	}

	meta.SetExternalName(cr, id)

	return managed.ExternalCreation{}, nil
}

// Update reconciles the mutable attributes and the state of the organization.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Organization)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganization)
	}

	current, err := e.observe(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if current == nil {
		return managed.ExternalUpdate{}, errors.New("cannot update an organization that does not exist")
	}

	if err := e.client.UpdateOrganization(ctx, current.ID, cr.Spec.ForProvider.Name); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update organization in Zitadel")
	}

	if err := e.client.SetOrganizationState(ctx, current.ID, current.State, common.Value(cr.Spec.ForProvider.State)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the organization from Zitadel.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Organization)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganization)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeleteOrganization(ctx, id); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete organization from Zitadel")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect releases the underlying Zitadel client.
func (e *external) Disconnect(ctx context.Context) error {
	return e.client.Close()
}
