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

// Package domains holds the controllers for the three Zitadel domain lists: the
// custom domains an instance answers on, the domains that may request one of
// its tokens, and the domains an organization owns.
//
// They differ only in which of the six calls they make and in whether they need
// an organization, so they share the harness in internal/controller/common/named.go.
package domains

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// Setup registers the domain controllers.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, s := range []func(ctrl.Manager, controller.Options) error{
		SetupInstanceCustomDomain,
		SetupInstanceTrustedDomain,
		SetupOrganizationDomain,
	} {
		if err := s(mgr, o); err != nil {
			return err
		}
	}

	return nil
}

// SetupInstanceCustomDomain registers the InstanceCustomDomain controller.
func SetupInstanceCustomDomain(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupNamedController(
		mgr, o,
		v1alpha1.InstanceCustomDomainGroupKind,
		v1alpha1.InstanceCustomDomainGroupVersionKind,
		&v1alpha1.InstanceCustomDomain{},
		&v1alpha1.InstanceCustomDomainList{},
		customDriver{},
	)
}

// SetupInstanceTrustedDomain registers the InstanceTrustedDomain controller.
func SetupInstanceTrustedDomain(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupNamedController(
		mgr, o,
		v1alpha1.InstanceTrustedDomainGroupKind,
		v1alpha1.InstanceTrustedDomainGroupVersionKind,
		&v1alpha1.InstanceTrustedDomain{},
		&v1alpha1.InstanceTrustedDomainList{},
		trustedDriver{},
	)
}

// SetupOrganizationDomain registers the OrganizationDomain controller.
func SetupOrganizationDomain(mgr ctrl.Manager, o controller.Options) error {
	return common.SetupNamedController(
		mgr, o,
		v1alpha1.OrganizationDomainGroupKind,
		v1alpha1.OrganizationDomainGroupVersionKind,
		&v1alpha1.OrganizationDomain{},
		&v1alpha1.OrganizationDomainList{},
		orgDriver{},
	)
}

// customDriver reconciles a custom domain of the instance.
//
// Zitadel answers on one generated domain as well as the ones added here. It is
// left out of the list the client returns, because it is not something an
// operator added and Zitadel will not remove it.
type customDriver struct{}

var _ common.NamedDriver[*v1alpha1.InstanceCustomDomain] = customDriver{}

func (customDriver) Kind() string { return "InstanceCustomDomain" }

// Scope is the instance: a custom domain belongs to no organization.
func (customDriver) Scope(_ context.Context, _ client.Client, _ *v1alpha1.InstanceCustomDomain) (string, error) {
	return "", nil
}

func (customDriver) List(ctx context.Context, c *zitadel.Client, _ string) ([]zitadel.Domain, error) {
	return c.ListCustomDomains(ctx)
}

func (customDriver) Add(ctx context.Context, c *zitadel.Client, _ string, name string) error {
	return c.AddCustomDomain(ctx, name)
}

func (customDriver) Remove(ctx context.Context, c *zitadel.Client, _ string, name string) error {
	return c.RemoveCustomDomain(ctx, name)
}

func (customDriver) Report(cr *v1alpha1.InstanceCustomDomain, entry zitadel.Domain) {
	cr.Status.AtProvider.Domain = entry.Name
	if entry.IsPrimary != nil {
		cr.Status.AtProvider.IsPrimary = *entry.IsPrimary
	}
}

// trustedDriver reconciles a trusted domain of the instance.
type trustedDriver struct{}

var _ common.NamedDriver[*v1alpha1.InstanceTrustedDomain] = trustedDriver{}

func (trustedDriver) Kind() string { return "InstanceTrustedDomain" }

func (trustedDriver) Scope(_ context.Context, _ client.Client, _ *v1alpha1.InstanceTrustedDomain) (string, error) {
	return "", nil
}

func (trustedDriver) List(ctx context.Context, c *zitadel.Client, _ string) ([]zitadel.Domain, error) {
	return c.ListTrustedDomains(ctx)
}

func (trustedDriver) Add(ctx context.Context, c *zitadel.Client, _ string, name string) error {
	return c.AddTrustedDomain(ctx, name)
}

