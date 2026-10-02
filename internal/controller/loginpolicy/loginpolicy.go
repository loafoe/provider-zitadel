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

// Package loginpolicy implements the controller for the LoginPolicy managed
// resource - an organization's login and authentication policy.
package loginpolicy

import (
	"context"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

const errNotLoginPolicy = "managed resource is not a LoginPolicy custom resource"

// Setup adds a controller that reconciles LoginPolicy managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupManagedResourceController(
		mgr, o,
		v1alpha1.LoginPolicyGroupKind,
		v1alpha1.LoginPolicyGroupVersionKind,
		&v1alpha1.LoginPolicy{},
		&v1alpha1.LoginPolicyList{},
		newExternal,
	)
}

type external struct {
	kube   client.Client
	client *zitadel.Client
}

func newExternal(_ context.Context, kube client.Client, _ resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
	return &external{kube: kube, client: zc}, nil
}

// Observe reports whether the organization has a custom login policy and
// whether it matches the desired state.
func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.LoginPolicy)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotLoginPolicy)
	}

	orgID, err := e.organizationID(ctx, cr)
	if err != nil {
		if meta.WasDeleted(cr) {
			// Nothing resolves while an object is terminating, so report the
			// external resource as gone. That lets the reconciler run Delete,
			// which lets the finalizer go, instead of retrying an observation
			// that can never succeed.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	// Zitadel keeps exactly one login policy per organization, so the
	// organization is the external name.
	if meta.GetExternalName(cr) == "" {
		meta.SetExternalName(cr, orgID)
	}

	p, err := e.client.GetLoginPolicy(ctx, orgID)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get the login policy from Zitadel")
	}

	// While the organization still uses the instance default, the policy is
	// reported as not existing so that it is created.
	if p.IsDefault {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	updateStatus(cr, orgID, p)
	cr.Status.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate(cr, p),
	}, nil
}

// Create installs a custom login policy for the organization.
func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.LoginPolicy)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotLoginPolicy)
	}

	cr.Status.SetConditions(xpv1.Creating())

	orgID, err := e.organizationID(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	if err := e.client.SetLoginPolicy(ctx, orgID, desired(cr)); err != nil {
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, orgID)

	return managed.ExternalCreation{}, nil
}

// Update applies the changed fields of the login policy.
func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, ok := mg.(*v1alpha1.LoginPolicy)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotLoginPolicy)
	}

	cr := mg.(*v1alpha1.LoginPolicy)

	// SetLoginPolicy distinguishes create from update itself and reconciles the
	// factor lists by diffing them, so Create and Update are the same call.
	orgID, err := e.organizationID(ctx, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.client.SetLoginPolicy(ctx, orgID, desired(cr)); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

// Delete reverts the organization to the instance default login policy.
// Zitadel has no way to remove a custom policy, so reverting is the closest
// equivalent to a delete and is what an operator means by it.
func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.LoginPolicy)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotLoginPolicy)
	}

	cr.Status.SetConditions(xpv1.Deleting())

	if common.NeverCreated(cr.Status.AtProvider.OrganizationID) {
		// Nothing was ever created in Zitadel, so there is nothing to detach
		// from. Returning success lets the finalizer go instead of leaving the
		// object stuck.
		return managed.ExternalDelete{}, nil
	}

	if err := e.client.ResetLoginPolicy(ctx, common.Deref(cr.Status.AtProvider.OrganizationID)); err != nil {
		return managed.ExternalDelete{}, err
	}

	return managed.ExternalDelete{}, nil
}

// Disconnect releases the underlying Zitadel client.
func (e *external) Disconnect(ctx context.Context) error {
	return e.client.Close()
}

// organizationID resolves the organization the policy belongs to.
func (e *external) organizationID(ctx context.Context, cr *v1alpha1.LoginPolicy) (string, error) {
	fp := cr.Spec.ForProvider

	orgDefault, err := common.ProviderConfigOrganizationID(ctx, e.kube, cr)
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	orgID, err := common.ResolveOrganizationID(ctx, e.kube, cr, fp.OrganizationRef, fp.OrganizationSelector, fp.OrganizationID, orgDefault,
		common.CurrentIfDeleting(meta.WasDeleted(cr), cr.Status.AtProvider.OrganizationID))
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	if orgID == "" {
		return "", errors.New(common.ErrNoOrganizationID.Error())
	}

	return orgID, nil
}

