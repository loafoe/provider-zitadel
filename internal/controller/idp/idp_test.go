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

package idp

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Only an OIDC and a JWT provider are read back by Zitadel, and that decides
// whether their settings can be compared at all.
//
// The bug this guards against is silent: an unobservable provider compared as if
// it were observable reports "up to date" forever, so a change to its settings is
// silently never applied.
func TestObservableOnlyForTheKindsZitadelReads(t *testing.T) {
	for name, tc := range map[string]struct {
		provider string
		want     bool
	}{
		"an OIDC provider":     {"OIDC", true},
		"a JWT provider":       {"JWT", true},
		"an OAuth provider":    {"OAuth", false},
		"a GitHub provider":    {"GitHub", false},
		"a SAML provider":      {"SAML", false},
		"an LDAP provider":     {"LDAP", false},
		"an Apple provider":    {"Apple", false},
		"an Azure AD provider": {"AzureAD", false},
		"a self hosted GitLab": {"GitLabSelfHosted", false},
	} {
		t.Run(name, func(t *testing.T) {
			d := base{provider: tc.provider, observable: tc.provider == "OIDC" || tc.provider == "JWT"}
			if got := d.Observable(); got != tc.want {
				t.Errorf("Provider() = %q, Observable() = %t, want %t", d.Provider(), got, tc.want)
			}
		})
	}
}

// An unset state is left empty rather than defaulted, so that an operator who
// never mentioned it gets Zitadel's own default instead of a value fought over
// on every reconcile.
func TestStateIsLeftUnsetWhenNotAskedFor(t *testing.T) {
	cr := &v1alpha1.OrgIDPOIDC{}

	in := withCommon(zitadel.ProviderInput{}, cr, cr.Spec.ForProvider.Name, cr.Spec.ForProvider.Scopes)
	if in.State != "" {
		t.Errorf("an unset state became %q, want it left empty", in.State)
	}

	inactive := v1alpha1.IDPStateInactive
	cr.Spec.ForProvider.State = &inactive

	in = withCommon(zitadel.ProviderInput{}, cr, cr.Spec.ForProvider.Name, cr.Spec.ForProvider.Scopes)
	if in.State != "Inactive" {
		t.Errorf("state = %q, want Inactive", in.State)
	}
}

// The desired state travels with the rest of the desired provider, because
// Zitadel applies it with a call of its own.
func TestWithCommonCarriesTheState(t *testing.T) {
	active := v1alpha1.IDPStateActive
	cr := &v1alpha1.OrgIDPGitHub{}
	cr.Spec.ForProvider.State = &active

	in := withCommon(zitadel.ProviderInput{}, cr, cr.Spec.ForProvider.Name, cr.Spec.ForProvider.Scopes)

	if in.State != "Active" {
		t.Errorf("withCommon(...).State = %q, want Active", in.State)
	}
	if in.Name != "" || in.Scopes != nil {
		t.Errorf("withCommon(...) should leave an unset name and scopes alone, got %q and %v", in.Name, in.Scopes)
	}
}

// Drift is judged against the desired provider, not against the status.
//
// The status holds what Zitadel reported, so comparing the two would compare
// Zitadel with itself and call every provider up to date.
func TestEqualComparesTheDesiredAgainstTheObserved(t *testing.T) {
	// The name every case below shares.
	const name = "P3 Example"

	want := zitadel.ProviderInput{
		Name: name,
		JWT:  &zitadel.JWTSettings{JWTEndpoint: "https://example.com/jwt", Issuer: "https://example.com"},
	}

	t.Run("the same provider is up to date", func(t *testing.T) {
		observed := zitadel.IdentityProvider{
			Name:  name,
			State: "Active",
			JWT:   &zitadel.JWTSettings{JWTEndpoint: "https://example.com/jwt", Issuer: "https://example.com"},
		}
		if !(base{}).Equal(want, observed) {
			t.Error("a provider that matches the spec reported as drift")
		}
	})

	t.Run("a changed endpoint is drift", func(t *testing.T) {
		observed := zitadel.IdentityProvider{
			Name: name,
			JWT:  &zitadel.JWTSettings{JWTEndpoint: "https://example.com/other", Issuer: "https://example.com"},
		}
		if (base{}).Equal(want, observed) {
			t.Error("a changed JWT endpoint reported as up to date")
		}
	})

	t.Run("a changed name is drift", func(t *testing.T) {
		observed := zitadel.IdentityProvider{Name: "Something Else", JWT: want.JWT}
		if (base{}).Equal(want, observed) {
			t.Error("a changed name reported as up to date")
		}
	})

	t.Run("auto creation is compared", func(t *testing.T) {
		withCreate := want
		withCreate.Options.IsAutoCreation = true
		observed := zitadel.IdentityProvider{Name: name, JWT: want.JWT}

		if (base{}).Equal(withCreate, observed) {
			t.Error("isAutoCreation turned on and not reported by Zitadel reported as up to date")
		}

		withCreate.Options.IsAutoCreation = false
		observed.AutoRegister = false
		if !(base{}).Equal(withCreate, observed) {
			t.Error("isAutoCreation turned off and not reported by Zitadel reported as drift")
		}
	})

	t.Run("a kind Zitadel reports no configuration for is never compared", func(t *testing.T) {
		// Nothing comes back for it, so there is nothing to compare, and claiming
		// otherwise would report drift no reconcile could settle.
		observed := zitadel.IdentityProvider{Name: name, State: "Active"}
		plain := zitadel.ProviderInput{Name: name, ClientID: "whatever"}

		if !(base{}).Equal(plain, observed) {
			t.Error("a provider whose configuration Zitadel does not return reported as drift")
		}
	})
}
