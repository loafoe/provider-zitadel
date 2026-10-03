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

// Package instancesettings holds the controllers for the last of Zitadel's
// instance wide settings: the feature flags, the registration restrictions and
// the secret generators.
//
// They differ from the policies in internal/controller/common/policy.go only in
// which four API calls they make, so they run on that same harness rather than
// growing one of their own.
package instancesettings

import (
	"context"
	"math"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// Setup registers the instance wide settings controllers.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, s := range []func(ctrl.Manager, controller.Options) error{
		SetupInstanceFeatures,
		SetupSystemFeatures,
		SetupInstanceRestrictions,
		SetupInstanceSecretGenerator,
	} {
		if err := s(mgr, o); err != nil {
			return err
		}
	}

	return nil
}

// SetupInstanceFeatures registers the InstanceFeatures controller.
func SetupInstanceFeatures(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.InstanceFeaturesGroupKind,
		v1alpha1.InstanceFeaturesGroupVersionKind,
		&v1alpha1.InstanceFeatures{},
		&v1alpha1.InstanceFeaturesList{},
		featuresDriver{instance: true},
	)
}

// SetupSystemFeatures registers the SystemFeatures controller.
func SetupSystemFeatures(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.SystemFeaturesGroupKind,
		v1alpha1.SystemFeaturesGroupVersionKind,
		&v1alpha1.SystemFeatures{},
		&v1alpha1.SystemFeaturesList{},
		featuresDriver{},
	)
}

// SetupInstanceRestrictions registers the InstanceRestrictions controller.
func SetupInstanceRestrictions(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.InstanceRestrictionsGroupKind,
		v1alpha1.InstanceRestrictionsGroupVersionKind,
		&v1alpha1.InstanceRestrictions{},
		&v1alpha1.InstanceRestrictionsList{},
		restrictionsDriver{},
	)
}

// SetupInstanceSecretGenerator registers the InstanceSecretGenerator controller.
func SetupInstanceSecretGenerator(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupPolicyController(
		mgr, o,
		v1alpha1.InstanceSecretGeneratorGroupKind,
		v1alpha1.InstanceSecretGeneratorGroupVersionKind,
		&v1alpha1.InstanceSecretGenerator{},
		&v1alpha1.InstanceSecretGeneratorList{},
		secretGeneratorDriver{},
	)
}

// The feature flags
//
// Zitadel keeps the instance and system flags in two tables with two of their
// three flags in common, so one driver serves both kinds and the only difference
// is which pair of calls it makes. Both are resettable, which is what lets
// deleting the resource hand the instance back its own default rather than
// leaving it configured by a manifest that no longer exists.

type featuresDriver struct {
	// instance selects the instance wide flags rather than the system wide ones.
	instance bool
}

var _ common.PolicyDriver[zitadel.FeatureFlags, zitadel.FeatureFlags] = featuresDriver{}

func (featuresDriver) Kind() string { return "FeatureFlags" }

func (featuresDriver) Scope(_ context.Context, _ client.Client, _ common.ManagedPolicy) (string, error) {
	// Both are the singleton of the instance, so there is no scope to resolve.
	return "", nil
}

func (d featuresDriver) Get(ctx context.Context, c *zitadel.Client, _ string) (zitadel.FeatureFlags, bool, error) {
	if d.instance {
		f, err := c.GetInstanceFeatures(ctx)
		if err != nil {
			return zitadel.FeatureFlags{}, false, err
		}

		return *f, false, nil
	}

	f, err := c.GetSystemFeatures(ctx)
	if err != nil {
		return zitadel.FeatureFlags{}, false, err
	}

	return *f, false, nil
}

func (d featuresDriver) Apply(ctx context.Context, c *zitadel.Client, _ string, want zitadel.FeatureFlags) error {
	if d.instance {
		return c.SetInstanceFeatures(ctx, want)
	}

	return c.SetSystemFeatures(ctx, want.LoginDefaultOrg, want.UserSchema)
}

// Resettable reports that Zitadel can put the flags back to their own default.
//
// This is what makes deleting one of these kinds a real deletion rather than a
// restore: the reset call exists, so the instance is left as it was found.
func (featuresDriver) Resettable() bool { return true }

func (d featuresDriver) Reset(ctx context.Context, c *zitadel.Client, _ string) error {
	if d.instance {
		return c.ResetInstanceFeatures(ctx)
	}

	return c.ResetSystemFeatures(ctx)
}