// desired translates the spec into the client input, applying the documented
// defaults for every field the spec leaves unset. The fields are pointers so
// that "unset" is distinguishable from "false", which matters for a policy
// where false is the restrictive value.
func desired(cr *v1alpha1.LoginPolicy) zitadel.LoginPolicyInput {
	fp := cr.Spec.ForProvider

	in := zitadel.LoginPolicyInput{
		AllowUsernamePassword:      valueOr(fp.AllowUsernamePassword, true),
		AllowRegister:              valueOr(fp.AllowRegister, false),
		AllowExternalIDP:           valueOr(fp.AllowExternalIDP, true),
		ForceMFA:                   valueOr(fp.ForceMFA, false),
		ForceMFALocalOnly:          valueOr(fp.ForceMFALocalOnly, false),
		HidePasswordReset:          valueOr(fp.HidePasswordReset, false),
		IgnoreUnknownUsernames:     valueOr(fp.IgnoreUnknownUsernames, true),
		AllowDomainDiscovery:       valueOr(fp.AllowDomainDiscovery, false),
		DisableLoginWithEmail:      valueOr(fp.DisableLoginWithEmail, false),
		DisableLoginWithPhone:      valueOr(fp.DisableLoginWithPhone, false),
		DefaultRedirectURI:         common.Deref(fp.DefaultRedirectURI),
		PasswordlessType:           common.Value(fp.PasswordlessType),
		PasswordCheckLifetime:      common.Deref(fp.PasswordCheckLifetime),
		ExternalLoginCheckLifetime: common.Deref(fp.ExternalLoginCheckLifetime),
		MFAInitSkipLifetime:        common.Deref(fp.MFAInitSkipLifetime),
		SecondFactorCheckLifetime:  common.Deref(fp.SecondFactorCheckLifetime),
		MultiFactorCheckLifetime:   common.Deref(fp.MultiFactorCheckLifetime),
		IDPs:                       fp.IDPs,
	}

	for _, f := range fp.SecondFactors {
		in.SecondFactors = append(in.SecondFactors, string(f))
	}

	for _, f := range fp.MultiFactors {
		in.MultiFactors = append(in.MultiFactors, string(f))
	}

	// Zitadel reports "NotAllowed" rather than an empty value, so the default is
	// spelled out here or every reconcile would see drift.
	if in.PasswordlessType == "" {
		in.PasswordlessType = string(v1alpha1.PasswordlessTypeNotAllowed)
	}

	return in
}

// valueOr dereferences a bool, falling back to a default.
func valueOr(b *bool, fallback bool) bool {
	if b == nil {
		return fallback
	}

	return *b
}

// isUpToDate reports whether the remote policy matches the desired state.
//
// Zitadel normalises durations, so "8760h" and "365d" are the same policy and
// are compared as durations rather than as strings.
//
//nolint:gocyclo // flat comparison of every policy field
func isUpToDate(cr *v1alpha1.LoginPolicy, p *zitadel.LoginPolicy) bool {
	want := desired(cr)

	if want.AllowUsernamePassword != p.AllowUsernamePassword ||
		want.AllowRegister != p.AllowRegister ||
		want.AllowExternalIDP != p.AllowExternalIDP ||
		want.ForceMFA != p.ForceMFA ||
		want.ForceMFALocalOnly != p.ForceMFALocalOnly ||
		want.HidePasswordReset != p.HidePasswordReset ||
		want.IgnoreUnknownUsernames != p.IgnoreUnknownUsernames ||
		want.AllowDomainDiscovery != p.AllowDomainDiscovery ||
		want.DisableLoginWithEmail != p.DisableLoginWithEmail ||
		want.DisableLoginWithPhone != p.DisableLoginWithPhone ||
		want.DefaultRedirectURI != p.DefaultRedirectURI ||
		want.PasswordlessType != p.PasswordlessType {
		return false
	}

	for _, d := range []struct{ want, got string }{
		{want.PasswordCheckLifetime, p.PasswordCheckLifetime},
		{want.ExternalLoginCheckLifetime, p.ExternalLoginCheckLifetime},
		{want.MFAInitSkipLifetime, p.MFAInitSkipLifetime},
		{want.SecondFactorCheckLifetime, p.SecondFactorCheckLifetime},
		{want.MultiFactorCheckLifetime, p.MultiFactorCheckLifetime},
	} {
		if !sameDuration(d.want, d.got) {
			return false
		}
	}

	if !sameStringSet(want.IDPs, p.IDPs) {
		return false
	}

	wantSecond := want.SecondFactors
	if !sameStringSet(wantSecond, p.SecondFactors) {
		return false
	}

	return sameStringSet(want.MultiFactors, p.MultiFactors)
}