func (trustedDriver) Remove(ctx context.Context, c *zitadel.Client, _ string, name string) error {
	return c.RemoveTrustedDomain(ctx, name)
}

func (trustedDriver) Report(cr *v1alpha1.InstanceTrustedDomain, entry zitadel.Domain) {
	cr.Status.AtProvider.Domain = entry.Name
}

// orgDriver reconciles a domain an organization owns.
//
// It is the only one of the three that needs an organization, and the only one
// with anything else to do: Zitadel adds a domain unverified and only marks it
// verified once the organization has published the proof, so asking for that
// check is part of reconciling one.
type orgDriver struct{}

var _ common.NamedDriver[*v1alpha1.OrganizationDomain] = orgDriver{}

func (orgDriver) Kind() string { return "OrganizationDomain" }

func (orgDriver) Scope(ctx context.Context, kube client.Client, org *v1alpha1.OrganizationDomain) (string, error) {
	fp := org.Spec.ForProvider

	// A ProviderConfig may name a default organization, which is what an
	// organization domain with no reference of its own belongs to.
	orgDefault, err := common.ProviderConfigOrganizationID(ctx, kube, org)
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	orgID, err := common.ResolveOrganizationID(ctx, kube, org, fp.OrganizationRef,
		fp.OrganizationSelector, fp.OrganizationID, orgDefault,
		common.CurrentIfDeleting(meta.WasDeleted(org), common.StringPtr(org.Status.Scope)))
	if err != nil {
		return "", common.Join(common.ErrNoOrganizationID, err)
	}

	if orgID == "" {
		return "", errors.New(common.ErrNoOrganizationID.Error())
	}

	return orgID, nil
}

func (orgDriver) List(ctx context.Context, c *zitadel.Client, scope string) ([]zitadel.Domain, error) {
	return c.ListOrganizationDomains(ctx, scope)
}

func (orgDriver) Add(ctx context.Context, c *zitadel.Client, scope string, name string) error {
	return c.AddOrganizationDomain(ctx, scope, name)
}

func (orgDriver) Remove(ctx context.Context, c *zitadel.Client, scope string, name string) error {
	return c.RemoveOrganizationDomain(ctx, scope, name)
}

// Report writes what Zitadel says about the domain.
func (orgDriver) Report(cr *v1alpha1.OrganizationDomain, entry zitadel.Domain) {
	cr.Status.AtProvider.Domain = entry.Name
	cr.Status.AtProvider.ValidationType = entry.ValidationType

	if entry.IsVerified != nil {
		cr.Status.AtProvider.IsVerified = *entry.IsVerified
	}
	if entry.IsPrimary != nil {
		cr.Status.AtProvider.IsPrimary = *entry.IsPrimary
	}
}

// Verify asks Zitadel what the organization still has to publish to prove it
// owns the domain.
//
// It is asked for only when the manifest asked for the check and the domain is
// not verified yet. Asking again once the proof is in would report a fresh
// token and URL for a domain that no longer needs proving.
func (orgDriver) Verify(ctx context.Context, c *zitadel.Client, scope string, cr *v1alpha1.OrganizationDomain, entry zitadel.Domain) {
	cr.Status.AtProvider.ValidationToken = ""
	cr.Status.AtProvider.ValidationURL = ""

	if !common.DerefBool(cr.Spec.ForProvider.Verify) || (entry.IsVerified != nil && *entry.IsVerified) {
		return
	}

	validationType := common.Deref(cr.Spec.ForProvider.ValidationType)
	if validationType == "" {
		validationType = entry.ValidationType
	}

	out, err := c.VerifyOrganizationDomain(ctx, scope, entry.Name, validationType)
	if err != nil {
		// Nothing is reported when Zitadel will not say, rather than reporting
		// an empty proof that looks like there is nothing to publish.
		return
	}

	cr.Status.AtProvider.ValidationType = out.ValidationType
	cr.Status.AtProvider.ValidationToken = out.Token
	cr.Status.AtProvider.ValidationURL = out.URL
}
