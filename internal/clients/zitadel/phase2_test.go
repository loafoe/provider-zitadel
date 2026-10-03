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
	"fmt"
	"strings"
	"testing"

	feature "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/feature/v2"
	orgv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/org/v2"
	settings "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/settings"

	"github.com/google/go-cmp/cmp"
	policyv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/policy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Values the tables below keep repeating.
const (
	testAlpha  = "alpha"
	testOwner  = "ORG_OWNER"
	testViewer = "ORG_USER_VIEWER"
)

func TestNormalizeMetadata(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     []MetadataEntry
		want   []MetadataEntry
	}{
		"SortedByKey": {
			reason: "Entries are sorted by key so that order cannot look like drift",
			in: []MetadataEntry{
				{Key: "zeta", Value: "z"},
				{Key: testAlpha, Value: "a"},
			},
			want: []MetadataEntry{
				{Key: testAlpha, Value: "a"},
				{Key: "zeta", Value: "z"},
			},
		},
		"EmptyKeysDropped": {
			reason: "An entry without a key is not a thing Zitadel can store",
			in: []MetadataEntry{
				{Key: "", Value: "orphan"},
				{Key: testAlpha, Value: "a"},
			},
			want: []MetadataEntry{{Key: testAlpha, Value: "a"}},
		},
		"EmptyValuesKept": {
			reason: "An empty value is a real value: it clears the key",
			in:     []MetadataEntry{{Key: testAlpha, Value: ""}},
			want:   []MetadataEntry{{Key: testAlpha, Value: ""}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, Normalize(tc.in)); diff != "" {
				t.Errorf("\n%s\nNormalize(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestEqualMetadata(t *testing.T) {
	cases := map[string]struct {
		reason string
		a      []MetadataEntry
		b      []MetadataEntry
		want   bool
	}{
		"OrderIrrelevant": {
			reason: "Zitadel stores metadata as a set, so order is not drift",
			a:      []MetadataEntry{{Key: "b", Value: "2"}, {Key: "a", Value: "1"}},
			b:      []MetadataEntry{{Key: "a", Value: "1"}, {Key: "b", Value: "2"}},
			want:   true,
		},
		"ValueDrift": {
			reason: "A changed value is drift",
			a:      []MetadataEntry{{Key: "a", Value: "1"}},
			b:      []MetadataEntry{{Key: "a", Value: "2"}},
			want:   false,
		},
		"AddedKey": {
			reason: "An extra key on the remote side is drift",
			a:      []MetadataEntry{{Key: "a", Value: "1"}},
			b:      []MetadataEntry{{Key: "a", Value: "1"}, {Key: "b", Value: "2"}},
			want:   false,
		},
		"BothEmpty": {
			reason: "Two empty sets are equal",
			want:   true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := EqualMetadata(tc.a, tc.b); got != tc.want {
				t.Errorf("\n%s\nEqualMetadata(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestValidateRoles(t *testing.T) {
	available := []string{testOwner, "ORG_USER_MANAGER", testViewer}

	cases := map[string]struct {
		reason    string
		requested []string
		want      []string
	}{
		"AllKnown": {
			reason:    "Roles the instance offers are accepted",
			requested: []string{testOwner, testViewer},
			want:      nil,
		},
		"OneUnknown": {
			reason:    "A typo is named back so it can be fixed",
			requested: []string{testOwner, "ORG_ADMIn"},
			want:      []string{"ORG_ADMIn"},
		},
		"AllUnknown": {
			reason:    "Every unknown role is reported, not just the first",
			requested: []string{"NOPE", "ALSO_NOPE"},
			want:      []string{"NOPE", "ALSO_NOPE"},
		},
		"NoneRequested": {
			reason: "An empty request has nothing to reject",
			want:   nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, ValidateRoles(tc.requested, available)); diff != "" {
				t.Errorf("\n%s\nValidateRoles(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestFormatUnknownRoles(t *testing.T) {
	err := FormatUnknownRoles("organization roles", []string{"ORG_ADMIn"}, []string{testOwner, "ORG_USER_MANAGER"})
	if err == nil {
		t.Fatal("FormatUnknownRoles(...): want an error, got nil")
	}

	msg := err.Error()
	for _, want := range []string{"ORG_ADMIn", testOwner, "ORG_USER_MANAGER"} {
		if !contains(msg, want) {
			t.Errorf("FormatUnknownRoles(...): the message does not mention %q: %s", want, msg)
		}
	}
}

func TestDecodeMachineKey(t *testing.T) {
	cases := map[string]struct {
		reason  string
		in      string
		wantErr bool
	}{
		"Valid": {
			reason: "A complete key.json is accepted",
			in:     `{"type":"serviceaccount","keyId":"1","userId":"2","key":"-----BEGIN RSA PRIVATE KEY-----"}`,
		},
		"NotJSON": {
			reason:  "Garbage is rejected",
			in:      `not json`,
			wantErr: true,
		},
		"MissingKeyID": {
			reason:  "A key without an ID cannot be used to identify itself",
			in:      `{"type":"serviceaccount","userId":"2","key":"k"}`,
			wantErr: true,
		},
		"MissingPrivateKey": {
			reason:  "A key without private material cannot authenticate anything",
			in:      `{"type":"serviceaccount","keyId":"1","userId":"2"}`,
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeMachineKey([]byte(tc.in))
			if (err != nil) != tc.wantErr {
				t.Errorf("\n%s\nDecodeMachineKey(...): wantErr %v, got %v", tc.reason, tc.wantErr, err)
			}
		})
	}
}

func TestProjectGrantExternalName(t *testing.T) {
	name := ProjectGrantExternalName("p1", "o1")
	if want := "p1/o1"; name != want {
		t.Errorf("ProjectGrantExternalName(...): want %q, got %q", want, name)
	}

	project, org, ok := SplitProjectGrantExternalName(name)
	if !ok || project != "p1" || org != "o1" {
		t.Errorf("SplitProjectGrantExternalName(%q): got %q/%q ok=%v", name, project, org, ok)
	}

	for _, bad := range []string{"", "p1", "/o1", "p1/"} {
		if _, _, ok := SplitProjectGrantExternalName(bad); ok {
			t.Errorf("SplitProjectGrantExternalName(%q): want ok=false, got true", bad)
		}
	}
}

func TestAPIAuthMethodRoundTrip(t *testing.T) {
	for _, name := range []string{AuthMethodBasic, AuthMethodPrivateKeyJwt} {
		t.Run(name, func(t *testing.T) {
			proto, err := APIAuthMethodToProto(name)
			if err != nil {
				t.Fatalf("APIAuthMethodToProto(%q): %v", name, err)
			}
			if got := APIAuthMethodFromProto(proto); got != name {
				t.Errorf("round trip: want %q, got %q", name, got)
			}
		})
	}

	if _, err := APIAuthMethodToProto("Magic"); err == nil {
		t.Error("APIAuthMethodToProto(\"Magic\"): want an error, got nil")
	}
}

func TestFactorNameRoundTrip(t *testing.T) {
	for _, name := range []string{"OTP", "U2F", "OTPEmail", "OTPSMS", "RecoveryCodes", "U2FWithVerification"} {
		t.Run(name, func(t *testing.T) {
			var (
				got  string
				err  error
				want = name
			)
			if name == "U2FWithVerification" {
				p, e := MultiFactorToProto(name)
				err = e
				got = factorNameFromMulti(p)
			} else {
				p, e := SecondFactorToProto(name)
				err = e
				got = factorNameFromSecond(p)
			}

			if err != nil {
				t.Fatalf("to proto: %v", err)
			}
			if got != want {
				t.Errorf("round trip: want %q, got %q", want, got)
			}
		})
	}
}

// factorNameFromSecond and factorNameFromMulti exist only to keep the table
// above readable.
func factorNameFromSecond(v policyv1.SecondFactorType) string {
	names, err := SecondFactorNames([]policyv1.SecondFactorType{v})
	if err != nil || len(names) == 0 {
		return ""
	}

	return names[0]
}

func factorNameFromMulti(v policyv1.MultiFactorType) string {
	names, err := MultiFactorNames([]policyv1.MultiFactorType{v})
	if err != nil || len(names) == 0 {
		return ""
	}

	return names[0]
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

// Zitadel's metadata endpoints are additive, so conveying "these are all the
// entries" means deleting the keys that are no longer wanted. Getting this
// wrong leaves stale keys behind forever: the controller observes a set that
// never shrinks, so it reports drift on every reconcile and the removed entry
// stays in Zitadel.
func TestStaleKeys(t *testing.T) {
	cases := map[string]struct {
		current []MetadataEntry
		desired []MetadataEntry
		want    []string
	}{
		"an entry dropped from the spec is stale": {
			current: []MetadataEntry{{Key: "a", Value: "1"}, {Key: "b", Value: "2"}},
			desired: []MetadataEntry{{Key: "a", Value: "1"}},
			want:    []string{"b"},
		},
		"a changed value keeps its key": {
			current: []MetadataEntry{{Key: "a", Value: "1"}},
			desired: []MetadataEntry{{Key: "a", Value: "2"}},
		},
		"an emptied spec makes every key stale": {
			current: []MetadataEntry{{Key: "a", Value: "1"}, {Key: "b", Value: "2"}},
			want:    []string{"a", "b"},
		},
		"nothing current means nothing stale": {
			desired: []MetadataEntry{{Key: "a", Value: "1"}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := staleKeys(tc.current, tc.desired)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("staleKeys(...): -want, +got:\n%s", diff)
			}
		})
	}
}

// Zitadel refuses to add a role to an existing project grant and reports it as
// an unknown project role. Recognising that lets the provider say what actually
// went wrong instead of leaving an operator to suspect a typo in a role key.
func TestIsRoleNotFound(t *testing.T) {
	wrapped := status.Error(codes.FailedPrecondition,
		`rpc error: code = FailedPrecondition desc = Errors.Project.Role.NotFound (COMMAND-6m9gd)`)

	for name, tc := range map[string]struct {
		err  error
		want bool
	}{
		"a wrapped role error":            {err: fmt.Errorf("cannot update: %w", wrapped), want: true},
		"the bare role error":             {err: wrapped, want: true},
		"some other precondition failure": {err: fmt.Errorf("x: %w", status.Error(codes.FailedPrecondition, "Errors.Org.LoginPolicy.NotChanged")), want: false},
		"a different code":                {err: fmt.Errorf("x: %w", status.Error(codes.NotFound, "Errors.Project.Role.NotFound")), want: false},
		"no error at all":                 {},
	} {
		t.Run(name, func(t *testing.T) {
			if got := isRoleNotFound(tc.err); got != tc.want {
				t.Errorf("isRoleNotFound(%v) = %t, want %t", tc.err, got, tc.want)
			}
		})
	}
}

// Zitadel sends these enums as numbers, so string(p) yields the character with
// that code point rather than the name.
//
// This is the bug a compile, a vet and a lint pass all agree with: the value
// only goes wrong once Zitadel answers, which is exactly when a live test is
// the only thing that catches it.
func TestEnumNamesAreReadRatherThanStringified(t *testing.T) {
	// These are the values Zitadel returns, as the numbers it sends them as.
	for _, tc := range []struct {
		got  int32
		want string
	}{
		{1, "SECRET_GENERATOR_TYPE_INIT_CODE"},
		{6, "SECRET_GENERATOR_TYPE_APP_SECRET"},
	} {
		got, ok := settings.SecretGeneratorType_name[tc.got]
		if !ok {
			t.Errorf("no name for the generator type %d", tc.got)
			continue
		}
		if got != tc.want {
			t.Errorf("the generator type %d read as %q, want %q", tc.got, got, tc.want)
		}

		// What the bug produced instead.
		if string(tc.got) == tc.want {
			t.Errorf("the generator type %d stringifies to its own name, which is the "+
				"coincidence this test guards against", tc.got)
		}
	}

	for _, tc := range []struct {
		got  int32
		want string
	}{
		{2, "IMPROVED_PERFORMANCE_PROJECT_GRANT"},
		{3, "IMPROVED_PERFORMANCE_PROJECT"},
		{4, "IMPROVED_PERFORMANCE_USER_GRANT"},
	} {
		got, ok := feature.ImprovedPerformance_name[tc.got]
		if !ok {
			t.Errorf("no name for the improved performance value %d", tc.got)
			continue
		}
		if got != tc.want {
			t.Errorf("the improved performance value %d read as %q, want %q", tc.got, got, tc.want)
		}
	}
}

// Zitadel sends enumerations as numbers, so converting one straight to a string
// yields the character with that code point rather than its name.
//
// It bit twice: first on the feature flags and the secret generator types, then
// again on a domain's validation type, where the unspecified value of 0 put a
// NUL character in the status. Every conversion is pinned here so the third
// does not happen either.
func TestEnumsAreReadThroughTheirNameMaps(t *testing.T) {
	for _, tc := range []struct {
		name  string
		got   int32
		known bool
	}{
		{"the unspecified validation type", 0, true},
		{"http validation", 1, true},
		{"dns validation", 2, true},
		{"a validation type Zitadel does not have", 99, false},
	} {
		got, ok := orgv2.DomainValidationType_name[tc.got]
		if ok != tc.known {
			t.Errorf("%s: the name map knows %d as %t, want %t", tc.name, tc.got, ok, tc.known)
			continue
		}

		if ok && got == string(tc.got) {
			t.Errorf("%s: %d read as %q, which is the code point rather than the name",
				tc.name, tc.got, got)
		}
	}
}
