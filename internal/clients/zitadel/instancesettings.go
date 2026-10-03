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
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/admin"
	feature "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/feature/v2"
	settings "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/settings"
)

// InstanceSettings covers the instance wide settings that are a single value
// rather than a list or an object of their own: the feature flags, the
// registration restrictions and the secret generator.
//
// Zitadel keeps these in three separate places, so each one has its own
// function here rather than being gathered into a single call that would have to
// paper over the differences.

// FeatureFlags is a feature flag as Zitadel reports it: a value, and whether it
// was set on this instance or inherited.
type FeatureFlags struct {
	// LoginDefaultOrg makes the login screen use the default organization's
	// settings when no organization context is set.
	LoginDefaultOrg bool

	// UserSchema enables the user schema API, which manages per-user data
	// schemas.
	UserSchema bool

	// DebugOidcParentError returns parent errors to OIDC clients. It can leak
	// details about the system, so it is off unless it is asked for.
	DebugOidcParentError bool

	// ImprovedPerformance lists the execution paths Zitadel may shortcut.
	ImprovedPerformance []string
}

// Input returns the flags in the shape the write API accepts.
//
// The shared harness restores a recorded snapshot by turning it back into a
// writable shape through this method, so it is what makes deleting one of these
// resources put the instance back rather than leave it configured by a manifest
// that no longer exists.
func (f FeatureFlags) Input() any {
	return FeatureFlags{
		LoginDefaultOrg:      f.LoginDefaultOrg,
		UserSchema:           f.UserSchema,
		DebugOidcParentError: f.DebugOidcParentError,
		ImprovedPerformance:  f.ImprovedPerformance,
	}
}

// SetInstanceFeatures sets the instance wide feature flags.
//
// Every flag is sent, including the ones an operator left out, because Zitadel
// treats a flag that is absent from the request as one to leave alone and
// there is no separate reset: an unset flag has to become Zitadel's default
// rather than whatever it happened to be.
func (c *Client) SetInstanceFeatures(ctx context.Context, f FeatureFlags) error {
	improved := make([]feature.ImprovedPerformance, 0, len(f.ImprovedPerformance))
	for _, p := range f.ImprovedPerformance {
		v, ok := feature.ImprovedPerformance_value[p]
		if !ok {
			return fmt.Errorf("zitadel has no improved performance value %q: it must be one of %v", p, feature.ImprovedPerformance_name)
		}

		improved = append(improved, feature.ImprovedPerformance(v))
	}

	// Inheritance is deliberately absent: Zitadel reads whether an organization
	// may override these flags but offers no call to set it on an instance, so
	// there is nothing here to apply.
	_, err := c.feature.SetInstanceFeatures(ctx, &feature.SetInstanceFeaturesRequest{
		LoginDefaultOrg:      boolPtr(f.LoginDefaultOrg),
		UserSchema:           boolPtr(f.UserSchema),
		DebugOidcParentError: boolPtr(f.DebugOidcParentError),
		ImprovedPerformance:  improved,
	})
	return err
}

// GetInstanceFeatures reads the instance wide feature flags.
func (c *Client) GetInstanceFeatures(ctx context.Context) (*FeatureFlags, error) {
	resp, err := c.feature.GetInstanceFeatures(ctx, &feature.GetInstanceFeaturesRequest{})
	if err != nil {
		return nil, err
	}

	out := &FeatureFlags{
		ImprovedPerformance: make([]string, 0, len(resp.GetImprovedPerformance().GetExecutionPaths())),
	}
	if f := resp.GetLoginDefaultOrg(); f != nil {
		out.LoginDefaultOrg = f.GetEnabled()
	}
	if f := resp.GetUserSchema(); f != nil {
		out.UserSchema = f.GetEnabled()
	}
	if f := resp.GetDebugOidcParentError(); f != nil {
		out.DebugOidcParentError = f.GetEnabled()
	}
	for _, p := range resp.GetImprovedPerformance().GetExecutionPaths() {
		// ZITADEL sends the enum as a number, so converting it straight to a
		// string would yield the character with that code point rather than the
		// path's name. The name map is what makes it readable.
		name, ok := feature.ImprovedPerformance_name[int32(p)]
		if !ok {
			return nil, fmt.Errorf("zitadel reported the unknown improved performance value %d", int32(p))
		}

		out.ImprovedPerformance = append(out.ImprovedPerformance, name)
	}

	return out, nil
}

