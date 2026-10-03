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

package instancesettings

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"

	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// The instance and system flag tables share two of their flags, so the same
// driver serves both. Getting that wrong is silent: a SystemFeatures compared
// against the instance table's fields would report drift on two flags Zitadel
// never reports for it, and reconcile forever.
func TestSystemFeaturesAreComparedAgainstTheirOwnFields(t *testing.T) {
	d := featuresDriver{}

	want := zitadel.FeatureFlags{LoginDefaultOrg: true, UserSchema: true}

	if !d.Equal(want, zitadel.FeatureFlags{LoginDefaultOrg: true, UserSchema: true}) {
		t.Error("matching system flags reported as drift")
	}

	if d.Equal(zitadel.FeatureFlags{LoginDefaultOrg: true, UserSchema: false},
		zitadel.FeatureFlags{LoginDefaultOrg: true, UserSchema: true}) {
		t.Error("a changed userSchema reported as up to date")
	}

	if d.Equal(zitadel.FeatureFlags{LoginDefaultOrg: false, UserSchema: true},
		zitadel.FeatureFlags{LoginDefaultOrg: true, UserSchema: true}) {
		t.Error("a changed loginDefaultOrg reported as up to date")
	}

	// The system table has no improved performance paths and no debug flag, so
	// nothing in a SystemFeatures can set them and nothing read for it can carry
	// them. Comparing them anyway would report drift on values Zitadel never
	// returns, and reconcile without ever settling.
	withExtras := zitadel.FeatureFlags{
		LoginDefaultOrg:      true,
		UserSchema:           true,
		DebugOidcParentError: true,
		ImprovedPerformance:  []string{"IMPROVED_PERFORMANCE_PROJECT"},
	}
	if !d.Equal(withExtras, zitadel.FeatureFlags{LoginDefaultOrg: true, UserSchema: true}) {
		t.Error("system flags reported as drift on fields the system table does not have")
	}

	// The instance table does report them, so a change there is real drift.
	inst := featuresDriver{instance: true}
	if inst.Equal(withExtras, zitadel.FeatureFlags{LoginDefaultOrg: true, UserSchema: true}) {
		t.Error("an instance flag turned off and asked for reported as up to date")
	}
}

// An empty list and a list with nothing in it are the same thing, and treating
// them as different would make the provider clear the improved performance
// paths and put them back on every reconcile.
func TestEmptyImprovedPerformanceIsNotDrift(t *testing.T) {
	d := featuresDriver{instance: true}

	if !d.Equal(zitadel.FeatureFlags{}, zitadel.FeatureFlags{ImprovedPerformance: []string{}}) {
		t.Error("an unset list compared against an empty one reported as drift")
	}

	if !d.Equal(
		zitadel.FeatureFlags{ImprovedPerformance: []string{"IMPROVED_PERFORMANCE_PROJECT"}},
		zitadel.FeatureFlags{ImprovedPerformance: []string{"IMPROVED_PERFORMANCE_PROJECT"}},
	) {
		t.Error("matching improved performance paths reported as drift")
	}

	if d.Equal(
		zitadel.FeatureFlags{ImprovedPerformance: []string{"IMPROVED_PERFORMANCE_PROJECT"}},
		zitadel.FeatureFlags{ImprovedPerformance: []string{"IMPROVED_PERFORMANCE_USER_GRANT"}},
	) {
		t.Error("a changed improved performance path reported as up to date")
	}
}

// The restrictions cannot be reset, so deleting the resource restores what it
// overwrote. That is the whole reason this driver reports Resettable as false,
// and getting it wrong would leave the instance configured by a manifest that
// no longer exists.
func TestRestrictionsAreNotResettable(t *testing.T) {
	if (restrictionsDriver{}).Resettable() {
		t.Error("the registration restrictions reported as resettable; Zitadel has no reset call")
	}

	d := restrictionsDriver{}
	want := zitadel.Restrictions{DisallowPublicOrgRegistration: true, AllowedLanguages: []string{"en", "de"}}

	if !d.Equal(want, zitadel.Restrictions{DisallowPublicOrgRegistration: true, AllowedLanguages: []string{"en", "de"}}) {
		t.Error("matching restrictions reported as drift")
	}

	if d.Equal(want, zitadel.Restrictions{DisallowPublicOrgRegistration: false, AllowedLanguages: []string{"en", "de"}}) {
		t.Error("a lifted registration restriction reported as up to date")
	}

	if d.Equal(want, zitadel.Restrictions{DisallowPublicOrgRegistration: true, AllowedLanguages: []string{"en"}}) {
		t.Error("a removed allowed language reported as up to date")
	}
}

