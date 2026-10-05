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
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Values the cases share, named so that a case reads as a statement about
// behaviour rather than about a repeated literal.
const (
	testToken = "token"
)

// Test values, named so that a case reads as a statement about behaviour
// rather than about a repeated string.
const (
	instanceURL = "https://zitadel.example.com"
)

// newTestCache returns a cache whose clients never dial, plus the slice of
// clients it has built.
//
// The real builder is replaced because a test must not need a reachable ZITADEL.
// The clients it hands out are marked shared exactly as the real path does, so
// the tests exercise the same contract, and Closed reports whether the cache
// released them - which is the thing being asserted.
func newTestCache(t *testing.T) (*ClientCache, *[]*zitadel.Client) {
	t.Helper()

	cache := NewClientCache()
	built := &[]*zitadel.Client{}

	cache.now = time.Now
	cache.build = func(context.Context, zitadel.Config) (*zitadel.Client, error) {
		zc := &zitadel.Client{}
		zc.SetShared()

		*built = append(*built, zc)

		return zc, nil
	}

	return cache, built
}

// closedCount reports how many of the clients the cache built have been
// released.
func closedCount(clients []*zitadel.Client) int {
	var n int

	for _, c := range clients {
		if c.Closed() {
			n++
		}
	}

	return n
}

// TestClientCacheShares checks the reason the cache exists: two callers with the
// same configuration get one client, and therefore one connection, between them.
func TestClientCacheShares(t *testing.T) {
	cache, built := newTestCache(t)

	cfg := zitadel.Config{
		URL:         instanceURL,
		Credentials: zitadel.Credentials{Token: testToken},
	}

	first, err := cache.Get(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Get(...): unexpected error: %v", err)
	}

	second, err := cache.Get(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Get(...): unexpected error: %v", err)
	}

	if first != second {
		t.Error("two callers with the same configuration got different clients, so each dialed its own connection")
	}

	if len(*built) != 1 {
		t.Errorf("want one client built, got %d", len(*built))
	}

	if cache.Len() != 1 {
		t.Errorf("Len(): want 1, got %d", cache.Len())
	}
}

// TestClientCacheDistinguishes checks that a client is never handed to a
// caller that needs a different one. In particular a rotated secret has to
// produce a new client, because reusing the old one would keep authenticating
// with a credential the user has already replaced.
func TestClientCacheDistinguishes(t *testing.T) {
	cache, built := newTestCache(t)

	base := zitadel.Config{
		URL:         instanceURL,
		Credentials: zitadel.Credentials{Token: testToken},
	}

	cases := map[string]struct {
		reason string
		mutate func(*zitadel.Config)
	}{
		"RotatedToken": {
			reason: "A rotated token gets a new client",
			mutate: func(c *zitadel.Config) { c.Credentials.Token = "rotated" },
		},
		"RotatedKey": {
			reason: "A rotated service account key gets a new client",
			mutate: func(c *zitadel.Config) {
				c.Credentials = zitadel.Credentials{ServiceAccountKey: []byte(`{"keyId":2}`)}
			},
		},
		"OtherInstance": {
			reason: "Another instance gets a new client",
			mutate: func(c *zitadel.Config) { c.URL = "https://other.example.com" },
		},
		"OtherOrganization": {
			reason: "Another default organization gets a new client",
			mutate: func(c *zitadel.Config) { c.OrganizationID = "org-2" },
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			original, err := cache.Get(context.Background(), base)
			if err != nil {
				t.Fatalf("Get(...): unexpected error: %v", err)
			}

			before := len(*built)

			mutated := base
			tc.mutate(&mutated)

			got, err := cache.Get(context.Background(), mutated)
			if err != nil {
				t.Fatalf("Get(...): unexpected error: %v", err)
			}

			if got == original {
				t.Errorf("\n%s\nGet(...): want a new client, got the one built for the previous configuration", tc.reason)
			}

			if len(*built) != before+1 {
				t.Errorf("\n%s\nwant one client built, got %d", tc.reason, len(*built)-before)
			}
		})
	}
}