// sameDuration compares two duration strings as durations.
func sameDuration(a, b string) bool {
	if a == b {
		return true
	}

	da, err := parseLifetime(a)
	if err != nil {
		return false
	}

	db, err := parseLifetime(b)
	if err != nil {
		return false
	}

	return da == db
}

// parseLifetime reads a lifetime, treating an unset value as zero.
//
// Zitadel reports a lifetime that was never configured as "0s", while an unset
// field in the spec is empty. Both spellings of "never" have to compare equal or
// every reconcile would see drift.
func parseLifetime(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}

	return time.ParseDuration(s)
}

// sameStringSet compares two string sets ignoring order.
func sameStringSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	seen := make(map[string]int, len(a))
	for _, v := range a {
		seen[v]++
	}

	for _, v := range b {
		seen[v]--
		if seen[v] < 0 {
			return false
		}
	}

	return true
}

// updateStatus copies the observed policy into the status of cr.
func updateStatus(cr *v1alpha1.LoginPolicy, orgID string, p *zitadel.LoginPolicy) {
	cr.Status.AtProvider.OrganizationID = common.StringPtr(orgID)
	cr.Status.AtProvider.IsDefault = common.BoolPtr(p.IsDefault)
	cr.Status.AtProvider.AllowUsernamePassword = common.BoolPtr(p.AllowUsernamePassword)
	cr.Status.AtProvider.AllowRegister = common.BoolPtr(p.AllowRegister)
	cr.Status.AtProvider.AllowExternalIDP = common.BoolPtr(p.AllowExternalIDP)
	cr.Status.AtProvider.ForceMFA = common.BoolPtr(p.ForceMFA)
	cr.Status.AtProvider.ForceMFALocalOnly = common.BoolPtr(p.ForceMFALocalOnly)
	cr.Status.AtProvider.HidePasswordReset = common.BoolPtr(p.HidePasswordReset)
	cr.Status.AtProvider.IgnoreUnknownUsernames = common.BoolPtr(p.IgnoreUnknownUsernames)
	cr.Status.AtProvider.AllowDomainDiscovery = common.BoolPtr(p.AllowDomainDiscovery)
	cr.Status.AtProvider.DisableLoginWithEmail = common.BoolPtr(p.DisableLoginWithEmail)
	cr.Status.AtProvider.DisableLoginWithPhone = common.BoolPtr(p.DisableLoginWithPhone)
	cr.Status.AtProvider.DefaultRedirectURI = common.StringPtr(p.DefaultRedirectURI)
	cr.Status.AtProvider.PasswordCheckLifetime = common.StringPtr(p.PasswordCheckLifetime)
	cr.Status.AtProvider.ExternalLoginCheckLifetime = common.StringPtr(p.ExternalLoginCheckLifetime)
	cr.Status.AtProvider.MFAInitSkipLifetime = common.StringPtr(p.MFAInitSkipLifetime)
	cr.Status.AtProvider.SecondFactorCheckLifetime = common.StringPtr(p.SecondFactorCheckLifetime)
	cr.Status.AtProvider.MultiFactorCheckLifetime = common.StringPtr(p.MultiFactorCheckLifetime)
	cr.Status.AtProvider.IDPs = p.IDPs

	if p.PasswordlessType != "" {
		pl := v1alpha1.PasswordlessType(p.PasswordlessType)
		cr.Status.AtProvider.PasswordlessType = &pl
	}

	cr.Status.AtProvider.SecondFactors = nil
	for _, f := range p.SecondFactors {
		cr.Status.AtProvider.SecondFactors = append(cr.Status.AtProvider.SecondFactors, v1alpha1.SecondFactor(f))
	}

	cr.Status.AtProvider.MultiFactors = nil
	for _, f := range p.MultiFactors {
		cr.Status.AtProvider.MultiFactors = append(cr.Status.AtProvider.MultiFactors, v1alpha1.MultiFactor(f))
	}
}
