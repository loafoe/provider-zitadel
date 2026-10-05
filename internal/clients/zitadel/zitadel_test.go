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
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Values the cases share, named so that a case reads as a statement about
// behaviour rather than about a repeated literal.
const (
	userActive   = "Active"
	userInactive = "Inactive"
)

// Test values, named so that a case reads as a statement about behaviour
// rather than about a repeated string.
const (
	garbage = "Nope"
)

func TestIsNotFound(t *testing.T) {
	cases := map[string]struct {
		reason string
		err    error
		want   bool
	}{
		"NoError": {
			reason: "A nil error is never a not-found error",
			err:    nil,
			want:   false,
		},
		"Sentinel": {
			reason: "The package sentinel is recognised",
			err:    ErrNotFound,
			want:   true,
		},
		"WrappedSentinel": {
			reason: "A wrapped sentinel is recognised",
			err:    errors.Wrap(ErrNotFound, "while getting the project"),
			want:   true,
		},
		"GRPCCode": {
			reason: "A gRPC NotFound status is recognised",
			err:    status.Error(codes.NotFound, "project not found"),
			want:   true,
		},
		"OtherGRPCCode": {
			reason: "Any other gRPC status is not a not-found error",
			err:    status.Error(codes.PermissionDenied, "nope"),
			want:   false,
		},
		"PlainError": {
			reason: "A plain Go error is not a not-found error",
			err:    errors.New("boom"),
			want:   false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.want {
				t.Errorf("\n%s\nIsNotFound(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestHostFromURL(t *testing.T) {
	cases := map[string]struct {
		reason  string
		url     string
		want    string
		wantErr bool
	}{
		"FullURL": {
			reason: "A full https URL yields the host",
			url:    "https://auth.internal.loafoe.com",
			want:   "auth.internal.loafoe.com",
		},
		"TrailingPath": {
			reason: "Any path is dropped",
			url:    "https://auth.internal.loafoe.com/some/path",
			want:   "auth.internal.loafoe.com",
		},
		"ExplicitPort": {
			reason: "An explicit port is kept",
			url:    "https://zitadel.local:8443",
			want:   "zitadel.local:8443",
		},
		"HTTP": {
			reason: "A plaintext URL yields the host without the scheme",
			url:    "http://zitadel.local",
			want:   "zitadel.local",
		},
		"NoScheme": {
			reason: "A bare host is accepted",
			url:    "zitadel.local",
			want:   "zitadel.local",
		},
		"Empty": {
			reason:  "An empty URL is rejected",
			url:     "",
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := hostFromURL(tc.url)
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\nhostFromURL(...): unexpected error: %v", tc.reason, err)
			}
			if got != tc.want {
				t.Errorf("\n%s\nhostFromURL(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

func TestIssuerFromURL(t *testing.T) {
	cases := map[string]struct {
		reason   string
		url      string
		insecure bool
		want     string
	}{
		"Secure": {
			reason: "A secure issuer is https",
			url:    "https://auth.internal.loafoe.com/",
			want:   "https://auth.internal.loafoe.com",
		},
		"Insecure": {
			reason:   "An insecure issuer is http",
			url:      "https://zitadel.local",
			insecure: true,
			want:     "http://zitadel.local",
		},
		"InsecureInput": {
			reason:   "An http input with insecure is idempotent",
			url:      "http://zitadel.local",
			insecure: true,
			want:     "http://zitadel.local",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := issuerFromURL(tc.url, tc.insecure)
			if err != nil {
				t.Fatalf("\n%s\nissuerFromURL(...): unexpected error: %v", tc.reason, err)
			}
			if got != tc.want {
				t.Errorf("\n%s\nissuerFromURL(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

func TestNewClientValidation(t *testing.T) {
	cases := map[string]struct {
		reason  string
		cfg     Config
		wantErr bool
	}{
		"NoURL": {
			reason: "A config without a URL is rejected",
			cfg: Config{
				Credentials: Credentials{Token: "token"},
			},
			wantErr: true,
		},
		"NoCredentials": {
			reason: "A config without credentials is rejected",
			cfg: Config{
				URL: "https://zitadel.local",
			},
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewClient(context.Background(), tc.cfg)
			if (err != nil) != tc.wantErr {
				t.Errorf("\n%s\nNewClient(...): wantErr %v, got %v", tc.reason, tc.wantErr, err)
			}
		})
	}
}

func TestUserStateToProto(t *testing.T) {
	cases := map[string]struct {
		reason  string
		state   string
		want    string
		wantErr bool
	}{
		"Empty":    {reason: "An empty state defaults to active", state: "", want: "USER_STATE_ACTIVE"},
		userActive: {reason: "Active maps to active", state: userActive, want: "USER_STATE_ACTIVE"},
		userInactive: {
			reason: "Inactive maps to inactive",
			state:  userInactive,
			want:   "USER_STATE_INACTIVE",
		},
		"Locked": {reason: "Locked maps to locked", state: "Locked", want: "USER_STATE_LOCKED"},
		"Initial": {
			reason: "Initial maps to initial",
			state:  "Initial",
			want:   "USER_STATE_INITIAL",
		},
		"Unknown": {reason: "An unknown state is rejected", state: garbage, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := UserStateToProto(tc.state)
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\nUserStateToProto(...): unexpected error: %v", tc.reason, err)
			}
			if !tc.wantErr && got.String() != tc.want {
				t.Errorf("\n%s\nUserStateToProto(...): want %s, got %s", tc.reason, tc.want, got.String())
			}
		})
	}
}

func TestUserStateRoundTrip(t *testing.T) {
	states := []string{userActive, userInactive, "Locked", "Initial"}
	for _, want := range states {
		t.Run(want, func(t *testing.T) {
			p, err := UserStateToProto(want)
			if err != nil {
				t.Fatalf("UserStateToProto(%q): unexpected error: %v", want, err)
			}
			if got := UserStateFromProto(p); got != want {
				t.Errorf("round trip: want %q, got %q", want, got)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	cases := map[string]struct {
		reason  string
		in      string
		wantErr bool
		want    string
	}{
		"Seconds": {reason: "A plain duration parses", in: "5s", want: "5s"},
		"Minutes": {reason: "A minute parses", in: "1m", want: "1m0s"},
		"Composite": {
			reason: "A composite duration parses",
			in:     "1m30s",
			want:   "1m30s",
		},
		"Negative": {reason: "A negative duration is rejected", in: "-5s", wantErr: true},
		"Garbage":  {reason: "Garbage is rejected", in: "soon", wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ParseDuration(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\nParseDuration(...): unexpected error: %v", tc.reason, err)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(tc.want, got.AsDuration().String()); diff != "" {
				t.Errorf("\n%s\nParseDuration(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	if got := formatTimestamp(nil); got != "" {
		t.Errorf("formatTimestamp(nil): want empty string, got %q", got)
	}

	ts := timestamppb.New(time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC))
	if got, want := formatTimestamp(ts), "2024-01-02T03:04:05Z"; got != want {
		t.Errorf("formatTimestamp(...): want %q, got %q", want, got)
	}
}
