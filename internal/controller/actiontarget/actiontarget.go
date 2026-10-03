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

package actiontarget

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

const errNotActionTarget = "managed resource is not an ActionTarget custom resource"

// Connection keys of an ActionTarget. The signing key is what makes the target
// verifiable from the other side, and Zitadel generates it once.
const (
	// ConnectionKeyTargetID is Zitadel's identifier of the target.
	ConnectionKeyTargetID = "targetID"

	// ConnectionKeySigningKey is the key Zitadel signs this target's payloads
	// with.
	ConnectionKeySigningKey = "signingKey"
)

// Setup adds a controller that reconciles ActionTarget managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.ActionTargetGroupKind,
		v1alpha1.ActionTargetGroupVersionKind,
		&v1alpha1.ActionTarget{},
		&v1alpha1.ActionTargetList{},
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
func (e *external) Disconnect(_ context.Context) error {
	return e.client.Close()
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ActionTarget)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotActionTarget)
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	t, err := e.client.GetActionTarget(ctx, id)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the action target from Zitadel")
	}

	updateStatus(cr, t)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate(cr, t),
		ConnectionDetails: managed.ConnectionDetails{
			ConnectionKeyTargetID:   []byte(t.ID),
			ConnectionKeySigningKey: []byte(t.SigningKey),
		},
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ActionTarget)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotActionTarget)
	}

	cr.Status.SetConditions(xpv1.Creating())

	id, err := e.client.CreateActionTarget(ctx, desired(cr))
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, id)

	// The signing key is only returned when the target is read back, so it is
	// fetched here rather than left for the next reconcile.
	t, err := e.client.GetActionTarget(ctx, id)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "the target was created but its signing key could not be read")
	}

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{
		ConnectionKeyTargetID:   []byte(t.ID),
		ConnectionKeySigningKey: []byte(t.SigningKey),
	}}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ActionTarget)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotActionTarget)
	}

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalUpdate{}, nil
	}

	if err := e.client.UpdateActionTarget(ctx, id, desired(cr)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ActionTarget)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotActionTarget)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	return managed.ExternalDelete{}, e.client.DeleteActionTarget(ctx, id)
}

// desired translates the spec into the client input, applying the documented
// defaults for every field the spec leaves unset.
func desired(cr *v1alpha1.ActionTarget) zitadel.ActionTarget {
	fp := cr.Spec.ForProvider

	return zitadel.ActionTarget{
		Name:        common.Deref(fp.Name),
		Type:        string(valueOrType(fp.Type)),
		Endpoint:    common.Deref(fp.Endpoint),
		Timeout:     valueOrTimeout(fp.Timeout, "10s"),
		PayloadType: string(valueOrPayload(fp.PayloadType)),
	}
}

// isUpToDate compares the desired target with the observed one.
//
// A field the operator left unset is not compared, so Zitadel's own defaults are
// never fought over.
func isUpToDate(cr *v1alpha1.ActionTarget, t *zitadel.ActionTarget) bool {
	fp := cr.Spec.ForProvider

	for _, f := range []struct {
		name     string
		set      bool
		want     string
		observed string
	}{
		{"name", fp.Name != nil, common.Deref(fp.Name), t.Name},
		{"type", fp.Type != nil, string(valueOrType(fp.Type)), t.Type},
		{"endpoint", fp.Endpoint != nil, common.Deref(fp.Endpoint), t.Endpoint},
		{"timeout", fp.Timeout != nil, valueOrTimeout(fp.Timeout, "10s"), t.Timeout},
		{"payloadType", fp.PayloadType != nil, string(valueOrPayload(fp.PayloadType)), t.PayloadType},
	} {
		if f.set && f.want != f.observed {
			return false
		}
	}

	return true
}

func updateStatus(cr *v1alpha1.ActionTarget, t *zitadel.ActionTarget) {
	cr.Status.AtProvider.ID = t.ID
	cr.Status.AtProvider.Name = common.StringPtr(t.Name)
	cr.Status.AtProvider.Type = common.StringPtr(t.Type)
	cr.Status.AtProvider.Endpoint = common.StringPtr(t.Endpoint)
	cr.Status.AtProvider.Timeout = common.StringPtr(t.Timeout)
	cr.Status.AtProvider.PayloadType = common.StringPtr(t.PayloadType)
	cr.Status.AtProvider.SigningKey = t.SigningKey
}

func valueOrType(t *v1alpha1.ActionTargetType) v1alpha1.ActionTargetType {
	if t == nil || *t == "" {
		return v1alpha1.ActionTargetTypeWebhook
	}

	return *t
}

func valueOrPayload(t *v1alpha1.ActionPayloadType) v1alpha1.ActionPayloadType {
	if t == nil || *t == "" {
		return v1alpha1.ActionPayloadTypeJson
	}

	return *t
}

func valueOrTimeout(t *string, fallback string) string {
	if t == nil || *t == "" {
		return fallback
	}

	return *t
}
