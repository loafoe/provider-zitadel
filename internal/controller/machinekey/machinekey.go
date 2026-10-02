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

// Package machinekey implements the controller for the MachineKey managed resource - an RSA key a Zitadel service account authenticates with.
package machinekey

import (
	"context"
	"time"

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

const errNotMachineKey = "managed resource is not a MachineKey custom resource"

// Setup adds a controller that reconciles MachineKey managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.MachineKeyGroupKind,
		v1alpha1.MachineKeyGroupVersionKind,
		&v1alpha1.MachineKey{},
		&v1alpha1.MachineKeyList{},
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

// resolve resolves the service account the key belongs to.
func (e *external) resolve(ctx context.Context, cr *v1alpha1.MachineKey) (serviceAccountID, orgID string, err error) {
	fp := cr.Spec.ForProvider

	// A machine key must be looked up per user, so an existing key is found
	// through the service account it was created for.
	serviceAccountID, orgID, err = common.ResolveServiceAccount(ctx, e.kube, cr,
		fp.ServiceAccountRef, fp.ServiceAccountSelector, fp.ServiceAccountID)
	if err != nil {
		return "", "", common.Join(errors.New("cannot resolve the service account of the key"), err)
	}

	if serviceAccountID == "" {
		return "", "", errors.New("serviceAccountID, serviceAccountRef or serviceAccountSelector must be set")
	}

	// Resolving by reference reads the organization straight from the service
	// account, which stays correct even after the service account has been
	// replaced. A service account given by ID cannot be looked up that way, so
	// there the organization observed on an existing key is the only source.
	if orgID == "" && fp.ServiceAccountRef == nil && fp.ServiceAccountSelector == nil {
		orgID = common.Deref(cr.Status.AtProvider.OrganizationID)
	}

	if orgID == "" {
		return "", "", errors.New("the organization of the service account is not known yet")
	}

	return serviceAccountID, orgID, nil
}

// Observe reports whether the key exists on the service account.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.MachineKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotMachineKey)
	}

	serviceAccountID, orgID, err := e.resolve(ctx, cr)
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

	keyID := meta.GetExternalName(cr)
	if keyID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	k, err := e.client.GetMachineKey(ctx, orgID, serviceAccountID, keyID)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the machine key from Zitadel")
	}

	// A key that moved to a different service account is not the key this
	// resource manages.
	if k.UserID != serviceAccountID {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	updateStatus(cr, k)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists: true,
		// The expiration date is fixed at creation: Zitadel has no way to
		// change it, so only a genuinely different key counts as drift. The
		// key material is never re-read, because Zitadel does not return the
		// private half.
		ResourceUpToDate: common.EqualStringSlices(
			[]string{common.FormatTime(cr.Spec.ForProvider.ExpirationDate)},
			[]string{k.ExpirationDate},
		),
		ConnectionDetails: managed.ConnectionDetails{
			v1alpha1.ConnectionKeyKeyID: []byte(k.KeyID),
		},
	}, nil
}

// Create generates a key pair, registers the public half with Zitadel and
// publishes the key.json document to the connection secret.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.MachineKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotMachineKey)
	}

	cr.Status.SetConditions(xpv1.Creating())

	serviceAccountID, _, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	expiration := expirationOf(cr)
	bits := zitadel.DefaultMachineKeyBits
	if cr.Spec.ForProvider.KeyBits != nil {
		bits = *cr.Spec.ForProvider.KeyBits
	}

	keyID, key, err := e.client.CreateMachineKey(ctx, zitadel.CreateMachineKeyInput{
		UserID:         serviceAccountID,
		ExpirationDate: expiration,
		Bits:           bits,
	})
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	encoded, err := zitadel.EncodeMachineKey(key)
	if err != nil {
		// The key exists in Zitadel but its private half would be lost. Fail
		// loudly so it is removed and recreated rather than silently orphaned.
		_ = e.client.DeleteMachineKey(ctx, serviceAccountID, keyID)
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, keyID)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
		v1alpha1.ConnectionKeyKeyID:   []byte(keyID),
		v1alpha1.ConnectionKeyKeyJSON: encoded,
	}}, nil
}

// Update is not possible: Zitadel cannot change the expiration date of a key
// and cannot rotate key material in place. Changing either replaces the key.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*v1alpha1.MachineKey)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotMachineKey)
	}

	return managed.ExternalUpdate{}, errors.New(
		"a machine key cannot be updated in place: Zitadel fixes the expiration date at creation and stores no private key material. " +
			"Delete the MachineKey and create it again to rotate the key.")
}

// Delete removes the key from the service account.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.MachineKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotMachineKey)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.ServiceAccountID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeleteMachineKey(ctx,
		common.Deref(cr.Status.AtProvider.ServiceAccountID), meta.GetExternalName(cr)); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{}, nil
}

// expirationOf returns the expiration date of the key, defaulting to a year out
// because Zitadel requires one.
func expirationOf(cr *v1alpha1.MachineKey) time.Time {
	if cr.Spec.ForProvider.ExpirationDate != nil {
		return cr.Spec.ForProvider.ExpirationDate.Time
	}

	return time.Now().Add(defaultMachineKeyValidity)
}

// defaultMachineKeyValidity is how long a key lives when no expiration date is
// given. A year is a conservative default: long enough to avoid surprise
// outages, short enough that rotation is a routine exercise.
const defaultMachineKeyValidity = 365 * 24 * time.Hour

// updateStatus copies the observed key into the status of cr.
func updateStatus(cr *v1alpha1.MachineKey, k *zitadel.MachineKey) {
	cr.Status.AtProvider.KeyID = common.StringPtr(k.KeyID)
	cr.Status.AtProvider.ServiceAccountID = common.StringPtr(k.UserID)

	if k.OrganizationID != "" {
		cr.Status.AtProvider.OrganizationID = common.StringPtr(k.OrganizationID)
	}
	cr.Status.AtProvider.ExpirationDate = common.ParseTime(k.ExpirationDate)
	cr.Status.AtProvider.CreationDate = common.ParseTime(k.CreationDate)
}
