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

	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Reusing a namespaced controller for a cluster scoped kind
//
// Nineteen of the kinds manage state belonging to a whole Zitadel instance, and
// each is a singleton: there is one set of instance feature flags, one login
// policy default, one SMTP provider. Namespaced, that is a footgun - two
// namespaces can each hold one and they fight over the same settings, which the
// second writer wins.
//
// The cluster scoped form removes the conflict by making it unrepresentable, but
// it must not become a second implementation. Every one of these kinds already
// has a controller that does the real work, and it is correct. So rather than a
// second copy of it, the cluster scoped resource is handed to the same one as
// the kind it mirrors, and what comes back is copied onto it.
//
// The kinds are the same object with the same fields and the same external
// resource, so nothing is translated: the spec, the observation and the status
// are one set of Go types, and only the metadata moves.

// clusterKind is a cluster scoped resource that can be handed to the controller
// that reconciles the namespaced kind it mirrors.
type clusterKind[C resource.ModernManaged] interface {
	resource.ModernManaged

	clusterAdapter[C]
}

// clusterAdapter is what lets an external client be reused across the two kinds.
type clusterAdapter[C any] interface {
	// AsNamespaced returns the resource as the kind the client already knows.
	AsNamespaced() C

	// AdoptNamespaced copies back what the client recorded.
	AdoptNamespaced(from C)
}

// reuseExternal reconciles a cluster scoped kind with the external client that
// already reconciles the namespaced one.
//
// Every call is the same two steps: hand the client the kind it expects, then
// copy back the external name, the finalizers and the status it produced.
type reuseExternal[C resource.ModernManaged] struct {
	inner managed.ExternalClient
}

func (e *reuseExternal[C]) Disconnect(ctx context.Context) error { return e.inner.Disconnect(ctx) }

func (e *reuseExternal[C]) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, inner, ok := e.split(mg)
	if !ok {
		return managed.ExternalObservation{}, errNotAClusterKind
	}

	o, err := e.inner.Observe(ctx, inner)
	if err != nil {
		// The observation is still worth keeping: an error from the inner client
		// may have recorded something, and adopting it costs nothing.
		cr.AdoptNamespaced(inner)

		return o, err
	}

	cr.AdoptNamespaced(inner)

	return o, nil
}

func (e *reuseExternal[C]) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, inner, ok := e.split(mg)
	if !ok {
		return managed.ExternalCreation{}, errNotAClusterKind
	}

	d, err := e.inner.Create(ctx, inner)
	cr.AdoptNamespaced(inner)

	return d, err
}

func (e *reuseExternal[C]) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, inner, ok := e.split(mg)
	if !ok {
		return managed.ExternalUpdate{}, errNotAClusterKind
	}

	d, err := e.inner.Update(ctx, inner)
	cr.AdoptNamespaced(inner)

	return d, err
}

func (e *reuseExternal[C]) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, inner, ok := e.split(mg)
	if !ok {
		return managed.ExternalDelete{}, errNotAClusterKind
	}

	d, err := e.inner.Delete(ctx, inner)
	cr.AdoptNamespaced(inner)

	return d, err
}

// split is the one place the conversion happens: it hands back both the cluster
// scoped resource to write back onto and the namespaced one the client wants.
func (e *reuseExternal[C]) split(mg resource.Managed) (clusterAdapter[C], C, bool) {
	cr, ok := mg.(clusterKind[C])
	if !ok {
		return nil, *new(C), false
	}

	return cr, cr.AsNamespaced(), true
}

// ReusingForCluster wraps the external client of a namespaced kind so that it
// reconciles a cluster scoped one.
//
// The returned function has the shape SetupManagedResourceController expects, so
// a cluster scoped controller is a single call naming the kind, the group kind
// and the client to reuse.
func ReusingForCluster[C resource.ModernManaged](newExternal NewExternalClientFn) NewExternalClientFn {
	return func(ctx context.Context, kube client.Client, mg resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
		// The client is built from the namespaced form, because that is the kind
		// it reconciles. Handing it the cluster scoped object instead would fail
		// its very first type assertion.
		cr, ok := mg.(clusterKind[C])
		if !ok {
			return nil, errNotAClusterKind
		}

		ec, err := newExternal(ctx, kube, cr.AsNamespaced(), zc)
		if err != nil {
			return nil, err
		}

		return &reuseExternal[C]{inner: ec}, nil
	}
}

// errNotAClusterKind is returned when a controller for a cluster scoped kind is
// handed something that is not one.
//
// It can only happen if a kind is registered with the wrong controller, and it is
// reported rather than panicked so that it surfaces as a reconcile error naming
// the problem.
var errNotAClusterKind = errors.New("managed resource is not a cluster scoped kind")
