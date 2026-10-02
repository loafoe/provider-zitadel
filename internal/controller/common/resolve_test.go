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

package common

import (
	"context"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/loafoe/provider-zitadel/apis"
	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
)

// testNamespace is the namespace the fixtures below live in.
const testNamespace = "team"

func TestResolveOrganizationID(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	org := &v1alpha1.Organization{
		ObjectMeta: metav1.ObjectMeta{Name: "customer", Namespace: testNamespace},
		Status: v1alpha1.OrganizationStatus{
			AtProvider: v1alpha1.OrganizationObservation{ID: strPtr("org-1")},
		},
	}

	grantee := &v1alpha1.ProjectGrantMember{
		ObjectMeta: metav1.ObjectMeta{Name: "member", Namespace: testNamespace},
		Spec: v1alpha1.ProjectGrantMemberSpec{
			ForProvider: v1alpha1.ProjectGrantMemberParameters{
				GrantedOrganizationRef: &xpv1.Reference{Name: "customer"},
			},
		},
	}

	kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(org, grantee).Build()

	cases := map[string]struct {
		reason  string
		ref     *xpv1.Reference
		direct  *string
		want    string
		wantErr bool
	}{
		"Reference": {
			reason: "A reference to a managed Organization resolves to its Zitadel ID",
			ref:    &xpv1.Reference{Name: "customer"},
			want:   "org-1",
		},
		"Direct": {
			reason: "A direct ID is used as is",
			direct: strPtr("org-2"),
			want:   "org-2",
		},
		"Neither": {
			// Nothing set yields nothing and no error: the caller turns that
			// into the actionable "must be set" message, so the resolver stays
			// free of policy about which fields are required.
			reason: "Nothing set yields an empty ID and no error",
			want:   "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := ResolveOrganizationID(context.Background(), kube, grantee, tc.ref, nil, tc.direct, "", "")
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\nResolveOrganizationID(...): wantErr %v, got %v", tc.reason, tc.wantErr, err)
			}
			if err == nil && got != tc.want {
				t.Errorf("\n%s\nResolveOrganizationID(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

func TestResolveUserIDAcrossKinds(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	sa := &v1alpha1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "worker", Namespace: testNamespace},
		Status: v1alpha1.ServiceAccountStatus{
			AtProvider: v1alpha1.ServiceAccountObservation{ID: strPtr("sa-1")},
		},
	}
	human := &v1alpha1.HumanUser{
		ObjectMeta: metav1.ObjectMeta{Name: "alice", Namespace: testNamespace},
		Status: v1alpha1.HumanUserStatus{
			AtProvider: v1alpha1.HumanUserObservation{ID: strPtr("user-1")},
		},
	}

	cases := map[string]struct {
		reason   string
		objects  []client.Object
		ref      string
		want     string
		wantFail bool
	}{
		"ServiceAccount": {
			reason:  "A reference to a ServiceAccount resolves, because a Personal Access Token usually belongs to one",
			objects: []client.Object{sa},
			ref:     "worker",
			want:    "sa-1",
		},
		"HumanUser": {
			reason:  "A reference to a HumanUser resolves too",
			objects: []client.Object{human},
			ref:     "alice",
			want:    "user-1",
		},
		"Neither": {
			reason:   "A reference matching neither kind is reported",
			objects:  []client.Object{sa, human},
			ref:      "nobody",
			wantFail: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			kube := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.objects...).Build()
			grantee := &v1alpha1.ProjectGrantMember{
				ObjectMeta: metav1.ObjectMeta{Name: "member", Namespace: testNamespace},
			}

			got, err := ResolveUserID(context.Background(), kube, grantee, &xpv1.Reference{Name: tc.ref}, nil, nil, "")
			if tc.wantFail {
				if err == nil {
					t.Errorf("\n%s\nResolveUserID(...): want an error, got %q", tc.reason, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("\n%s\nResolveUserID(...): %v", tc.reason, err)
			}
			if got != tc.want {
				t.Errorf("\n%s\nResolveUserID(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

func TestIsReferenceGone(t *testing.T) {
	if IsReferenceGone(nil) {
		t.Error("IsReferenceGone(nil): want false, got true")
	}
}

func strPtr(s string) *string { return &s }
