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

package actiontargetpublickey

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

const errNotKey = "managed resource is not an ActionTargetPublicKey custom resource"

// Setup adds a controller that reconciles ActionTargetPublicKey managed
// resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ActionTargetPublicKeyGroupKind,
		v1alpha1.ActionTargetPublicKeyGroupVersionKind,
		&v1alpha1.ActionTargetPublicKey{},
		&v1alpha1.ActionTargetPublicKeyList{},
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
func (e *external) Disconnect(_ context.Context) error {
	e.client.Release()

	return nil
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ActionTargetPublicKey)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotKey)
	}

	keyID := meta.GetExternalName(cr)
	if keyID == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	targetID, err := e.targetID(ctx, cr)
	if err != nil {
		if meta.WasDeleted(cr) {
			// Nothing resolves while an object is terminating, so report the key
			// as gone and let the finalizer go.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	k, err := e.client.GetActionTargetPublicKey(ctx, targetID, keyID)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the public key from Zitadel")
	}

	updateStatus(cr, targetID, k)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate(cr, k),
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ActionTargetPublicKey)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotKey)
	}

	cr.Status.SetConditions(xpv1.Creating())

	targetID, err := e.targetID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	keyID, err := e.client.AddActionTargetPublicKey(ctx, targetID, common.Deref(cr.Spec.ForProvider.PublicKey))
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, keyID)

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ActionTargetPublicKey)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotKey)
	}

	keyID := meta.GetExternalName(cr)
	if keyID == "" {
		return managed.ExternalUpdate{}, nil
	}

	// A key belongs to the target it was added to and to nothing else, so only
	// whether it is used for encryption can change. The public key itself
	// cannot: Zitadel ties a key to the payload encryption of the target, so
	// rotating one means adding a new key and removing the old.
	targetID, err := e.targetID(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	// Whether the key is used for encryption is the one thing about it that can
	// change, so it is the only thing an update has to do - and only when the
	// spec actually asks for something different from what Zitadel reports.
	want := valueOrActive(cr.Spec.ForProvider.Active)
	if cr.Spec.ForProvider.Active == nil || want == common.DerefBool(cr.Status.AtProvider.Active) {
		return managed.ExternalUpdate{}, nil
	}

	return managed.ExternalUpdate{}, e.client.SetActionTargetPublicKeyActive(ctx, targetID, keyID, want)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ActionTargetPublicKey)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotKey)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	keyID := meta.GetExternalName(cr)
	if keyID == "" {
		return managed.ExternalDelete{}, nil
	}

	// Deleting acts on the target the key was added to, not on whichever one the
	// reference points at now.
	targetID := common.Deref(cr.Status.AtProvider.TargetID)
	if targetID == "" {
		resolved, err := e.targetID(ctx, cr)
		switch {
		case err == nil:
			targetID = resolved

		case common.IsReferenceGone(err):
			// The target is gone, and with it the key.

		default:
			return managed.ExternalDelete{}, err
		}
	}

	if targetID == "" {
		return managed.ExternalDelete{}, nil
	}

	return managed.ExternalDelete{}, e.client.RemoveActionTargetPublicKey(ctx, targetID, keyID)
}

// targetID resolves the target the key belongs to.
func (e *external) targetID(ctx context.Context, cr *v1alpha1.ActionTargetPublicKey) (string, error) {
	fp := cr.Spec.ForProvider

	id, err := common.ResolveTargetID(ctx, e.kube, cr, fp.TargetRef, fp.TargetSelector, fp.TargetID)
	if err != nil {
		return "", common.Join(common.ErrNoUserID, err)
	}

	if id == "" {
		return "", errors.New("targetID, targetRef or targetSelector must be set")
	}

	return id, nil
}

func isUpToDate(cr *v1alpha1.ActionTargetPublicKey, k *zitadel.ActionTargetPublicKey) bool {
	if cr.Spec.ForProvider.Active == nil {
		// An operator who never mentioned whether the key should be used is not
		// asking for it to be flipped.
		return true
	}

	return valueOrActive(cr.Spec.ForProvider.Active) == k.Active
}

func updateStatus(cr *v1alpha1.ActionTargetPublicKey, targetID string, k *zitadel.ActionTargetPublicKey) {
	cr.Status.AtProvider.TargetID = common.StringPtr(targetID)
	cr.Status.AtProvider.KeyID = k.KeyID
	cr.Status.AtProvider.PublicKey = common.StringPtr(k.PublicKey)
	cr.Status.AtProvider.Active = common.BoolPtr(k.Active)
	cr.Status.AtProvider.Fingerprint = k.Fingerprint
	cr.Status.AtProvider.CreationDate = k.CreationDate
}

func valueOrActive(a *bool) bool {
	if a == nil {
		return true
	}

	return *a
}
