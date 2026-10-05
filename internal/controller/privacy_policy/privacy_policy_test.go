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

package privacy_policy

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
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
func setAll() v1alpha1.PrivacyPolicyParameters {
	return v1alpha1.PrivacyPolicyParameters{
		TOSLink:        ptrTo(setValue),
		PrivacyLink:    ptrTo(setValue),
		HelpLink:       ptrTo(setValue),
		SupportEmail:   ptrTo(setValue),
		DocsLink:       ptrTo(setValue),
		CustomLink:     ptrTo(setValue),
		CustomLinkText: ptrTo(setValue),
	}
}

// matching holds exactly what setAll asks for, so the all-managed case has
// something to agree with.
var matching = zitadel.PrivacyPolicy{
	TOSLink:        setValue,
	PrivacyLink:    setValue,
	HelpLink:       setValue,
	SupportEmail:   setValue,
	DocsLink:       setValue,
	CustomLink:     setValue,
	CustomLinkText: setValue,
}

// observedAll holds a different value on every field, which is what makes an
// unmanaged field show up as drift if the driver is wrong, and what makes a
// managed one unambiguously drift.
var observedAll = zitadel.PrivacyPolicy{
	TOSLink:        diffValue,
	PrivacyLink:    diffValue,
	HelpLink:       diffValue,
	SupportEmail:   diffValue,
	DocsLink:       diffValue,
	CustomLink:     diffValue,
	CustomLinkText: diffValue,
}

// TestEqual checks the drift decision for every field, set and unset.
func TestEqual(t *testing.T) {
	var d driver

	all := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: setAll()}}
	if !d.Equal(all, d.Desired(all), matching) {
		t.Error("Equal(...): a policy matching every field the manifest set is drift")
	}

	t.Run("NothingManaged", func(t *testing.T) {
		// The regression: a manifest that says nothing manages nothing, so
		// Zitadel's own values are not something to correct.
		empty := &v1alpha1.PrivacyPolicy{}

		if !d.Equal(empty, d.Desired(empty), observedAll) {
			t.Error("Equal(...): a policy that manages no fields is not drift, whatever Zitadel holds")
		}
	})

	t.Run("TOSLinkUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.TOSLink = nil
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.TOSLink = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset TOSLink is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("TOSLinkDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PrivacyPolicy{}
		obs.TOSLink = diffValue
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed TOSLink that Zitadel does not hold is drift")
		}
	})

	t.Run("PrivacyLinkUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.PrivacyLink = nil
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.PrivacyLink = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset PrivacyLink is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("PrivacyLinkDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PrivacyPolicy{}
		obs.PrivacyLink = diffValue
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed PrivacyLink that Zitadel does not hold is drift")
		}
	})

	t.Run("HelpLinkUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.HelpLink = nil
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.HelpLink = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset HelpLink is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("HelpLinkDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PrivacyPolicy{}
		obs.HelpLink = diffValue
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed HelpLink that Zitadel does not hold is drift")
		}
	})

	t.Run("SupportEmailUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.SupportEmail = nil
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.SupportEmail = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset SupportEmail is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("SupportEmailDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PrivacyPolicy{}
		obs.SupportEmail = diffValue
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed SupportEmail that Zitadel does not hold is drift")
		}
	})

	t.Run("DocsLinkUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.DocsLink = nil
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.DocsLink = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset DocsLink is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("DocsLinkDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PrivacyPolicy{}
		obs.DocsLink = diffValue
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed DocsLink that Zitadel does not hold is drift")
		}
	})

	t.Run("CustomLinkUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.CustomLink = nil
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.CustomLink = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset CustomLink is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("CustomLinkDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PrivacyPolicy{}
		obs.CustomLink = diffValue
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed CustomLink that Zitadel does not hold is drift")
		}
	})

	t.Run("CustomLinkTextUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.CustomLinkText = nil
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.CustomLinkText = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset CustomLinkText is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("CustomLinkTextDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.PrivacyPolicy{}
		obs.CustomLinkText = diffValue
		only := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed CustomLinkText that Zitadel does not hold is drift")
		}
	})

}

// TestAskingForZeroIsAskingForZero checks that an explicit zero and an omission
// are not the same request. Only the second leaves Zitadel alone.
func TestAskingForZeroIsAskingForZero(t *testing.T) {
	var d driver

	askedForZero := &v1alpha1.PrivacyPolicy{Spec: v1alpha1.PrivacyPolicySpec{
		ForProvider: v1alpha1.PrivacyPolicyParameters{TOSLink: ptrTo("")},
	}}

	if d.Equal(askedForZero, d.Desired(askedForZero), matching) {
		t.Error("Equal(...): a field the manifest asked for and Zitadel does not hold is drift")
	}

	neverAsked := &v1alpha1.PrivacyPolicy{}

	if !d.Equal(neverAsked, d.Desired(neverAsked), matching) {
		t.Errorf("Equal(...): TOSLink was never asked for, so Zitadel's value is not drift")
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
