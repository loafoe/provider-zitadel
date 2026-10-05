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

// Package organizationmetadata implements the controller for the OrganizationMetadata managed resource - the metadata of an organization.
package organizationmetadata

import (
	"context"
	"encoding/json"

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

const errNotOrganizationMetadata = "managed resource is not a OrganizationMetadata custom resource"

// Setup adds a controller that reconciles OrganizationMetadata managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.OrganizationMetadataGroupKind,
		v1alpha1.OrganizationMetadataGroupVersionKind,
		&v1alpha1.OrganizationMetadata{},
		&v1alpha1.OrganizationMetadataList{},
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

// resolve resolves the organization whose metadata is managed.
func (e *external) resolve(ctx context.Context, cr *v1alpha1.OrganizationMetadata) (string, error) {
	fp := cr.Spec.ForProvider

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	orgID, err := common.ResolveOrganizationID(ctx, e.kube, cr, fp.OrganizationRef, fp.OrganizationSelector, fp.OrganizationID, orgDefault,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.OrganizationID))
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	if orgID == "" {
		return "", errors.New(common.ErrNoOrganizationID.Error())
	}

	return orgID, nil
}

// Observe reports whether the metadata of the organization matches.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.OrganizationMetadata)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotOrganizationMetadata)
	}

	orgID, err := e.resolve(ctx, cr)
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
		meta.SetExternalName(cr, orgID)
	}

	current, err := e.client.GetOrganizationMetadata(ctx, orgID)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the organization metadata from Zitadel")
	}

	updateStatus(cr, orgID, current)
	cr.Status.SetConditions(xpv1.Available())

	// Zitadel has no notion of a metadata set existing: a subject either carries
	// keys or it carries none. Deciding existence by whether any key is present is
	// what lets a delete finish, since the reconciler only releases the finalizer
	// once the external resource is gone.
	//
	// An empty set in the spec is already satisfied and there is nothing to
	// create, so it counts as existing. While the object is terminating only the
	// observed keys matter.
	exists := len(current) > 0 || len(desired(cr)) == 0
	if meta.WasDeleted(cr) {
		exists = len(current) > 0
	}

	return managed.ExternalObservation{
		ResourceExists:    exists,
		ResourceUpToDate:  zitadel.EqualMetadata(desired(cr), current),
		ConnectionDetails: metadataDetails(current),
	}, nil
}

// Create writes the desired metadata set. Zitadel replaces rather than merges
// metadata, so this is the same operation as an update.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.OrganizationMetadata)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotOrganizationMetadata)
	}

	cr.Status.SetConditions(xpv1.Creating())

	orgID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.client.ApplyOrganizationMetadata(ctx, orgID, desired(cr)); err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, orgID)

	return managed.ExternalCreation{ConnectionDetails: metadataDetails(desired(cr))}, nil
}

// Update replaces the metadata set.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.OrganizationMetadata)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotOrganizationMetadata)
	}

	orgID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.ApplyOrganizationMetadata(ctx, orgID, desired(cr)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{ConnectionDetails: metadataDetails(desired(cr))}, nil
}

// Delete clears the metadata of the organization.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.OrganizationMetadata)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotOrganizationMetadata)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.OrganizationID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeleteOrganizationMetadata(ctx, common.Deref(cr.Status.AtProvider.OrganizationID)); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{}, nil
}

// desired returns the metadata entries the resource asks for.
func desired(cr *v1alpha1.OrganizationMetadata) []zitadel.MetadataEntry {
	out := make([]zitadel.MetadataEntry, 0, len(cr.Spec.ForProvider.Metadata))
	for _, m := range cr.Spec.ForProvider.Metadata {
		out = append(out, zitadel.MetadataEntry{Key: m.Key, Value: m.Value})
	}

	return zitadel.Normalize(out)
}

// metadataDetails publishes the metadata as JSON.
func metadataDetails(entries []zitadel.MetadataEntry) managed.ConnectionDetails {
	out := make(map[string]string, len(entries))
	for _, e := range entries {
		out[e.Key] = e.Value
	}

	encoded, err := json.Marshal(out)
	if err != nil {
		return nil
	}

	return managed.ConnectionDetails{v1alpha1.ConnectionKeyMetadata: encoded}
}

// updateStatus copies the observed metadata into the status of cr.
func updateStatus(cr *v1alpha1.OrganizationMetadata, orgID string, entries []zitadel.MetadataEntry) {
	cr.Status.AtProvider.OrganizationID = common.StringPtr(orgID)

	list := make(v1alpha1.MetadataList, 0, len(entries))
	for _, e := range entries {
		list = append(list, v1alpha1.MetadataEntry{Key: e.Key, Value: e.Value})
	}
	cr.Status.AtProvider.Metadata = list
}