func (featuresDriver) Desired(cr common.ManagedPolicy) zitadel.FeatureFlags {
	switch fp := cr.(type) {
	case *v1alpha1.InstanceFeatures:
		return zitadel.FeatureFlags{
			LoginDefaultOrg:      common.DerefBool(fp.Spec.ForProvider.LoginDefaultOrg),
			UserSchema:           common.DerefBool(fp.Spec.ForProvider.UserSchema),
			DebugOidcParentError: common.DerefBool(fp.Spec.ForProvider.DebugOidcParentError),
			ImprovedPerformance:  fp.Spec.ForProvider.ImprovedPerformance,
		}

	case *v1alpha1.SystemFeatures:
		// The system table has two flags and no more, so asking for the others
		// would compare them against a read that never reports them.
		return zitadel.FeatureFlags{
			LoginDefaultOrg: common.DerefBool(fp.Spec.ForProvider.LoginDefaultOrg),
			UserSchema:      common.DerefBool(fp.Spec.ForProvider.UserSchema),
		}

	default:
		return zitadel.FeatureFlags{}
	}
}

func (featuresDriver) Report(mg common.ManagedPolicy, observed zitadel.FeatureFlags) {
	switch cr := mg.(type) {
	case *v1alpha1.InstanceFeatures:
		cr.Status.AtProvider.LoginDefaultOrg = observed.LoginDefaultOrg
		cr.Status.AtProvider.UserSchema = observed.UserSchema
		cr.Status.AtProvider.DebugOidcParentError = observed.DebugOidcParentError
		cr.Status.AtProvider.ImprovedPerformance = observed.ImprovedPerformance

	case *v1alpha1.SystemFeatures:
		cr.Status.AtProvider.LoginDefaultOrg = observed.LoginDefaultOrg
		cr.Status.AtProvider.UserSchema = observed.UserSchema
	}
}

// Equal reports whether the flags Zitadel holds are the ones the manifest asks
// for.
//
// The two kinds are compared against their own field sets: a SystemFeatures
// observed value carries no improved performance paths, because Zitadel reports
// none for that table.
func (d featuresDriver) Equal(want zitadel.FeatureFlags, observed zitadel.FeatureFlags) bool {
	if want.LoginDefaultOrg != observed.LoginDefaultOrg || want.UserSchema != observed.UserSchema {
		return false
	}

	if d.instance {
		return want.DebugOidcParentError == observed.DebugOidcParentError &&
			common.EqualStringSlices(want.ImprovedPerformance, observed.ImprovedPerformance)
	}

	return true
}

// The registration restrictions
//
// Zitadel offers no way to put these back, so deleting the resource restores
// the value it overwrote rather than resetting it. That is the same bargain the
// instance wide policies strike, and it is recorded in the status rather than
// guessed at.

type restrictionsDriver struct{}

var _ common.PolicyDriver[zitadel.Restrictions, zitadel.Restrictions] = restrictionsDriver{}

func (restrictionsDriver) Kind() string { return "InstanceRestrictions" }

func (restrictionsDriver) Scope(_ context.Context, _ client.Client, _ common.ManagedPolicy) (string, error) {
	return "", nil
}

func (restrictionsDriver) Get(ctx context.Context, c *zitadel.Client, _ string) (zitadel.Restrictions, bool, error) {
	r, err := c.GetRestrictions(ctx)
	if err != nil {
		return zitadel.Restrictions{}, false, err
	}

	return *r, false, nil
}

func (restrictionsDriver) Apply(ctx context.Context, c *zitadel.Client, _ string, want zitadel.Restrictions) error {
	return c.SetRestrictions(ctx, want)
}

// Resettable reports that these restrictions cannot be reset.
func (restrictionsDriver) Resettable() bool { return false }

func (restrictionsDriver) Reset(_ context.Context, _ *zitadel.Client, _ string) error {
	return errors.New("the instance registration restrictions cannot be reset; deleting the resource restores what it overwrote")
}

func (restrictionsDriver) Desired(cr common.ManagedPolicy) zitadel.Restrictions {
	fp := cr.(*v1alpha1.InstanceRestrictions).Spec.ForProvider

	return zitadel.Restrictions{
		DisallowPublicOrgRegistration: common.DerefBool(fp.DisallowPublicOrgRegistration),
		AllowedLanguages:              fp.AllowedLanguages,
	}
}

func (restrictionsDriver) Report(mg common.ManagedPolicy, observed zitadel.Restrictions) {
	cr := mg.(*v1alpha1.InstanceRestrictions)

	cr.Status.AtProvider.DisallowPublicOrgRegistration = observed.DisallowPublicOrgRegistration
	cr.Status.AtProvider.AllowedLanguages = observed.AllowedLanguages
}

func (restrictionsDriver) Equal(want zitadel.Restrictions, observed zitadel.Restrictions) bool {
	return want.DisallowPublicOrgRegistration == observed.DisallowPublicOrgRegistration &&
		common.EqualStringSlices(want.AllowedLanguages, observed.AllowedLanguages)
}

// The secret generators
//
// There is one generator per code type - an init code, an application secret, a
// password reset code - so unlike the three above this kind has an identity of
// its own. Its scope is the generator type, which is what lets two of them
// exist at once and what a terminating resource acts on.

