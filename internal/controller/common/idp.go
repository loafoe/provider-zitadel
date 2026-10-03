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

package common

import (
	"context"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Identity providers
//
// Zitadel has twenty three kinds of identity provider across two levels - one
// per provider type, instance wide and per organization - and they are the same
// resource with a different field set. The lifecycle is therefore here once:
//
//   - the provider type decides which of Zitadel's per type calls is made, since
//     there is no generic one;
//   - reading back is the real asymmetry: Zitadel reports an OIDC or a JWT
//     provider's configuration and nothing at all for any other kind, so those
//     settings are applied once and cannot be compared or changed;
//   - what can change on any kind is the name, whether users are registered
//     automatically, and whether the provider is offered on the login page.

// ManagedIDP is what an identity provider managed resource adds to
// resource.Managed so that the shared harness can drive it.
type ManagedIDP interface {
	resource.ModernManaged
	resource.Conditioned

	// SetIDPOrganization records the organization the provider belongs to, or the
	// empty string for an instance wide one.
	SetIDPOrganization(orgID string)

	// IDPOrganization returns the recorded organization.
	IDPOrganization() string
}

// IDPDriver is everything that differs between one identity provider kind and
// the next.
type IDPDriver[CR ManagedIDP] interface {
	// Kind names the managed resource, for errors and events.
	Kind() string

	// Provider names the Zitadel provider type, which is what selects the call.
	Provider() string

	// OrganizationScoped reports whether this kind belongs to an organization. An
	// instance wide one has no reference to resolve and speaks the admin API.
	OrganizationScoped() bool

	// Observable reports whether Zitadel will read this kind of provider back.
	//
	// Only an OIDC or a JWT provider is: for anything else the by-ID read answers
	// "Identity Provider Configuration doesn't exist" and the list leaves it out,
	// even though it was created. Those kinds are therefore created and removed
	// successfully but cannot be observed, so their existence is taken from the
	// identifier the provider was created with rather than from Zitadel - and no
	// drift detection is claimed for them.
	Observable() bool

	// Settings returns the settings of an OIDC or a JWT provider, which are the
	// only ones Zitadel reads back and so the only ones that can be updated in
	// place. It is nil for every other kind.
	Settings(ctx context.Context, kube client.Client, cr CR) zitadel.IDPSettings

	// Desired reads the desired provider out of the managed resource.
	//
	// The context and the cluster client are needed by the kinds that read a
	// secret, such as a client secret or a directory bind password: a credential
	// has no safe place in a custom resource, so it is read rather than written.
	Desired(ctx context.Context, kube client.Client, cr CR) zitadel.ProviderInput

	// Observation returns the status the driver writes into and compares. It is
	// the one place that knows which kind's status this is: all twenty three
	// kinds share a single observation type, so this is a field access rather
	// than a conversion.
	Observation(cr CR) *v1alpha1.IDPObservation

	// Report writes what Zitadel reported into the managed resource's status.
	Report(out *v1alpha1.IDPObservation, observed zitadel.IdentityProvider)

	// Equal reports whether the observed provider already matches the desired one.
	//
	// The desired provider is passed rather than the status, because the status
	// holds what Zitadel reported: comparing the two would compare Zitadel with
	// itself and call every provider up to date.
	Equal(want zitadel.ProviderInput, observed zitadel.IdentityProvider) bool

	// SettingsEqual reports whether the readable settings match. It is only
	// reached for a kind whose settings Zitadel reports.
	SettingsEqual(want zitadel.IDPSettings, got zitadel.IDPSettings) bool
}

// idpExternal reconciles one identity provider.
type idpExternal[CR ManagedIDP] struct {
	kube client.Client
	zc   *zitadel.Client
	d    IDPDriver[CR]
}

// Disconnect releases the underlying Zitadel client.
func (e *idpExternal[CR]) Disconnect(_ context.Context) error {
	return e.zc.Close()
}

func (e *idpExternal[CR]) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalObservation{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	orgID, err := e.scope(ctx, cr)
	if err != nil {
		if meta.WasDeleted(cr) {
			// Nothing resolves while an object is terminating, so report the
			// provider as gone and let the finalizer go.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	// Crossplane seeds the external name with the object's own name, so an
	// external name that is not a Zitadel ID means Create has not run yet. It
	// matters most for the kinds Zitadel does not read back: they have no other
	// way to be found, and taking the seeded name for a real one would mean the
	// provider was never created.
	id := meta.GetExternalName(cr)
	if !IsZitadelID(id) {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.SetIDPOrganization(orgID)

	// A kind Zitadel does not read back cannot be observed at all. Its existence
	// is taken from the identifier it was created with: Zitadel accepted the
	// create and returns nothing since, so there is nothing else to go on, and
	// claiming otherwise would mean reporting a provider as gone the moment after
	// it was made.
	if !e.d.Observable() {
		cr.SetConditions(xpv1.Available())

		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
	}

	p, err := e.zc.GetIdentityProvider(ctx, orgID, id)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	out := e.d.Observation(cr)

	// An organization wide provider is reported with its organization, which is
	// what a terminating object acts on.
	if orgID != "" {
		out.OrganizationID = &orgID
	}

	e.d.Report(out, *p)
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: e.d.Equal(e.d.Desired(ctx, e.kube, cr), *p),
	}, nil
}

func (e *idpExternal[CR]) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalCreation{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	cr.SetConditions(xpv1.Creating())

	orgID, err := e.scope(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	id, err := e.zc.CreateIdentityProvider(ctx, orgID, e.d.Provider(), e.d.Desired(ctx, e.kube, cr))
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, id)
	cr.SetIDPOrganization(orgID)

	// A provider that is asked for inactive is created and then switched off,
	// which is the only order Zitadel accepts.
	if want := e.d.Desired(ctx, e.kube, cr); want.State == "Inactive" {
		if err := e.zc.SetIdentityProviderState(ctx, orgID, id, want.State); err != nil {
			return managed.ExternalCreation{}, err
		}
	}

	return managed.ExternalCreation{}, nil
}

//nolint:gocyclo // One step per thing Zitadel can be asked to change on a provider.
func (e *idpExternal[CR]) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalUpdate{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	// Deleting acts on the organization the provider was created in, not on
	// whichever one the reference points at now.
	orgID := cr.IDPOrganization()
	if orgID == "" {
		resolved, err := e.scope(ctx, cr)
		switch {
		case err == nil:
			orgID = resolved

		case IsReferenceGone(err):
			// The organization is gone, and with it the provider.

		default:
			return managed.ExternalUpdate{}, err
		}
	}

	id := meta.GetExternalName(cr)
	want := e.d.Desired(ctx, e.kube, cr)

	// The settings Zitadel reads back are the only ones it can be asked to
	// change, and only an OIDC or a JWT provider has any.
	if settings := e.d.Settings(ctx, e.kube, cr); settings.OIDC != nil || settings.JWT != nil {
		if err := e.zc.UpdateIdentityProviderSettings(ctx, orgID, id, settings); err != nil {
			return managed.ExternalUpdate{}, err
		}
	}

	if err := e.zc.UpdateIdentityProvider(ctx, orgID, id, want); err != nil {
		return managed.ExternalUpdate{}, err
	}

	// The state is applied by its own call, and only when it is actually
	// different: Zitadel refuses to activate a provider that is already active.
	if want.State != "" {
		p, err := e.zc.GetIdentityProvider(ctx, orgID, id)
		switch {
		case err != nil:
			return managed.ExternalUpdate{}, err

		case p.State != want.State:
			if err := e.zc.SetIdentityProviderState(ctx, orgID, id, want.State); err != nil {
				return managed.ExternalUpdate{}, err
			}
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (e *idpExternal[CR]) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalDelete{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	cr.SetConditions(xpv1.Deleting())

	id := meta.GetExternalName(cr)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	orgID := cr.IDPOrganization()
	if orgID == "" {
		resolved, err := e.scope(ctx, cr)
		switch {
		case err == nil:
			orgID = resolved

		case IsReferenceGone(err):
			// The organization is gone, and with it the provider.

		default:
			return managed.ExternalDelete{}, err
		}
	}

	return managed.ExternalDelete{}, e.zc.RemoveIdentityProvider(ctx, orgID, id)
}

// scope resolves the organization an identity provider belongs to.
//
// An instance wide provider has nothing to resolve, which is what tells the
// harness to speak the admin API rather than the management one.
func (e *idpExternal[CR]) scope(ctx context.Context, cr CR) (string, error) {
	if !e.d.OrganizationScoped() {
		return "", nil
	}

	refs := idpScopeOf(cr)

	orgDefault, err := ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return "", Join(ErrNoOrganizationID, err)
	}

	if refs == nil {
		return "", errors.New(ErrNoOrganizationID.Error())
	}

	orgID, err := ResolveOrganizationID(ctx, e.kube, cr, refs.GetOrganizationRef(),
		refs.GetOrganizationSelector(), refs.GetOrganizationID(), orgDefault,
		CurrentIfDeleting(meta.WasDeleted(cr), refs.GetObservedOrganization()))
	if err != nil {
		return "", Join(ErrNoOrganizationID, err)
	}

	if orgID == "" {
		return "", errors.New(ErrNoOrganizationID.Error())
	}

	return orgID, nil
}

// IDPScoped is how a scoped identity provider exposes the reference its scope is
// resolved from, and the organization it was last read from.
type IDPScoped interface {
	GetOrganizationRef() *xpv1.Reference
	GetOrganizationSelector() *xpv1.Selector
	GetOrganizationID() *string
	GetObservedOrganization() *string
}

// idpScopeOf reads the scope of an identity provider that has one. An instance
// wide provider does not implement it, and has no organization to resolve.
func idpScopeOf(cr ManagedIDP) IDPScoped {
	if s, ok := cr.(IDPScoped); ok {
		return s
	}

	return nil
}

// SetupIDPController wires up the managed resource reconciler for one identity
// provider kind, driven by d.
func SetupIDPController[CR ManagedIDP](mgr ctrl.Manager, o controller.Options,
	groupKind string, gvk schema.GroupVersionKind, obj CR, list resource.ManagedList,
	d IDPDriver[CR],
) error {
	return SetupManagedResourceController(mgr, o, groupKind, gvk, obj, list,
		func(_ context.Context, kube client.Client, mg resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
			return &idpExternal[CR]{kube: kube, zc: zc, d: d}, nil
		})
}
