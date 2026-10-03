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
	"os"
	"testing"
)

// TestProbe reads the instance wide settings straight from Zitadel, so that what
// a reconciliation or a deletion actually did can be checked rather than
// inferred from a resource's own status.
//
// It is a verification tool rather than a test of the provider. It reads, it
// changes nothing, and it does nothing unless both ZITADEL_KEY (the JSON of a
// service account's machine key) and ZITADEL_URL are set:
//
//	ZITADEL_KEY="$(cat key.json)" ZITADEL_URL=https://instance.example \
//	    go test ./internal/clients/zitadel/ -run TestProbe -v
//
// The key is a credential and has no business in the suite, so it is read from
// the environment and never committed.
func TestProbe(t *testing.T) {
	key, url := os.Getenv("ZITADEL_KEY"), os.Getenv("ZITADEL_URL")
	if key == "" || url == "" {
		t.Skip("ZITADEL_KEY and ZITADEL_URL are not set; this is a manual verification tool")
	}

	c, err := NewClient(t.Context(), Config{
		URL:         url,
		Credentials: Credentials{ServiceAccountKey: []byte(key)},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if f, err := c.GetInstanceFeatures(t.Context()); err == nil {
		t.Logf("instance features   : %+v", *f)
	} else {
		t.Logf("instance features   : %v", err)
	}

	if f, err := c.GetSystemFeatures(t.Context()); err == nil {
		t.Logf("system features     : %+v", *f)
	} else {
		t.Logf("system features     : %v", err)
	}

	if r, err := c.GetRestrictions(t.Context()); err == nil {
		t.Logf("restrictions        : %+v", *r)
	} else {
		t.Logf("restrictions        : %v", err)
	}

	for _, gt := range []string{
		"SECRET_GENERATOR_TYPE_INIT_CODE",
		"SECRET_GENERATOR_TYPE_VERIFY_EMAIL_CODE",
		"SECRET_GENERATOR_TYPE_PASSWORD_RESET_CODE",
		"SECRET_GENERATOR_TYPE_APP_SECRET",
	} {
		g, err := c.SecretGenerator(t.Context(), gt)
		switch {
		case err != nil:
			t.Logf("generator %-45s: %v", gt, err)

		case g == nil:
			t.Logf("generator %-45s: not configured", gt)

		default:
			t.Logf("generator %-45s: %+v", gt, *g)
		}
	}
}
