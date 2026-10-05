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

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apisv1alpha1 "github.com/loafoe/provider-zitadel/apis/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// NewExternalClientFn builds the external client of a managed resource from an
// authenticated Zitadel client.
type NewExternalClientFn func(ctx context.Context, kube client.Client, mg resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error)

// connector resolves the ProviderConfig of a managed resource and builds the
// external client used to reconcile it.
type connector struct {
	kube  client.Client
	usage *resource.ProviderConfigUsageTracker

	// clusterUsage tracks usage of a ClusterProviderConfig, which is what a
	// cluster scoped resource references. A namespaced resource that points at
	// one is tracked here too, so that deleting the configuration can find
	// everything using it.
	clusterUsage *resource.ProviderConfigUsageTracker

	// cache shares one Zitadel client between every resource that authenticates
	// with the same configuration. Without it each reconcile would dial a fresh
	// connection and a fresh token, and throw them away again; see ClientCache.
	cache *ClientCache

	newExternal NewExternalClientFn
}

// Connect tracks the ProviderConfig usage of mg, builds an authenticated
// Zitadel client and hands both to the controller specific external client
// factory.
//
// The client is shared and outlives this call, so the external client is given
// one it must Release rather than Close. See ClientCache.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	m, ok := mg.(resource.ModernManaged)
	if !ok {
		return nil, errors.New("managed resource does not support provider config references")
	}

	if err := c.usageTracker(m).Track(ctx, m); err != nil {
		return nil, errors.Wrap(err, "cannot track ProviderConfig usage")
	}

	zc, err := c.cache.ClientFor(ctx, c.kube, m)
	if err != nil {
		return nil, err
	}

	ec, err := c.newExternal(ctx, c.kube, m, zc)
	if err != nil {
		// The client is the cache's, so releasing here is a no-op for it. It
		// matters only for a client built outside the cache.
		zc.Release()

		return nil, zitadel.WrapError(err)
	}

	return &externalClient{ExternalClient: ec}, nil
}

// usageTracker picks the tracker that matches what the resource references.
//
// The two track different objects: a resource pointing at a ClusterProviderConfig
// has to be recorded as a ClusterProviderConfigUsage, because a namespaced usage
// cannot be created for a cluster scoped resource and would not be found when the
// configuration is deleted.
func (c *connector) usageTracker(m resource.ModernManaged) *resource.ProviderConfigUsageTracker {
	ref := m.GetProviderConfigReference()
	if ref != nil && ref.Kind == "ClusterProviderConfig" {
		return c.clusterUsage
	}

	return c.usage
}

// externalClient decorates the errors of an ExternalClient. It is the single
// place where Zitadel API errors are turned into actionable messages, so that
// every controller reports them consistently.
type externalClient struct {
	managed.ExternalClient
}

func (c *externalClient) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	o, err := c.ExternalClient.Observe(ctx, mg)
	return o, zitadel.WrapError(err)
}

func (c *externalClient) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	o, err := c.ExternalClient.Create(ctx, mg)
	return o, zitadel.WrapError(err)
}

func (c *externalClient) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	o, err := c.ExternalClient.Update(ctx, mg)
	return o, zitadel.WrapError(err)
}

func (c *externalClient) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	o, err := c.ExternalClient.Delete(ctx, mg)
	return o, zitadel.WrapError(err)
}

// SetupManagedResourceController wires up the Crossplane managed resource
// reconciler for a single kind, applying the feature flags and metric options
// of the supplied controller options.
func SetupManagedResourceController(mgr ctrl.Manager, o controller.Options, groupKind string, gvk schema.GroupVersionKind, obj client.Object, list resource.ManagedList, newExternal NewExternalClientFn) error {
	name := managed.ControllerName(groupKind)

	// One cache for the whole provider, so that a resource, the organization it
	// belongs to and the project that organization holds share a connection
	// rather than opening one each. The manager owns its lifetime: the cache
	// registers a shutdown hook once, however many kinds ask for it.
	cache := DefaultClientCache()
	if err := cache.Register(mgr); err != nil {
		return errors.Wrap(err, "cannot register the Zitadel client cache")
	}

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			clusterUsage: resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ClusterProviderConfigUsage{}),
			cache:        cache,
			newExternal:  newExternal,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}

	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		recorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, list, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(recorder); err != nil {
			return errors.Wrap(err, "cannot register managed resource state metrics recorder")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(gvk), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(obj).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// NeverCreated reports whether a managed resource's external identity was never
// established, which means Create never got far enough to record it.
//
// Deletion must not depend on resolving the resource's references. Two things
// make that necessary:
//
//   - While an object is terminating, the crossplane reference resolver
//     deliberately refuses to re-resolve and returns the caller's current value
//     instead. A resource whose status has nothing to offer therefore resolves to
//     nothing while it is being deleted.
//   - A resource whose very first Create failed has no identity at all.
//
// In both cases there is nothing in Zitadel to detach from, so Delete returns
// success and lets the finalizer go, rather than blocking the object forever.
func NeverCreated(ids ...*string) bool {
	for _, id := range ids {
		if id == nil || *id == "" {
			return true
		}
	}

	return false
}