// ResetInstanceFeatures puts the instance wide flags back to Zitadel's own
// defaults.
//
// This is what makes deleting an InstanceFeatures a deletion rather than a
// restore: the instance is left as it was found instead of configured by a
// manifest that no longer exists.
func (c *Client) ResetInstanceFeatures(ctx context.Context) error {
	_, err := c.feature.ResetInstanceFeatures(ctx, &feature.ResetInstanceFeaturesRequest{})
	return err
}

// ResetSystemFeatures puts the system wide flags back to Zitadel's defaults.
func (c *Client) ResetSystemFeatures(ctx context.Context) error {
	_, err := c.feature.ResetSystemFeatures(ctx, &feature.ResetSystemFeaturesRequest{})
	return err
}

// SetSystemFeatures sets the flags that apply to the system rather than to one
// organization.
func (c *Client) SetSystemFeatures(ctx context.Context, loginDefaultOrg, userSchema bool) error {
	_, err := c.feature.SetSystemFeatures(ctx, &feature.SetSystemFeaturesRequest{
		LoginDefaultOrg: boolPtr(loginDefaultOrg),
		UserSchema:      boolPtr(userSchema),
	})
	return err
}

// GetSystemFeatures reads the system wide feature flags.
func (c *Client) GetSystemFeatures(ctx context.Context) (*FeatureFlags, error) {
	resp, err := c.feature.GetSystemFeatures(ctx, &feature.GetSystemFeaturesRequest{})
	if err != nil {
		return nil, err
	}

	out := &FeatureFlags{}
	if f := resp.GetLoginDefaultOrg(); f != nil {
		out.LoginDefaultOrg = f.GetEnabled()
	}
	if f := resp.GetUserSchema(); f != nil {
		out.UserSchema = f.GetEnabled()
	}

	return out, nil
}

// Restrictions are the instance wide limits on what may be registered.
type Restrictions struct {
	// DisallowPublicOrgRegistration stops anyone signing up their own
	// organization.
	DisallowPublicOrgRegistration bool

	// AllowedLanguages is the set of languages the login screen offers, or empty
	// for every language Zitadel supports.
	AllowedLanguages []string
}

// Input returns the restrictions in the shape the write API accepts, so that a
// recorded snapshot can be written back on deletion.
func (r Restrictions) Input() any { return r }

// SetRestrictions sets the registration restrictions.
func (c *Client) SetRestrictions(ctx context.Context, r Restrictions) error {
	_, err := c.admin.SetRestrictions(ctx, &admin.SetRestrictionsRequest{
		DisallowPublicOrgRegistration: boolPtr(r.DisallowPublicOrgRegistration),
		AllowedLanguages: &admin.SelectLanguages{ //nolint:staticcheck // the admin API still carries the v1 type
			List: r.AllowedLanguages,
		},
	})
	return err
}

// GetRestrictions reads the registration restrictions.
func (c *Client) GetRestrictions(ctx context.Context) (*Restrictions, error) {
	resp, err := c.admin.GetRestrictions(ctx, &admin.GetRestrictionsRequest{})
	if err != nil {
		return nil, err
	}

	return &Restrictions{
		DisallowPublicOrgRegistration: resp.GetDisallowPublicOrgRegistration(),
		AllowedLanguages:              resp.GetAllowedLanguages(),
	}, nil
}

// SecretGenerator is the shape of the codes Zitadel sends out: an init code, a
// verification code, a password reset code, an application secret.
type SecretGenerator struct {
	// Type is which of Zitadel's codes this shapes, such as
	// `SECRET_GENERATOR_TYPE_INIT_CODE`. Each type is a generator of its own, so
	// this is what tells two of them apart.
	Type string

	Length              uint32
	Expiry              string
	IncludeLowerLetters bool
	IncludeUpperLetters bool
	IncludeDigits       bool
	IncludeSymbols      bool
}

// Input returns the generator in the shape the write API accepts, so that a
// recorded snapshot can be written back on deletion.
func (g SecretGenerator) Input() any { return g }

