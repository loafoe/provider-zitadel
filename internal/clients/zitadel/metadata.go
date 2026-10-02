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
	"sort"

	filterv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/filter/v2"
	orgv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/org/v2"
	userv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user/v2"
)

// MetadataEntry is a single metadata key/value pair.
//
// These resources manage the complete metadata set of one user or organization
// rather than one key per resource, so that two resources cannot fight over the
// same subject by each replacing the other's entries.
//
// Zitadel's metadata endpoints are additive: writing a key adds it or overwrites
// it, and keys that are not written are left alone. Conveying "this is the
// whole set" therefore takes both a write and an explicit delete of the keys
// that are no longer wanted, which is what ApplyUserMetadata and
// ApplyOrganizationMetadata do.
type MetadataEntry struct {
	Key   string
	Value string
}

// Normalize sorts entries by key and drops empty values, so that two
// equivalent desired states compare equal regardless of the order they were
// written in.
func Normalize(entries []MetadataEntry) []MetadataEntry {
	out := make([]MetadataEntry, 0, len(entries))
	for _, e := range entries {
		if e.Key == "" {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })

	return out
}

// EqualMetadata reports whether two metadata sets are equivalent.
func EqualMetadata(a, b []MetadataEntry) bool {
	na, nb := Normalize(a), Normalize(b)
	if len(na) != len(nb) {
		return false
	}
	for i := range na {
		if na[i] != nb[i] {
			return false
		}
	}

	return true
}

// GetUserMetadata returns the complete metadata set of a user.
func (c *Client) GetUserMetadata(ctx context.Context, userID string) ([]MetadataEntry, error) {
	resp, err := c.user.ListUserMetadata(ctx, &userv2.ListUserMetadataRequest{
		UserId:     userID,
		Pagination: &filterv2.PaginationRequest{Limit: 1000},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the metadata of user %s: %w", userID, err)
	}

	out := make([]MetadataEntry, 0, len(resp.GetMetadata()))
	for _, m := range resp.GetMetadata() {
		out = append(out, MetadataEntry{Key: m.GetKey(), Value: string(m.GetValue())})
	}

	return Normalize(out), nil
}

// ApplyUserMetadata makes the user's metadata set exactly the given entries.
//
// Writing the wanted entries is not enough, because Zitadel keeps every key that
// is not written. Keys that exist on the user but are no longer wanted are
// deleted, so that dropping an entry from the spec removes it from Zitadel.
func (c *Client) ApplyUserMetadata(ctx context.Context, userID string, entries []MetadataEntry) error {
	desired := Normalize(entries)

	current, err := c.GetUserMetadata(ctx, userID)
	if err != nil {
		return err
	}

	if keys := staleKeys(current, desired); len(keys) > 0 {
		if _, err := c.user.DeleteUserMetadata(ctx, &userv2.DeleteUserMetadataRequest{
			UserId: userID,
			Keys:   keys,
		}); err != nil {
			return fmt.Errorf("cannot delete the metadata of user %s: %w", userID, err)
		}
	}

	if len(desired) == 0 {
		return nil
	}

	req := &userv2.SetUserMetadataRequest{UserId: userID}
	for _, e := range desired {
		req.Metadata = append(req.Metadata, &userv2.Metadata{Key: e.Key, Value: []byte(e.Value)})
	}

	if _, err := c.user.SetUserMetadata(ctx, req); err != nil {
		return fmt.Errorf("cannot set the metadata of user %s: %w", userID, err)
	}

	return nil
}

// DeleteUserMetadata removes every metadata key of a user.
func (c *Client) DeleteUserMetadata(ctx context.Context, userID string) error {
	current, err := c.GetUserMetadata(ctx, userID)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}

		return err
	}

	if len(current) == 0 {
		return nil
	}

	keys := make([]string, 0, len(current))
	for _, e := range current {
		keys = append(keys, e.Key)
	}

	if _, err := c.user.DeleteUserMetadata(ctx, &userv2.DeleteUserMetadataRequest{
		UserId: userID,
		Keys:   keys,
	}); err != nil {
		return fmt.Errorf("cannot delete the metadata of user %s: %w", userID, err)
	}

	return nil
}

// staleKeys returns the keys that exist on the subject but are absent from the
// desired set.
func staleKeys(current, desired []MetadataEntry) []string {
	wanted := make(map[string]struct{}, len(desired))
	for _, e := range desired {
		wanted[e.Key] = struct{}{}
	}

	var keys []string

	for _, e := range current {
		if _, ok := wanted[e.Key]; !ok {
			keys = append(keys, e.Key)
		}
	}

	return keys
}

// GetOrganizationMetadata returns the complete metadata set of an organization.
func (c *Client) GetOrganizationMetadata(ctx context.Context, orgID string) ([]MetadataEntry, error) {
	resp, err := c.org.ListOrganizationMetadata(ctx, &orgv2.ListOrganizationMetadataRequest{
		OrganizationId: orgID,
		Pagination:     &filterv2.PaginationRequest{Limit: 1000},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the metadata of organization %s: %w", orgID, err)
	}

	out := make([]MetadataEntry, 0, len(resp.GetMetadata()))
	for _, m := range resp.GetMetadata() {
		out = append(out, MetadataEntry{Key: m.GetKey(), Value: string(m.GetValue())})
	}

	return Normalize(out), nil
}

// ApplyOrganizationMetadata makes the organization's metadata set exactly the
// given entries, deleting any key that is no longer wanted.
func (c *Client) ApplyOrganizationMetadata(ctx context.Context, orgID string, entries []MetadataEntry) error {
	desired := Normalize(entries)

	current, err := c.GetOrganizationMetadata(ctx, orgID)
	if err != nil {
		return err
	}

	if keys := staleKeys(current, desired); len(keys) > 0 {
		if _, err := c.org.DeleteOrganizationMetadata(ctx, &orgv2.DeleteOrganizationMetadataRequest{
			OrganizationId: orgID,
			Keys:           keys,
		}); err != nil {
			return fmt.Errorf("cannot delete the metadata of organization %s: %w", orgID, err)
		}
	}

	if len(desired) == 0 {
		return nil
	}

	req := &orgv2.SetOrganizationMetadataRequest{OrganizationId: orgID}
	for _, e := range desired {
		req.Metadata = append(req.Metadata, &orgv2.Metadata{Key: e.Key, Value: []byte(e.Value)})
	}

	if _, err := c.org.SetOrganizationMetadata(ctx, req); err != nil {
		return fmt.Errorf("cannot set the metadata of organization %s: %w", orgID, err)
	}

	return nil
}

// DeleteOrganizationMetadata removes every metadata key of an organization.
func (c *Client) DeleteOrganizationMetadata(ctx context.Context, orgID string) error {
	current, err := c.GetOrganizationMetadata(ctx, orgID)
	if err != nil {
		if IsNotFound(err) {
			return nil
		}

		return err
	}

	if len(current) == 0 {
		return nil
	}

	keys := make([]string, 0, len(current))
	for _, e := range current {
		keys = append(keys, e.Key)
	}

	if _, err := c.org.DeleteOrganizationMetadata(ctx, &orgv2.DeleteOrganizationMetadataRequest{
		OrganizationId: orgID,
		Keys:           keys,
	}); err != nil {
		return fmt.Errorf("cannot delete the metadata of organization %s: %w", orgID, err)
	}

	return nil
}
