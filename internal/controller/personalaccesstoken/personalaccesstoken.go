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

// Package personalaccesstoken implements the controller for the
// PersonalAccessToken managed resource.
package personalaccesstoken

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

const (
	errNotPersonalAccessToken = "managed resource is not a PersonalAccessToken custom resource"
	errNoUserID               = "cannot determine the user of the token"
	errResolveUser            = "cannot resolve the user of the token"
)

// DefaultPersonalAccessTokenExpiration is used when a PersonalAccessToken does
// not request a specific expiration. Zitadel always requires an expiration date
// when issuing a Personal Access Token, so this keeps the resource concise
// without leaving the value unbounded.
const DefaultPersonalAccessTokenExpiration = 365 * 24 * time.Hour

// Setup adds a controller that reconciles PersonalAccessToken managed
// resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.PersonalAccessTokenGroupKind,
		v1alpha1.PersonalAccessTokenGroupVersionKind,
		&v1alpha1.PersonalAccessToken{},
		&v1alpha1.PersonalAccessTokenList{},
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

// Observe fetches the token and reports whether it exists and matches the
// desired state.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.PersonalAccessToken)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotPersonalAccessToken)
	}

	if _, err := e.userID(ctx, cr); err != nil {
		if meta.WasDeleted(cr) {
			// Nothing resolves while an object is terminating, so report the
			// external resource as gone. That lets the reconciler run Delete,
			// which lets the finalizer go, instead of retrying an observation
			// that can never succeed.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	t, err := e.client.GetPersonalAccessToken(ctx, id)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get personal access token from Zitadel")
	}

	updateStatus(cr, t)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate(cr, t),
	}, nil
}

// Create issues a new Personal Access Token and publishes it to the connection
// secret. Zitadel only returns the token value once, at creation time.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.PersonalAccessToken)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotPersonalAccessToken)
	}

	cr.Status.SetConditions(xpv1.Creating())

	userID, err := e.userID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if userID == "" {
		return managed.ExternalCreation{}, errors.New(errNoUserID)
	}

	// Zitadel always requires an expiration date, so an unset one falls back to
	// DefaultPersonalAccessTokenExpiration.
	expiration := time.Now().UTC().Add(DefaultPersonalAccessTokenExpiration)
	if cr.Spec.ForProvider.ExpirationDate != nil {
		expiration = cr.Spec.ForProvider.ExpirationDate.UTC()
	}

	tokenID, token, err := e.client.CreatePersonalAccessToken(ctx, userID, expiration.Format(time.RFC3339))
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, tokenID)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
		v1alpha1.ConnectionKeyUserID:  []byte(userID),
		v1alpha1.ConnectionKeyTokenID: []byte(tokenID),
		v1alpha1.ConnectionKeyToken:   []byte(token),
	}}, nil
}

// Update is a no-op: the expiration date of a Personal Access Token cannot be
// changed. The Crossplane reconciler re-creates the token instead, which
// issues a new one and revokes the old one.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*v1alpha1.PersonalAccessToken)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotPersonalAccessToken)
	}

	return managed.ExternalUpdate{}, errors.New("the expiration date of a personal access token cannot be updated; delete the resource to issue a new token")
}

// Delete revokes the Personal Access Token.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.PersonalAccessToken)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotPersonalAccessToken)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.UserID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	userID := cr.Status.AtProvider.UserID
	if userID == nil {
		userID = common.StringPtr(common.Deref(cr.Spec.ForProvider.UserID))
	}

	if *userID == "" {
		// Without a user ID we cannot revoke the token. The token is left in
		// place; deleting the owning user removes it anyway.
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeletePersonalAccessToken(ctx, *userID, id); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot revoke personal access token")
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

// userID resolves the user the token belongs to.
func (e *external) userID(ctx context.Context, cr *v1alpha1.PersonalAccessToken) (string, error) {
	fp := cr.Spec.ForProvider

	// Zitadel only issues a personal access token for a machine user, so the
	// reference is resolved against a ServiceAccount. Resolving a HumanUser here
	// would fail in Zitadel with "Not allowed for this user type".
	id, _, err := common.ResolveServiceAccount(ctx, e.kube, cr, fp.UserRef, fp.UserSelector, fp.UserID)
	if err != nil {
		return "", common.Join(common.ErrResolveUser, err)
	}

	return id, nil
}

// updateStatus copies the observed state of t into the status of cr.
func updateStatus(cr *v1alpha1.PersonalAccessToken, t *zitadel.PersonalAccessToken) {
	cr.Status.AtProvider.TokenID = common.StringPtr(t.TokenID)
	cr.Status.AtProvider.UserID = common.StringPtr(t.UserID)
	cr.Status.AtProvider.OrganizationID = common.StringPtr(t.OrganizationID)
	cr.Status.AtProvider.ExpirationDate = common.ParseTime(t.ExpirationDate)
}

// isUpToDate reports whether the remote token matches the desired state
// described by cr.
func isUpToDate(cr *v1alpha1.PersonalAccessToken, t *zitadel.PersonalAccessToken) bool {
	if cr.Spec.ForProvider.ExpirationDate == nil {
		return true
	}

	return cr.Spec.ForProvider.ExpirationDate.UTC().Format(time.RFC3339) == t.ExpirationDate
}
