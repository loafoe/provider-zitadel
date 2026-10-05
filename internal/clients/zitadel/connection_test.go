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

package zitadel

import (
	"context"
	"strings"
	"testing"

	actionv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/action/v2"
	adminv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	applicationv2app "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"
	featurev2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/feature/v2"
	instancev2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/instance/v2"
	permissionv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/internal_permission/v2"
	orgv2api "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/org/v2"
	projectv2project "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/project/v2"
	settingsv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/settings/v2"
	userv2user "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user/v2"
	webkeyv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/webkey/v2"
)

// testConfig is a configuration that passes validation, so that a test can get
// as far as dialling. The host does not resolve and does not have to: gRPC
// connects lazily, so building the client performs no I/O.
func testConfig() Config {
	return Config{
		URL:         "https://zitadel.example.com",
		Credentials: Credentials{Token: "a-personal-access-token"},
	}
}

// TestNewClientSharesOneConnection pins down the reason the cache matters.
//
// A client used to dial one connection per Zitadel service it wrapped, twelve
// in all, each with its own copy of the authentication interceptor. Every
// reconcile paid twelve TLS handshakes and twelve token exchanges and then threw
// all of them away. All of the services are gRPC services on one endpoint, so
// they share a connection and a token source; this asserts that they still do,
// by checking that every service the client exposes is built from the same
// connection rather than one of its own.
func TestNewClientSharesOneConnection(t *testing.T) {
	zc, err := NewClient(context.Background(), testConfig())
	if err != nil {
		t.Fatalf("NewClient(...): unexpected error: %v", err)
	}

	defer func() { _ = zc.Close() }()

	if zc.conn == nil {
		t.Fatal("NewClient(...): no connection was established")
	}

	// Every convenience client the SDK offers carries the connection it was
	// built from, so comparing them against the client's own connection is
	// enough to tell a shared connection from a duplicated one.
	shared := map[string]bool{
		"user":        zc.user.Connection == zc.conn,
		"project":     zc.project.Connection == zc.conn,
		"application": zc.application.Connection == zc.conn,
		"org":         zc.org.Connection == zc.conn,
		"admin":       zc.admin.Connection == zc.conn,
	}

	for name, ok := range shared {
		if !ok {
			t.Errorf("the %s client does not share the client's connection: it has one of its own", name)
		}
	}

	// A second client is a separate connection: sharing is per client, not
	// global, so two configurations can never talk over each other's socket.
	other, err := NewClient(context.Background(), testConfig())
	if err != nil {
		t.Fatalf("NewClient(...): unexpected error: %v", err)
	}

	defer func() { _ = other.Close() }()

	if other.conn == zc.conn {
		t.Error("two clients share one connection, which would mix their credentials")
	}
}

// TestCloseReleasesEverything checks that closing a client releases the shared
// connection and the per organization clients built lazily on top of it, so a
// cached client that is retired does not leak a connection.
func TestCloseReleasesEverything(t *testing.T) {
	zc, err := NewClient(context.Background(), testConfig())
	if err != nil {
		t.Fatalf("NewClient(...): unexpected error: %v", err)
	}

	// The per organization clients are created lazily, so one is forced into
	// existence before Close is called.
	mc, err := zc.managementClient(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("managementClient(...): unexpected error: %v", err)
	}

	if mc == nil {
		t.Fatal("managementClient(...): no client returned")
	}

	if err := zc.Close(); err != nil {
		t.Errorf("Close(): unexpected error: %v", err)
	}

	if len(zc.management) != 0 {
		t.Errorf("Close(): want no per organization clients left, got %d", len(zc.management))
	}

	// Closing twice must not panic: the reconciler can call Disconnect after the
	// cache has already retired the client.
	if err := zc.Close(); err != nil {
		t.Errorf("second Close(): unexpected error: %v", err)
	}
}

// TestRelease distinguishes the two kinds of client.
//
// A client a reconcile owns has to be closed when it is done with. A client the
// cache owns is shared, so closing it at the end of one reconcile would break
// every other resource using the same configuration.
func TestRelease(t *testing.T) {
	t.Run("Owned", func(t *testing.T) {
		zc, err := NewClient(context.Background(), testConfig())
		if err != nil {
			t.Fatalf("NewClient(...): unexpected error: %v", err)
		}

		zc.Release()

		// A released owned client is closed, so its connection is shut down.
		if state := zc.conn.GetState().String(); state == "READY" {
			t.Errorf("Release() left the connection in state %s, want it closed", state)
		}
	})

	t.Run("Shared", func(t *testing.T) {
		zc, err := NewClient(context.Background(), testConfig())
		if err != nil {
			t.Fatalf("NewClient(...): unexpected error: %v", err)
		}

		defer func() { _ = zc.Close() }()

		zc.SetShared()
		zc.Release()

		// A released shared client survives, because the cache still hands it out.
		if zc.conn.GetState().String() == "SHUTDOWN" {
			t.Error("Release() closed a shared client, which would break every other user of it")
		}
	})
}

