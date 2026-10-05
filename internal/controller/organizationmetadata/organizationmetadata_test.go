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

package organizationmetadata

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Test values, named so that a case reads as a statement about behaviour
// rather than about a repeated string.
const (
	keyA = "alpha"
)

// Organization metadata replaces a whole set rather than adding one key, so that
// two resources cannot fight over the same organization. Normalising the set is
// what stops a different write order from looking like drift on every poll.

func TestDesired(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     v1alpha1.MetadataList
		want   []zitadel.MetadataEntry
	}{
		"Empty": {
			reason: "No metadata asks for nothing",
			in:     nil,
			want:   []zitadel.MetadataEntry{},
		},
		"Sorted": {
			reason: "Entries come back sorted by key",
			in: v1alpha1.MetadataList{
				{Key: "zebra", Value: "z"},
				{Key: keyA, Value: "a"},
			},
			want: []zitadel.MetadataEntry{
				{Key: keyA, Value: "a"},
				{Key: "zebra", Value: "z"},
			},
		},
		"UnkeyedDropped": {
			reason: "An entry with no key is dropped rather than sent",
			in: v1alpha1.MetadataList{
				{Key: "", Value: "orphan"},
				{Key: keyA, Value: "a"},
			},
			want: []zitadel.MetadataEntry{{Key: keyA, Value: "a"}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.OrganizationMetadata{
				Spec: v1alpha1.OrganizationMetadataSpec{
					ForProvider: v1alpha1.OrganizationMetadataParameters{Metadata: tc.in},
				},
			}

			if diff := cmp.Diff(tc.want, desired(cr)); diff != "" {
				t.Errorf("\n%s\ndesired(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestMetadataDetails(t *testing.T) {
	entries := []zitadel.MetadataEntry{{Key: "b", Value: "2"}, {Key: "a", Value: "1"}}

	got := metadataDetails(entries)
	if got == nil {
		t.Fatal("metadataDetails(...): want details, got nil")
	}

	raw, ok := got[v1alpha1.ConnectionKeyMetadata]
	if !ok {
		t.Fatalf("metadataDetails(...): want the %q key, got %v", v1alpha1.ConnectionKeyMetadata, got)
	}

	if want := `{"a":"1","b":"2"}`; string(raw) != want {
		t.Errorf("metadataDetails(...): want %s, got %s", want, string(raw))
	}

	var decoded map[string]string
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Errorf("metadataDetails(...): the published value is not valid JSON: %v", err)
	}
}

func TestUpdateStatus(t *testing.T) {
	entries := []zitadel.MetadataEntry{
		{Key: keyA, Value: "a"},
		{Key: "beta", Value: "b"},
	}

	cr := &v1alpha1.OrganizationMetadata{}
	updateStatus(cr, "org-1", entries)

	if cr.Status.AtProvider.OrganizationID == nil || *cr.Status.AtProvider.OrganizationID != "org-1" {
		t.Errorf("OrganizationID: want %q, got %v", "org-1", cr.Status.AtProvider.OrganizationID)
	}

	want := v1alpha1.MetadataList{
		{Key: keyA, Value: "a"},
		{Key: "beta", Value: "b"},
	}

	if diff := cmp.Diff(want, cr.Status.AtProvider.Metadata); diff != "" {
		t.Errorf("Metadata: -want, +got:\n%s", diff)
	}
}
