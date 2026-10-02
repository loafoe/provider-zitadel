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

// Package serviceaccount implements the controller for the ServiceAccount
// managed resource - a Zitadel machine user.
package serviceaccount

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
	errNotServiceAccount = "managed resource is not a ServiceAccount custom resource"
	errNoOrgID           = "cannot determine the organization of the service account"
)

// Setup adds a controller that reconciles ServiceAccount managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ServiceAccountGroupKind,
		v1alpha1.ServiceAccountGroupVersionKind,
		&v1alpha1.ServiceAccount{},
		&v1alpha1.ServiceAccountList{},
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

// Observe fetches the service account and reports whether it exists and matches
// the desired state.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ServiceAccount)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotServiceAccount)
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

	u, err := e.observe(ctx, cr, orgID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if u == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	updateStatus(cr, u)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  isUpToDate(cr, u),
		ConnectionDetails: connectionDetails(u),
	}, nil
}

// observe returns the service account managed by cr, recovering from a lost
// external name by looking the account up by username or name within the
// organization.
func (e *external) observe(ctx context.Context, cr *v1alpha1.ServiceAccount, orgID string) (*zitadel.User, error) {
	u, found, err := common.Recover(
		meta.GetExternalName(cr),
		common.Deref(cr.Spec.ForProvider.ID),
		func(id string) (*zitadel.User, error) { return e.client.GetUser(ctx, id) },
		func() (*zitadel.User, error) {
			if orgID == "" {
				return nil, nil
			}

			users, err := e.client.ListUsersByOrgID(ctx, orgID)
			if err != nil {
				return nil, err
			}

			return findServiceAccount(cr, users), nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "cannot look up service account in Zitadel")
	}

	if !found {
		return nil, nil
	}

	meta.SetExternalName(cr, u.UserID)

	return u, nil
}

// findServiceAccount returns the machine user in users that matches the desired
// state of cr. A username match takes precedence over a display name match. It
// returns nil when no service account matches.
func findServiceAccount(cr *v1alpha1.ServiceAccount, users []*zitadel.User) *zitadel.User {
	fp := cr.Spec.ForProvider

	for _, u := range users {
		if u.Machine == nil {
			continue
		}

		if fp.UserName != nil && u.UserName == *fp.UserName {
			return u
		}
	}

	if fp.UserName != nil || fp.Name == nil {
		return nil
	}

	for _, u := range users {
		if u.Machine != nil && u.Machine.Name == *fp.Name {
			return u
		}
	}

	return nil
}

// Create creates the machine user in Zitadel.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ServiceAccount)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotServiceAccount)
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

	tokenType, err := zitadel.AccessTokenTypeToProto(common.Value(fp.AccessTokenType))
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot set access token type")
	}

	in := zitadel.CreateServiceAccountInput{
		OrganizationID:  orgID,
		UserID:          common.Deref(fp.ID),
		UserName:        common.Deref(fp.UserName),
		Name:            common.Deref(fp.Name),
		Description:     common.Deref(fp.Description),
		AccessTokenType: tokenType,
	}

	for _, m := range fp.Metadata {
		in.Metadata = append(in.Metadata, zitadel.MetadataEntry{Key: m.Key, Value: m.Value})
	}

	id, err := e.client.CreateServiceAccount(ctx, in)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create service account in Zitadel")
	}

	meta.SetExternalName(cr, id)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
		v1alpha1.ConnectionKeyUserID: []byte(id),
	}}, nil
}