// TestConfigKey covers the identity a cached client is stored under.
//
// Everything that changes the client has to change the key, or a cached client
// would be reused for a configuration it was not built from - which is how a
// rotated secret ends up still authenticating with the old credential. And two
// configurations that really are the same have to produce the same key, or
// nothing would ever be shared.
func TestConfigKey(t *testing.T) {
	base := testConfig()

	cases := map[string]struct {
		reason string
		mutate func(*Config)
		// sameAsBase says whether the mutated configuration is still the same
		// configuration and therefore has to share a key.
		sameAsBase bool
	}{
		"NoChange": {
			reason:     "An identical configuration shares a key",
			mutate:     func(*Config) {},
			sameAsBase: true,
		},
		"DifferentURL": {
			reason:     "A different instance is a different client",
			mutate:     func(c *Config) { c.URL = "https://other.example.com" },
			sameAsBase: false,
		},
		"DifferentOrganization": {
			reason:     "A different default organization is a different client",
			mutate:     func(c *Config) { c.OrganizationID = "org-2" },
			sameAsBase: false,
		},
		"DifferentInsecure": {
			reason:     "Plaintext instead of TLS is a different client",
			mutate:     func(c *Config) { c.Insecure = true },
			sameAsBase: false,
		},
		"DifferentSkipVerify": {
			reason:     "Skipping certificate verification is a different client",
			mutate:     func(c *Config) { c.InsecureSkipTLSVerify = true },
			sameAsBase: false,
		},
		"RotatedToken": {
			reason:     "A rotated token is a different client",
			mutate:     func(c *Config) { c.Credentials.Token = "a-different-token" },
			sameAsBase: false,
		},
		"TokenWhitespace": {
			reason: "A token differing only in surrounding whitespace is the same client",
			// connectionOptions trims the token before authenticating, so the two
			// configurations behave identically and should not each hold a
			// connection.
			mutate:     func(c *Config) { c.Credentials.Token = "  a-personal-access-token\n" },
			sameAsBase: true,
		},
		"TokenInsteadOfKey": {
			reason:     "Authenticating with a token instead of a key is a different client",
			mutate:     func(c *Config) { c.Credentials = Credentials{ServiceAccountKey: []byte(`{"type":"serviceaccount"}`)} },
			sameAsBase: false,
		},
		"RotatedServiceAccountKey": {
			reason: "A rotated service account key is a different client",
			mutate: func(c *Config) {
				c.Credentials = Credentials{ServiceAccountKey: []byte(`{"keyId":2}`)}
			},
			sameAsBase: false,
		},
		"NoCredentials": {
			reason:     "A configuration without credentials is distinct from either method",
			mutate:     func(c *Config) { c.Credentials = Credentials{} },
			sameAsBase: false,
		},
		"FieldBoundary": {
			reason: "Fields cannot be shifted across the separator to forge a match",
			// Without length prefixing these two would hash the same, and one
			// would be served the other's client.
			mutate:     func(c *Config) { c.URL = "https://zitadel.example.co"; c.OrganizationID = "m" },
			sameAsBase: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			mutated := base
			tc.mutate(&mutated)

			got, want := mutated.Key(), base.Key()
			if same := got == want; same != tc.sameAsBase {
				t.Errorf("\n%s\nKey(): want the key to be shared=%v, got shared=%v", tc.reason, tc.sameAsBase, same)
			}
		})
	}
}

// TestConfigKeyIsStable checks the two properties a hash based cache key needs
// beyond discrimination: the same input always gives the same key, and the key
// carries no secret material.
func TestConfigKeyIsStable(t *testing.T) {
	cfg := Config{
		URL:         "https://zitadel.example.com",
		Credentials: Credentials{Token: "super-secret-token"},
	}

	first := cfg.Key()
	if second := cfg.Key(); first != second {
		t.Errorf("Key() is not stable: %q then %q", first, second)
	}

	// A key ends up in a map and, when something goes wrong, in a log line.
	for _, secret := range []string{"super-secret-token", "super-secret", "secret"} {
		if strings.Contains(first, secret) {
			t.Errorf("Key() leaks %q", secret)
		}
	}
}

