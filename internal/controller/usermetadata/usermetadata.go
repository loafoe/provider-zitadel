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

// Package usermetadata implements the controller for the UserMetadata managed resource - the metadata of a user.
package usermetadata

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

const errNotUserMetadata = "managed resource is not a UserMetadata custom resource"

// Setup adds a controller that reconciles UserMetadata managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.UserMetadataGroupKind,
		v1alpha1.UserMetadataGroupVersionKind,
		&v1alpha1.UserMetadata{},
		&v1alpha1.UserMetadataList{},
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

// resolve resolves the user whose metadata is managed.
func (e *external) resolve(ctx context.Context, cr *v1alpha1.UserMetadata) (string, error) {
	fp := cr.Spec.ForProvider

	userID, err := common.ResolveUserID(ctx, e.kube, cr, fp.UserRef, fp.UserSelector, fp.UserID, common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.UserID))
	if err != nil {
		return "", common.Join(common.ErrNoUserID, err)
	}

	if userID == "" {
		return "", errors.New(common.ErrNoUserID.Error())
	}

	return userID, nil
}

// Observe reports whether the metadata of the user matches the desired set.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.UserMetadata)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserMetadata)
	}

	userID, err := e.resolve(ctx, cr)
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

	current, err := e.client.GetUserMetadata(ctx, userID)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the user metadata from Zitadel")
	}

	updateStatus(cr, userID, current)
	cr.Status.SetConditions(xpv1.Available())

	upToDate := zitadel.EqualMetadata(desired(cr), current)

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
		ResourceUpToDate:  upToDate,
		ConnectionDetails: metadataDetails(current),
	}, nil
}

// Create writes the desired metadata set.
//
// Zitadel answers a metadata write for a user that does not exist with a Not
// Found, which is what makes this resource's Create equivalent to Update: there
// is nothing to create, only a set to replace.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.UserMetadata)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUserMetadata)
	}

	cr.Status.SetConditions(xpv1.Creating())

	userID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.client.ApplyUserMetadata(ctx, userID, desired(cr)); err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, userID)

	return managed.ExternalCreation{ConnectionDetails: metadataDetails(desired(cr))}, nil
}

// Update replaces the metadata set.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.UserMetadata)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUserMetadata)
	}

	userID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.ApplyUserMetadata(ctx, userID, desired(cr)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{ConnectionDetails: metadataDetails(desired(cr))}, nil
}

// Delete clears the metadata of the user.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.UserMetadata)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUserMetadata)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.UserID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeleteUserMetadata(ctx, common.Deref(cr.Status.AtProvider.UserID)); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{}, nil
}

// desired returns the metadata entries the resource asks for.
func desired(cr *v1alpha1.UserMetadata) []zitadel.MetadataEntry {
	out := make([]zitadel.MetadataEntry, 0, len(cr.Spec.ForProvider.Metadata))
	for _, m := range cr.Spec.ForProvider.Metadata {
		out = append(out, zitadel.MetadataEntry{Key: m.Key, Value: m.Value})
	}

	return zitadel.Normalize(out)
}

// metadataDetails publishes the metadata as JSON, which is easier to consume
// from a shell than a set of individual keys.
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
func updateStatus(cr *v1alpha1.UserMetadata, userID string, entries []zitadel.MetadataEntry) {
	cr.Status.AtProvider.UserID = common.StringPtr(userID)

	list := make(v1alpha1.MetadataList, 0, len(entries))
	for _, e := range entries {
		list = append(list, v1alpha1.MetadataEntry{Key: e.Key, Value: e.Value})
	}
	cr.Status.AtProvider.Metadata = list
}
