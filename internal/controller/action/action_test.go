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

package action

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// Values the cases share, named so that a case reads as a statement about
// behaviour rather than about a repeated literal.
const (
	defaultTimeout = "10s"
)

// Test values, named so that a case reads as a statement about behaviour
// rather than about a repeated string.
const (
	actionName = "notify"
)

// An action holds a script and a timeout, and both are optional in the spec.
// What is tested here is what the controller decides to send and what it treats
// as drift, because both are places where a mistake shows up as a resource that
// is rewritten on every poll or never converges.

func ptr[T any](v T) *T { return &v }

func TestValueOrTimeout(t *testing.T) {
	cases := map[string]struct {
		reason   string
		in       *string
		fallback string
		want     string
	}{
		"Unset": {
			reason:   "An unset timeout falls back",
			in:       nil,
			fallback: defaultTimeout,
			want:     defaultTimeout,
		},
		"Empty": {
			reason:   "An empty timeout falls back rather than sending an unusable value",
			in:       ptr(""),
			fallback: defaultTimeout,
			want:     defaultTimeout,
		},
		"Set": {
			reason:   "A timeout the operator gave is used",
			in:       ptr("30s"),
			fallback: defaultTimeout,
			want:     "30s",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := valueOrTimeout(tc.in, tc.fallback); got != tc.want {
				t.Errorf("\n%s\nvalueOrTimeout(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

func TestDesired(t *testing.T) {
	cases := map[string]struct {
		reason string
		fp     v1alpha1.ActionParameters
		want   zitadel.ActionInput
	}{
		"Defaults": {
			reason: "An empty spec gets the documented defaults rather than empty values",
			fp:     v1alpha1.ActionParameters{},
			want:   zitadel.ActionInput{Timeout: defaultTimeout},
		},
		"EverythingSet": {
			reason: "Every field the operator gave is carried through",
			fp: v1alpha1.ActionParameters{
				Name:          ptr(actionName),
				Script:        ptr("ctx.user.id"),
				Timeout:       ptr("30s"),
				AllowedToFail: ptr(true),
			},
			want: zitadel.ActionInput{
				Name:          actionName,
				Script:        "ctx.user.id",
				Timeout:       "30s",
				AllowedToFail: true,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.Action{Spec: v1alpha1.ActionSpec{ForProvider: tc.fp}}

			if diff := cmp.Diff(tc.want, desired(cr)); diff != "" {
				t.Errorf("\n%s\ndesired(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestIsUpToDate(t *testing.T) {
	observed := &zitadel.Action{
		ID:            "a1",
		Name:          actionName,
		Timeout:       "30s",
		AllowedToFail: true,
		State:         "Active",
	}

	cases := map[string]struct {
		reason string
		fp     v1alpha1.ActionParameters
		want   bool
	}{
		"Unspecified": {
			reason: "A spec that mentions nothing is up to date, so Zitadel's own state is not fought over",
			fp:     v1alpha1.ActionParameters{},
			want:   true,
		},
		"Matching": {
			reason: "Every field the spec set matches",
			fp: v1alpha1.ActionParameters{
				Name:          ptr(actionName),
				Timeout:       ptr("30s"),
				AllowedToFail: ptr(true),
			},
			want: true,
		},
		"NameDrift": {
			reason: "A differing name is drift",
			fp:     v1alpha1.ActionParameters{Name: ptr("other")},
			want:   false,
		},
		"TimeoutDrift": {
			reason: "A differing timeout is drift",
			fp:     v1alpha1.ActionParameters{Timeout: ptr("5s")},
			want:   false,
		},
		"AllowedToFailDrift": {
			reason: "A differing failure behaviour is drift",
			fp:     v1alpha1.ActionParameters{AllowedToFail: ptr(false)},
			want:   false,
		},
		"StateMatch": {
			reason: "A state the spec set and Zitadel agrees on is up to date",
			fp:     v1alpha1.ActionParameters{State: ptr(v1alpha1.ActionState("Active"))},
			want:   true,
		},
		"StateDrift": {
			reason: "A state the spec set and Zitadel disagrees on is drift",
			fp:     v1alpha1.ActionParameters{State: ptr(v1alpha1.ActionState("Inactive"))},
			want:   false,
		},
		"StateUnmentioned": {
			// This is the case that stops the controller reactivating an action
			// somebody deactivated in the console on every poll.
			reason: "A state the spec does not mention is left as Zitadel has it",
			fp:     v1alpha1.ActionParameters{Name: ptr(actionName)},
			want:   true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.Action{Spec: v1alpha1.ActionSpec{ForProvider: tc.fp}}

			if got := isUpToDate(cr, observed); got != tc.want {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.Action{}

	updateStatus(cr, "org-1", &zitadel.Action{
		ID:            "a1",
		Name:          actionName,
		Timeout:       "30s",
		AllowedToFail: true,
		State:         "Active",
	})

	ap := cr.Status.AtProvider

	if ap.ID != "a1" {
		t.Errorf("ID: want %q, got %q", "a1", ap.ID)
	}

	for name, tc := range map[string]struct{ got, want *string }{
		"OrganizationID": {ap.OrganizationID, common.StringPtr("org-1")},
		"Name":           {ap.Name, common.StringPtr(actionName)},
		"Timeout":        {ap.Timeout, common.StringPtr("30s")},
		"State":          {ap.State, common.StringPtr("Active")},
	} {
		if diff := cmp.Diff(tc.want, tc.got); diff != "" {
			t.Errorf("%s: -want, +got:\n%s", name, diff)
		}
	}

	if ap.AllowedToFail == nil || !*ap.AllowedToFail {
		t.Errorf("AllowedToFail: want true, got %v", ap.AllowedToFail)
	}
}
