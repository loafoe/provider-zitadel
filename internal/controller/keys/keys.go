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

// Package keys holds the controllers for Zitadel's signing keys, the key an
// application authenticates itself with, and a user's membership of a project.
//
// The three are unrelated to each other and share nothing but the provider, so
// each is its own controller here rather than one harness. What they have in
// common is that Zitadel generates their material: the private half of a signing
// key never leaves Zitadel, and the private half of an application key is
// returned exactly once, at creation.
package keys

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

// Setup registers the key and membership controllers.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, s := range []func(ctrl.Manager, controller.Options) error{
		SetupWebKey,
		SetupApplicationKey,
		SetupProjectMember,
	} {
		if err := s(mgr, o); err != nil {
			return err
		}
	}

	return nil
}

// SetupWebKey registers the WebKey controller.
func SetupWebKey(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.WebKeyGroupKind,
		v1alpha1.WebKeyGroupVersionKind,
		&v1alpha1.WebKey{},
		&v1alpha1.WebKeyList{},
		newWebKeyExternal,
	)
}

// SetupApplicationKey registers the ApplicationKey controller.
func SetupApplicationKey(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ApplicationKeyGroupKind,
		v1alpha1.ApplicationKeyGroupVersionKind,
		&v1alpha1.ApplicationKey{},
		&v1alpha1.ApplicationKeyList{},
		newApplicationKeyExternal,
	)
}

// SetupProjectMember registers the ProjectMember controller.
func SetupProjectMember(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ProjectMemberGroupKind,
		v1alpha1.ProjectMemberGroupVersionKind,
		&v1alpha1.ProjectMember{},
		&v1alpha1.ProjectMemberList{},
		newProjectMemberExternal,
	)
}

// Web keys
//
// A signing key has an identity Zitadel hands out and no material to compare:
// the private half never leaves Zitadel, and the public half is not something
// Zitadel will read back either. So there is nothing to detect drift on beyond
// the algorithm, which cannot be changed on an existing key.

type webKeyExternal struct {
	client *zitadel.Client
}

func newWebKeyExternal(_ context.Context, _ client.Client, _ resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
	return &webKeyExternal{client: zc}, nil
}

func (e *webKeyExternal) Disconnect(_ context.Context) error { return e.client.Close() }

func (e *webKeyExternal) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.WebKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New("managed resource is not a WebKey")
	}

	id := meta.GetExternalName(cr)

	// The identifier has to be one Zitadel handed out. Anything else is a
	// leftover from a resource that never created one, and reporting it as
	// present would leave Zitadel without the key the manifest describes.
	if !common.IsZitadelID(id) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	k, err := e.client.GetWebKey(ctx, id)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if k == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.ID = k.ID
	cr.Status.AtProvider.Algorithm = k.Algorithm
	cr.Status.AtProvider.State = k.State
	cr.Status.AtProvider.CreationDate = k.CreationDate
	cr.Status.AtProvider.ChangeDate = k.ChangeDate
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *webKeyExternal) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.WebKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New("managed resource is not a WebKey")
	}

	cr.SetConditions(xpv1.Creating())

	fp := cr.Spec.ForProvider
	id, err := e.client.CreateWebKey(ctx, zitadel.WebKeyAlgorithm{
		Type:      common.Deref(fp.Algorithm),
		RSABits:   common.Deref(fp.RSABits),
		RSAHasher: common.Deref(fp.RSAHasher),
	})
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, id)

	return managed.ExternalCreation{}, nil
}

// Update is a no-op: Zitadel cannot change a key, only replace it. The
// algorithm in the manifest is therefore read once at creation and a change to
// it is reported here rather than silently ignored.
func (e *webKeyExternal) Update(_ context.Context, _ resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (e *webKeyExternal) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.WebKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New("managed resource is not a WebKey")
	}

	cr.SetConditions(xpv1.Deleting())

	return managed.ExternalDelete{}, e.client.DeleteWebKey(ctx, meta.GetExternalName(cr))
}

// Application keys
//
// The private half is returned exactly once, at creation, so the connection
// secret is the only place it can be kept. A key whose secret was lost is still
// readable by Zitadel and cannot be re-read, which is why creating one deletes
// it again rather than leaving an orphan nobody holds the key to.

type applicationKeyExternal struct {
	kube   client.Client
	client *zitadel.Client
}

func newApplicationKeyExternal(_ context.Context, kube client.Client, _ resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
	return &applicationKeyExternal{kube: kube, client: zc}, nil
}

func (e *applicationKeyExternal) Disconnect(ctx context.Context) error { return e.client.Close() }

