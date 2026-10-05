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
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Why a cache?
//
// A Zitadel client owns a gRPC connection, and a connection to an
// authentication service is not free to establish: there is a TLS handshake, and
// with a service account key a token exchange with the issuer on top. Before
// this cache a client was built and torn down inside Connect, so every
// reconcile of every managed resource paid that cost again - and paid it twelve
// times over, because the client used to dial one connection per Zitadel
// service it wrapped, whether or not the reconcile touched it.
//
// The numbers matter because a provider is polled continuously. With the
// default poll interval and a hundred managed resources, "reconnect everything
// every minute" is the steady state rather than an occasional event, and it is
// the sort of load a ZITADEL instance, or the identity provider in front of it,
// feels long before a human does.
//
// So a client is kept for as long as the configuration it was built from is
// unchanged, and handed to every reconcile that needs the same one. Two things
// fall out of that:
//
//   - Resources sharing a ProviderConfig share a client, and so share a
//     connection, a token source and a set of lazily created per organization
//     management clients.
//   - The cost of connecting is paid when the configuration appears, not once
//     per resource and not once per minute.
//
// A client is never handed out for a configuration it was not built from.
// zitadel.Config.Key hashes everything that affects the client, including both
// credentials, so a rotated secret produces a different key and therefore a new
// client rather than a stale one that keeps authenticating with the old token.

// Defaults for a cache. Both bounds exist so that a provider which sees many
// distinct configurations - or whose credentials are rotated often - releases
// connections it no longer needs instead of accumulating them.
const (
	// defaultClientCacheMax is the most clients held at once. A cluster has a
	// handful of ProviderConfigs in practice, so this is generous; it exists to
	// bound a pathological case, not to be reached in normal operation.
	defaultClientCacheMax = 32

	// defaultClientCacheTTL is how long an unused client is kept. Anything still
	// in use is touched on every reconcile, so this only ever retires clients
	// belonging to a configuration that has gone away.
	defaultClientCacheTTL = 30 * time.Minute
)

// clientBuilder builds a client for a configuration. It is a field so that a
// test can supply a client that does not need a reachable ZITADEL.
type clientBuilder func(ctx context.Context, cfg zitadel.Config) (*zitadel.Client, error)

// cacheEntry is one cached client and when it was last handed out.
type cacheEntry struct {
	client   *zitadel.Client
	lastUsed time.Time
}

// ClientCache hands out Zitadel clients, sharing one between every caller that
// needs the same configuration. It is safe for concurrent use.
//
// A Client is not safe for concurrent use across all of its surfaces, but gRPC
// connections and client stubs are, and the clients it caches are only ever
// closed by the cache itself. Callers obtain a client with Get and give up
// their claim on it with Release; they must not Close it.
type ClientCache struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry

	build clientBuilder
	now   func() time.Time
	ttl   time.Duration
	max   int

	// registered ensures the shutdown hook is added to the manager once. Every
	// controller asks for the cache, and adding the same runnable once per kind
	// would start one watcher per kind for a job there is only one of.
	registered sync.Once
}

// NewClientCache returns a cache with the default bounds.
func NewClientCache() *ClientCache {
	return &ClientCache{
		entries: map[string]*cacheEntry{},
		build:   zitadel.NewClient,
		now:     time.Now,
		ttl:     defaultClientCacheTTL,
		max:     defaultClientCacheMax,
	}
}

// Get returns the client for cfg, building it if this is the first time it has
// been asked for.
//
// The returned client is shared: the caller must Release it rather than Close
// it, and must not assume it is the only holder.
func (c *ClientCache) Get(ctx context.Context, cfg zitadel.Config) (*zitadel.Client, error) {
	key := cfg.Key()

	// The lock is held across the build on purpose. A dial is lazy and cheap,
	// and holding the lock means that N controllers noticing the same new
	// configuration at the same moment produce one client rather than N, each of
	// which would then have to be closed again.
	c.mu.Lock()
	defer c.mu.Unlock()

	// The lookup comes before the prune deliberately. Pruning first would drop
	// a client that this very call is about to use, so a provider whose poll
	// interval is longer than the idle window would rebuild and redial its
	// client on every reconcile - the churn this cache exists to remove.
	if e, ok := c.entries[key]; ok {
		e.lastUsed = c.now()

		return e.client, nil
	}

	c.pruneLocked()

	zc, err := c.build(ctx, cfg)
	if err != nil {
		return nil, err
	}

	zc.SetShared()

	c.entries[key] = &cacheEntry{client: zc, lastUsed: c.now()}

	return zc, nil
}

// ClientFor resolves the ProviderConfig of mg and returns the shared client for
// it.
func (c *ClientCache) ClientFor(ctx context.Context, kube client.Client, mg resource.ModernManaged) (*zitadel.Client, error) {
	cfg, err := ConfigForProviderConfig(ctx, kube, mg)
	if err != nil {
		return nil, err
	}

	return c.Get(ctx, cfg)
}

