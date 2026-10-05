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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/testing/protocmp"
)

// The registration restrictions and the secret generators are instance wide
// settings with no v2 API, so they are read and written through the v1 admin
// service. The generator is the interesting one: ZITADEL reports an unconfigured
// generator as absent rather than as an error, and getting that backwards would
// make the controller recreate a generator on every poll.

// TestRestrictions covers the read and the write, including the shape change the
// write performs: the flag goes out as a pointer and the list as a wrapper, and
// both have to come back as plain values.
func TestRestrictions(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		zc, _ := newAdminClient(t)
		ctx := testContext(t)

		want := &Restrictions{DisallowPublicOrgRegistration: true, AllowedLanguages: []string{"en", "de"}}

		if err := zc.SetRestrictions(ctx, *want); err != nil {
			t.Fatalf("SetRestrictions(...): unexpected error: %v", err)
		}

		got, err := zc.GetRestrictions(ctx)
		if err != nil {
			t.Fatalf("GetRestrictions(...): unexpected error: %v", err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("round trip: -want, +got:\n%s", diff)
		}
	})

	t.Run("Unset", func(t *testing.T) {
		// Nothing has been written, so the read reports the zero restrictions
		// rather than failing: a fresh instance has none of its own and that is
		// a value, not an error.
		zc, _ := newAdminClient(t)

		got, err := zc.GetRestrictions(testContext(t))
		if err != nil {
			t.Fatalf("GetRestrictions(...): unexpected error: %v", err)
		}

		if diff := cmp.Diff(&Restrictions{}, got); diff != "" {
			t.Errorf("want the zero restrictions, got:\n%s", diff)
		}
	})

	t.Run("ReadFailure", func(t *testing.T) {
		zc, a := newAdminClient(t)
		a.getErr = status.Error(codes.PermissionDenied, "nope")

		if _, err := zc.GetRestrictions(testContext(t)); err == nil {
			t.Error("a permission failure was swallowed")
		}
	})

	t.Run("WriteFailure", func(t *testing.T) {
		zc, a := newAdminClient(t)
		a.updateErr = status.Error(codes.PermissionDenied, "nope")

		if err := zc.SetRestrictions(testContext(t), Restrictions{}); err == nil {
			t.Error("a permission failure was swallowed")
		}
	})
}

// TestSecretGenerator covers the generator, which has the two distinctions that
// matter: an unknown type is refused before anything is sent, and an
// unconfigured generator reads as absent rather than as an error.
func TestSecretGenerator(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		zc, _ := newAdminClient(t)
		ctx := testContext(t)

		want := &SecretGenerator{
			Type:                "SECRET_GENERATOR_TYPE_INIT_CODE",
			Length:              8,
			Expiry:              "5m0s",
			IncludeLowerLetters: true,
			IncludeDigits:       true,
		}

		if err := zc.SetSecretGenerator(ctx, *want); err != nil {
			t.Fatalf("SetSecretGenerator(...): unexpected error: %v", err)
		}

		got, err := zc.SecretGenerator(ctx, want.Type)
		if err != nil {
			t.Fatalf("SecretGenerator(...): unexpected error: %v", err)
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("round trip: -want, +got:\n%s", diff)
		}
	})

	t.Run("Unconfigured", func(t *testing.T) {
		// There is no create for a generator: writing one brings it into being.
		// Until then ZITADEL reports it as absent, and that is an ordinary
		// observation rather than a failure.
		zc, _ := newAdminClient(t)

		got, err := zc.SecretGenerator(testContext(t), "SECRET_GENERATOR_TYPE_INIT_CODE")
		if err != nil {
			t.Fatalf("SecretGenerator(...): an unconfigured generator must not be an error: %v", err)
		}

		if got != nil {
			t.Errorf("an unconfigured generator returned %v, want nil", got)
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		zc, a := newAdminClient(t)
		a.getErr = notFound

		got, err := zc.SecretGenerator(testContext(t), "SECRET_GENERATOR_TYPE_INIT_CODE")
		if err != nil {
			t.Errorf("a missing generator must not be an error: %v", err)
		}

		if got != nil {
			t.Errorf("a missing generator returned %v, want nil", got)
		}
	})

	t.Run("UnknownTypeOnRead", func(t *testing.T) {
		// Refused before the call, because there is no such generator to ask
		// about and the error can say what the valid ones are.
		zc, a := newAdminClient(t)

		if _, err := zc.SecretGenerator(testContext(t), "SECRET_GENERATOR_TYPE_TELEPATHY"); err == nil {
			t.Error("an unknown generator type was accepted")
		}

		if a.lastWrite() == "get-secret-generator" {
			t.Error("an unknown generator type was sent to ZITADEL rather than refused")
		}
	})

	t.Run("UnknownTypeOnWrite", func(t *testing.T) {
		zc, a := newAdminClient(t)

		if err := zc.SetSecretGenerator(testContext(t), SecretGenerator{Type: "SECRET_GENERATOR_TYPE_TELEPATHY"}); err == nil {
			t.Error("an unknown generator type was accepted")
		}

		if a.lastWrite() == "update-secret-generator" {
			t.Error("an unknown generator type was sent to ZITADEL rather than refused")
		}
	})

	t.Run("UnparseableExpiry", func(t *testing.T) {
		zc, _ := newAdminClient(t)

		err := zc.SetSecretGenerator(testContext(t), SecretGenerator{
			Type:   "SECRET_GENERATOR_TYPE_INIT_CODE",
			Expiry: "soon",
		})
		if err == nil {
			t.Error("an unparseable expiry was accepted rather than reported")
		}
	})

	t.Run("AbsentExpiry", func(t *testing.T) {
		// No expiry means "leave it alone", so the field is not sent at all
		// rather than being sent as zero.
		zc, _ := newAdminClient(t)

		if err := zc.SetSecretGenerator(testContext(t), SecretGenerator{
			Type: "SECRET_GENERATOR_TYPE_INIT_CODE",
		}); err != nil {
			t.Fatalf("SetSecretGenerator(...): unexpected error: %v", err)
		}

		got, err := zc.SecretGenerator(testContext(t), "SECRET_GENERATOR_TYPE_INIT_CODE")
		if err != nil {
			t.Fatalf("SecretGenerator(...): unexpected error: %v", err)
		}

		if got.Expiry != "" {
			t.Errorf("Expiry: want empty when none was set, got %q", got.Expiry)
		}
	})
}

