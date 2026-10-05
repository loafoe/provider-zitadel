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

package default_notification_policy

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
func setAll() v1alpha1.DefaultNotificationPolicyParameters {
	return v1alpha1.DefaultNotificationPolicyParameters{
		PasswordChange: ptrTo(true),
	}
}

// matching holds exactly what setAll asks for, so the all-managed case has
// something to agree with.
var matching = zitadel.DefaultNotificationPolicy{
	PasswordChange: true,
}

// observedAll holds a different value on every field, which is what makes an
// unmanaged field show up as drift if the driver is wrong, and what makes a
// managed one unambiguously drift.
var observedAll = zitadel.DefaultNotificationPolicy{
	PasswordChange: false,
}

// TestEqual checks the drift decision for every field, set and unset.
func TestEqual(t *testing.T) {
	var d driver

	all := &v1alpha1.DefaultNotificationPolicy{Spec: v1alpha1.DefaultNotificationPolicySpec{ForProvider: setAll()}}
	if !d.Equal(all, d.Desired(all), matching) {
		t.Error("Equal(...): a policy matching every field the manifest set is drift")
	}

	t.Run("NothingManaged", func(t *testing.T) {
		// The regression: a manifest that says nothing manages nothing, so
		// Zitadel's own values are not something to correct.
		empty := &v1alpha1.DefaultNotificationPolicy{}

		if !d.Equal(empty, d.Desired(empty), observedAll) {
			t.Error("Equal(...): a policy that manages no fields is not drift, whatever Zitadel holds")
		}
	})

	t.Run("PasswordChangeUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.PasswordChange = nil
		only := &v1alpha1.DefaultNotificationPolicy{Spec: v1alpha1.DefaultNotificationPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.PasswordChange = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset PasswordChange is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("PasswordChangeDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultNotificationPolicy{}
		obs.PasswordChange = false
		only := &v1alpha1.DefaultNotificationPolicy{Spec: v1alpha1.DefaultNotificationPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed PasswordChange that Zitadel does not hold is drift")
		}
	})

}

// TestAskingForZeroIsAskingForZero checks that an explicit zero and an omission
// are not the same request. Only the second leaves Zitadel alone.
func TestAskingForZeroIsAskingForZero(t *testing.T) {
	var d driver

	askedForZero := &v1alpha1.DefaultNotificationPolicy{Spec: v1alpha1.DefaultNotificationPolicySpec{
		ForProvider: v1alpha1.DefaultNotificationPolicyParameters{PasswordChange: ptrTo(false)},
	}}

	if d.Equal(askedForZero, d.Desired(askedForZero), matching) {
		t.Error("Equal(...): a field the manifest asked for and Zitadel does not hold is drift")
	}

	neverAsked := &v1alpha1.DefaultNotificationPolicy{}

	if !d.Equal(neverAsked, d.Desired(neverAsked), matching) {
		t.Errorf("Equal(...): PasswordChange was never asked for, so Zitadel's value is not drift")
	}
}

// TestResettable pins the answer about whether deleting this resource can put
// Zitadel back the way it was.
func TestResettable(t *testing.T) {
	var d driver

	if d.Resettable() != false {
		t.Errorf("Resettable(): want %v, got %v", false, d.Resettable())
	}
}

// TestResetRefuses checks that an unresettable policy says so rather than
// reporting success. Deleting this resource restores the value it overwrote, so
// a reset that quietly did nothing would leave Zitadel configured by a manifest
// that no longer exists.
func TestResetRefuses(t *testing.T) {
	var d driver

	if err := d.Reset(t.Context(), nil, ""); err == nil {
		t.Error("Reset(...): want an error explaining that the policy cannot be reset, got nil")
	}
}
