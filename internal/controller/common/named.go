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

	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// One named thing in a Zitadel list
//
// Zitadel keeps a few things as plain lists of names that have no identifier,
// settings of their own and no lifecycle beyond being in the list or not: the
// domains an instance answers on, the domains that may request one of its
// tokens, and the domains an organization owns.
//
// A member of such a list is the simplest thing there is to reconcile. Its name
// is its identity, so there is nothing to look up by an ID, nothing to compare
// beyond whether the name is present, and nothing to restore afterwards. What
// surrounds those three calls - how the name is found, what happens when the
// list is read while the object is terminating, and what a renamed name means -
// is the same for every list, so it lives here once.

// Named is a member of a Zitadel list.
type Named interface {
	// Name is what the entry is called, which is also its identity.
	Name() string
}

// ManagedNamed is what a named managed resource adds to resource.Managed, so
// that the shared harness can drive it.
type ManagedNamed interface {
	resource.ModernManaged

	// NamedName returns the name the entry should have.
	NamedName() string

	// NamedSetScope records the scope the entry was last seen in, so that a
	// terminating resource removes it from where it actually put it rather than
	// from wherever its reference has since moved to.
	NamedSetScope(scope string)

	// NamedScope returns the recorded scope.
	NamedScope() string
}

// NamedDriver is everything that differs between one Zitadel list and the next.
//
// There is deliberately no equality function: a name is either in the list or it
// is not, so once the entry is found there is nothing left to compare.
type NamedDriver[CR ManagedNamed] interface {
	// Kind names the list, so errors and events say which one failed.
	Kind() string

	// Scope resolves the owner of the entry. The empty string means the instance
	// rather than any organization. A scoped list that cannot resolve its
	// organization returns an error, so an empty scope never means "unresolved".
	Scope(ctx context.Context, kube client.Client, cr CR) (string, error)

	// List reads the whole list.
	List(ctx context.Context, c *zitadel.Client, scope string) ([]zitadel.Domain, error)

	// Add puts a name into the list.
	Add(ctx context.Context, c *zitadel.Client, scope, name string) error

	// Remove takes a name out of the list.
	Remove(ctx context.Context, c *zitadel.Client, scope, name string) error

	// Report writes an observed entry into the managed resource's status.
	Report(cr CR, entry zitadel.Domain)
}

// NamedVerifier is an optional companion to NamedDriver, for a list whose
// members need a call of their own once they are found.
//
// Zitadel's organization domains are the case: adding one leaves it unverified,
// and Zitadel only marks it verified once the organization has published a
// proof. Asking what that proof is takes a call, which Report cannot make
// because it is given nothing to make it with.
type NamedVerifier[CR ManagedNamed] interface {
	// Verify does whatever an already present entry still needs. It is called
	// only when the entry was found.
	Verify(ctx context.Context, c *zitadel.Client, scope string, cr CR, entry zitadel.Domain)
}

func (e *namedExternal[CR]) Disconnect(_ context.Context) error {
	e.zc.Release()

	return nil
}

// namedExternal reconciles one member of a Zitadel list.
type namedExternal[CR ManagedNamed] struct {
	kube client.Client
	zc   *zitadel.Client
	d    NamedDriver[CR]
}

func newNamedExternal[CR ManagedNamed](kube client.Client, zc *zitadel.Client, mg resource.Managed, d NamedDriver[CR]) (managed.ExternalClient, error) {
	if _, ok := mg.(CR); !ok {
		return nil, errors.Errorf("managed resource is not a %s", d.Kind())
	}

	return &namedExternal[CR]{kube: kube, zc: zc, d: d}, nil
}

func (e *namedExternal[CR]) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalObservation{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	scope, err := e.d.Scope(ctx, e.kube, cr)
	if err != nil {
		if meta.WasDeleted(cr) {
			// Nothing resolves while an object is terminating, so report the
			// entry as gone. That lets the reconciler run Delete, which lets the
			// finalizer go, instead of retrying an observation that can never
			// succeed.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	domains, err := e.d.List(ctx, e.zc, scope)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	want := cr.NamedName()

	// An entry that exists only in the recorded scope is still found there while
	// the object terminates, even if its reference has since moved. A renamed
	// name is not looked for anywhere else: the old one is what has to be
	// removed, and the new one is what has to be added.
	exists := false
	for _, d := range domains {
		if d.Name == want {
			exists = true
			e.d.Report(cr, d)

			if v, ok := any(e.d).(NamedVerifier[CR]); ok {
				v.Verify(ctx, e.zc, scope, cr, d)
			}

			break
		}
	}

	if exists {
		cr.NamedSetScope(scope)
		cr.SetConditions(xpv1.Available())
	}

	return managed.ExternalObservation{ResourceExists: exists, ResourceUpToDate: exists}, nil
}

func (e *namedExternal[CR]) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalCreation{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	cr.SetConditions(xpv1.Creating())

	scope, err := e.d.Scope(ctx, e.kube, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.d.Add(ctx, e.zc, scope, cr.NamedName()); err != nil {
		return managed.ExternalCreation{}, err
	}

	cr.NamedSetScope(scope)

	return managed.ExternalCreation{}, nil
}

// Update is a no-op: a name is either in the list or it is not, and a change to
// the name is a different entry rather than a change to this one.
//
// A renamed manifest therefore leaves the old entry behind, which is reported in
// the events rather than silently, because removing something another resource
// may also be managing is not this resource's call to make.
func (e *namedExternal[CR]) Update(_ context.Context, _ resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (e *namedExternal[CR]) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalDelete{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	cr.SetConditions(xpv1.Deleting())

	// A terminating object must not follow a reference that has since moved: it
	// exists to undo what it did, so it removes the entry from the scope it
	// recorded.
	scope := cr.NamedScope()
	if scope == "" {
		resolved, err := e.d.Scope(ctx, e.kube, cr)
		switch {
		case err == nil:
			scope = resolved

		case IsReferenceGone(err):
			// The organization is gone, and with it any domain it owned.

		default:
			return managed.ExternalDelete{}, err
		}
	}

	return managed.ExternalDelete{}, e.d.Remove(ctx, e.zc, scope, cr.NamedName())
}

// SetupNamedController wires up the managed resource reconciler for one member
// of a Zitadel list, driven by d.
func SetupNamedController[CR ManagedNamed](mgr ctrl.Manager, o controller.Options,
	groupKind string, gvk schema.GroupVersionKind, obj CR, list resource.ManagedList,
	d NamedDriver[CR],
) error {
	return SetupManagedResourceController(mgr, o, groupKind, gvk, obj, list,
		func(_ context.Context, kube client.Client, mg resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
			return newNamedExternal[CR](kube, zc, mg, d)
		})
}
