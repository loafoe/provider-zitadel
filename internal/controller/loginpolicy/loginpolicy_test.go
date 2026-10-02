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

package loginpolicy

import (
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/google/go-cmp/cmp"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

func TestDesiredAppliesDefaults(t *testing.T) {
	cases := map[string]struct {
		reason string
		fp     v1alpha1.LoginPolicyParameters
		want   zitadel.LoginPolicyInput
	}{
		"Unset": {
			reason: "Every unset field falls back to the documented default",
			want: zitadel.LoginPolicyInput{
				AllowUsernamePassword:  true,
				AllowRegister:          false,
				AllowExternalIDP:       true,
				IgnoreUnknownUsernames: true,
				PasswordlessType:       string(v1alpha1.PasswordlessTypeNotAllowed),
			},
		},
		"ExplicitFalseIsNotTheDefault": {
			reason: "An explicit false is respected, not overwritten by a true default",
			fp: v1alpha1.LoginPolicyParameters{
				AllowUsernamePassword: ptr(false),
				AllowExternalIDP:      ptr(false),
			},
			want: zitadel.LoginPolicyInput{
				AllowUsernamePassword:  false,
				AllowRegister:          false,
				AllowExternalIDP:       false,
				IgnoreUnknownUsernames: true,
				PasswordlessType:       string(v1alpha1.PasswordlessTypeNotAllowed),
			},
		},
		"RegisterEnabled": {
			reason: "Self registration can be turned on",
			fp:     v1alpha1.LoginPolicyParameters{AllowRegister: ptr(true)},
			want: zitadel.LoginPolicyInput{
				AllowUsernamePassword:  true,
				AllowRegister:          true,
				AllowExternalIDP:       true,
				IgnoreUnknownUsernames: true,
				PasswordlessType:       string(v1alpha1.PasswordlessTypeNotAllowed),
			},
		},
		"Factors": {
			reason: "Factors are passed through using the API spelling",
			fp: v1alpha1.LoginPolicyParameters{
				SecondFactors: []v1alpha1.SecondFactor{v1alpha1.SecondFactorOTP},
				MultiFactors:  []v1alpha1.MultiFactor{v1alpha1.MultiFactorU2FWithVerification},
			},
			want: zitadel.LoginPolicyInput{
				AllowUsernamePassword:  true,
				AllowExternalIDP:       true,
				IgnoreUnknownUsernames: true,
				PasswordlessType:       string(v1alpha1.PasswordlessTypeNotAllowed),
				SecondFactors:          []string{"OTP"},
				MultiFactors:           []string{"U2FWithVerification"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := desired(&v1alpha1.LoginPolicy{Spec: v1alpha1.LoginPolicySpec{ForProvider: tc.fp}})
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\n%s\ndesired(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

// The values the drift table keeps repeating, so the table stays about what
// differs rather than about boilerplate.
const (
	testLifetime = "8760h"
	testIDP      = "idp1"
)

func matchingPolicy(p v1alpha1.LoginPolicyParameters) v1alpha1.LoginPolicyParameters {
	p.AllowRegister = ptr(true)
	p.PasswordCheckLifetime = ptr(testLifetime)
	p.SecondFactors = []v1alpha1.SecondFactor{v1alpha1.SecondFactorOTP, v1alpha1.SecondFactorU2F}
	p.MultiFactors = []v1alpha1.MultiFactor{v1alpha1.MultiFactorU2FWithVerification}
	p.IDPs = []string{testIDP}

	return p
}

func TestSamePolicy(t *testing.T) {
	observed := &zitadel.LoginPolicy{
		AllowUsernamePassword:  true,
		AllowRegister:          true,
		AllowExternalIDP:       true,
		IgnoreUnknownUsernames: true,
		PasswordlessType:       string(v1alpha1.PasswordlessTypeNotAllowed),
		PasswordCheckLifetime:  testLifetime,
		SecondFactors:          []string{"OTP", "U2F"},
		MultiFactors:           []string{"U2FWithVerification"},
		IDPs:                   []string{testIDP},
	}

	cases := map[string]struct {
		reason string
		fp     v1alpha1.LoginPolicyParameters
		want   bool
	}{
		"Matching": {
			reason: "A policy matching the defaults and the reported values is up to date",
			fp:     matchingPolicy(v1alpha1.LoginPolicyParameters{}),
			want:   true,
		},
		"EquivalentDuration": {
			reason: "A duration Zitadel normalised to another spelling is not drift. Go durations have no day unit, so this uses minutes against the hours Zitadel reports.",
			fp:     withLifetime(matchingPolicy(v1alpha1.LoginPolicyParameters{}), "525600m"),
			want:   true,
		},
		"RegisterDrift": {
			reason: "Turning self registration off is drift",
			fp: v1alpha1.LoginPolicyParameters{
				AllowRegister:         ptr(false),
				PasswordCheckLifetime: ptr("8760h"),
				SecondFactors:         []v1alpha1.SecondFactor{v1alpha1.SecondFactorOTP},
				MultiFactors:          []v1alpha1.MultiFactor{v1alpha1.MultiFactorU2FWithVerification},
				IDPs:                  []string{"idp1"},
			},
			want: false,
		},
		"DurationDrift": {
			reason: "A genuinely different lifetime is drift",
			fp: v1alpha1.LoginPolicyParameters{
				AllowRegister:         ptr(true),
				PasswordCheckLifetime: ptr("24h"),
				SecondFactors:         []v1alpha1.SecondFactor{v1alpha1.SecondFactorOTP},
				MultiFactors:          []v1alpha1.MultiFactor{v1alpha1.MultiFactorU2FWithVerification},
				IDPs:                  []string{"idp1"},
			},
			want: false,
		},
		"FactorDrift": {
			reason: "A factor the remote does not have is drift",
			fp:     factorDrift(matchingPolicy(v1alpha1.LoginPolicyParameters{})),
			want:   false,
		},
		"FactorOrderIrrelevant": {
			reason: "Factors are a set, so listing them in another order is not drift",
			fp:     withFactors(matchingPolicy(v1alpha1.LoginPolicyParameters{}), v1alpha1.SecondFactorU2F, v1alpha1.SecondFactorOTP),
			want:   true,
		},
		"IDPDrift": {
			reason: "An identity provider that is no longer offered is drift",
			fp:     idpDrift(matchingPolicy(v1alpha1.LoginPolicyParameters{})),
			want:   false,
		},
		"PasswordlessDrift": {
			reason: "Allowing passwordless login is drift",
			fp: v1alpha1.LoginPolicyParameters{
				AllowRegister:         ptr(true),
				PasswordlessType:      ptr(v1alpha1.PasswordlessTypeAllowed),
				PasswordCheckLifetime: ptr("8760h"),
				SecondFactors:         []v1alpha1.SecondFactor{v1alpha1.SecondFactorOTP},
				MultiFactors:          []v1alpha1.MultiFactor{v1alpha1.MultiFactorU2FWithVerification},
				IDPs:                  []string{"idp1"},
			},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.LoginPolicy{Spec: v1alpha1.LoginPolicySpec{ForProvider: tc.fp}}
			if got := samePolicy(desired(cr), observed); got != tc.want {
				t.Errorf("\n%ssamePolicy(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestSameDuration(t *testing.T) {
	cases := map[string]struct {
		reason string
		a      string
		b      string
		want   bool
	}{
		"Identical": {reason: "Identical strings are equal", a: "5s", b: "5s", want: true},
		"Equivalent": {
			reason: "Two spellings of the same duration are equal",
			a:      "300s",
			b:      "5m",
			want:   true,
		},
		"Different": {reason: "Different durations are not equal", a: "5s", b: "10s"},
		"EmptyLeft": {reason: "An unset duration never matches a set one", a: "", b: "5s"},
		"Garbage":   {reason: "Garbage does not match anything", a: "soon", b: "5s"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := sameDuration(tc.a, tc.b); got != tc.want {
				t.Errorf("\n%s\nsameDuration(%q, %q): want %v, got %v", tc.reason, tc.a, tc.b, tc.want, got)
			}
		})
	}
}

func TestSameStringSet(t *testing.T) {
	cases := map[string]struct {
		reason string
		a      []string
		b      []string
		want   bool
	}{
		"OrderIrrelevant": {
			reason: "Sets are compared without order",
			a:      []string{"b", "a"},
			b:      []string{"a", "b"},
			want:   true,
		},
		"ExtraEntry": {reason: "A different number of entries is a difference", a: []string{"a"}, b: []string{"a", "b"}},
		"BothEmpty":  {reason: "Two empty sets are equal", want: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := sameStringSet(tc.a, tc.b); got != tc.want {
				t.Errorf("\n%s\nsameStringSet(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func withLifetime(p v1alpha1.LoginPolicyParameters, v string) v1alpha1.LoginPolicyParameters {
	p.PasswordCheckLifetime = ptr(v)
	return p
}

// idpDrift removes an identity provider the remote still offers.
func idpDrift(p v1alpha1.LoginPolicyParameters) v1alpha1.LoginPolicyParameters {
	p.IDPs = nil
	return p
}

// factorDrift adds a second factor the remote does not have.
func factorDrift(p v1alpha1.LoginPolicyParameters) v1alpha1.LoginPolicyParameters {
	p.SecondFactors = append(p.SecondFactors, v1alpha1.SecondFactorOTPSMS)
	return p
}

// withFactors replaces the second factors with the given list.
func withFactors(p v1alpha1.LoginPolicyParameters, factors ...v1alpha1.SecondFactor) v1alpha1.LoginPolicyParameters {
	p.SecondFactors = factors
	return p
}

func ptr[T any](v T) *T { return &v }

// An unset lifetime must compare equal to the zero Zitadel reports, and a
// policy that matches the spec must be seen as up to date. Getting this wrong
// puts the controller in a loop: it would keep asking Zitadel to update a
// policy Zitadel already matches, and Zitadel refuses every such update.
func TestIsUpToDateAgainstObservedPolicy(t *testing.T) {
	observed := &zitadel.LoginPolicy{
		AllowDomainDiscovery: false, AllowExternalIDP: true, AllowRegister: false,
		AllowUsernamePassword: true, DefaultRedirectURI: "", DisableLoginWithEmail: false,
		DisableLoginWithPhone: false, ExternalLoginCheckLifetime: "0s", ForceMFA: false,
		ForceMFALocalOnly: false, HidePasswordReset: false, IgnoreUnknownUsernames: true,
		MFAInitSkipLifetime: "0s", MultiFactorCheckLifetime: "0s",
		PasswordCheckLifetime: "8760h0m0s", PasswordlessType: "NotAllowed",
		SecondFactorCheckLifetime: "0s", SecondFactors: []string{"OTP", "OTPEmail"},
	}

	cr := &v1alpha1.LoginPolicy{Spec: v1alpha1.LoginPolicySpec{
		ForProvider: v1alpha1.LoginPolicyParameters{
			OrganizationRef:        &xpv1.Reference{Name: "platform"},
			AllowRegister:          ptr(false),
			IgnoreUnknownUsernames: ptr(true),
			PasswordCheckLifetime:  ptr("8760h"),
			SecondFactors: []v1alpha1.SecondFactor{
				v1alpha1.SecondFactorOTP, v1alpha1.SecondFactorOTPEmail,
			},
		},
	}}

	if !samePolicy(desired(cr), observed) {
		t.Errorf("policy matching the spec reported as drift:\nwant %+v\ngot  %+v", desired(cr), *observed)
	}

	// A real change must still be detected.
	observed.ForceMFA = true
	if samePolicy(desired(cr), observed) {
		t.Error("a changed forceMFA reported as up to date")
	}
}

func TestSameLifetime(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{a: "", b: "0s", want: true},
		{a: "8760h", b: "8760h0m0s", want: true},
		{a: "8760h", b: "720h", want: false},
		{a: "", b: "24h", want: false},
		{a: "nonsense", b: "", want: false},
	} {
		if got := sameDuration(tc.a, tc.b); got != tc.want {
			t.Errorf("sameDuration(%q, %q) = %t, want %t", tc.a, tc.b, got, tc.want)
		}
	}
}