// SecretGeneratorTypes lists the generators Zitadel can be asked to reshape.
//
// It is read from the instance rather than written down here, so that a Zitadel
// release which adds one is covered without a change to this provider.
func (c *Client) SecretGeneratorTypes(ctx context.Context) ([]string, error) {
	resp, err := c.admin.ListSecretGenerators(ctx, &admin.ListSecretGeneratorsRequest{}) //nolint:staticcheck // no v2 equivalent exists
	if err != nil {
		return nil, err
	}

	out := make([]string, 0, len(resp.GetResult()))
	for _, g := range resp.GetResult() {
		t, ok := settings.SecretGeneratorType_name[int32(g.GetGeneratorType())]
		if !ok {
			return nil, fmt.Errorf("zitadel reported the unknown secret generator type %d", int32(g.GetGeneratorType()))
		}

		out = append(out, t)
	}

	return out, nil
}

// SecretGenerator reads one generator, or nil when Zitadel has never configured
// it.
//
// Zitadel answers with a NotFound for a type it has not been told about, which
// is the state a freshly created one is in, so it is turned into a nil rather
// than an error.
func (c *Client) SecretGenerator(ctx context.Context, generatorType string) (*SecretGenerator, error) {
	t, ok := settings.SecretGeneratorType_value[generatorType]
	if !ok {
		return nil, fmt.Errorf("zitadel has no secret generator of type %q: it must be one of %v", generatorType, settings.SecretGeneratorType_name)
	}

	resp, err := c.admin.GetSecretGenerator(ctx, &admin.GetSecretGeneratorRequest{ //nolint:staticcheck // no v2 equivalent exists
		GeneratorType: settings.SecretGeneratorType(t),
	})
	if err != nil {
		if IsNotFound(err) {
			return nil, nil //nolint:nilnil // an unconfigured generator reads as absent, which is what it is
		}

		return nil, err
	}

	g := resp.GetSecretGenerator()
	if g == nil {
		return nil, nil //nolint:nilnil // an absent generator reads as absent
	}

	name, known := settings.SecretGeneratorType_name[int32(g.GetGeneratorType())]
	if !known {
		return nil, fmt.Errorf("zitadel reported the unknown secret generator type %d", int32(g.GetGeneratorType()))
	}

	out := &SecretGenerator{
		Type:                name,
		Length:              g.GetLength(),
		IncludeLowerLetters: g.GetIncludeLowerLetters(),
		IncludeUpperLetters: g.GetIncludeUpperLetters(),
		IncludeDigits:       g.GetIncludeDigits(),
		IncludeSymbols:      g.GetIncludeSymbols(),
	}
	if e := g.GetExpiry(); e != nil {
		out.Expiry = e.AsDuration().String()
	}

	return out, nil
}

// SetSecretGenerator reshapes one of Zitadel's code generators.
//
// There is no create and no remove: a generator that has never been configured
// is updated into being one, and one that has been cannot go back.
func (c *Client) SetSecretGenerator(ctx context.Context, g SecretGenerator) error {
	t, ok := settings.SecretGeneratorType_value[g.Type]
	if !ok {
		return fmt.Errorf("zitadel has no secret generator of type %q: it must be one of %v", g.Type, settings.SecretGeneratorType_name)
	}

	req := &admin.UpdateSecretGeneratorRequest{ //nolint:staticcheck // no v2 equivalent exists
		GeneratorType:       settings.SecretGeneratorType(t),
		Length:              g.Length,
		IncludeLowerLetters: g.IncludeLowerLetters,
		IncludeUpperLetters: g.IncludeUpperLetters,
		IncludeDigits:       g.IncludeDigits,
		IncludeSymbols:      g.IncludeSymbols,
	}
	if g.Expiry != "" {
		d, err := parseExpiry(g.Expiry)
		if err != nil {
			return fmt.Errorf("cannot read the expiry %q: %w", g.Expiry, err)
		}
		req.Expiry = d
	}

	_, err := c.admin.UpdateSecretGenerator(ctx, req) //nolint:staticcheck // no v2 equivalent exists
	return err
}

func boolPtr(b bool) *bool { return &b }

// parseExpiry reads a duration the way the manifest writes one.
func parseExpiry(s string) (*durationpb.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return nil, err
	}

	return durationpb.New(d), nil
}
