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

package usermetadata

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

// User metadata replaces a whole set rather than adding one key, because two
// resources managing one key each would each replace the other's. That makes
// normalisation the interesting part: without it, a set written in a different
// order than the one in the spec looks like drift on every poll.

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
			reason: "Entries come back sorted by key, whatever order they were written in",
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
		"EmptyValueKept": {
			reason: "An entry with an empty value is kept, because that is how a key is cleared",
			in:     v1alpha1.MetadataList{{Key: keyA, Value: ""}},
			want:   []zitadel.MetadataEntry{{Key: keyA, Value: ""}},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.UserMetadata{Spec: v1alpha1.UserMetadataSpec{ForProvider: v1alpha1.UserMetadataParameters{Metadata: tc.in}}}

			if diff := cmp.Diff(tc.want, desired(cr)); diff != "" {
				t.Errorf("\n%s\ndesired(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestMetadataDetails(t *testing.T) {
	cases := map[string]struct {
		reason  string
		entries []zitadel.MetadataEntry
		want    string
		wantNil bool
	}{
		"Empty": {
			reason:  "No metadata publishes an empty object rather than nothing",
			entries: nil,
			want:    `{}`,
		},
		"Entries": {
			reason:  "Every entry is published as JSON",
			entries: []zitadel.MetadataEntry{{Key: "b", Value: "2"}, {Key: "a", Value: "1"}},
			want:    `{"a":"1","b":"2"}`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := metadataDetails(tc.entries)
			if got == nil {
				t.Fatalf("\n%s\nmetadataDetails(...): want details, got nil", tc.reason)
			}

			raw, ok := got[v1alpha1.ConnectionKeyMetadata]
			if !ok {
				t.Fatalf("\n%s\nmetadataDetails(...): want the %q key, got %v", tc.reason, v1alpha1.ConnectionKeyMetadata, got)
			}

			if string(raw) != tc.want {
				t.Errorf("\n%s\nmetadataDetails(...): want %s, got %s", tc.reason, tc.want, string(raw))
			}

			// It has to be readable, since the whole point is that a shell can
			// consume it.
			var decoded map[string]string
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Errorf("\n%s\nmetadataDetails(...): the published value is not valid JSON: %v", tc.reason, err)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	entries := []zitadel.MetadataEntry{
		{Key: keyA, Value: "a"},
		{Key: "beta", Value: "b"},
	}

	cr := &v1alpha1.UserMetadata{}
	updateStatus(cr, "u1", entries)

	if cr.Status.AtProvider.UserID == nil || *cr.Status.AtProvider.UserID != "u1" {
		t.Errorf("UserID: want %q, got %v", "u1", cr.Status.AtProvider.UserID)
	}

	want := v1alpha1.MetadataList{
		{Key: keyA, Value: "a"},
		{Key: "beta", Value: "b"},
	}

	if diff := cmp.Diff(want, cr.Status.AtProvider.Metadata); diff != "" {
		t.Errorf("Metadata: -want, +got:\n%s", diff)
	}
}

func TestUpdateStatusWithNoEntries(t *testing.T) {
	cr := &v1alpha1.UserMetadata{}
	updateStatus(cr, "u1", nil)

	// An empty list rather than a nil one, so that "no metadata" reads as an
	// empty set rather than as an unset field.
	if cr.Status.AtProvider.Metadata == nil {
		t.Error("Metadata: want an empty list, got nil")
	}

	if len(cr.Status.AtProvider.Metadata) != 0 {
		t.Errorf("Metadata: want an empty list, got %v", cr.Status.AtProvider.Metadata)
	}
}
