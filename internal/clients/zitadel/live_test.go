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

package zitadel_test

import (
	"context"
	"os"
	"testing"

	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// These tests talk to a real Zitadel instance. They are skipped unless the
// environment provides the credentials, so `make test` stays hermetic.
//
//	ZP_URL   the base URL of the instance, e.g. https://my-instance.zitadel.cloud
//	ZP_PAT   a Personal Access Token of a service account
//	ZP_KEY   the path to a machine key file (key.json) of a service account
//
// A service account used with ZP_KEY must have the "Jwt" access token type and
// at least ORG_OWNER (or IAM_OWNER) on the organization under test, otherwise
// Zitadel accepts the token but rejects the authorization.

func liveConfig(t *testing.T) zitadel.Config {
	t.Helper()

	url := os.Getenv("ZP_URL")
	if url == "" {
		t.Skip("ZP_URL is not set, skipping the live Zitadel test")
	}

	cfg := zitadel.Config{URL: url}

	if key := os.Getenv("ZP_KEY"); key != "" {
		data, err := os.ReadFile(key)
		if err != nil {
			t.Fatalf("cannot read ZP_KEY: %v", err)
		}
		cfg.Credentials.ServiceAccountKey = data
	}

	if pat := os.Getenv("ZP_PAT"); pat != "" {
		cfg.Credentials.Token = pat
	}

	if len(cfg.Credentials.ServiceAccountKey) == 0 && cfg.Credentials.Token == "" {
		t.Skip("neither ZP_KEY nor ZP_PAT is set, skipping the live Zitadel test")
	}

	return cfg
}

// TestLiveServiceAccountKeyAuth proves that a machine key is accepted by the
// Zitadel API. This is the regression test for the API audience scope: without
// ScopeZitadelAPI the issued token carries the user name as its audience and
// every call fails with "Unauthenticated: Errors.Token.Invalid".
func TestLiveServiceAccountKeyAuth(t *testing.T) {
	key := os.Getenv("ZP_KEY")
	if key == "" {
		t.Skip("ZP_KEY is not set, skipping the live service account key test")
	}

	cfg := liveConfig(t)

	c, err := zitadel.NewClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("cannot create the Zitadel client: %v", err)
	}
	defer c.Close()

	// ListOrganizations is an instance level call that only needs a valid
	// token. An unprivileged service account legitimately sees nothing here;
	// what matters is that the call does not fail with Unauthenticated.
	orgs, err := c.ListOrganizations(context.Background())
	if err != nil {
		if zitadel.IsInvalidToken(err) {
			t.Fatalf("Zitadel rejected the credentials: %v", err)
		}
		t.Fatalf("cannot list organizations: %v", err)
	}

	t.Logf("authenticated, %d organization(s) visible", len(orgs))
}

// TestLiveTokenAuth does the same for a Personal Access Token.
func TestLiveTokenAuth(t *testing.T) {
	if os.Getenv("ZP_PAT") == "" {
		t.Skip("ZP_PAT is not set, skipping the live token test")
	}

	c, err := zitadel.NewClient(context.Background(), liveConfig(t))
	if err != nil {
		t.Fatalf("cannot create the Zitadel client: %v", err)
	}
	defer c.Close()

	orgs, err := c.ListOrganizations(context.Background())
	if err != nil {
		t.Fatalf("cannot list organizations: %v", err)
	}

	t.Logf("authenticated, %d organization(s) visible", len(orgs))
}