// TestParseExpiry covers the duration parsing on its own, including the negative
// case: ZITADEL rejects a negative expiry, so it has to be caught before it is
// sent rather than surfacing as an opaque API failure.
func TestParseExpiry(t *testing.T) {
	cases := map[string]struct {
		reason  string
		in      string
		want    time.Duration
		wantErr bool
	}{
		"Minutes":   {reason: "A minute parses", in: "5m", want: 5 * time.Minute},
		"Seconds":   {reason: "Seconds parse", in: "30s", want: 30 * time.Second},
		"Composite": {reason: "A composite duration parses", in: "1h30m", want: 90 * time.Minute},
		"Zero":      {reason: "Zero parses, and ZITADEL accepts it", in: "0s", want: 0},
		"Garbage":   {reason: "Garbage is rejected", in: "soon", wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := parseExpiry(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\nparseExpiry(%q): wantErr %v, got %v", tc.reason, tc.in, tc.wantErr, err)
			}

			if tc.wantErr {
				return
			}

			if diff := cmp.Diff(tc.want, got.AsDuration(), protocmp.Transform()); diff != "" {
				t.Errorf("\n%s\nparseExpiry(%q): -want, +got:\n%s", tc.reason, tc.in, diff)
			}
		})
	}
}

// TestBoolPtr covers the pointer the restrictions write needs: a bool that is
// false still has to be sent as false rather than as "unset", which is a
// different thing to ZITADEL.
func TestBoolPtr(t *testing.T) {
	for _, v := range []bool{true, false} {
		if got := boolPtr(v); got == nil || *got != v {
			t.Errorf("boolPtr(%v) = %v, want a pointer to %v", v, got, v)
		}
	}
}

// TestInputShapes covers the two Input methods, which exist so that a recorded
// snapshot can be written back on deletion. They have to convert to the write
// API's own type, or a deletion restores a differently shaped policy.
func TestInputShapes(t *testing.T) {
	restrictions := Restrictions{DisallowPublicOrgRegistration: true, AllowedLanguages: []string{"en"}}

	if got, ok := restrictions.Input().(Restrictions); !ok || cmp.Diff(restrictions, got) != "" {
		t.Errorf("Restrictions.Input(): want the same value, got %#v", got)
	}

	generator := SecretGenerator{Type: "SECRET_GENERATOR_TYPE_INIT_CODE", Length: 6}

	if got, ok := generator.Input().(SecretGenerator); !ok || cmp.Diff(generator, got) != "" {
		t.Errorf("SecretGenerator.Input(): want the same value, got %#v", got)
	}
}

// TestFeatureFlagInputShape is here to keep the third Input method honest.
func TestFeatureFlagInputShape(t *testing.T) {
	flags := FeatureFlags{LoginDefaultOrg: true, UserSchema: true}

	if got, ok := flags.Input().(FeatureFlags); !ok || cmp.Diff(flags, got) != "" {
		t.Errorf("FeatureFlags.Input(): want the same value, got %#v", got)
	}
}
