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

package password_complexity_policy

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Every policy in this provider is driven the same way, and the drift decision
// is the part that decides whether a reconcile writes or waits.
//
// The case that matters most is the one a manifest produces by not mentioning a
// field. Desired has to turn an unset field into something, and it turns it into
// the zero value; comparing that against whatever Zitadel happens to hold would
// report drift on every poll, forever, for a field the operator never asked to
// have managed. So an unset field is not compared, and these tests pin that down
// field by field.

// ptrTo names the type it returns, which matters for the fields that are
// defined string types rather than plain strings.
func ptrTo[T any](v T) *T { return &v }

// setAll asks for a value on every field, so a case can clear exactly one and
// leave the rest managed.
func setAll() v1alpha1.PasswordComplexityPolicyParameters {
	return v1alpha1.PasswordComplexityPolicyParameters{
		HasUppercase: ptrTo(true),
		HasLowercase: ptrTo(true),
		HasNumber:    ptrTo(true),
		HasSymbol:    ptrTo(true),
		MinLength:    ptrTo(int64(5)),
	}
}

// matching holds exactly what setAll asks for, so the all-managed case has
// something to agree with.
var matching = zitadel.PasswordComplexityPolicy{
	HasUppercase: true,
	HasLowercase: true,
	HasNumber:    true,
	HasSymbol:    true,
	MinLength:    5,
}

// observedAll holds a different value on every field, which is what makes an
// unmanaged field show up as drift if the driver is wrong, and what makes a
// managed one unambiguously drift.
var observedAll = zitadel.PasswordComplexityPolicy{
	HasUppercase: false,
	HasLowercase: false,
	HasNumber:    false,
	HasSymbol:    false,
	MinLength:    6,
}

// TestEqual checks the drift decision for every field, set and unset.
func TestEqual(t *testing.T) {
	var d driver

	all := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: setAll()}}
	if !d.Equal(all, d.Desired(all), matching) {
		t.Error("Equal(...): a policy matching every field the manifest set is drift")
	}

	t.Run("NothingManaged", func(t *testing.T) {
		// The regression: a manifest that says nothing manages nothing, so
		// Zitadel's own values are not something to correct.
		empty := &v1alpha1.PasswordComplexityPolicy{}

		if !d.Equal(empty, d.Desired(empty), observedAll) {
			t.Error("Equal(...): a policy that manages no fields is not drift, whatever Zitadel holds")
		}
	})

	t.Run("HasUppercaseUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.HasUppercase = nil
		only := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.HasUppercase = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset HasUppercase is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("HasUppercaseDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PasswordComplexityPolicy{}
		obs.HasUppercase = false
		only := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed HasUppercase that Zitadel does not hold is drift")
		}
	})

	t.Run("HasLowercaseUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.HasLowercase = nil
		only := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.HasLowercase = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset HasLowercase is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("HasLowercaseDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PasswordComplexityPolicy{}
		obs.HasLowercase = false
		only := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed HasLowercase that Zitadel does not hold is drift")
		}
	})

	t.Run("HasNumberUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.HasNumber = nil
		only := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.HasNumber = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset HasNumber is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("HasNumberDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PasswordComplexityPolicy{}
		obs.HasNumber = false
		only := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed HasNumber that Zitadel does not hold is drift")
		}
	})

	t.Run("HasSymbolUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.HasSymbol = nil
		only := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.HasSymbol = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset HasSymbol is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("HasSymbolDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PasswordComplexityPolicy{}
		obs.HasSymbol = false
		only := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed HasSymbol that Zitadel does not hold is drift")
		}
	})

	t.Run("MinLengthUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.MinLength = nil
		only := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.MinLength = 6

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset MinLength is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("MinLengthDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PasswordComplexityPolicy{}
		obs.MinLength = 6
		only := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed MinLength that Zitadel does not hold is drift")
		}
	})

}

// TestAskingForZeroIsAskingForZero checks that an explicit zero and an omission
// are not the same request. Only the second leaves Zitadel alone.
func TestAskingForZeroIsAskingForZero(t *testing.T) {
	var d driver

	askedForZero := &v1alpha1.PasswordComplexityPolicy{Spec: v1alpha1.PasswordComplexityPolicySpec{
		ForProvider: v1alpha1.PasswordComplexityPolicyParameters{HasUppercase: ptrTo(false)},
	}}

	if d.Equal(askedForZero, d.Desired(askedForZero), matching) {
		t.Error("Equal(...): a field the manifest asked for and Zitadel does not hold is drift")
	}

	neverAsked := &v1alpha1.PasswordComplexityPolicy{}

	if !d.Equal(neverAsked, d.Desired(neverAsked), matching) {
		t.Errorf("Equal(...): HasUppercase was never asked for, so Zitadel's value is not drift")
	}
}

// TestResettable pins the answer about whether deleting this resource can put
// Zitadel back the way it was.
func TestResettable(t *testing.T) {
	var d driver

	if d.Resettable() != true {
		t.Errorf("Resettable(): want %v, got %v", true, d.Resettable())
	}
}

// TestResetReachesZitadel checks that a resettable policy does have somewhere to
// reset to. The call itself needs an instance, so only the refusal is checked:
// what matters here is that the driver offers the path at all.
func TestResetReachesZitadel(t *testing.T) {
	var d driver

	// A nil client is the cheapest way to prove the driver did not quietly turn
	// a reset into a no-op: something has to be called on it.
	if err := d.Reset(t.Context(), nil, ""); err == nil {
		t.Error("Reset(...): want the call to be attempted rather than silently skipped")
	}
}