// Update reconciles the mutable attributes and the state of the service
// account.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ServiceAccount)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotServiceAccount)
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
		return managed.ExternalUpdate{}, errors.New("cannot update a service account that does not exist")
	}

	if err := e.reconcileAttributes(ctx, cr, current); err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.SetUserState(ctx, current.UserID, current.State, common.Value(cr.Spec.ForProvider.State)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// reconcileAttributes applies the username, name, description and access token
// type drift of a service account.
func (e *external) reconcileAttributes(ctx context.Context, cr *v1alpha1.ServiceAccount, current *zitadel.User) error {
	fp := cr.Spec.ForProvider

	if want := common.Deref(fp.UserName); want != "" && want != current.UserName {
		if err := e.client.UpdateUsername(ctx, current.UserID, want, true); err != nil {
			return err
		}
	}

	if err := e.reconcileMachine(ctx, current, fp); err != nil {
		return err
	}

	return e.reconcileAccessTokenType(ctx, fp, current)
}

// reconcileMachine applies the display name and description drift of a service
// account.
func (e *external) reconcileMachine(ctx context.Context, current *zitadel.User, fp v1alpha1.ServiceAccountParameters) error {
	name := common.Deref(fp.Name)
	hasNameDrift := name != "" && (current.Machine == nil || name != current.Machine.Name)

	var description *string
	if fp.Description != nil && (current.Machine == nil || *fp.Description != current.Machine.Description) {
		description = fp.Description
	}

	if !hasNameDrift && description == nil {
		return nil
	}

	return e.client.UpdateServiceAccount(ctx, current.UserID, name, description, nil)
}

// reconcileAccessTokenType applies accessTokenType drift. Switching a service
// account from opaque "Bearer" tokens to signed "Jwt" tokens is what makes it
// usable as a ProviderConfig, so it is not an exotic setting.
func (e *external) reconcileAccessTokenType(ctx context.Context, fp v1alpha1.ServiceAccountParameters, current *zitadel.User) error {
	if fp.AccessTokenType == nil {
		return nil
	}

	want, err := zitadel.AccessTokenTypeToProto(string(*fp.AccessTokenType))
	if err != nil {
		return errors.Wrap(err, "cannot set the access token type")
	}

	if current.Machine != nil && current.Machine.AccessTokenType == string(*fp.AccessTokenType) {
		return nil
	}

	return e.client.UpdateServiceAccount(ctx, current.UserID, "", nil, &want)
}

// Delete removes the machine user from Zitadel.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ServiceAccount)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotServiceAccount)
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

	if err := e.client.DeleteUser(ctx, id); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete service account from Zitadel")
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect releases the underlying Zitadel client.
func (e *external) Disconnect(ctx context.Context) error {
	return e.client.Close()
}

// organizationID resolves the organization the service account belongs to.
func (e *external) organizationID(ctx context.Context, cr *v1alpha1.ServiceAccount) (string, error) {
	fp := cr.Spec.ForProvider

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

// connectionDetails returns the connection details of an existing service
// account.
func connectionDetails(u *zitadel.User) managed.ConnectionDetails {
	details := managed.ConnectionDetails{
		v1alpha1.ConnectionKeyUserID: []byte(u.UserID),
	}

	name := u.PreferredLoginName
	if name == "" {
		name = u.UserName
	}
	if name != "" {
		details[v1alpha1.ConnectionKeyUsername] = []byte(name)
	}

	return details
}

// updateStatus copies the observed state of u into the status of cr.
func updateStatus(cr *v1alpha1.ServiceAccount, u *zitadel.User) {
	cr.Status.AtProvider.ID = common.StringPtr(u.UserID)

	if u.OrganizationID != "" {
		cr.Status.AtProvider.OrganizationID = common.StringPtr(u.OrganizationID)
	}
	cr.Status.AtProvider.UserName = common.StringPtr(u.UserName)
	cr.Status.AtProvider.CreationDate = common.ParseTime(u.CreationDate)
	cr.Status.AtProvider.ChangeDate = common.ParseTime(u.ChangeDate)

	if u.PreferredLoginName != "" {
		cr.Status.AtProvider.PreferredLoginName = common.StringPtr(u.PreferredLoginName)
	}

	if u.State != "" {
		state := v1alpha1.UserState(u.State)
		cr.Status.AtProvider.State = &state
	}

	if u.Machine == nil {
		return
	}

	if u.Machine.Name != "" {
		cr.Status.AtProvider.Name = common.StringPtr(u.Machine.Name)
	}
	if u.Machine.Description != "" {
		cr.Status.AtProvider.Description = common.StringPtr(u.Machine.Description)
	}
	if u.Machine.AccessTokenType != "" {
		t := v1alpha1.AccessTokenType(u.Machine.AccessTokenType)
		cr.Status.AtProvider.AccessTokenType = &t
	}

	cr.Status.AtProvider.HasSecret = common.BoolPtr(u.Machine.HasSecret)
}

// isUpToDate reports whether the remote service account matches the desired
// state described by cr. Metadata is create-only and cannot be drift detected
// because Zitadel never returns it.
//
//nolint:gocyclo // flat comparison of the optional forProvider fields
func isUpToDate(cr *v1alpha1.ServiceAccount, u *zitadel.User) bool {
	fp := cr.Spec.ForProvider

	if u.Machine == nil {
		return false
	}

	if fp.UserName != nil && *fp.UserName != u.UserName {
		return false
	}

	if fp.Name != nil && *fp.Name != u.Machine.Name {
		return false
	}

	if fp.Description != nil && *fp.Description != u.Machine.Description {
		return false
	}

	if fp.AccessTokenType != nil && string(*fp.AccessTokenType) != u.Machine.AccessTokenType {
		return false
	}

	if fp.State != nil && string(*fp.State) != u.State {
		return false
	}

	return true
}