type secretGeneratorDriver struct{}

var _ common.PolicyDriver[zitadel.SecretGenerator, zitadel.SecretGenerator] = secretGeneratorDriver{}

func (secretGeneratorDriver) Kind() string { return "InstanceSecretGenerator" }

// Scope returns the generator type this resource manages.
//
// A terminating object must not follow a reference that has since moved, so the
// recorded scope wins when there is one. That matters here because a manifest
// that changes its generator type is describing a different object entirely.
func (secretGeneratorDriver) Scope(_ context.Context, _ client.Client, cr common.ManagedPolicy) (string, error) {
	if recorded := cr.PolicyScope(); recorded != "" {
		return recorded, nil
	}

	return common.Deref(cr.(*v1alpha1.InstanceSecretGenerator).Spec.ForProvider.GeneratorType), nil
}

func (secretGeneratorDriver) Get(ctx context.Context, c *zitadel.Client, scope string) (zitadel.SecretGenerator, bool, error) {
	g, err := c.SecretGenerator(ctx, scope)
	switch {
	case err != nil:
		return zitadel.SecretGenerator{}, false, err

	case g == nil:
		// Zitadel has never been told about this generator, so there is nothing
		// to observe. Reporting it as absent is what routes the write through
		// Create.
		return zitadel.SecretGenerator{}, false, nil
	}

	return *g, false, nil
}

func (secretGeneratorDriver) Apply(ctx context.Context, c *zitadel.Client, scope string, want zitadel.SecretGenerator) error {
	// The type travels with the scope rather than with the settings, because
	// that is where Zitadel keeps it.
	want.Type = scope

	return c.SetSecretGenerator(ctx, want)
}

// Resettable reports that a generator cannot be reset: Zitadel can reshape one
// or leave it, but there is no call to say "as it was".
func (secretGeneratorDriver) Resettable() bool { return false }

func (secretGeneratorDriver) Reset(_ context.Context, _ *zitadel.Client, _ string) error {
	return errors.New("a secret generator cannot be reset; deleting the resource restores the generator it replaced")
}

func (secretGeneratorDriver) Desired(cr common.ManagedPolicy) zitadel.SecretGenerator {
	fp := cr.(*v1alpha1.InstanceSecretGenerator).Spec.ForProvider

	return zitadel.SecretGenerator{
		Type: common.Deref(fp.GeneratorType),
		// Zitadel refuses a length outside the range this casts to, so the value
		// is bounded rather than trusted.
		Length:              uint32(max(0, min(common.DerefInt64(fp.Length), math.MaxUint32))),
		Expiry:              common.Deref(fp.Expiry),
		IncludeLowerLetters: common.DerefBool(fp.IncludeLowerLetters),
		IncludeUpperLetters: common.DerefBool(fp.IncludeUpperLetters),
		IncludeDigits:       common.DerefBool(fp.IncludeDigits),
		IncludeSymbols:      common.DerefBool(fp.IncludeSymbols),
	}
}

func (secretGeneratorDriver) Report(mg common.ManagedPolicy, observed zitadel.SecretGenerator) {
	cr := mg.(*v1alpha1.InstanceSecretGenerator)

	cr.Status.AtProvider.GeneratorType = observed.Type
	cr.Status.AtProvider.Length = int64(observed.Length)
	cr.Status.AtProvider.Expiry = observed.Expiry
	cr.Status.AtProvider.IncludeLowerLetters = observed.IncludeLowerLetters
	cr.Status.AtProvider.IncludeUpperLetters = observed.IncludeUpperLetters
	cr.Status.AtProvider.IncludeDigits = observed.IncludeDigits
	cr.Status.AtProvider.IncludeSymbols = observed.IncludeSymbols
}

func (secretGeneratorDriver) Equal(want zitadel.SecretGenerator, observed zitadel.SecretGenerator) bool {
	return want.Type == observed.Type &&
		want.Length == observed.Length &&
		sameDuration(want.Expiry, observed.Expiry) &&
		want.IncludeLowerLetters == observed.IncludeLowerLetters &&
		want.IncludeUpperLetters == observed.IncludeUpperLetters &&
		want.IncludeDigits == observed.IncludeDigits &&
		want.IncludeSymbols == observed.IncludeSymbols
}

// sameDuration compares two durations by what they mean rather than by how they
// are written.
//
// Zitadel hands back what it was given, normalised: a manifest that says `5m`
// is read back as `5m0s`, and a straight string comparison would call that drift
// on every reconcile and never settle.
func sameDuration(want, observed string) bool {
	if want == observed {
		return true
	}

	w, err := time.ParseDuration(want)
	if err != nil {
		// An expiry that cannot be parsed is reported by Zitadel on the write
		// rather than here, so it is compared as written and fails loudly.
		return false
	}

	o, err := time.ParseDuration(observed)
	if err != nil {
		return false
	}

	return w == o
}