// A secret generator is identified by its type, and that type is its scope.
//
// The scope has to come out of the manifest before the recorded one, or a
// terminating resource would restore a generator it never configured.
func TestSecretGeneratorScopePrefersTheRecordedOne(t *testing.T) {
	cr := newSecretGenerator("SECRET_GENERATOR_TYPE_APP_SECRET")

	got, err := (secretGeneratorDriver{}).Scope(t.Context(), nil, cr)
	if err != nil {
		t.Fatalf("Scope() returned %v", err)
	}
	if got != "SECRET_GENERATOR_TYPE_APP_SECRET" {
		t.Errorf("Scope() = %q, want the generator type from the manifest", got)
	}

	// Once something has been read, that is the scope a delete must act on.
	cr.SetPolicyScope("SECRET_GENERATOR_TYPE_INIT_CODE")

	got, err = (secretGeneratorDriver{}).Scope(t.Context(), nil, cr)
	if err != nil {
		t.Fatalf("Scope() returned %v", err)
	}
	if got != "SECRET_GENERATOR_TYPE_INIT_CODE" {
		t.Errorf("Scope() = %q, want the recorded scope a terminating resource acts on", got)
	}
}

// A secret generator cannot be reset either: Zitadel can reshape one or leave
// it, but there is no call to say "as it was".
func TestSecretGeneratorIsNotResettable(t *testing.T) {
	if (secretGeneratorDriver{}).Resettable() {
		t.Error("a secret generator reported as resettable; Zitadel has no reset call")
	}

	d := secretGeneratorDriver{}
	want := zitadel.SecretGenerator{
		Type:                "SECRET_GENERATOR_TYPE_INIT_CODE",
		Length:              6,
		Expiry:              "5m",
		IncludeLowerLetters: true,
		IncludeDigits:       true,
	}

	if !d.Equal(want, want) {
		t.Error("a generator that matches itself reported as drift")
	}

	changed := want
	changed.Length = 8
	if d.Equal(want, changed) {
		t.Error("a changed length reported as up to date")
	}

	changed = want
	changed.IncludeSymbols = true
	if d.Equal(want, changed) {
		t.Error("a changed character class reported as up to date")
	}

	changed = want
	changed.Type = "SECRET_GENERATOR_TYPE_APP_SECRET"
	if d.Equal(want, changed) {
		t.Error("a changed generator type reported as up to date")
	}
}

// The two feature kinds are both resettable, and that is what makes deleting
// one a real deletion rather than a restore.
func TestFeatureFlagsAreResettable(t *testing.T) {
	if !(featuresDriver{}).Resettable() {
		t.Error("the system feature flags reported as not resettable; Zitadel can reset them")
	}

	if !(featuresDriver{instance: true}).Resettable() {
		t.Error("the instance feature flags reported as not resettable; Zitadel can reset them")
	}
}

// Every driver has to name what it manages, since that name is what an error
// and an event say.
func TestDriversNameThemselves(t *testing.T) {
	for name, d := range map[string]interface{ Kind() string }{
		"restrictions":     restrictionsDriver{},
		"secretGenerator":  secretGeneratorDriver{},
		"systemFeatures":   featuresDriver{},
		"instanceFeatures": featuresDriver{instance: true},
	} {
		t.Run(name, func(t *testing.T) {
			if d.Kind() == "" {
				t.Error("Kind() is empty, so an error could not say what failed")
			}
		})
	}
}

// The shared harness needs these kinds to satisfy it, and saying so here names
// the missing method once rather than at four call sites.
var (
	_ common.PolicyDriver[zitadel.FeatureFlags, zitadel.FeatureFlags]       = featuresDriver{instance: true}
	_ common.PolicyDriver[zitadel.Restrictions, zitadel.Restrictions]       = restrictionsDriver{}
	_ common.PolicyDriver[zitadel.SecretGenerator, zitadel.SecretGenerator] = secretGeneratorDriver{}
)

// newSecretGenerator builds a resource with one generator type in it, which is
// the only thing the scope is read from.
func newSecretGenerator(generatorType string) *v1alpha1.InstanceSecretGenerator {
	cr := &v1alpha1.InstanceSecretGenerator{}
	cr.Spec.ForProvider.GeneratorType = common.StringPtr(generatorType)

	return cr
}

// Zitadel reads a duration back normalised: a manifest that says `5m` comes
// back as `5m0s`. Compared as strings that is drift on every reconcile, and the
// resource never settles.
func TestExpiryIsComparedByMeaningNotBySpelling(t *testing.T) {
	d := secretGeneratorDriver{}
	base := zitadel.SecretGenerator{Type: "SECRET_GENERATOR_TYPE_INIT_CODE", Length: 6}

	for name, tc := range map[string]struct {
		want, observed string
		same           bool
	}{
		"the same spelling":            {"5m", "5m", true},
		"zitadel's normalisation":      {"5m", "5m0s", true},
		"the other way round":          {"5m0s", "5m", true},
		"seconds to minutes":           {"300s", "5m", true},
		"a genuinely different expiry": {"5m", "10m", false},
		"an unparseable expiry":        {"soon", "5m0s", false},
	} {
		t.Run(name, func(t *testing.T) {
			w, o := base, base
			w.Expiry, o.Expiry = tc.want, tc.observed

			if got := d.Equal(w, o); got != tc.same {
				t.Errorf("Equal(%q, %q) = %t, want %t", tc.want, tc.observed, got, tc.same)
			}
		})
	}
}
