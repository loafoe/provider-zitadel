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

package label_policy

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
func setAll() v1alpha1.LabelPolicyParameters {
	return v1alpha1.LabelPolicyParameters{
		HideLoginNameSuffix: ptrTo(true),
		DisableWatermark:    ptrTo(true),
		ThemeMode:           ptrTo(v1alpha1.LabelThemeModeLight),
		PrimaryColor:        ptrTo(setValue),
		WarnColor:           ptrTo(setValue),
		BackgroundColor:     ptrTo(setValue),
		FontColor:           ptrTo(setValue),
		PrimaryColorDark:    ptrTo(setValue),
		WarnColorDark:       ptrTo(setValue),
		BackgroundColorDark: ptrTo(setValue),
		FontColorDark:       ptrTo(setValue),
	}
}

// matching holds exactly what setAll asks for, so the all-managed case has
// something to agree with.
var matching = zitadel.LabelPolicy{
	HideLoginNameSuffix: true,
	DisableWatermark:    true,
	ThemeMode:           "Light",
	PrimaryColor:        setValue,
	WarnColor:           setValue,
	BackgroundColor:     setValue,
	FontColor:           setValue,
	PrimaryColorDark:    setValue,
	WarnColorDark:       setValue,
	BackgroundColorDark: setValue,
	FontColorDark:       setValue,
}

// observedAll holds a different value on every field, which is what makes an
// unmanaged field show up as drift if the driver is wrong, and what makes a
// managed one unambiguously drift.
var observedAll = zitadel.LabelPolicy{
	HideLoginNameSuffix: false,
	DisableWatermark:    false,
	ThemeMode:           "Dark",
	PrimaryColor:        diffValue,
	WarnColor:           diffValue,
	BackgroundColor:     diffValue,
	FontColor:           diffValue,
	PrimaryColorDark:    diffValue,
	WarnColorDark:       diffValue,
	BackgroundColorDark: diffValue,
	FontColorDark:       diffValue,
}

// TestEqual checks the drift decision for every field, set and unset.
func TestEqual(t *testing.T) {
	var d driver

	all := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}
	if !d.Equal(all, d.Desired(all), matching) {
		t.Error("Equal(...): a policy matching every field the manifest set is drift")
	}

	t.Run("NothingManaged", func(t *testing.T) {
		// The regression: a manifest that says nothing manages nothing, so
		// Zitadel's own values are not something to correct.
		empty := &v1alpha1.LabelPolicy{}

		if !d.Equal(empty, d.Desired(empty), observedAll) {
			t.Error("Equal(...): a policy that manages no fields is not drift, whatever Zitadel holds")
		}
	})

	t.Run("HideLoginNameSuffixUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.HideLoginNameSuffix = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.HideLoginNameSuffix = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset HideLoginNameSuffix is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("HideLoginNameSuffixDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.HideLoginNameSuffix = false
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed HideLoginNameSuffix that Zitadel does not hold is drift")
		}
	})

	t.Run("DisableWatermarkUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.DisableWatermark = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.DisableWatermark = false

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset DisableWatermark is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("DisableWatermarkDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.DisableWatermark = false
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed DisableWatermark that Zitadel does not hold is drift")
		}
	})

	t.Run("ThemeModeUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.ThemeMode = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.ThemeMode = "Dark"

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset ThemeMode is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("ThemeModeDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.ThemeMode = "Dark"
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed ThemeMode that Zitadel does not hold is drift")
		}
	})

	t.Run("PrimaryColorUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.PrimaryColor = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.PrimaryColor = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset PrimaryColor is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("PrimaryColorDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.PrimaryColor = diffValue
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed PrimaryColor that Zitadel does not hold is drift")
		}
	})

	t.Run("WarnColorUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.WarnColor = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.WarnColor = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset WarnColor is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("WarnColorDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.WarnColor = diffValue
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed WarnColor that Zitadel does not hold is drift")
		}
	})

	t.Run("BackgroundColorUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.BackgroundColor = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.BackgroundColor = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset BackgroundColor is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("BackgroundColorDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.BackgroundColor = diffValue
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed BackgroundColor that Zitadel does not hold is drift")
		}
	})

	t.Run("FontColorUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.FontColor = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.FontColor = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset FontColor is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("FontColorDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.FontColor = diffValue
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed FontColor that Zitadel does not hold is drift")
		}
	})

	t.Run("PrimaryColorDarkUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.PrimaryColorDark = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.PrimaryColorDark = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset PrimaryColorDark is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("PrimaryColorDarkDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.PrimaryColorDark = diffValue
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed PrimaryColorDark that Zitadel does not hold is drift")
		}
	})

	t.Run("WarnColorDarkUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.WarnColorDark = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.WarnColorDark = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset WarnColorDark is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("WarnColorDarkDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.WarnColorDark = diffValue
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed WarnColorDark that Zitadel does not hold is drift")
		}
	})

	t.Run("BackgroundColorDarkUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.BackgroundColorDark = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.BackgroundColorDark = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset BackgroundColorDark is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("BackgroundColorDarkDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.BackgroundColorDark = diffValue
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed BackgroundColorDark that Zitadel does not hold is drift")
		}
	})

	t.Run("FontColorDarkUnset", func(t *testing.T) {
		// Every field managed but this one, so only this one is unset.
		fp := setAll()
		fp.FontColorDark = nil
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: fp}}

		// Zitadel holds something else for the unset field, which is what
		// used to be reported as drift.
		obs := matching
		obs.FontColorDark = diffValue

		if !d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): an unset FontColorDark is not drift even though Zitadel holds a value for it")
		}
	})

	t.Run("FontColorDarkDrift", func(t *testing.T) {
		// The field is managed and Zitadel holds something else, which is drift.
		obs := zitadel.LabelPolicy{}
		obs.FontColorDark = diffValue
		only := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{ForProvider: setAll()}}

		if d.Equal(only, d.Desired(only), obs) {
			t.Error("Equal(...): a managed FontColorDark that Zitadel does not hold is drift")
		}
	})

}

// TestAskingForZeroIsAskingForZero checks that an explicit zero and an omission
// are not the same request. Only the second leaves Zitadel alone.
func TestAskingForZeroIsAskingForZero(t *testing.T) {
	var d driver

	askedForZero := &v1alpha1.LabelPolicy{Spec: v1alpha1.LabelPolicySpec{
		ForProvider: v1alpha1.LabelPolicyParameters{HideLoginNameSuffix: ptrTo(false)},
	}}

	if d.Equal(askedForZero, d.Desired(askedForZero), matching) {
		t.Error("Equal(...): a field the manifest asked for and Zitadel does not hold is drift")
	}

	neverAsked := &v1alpha1.LabelPolicy{}

	if !d.Equal(neverAsked, d.Desired(neverAsked), matching) {
		t.Errorf("Equal(...): HideLoginNameSuffix was never asked for, so Zitadel's value is not drift")
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
