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
	stderrors "errors"
	"math"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/loafoe/provider-zitadel/apis"
	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
)

// Resolving a reference is what makes a resource point at the right thing in
// Zitadel, and a mistake here is silent: the resource simply reconciles against
// the wrong organization, or gets stuck waiting for an identifier that never
// arrives. So each of these covers what it resolves, what it refuses, and the
// difference between a reference and a direct identifier.

// resolveKube builds a fake client holding the supplied objects.
func resolveKube(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := apis.AddToScheme(scheme); err != nil {
		t.Fatalf("cannot register the scheme: %v", err)
	}

	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("cannot register core types: %v", err)
	}

	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

// aSubject is any managed resource, standing in for whatever is being
// reconciled. The resolvers only need it to be a ModernManaged.
func aSubject(namespace string) *v1alpha1.ProjectRole {
	return &v1alpha1.ProjectRole{ObjectMeta: metav1.ObjectMeta{Name: "subject", Namespace: namespace}}
}

// --- The extractors -----------------------------------------------------
//
// These run against whatever the reference resolver finds, so they have to cope
// with a kind that publishes no identifier yet as well as one it never will.

func TestExtractOrganizationID(t *testing.T) {
	extract := ExtractOrganizationID()

	org := &v1alpha1.Organization{Status: v1alpha1.OrganizationStatus{
		AtProvider: v1alpha1.OrganizationObservation{ID: strPtr("org-1")},
	}}
	notYet := &v1alpha1.Organization{}
	other := &v1alpha1.Project{}

	for name, tc := range map[string]struct {
		reason string
		mg     resource.Managed
		want   string
	}{
		"Set":        {reason: "An organization with an ID yields it", mg: org, want: "org-1"},
		"NotCreated": {reason: "One that has not been created yet yields nothing", mg: notYet, want: ""},
		"WrongKind":  {reason: "A kind that is not an organization yields nothing", mg: other, want: ""},
	} {
		t.Run(name, func(t *testing.T) {
			if got := extract(tc.mg); got != tc.want {
				t.Errorf("\n%s\nExtractOrganizationID(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

func TestExtractProjectID(t *testing.T) {
	extract := ExtractProjectID()

	p := &v1alpha1.Project{Status: v1alpha1.ProjectStatus{
		AtProvider: v1alpha1.ProjectObservation{ID: strPtr("p-1")},
	}}

	if got := extract(p); got != "p-1" {
		t.Errorf("ExtractProjectID(...): want %q, got %q", "p-1", got)
	}

	if got := extract(&v1alpha1.Project{}); got != "" {
		t.Errorf("a project with no ID must yield nothing, got %q", got)
	}

	if got := extract(&v1alpha1.Organization{}); got != "" {
		t.Errorf("a kind that is not a project must yield nothing, got %q", got)
	}
}

// TestExtractUserID covers both kinds a user reference may point at. They are
// different Go types publishing the same field, and a reference is tried
// against each in turn, so both have to answer.
func TestExtractUserID(t *testing.T) {
	extract := ExtractUserID()

	cases := map[string]struct {
		reason string
		mg     resource.Managed
		want   string
	}{
		"HumanUser": {
			reason: "A human user with an ID yields it",
			mg: &v1alpha1.HumanUser{Status: v1alpha1.HumanUserStatus{
				AtProvider: v1alpha1.HumanUserObservation{ID: strPtr("u-1")},
			}},
			want: "u-1",
		},
		"ServiceAccount": {
			reason: "A service account with an ID yields it",
			mg: &v1alpha1.ServiceAccount{Status: v1alpha1.ServiceAccountStatus{
				AtProvider: v1alpha1.ServiceAccountObservation{ID: strPtr("u-2")},
			}},
			want: "u-2",
		},
		"HumanUserNotCreated": {
			reason: "A human user with no ID yields nothing",
			mg:     &v1alpha1.HumanUser{},
		},
		"ServiceAccountNotCreated": {
			reason: "A service account with no ID yields nothing",
			mg:     &v1alpha1.ServiceAccount{},
		},
		"OtherKind": {
			reason: "A kind that is not a user yields nothing",
			mg:     &v1alpha1.Organization{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := extract(tc.mg); got != tc.want {
				t.Errorf("\n%s\nExtractUserID(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

// --- The resolvers ------------------------------------------------------

func TestResolveProjectID(t *testing.T) {
	p := &v1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "platform", Namespace: testNamespace},
		Status: v1alpha1.ProjectStatus{
			AtProvider: v1alpha1.ProjectObservation{ID: strPtr("p-1")},
		},
	}

	subject := aSubject(testNamespace)
	kube := resolveKube(t, p)

	t.Run("Reference", func(t *testing.T) {
		got, err := ResolveProjectID(context.Background(), kube, subject,
			&xpv1.Reference{Name: "platform"}, nil, nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "p-1" {
			t.Errorf("want %q, got %q", "p-1", got)
		}
	})

	t.Run("Direct", func(t *testing.T) {
		got, err := ResolveProjectID(context.Background(), kube, subject, nil, nil, strPtr("p-2"), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "p-2" {
			t.Errorf("want %q, got %q", "p-2", got)
		}
	})

	t.Run("NothingSet", func(t *testing.T) {
		// A resource with no project at all is a manifest error, and saying so
		// is more useful than resolving to an empty string and failing later.
		_, err := ResolveProjectID(context.Background(), kube, subject, nil, nil, nil, "")
		if err == nil {
			t.Fatal("want an error when no project is named at all")
		}

		if !errorsIs(err, ErrNoProjectID) {
			t.Errorf("want the no-project error, got %v", err)
		}
	})

	t.Run("MissingReference", func(t *testing.T) {
		if _, err := ResolveProjectID(context.Background(), kube, subject,
			&xpv1.Reference{Name: "absent"}, nil, nil, ""); err == nil {
			t.Error("want an error for a reference to a project that does not exist")
		}
	})
}

// TestResolveUserIDAcrossKindsIsNotDuplicated lives in resolve_test.go already;
// what is here is the failure paths, which is where the "tried every kind in
// turn" logic is decided.
func TestResolveUserIDFailures(t *testing.T) {
	subject := aSubject(testNamespace)
	kube := resolveKube(t)

	t.Run("NothingSet", func(t *testing.T) {
		_, err := ResolveUserID(context.Background(), kube, subject, nil, nil, nil, "")
		if err == nil {
			t.Fatal("want an error when no user is named at all")
		}

		if !errorsIs(err, ErrNoUserID) {
			t.Errorf("want the no-user error, got %v", err)
		}
	})

	t.Run("Direct", func(t *testing.T) {
		got, err := ResolveUserID(context.Background(), kube, subject, nil, nil, strPtr("u-9"), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "u-9" {
			t.Errorf("want %q, got %q", "u-9", got)
		}
	})

	t.Run("MatchesNeitherKind", func(t *testing.T) {
		// The reference matches no kind. That has to read as one clear failure
		// rather than the last kind's error, or the operator is told about the
		// wrong kind of thing.
		_, err := ResolveUserID(context.Background(), kube, subject,
			&xpv1.Reference{Name: "absent"}, nil, nil, "")
		if err == nil {
			t.Fatal("want an error for a reference matching neither kind")
		}
	})
}

// TestResolveServiceAccountID covers the resolver that must not accept a human
// user: a machine key can only belong to a machine user, so a reference that
// resolves to a HumanUser has to be rejected rather than used.
func TestResolveServiceAccountID(t *testing.T) {
	sa := &v1alpha1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "robot", Namespace: testNamespace},
		Status: v1alpha1.ServiceAccountStatus{
			AtProvider: v1alpha1.ServiceAccountObservation{ID: strPtr("u-1")},
		},
	}
	human := &v1alpha1.HumanUser{
		ObjectMeta: metav1.ObjectMeta{Name: "person", Namespace: testNamespace},
		Status: v1alpha1.HumanUserStatus{
			AtProvider: v1alpha1.HumanUserObservation{ID: strPtr("u-2")},
		},
	}

	subject := aSubject(testNamespace)
	kube := resolveKube(t, sa, human)

	t.Run("ResolvesAServiceAccount", func(t *testing.T) {
		got, err := ResolveServiceAccountID(context.Background(), kube, subject,
			&xpv1.Reference{Name: "robot"}, nil, nil, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "u-1" {
			t.Errorf("want %q, got %q", "u-1", got)
		}
	})

	t.Run("RejectsAHumanUser", func(t *testing.T) {
		// The two kinds are separate Kubernetes objects, so a reference to the
		// human one does not resolve here even though it is a Zitadel user.
		if _, err := ResolveServiceAccountID(context.Background(), kube, subject,
			&xpv1.Reference{Name: "person"}, nil, nil, ""); err == nil {
			t.Error("a reference to a HumanUser resolved as a service account, which would attach a machine key to a human user")
		}
	})

	t.Run("Direct", func(t *testing.T) {
		got, err := ResolveServiceAccountID(context.Background(), kube, subject, nil, nil, strPtr("u-3"), "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "u-3" {
			t.Errorf("want %q, got %q", "u-3", got)
		}
	})

	t.Run("NothingSet", func(t *testing.T) {
		_, err := ResolveServiceAccountID(context.Background(), kube, subject, nil, nil, nil, "")
		if err == nil {
			t.Fatal("want an error when no service account is named at all")
		}

		if !errorsIs(err, ErrNoUserID) {
			t.Errorf("want the no-user error, got %v", err)
		}
	})
}

// TestResolveServiceAccount covers the two-in-one resolver: Zitadel lists a
// user's machine keys through an organization scoped API, so a caller needs both
// identifiers and resolving twice would be wasteful.
func TestResolveServiceAccount(t *testing.T) {
	sa := &v1alpha1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "robot", Namespace: testNamespace},
		Status: v1alpha1.ServiceAccountStatus{
			AtProvider: v1alpha1.ServiceAccountObservation{
				ID:             strPtr("u-1"),
				OrganizationID: strPtr("org-1"),
			},
		},
	}

	subject := aSubject(testNamespace)
	kube := resolveKube(t, sa)

	t.Run("Reference", func(t *testing.T) {
		id, orgID, err := ResolveServiceAccount(context.Background(), kube, subject,
			&xpv1.Reference{Name: "robot"}, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if id != "u-1" {
			t.Errorf("service account ID: want %q, got %q", "u-1", id)
		}

		if orgID != "org-1" {
			t.Errorf("organization ID: want %q, got %q", "org-1", orgID)
		}
	})

	t.Run("DirectHasNoOrganization", func(t *testing.T) {
		// A direct identifier names no organization, so there is nothing to
		// report and the caller has to find it another way.
		id, orgID, err := ResolveServiceAccount(context.Background(), kube, subject, nil, nil, strPtr("u-9"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if id != "u-9" {
			t.Errorf("service account ID: want %q, got %q", "u-9", id)
		}

		if orgID != "" {
			t.Errorf("organization ID: want empty for a direct identifier, got %q", orgID)
		}
	})

	t.Run("NothingSet", func(t *testing.T) {
		if _, _, err := ResolveServiceAccount(context.Background(), kube, subject, nil, nil, nil); err == nil {
			t.Error("want an error when no service account is named at all")
		}
	})
}

// TestResolveWebKeyID covers the instance wide signing key. A WebKey's
// identifier is the external name it was created with, so the "not created
// yet" case has to read as a clear error rather than as an empty identifier the
// caller then uses.
func TestResolveWebKeyID(t *testing.T) {
	ready := &v1alpha1.WebKey{
		ObjectMeta: metav1.ObjectMeta{Name: "signing", Namespace: testNamespace},
		Status: v1alpha1.WebKeyStatus{
			AtProvider: v1alpha1.WebKeyObservation{ID: "key-1"},
		},
	}
	notYet := &v1alpha1.WebKey{
		ObjectMeta: metav1.ObjectMeta{Name: "pending", Namespace: testNamespace},
	}

	subject := aSubject(testNamespace)
	kube := resolveKube(t, ready, notYet)

	t.Run("Reference", func(t *testing.T) {
		got, err := ResolveWebKeyID(context.Background(), kube, subject, &xpv1.Reference{Name: "signing"}, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "key-1" {
			t.Errorf("want %q, got %q", "key-1", got)
		}
	})

	t.Run("NotCreatedYet", func(t *testing.T) {
		_, err := ResolveWebKeyID(context.Background(), kube, subject, &xpv1.Reference{Name: "pending"}, nil, nil)
		if err == nil {
			t.Fatal("want an error for a key that has no identifier yet")
		}

		if !errorsIs(err, ErrNoWebKeyID) {
			t.Errorf("want the no-web-key error, got %v", err)
		}
	})

	t.Run("Direct", func(t *testing.T) {
		got, err := ResolveWebKeyID(context.Background(), kube, subject, nil, nil, strPtr("key-9"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "key-9" {
			t.Errorf("want %q, got %q", "key-9", got)
		}
	})

	t.Run("NothingSet", func(t *testing.T) {
		if _, err := ResolveWebKeyID(context.Background(), kube, subject, nil, nil, nil); err == nil {
			t.Error("want an error when no key is named at all")
		}
	})
}

// TestResolveApplicationID covers the resolver that has to try two kinds, because
// both are applications in Zitadel but are different objects here.
func TestResolveApplicationID(t *testing.T) {
	oidc := &v1alpha1.OIDCApplication{
		ObjectMeta: metav1.ObjectMeta{Name: "web", Namespace: testNamespace},
		Status: v1alpha1.OIDCApplicationStatus{
			AtProvider: v1alpha1.OIDCApplicationObservation{ID: strPtr("app-1")},
		},
	}
	api := &v1alpha1.ApplicationAPI{
		ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: testNamespace},
		Status: v1alpha1.ApplicationAPIStatus{
			AtProvider: v1alpha1.ApplicationAPIObservation{ApplicationID: strPtr("app-2")},
		},
	}

	subject := aSubject(testNamespace)
	kube := resolveKube(t, oidc, api)

	t.Run("ResolvesAnOIDCApplication", func(t *testing.T) {
		got, err := ResolveApplicationID(context.Background(), kube, subject, &xpv1.Reference{Name: "web"}, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "app-1" {
			t.Errorf("want %q, got %q", "app-1", got)
		}
	})

	t.Run("ResolvesAnApplicationAPI", func(t *testing.T) {
		got, err := ResolveApplicationID(context.Background(), kube, subject, &xpv1.Reference{Name: "api"}, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "app-2" {
			t.Errorf("want %q, got %q", "app-2", got)
		}
	})

	t.Run("MatchesNeither", func(t *testing.T) {
		if _, err := ResolveApplicationID(context.Background(), kube, subject,
			&xpv1.Reference{Name: "absent"}, nil, nil); err == nil {
			t.Error("want an error for a reference matching neither application kind")
		}
	})

	t.Run("Direct", func(t *testing.T) {
		got, err := ResolveApplicationID(context.Background(), kube, subject, nil, nil, strPtr("app-9"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "app-9" {
			t.Errorf("want %q, got %q", "app-9", got)
		}
	})

	t.Run("NothingSet", func(t *testing.T) {
		if _, err := ResolveApplicationID(context.Background(), kube, subject, nil, nil, nil); err == nil {
			t.Error("want an error when no application is named at all")
		}
	})

	t.Run("AmbiguousSelector", func(t *testing.T) {
		// A selector matching both kinds cannot be resolved to one, and picking
		// either would configure a key onto an application the manifest did not
		// name. It has to be reported so the selector can be narrowed.
		both := &v1alpha1.OIDCApplication{
			ObjectMeta: metav1.ObjectMeta{
				Name: "shared", Namespace: testNamespace,
				Labels: map[string]string{"team": "platform"},
			},
			Status: v1alpha1.OIDCApplicationStatus{
				AtProvider: v1alpha1.OIDCApplicationObservation{ID: strPtr("app-1")},
			},
		}
		alsoAPI := &v1alpha1.ApplicationAPI{
			ObjectMeta: metav1.ObjectMeta{
				Name: "shared-api", Namespace: testNamespace,
				Labels: map[string]string{"team": "platform"},
			},
			Status: v1alpha1.ApplicationAPIStatus{
				AtProvider: v1alpha1.ApplicationAPIObservation{ApplicationID: strPtr("app-2")},
			},
		}

		k := resolveKube(t, both, alsoAPI)

		_, err := ResolveApplicationID(context.Background(), k, subject, nil,
			&xpv1.Selector{MatchLabels: map[string]string{"team": "platform"}}, nil)
		if err == nil {
			t.Fatal("want an error for a selector matching both application kinds")
		}

		if !containsSubstring(err.Error(), "narrow it") {
			t.Errorf("want the error to say how to fix it, got %v", err)
		}
	})
}

// TestResolveTargetID covers the action target resolver, which is deliberately
// narrow: it resolves against the kind rather than accepting an arbitrary ID.
func TestResolveTargetID(t *testing.T) {
	target := &v1alpha1.ActionTarget{
		ObjectMeta: metav1.ObjectMeta{Name: "hook", Namespace: testNamespace},
		Status: v1alpha1.ActionTargetStatus{
			AtProvider: v1alpha1.ActionTargetObservation{ID: "t-1"},
		},
	}

	subject := aSubject(testNamespace)
	kube := resolveKube(t, target)

	t.Run("Reference", func(t *testing.T) {
		got, err := ResolveTargetID(context.Background(), kube, subject, &xpv1.Reference{Name: "hook"}, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "t-1" {
			t.Errorf("want %q, got %q", "t-1", got)
		}
	})

	t.Run("Direct", func(t *testing.T) {
		got, err := ResolveTargetID(context.Background(), kube, subject, nil, nil, strPtr("t-9"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != "t-9" {
			t.Errorf("want %q, got %q", "t-9", got)
		}
	})

	t.Run("NothingSet", func(t *testing.T) {
		if _, err := ResolveTargetID(context.Background(), kube, subject, nil, nil, nil); err == nil {
			t.Error("want an error when no target is named at all")
		}
	})
}

// --- The small helpers --------------------------------------------------

// TestCurrentIfDeleting covers the one rule that keeps a terminating resource
// from deleting the wrong thing.
//
// Resolution normally starts from the spec, so a dependent follows a reference
// that has moved. A terminating resource is the exception: it exists to remove
// what it created, so it acts on the identity it recorded.
func TestCurrentIfDeleting(t *testing.T) {
	observed := "org-1"

	cases := map[string]struct {
		reason   string
		deleting bool
		observed *string
		want     string
	}{
		"Live": {
			reason:   "A live resource resolves from the spec, so the observed value is not used",
			deleting: false,
			observed: &observed,
			want:     "",
		},
		"Deleting": {
			reason:   "A terminating resource acts on the identity it recorded",
			deleting: true,
			observed: &observed,
			want:     "org-1",
		},
		"DeletingWithNothingRecorded": {
			reason:   "A terminating resource with nothing recorded has nothing to act on",
			deleting: true,
			observed: nil,
			want:     "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := CurrentIfDeleting(tc.deleting, tc.observed); got != tc.want {
				t.Errorf("\n%s\nCurrentIfDeleting(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

// TestToUint32 covers the narrowing of a count from a spec.
func TestToUint32(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     int64
		want   uint32
	}{
		"Zero":      {reason: "Zero is zero", in: 0, want: 0},
		"Plausible": {reason: "A plausible count is unchanged", in: 12, want: 12},
		"Max":       {reason: "The largest representable value is unchanged", in: math.MaxUint32, want: math.MaxUint32},
		"Negative":  {reason: "A negative count clamps to zero rather than wrapping", in: -1, want: 0},
		"OverMax":   {reason: "A value past the maximum saturates rather than wrapping to a small one", in: math.MaxUint32 + 1, want: math.MaxUint32},
		"VeryLarge": {reason: "The largest input saturates", in: math.MaxInt64, want: math.MaxUint32},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := ToUint32(tc.in); got != tc.want {
				t.Errorf("\n%s\nToUint32(%d): want %d, got %d", tc.reason, tc.in, tc.want, got)
			}
		})
	}
}

// TestPointerHelpers covers the deref and pointer helpers together, because they
// are one convention: a nil pointer means "not set", and every one of them turns
// that into a usable value rather than a panic.
func TestPointerHelpers(t *testing.T) {
	type named string

	if got := Deref(nil); got != "" {
		t.Errorf("Deref(nil): want empty, got %q", got)
	}

	if got := Deref(strPtr("x")); got != "x" {
		t.Errorf("Deref(...): want %q, got %q", "x", got)
	}

	if got := Value[named](nil); got != "" {
		t.Errorf("Value[named](nil): want empty, got %q", got)
	}

	if got := Value(ptrTo(named("Active"))); got != "Active" {
		t.Errorf("Value(...): want %q, got %q", "Active", got)
	}

	if got := DerefBool(nil); got {
		t.Error("DerefBool(nil): want false")
	}

	if got := DerefBool(boolPtr2(true)); !got {
		t.Error("DerefBool(true): want true")
	}

	if got := DerefInt64(nil); got != 0 {
		t.Errorf("DerefInt64(nil): want 0, got %d", got)
	}

	if got := DerefInt64(Int64Ptr(7)); got != 7 {
		t.Errorf("DerefInt64(...): want 7, got %d", got)
	}

	if got := *StringPtr("s"); got != "s" {
		t.Errorf("StringPtr(...): want %q, got %q", "s", got)
	}

	if got := *BoolPtr(false); got {
		t.Error("BoolPtr(false): want a pointer to false")
	}

	if got := *Int64Ptr(3); got != 3 {
		t.Errorf("Int64Ptr(...): want 3, got %d", got)
	}
}

// TestEnumNamesAndValues covers the projection between the API's typed string
// enums and the plain strings they are compared and written as.
func TestEnumNamesAndValues(t *testing.T) {
	type state string

	const (
		active   state = "Active"
		inactive state = "Inactive"
	)

	names := EnumNames([]state{active, inactive})
	if diff := cmp.Diff([]string{"Active", "Inactive"}, names); diff != "" {
		t.Errorf("EnumNames(...): -want, +got:\n%s", diff)
	}

	// An empty result must be empty and not nil, so that "not configured" and
	// "configured with nothing" compare the same way.
	empty := EnumNames([]state(nil))
	if empty == nil {
		t.Error("EnumNames(nil): want an empty list rather than nil, so it compares equal to an empty one")
	}

	values := EnumValues[state]([]string{"Active", "Inactive"})
	if diff := cmp.Diff([]state{active, inactive}, values); diff != "" {
		t.Errorf("EnumValues(...): -want, +got:\n%s", diff)
	}

	back := EnumValues[state](nil)
	if back == nil {
		t.Error("EnumValues(nil): want an empty list rather than nil")
	}

	// The two are inverses, which is what lets a value written to Zitadel be
	// compared with the one that comes back.
	if diff := cmp.Diff(names, EnumNames(values)); diff != "" {
		t.Errorf("round trip: -want, +got:\n%s", diff)
	}
}

// TestEqualStringPointers covers the "an unset field is not compared" rule for
// an optional string. ZITADEL never returns a client secret or a signing key, so
// this is how a value that can only be written, never read, stays out of the
// drift comparison.
func TestEqualStringPointers(t *testing.T) {
	cases := map[string]struct {
		reason string
		want   *string
		got    string
		equal  bool
	}{
		"Unset": {
			reason: "A field the manifest never set is not compared, so ZITADEL's own value stands",
			want:   nil,
			got:    "whatever-zitadel-has",
			equal:  true,
		},
		"Matching": {
			reason: "A field that matches is equal",
			want:   strPtr("x"),
			got:    "x",
			equal:  true,
		},
		"Different": {
			reason: "A field that differs is not equal",
			want:   strPtr("x"),
			got:    "y",
			equal:  false,
		},
		"SetToEmpty": {
			reason: "Asking for an empty value is a real request, not an omission",
			want:   strPtr(""),
			got:    "y",
			equal:  false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := EqualStringPointers(tc.want, tc.got); got != tc.equal {
				t.Errorf("\n%s\nEqualStringPointers(...): want %v, got %v", tc.reason, tc.equal, got)
			}
		})
	}
}

// TestSecretValue covers reading a credential out of a secret, including the
// namespace rule: a cluster scoped resource has no namespace of its own to fall
// back on, so the reference has to name one.
func TestSecretValue(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: testNamespace},
		Data:       map[string][]byte{"key.json": []byte(`{"type":"serviceaccount"}`)},
	}
	elsewhere := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "shared", Namespace: "other"},
		Data:       map[string][]byte{"token": []byte("a-pat")},
	}

	kube := resolveKube(t, secret, elsewhere)
	ctx := context.Background()

	cases := map[string]struct {
		reason string
		ns     string
		ref    *v1alpha1.SecretKeySelector
		want   string
	}{
		"Local": {
			reason: "A reference without a namespace reads the secret where the resource lives",
			ns:     testNamespace,
			ref:    &v1alpha1.SecretKeySelector{Name: "credentials", Key: "key.json"},
			want:   `{"type":"serviceaccount"}`,
		},
		"NamedNamespace": {
			reason: "A reference naming a namespace reads from there, which is how a cluster scoped resource does it",
			ns:     "",
			ref:    &v1alpha1.SecretKeySelector{Name: "shared", Namespace: "other", Key: "token"},
			want:   "a-pat",
		},
		"NilRef": {
			reason: "No reference at all reads nothing",
			ns:     testNamespace,
			ref:    nil,
		},
		"NoName": {
			reason: "A reference with no name reads nothing",
			ns:     testNamespace,
			ref:    &v1alpha1.SecretKeySelector{Key: "key.json"},
		},
		"NoKey": {
			reason: "A reference with no key reads nothing",
			ns:     testNamespace,
			ref:    &v1alpha1.SecretKeySelector{Name: "credentials"},
		},
		"MissingSecret": {
			reason: "A secret that does not exist reads nothing rather than failing the reconcile",
			ns:     testNamespace,
			ref:    &v1alpha1.SecretKeySelector{Name: "absent", Key: "key.json"},
		},
		"MissingKey": {
			reason: "A key that is not in the secret reads nothing",
			ns:     testNamespace,
			ref:    &v1alpha1.SecretKeySelector{Name: "credentials", Key: "absent"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := SecretValue(ctx, kube, tc.ns, tc.ref); got != tc.want {
				t.Errorf("\n%s\nSecretValue(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}
}

// --- helpers ------------------------------------------------------------

func ptrTo[T any](v T) *T { return &v }

func boolPtr2(v bool) *bool { return &v }

func containsSubstring(haystack, needle string) bool {
	return len(needle) == 0 || indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}

	return -1
}

// errorsIs reports whether err mentions the sentinel. errors.Join does not
// implement Unwrap for a []error the way a chain does, so the message is checked
// as well as the identity.
func errorsIs(err error, target error) bool {
	return stderrors.Is(err, target) || containsSubstring(err.Error(), target.Error())
}

// unused keeps the time import honest for the FormatTime related cases above.
var _ = time.RFC3339
