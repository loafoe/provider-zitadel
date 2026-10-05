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

	"github.com/google/go-cmp/cmp"
	idpv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/idp"
	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	policyv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/policy"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/durationpb"
)

// The login policy is the widest thing this provider sends: five lifetimes, a
// passwordless mode and two factor lists, all of which have to survive the round
// trip through Zitadel's own spelling of them. Each mapping below is therefore
// covered in both directions, because a value that comes back differently is a
// resource that never settles.

// TestDifference covers the set arithmetic behind the factor lists, which are
// written additively by ZITADEL and so have to be reconciled by hand.
func TestDifference(t *testing.T) {
	cases := map[string]struct {
		reason string
		a, b   []string
		want   []string
	}{
		"Empty": {
			reason: "Nothing to reconcile from nothing",
			a:      nil, b: nil,
			want: nil,
		},
		"Disjoint": {
			reason: "Every value is new",
			a:      []string{"a", "b"}, b: []string{"c"},
			want: []string{"a", "b"},
		},
		"Overlapping": {
			reason: "Only the values ZITADEL does not hold are new",
			a:      []string{"a", "b", "c"}, b: []string{"b"},
			want: []string{"a", "c"},
		},
		"Subset": {
			reason: "Everything asked for is already held",
			a:      []string{"a"}, b: []string{"a", "b"},
			want: nil,
		},
		"Duplicates": {
			reason: "A repeated value that is not held is reported twice",
			a:      []string{"a", "a"}, b: nil,
			want: []string{"a", "a"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, difference(tc.a, tc.b)); diff != "" {
				t.Errorf("\n%s\ndifference(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     []string
		want   []string
	}{
		"Empty":         {reason: "Nothing sorts to nothing", in: nil, want: []string{}},
		"Sorted":        {reason: "Values come back sorted", in: []string{"c", "a", "b"}, want: []string{"a", "b", "c"}},
		"AlreadySorted": {reason: "An already sorted list is unchanged", in: []string{"a", "b"}, want: []string{"a", "b"}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, normalize(tc.in)); diff != "" {
				t.Errorf("\n%s\nnormalize(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

// TestDurationRoundTrip covers the five lifetimes, which are the part of the
// policy most likely to read back differently: ZITADEL normalises them, so a
// comparison has to be by meaning rather than by spelling.
func TestDurationRoundTrip(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     string
		want   string
	}{
		"Unset":     {reason: "An unset lifetime stays absent rather than becoming zero", in: "", want: ""},
		"Minutes":   {reason: "A minute survives the round trip", in: "1m", want: "1m0s"},
		"Seconds":   {reason: "A lifetime in seconds survives", in: "30s", want: "30s"},
		"Composite": {reason: "A composite lifetime survives", in: "1h30m", want: "1h30m0s"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			proto := durationProto(tc.in)

			if tc.in == "" {
				if proto != nil {
					t.Errorf("\n%s\ndurationProto(%q): want nil so the field is left alone, got %v", tc.reason, tc.in, proto)
				}

				if got := durationString(proto); got != "" {
					t.Errorf("\n%s\ndurationString(nil): want empty, got %q", tc.reason, got)
				}

				return
			}

			if got := durationString(proto); got != tc.want {
				t.Errorf("\n%s\nround trip of %q: want %q, got %q", tc.reason, tc.in, tc.want, got)
			}
		})
	}
}

// TestDurationProtoRejectsGarbage covers the failure that has to be invisible: an
// unparseable lifetime is dropped rather than written as something ZITADEL will
// reject, because a lifetime is a preference rather than the policy itself.
func TestDurationProtoRejectsGarbage(t *testing.T) {
	for _, in := range []string{"soon", "-5s", "5"} {
		if got := durationProto(in); got != nil {
			t.Errorf("durationProto(%q): want nil rather than %v", in, got)
		}
	}
}

// TestLoginPolicyDurations covers the five lifetimes as a group, including the
// error that names which one is wrong - a message that cannot say which field is
// wrong sends the operator looking in the wrong place.
func TestLoginPolicyDurations(t *testing.T) {
	cases := map[string]struct {
		reason   string
		in       LoginPolicyInput
		want     policyDurations
		wantErr  bool
		errField string
	}{
		"None": {
			reason: "No lifetimes at all leaves them all absent",
			want:   policyDurations{},
		},
		"All": {
			reason: "Every lifetime is parsed",
			in: LoginPolicyInput{
				PasswordCheckLifetime:      "1h",
				ExternalLoginCheckLifetime: "2h",
				MFAInitSkipLifetime:        "3h",
				SecondFactorCheckLifetime:  "4h",
				MultiFactorCheckLifetime:   "5h",
			},
			want: policyDurations{
				passwordCheckLifetime:      durationpb.New(3600000000000),
				externalLoginCheckLifetime: durationpb.New(7200000000000),
				mfaInitSkipLifetime:        durationpb.New(10800000000000),
				secondFactorCheckLifetime:  durationpb.New(14400000000000),
				multiFactorCheckLifetime:   durationpb.New(18000000000000),
			},
		},
		"Partly": {
			reason: "Only the lifetimes that were set are parsed, and the rest stay absent",
			in:     LoginPolicyInput{MFAInitSkipLifetime: "3h"},
			want:   policyDurations{mfaInitSkipLifetime: durationpb.New(10800000000000)},
		},
		"Garbage": {
			reason:   "An unparseable lifetime is reported by name",
			in:       LoginPolicyInput{MFAInitSkipLifetime: "soon"},
			wantErr:  true,
			errField: "mfaInitSkipLifetime",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := loginPolicyDurations(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\nloginPolicyDurations(...): wantErr %v, got %v", tc.reason, tc.wantErr, err)
			}

			if tc.wantErr {
				if got := err.Error(); !contains(got, tc.errField) {
					t.Errorf("\n%s\nloginPolicyDurations(...): want the error to name %q, got %q", tc.reason, tc.errField, got)
				}

				return
			}

			if diff := cmp.Diff(tc.want, got, cmpOpts()); diff != "" {
				t.Errorf("\n%s\nloginPolicyDurations(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestApplyDurationsToAdd(t *testing.T) {
	d := policyDurations{
		passwordCheckLifetime:      durationpb.New(1000000000),
		externalLoginCheckLifetime: durationpb.New(2000000000),
		mfaInitSkipLifetime:        durationpb.New(3000000000),
		secondFactorCheckLifetime:  durationpb.New(4000000000),
		multiFactorCheckLifetime:   durationpb.New(5000000000),
	}

	req := &managementv1.AddCustomLoginPolicyRequest{}
	applyDurationsToAdd(req, d)

	// Every lifetime has to reach the request: one left behind is a lifetime the
	// operator asked for that ZITADEL never hears about.
	for name, tc := range map[string]struct{ got, want *durationpb.Duration }{
		"PasswordCheckLifetime":      {req.GetPasswordCheckLifetime(), d.passwordCheckLifetime},
		"ExternalLoginCheckLifetime": {req.GetExternalLoginCheckLifetime(), d.externalLoginCheckLifetime},
		"MfaInitSkipLifetime":        {req.GetMfaInitSkipLifetime(), d.mfaInitSkipLifetime},
		"SecondFactorCheckLifetime":  {req.GetSecondFactorCheckLifetime(), d.secondFactorCheckLifetime},
		"MultiFactorCheckLifetime":   {req.GetMultiFactorCheckLifetime(), d.multiFactorCheckLifetime},
	} {
		if diff := cmp.Diff(tc.want, tc.got, cmpOpts()); diff != "" {
			t.Errorf("%s: -want, +got:\n%s", name, diff)
		}
	}

	// An absent lifetime stays absent rather than being written as zero.
	empty := &managementv1.AddCustomLoginPolicyRequest{}
	applyDurationsToAdd(empty, policyDurations{})

	if empty.GetPasswordCheckLifetime() != nil {
		t.Error("an absent lifetime was written as an explicit duration")
	}
}

// TestSecondFactors covers the second factor mapping in both directions.
func TestSecondFactors(t *testing.T) {
	names := map[string]policyv1.SecondFactorType{
		"OTP":           policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP,
		"U2F":           policyv1.SecondFactorType_SECOND_FACTOR_TYPE_U2F,
		"OTPEmail":      policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP_EMAIL,
		"OTPSMS":        policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP_SMS,
		"RecoveryCodes": policyv1.SecondFactorType_SECOND_FACTOR_TYPE_RECOVERY_CODES,
	}

	for name, want := range names {
		t.Run(name, func(t *testing.T) {
			got, err := SecondFactorToProto(name)
			if err != nil {
				t.Fatalf("SecondFactorToProto(%q): unexpected error: %v", name, err)
			}

			if got != want {
				t.Errorf("SecondFactorToProto(%q) = %v, want %v", name, got, want)
			}

			back, err := SecondFactorNames([]policyv1.SecondFactorType{want})
			if err != nil {
				t.Fatalf("SecondFactorNames(...): unexpected error: %v", err)
			}

			if len(back) != 1 || back[0] != name {
				t.Errorf("round trip: want %q, got %v", name, back)
			}
		})
	}

	t.Run("Unknown", func(t *testing.T) {
		if _, err := SecondFactorToProto("Telepathy"); err == nil {
			t.Error("an unknown second factor was accepted")
		}

		if _, err := SecondFactorNames([]policyv1.SecondFactorType{policyv1.SecondFactorType(99)}); err == nil {
			t.Error("an unknown second factor enum was accepted")
		}
	})
}

// TestMultiFactors covers the multi factor mapping in both directions.
func TestMultiFactors(t *testing.T) {
	t.Run("U2FWithVerification", func(t *testing.T) {
		got, err := MultiFactorToProto("U2FWithVerification")
		if err != nil {
			t.Fatalf("MultiFactorToProto(...): unexpected error: %v", err)
		}

		want := policyv1.MultiFactorType_MULTI_FACTOR_TYPE_U2F_WITH_VERIFICATION
		if got != want {
			t.Errorf("MultiFactorToProto(...) = %v, want %v", got, want)
		}

		back, err := MultiFactorNames([]policyv1.MultiFactorType{want})
		if err != nil {
			t.Fatalf("MultiFactorNames(...): unexpected error: %v", err)
		}

		if len(back) != 1 || back[0] != "U2FWithVerification" {
			t.Errorf("round trip: want %q, got %v", "U2FWithVerification", back)
		}
	})

	t.Run("NotAllowed", func(t *testing.T) {
		// "NotAllowed" and the empty string both mean the multi factor is off,
		// which is ZITADEL's unspecified value rather than a factor.
		for _, in := range []string{"", "NotAllowed"} {
			got, err := MultiFactorToProto(in)
			if err != nil {
				t.Fatalf("MultiFactorToProto(%q): unexpected error: %v", in, err)
			}

			if got != policyv1.MultiFactorType_MULTI_FACTOR_TYPE_UNSPECIFIED {
				t.Errorf("MultiFactorToProto(%q) = %v, want unspecified", in, got)
			}
		}
	})

	t.Run("UnsetIsDropped", func(t *testing.T) {
		// An unspecified factor carries no name, so it contributes nothing to the
		// list rather than an empty entry.
		got, err := SecondFactorNames([]policyv1.SecondFactorType{
			policyv1.SecondFactorType_SECOND_FACTOR_TYPE_UNSPECIFIED,
			policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP,
		})
		if err != nil {
			t.Fatalf("SecondFactorNames(...): unexpected error: %v", err)
		}

		if len(got) != 1 || got[0] != "OTP" {
			t.Errorf("SecondFactorNames(...): want only the OTP, got %v", got)
		}
	})

	t.Run("Unknown", func(t *testing.T) {
		if _, err := MultiFactorToProto("Telepathy"); err == nil {
			t.Error("an unknown multi factor was accepted")
		}

		if _, err := MultiFactorNames([]policyv1.MultiFactorType{policyv1.MultiFactorType(99)}); err == nil {
			t.Error("an unknown multi factor enum was accepted")
		}
	})
}

// TestPasswordlessType covers the one field where an empty input is meaningful:
// it means "not allowed", which is what ZITADEL itself reports.
func TestPasswordlessType(t *testing.T) {
	cases := map[string]struct {
		reason  string
		in      string
		want    policyv1.PasswordlessType
		wantErr bool
	}{
		"Empty": {
			reason: "An unset passwordless type means not allowed, which is what ZITADEL reports",
			in:     "",
			want:   policyv1.PasswordlessType_PASSWORDLESS_TYPE_NOT_ALLOWED,
		},
		"NotAllowed": {
			reason: "An explicit not allowed is kept",
			in:     "NotAllowed",
			want:   policyv1.PasswordlessType_PASSWORDLESS_TYPE_NOT_ALLOWED,
		},
		"Allowed": {
			reason: "Passwordless is kept",
			in:     "Allowed",
			want:   policyv1.PasswordlessType_PASSWORDLESS_TYPE_ALLOWED,
		},
		"Unknown": {
			reason:  "An unknown value is rejected rather than silently treated as not allowed",
			in:      "Maybe",
			want:    policyv1.PasswordlessType_PASSWORDLESS_TYPE_NOT_ALLOWED,
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := passwordlessType(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\npasswordlessType(%q): wantErr %v, got %v", tc.reason, tc.in, tc.wantErr, err)
			}

			if got != tc.want {
				t.Errorf("\n%s\npasswordlessType(%q): want %v, got %v", tc.reason, tc.in, tc.want, got)
			}
		})
	}

	t.Run("FromProto", func(t *testing.T) {
		for in, want := range map[policyv1.PasswordlessType]string{
			policyv1.PasswordlessType_PASSWORDLESS_TYPE_ALLOWED:     "Allowed",
			policyv1.PasswordlessType_PASSWORDLESS_TYPE_NOT_ALLOWED: "NotAllowed",
			// The generated enum has no unspecified member: zero is NOT_ALLOWED.
			// A value from a newer ZITADEL falls through to unmanaged rather than
			// being misreported as something.
			policyv1.PasswordlessType(99): "",
		} {
			if got := passwordlessTypeFromProto(in); got != want {
				t.Errorf("passwordlessTypeFromProto(%v): want %q, got %q", in, want, got)
			}
		}
	})
}

// TestFromProtoLoginPolicy covers the whole read side, including the fallback
// that keeps an unrecognised factor visible rather than failing the observation
// and leaving the resource stuck.
func TestFromProtoLoginPolicy(t *testing.T) {
	t.Run("Full", func(t *testing.T) {
		in := &policyv1.LoginPolicy{
			AllowUsernamePassword:      true,
			AllowRegister:              true,
			AllowExternalIdp:           true,
			ForceMfa:                   true,
			ForceMfaLocalOnly:          true,
			HidePasswordReset:          true,
			IgnoreUnknownUsernames:     true,
			AllowDomainDiscovery:       true,
			DisableLoginWithEmail:      true,
			DisableLoginWithPhone:      true,
			DefaultRedirectUri:         "https://app.example.com",
			PasswordlessType:           policyv1.PasswordlessType_PASSWORDLESS_TYPE_ALLOWED,
			PasswordCheckLifetime:      durationpb.New(3600000000000),
			ExternalLoginCheckLifetime: durationpb.New(7200000000000),
			MfaInitSkipLifetime:        durationpb.New(10800000000000),
			SecondFactorCheckLifetime:  durationpb.New(14400000000000),
			MultiFactorCheckLifetime:   durationpb.New(18000000000000),
			SecondFactors: []policyv1.SecondFactorType{
				policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP,
				policyv1.SecondFactorType_SECOND_FACTOR_TYPE_RECOVERY_CODES,
			},
			MultiFactors: []policyv1.MultiFactorType{
				policyv1.MultiFactorType_MULTI_FACTOR_TYPE_U2F_WITH_VERIFICATION,
			},
			Idps: []*idpv1.IDPLoginPolicyLink{{IdpId: "idp-1"}, {IdpId: "idp-2"}},
		}

		got := fromProtoLoginPolicy("org-1", in, false)

		if got.OrgID != "org-1" {
			t.Errorf("OrgID: want %q, got %q", "org-1", got.OrgID)
		}

		if got.IsDefault {
			t.Error("IsDefault: want false")
		}

		for name, tc := range map[string]struct{ got, want string }{
			"DefaultRedirectURI":         {got.DefaultRedirectURI, "https://app.example.com"},
			"PasswordlessType":           {got.PasswordlessType, "Allowed"},
			"PasswordCheckLifetime":      {got.PasswordCheckLifetime, "1h0m0s"},
			"ExternalLoginCheckLifetime": {got.ExternalLoginCheckLifetime, "2h0m0s"},
			"MFAInitSkipLifetime":        {got.MFAInitSkipLifetime, "3h0m0s"},
			"SecondFactorCheckLifetime":  {got.SecondFactorCheckLifetime, "4h0m0s"},
			"MultiFactorCheckLifetime":   {got.MultiFactorCheckLifetime, "5h0m0s"},
		} {
			if tc.got != tc.want {
				t.Errorf("%s: want %q, got %q", name, tc.want, tc.got)
			}
		}

		for name, tc := range map[string]struct{ got, want bool }{
			"AllowUsernamePassword":  {got.AllowUsernamePassword, true},
			"AllowRegister":          {got.AllowRegister, true},
			"AllowExternalIDP":       {got.AllowExternalIDP, true},
			"ForceMFA":               {got.ForceMFA, true},
			"ForceMFALocalOnly":      {got.ForceMFALocalOnly, true},
			"HidePasswordReset":      {got.HidePasswordReset, true},
			"IgnoreUnknownUsernames": {got.IgnoreUnknownUsernames, true},
			"AllowDomainDiscovery":   {got.AllowDomainDiscovery, true},
			"DisableLoginWithEmail":  {got.DisableLoginWithEmail, true},
			"DisableLoginWithPhone":  {got.DisableLoginWithPhone, true},
		} {
			if tc.got != tc.want {
				t.Errorf("%s: want %v, got %v", name, tc.want, tc.got)
			}
		}

		if diff := cmp.Diff([]string{"OTP", "RecoveryCodes"}, got.SecondFactors); diff != "" {
			t.Errorf("SecondFactors: -want, +got:\n%s", diff)
		}

		if diff := cmp.Diff([]string{"U2FWithVerification"}, got.MultiFactors); diff != "" {
			t.Errorf("MultiFactors: -want, +got:\n%s", diff)
		}

		if diff := cmp.Diff([]string{"idp-1", "idp-2"}, got.IDPs); diff != "" {
			t.Errorf("IDPs: -want, +got:\n%s", diff)
		}
	})

	t.Run("Empty", func(t *testing.T) {
		got := fromProtoLoginPolicy("org-1", &policyv1.LoginPolicy{}, true)

		if !got.IsDefault {
			t.Error("IsDefault: want true, as it was passed in")
		}

		// The generated enum has no unspecified member, so its zero value is
		// NotAllowed and that is what an unset policy reads back as.
		if got.PasswordlessType != "NotAllowed" {
			t.Errorf("PasswordlessType: want %q for an unset policy, got %q", "NotAllowed", got.PasswordlessType)
		}

		if len(got.IDPs) != 0 {
			t.Errorf("IDPs: want none, got %v", got.IDPs)
		}
	})

	t.Run("UnknownFactorIsSurfaced", func(t *testing.T) {
		// ZITADEL can report a factor this build does not know about. Failing the
		// whole observation would leave the resource permanently unready, so the
		// value is reported verbatim instead.
		got := fromProtoLoginPolicy("org-1", &policyv1.LoginPolicy{
			SecondFactors: []policyv1.SecondFactorType{
				policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP,
				policyv1.SecondFactorType(99),
			},
			MultiFactors: []policyv1.MultiFactorType{policyv1.MultiFactorType(99)},
		}, false)

		// Nothing is lost and the observation still succeeds, which is the point:
		// a factor this build does not know about must not leave the resource
		// permanently unready. Note that the fallback re-spells every factor as
		// its protobuf constant, so the known one changes spelling too - which
		// will read as drift on it.
		if len(got.SecondFactors) != 2 {
			t.Errorf("SecondFactors: want both factors, got %v", got.SecondFactors)
		}

		if got.SecondFactors[0] != policyv1.SecondFactorType_SECOND_FACTOR_TYPE_OTP.String() {
			t.Errorf("SecondFactors: want the known factor reported verbatim, got %v", got.SecondFactors)
		}

		if len(got.MultiFactors) != 1 {
			t.Errorf("MultiFactors: want the unknown value reported, got %v", got.MultiFactors)
		}
	})
}

// cmpOpts compares the lifetimes by value rather than by identity, which is what
// a duration is meant to be.
//
// Two things are needed: the durations inside are protobufs, whose generated
// structs carry unexported state, and policyDurations itself is unexported.
func cmpOpts() cmp.Option {
	return cmp.Options{
		protocmp.Transform(),
		cmp.AllowUnexported(policyDurations{}),
	}
}
