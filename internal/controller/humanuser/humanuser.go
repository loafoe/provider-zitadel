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

// Package humanuser implements the controller for the HumanUser managed
// resource.
package humanuser

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
	errNotHumanUser = "managed resource is not a HumanUser custom resource"
	errNoOrgID      = "cannot determine the organization of the user"
)

// Setup adds a controller that reconciles HumanUser managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.HumanUserGroupKind,
		v1alpha1.HumanUserGroupVersionKind,
		&v1alpha1.HumanUser{},
		&v1alpha1.HumanUserList{},
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

// Observe fetches the user and reports whether it exists and matches the
// desired state.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.HumanUser)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotHumanUser)
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

// observe returns the user managed by cr, recovering from a lost external name
// by looking the user up by username or email within the organization.
func (e *external) observe(ctx context.Context, cr *v1alpha1.HumanUser, orgID string) (*zitadel.User, error) {
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

			return e.findHumanUser(cr, users), nil
		},
	)
	if err != nil {
		return nil, errors.Wrap(err, "cannot look up user in Zitadel")
	}

	if !found {
		return nil, nil
	}

	meta.SetExternalName(cr, u.UserID)

	return u, nil
}

// findHumanUser returns the human user in users that matches the desired state
// of cr, preferring a username match over an email match. It returns nil when
// no user matches.
func (e *external) findHumanUser(cr *v1alpha1.HumanUser, users []*zitadel.User) *zitadel.User {
	fp := cr.Spec.ForProvider

	for _, u := range users {
		if u.Human == nil {
			continue
		}

		if fp.UserName != nil && u.UserName == *fp.UserName {
			return u
		}
	}

	for _, u := range users {
		if u.Human == nil {
			continue
		}

		if u.Human.Email == fp.Email {
			return u
		}
	}

	return nil
}

// Create creates the human user in Zitadel. Verification codes returned by
// Zitadel are published to the connection secret.
//
//nolint:gocyclo // flat translation of the optional forProvider fields
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.HumanUser)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotHumanUser)
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

	in := zitadel.CreateHumanUserInput{
		OrganizationID: orgID,
		UserID:         common.Deref(fp.ID),
		UserName:       common.Deref(fp.UserName),
		Email:          fp.Email,
		Profile:        *profile(fp),
		Metadata:       metadata(fp),
	}

	in.EmailVerification = common.Value(fp.EmailVerification)
	if in.EmailVerification == "" {
		in.EmailVerification = string(v1alpha1.EmailVerificationTypeSendCode)
	}

	if fp.Phone != nil {
		in.Phone = *fp.Phone
		in.PhoneVerification = common.Value(fp.PhoneVerification)
		if in.PhoneVerification == "" {
			in.PhoneVerification = string(v1alpha1.PhoneVerificationTypeSendCode)
		}
	}

	for _, l := range fp.IDPLinks {
		in.IDPLinks = append(in.IDPLinks, zitadel.IDPLink{
			IDPID:    l.IDPID,
			UserID:   l.UserID,
			UserName: l.UserName,
		})
	}

	if fp.PasswordSecretRef != nil {
		pw, err := e.password(ctx, cr, fp.PasswordSecretRef)
		if err != nil {
			return managed.ExternalCreation{}, err
		}
		in.Password = pw
	}

	if fp.PasswordChangeRequired != nil {
		in.PasswordChangeRequired = *fp.PasswordChangeRequired
	}

	res, err := e.client.CreateHumanUser(ctx, in)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create human user in Zitadel")
	}

	meta.SetExternalName(cr, res.UserID)

	details := managed.ConnectionDetails{
		v1alpha1.ConnectionKeyUserID: []byte(res.UserID),
	}
	if res.EmailCode != "" {
		details[v1alpha1.ConnectionKeyEmailCode] = []byte(res.EmailCode)
	}
	if res.PhoneCode != "" {
		details[v1alpha1.ConnectionKeyPhoneCode] = []byte(res.PhoneCode)
	}

	return managed.ExternalCreation{ConnectionDetails: details}, nil
}

// Update reconciles the mutable attributes and the state of the user.
//
//nolint:gocyclo // flat translation of the optional forProvider fields
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.HumanUser)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotHumanUser)
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
		return managed.ExternalUpdate{}, errors.New("cannot update a user that does not exist")
	}

	fp := cr.Spec.ForProvider

	if want := common.Deref(fp.UserName); want != "" && want != current.UserName {
		if err := e.client.UpdateUsername(ctx, current.UserID, want, false); err != nil {
			return managed.ExternalUpdate{}, err
		}
	}

	if err := e.client.UpdateHumanUser(ctx, current.UserID, profile(fp)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	if current.Human != nil && current.Human.Email != fp.Email {
		verification := common.Value(fp.EmailVerification)
		if verification == "" {
			verification = string(v1alpha1.EmailVerificationTypeSendCode)
		}
		if err := e.client.UpdateHumanEmail(ctx, current.UserID, fp.Email, verification); err != nil {
			return managed.ExternalUpdate{}, err
		}
	}

	if fp.Phone != nil {
		want := *fp.Phone
		have := ""
		if current.Human != nil {
			have = current.Human.Phone
		}

		if want != have {
			verification := common.Value(fp.PhoneVerification)
			if verification == "" {
				verification = string(v1alpha1.PhoneVerificationTypeSendCode)
			}
			if err := e.client.UpdateHumanPhone(ctx, current.UserID, want, verification); err != nil {
				return managed.ExternalUpdate{}, err
			}
		}
	}

	if err := e.client.SetUserState(ctx, current.UserID, current.State, common.Value(fp.State)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete removes the user from Zitadel.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.HumanUser)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotHumanUser)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.OrganizationID) {
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
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete user from Zitadel")
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

// organizationID resolves the organization the user belongs to.
func (e *external) organizationID(ctx context.Context, cr *v1alpha1.HumanUser) (string, error) {
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

// password reads the initial password of the user from the referenced secret in
// the namespace of the managed resource.
func (e *external) password(ctx context.Context, cr *v1alpha1.HumanUser, ref *xpv1.LocalSecretKeySelector) (string, error) {
	secretRef := xpv1.SecretKeySelector{
		SecretReference: xpv1.SecretReference{Name: ref.Name, Namespace: cr.GetNamespace()},
		Key:             ref.Key,
	}

	data, err := resource.CommonCredentialExtractor(ctx, xpv1.CredentialsSourceSecret, e.kube,
		xpv1.CommonCredentialSelectors{SecretRef: &secretRef})
	if err != nil {
		return "", errors.Wrap(err, "cannot read initial password of user")
	}

	return string(data), nil
}

// connectionDetails returns the connection details of an existing user. The
// initial password and the verification codes are only available at creation
// time, so they cannot be republished.
func connectionDetails(u *zitadel.User) managed.ConnectionDetails {
	details := managed.ConnectionDetails{
		v1alpha1.ConnectionKeyUserID: []byte(u.UserID),
	}

	if u.PreferredLoginName != "" {
		details[v1alpha1.ConnectionKeyUsername] = []byte(u.PreferredLoginName)
	}

	return details
}
