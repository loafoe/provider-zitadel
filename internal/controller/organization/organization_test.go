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

package organization

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

const (
	testOrgName = "Platform"
)

func TestIsUpToDate(t *testing.T) {
	cases := map[string]struct {
		reason string
		fp     v1alpha1.OrganizationParameters
		remote zitadel.Organization
		want   bool
	}{
		"Unchanged": {
			reason: "An organization matching the desired state is up to date",
			fp:     v1alpha1.OrganizationParameters{Name: testOrgName},
			remote: zitadel.Organization{Name: testOrgName, State: "Active"},
			want:   true,
		},
		"Renamed": {
			reason: "A differing name is drift",
			fp:     v1alpha1.OrganizationParameters{Name: testOrgName},
			remote: zitadel.Organization{Name: "Platform v2", State: "Active"},
			want:   false,
		},
		"StateDrift": {
			reason: "A differing state is drift",
			fp:     v1alpha1.OrganizationParameters{Name: testOrgName, State: ptr(v1alpha1.OrganizationStateInactive)},
			remote: zitadel.Organization{Name: testOrgName, State: "Active"},
			want:   false,
		},
		"StateMatch": {
			reason: "A matching state is not drift",
			fp:     v1alpha1.OrganizationParameters{Name: testOrgName, State: ptr(v1alpha1.OrganizationStateInactive)},
			remote: zitadel.Organization{Name: testOrgName, State: "Inactive"},
			want:   true,
		},
		"UnsetState": {
			reason: "An unset state is not compared",
			fp:     v1alpha1.OrganizationParameters{Name: testOrgName},
			remote: zitadel.Organization{Name: testOrgName, State: "Inactive"},
			want:   true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.Organization{Spec: v1alpha1.OrganizationSpec{ForProvider: tc.fp}}
			if got := isUpToDate(cr, &tc.remote); got != tc.want {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.Organization{}

	updateStatus(cr, &zitadel.Organization{
		ID:            "1",
		Name:          testOrgName,
		PrimaryDomain: "platform.example.com",
		State:         "Active",
		CreationDate:  "2024-01-02T03:04:05Z",
		ChangeDate:    "2024-02-03T04:05:06Z",
	})

	if got := common.Deref(cr.Status.AtProvider.ID); got != "1" {
		t.Errorf("id: want %q, got %q", "1", got)
	}
	if got := common.Deref(cr.Status.AtProvider.PrimaryDomain); got != "platform.example.com" {
		t.Errorf("primaryDomain: want %q, got %q", "platform.example.com", got)
	}
	if cr.Status.AtProvider.State == nil || *cr.Status.AtProvider.State != v1alpha1.OrganizationStateActive {
		t.Errorf("state: want %q, got %v", v1alpha1.OrganizationStateActive, cr.Status.AtProvider.State)
	}
	if cr.Status.AtProvider.ChangeDate == nil {
		t.Error("changeDate: want a timestamp, got nil")
	}
}

func ptr[T any](v T) *T { return &v }
