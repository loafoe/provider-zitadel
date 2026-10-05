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

// Package apiapplication implements the controller for the ApplicationAPI managed resource - an OAuth2 client for service to service calls.
package apiapplication

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	apiv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"

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

const errNotApplicationAPI = "managed resource is not a ApplicationAPI custom resource"

// Setup adds a controller that reconciles ApplicationAPI managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ApplicationAPIGroupKind,
		v1alpha1.ApplicationAPIGroupVersionKind,
		&v1alpha1.ApplicationAPI{},
		&v1alpha1.ApplicationAPIList{},
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

// resolve resolves the project the application belongs to.
func (e *external) resolve(ctx context.Context, cr *v1alpha1.ApplicationAPI) (string, error) {
	fp := cr.Spec.ForProvider

	projectID, err := common.ResolveProjectID(ctx, e.kube, cr, fp.ProjectRef, fp.ProjectSelector, fp.ProjectID,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.ProjectID))
	if err != nil {
		return "", common.Join(common.ErrNoProjectID, err)
	}

	if projectID == "" {
		return "", errors.New(common.ErrNoProjectID.Error())
	}

	return projectID, nil
}

// authMethod resolves the desired client authentication method.
func authMethod(cr *v1alpha1.ApplicationAPI) (apiv2.APIAuthMethodType, error) {
	return zitadel.APIAuthMethodToProto(common.Value(cr.Spec.ForProvider.AuthMethodType))
}

// Observe reports whether the application exists and matches.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ApplicationAPI)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotApplicationAPI)
	}

	projectID, err := e.resolve(ctx, cr)
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

	appID := meta.GetExternalName(cr)
	if appID == "" {
		appID = common.Deref(cr.Spec.ForProvider.ID)
	}

	if appID == "" {
		found, err := e.client.FindAPIApplicationByName(ctx, projectID, cr.Spec.ForProvider.Name)
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, "cannot look up the application in Zitadel")
		}
		if found == nil {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		appID = found.ApplicationID
	}

	app, err := e.client.GetAPIApplication(ctx, appID)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the API application from Zitadel")
	}

	// The v2 UpdateApplication does not persist OIDC configuration changes on
	// Zitadel 4.x, so the standalone controller image tag is published but the
	// API application is updated through the same call only after the name and
	// auth method actually drifted.
	meta.SetExternalName(cr, app.ApplicationID)

	updateStatus(cr, app)
	cr.Status.SetConditions(xpv1.Available())

	details := managed.ConnectionDetails{
		v1alpha1.ConnectionKeyClientID: []byte(app.ClientID),
	}

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  isUpToDate(cr, app),
		ConnectionDetails: details,
	}, nil
}

// Create creates the application and publishes its client credentials. The
// client secret is only returned once, so it is written to the connection
// secret here and never re-read.
//
//nolint:gocyclo // each branch decides one thing that goes into the connection secret
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ApplicationAPI)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotApplicationAPI)
	}

	cr.Status.SetConditions(xpv1.Creating())

	projectID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	method, err := authMethod(cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	app, secret, err := e.client.CreateAPIApplication(ctx, zitadel.CreateAPIApplicationInput{
		ProjectID:      projectID,
		ID:             common.Deref(cr.Spec.ForProvider.ID),
		Name:           cr.Spec.ForProvider.Name,
		AuthMethodType: method,
	})
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, app.ApplicationID)

	if state := cr.Spec.ForProvider.State; state != nil && *state == v1alpha1.ApplicationStateInactive {
		if err := e.client.SetAPIApplicationState(ctx, app.ApplicationID, zitadel.StateActive, zitadel.StateInactive); err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	details := managed.ConnectionDetails{
		v1alpha1.ConnectionKeyClientID: []byte(app.ClientID),
	}
	if secret != "" {
		details[v1alpha1.ConnectionKeyClientSecret] = []byte(secret)
	}

	if e.generatePrivateKey(cr) {
		key, err := generateRSAKey()
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		details[v1alpha1.ConnectionKeyPrivateKey] = key
	}

	return managed.ExternalCreation{ConnectionDetails: details}, nil
}

// Update renames the application, changes its auth method and activates or
// deactivates it.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ApplicationAPI)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotApplicationAPI)
	}

	projectID, err := e.resolve(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	current, err := e.client.GetAPIApplication(ctx, meta.GetExternalName(cr))
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if current.Name != cr.Spec.ForProvider.Name {
		method, err := authMethod(cr)
		if err != nil {
			return managed.ExternalUpdate{}, err
		}
		if err := e.client.UpdateAPIApplication(ctx, current.ApplicationID, projectID, cr.Spec.ForProvider.Name, method); err != nil {
			return managed.ExternalUpdate{}, err
		}
	}

	if err := e.client.SetAPIApplicationState(ctx, current.ApplicationID, current.State, common.Value(cr.Spec.ForProvider.State)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the application.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ApplicationAPI)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotApplicationAPI)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.ProjectID, cr.Status.AtProvider.ApplicationID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	appID := common.Deref(cr.Status.AtProvider.ApplicationID)
	if appID == "" {
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.DeleteAPIApplication(ctx, appID, common.Deref(cr.Status.AtProvider.ProjectID)); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{}, nil
}

// generateRSAKey creates the RSA key pair a PrivateKeyJwt client signs with.
func generateRSAKey() ([]byte, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, errors.Wrap(err, "cannot generate an RSA key for the application")
	}

	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}), nil
}

// generatePrivateKey reports whether the resource asked for a key to be
// generated for it.
func (e *external) generatePrivateKey(cr *v1alpha1.ApplicationAPI) bool {
	if cr.Spec.ForProvider.GeneratePrivateKey == nil {
		return false
	}

	return *cr.Spec.ForProvider.GeneratePrivateKey &&
		common.Value(cr.Spec.ForProvider.AuthMethodType) == string(v1alpha1.APIAuthMethodTypePrivateKeyJwt)
}

// isUpToDate reports whether the remote application matches the desired state.
// The client secret is never compared: Zitadel does not return it again.
func isUpToDate(cr *v1alpha1.ApplicationAPI, app *zitadel.APIApplication) bool {
	fp := cr.Spec.ForProvider

	if fp.Name != app.Name {
		return false
	}

	if common.Value(fp.AuthMethodType) != app.AuthMethodType {
		return false
	}

	if fp.State != nil && string(*fp.State) != app.State {
		return false
	}

	return true
}

// updateStatus copies the observed application into the status of cr.
func updateStatus(cr *v1alpha1.ApplicationAPI, app *zitadel.APIApplication) {
	cr.Status.AtProvider.ApplicationID = common.StringPtr(app.ApplicationID)
	cr.Status.AtProvider.ProjectID = common.StringPtr(app.ProjectID)
	cr.Status.AtProvider.Name = common.StringPtr(app.Name)
	cr.Status.AtProvider.ClientID = common.StringPtr(app.ClientID)

	if app.State != "" {
		state := v1alpha1.ApplicationState(app.State)
		cr.Status.AtProvider.State = &state
	}
	if app.AuthMethodType != "" {
		method := v1alpha1.APIAuthMethodType(app.AuthMethodType)
		cr.Status.AtProvider.AuthMethodType = &method
	}

	cr.Status.AtProvider.CreationDate = common.ParseTime(app.CreationDate)
	cr.Status.AtProvider.ChangeDate = common.ParseTime(app.ChangeDate)
}
