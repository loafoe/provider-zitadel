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

package projectrole

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

const (
	testRoleKey  = "reader"
	testRoleName = "Reader"
)

func TestExternalNameFor(t *testing.T) {
	if got, want := externalNameFor("p1", testRoleKey), "p1/reader"; got != want {
		t.Errorf("externalNameFor(...): want %q, got %q", want, got)
	}
}

func TestIsUpToDate(t *testing.T) {
	remote := zitadel.ProjectRole{
		Key:         testRoleKey,
		DisplayName: testRoleName,
		Group:       "Billing",
	}

	cases := map[string]struct {
		reason string
		fp     v1alpha1.ProjectRoleParameters
		want   bool
	}{
		"Unchanged": {
			reason: "A role matching the desired state is up to date",
			fp:     v1alpha1.ProjectRoleParameters{Key: testRoleKey, DisplayName: testRoleName, Group: ptr("Billing")},
			want:   true,
		},
		"DisplayNameDrift": {
			reason: "A differing display name is drift",
			fp:     v1alpha1.ProjectRoleParameters{Key: testRoleKey, DisplayName: "Viewer"},
			want:   false,
		},
		"GroupDrift": {
			reason: "A differing group is drift",
			fp:     v1alpha1.ProjectRoleParameters{Key: testRoleKey, DisplayName: testRoleName, Group: ptr("Other")},
			want:   false,
		},
		"UnsetGroup": {
			reason: "An unset group is not compared",
			fp:     v1alpha1.ProjectRoleParameters{Key: testRoleKey, DisplayName: testRoleName},
			want:   true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.ProjectRole{Spec: v1alpha1.ProjectRoleSpec{ForProvider: tc.fp}}
			if got := isUpToDate(cr, &remote); got != tc.want {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.ProjectRole{}

	updateStatus(cr, &zitadel.ProjectRole{
		Key:          testRoleKey,
		DisplayName:  testRoleName,
		Group:        "Billing",
		CreationDate: "2024-01-02T03:04:05Z",
	})

	if got := common.Deref(cr.Status.AtProvider.Key); got != testRoleKey {
		t.Errorf("key: want %q, got %q", testRoleKey, got)
	}
	if got := common.Deref(cr.Status.AtProvider.Group); got != "Billing" {
		t.Errorf("group: want %q, got %q", "Billing", got)
	}
	if cr.Status.AtProvider.CreationDate == nil {
		t.Error("creationDate: want a timestamp, got nil")
	}
}

func ptr[T any](v T) *T { return &v }