func (e *applicationKeyExternal) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ApplicationKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New("managed resource is not an ApplicationKey")
	}

	id := meta.GetExternalName(cr)
	if !common.IsZitadelID(id) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	k, err := e.client.GetAppKey(ctx, id)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if k == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Zitadel refuses to remove a key it cannot place, so the project and the
	// application it belongs to are recorded while the key is live. A terminating
	// object must not resolve references: nothing guarantees they still resolve,
	// and what it needs is already in its status.
	if !meta.WasDeleted(cr) {
		projectID, appID, err := e.application(ctx, cr)
		if err != nil {
			return managed.ExternalObservation{}, err
		}

		cr.Status.AtProvider.ProjectID = projectID
		cr.Status.AtProvider.ApplicationID = appID
	}
	cr.Status.AtProvider.ID = k.ID
	cr.Status.AtProvider.CreationDate = k.CreationDate
	cr.Status.AtProvider.ExpirationDate = k.ExpirationDate
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (e *applicationKeyExternal) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ApplicationKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New("managed resource is not an ApplicationKey")
	}

	cr.SetConditions(xpv1.Creating())

	if _, err := e.organization(ctx, cr); err != nil {
		return managed.ExternalCreation{}, err
	}

	projectID, appID, err := e.application(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	id, encoded, err := e.client.AddAppKey(ctx, projectID, appID, common.Deref(cr.Spec.ForProvider.ExpirationDate))
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if len(encoded) == 0 {
		// The key exists in Zitadel but its private half would be lost, and it
		// cannot be read again. Removing it lets a fresh one be made rather than
		// leaving a key nobody holds.
		_ = e.client.DeleteAppKey(ctx, projectID, appID, id)
		return managed.ExternalCreation{}, errors.New("zitadel returned no private key material, so this key could never be used; it has been removed and will be made again")
	}

	meta.SetExternalName(cr, id)

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
		v1alpha1.ConnectionKeyKeyID:   []byte(id),
		v1alpha1.ConnectionKeyKeyJSON: encoded,
	}}, nil
}

// Update is a no-op: Zitadel cannot change a key's expiry, only replace it.
func (e *applicationKeyExternal) Update(_ context.Context, _ resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (e *applicationKeyExternal) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ApplicationKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New("managed resource is not an ApplicationKey")
	}

	cr.SetConditions(xpv1.Deleting())

	// The project and the application were recorded when the key was made, which
	// is what a terminating object uses: references are not guaranteed to still
	// resolve, and the key has to be removed from where it actually is.
	projectID, appID := cr.Status.AtProvider.ProjectID, cr.Status.AtProvider.ApplicationID
	if projectID == "" || appID == "" {
		resolvedProject, resolvedApp, err := e.application(ctx, cr)
		switch {
		case err == nil:
			projectID, appID = resolvedProject, resolvedApp

		case common.IsReferenceGone(err):
			// The application is gone, and with it the key.
			return managed.ExternalDelete{}, nil

		default:
			return managed.ExternalDelete{}, err
		}
	}

	return managed.ExternalDelete{}, e.client.DeleteAppKey(ctx, projectID, appID, meta.GetExternalName(cr))
}

func (e *applicationKeyExternal) organization(ctx context.Context, cr *v1alpha1.ApplicationKey) (string, error) {
	fp := cr.Spec.ForProvider

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	orgID, err := common.ResolveOrganizationID(ctx, e.kube, cr, fp.OrganizationRef,
		fp.OrganizationSelector, fp.OrganizationID, orgDefault,
		common.CurrentIfDeleting(meta.WasDeleted(cr), fp.OrganizationID))
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	if orgID == "" {
		return "", errors.New(common.ErrNoOrganizationID.Error())
	}

	return orgID, nil
}

// application resolves the project and the application, which is what Zitadel
// identifies a key by.
//
// The organization is not asked for: Zitadel's v2 application API addresses a
// key by its own ID, and the organization only matters for the v1 management API
// that is no longer used.
func (e *applicationKeyExternal) application(ctx context.Context, cr *v1alpha1.ApplicationKey) (string, string, error) {
	fp := cr.Spec.ForProvider

	projectID, err := common.ResolveProjectID(ctx, e.kube, cr, fp.ProjectRef,
		fp.ProjectSelector, fp.ProjectID, "")
	if err != nil {
		return "", "", common.Join(common.ErrNoProjectID, err)
	}

	appID, err := common.ResolveApplicationID(ctx, e.kube, cr, fp.ApplicationRef,
		fp.ApplicationSelector, fp.ApplicationID)
	if err != nil {
		return "", "", common.Join(common.ErrNoApplicationID, err)
	}

	return projectID, appID, nil
}

