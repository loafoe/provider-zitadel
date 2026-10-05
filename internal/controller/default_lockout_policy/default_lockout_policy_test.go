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

package default_lockout_policy

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Every instance wide policy is driven the same way: the driver reads the
// desired policy out of the spec, reports what Zitadel holds into the status,
// and decides whether the two already agree. The third of those decides whether
// the controller writes on a poll or waits, so it is what these cover.

func i64(v int64) *int64 { return &v }

// withSpec builds a resource whose spec sets the fields it is given, and leaves
// every other one unset - which is what an incomplete manifest looks like.
func withSpec(fp v1alpha1.DefaultLockoutPolicyParameters) *v1alpha1.DefaultLockoutPolicy {
	return &v1alpha1.DefaultLockoutPolicy{Spec: v1alpha1.DefaultLockoutPolicySpec{ForProvider: fp}}
}

// TestEqual covers the drift decision, and specifically what happens to a field
// the manifest never set.
func TestEqual(t *testing.T) {
	cases := map[string]struct {
		reason   string
		fp       v1alpha1.DefaultLockoutPolicyParameters
		observed zitadel.DefaultLockoutPolicy
		equal    bool
	}{
		"BothSetAndMatching": {
			reason: "A policy that already matches needs no write",
			fp:     v1alpha1.DefaultLockoutPolicyParameters{MaxPasswordAttempts: i64(5), MaxOTPAttempts: i64(3)},
			observed: zitadel.DefaultLockoutPolicy{
				MaxPasswordAttempts: 5, MaxOTPAttempts: 3,
			},
			equal: true,
		},
		"PasswordDrift": {
			reason: "A differing attempt count the manifest asked for is drift",
			fp:     v1alpha1.DefaultLockoutPolicyParameters{MaxPasswordAttempts: i64(5)},
			observed: zitadel.DefaultLockoutPolicy{
				MaxPasswordAttempts: 6, MaxOTPAttempts: 3,
			},
			equal: false,
		},
		"OTPDrift": {
			reason: "A differing one time password count is drift",
			fp:     v1alpha1.DefaultLockoutPolicyParameters{MaxOTPAttempts: i64(3)},
			observed: zitadel.DefaultLockoutPolicy{
				MaxPasswordAttempts: 5, MaxOTPAttempts: 4,
			},
			equal: false,
		},
		"UnsetIsNotDrift": {
			// This is the case the driver used to get wrong. Desired turns an
			// unset field into zero, so comparing it against Zitadel's own value
			// would report drift on every poll for a manifest that never asked
			// for the field to be managed.
			reason: "A field the manifest left unset is Zitadel's to decide, not drift",
			fp:     v1alpha1.DefaultLockoutPolicyParameters{},
			observed: zitadel.DefaultLockoutPolicy{
				MaxPasswordAttempts: 5, MaxOTPAttempts: 3,
			},
			equal: true,
		},
		"OneFieldSetOneUnset": {
			// The important half case: one field is managed, the other is not, so
			// only the managed one can be drift.
			reason: "An unset field alongside a managed one is still not compared",
			fp:     v1alpha1.DefaultLockoutPolicyParameters{MaxOTPAttempts: i64(3)},
			observed: zitadel.DefaultLockoutPolicy{
				MaxPasswordAttempts: 5, MaxOTPAttempts: 3,
			},
			equal: true,
		},
		"OneFieldSetOneUnsetWithDrift": {
			reason: "The managed field is still compared when its sibling is unset",
			fp:     v1alpha1.DefaultLockoutPolicyParameters{MaxOTPAttempts: i64(3)},
			observed: zitadel.DefaultLockoutPolicy{
				MaxPasswordAttempts: 5, MaxOTPAttempts: 9,
			},
			equal: false,
		},
		"ExplicitZeroIsDrift": {
			// Asking for zero and Zitadel holding five is a real difference: the
			// manifest did ask, and zero is not what Zitadel has.
			reason: "Asking for zero explicitly is drift when Zitadel holds something else",
			fp:     v1alpha1.DefaultLockoutPolicyParameters{MaxPasswordAttempts: i64(0)},
			observed: zitadel.DefaultLockoutPolicy{
				MaxPasswordAttempts: 5, MaxOTPAttempts: 3,
			},
			equal: false,
		},
		"ExplicitZeroMatches": {
			reason: "Asking for zero and Zitadel holding zero agrees",
			fp:     v1alpha1.DefaultLockoutPolicyParameters{MaxPasswordAttempts: i64(0)},
			observed: zitadel.DefaultLockoutPolicy{
				MaxOTPAttempts: 3,
			},
			equal: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := withSpec(tc.fp)

			var d driver

			if got := d.Equal(cr, d.Desired(cr), tc.observed); got != tc.equal {
				t.Errorf("\n%s\nEqual(...): want %v, got %v", tc.reason, tc.equal, got)
			}
		})
	}
}

// TestEqualUsesTheSpecNotTheInput checks the two halves agree on what was asked.
//
// Equal is handed both the resource and the value Desired derived from it. They
// have to be read from the same place, or a field could be managed that the
// manifest never mentioned - or worse, the other way round.
func TestEqualUsesTheSpecNotTheInput(t *testing.T) {
	fp := v1alpha1.DefaultLockoutPolicyParameters{MaxPasswordAttempts: i64(5)}

	cr := withSpec(fp)

	var d driver

	in := d.Desired(cr)
	if in.MaxPasswordAttempts != 5 {
		t.Errorf("Desired(...): want %d, got %d", 5, in.MaxPasswordAttempts)
	}

	// The unset field arrives in the input as zero, and must stay uncompared.
	if d.Equal(cr, in, zitadel.DefaultLockoutPolicy{MaxPasswordAttempts: 5, MaxOTPAttempts: 3}) {
		return
	}

	t.Error("Equal(...): the unset one time password count was compared against Zitadel's")
}

// TestResettableAndReset pins the answer about whether deleting this resource can
// put Zitadel back the way it was.
//
// It is false for every instance wide policy, because Zitadel has no endpoint
// that clears one; deletion instead restores whatever the policy overwrote. The
// Reset error exists so that a mistake reaches the user as a message rather than
// as a silent no-op.
func TestResettableAndReset(t *testing.T) {
	var d driver

	if d.Resettable() {
		t.Error("Resettable(): an instance wide policy cannot be reset, but the driver claims it can")
	}

	if err := d.Reset(t.Context(), nil, ""); err == nil {
		t.Error("Reset(...): want an error explaining that the policy cannot be reset, got nil")
	}
}

// TestReportWritesTheStatus checks that what Zitadel holds is what the status
// ends up saying.
func TestReportWritesTheStatus(t *testing.T) {
	cr := &v1alpha1.DefaultLockoutPolicy{}

	(driver{}).Report(cr, zitadel.DefaultLockoutPolicy{MaxPasswordAttempts: 5, MaxOTPAttempts: 3})

	ap := cr.Status.AtProvider

	if ap.IsDefault == nil || *ap.IsDefault {
		t.Error("IsDefault: want false, because an instance wide policy is never inherited from elsewhere")
	}

	if ap.MaxPasswordAttempts == nil || *ap.MaxPasswordAttempts != 5 {
		t.Errorf("MaxPasswordAttempts: want %d, got %v", 5, ap.MaxPasswordAttempts)
	}

	if ap.MaxOTPAttempts == nil || *ap.MaxOTPAttempts != 3 {
		t.Errorf("MaxOTPAttempts: want %d, got %v", 3, ap.MaxOTPAttempts)
	}
}