// TestClientCacheBuildFailure checks that a configuration that cannot be built
// is not remembered, so the next reconcile retries rather than being told the
// client does not exist.
func TestClientCacheBuildFailure(t *testing.T) {
	cache := NewClientCache()
	cache.now = time.Now

	wantErr := errors.New("cannot reach the instance")
	cache.build = func(context.Context, zitadel.Config) (*zitadel.Client, error) {
		return nil, wantErr
	}

	cfg := zitadel.Config{
		URL:         instanceURL,
		Credentials: zitadel.Credentials{Token: testToken},
	}

	if _, err := cache.Get(context.Background(), cfg); !errors.Is(err, wantErr) {
		t.Fatalf("Get(...): want %v, got %v", wantErr, err)
	}

	if cache.Len() != 0 {
		t.Errorf("a failed build was cached, so the next reconcile would not retry: Len() = %d", cache.Len())
	}
}

// TestClientCachePrunesIdle checks that a client whose configuration has gone
// away is eventually released, so a provider that sees many configurations over
// time does not accumulate connections.
//
// It also pins the ordering that makes the cache worth having: a client that is
// still being asked for survives however far the clock has moved on its TTL,
// because the lookup happens before anything is retired.
func TestClientCachePrunesIdle(t *testing.T) {
	cache, built := newTestCache(t)

	now := time.Now()
	cache.now = func() time.Time { return now }

	idle := zitadel.Config{
		URL:         "https://idle.example.com",
		Credentials: zitadel.Credentials{Token: testToken},
	}
	busy := zitadel.Config{
		URL:         "https://busy.example.com",
		Credentials: zitadel.Credentials{Token: testToken},
	}
	fresh := zitadel.Config{
		URL:         "https://fresh.example.com",
		Credentials: zitadel.Credentials{Token: testToken},
	}

	for _, cfg := range []zitadel.Config{idle, busy} {
		if _, err := cache.Get(context.Background(), cfg); err != nil {
			t.Fatalf("Get(...): unexpected error: %v", err)
		}
	}

	// Well past the idle window. A provider polled this slowly would rebuild its
	// client every reconcile if the prune ran before the lookup.
	now = now.Add(defaultClientCacheTTL + time.Minute)

	first, err := cache.Get(context.Background(), busy)
	if err != nil {
		t.Fatalf("Get(...): unexpected error: %v", err)
	}

	if len(*built) != 2 {
		t.Errorf("want the client still in use to be reused, got %d clients built", len(*built))
	}

	if first.Closed() {
		t.Fatal("a client that is still in use was closed")
	}

	// A miss is what triggers the prune, and it must retire the configuration
	// nobody has asked for while leaving the one just touched alone.
	if _, err := cache.Get(context.Background(), fresh); err != nil {
		t.Fatalf("Get(...): unexpected error: %v", err)
	}

	if cache.Len() != 2 {
		t.Errorf("want the idle client retired, got %d clients", cache.Len())
	}

	if got := closedCount(*built); got != 1 {
		t.Errorf("want only the idle client closed, got %d of %d closed", got, len(*built))
	}

	if first.Closed() {
		t.Error("a client that was just used was closed by the prune")
	}
}

// TestClientCacheBounds checks the hard cap. A provider configured with many
// ProviderConfigs, or one whose credentials are rotated often, must not hold an
// unbounded number of connections.
func TestClientCacheBounds(t *testing.T) {
	cache, built := newTestCache(t)
	cache.max = 3

	for i := range 10 {
		cfg := zitadel.Config{
			URL:         instanceURL,
			Credentials: zitadel.Credentials{Token: string(rune('a' + i))},
		}

		if _, err := cache.Get(context.Background(), cfg); err != nil {
			t.Fatalf("Get(...): unexpected error: %v", err)
		}
	}

	if cache.Len() > cache.max {
		t.Errorf("want at most %d clients, got %d", cache.max, cache.Len())
	}

	// Every client the cache dropped has to have been closed, or the cap only
	// bounds the map and not the connections.
	if want := len(*built) - cache.Len(); closedCount(*built) != want {
		t.Errorf("want %d dropped clients closed, got %d", want, closedCount(*built))
	}
}

// TestClientCacheEvict checks the explicit removal path.
func TestClientCacheEvict(t *testing.T) {
	cache, built := newTestCache(t)

	cfg := zitadel.Config{
		URL:         instanceURL,
		Credentials: zitadel.Credentials{Token: testToken},
	}

	if _, err := cache.Get(context.Background(), cfg); err != nil {
		t.Fatalf("Get(...): unexpected error: %v", err)
	}

	cache.Evict(cfg)

	if cache.Len() != 0 {
		t.Errorf("Len(): want 0 after evicting, got %d", cache.Len())
	}

	if !(*built)[0].Closed() {
		t.Error("the evicted client's connection was not closed")
	}

	// Evicting something that is not cached is not an error: a reconcile racing
	// a prune must not fail because of it.
	cache.Evict(cfg)

	if _, err := cache.Get(context.Background(), cfg); err != nil {
		t.Fatalf("Get(...): unexpected error: %v", err)
	}
}