// Project memberships
//
// A grant rather than a setting: the roles are the only thing that can differ,
// so an update replaces them and everything else about the membership is
// reported.

type projectMemberExternal struct {
	kube   client.Client
	client *zitadel.Client
}

func newProjectMemberExternal(_ context.Context, kube client.Client, _ resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
	return &projectMemberExternal{kube: kube, client: zc}, nil
}

func (e *projectMemberExternal) Disconnect(ctx context.Context) error { return e.client.Close() }

func (e *projectMemberExternal) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ProjectMember)
	if !ok {
		return managed.ExternalObservation{}, errors.New("managed resource is not a ProjectMember")
	}

	scope, err := e.scope(ctx, cr)
	if err != nil {
		if meta.WasDeleted(cr) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	m, err := e.client.GetProjectMember(ctx, scope.projectID, scope.userID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	if m == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.UserID = m.UserID
	cr.Status.AtProvider.Roles = m.Roles
	cr.Status.AtProvider.UserName = m.UserName
	cr.Status.AtProvider.DisplayName = m.DisplayName
	cr.Status.AtProvider.ProjectName = m.ProjectName
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists: true,
		// Zitadel returns the roles in an order of its own, so they are compared
		// as a set rather than as written.
		ResourceUpToDate: common.EqualStringSlices(cr.Spec.ForProvider.Roles, m.Roles),
	}, nil
}

func (e *projectMemberExternal) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ProjectMember)
	if !ok {
		return managed.ExternalCreation{}, errors.New("managed resource is not a ProjectMember")
	}

	cr.SetConditions(xpv1.Creating())

	scope, err := e.scope(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, e.client.AddProjectMember(ctx, scope.projectID, scope.userID,
		cr.Spec.ForProvider.Roles)
}

func (e *projectMemberExternal) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ProjectMember)
	if !ok {
		return managed.ExternalUpdate{}, errors.New("managed resource is not a ProjectMember")
	}

	scope, err := e.scope(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, e.client.UpdateProjectMember(ctx, scope.projectID, scope.userID,
		cr.Spec.ForProvider.Roles)
}

func (e *projectMemberExternal) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ProjectMember)
	if !ok {
		return managed.ExternalDelete{}, errors.New("managed resource is not a ProjectMember")
	}

	cr.SetConditions(xpv1.Deleting())

	// A terminating object must not follow a reference that has since moved: it
	// exists to undo what it did, so it acts on the user it recorded. The project
	// still has to be resolved, because Zitadel addresses the grant by both.
	recorded := cr.Status.AtProvider.UserID

	scope, err := e.scope(ctx, cr)
	switch {
	case common.IsReferenceGone(err):
		// The project is gone, and with it the membership.

		return managed.ExternalDelete{}, nil

	case err != nil:
		return managed.ExternalDelete{}, err
	}

	userID := recorded
	if userID == "" {
		userID = scope.userID
	}

	return managed.ExternalDelete{}, e.client.RemoveProjectMember(ctx, scope.projectID, userID)
}

// memberScope is the project and user a membership is between.
type memberScope struct {
	projectID string
	userID    string
}

func (e *projectMemberExternal) scope(ctx context.Context, cr *v1alpha1.ProjectMember) (memberScope, error) {
	fp := cr.Spec.ForProvider

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return memberScope{}, common.Join(common.ErrNoOrganizationID, err)
	}

	orgID, err := common.ResolveOrganizationID(ctx, e.kube, cr, fp.OrganizationRef,
		fp.OrganizationSelector, fp.OrganizationID, orgDefault,
		common.CurrentIfDeleting(meta.WasDeleted(cr), fp.OrganizationID))
	if err != nil {
		return memberScope{}, common.Join(common.ErrNoOrganizationID, err)
	}

	projectID, err := common.ResolveProjectID(ctx, e.kube, cr, fp.ProjectRef,
		fp.ProjectSelector, fp.ProjectID, "")
	if err != nil {
		return memberScope{}, common.Join(common.ErrNoProjectID, err)
	}

	userID, err := common.ResolveUserID(ctx, e.kube, cr, fp.UserRef, fp.UserSelector, fp.UserID, "")
	if err != nil {
		return memberScope{}, common.Join(common.ErrNoUserID, err)
	}

	_ = orgID

	return memberScope{projectID: projectID, userID: userID}, nil
}
