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

package default_login_policy

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// A value shared by the cases, named so that a case reads as a statement about
// behaviour rather than about a repeated literal.
const (
	otherFactor = "SomethingElse"
)

// setValue and diffValue are the two spellings the cases share: one is what the
// manifest asks for, the other is what Zitadel is made to hold instead.
const (
	setValue  = "https://set.example.com/value"
	diffValue = "https://other.example.com/value"
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
func setAll() v1alpha1.DefaultLoginPolicyParameters {
	return v1alpha1.DefaultLoginPolicyParameters{
		AllowUsernamePassword:      ptrTo(true),
		AllowRegister:              ptrTo(true),
		AllowExternalIDP:           ptrTo(true),
		ForceMFA:                   ptrTo(true),
		ForceMFALocalOnly:          ptrTo(true),
		HidePasswordReset:          ptrTo(true),
		IgnoreUnknownUsernames:     ptrTo(true),
		AllowDomainDiscovery:       ptrTo(true),
		DisableLoginWithEmail:      ptrTo(true),
		DisableLoginWithPhone:      ptrTo(true),
		PasswordlessType:           ptrTo(v1alpha1.PasswordlessTypeAllowed),
		DefaultRedirectURI:         ptrTo(setValue),
		PasswordCheckLifetime:      ptrTo(setValue),
		ExternalLoginCheckLifetime: ptrTo(setValue),
		MFAInitSkipLifetime:        ptrTo(setValue),
		SecondFactorCheckLifetime:  ptrTo(setValue),
		MultiFactorCheckLifetime:   ptrTo(setValue),
		SecondFactors:              []v1alpha1.SecondFactor{v1alpha1.SecondFactorRecoveryCodes},
		MultiFactors:               []v1alpha1.MultiFactor{v1alpha1.MultiFactorU2FWithVerification},
	}
}

// matching holds exactly what setAll asks for, so the all-managed case has
// something to agree with.
var matching = zitadel.DefaultLoginPolicy{
	AllowUsernamePassword:      true,
	AllowRegister:              true,
	AllowExternalIDP:           true,
	ForceMFA:                   true,
	ForceMFALocalOnly:          true,
	HidePasswordReset:          true,
	IgnoreUnknownUsernames:     true,
	AllowDomainDiscovery:       true,
	DisableLoginWithEmail:      true,
	DisableLoginWithPhone:      true,
	PasswordlessType:           "Allowed",
	DefaultRedirectURI:         setValue,
	PasswordCheckLifetime:      setValue,
	ExternalLoginCheckLifetime: setValue,
	MFAInitSkipLifetime:        setValue,
	SecondFactorCheckLifetime:  setValue,
	MultiFactorCheckLifetime:   setValue,
	SecondFactors:              []string{"RecoveryCodes"},
	MultiFactors:               []string{"U2FWithVerification"},
}

// observedAll holds a different value on every field, which is what makes an
// unmanaged field show up as drift if the driver is wrong, and what makes a
// managed one unambiguously drift.
var observedAll = zitadel.DefaultLoginPolicy{
	AllowUsernamePassword:      false,
	AllowRegister:              false,
	AllowExternalIDP:           false,
	ForceMFA:                   false,
	ForceMFALocalOnly:          false,
	HidePasswordReset:          false,
	IgnoreUnknownUsernames:     false,
	AllowDomainDiscovery:       false,
	DisableLoginWithEmail:      false,
	DisableLoginWithPhone:      false,
	PasswordlessType:           "NotAllowed",
	DefaultRedirectURI:         diffValue,
	PasswordCheckLifetime:      diffValue,
	ExternalLoginCheckLifetime: diffValue,
	MFAInitSkipLifetime:        diffValue,
	SecondFactorCheckLifetime:  diffValue,
	MultiFactorCheckLifetime:   diffValue,
	SecondFactors:              []string{otherFactor},
	MultiFactors:               []string{otherFactor},
}

// TestEqual checks the drift decision for every field, set and unset.
func TestEqual(t *testing.T) {
	var d driver

	all := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}
	if !d.Equal(all, d.Desired(all), matching) {
		t.Error("Equal(...): a policy matching every field the manifest set is drift")
	}

	t.Run("NothingManaged", func(t *testing.T) {
		// The regression: a manifest that says nothing manages nothing, so
		// Zitadel's own values are not something to correct.
		empty := &v1alpha1.DefaultLoginPolicy{}

		if !d.Equal(empty, d.Desired(empty), observedAll) {
			t.Error("Equal(...): a policy that manages no fields is not drift, whatever Zitadel holds")
		}
	})

	t.Run("AllowUsernamePasswordUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.AllowUsernamePassword = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.AllowUsernamePassword = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset AllowUsernamePassword is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("AllowUsernamePasswordDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.AllowUsernamePassword = false
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed AllowUsernamePassword that Zitadel does not hold is drift")
		}
	})

	t.Run("AllowRegisterUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.AllowRegister = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.AllowRegister = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset AllowRegister is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("AllowRegisterDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.AllowRegister = false
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed AllowRegister that Zitadel does not hold is drift")
		}
	})

	t.Run("AllowExternalIDPUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.AllowExternalIDP = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.AllowExternalIDP = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset AllowExternalIDP is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("AllowExternalIDPDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.AllowExternalIDP = false
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed AllowExternalIDP that Zitadel does not hold is drift")
		}
	})

	t.Run("ForceMFAUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.ForceMFA = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.ForceMFA = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset ForceMFA is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("ForceMFADrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.ForceMFA = false
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed ForceMFA that Zitadel does not hold is drift")
		}
	})

	t.Run("ForceMFALocalOnlyUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.ForceMFALocalOnly = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.ForceMFALocalOnly = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset ForceMFALocalOnly is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("ForceMFALocalOnlyDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.ForceMFALocalOnly = false
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed ForceMFALocalOnly that Zitadel does not hold is drift")
		}
	})

	t.Run("HidePasswordResetUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.HidePasswordReset = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.HidePasswordReset = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset HidePasswordReset is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("HidePasswordResetDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.HidePasswordReset = false
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed HidePasswordReset that Zitadel does not hold is drift")
		}
	})

	t.Run("IgnoreUnknownUsernamesUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.IgnoreUnknownUsernames = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.IgnoreUnknownUsernames = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset IgnoreUnknownUsernames is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("IgnoreUnknownUsernamesDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.IgnoreUnknownUsernames = false
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed IgnoreUnknownUsernames that Zitadel does not hold is drift")
		}
	})

	t.Run("AllowDomainDiscoveryUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.AllowDomainDiscovery = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.AllowDomainDiscovery = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset AllowDomainDiscovery is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("AllowDomainDiscoveryDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.AllowDomainDiscovery = false
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed AllowDomainDiscovery that Zitadel does not hold is drift")
		}
	})

	t.Run("DisableLoginWithEmailUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.DisableLoginWithEmail = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.DisableLoginWithEmail = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset DisableLoginWithEmail is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("DisableLoginWithEmailDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.DisableLoginWithEmail = false
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed DisableLoginWithEmail that Zitadel does not hold is drift")
		}
	})

	t.Run("DisableLoginWithPhoneUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.DisableLoginWithPhone = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.DisableLoginWithPhone = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset DisableLoginWithPhone is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("DisableLoginWithPhoneDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.DisableLoginWithPhone = false
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed DisableLoginWithPhone that Zitadel does not hold is drift")
		}
	})

	t.Run("PasswordlessTypeUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.PasswordlessType = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.PasswordlessType = "NotAllowed"

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset PasswordlessType is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("PasswordlessTypeDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.PasswordlessType = "NotAllowed"
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed PasswordlessType that Zitadel does not hold is drift")
		}
	})

	t.Run("DefaultRedirectURIUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.DefaultRedirectURI = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.DefaultRedirectURI = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset DefaultRedirectURI is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("DefaultRedirectURIDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.DefaultRedirectURI = diffValue
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed DefaultRedirectURI that Zitadel does not hold is drift")
		}
	})

	t.Run("PasswordCheckLifetimeUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.PasswordCheckLifetime = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.PasswordCheckLifetime = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset PasswordCheckLifetime is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("PasswordCheckLifetimeDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.PasswordCheckLifetime = diffValue
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed PasswordCheckLifetime that Zitadel does not hold is drift")
		}
	})

	t.Run("ExternalLoginCheckLifetimeUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.ExternalLoginCheckLifetime = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.ExternalLoginCheckLifetime = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset ExternalLoginCheckLifetime is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("ExternalLoginCheckLifetimeDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.ExternalLoginCheckLifetime = diffValue
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed ExternalLoginCheckLifetime that Zitadel does not hold is drift")
		}
	})

	t.Run("MFAInitSkipLifetimeUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.MFAInitSkipLifetime = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.MFAInitSkipLifetime = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset MFAInitSkipLifetime is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("MFAInitSkipLifetimeDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.MFAInitSkipLifetime = diffValue
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed MFAInitSkipLifetime that Zitadel does not hold is drift")
		}
	})

	t.Run("SecondFactorCheckLifetimeUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.SecondFactorCheckLifetime = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.SecondFactorCheckLifetime = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset SecondFactorCheckLifetime is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("SecondFactorCheckLifetimeDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.SecondFactorCheckLifetime = diffValue
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed SecondFactorCheckLifetime that Zitadel does not hold is drift")
		}
	})

	t.Run("MultiFactorCheckLifetimeUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.MultiFactorCheckLifetime = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.MultiFactorCheckLifetime = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset MultiFactorCheckLifetime is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("MultiFactorCheckLifetimeDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.MultiFactorCheckLifetime = diffValue
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed MultiFactorCheckLifetime that Zitadel does not hold is drift")
		}
	})

	t.Run("SecondFactorsUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.SecondFactors = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.SecondFactors = []string{otherFactor}

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset SecondFactors is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("SecondFactorsDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.SecondFactors = []string{otherFactor}
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed SecondFactors that Zitadel does not hold is drift")
		}
	})

	t.Run("MultiFactorsUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.MultiFactors = nil
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.MultiFactors = []string{otherFactor}

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset MultiFactors is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("MultiFactorsDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.DefaultLoginPolicy{}
		obs.MultiFactors = []string{otherFactor}
		only := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed MultiFactors that Zitadel does not hold is drift")
		}
	})

}

// TestAskingForZeroIsAskingForZero checks that an explicit zero and an omission
// are not the same request. Only the second leaves Zitadel alone.
func TestAskingForZeroIsAskingForZero(t *testing.T) {
	var d driver

	askedForZero := &v1alpha1.DefaultLoginPolicy{Spec: v1alpha1.DefaultLoginPolicySpec{
		ForProvider: v1alpha1.DefaultLoginPolicyParameters{AllowUsernamePassword: ptrTo(false)},
	}}

	if d.Equal(askedForZero, d.Desired(askedForZero), matching) {
		t.Error("Equal(...): a field the manifest asked for and Zitadel does not hold is drift")
	}

	neverAsked := &v1alpha1.DefaultLoginPolicy{}

	if !d.Equal(neverAsked, d.Desired(neverAsked), matching) {
		t.Errorf("Equal(...): AllowUsernamePassword was never asked for, so Zitadel's value is not drift")
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