// TestClientCacheClose checks that shutting the cache down releases everything.
func TestClientCacheClose(t *testing.T) {
	cache, built := newTestCache(t)

	for i := range 3 {
		cfg := zitadel.Config{
			URL:         instanceURL,
			Credentials: zitadel.Credentials{Token: string(rune('a' + i))},
		}

		if _, err := cache.Get(context.Background(), cfg); err != nil {
			t.Fatalf("Get(...): unexpected error: %v", err)
		}
	}

	if err := cache.Close(); err != nil {
		t.Errorf("Close(): unexpected error: %v", err)
	}

	if cache.Len() != 0 {
		t.Errorf("Len(): want 0 after closing, got %d", cache.Len())
	}

	if got := closedCount(*built); got != len(*built) {
		t.Errorf("want all %d clients closed, got %d", len(*built), got)
	}
}

// TestClientCacheConcurrent checks the cache under the access pattern it
// actually sees: every controller in the provider reconciling at once, most of
// them wanting the same client.
//
// Without a race detector this cannot fail, but it can hang or produce a
// mismatched client, which is what it is here to catch. Run with -race it also
// proves the locking is correct.
func TestClientCacheConcurrent(t *testing.T) {
	cache, built := newTestCache(t)

	cfg := zitadel.Config{
		URL:         instanceURL,
		Credentials: zitadel.Credentials{Token: testToken},
	}

	const workers = 20

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		first *zitadel.Client
	)

	wg.Add(workers)

	for range workers {
		go func() {
			defer wg.Done()

			got, err := cache.Get(context.Background(), cfg)
			if err != nil {
				t.Errorf("Get(...): unexpected error: %v", err)

				return
			}

			mu.Lock()
			defer mu.Unlock()

			if first == nil {
				first = got

				return
			}

			if got != first {
				t.Error("concurrent callers got different clients for one configuration")
			}
		}()
	}

	wg.Wait()

	// The lock is held across the build on purpose, so concurrent callers that
	// miss the cache produce one client, not one per caller.
	if len(*built) != 1 {
		t.Errorf("want one client built under concurrency, got %d", len(*built))
	}
}

// TestClientCacheRunnable checks that the cache can be handed to a manager, and
// that stopping the manager releases the connections.
func TestClientCacheRunnable(t *testing.T) {
	cache, built := newTestCache(t)

	cfg := zitadel.Config{
		URL:         instanceURL,
		Credentials: zitadel.Credentials{Token: testToken},
	}

	if _, err := cache.Get(context.Background(), cfg); err != nil {
		t.Fatalf("Get(...): unexpected error: %v", err)
	}

	r := cache.Runnable()

	// Closing connections on shutdown is safe from whichever instance is going
	// away, so it must not wait for a leader lease: a provider that loses its
	// lease would otherwise leak its connections until the process exits.
	if r.NeedLeaderElection() {
		t.Error("the cache runnable requires a leader election lease, so its connections leak on failover")
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)

	go func() { done <- r.Start(ctx) }()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start(...): unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Start(...) did not return after its context was cancelled")
	}

	if !(*built)[0].Closed() {
		t.Error("stopping the runnable did not release the cached connection")
	}
}

// TestClientCacheRegistersOnce checks that the shutdown hook is added to the
// manager once however many controllers ask for it.
//
// Every kind in the provider calls SetupManagedResourceController, so a
// registration that were not idempotent would start one watcher per kind -
// 104 of them - for the single job of closing the connections on shutdown.
func TestClientCacheRegistersOnce(t *testing.T) {
	fake := &countingManager{}
	cache := NewClientCache()

	for range 5 {
		if err := cache.Register(fake); err != nil {
			t.Fatalf("Register(...): unexpected error: %v", err)
		}
	}

	if fake.added != 1 {
		t.Errorf("Register(...): want the shutdown hook added once, got %d", fake.added)
	}
}

// countingManager records how many runnables were added to it.
type countingManager struct {
	ctrl.Manager

	added int
}

func (m *countingManager) Add(manager.Runnable) error {
	m.added++

	return nil
}