// TestDefaultOrganizationID checks that a controller can read the ProviderConfig
// default organization off the client it was already given, rather than
// fetching the ProviderConfig a second time.
func TestDefaultOrganizationID(t *testing.T) {
	t.Run("Set", func(t *testing.T) {
		cfg := testConfig()
		cfg.OrganizationID = "org-7"

		zc, err := NewClient(context.Background(), cfg)
		if err != nil {
			t.Fatalf("NewClient(...): unexpected error: %v", err)
		}

		defer func() { _ = zc.Close() }()

		if got := zc.DefaultOrganizationID(); got != "org-7" {
			t.Errorf("DefaultOrganizationID(): want %q, got %q", "org-7", got)
		}
	})

	t.Run("Unset", func(t *testing.T) {
		zc, err := NewClient(context.Background(), testConfig())
		if err != nil {
			t.Fatalf("NewClient(...): unexpected error: %v", err)
		}

		defer func() { _ = zc.Close() }()

		if got := zc.DefaultOrganizationID(); got != "" {
			t.Errorf("DefaultOrganizationID(): want an empty string, got %q", got)
		}
	})
}

// TestClientDialsOnceAgainstAServer checks the connection sharing against a real
// gRPC server rather than by inspecting the struct.
//
// The struct comparison in TestNewClientSharesOneConnection proves the services
// are built from one connection; this proves the connection is one on the wire.
// Both matter: a client that pointed its twelve services at twelve sockets would
// still pass the structural check if they happened to share a struct field.
func TestClientDialsOnceAgainstAServer(t *testing.T) {
	// The fake registers nothing: this test makes calls, it does not answer
	// them, and Unimplemented is a perfectly good answer.
	fake := newFakeZitadel(t, nil)

	zc := fake.client(t)
	ctx := testContext(t)

	// One call per service surface the client wraps. Each would have needed its
	// own socket before the client shared one.
	calls := map[string]func(context.Context) error{
		"user": func(ctx context.Context) error {
			_, err := zc.user.GetUserByID(ctx, &userv2user.GetUserByIDRequest{UserId: "u1"})

			return err
		},
		"project": func(ctx context.Context) error {
			_, err := zc.project.ListProjects(ctx, &projectv2project.ListProjectsRequest{})

			return err
		},
		"application": func(ctx context.Context) error {
			_, err := zc.application.ListApplications(ctx, &applicationv2app.ListApplicationsRequest{})

			return err
		},
		"org": func(ctx context.Context) error {
			_, err := zc.org.ListOrganizations(ctx, &orgv2api.ListOrganizationsRequest{})

			return err
		},
		"action": func(ctx context.Context) error {
			_, err := zc.action.ListTargets(ctx, &actionv2.ListTargetsRequest{})

			return err
		},
		"settings": func(ctx context.Context) error {
			_, err := zc.settings.GetLoginSettings(ctx, &settingsv2.GetLoginSettingsRequest{})

			return err
		},
		"feature": func(ctx context.Context) error {
			_, err := zc.feature.GetSystemFeatures(ctx, &featurev2.GetSystemFeaturesRequest{})

			return err
		},
		"instance": func(ctx context.Context) error {
			_, err := zc.instance.GetInstance(ctx, &instancev2.GetInstanceRequest{InstanceId: "i1"})

			return err
		},
		"orgDomain": func(ctx context.Context) error {
			_, err := zc.orgDomain.ListOrganizationDomains(ctx, &orgv2api.ListOrganizationDomainsRequest{
				OrganizationId: "o1",
			})

			return err
		},
		"webkey": func(ctx context.Context) error {
			_, err := zc.webkey.ListWebKeys(ctx, &webkeyv2.ListWebKeysRequest{})

			return err
		},
		"permission": func(ctx context.Context) error {
			_, err := zc.permission.ListAdministrators(ctx, &permissionv2.ListAdministratorsRequest{})

			return err
		},
		"admin": func(ctx context.Context) error {
			_, err := zc.admin.GetSupportedLanguages(ctx, &adminv1.GetSupportedLanguagesRequest{})

			return err
		},
	}

	for name, call := range calls {
		if err := call(ctx); err == nil {
			t.Errorf("%s: want an Unimplemented error from the empty fake, got nil", name)
		}
	}

	if !waitFor(t, func() bool { return fake.connections() > 0 }) {
		t.Fatal("no connection reached the fake")
	}

	// Twelve services, all over one socket.
	// Before the client shared its connection this was twelve.
	if got := fake.connections(); got != 1 {
		t.Errorf("want the client to dial exactly one connection, got %d", got)
	}
}
