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

	instancev2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/instance/v2"
	orgv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/org/v2"
)

// Domains
//
// Zitadel has three sets of domains, all of them lists of names rather than
// settings: the custom domains an instance answers on, the trusted domains that
// may request one, and the domains an organization owns.
//
// A domain has no identifier of its own - its name is the identity - so all
// three are one add, one list and one remove.

// Domain is one entry of a list of domains.
//
// What Zitadel reports beyond the name differs per list, so the fields are
// pointers: absent means Zitadel did not say, rather than that it is false.
type Domain struct {
	Name string

	// IsVerified reports whether an organization's domain has been proved to
	// belong to it.
	IsVerified *bool

	// IsPrimary reports whether this is the domain its organization signs users
	// in with.
	IsPrimary *bool

	// ValidationType is how the domain was proved, `DOMAIN_VALIDATION_TYPE_DNS`
	// or `DOMAIN_VALIDATION_TYPE_HTTP`.
	ValidationType string

	// Token and URL are what the organization has to publish or follow to prove
	// the domain, and are only present while that is outstanding.
	Token string
	URL   string
}

// Input returns the domain in the shape the write API accepts.
//
// The shared harness restores a recorded snapshot by turning it back into a
// writable shape through this method.
func (d Domain) Input() any { return d }

// ListCustomDomains returns the domains this instance answers on, beyond the
// generated one it made for itself.
func (c *Client) ListCustomDomains(ctx context.Context) ([]Domain, error) {
	resp, err := c.instance.ListCustomDomains(ctx, &instancev2.ListCustomDomainsRequest{})
	if err != nil {
		return nil, err
	}

	out := make([]Domain, 0, len(resp.GetDomains()))
	for _, d := range resp.GetDomains() {
		// The generated domain is the one Zitadel made for itself; it is not
		// something an operator adds, so it is left out of the list this
		// provider reconciles.
		if d.GetGenerated() {
			continue
		}

		out = append(out, Domain{
			Name:      d.GetDomain(),
			IsPrimary: boolPtr(d.GetPrimary()),
		})
	}

	return out, nil
}

// AddCustomDomain makes this instance answer on a domain.
func (c *Client) AddCustomDomain(ctx context.Context, domain string) error {
	_, err := c.instance.AddCustomDomain(ctx, &instancev2.AddCustomDomainRequest{CustomDomain: domain})
	return err
}

// RemoveCustomDomain stops this instance answering on a domain.
func (c *Client) RemoveCustomDomain(ctx context.Context, domain string) error {
	_, err := c.instance.RemoveCustomDomain(ctx, &instancev2.RemoveCustomDomainRequest{CustomDomain: domain})
	return err
}

// ListTrustedDomains returns the domains allowed to request one of this
// instance's tokens.
func (c *Client) ListTrustedDomains(ctx context.Context) ([]Domain, error) {
	resp, err := c.instance.ListTrustedDomains(ctx, &instancev2.ListTrustedDomainsRequest{})
	if err != nil {
		return nil, err
	}

	out := make([]Domain, 0, len(resp.GetTrustedDomain()))
	for _, d := range resp.GetTrustedDomain() {
		out = append(out, Domain{Name: d.GetDomain()})
	}

	return out, nil
}

// AddTrustedDomain lets a domain request one of this instance's tokens.
func (c *Client) AddTrustedDomain(ctx context.Context, domain string) error {
	_, err := c.instance.AddTrustedDomain(ctx, &instancev2.AddTrustedDomainRequest{TrustedDomain: domain})
	return err
}

// RemoveTrustedDomain stops a domain being able to request one of this
// instance's tokens.
func (c *Client) RemoveTrustedDomain(ctx context.Context, domain string) error {
	_, err := c.instance.RemoveTrustedDomain(ctx, &instancev2.RemoveTrustedDomainRequest{TrustedDomain: domain})
	return err
}

// ListOrganizationDomains returns the domains an organization owns.
func (c *Client) ListOrganizationDomains(ctx context.Context, orgID string) ([]Domain, error) {
	resp, err := c.orgDomain.ListOrganizationDomains(ctx, &orgv2.ListOrganizationDomainsRequest{
		OrganizationId: orgID,
	})
	if err != nil {
		return nil, err
	}

	out := make([]Domain, 0, len(resp.GetDomains()))
	for _, d := range resp.GetDomains() {
		// The validation type arrives as a number, so it is read through its name
		// map rather than converted straight to a string, which would yield the
		// character with that code point. The unspecified type is 0 and is not an
		// error: Zitadel sends it for a domain that has never been proved.
		name, ok := orgv2.DomainValidationType_name[int32(d.GetValidationType())]
		if !ok {
			return nil, fmt.Errorf("zitadel reported the unknown domain validation type %d",
				int32(d.GetValidationType()))
		}

		out = append(out, Domain{
			Name:           d.GetDomain(),
			IsVerified:     boolPtr(d.GetIsVerified()),
			IsPrimary:      boolPtr(d.GetIsPrimary()),
			ValidationType: name,
		})
	}

	return out, nil
}

// AddOrganizationDomain gives an organization a domain.
//
// The domain is added unverified: Zitadel only marks it verified once the
// organization publishes the token or follows the URL that
// VerifyOrganizationDomain reports.
func (c *Client) AddOrganizationDomain(ctx context.Context, orgID, domain string) error {
	_, err := c.orgDomain.AddOrganizationDomain(ctx, &orgv2.AddOrganizationDomainRequest{
		OrganizationId: orgID,
		Domain:         domain,
	})

	return err
}

// RemoveOrganizationDomain takes a domain away from an organization.
func (c *Client) RemoveOrganizationDomain(ctx context.Context, orgID, domain string) error {
	_, err := c.orgDomain.DeleteOrganizationDomain(ctx, &orgv2.DeleteOrganizationDomainRequest{
		OrganizationId: orgID,
		Domain:         domain,
	})

	return err
}

// VerifyOrganizationDomain asks Zitadel to check that an organization has proved
// it owns a domain, and reports what the organization has to publish or follow.
//
// The check is not immediate: the token and URL are what get published, and the
// organization only becomes verified once Zitadel sees them. So this returns
// what is outstanding rather than whether it worked.
func (c *Client) VerifyOrganizationDomain(ctx context.Context, orgID, domain string, validationType string) (*Domain, error) {
	if _, ok := orgv2.DomainValidationType_value[validationType]; !ok {
		return nil, fmt.Errorf("zitadel has no domain validation type %q: it must be one of %v",
			validationType, orgv2.DomainValidationType_name)
	}

	resp, err := c.orgDomain.GenerateOrganizationDomainValidation(ctx, &orgv2.GenerateOrganizationDomainValidationRequest{
		OrganizationId: orgID,
		Domain:         domain,
		Type:           orgv2.DomainValidationType(orgv2.DomainValidationType_value[validationType]),
	})
	if err != nil {
		return nil, err
	}

	return &Domain{
		Name:           domain,
		ValidationType: validationType,
		Token:          resp.GetToken(),
		URL:            resp.GetUrl(),
	}, nil
}