// pruneLocked retires clients that are no longer wanted, before a new entry is
// added. It must be called with the lock held.
//
// The caller has already looked up the configuration it wants, so anything still
// in use has just been touched and cannot be retired here.
func (c *ClientCache) pruneLocked() {
	now := c.now()

	for key, e := range c.entries {
		if now.Sub(e.lastUsed) > c.ttl {
			c.dropLocked(key, e)
		}
	}

	// An idle entry is the one least likely to be needed again, so it is the one
	// to make room for by.
	for len(c.entries) >= c.max {
		victim := c.leastRecentlyUsedLocked()
		if victim == nil {
			return
		}

		c.dropLocked(victim.key, victim.entry)
	}
}

// cacheKeyed pairs an entry with its key, for eviction.
type cacheKeyed struct {
	key   string
	entry *cacheEntry
}

func (c *ClientCache) leastRecentlyUsedLocked() *cacheKeyed {
	oldest := make([]cacheKeyed, 0, len(c.entries))
	for key, e := range c.entries {
		oldest = append(oldest, cacheKeyed{key: key, entry: e})
	}

	if len(oldest) == 0 {
		return nil
	}

	sort.Slice(oldest, func(i, j int) bool {
		return oldest[i].entry.lastUsed.Before(oldest[j].entry.lastUsed)
	})

	return &oldest[0]
}

// dropLocked closes and removes an entry. It must be called with the lock held.
func (c *ClientCache) dropLocked(key string, e *cacheEntry) {
	delete(c.entries, key)
	_ = e.client.Close()
}

// Evict retires the client for cfg, if one is cached. The next caller builds a
// fresh one. It exists for the case where a client is known to be unusable -
// for instance because the instance moved - and the cache cannot tell.
func (c *ClientCache) Evict(cfg zitadel.Config) {
	key := cfg.Key()

	c.mu.Lock()
	defer c.mu.Unlock()

	if e, ok := c.entries[key]; ok {
		c.dropLocked(key, e)
	}
}

// Len reports how many clients are currently cached.
func (c *ClientCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return len(c.entries)
}

// Close releases every cached client. It is called when the provider shuts
// down.
func (c *ClientCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, e := range c.entries {
		c.dropLocked(key, e)
	}

	c.entries = map[string]*cacheEntry{}

	return nil
}

// Runnable adapts a ClientCache to the manager's lifecycle, so that the
// connections it holds are released when the provider stops.
type cacheCloser struct {
	cache *ClientCache
}

func (c cacheCloser) Start(ctx context.Context) error {
	<-ctx.Done()

	return c.cache.Close()
}

// NeedLeaderElection reports whether this has to run on the elected leader.
// Closing connections is safe on whichever instance is shutting down, and the
// leader changes on failover, so it must not wait for a lease.
func (c cacheCloser) NeedLeaderElection() bool { return false }

// Runnables returns the manager runnable that releases a cache on shutdown.
func (c *ClientCache) Runnable() interface {
	Start(context.Context) error
	NeedLeaderElection() bool
} {
	return cacheCloser{cache: c}
}

// Register adds the cache's shutdown hook to a manager, at most once.
//
// Every controller in the provider shares one cache and every one of them needs
// its connections released when the provider stops, so the registration is
// idempotent rather than something each caller has to get right.
func (c *ClientCache) Register(mgr ctrl.Manager) error {
	var err error

	c.registered.Do(func() { err = mgr.Add(c.Runnable()) })

	return err
}

// defaultCache is the cache every controller in this process shares. There is
// one manager per provider process, so there is one set of credentials worth
// caching, and sharing it is what lets a resource and its parent share a
// connection.
var defaultCache = NewClientCache()

// DefaultClientCache returns the process wide client cache.
func DefaultClientCache() *ClientCache { return defaultCache }

// ConfigForProviderConfig reads the ProviderConfig referenced by mg and returns
// the client configuration it describes. It is the single place that decides
// which of the two configuration kinds a resource refers to, so every caller
// resolves one the same way.
func ConfigForProviderConfig(ctx context.Context, kube client.Client, mg resource.ModernManaged) (zitadel.Config, error) {
	if mg.GetProviderConfigReference() == nil {
		return zitadel.Config{}, errors.New("no provider config reference set")
	}

	ref := mg.GetProviderConfigReference()
	res, err := resolveProviderConfig(ctx, kube, mg, ref)
	if err != nil {
		return zitadel.Config{}, err
	}

	creds, err := res.creds(ctx, kube)
	if err != nil {
		return zitadel.Config{}, err
	}

	settings := res.settings
	cfg := zitadel.Config{
		URL:                   settings.URL,
		Credentials:           creds,
		Insecure:              derefBool(settings.Insecure),
		InsecureSkipTLSVerify: derefBool(settings.InsecureSkipTLSVerify),
	}

	if settings.OrganizationID != nil {
		cfg.OrganizationID = *settings.OrganizationID
	}

	return cfg, nil
}
