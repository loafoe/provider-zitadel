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
	"context"
	"net"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	projectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/project/v2"
	"google.golang.org/grpc"

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

	updateStatus(cr, "p1", &zitadel.ProjectRole{
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
	// Delete addresses the role by project+key and checks this field to decide
	// whether the role was ever created, so it has to be populated on every
	// observation.
	if got := common.Deref(cr.Status.AtProvider.ProjectID); got != "p1" {
		t.Errorf("projectID: want %q, got %q", "p1", got)
	}
}

// fakeProjectService records the project roles it is asked to remove, so a
// test can tell whether Delete actually reached the API.
type fakeProjectService struct {
	projectv2.UnimplementedProjectServiceServer

	removed []string
}

func (f *fakeProjectService) RemoveProjectRole(_ context.Context, req *projectv2.RemoveProjectRoleRequest) (*projectv2.RemoveProjectRoleResponse, error) {
	f.removed = append(f.removed, req.GetProjectId()+"/"+req.GetRoleKey())

	return &projectv2.RemoveProjectRoleResponse{}, nil
}

// fakeExternal returns an external whose client talks to an in-process Zitadel
// that records RemoveProjectRole calls.
func fakeExternal(t *testing.T) (*external, *fakeProjectService) {
	t.Helper()

	var lc net.ListenConfig

	lis, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("cannot listen: %v", err)
	}

	fake := &fakeProjectService{}
	srv := grpc.NewServer()
	projectv2.RegisterProjectServiceServer(srv, fake)

	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	zc, err := zitadel.NewClient(context.Background(), zitadel.Config{
		URL:         "http://" + lis.Addr().String(),
		Insecure:    true,
		Credentials: zitadel.Credentials{Token: "a-personal-access-token"},
	})
	if err != nil {
		t.Fatalf("cannot build a client for the fake: %v", err)
	}

	return &external{client: zc}, fake
}

// TestDeleteRemovesRole is the regression test for the bug where every
// ProjectRole was treated as "never created": Delete returned success without
// calling Zitadel, so deleting the MR left the role behind.
func TestDeleteRemovesRole(t *testing.T) {
	cases := map[string]struct {
		reason       string
		statusID     *string
		externalName string
		wantRemoved  []string
	}{
		"FromStatus": {
			reason:       "The project id in status addresses the role",
			statusID:     ptr("p1"),
			externalName: "p1/reader",
			wantRemoved:  []string{"p1/reader"},
		},
		"FromExternalName": {
			reason: "An object written before ProjectID was recorded still has " +
				"the project in its external name",
			externalName: "p1/reader",
			wantRemoved:  []string{"p1/reader"},
		},
		"NeverCreated": {
			reason:      "Nothing was created, so there is nothing to remove",
			wantRemoved: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e, fake := fakeExternal(t)

			cr := &v1alpha1.ProjectRole{Spec: v1alpha1.ProjectRoleSpec{
				ForProvider: v1alpha1.ProjectRoleParameters{Key: testRoleKey},
			}}
			cr.Status.AtProvider.ProjectID = tc.statusID
			if tc.externalName != "" {
				meta.SetExternalName(cr, tc.externalName)
			}

			if _, err := e.Delete(context.Background(), cr); err != nil {
				t.Fatalf("\n%s\nDelete(...): unexpected error: %v", tc.reason, err)
			}

			if len(fake.removed) != len(tc.wantRemoved) {
				t.Fatalf("\n%s\nremoved: want %v, got %v", tc.reason, tc.wantRemoved, fake.removed)
			}
			for i := range tc.wantRemoved {
				if fake.removed[i] != tc.wantRemoved[i] {
					t.Errorf("\n%s\nremoved[%d]: want %q, got %q", tc.reason, i, tc.wantRemoved[i], fake.removed[i])
				}
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
