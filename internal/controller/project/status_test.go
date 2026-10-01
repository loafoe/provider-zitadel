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

package project

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

const (
	testProjectName = "billing"
)

func TestIsUpToDate(t *testing.T) {
	cases := map[string]struct {
		reason string
		fp     v1alpha1.ProjectParameters
		remote zitadel.Project
		want   bool
	}{
		"Unchanged": {
			reason: "A project that matches the desired state is up to date",
			fp:     v1alpha1.ProjectParameters{Name: testProjectName},
			remote: zitadel.Project{Name: testProjectName},
			want:   true,
		},
		"Renamed": {
			reason: "A project with a different name is not up to date",
			fp:     v1alpha1.ProjectParameters{Name: testProjectName},
			remote: zitadel.Project{Name: "billing-v2"},
			want:   false,
		},
		"RoleAssertionDrift": {
			reason: "A differing projectRoleAssertion is drift",
			fp:     v1alpha1.ProjectParameters{Name: testProjectName, ProjectRoleAssertion: common.BoolPtr(true)},
			remote: zitadel.Project{Name: testProjectName, ProjectRoleAssertion: false},
			want:   false,
		},
		"RoleAssertionMatch": {
			reason: "A matching projectRoleAssertion is not drift",
			fp:     v1alpha1.ProjectParameters{Name: testProjectName, ProjectRoleAssertion: common.BoolPtr(true)},
			remote: zitadel.Project{Name: testProjectName, ProjectRoleAssertion: true},
			want:   true,
		},
		"UnsetIsNotDrift": {
			reason: "An unset projectRoleAssertion is not compared",
			fp:     v1alpha1.ProjectParameters{Name: testProjectName},
			remote: zitadel.Project{Name: testProjectName, ProjectRoleAssertion: true},
			want:   true,
		},
		"AuthorizationDrift": {
			reason: "A differing authorizationRequired is drift",
			fp:     v1alpha1.ProjectParameters{Name: testProjectName, AuthorizationRequired: common.BoolPtr(true)},
			remote: zitadel.Project{Name: testProjectName, AuthorizationRequired: false},
			want:   false,
		},
		"AccessDrift": {
			reason: "A differing projectAccessRequired is drift",
			fp:     v1alpha1.ProjectParameters{Name: testProjectName, ProjectAccessRequired: common.BoolPtr(true)},
			remote: zitadel.Project{Name: testProjectName, ProjectAccessRequired: false},
			want:   false,
		},
		"LabelingDrift": {
			reason: "A differing privateLabelingSetting is drift",
			fp: v1alpha1.ProjectParameters{
				Name:                   testProjectName,
				PrivateLabelingSetting: ptr(v1alpha1.PrivateLabelingSettingEnforceProjectResourceOwnerPolicy),
			},
			remote: zitadel.Project{Name: testProjectName, PrivateLabelingSetting: "AllowLoginUserResourceOwnerPolicy"},
			want:   false,
		},
		"StateDrift": {
			reason: "A differing state is drift",
			fp:     v1alpha1.ProjectParameters{Name: testProjectName, State: ptr(v1alpha1.ProjectStateInactive)},
			remote: zitadel.Project{Name: testProjectName, State: "Active"},
			want:   false,
		},
		"StateMatch": {
			reason: "A matching state is not drift",
			fp:     v1alpha1.ProjectParameters{Name: testProjectName, State: ptr(v1alpha1.ProjectStateInactive)},
			remote: zitadel.Project{Name: testProjectName, State: "Inactive"},
			want:   true,
		},
	}

	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			cr := &v1alpha1.Project{
				Spec: v1alpha1.ProjectSpec{ForProvider: tc.fp},
			}
			if got := isUpToDate(cr, &tc.remote); got != tc.want {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.Project{}

	updateStatus(cr, &zitadel.Project{
		ProjectID:              "p1",
		OrganizationID:         "o1",
		Name:                   testProjectName,
		State:                  "Active",
		ProjectRoleAssertion:   true,
		AuthorizationRequired:  true,
		ProjectAccessRequired:  true,
		PrivateLabelingSetting: "EnforceProjectResourceOwnerPolicy",
		GrantedOrganizationID:  "o2",
		GrantedState:           "Active",
		CreationDate:           "2024-01-02T03:04:05Z",
		ChangeDate:             "2024-02-03T04:05:06Z",
	})

	want := v1alpha1.ProjectObservation{
		ID:                     common.StringPtr("p1"),
		OrganizationID:         common.StringPtr("o1"),
		Name:                   common.StringPtr(testProjectName),
		State:                  ptr(v1alpha1.ProjectStateActive),
		ProjectRoleAssertion:   common.BoolPtr(true),
		AuthorizationRequired:  common.BoolPtr(true),
		ProjectAccessRequired:  common.BoolPtr(true),
		PrivateLabelingSetting: ptr(v1alpha1.PrivateLabelingSettingEnforceProjectResourceOwnerPolicy),
		GrantedOrganizationID:  common.StringPtr("o2"),
		GrantedState:           common.StringPtr("Active"),
		CreationDate:           common.ParseTime("2024-01-02T03:04:05Z"),
		ChangeDate:             common.ParseTime("2024-02-03T04:05:06Z"),
	}

	if diff := cmp.Diff(want, cr.Status.AtProvider); diff != "" {
		t.Errorf("updateStatus(...): -want, +got:\n%s", diff)
	}
}

func ptr[T any](v T) *T { return &v }
