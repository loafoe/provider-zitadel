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
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsZitadelID(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     string
		want   bool
	}{
		"Digits":   {reason: "A numeric ID is recognised", in: "1234567890", want: true},
		"Alpha":    {reason: "An alpha external name is not an ID", in: "my-resource", want: false},
		"Empty":    {reason: "An empty string is not an ID", in: "", want: false},
		"Mixed":    {reason: "A mixed string is not an ID", in: "123abc", want: false},
		"Negative": {reason: "A negative value is not an ID", in: "-1", want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := IsZitadelID(tc.in); got != tc.want {
				t.Errorf("\n%s\nIsZitadelID(%q): want %v, got %v", tc.reason, tc.in, tc.want, got)
			}
		})
	}
}

func TestRecover(t *testing.T) {
	type thing struct{ ID string }

	notFound := status.Error(codes.NotFound, "nope")
	boom := errors.New("boom")

	found := &thing{ID: "42"}

	cases := map[string]struct {
		reason    string
		external  string
		pinned    string
		byID      func(string) (*thing, error)
		byName    func() (*thing, error)
		wantID    string
		wantFound bool
		wantErr   bool
	}{
		"ByExternalName": {
			reason:    "A numeric external name is looked up directly",
			external:  "42",
			byID:      func(id string) (*thing, error) { return found, nil },
			byName:    func() (*thing, error) { return nil, nil },
			wantID:    "42",
			wantFound: true,
		},
		"ByPinnedID": {
			reason:    "A pinned spec ID is used when there is no external name",
			pinned:    "42",
			byID:      func(id string) (*thing, error) { return found, nil },
			byName:    func() (*thing, error) { return nil, nil },
			wantID:    "42",
			wantFound: true,
		},
		"ExternalNameWins": {
			reason:    "The external name takes precedence over the pinned ID",
			external:  "42",
			pinned:    "43",
			byID:      func(id string) (*thing, error) { return &thing{ID: id}, nil },
			byName:    func() (*thing, error) { return nil, nil },
			wantID:    "42",
			wantFound: true,
		},
		"ByName": {
			reason:    "Without an ID the resource is searched by name",
			byID:      func(id string) (*thing, error) { return nil, notFound },
			byName:    func() (*thing, error) { return found, nil },
			wantID:    "42",
			wantFound: true,
		},
		"ByNameNotFound": {
			reason:    "A name search that finds nothing reports not-found",
			byID:      func(id string) (*thing, error) { return nil, notFound },
			byName:    func() (*thing, error) { return nil, nil },
			wantFound: false,
		},
		"ByNameError": {
			reason:  "A name search error is propagated",
			byID:    func(id string) (*thing, error) { return nil, notFound },
			byName:  func() (*thing, error) { return nil, boom },
			wantErr: true,
		},
		"UserSuppliedExternalName": {
			reason:    "A user supplied external name is not resolved",
			external:  "my-resource",
			byID:      func(id string) (*thing, error) { return found, nil },
			byName:    func() (*thing, error) { return found, nil },
			wantFound: false,
		},
		"IDNotFound": {
			reason:    "A Zitadel not-found on the external name reports not-found",
			external:  "42",
			byID:      func(id string) (*thing, error) { return nil, notFound },
			byName:    func() (*thing, error) { return found, nil },
			wantFound: false,
		},
		"IDError": {
			reason:   "Any other lookup error is propagated",
			external: "42",
			byID:     func(id string) (*thing, error) { return nil, boom },
			byName:   func() (*thing, error) { return found, nil },
			wantErr:  true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, found, err := Recover(tc.external, tc.pinned, tc.byID, tc.byName)
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\nRecover(...): wantErr %v, got %v", tc.reason, tc.wantErr, err)
			}
			if found != tc.wantFound {
				t.Errorf("\n%s\nRecover(...): want found %v, got %v", tc.reason, tc.wantFound, found)
			}
			if found && got.ID != tc.wantID {
				t.Errorf("\n%s\nRecover(...): want id %q, got %q", tc.reason, tc.wantID, got.ID)
			}
		})
	}
}

func TestEqualStringSlices(t *testing.T) {
	cases := map[string]struct {
		reason string
		a      []string
		b      []string
		want   bool
	}{
		"BothEmpty":    {reason: "Two empty slices are equal", a: nil, b: []string{}, want: true},
		"Same":         {reason: "Identical slices are equal", a: []string{"a", "b"}, b: []string{"a", "b"}, want: true},
		"OrderMatters": {reason: "Order is significant", a: []string{"a", "b"}, b: []string{"b", "a"}, want: false},
		"DifferentLen": {reason: "Slices of different length are not equal", a: []string{"a"}, b: []string{"a", "b"}, want: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := EqualStringSlices(tc.a, tc.b); got != tc.want {
				t.Errorf("\n%s\nEqualStringSlices(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestValue(t *testing.T) {
	type state string

	if got := Value[state](nil); got != "" {
		t.Errorf("Value(nil): want empty string, got %q", got)
	}

	if got := Value(ptr[state]("Active")); got != "Active" {
		t.Errorf("Value(ptr(\"Active\")): want %q, got %q", "Active", got)
	}
}

func TestBool(t *testing.T) {
	if Bool(nil) {
		t.Error("Bool(nil): want false, got true")
	}
	if !Bool(ptr(true)) {
		t.Error("Bool(ptr(true)): want true, got false")
	}
}

func TestParseTime(t *testing.T) {
	if ParseTime("") != nil {
		t.Error("ParseTime(\"\"): want nil, got a timestamp")
	}
	if ParseTime("not a time") != nil {
		t.Error("ParseTime(\"not a time\"): want nil, got a timestamp")
	}
	if ParseTime("2024-01-02T03:04:05Z") == nil {
		t.Error("ParseTime(valid): want a timestamp, got nil")
	}
}

func TestFormatTime(t *testing.T) {
	if got := FormatTime(nil); got != "" {
		t.Errorf("FormatTime(nil): want empty string, got %q", got)
	}

	parsed := ParseTime("2024-01-02T03:04:05Z")
	if got := FormatTime(parsed); got != "2024-01-02T03:04:05Z" {
		t.Errorf("FormatTime(...): want %q, got %q", "2024-01-02T03:04:05Z", got)
	}
}

// ptr is a tiny helper so the tests read better.
func ptr[T any](v T) *T { return &v }
