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

package humanuser

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	userv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user/v2"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

const (
	testGivenName  = "Alice"
	testFamilyName = "Smith"
	testEmail      = "alice@example.com"
)

func TestProfile(t *testing.T) {
	cases := map[string]struct {
		reason string
		fp     v1alpha1.HumanUserParameters
		want   zitadel.SetHumanProfile
	}{
		"RequiredOnly": {
			reason: "Only the required name fields are set",
			fp:     v1alpha1.HumanUserParameters{GivenName: testGivenName, FamilyName: testFamilyName},
			want: zitadel.SetHumanProfile{
				GivenName:  testGivenName,
				FamilyName: testFamilyName,
			},
		},
		"OptionalFields": {
			reason: "Set optional fields are marked as present",
			fp: v1alpha1.HumanUserParameters{
				GivenName:         testGivenName,
				FamilyName:        testFamilyName,
				NickName:          ptr("Ali"),
				DisplayName:       ptr("Alice S."),
				PreferredLanguage: ptr("en"),
				Gender:            ptr(v1alpha1.GenderFemale),
			},
			want: zitadel.SetHumanProfile{
				GivenName:         testGivenName,
				FamilyName:        testFamilyName,
				NickName:          "Ali",
				HasNickName:       true,
				DisplayName:       "Alice S.",
				HasDisplayName:    true,
				PreferredLanguage: "en",
				HasLanguage:       true,
				Gender:            userv2.Gender_GENDER_FEMALE,
				HasGender:         true,
			},
		},
		"EmptyOptional": {
			reason: "An explicitly empty optional field is still sent as present",
			fp: v1alpha1.HumanUserParameters{
				GivenName:   testGivenName,
				FamilyName:  testFamilyName,
				NickName:    ptr(""),
				DisplayName: ptr(""),
			},
			want: zitadel.SetHumanProfile{
				GivenName:      testGivenName,
				FamilyName:     testFamilyName,
				NickName:       "",
				HasNickName:    true,
				DisplayName:    "",
				HasDisplayName: true,
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if diff := cmp.Diff(tc.want, *profile(tc.fp)); diff != "" {
				t.Errorf("\n%s\nprofile(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestIsUpToDate(t *testing.T) {
	remote := zitadel.User{
		UserID:   "u1",
		UserName: testEmail,
		State:    "Active",
		Human: &zitadel.HumanUserDetails{
			GivenName:         testGivenName,
			FamilyName:        testFamilyName,
			Email:             testEmail,
			Phone:             "+491234567890",
			DisplayName:       "Alice S.",
			NickName:          "Ali",
			PreferredLanguage: "en",
			Gender:            "Female",
		},
	}

	cases := map[string]struct {
		reason string
		fp     v1alpha1.HumanUserParameters
		want   bool
	}{
		"Unchanged": {
			reason: "A user matching the desired state is up to date",
			fp: v1alpha1.HumanUserParameters{
				GivenName:  testGivenName,
				FamilyName: testFamilyName,
				Email:      testEmail,
			},
			want: true,
		},
		"Renamed": {
			reason: "A differing given name is drift",
			fp: v1alpha1.HumanUserParameters{
				GivenName:  "Alicia",
				FamilyName: testFamilyName,
				Email:      testEmail,
			},
			want: false,
		},
		"EmailDrift": {
			reason: "A differing email is drift",
			fp: v1alpha1.HumanUserParameters{
				GivenName:  testGivenName,
				FamilyName: testFamilyName,
				Email:      "alice@other.example.com",
			},
			want: false,
		},
		"PhoneDrift": {
			reason: "A differing phone is drift",
			fp: v1alpha1.HumanUserParameters{
				GivenName:  testGivenName,
				FamilyName: testFamilyName,
				Email:      testEmail,
				Phone:      ptr("+4900000000"),
			},
			want: false,
		},
		"GenderDrift": {
			reason: "A differing gender is drift",
			fp: v1alpha1.HumanUserParameters{
				GivenName:  testGivenName,
				FamilyName: testFamilyName,
				Email:      testEmail,
				Gender:     ptr(v1alpha1.GenderMale),
			},
			want: false,
		},
		"StateDrift": {
			reason: "A differing state is drift",
			fp: v1alpha1.HumanUserParameters{
				GivenName:  testGivenName,
				FamilyName: testFamilyName,
				Email:      testEmail,
				State:      ptr(v1alpha1.UserStateInactive),
			},
			want: false,
		},
		"UserNameDrift": {
			reason: "A differing userName is drift",
			fp: v1alpha1.HumanUserParameters{
				GivenName:  testGivenName,
				FamilyName: testFamilyName,
				Email:      testEmail,
				UserName:   ptr("alice"),
			},
			want: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.HumanUser{Spec: v1alpha1.HumanUserSpec{ForProvider: tc.fp}}
			if got := isUpToDate(cr, &remote); got != tc.want {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestIsUpToDateMachineUser(t *testing.T) {
	cr := &v1alpha1.HumanUser{
		Spec: v1alpha1.HumanUserSpec{
			ForProvider: v1alpha1.HumanUserParameters{GivenName: testGivenName, FamilyName: testFamilyName, Email: testEmail},
		},
	}

	if isUpToDate(cr, &zitadel.User{Machine: &zitadel.MachineUserDetails{Name: "worker"}}) {
		t.Error("isUpToDate(...): want false for a machine user, got true")
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.HumanUser{}

	updateStatus(cr, &zitadel.User{
		UserID:             "u1",
		UserName:           testEmail,
		PreferredLoginName: testEmail,
		LoginNames:         []string{testEmail, "alice"},
		State:              "Active",
		CreationDate:       "2024-01-02T03:04:05Z",
		ChangeDate:         "2024-02-03T04:05:06Z",
		Human: &zitadel.HumanUserDetails{
			Email:                 testEmail,
			EmailVerified:         true,
			Phone:                 "+491234567890",
			PhoneVerified:         true,
			DisplayName:           "Alice S.",
			NickName:              "Ali",
			PasswordChangeRequire: true,
		},
	})

	if got := common.Deref(cr.Status.AtProvider.ID); got != "u1" {
		t.Errorf("id: want %q, got %q", "u1", got)
	}
	if got := common.Deref(cr.Status.AtProvider.PreferredLoginName); got != testEmail {
		t.Errorf("preferredLoginName: want %q, got %q", testEmail, got)
	}
	if !common.Bool(cr.Status.AtProvider.EmailVerified) {
		t.Error("emailVerified: want true, got false")
	}
	if !common.Bool(cr.Status.AtProvider.PasswordChangeRequired) {
		t.Error("passwordChangeRequired: want true, got false")
	}
	if cr.Status.AtProvider.State == nil || *cr.Status.AtProvider.State != v1alpha1.UserStateActive {
		t.Errorf("state: want %q, got %v", v1alpha1.UserStateActive, cr.Status.AtProvider.State)
	}
	if cr.Status.AtProvider.CreationDate == nil {
		t.Error("creationDate: want a timestamp, got nil")
	}
}

func ptr[T any](v T) *T { return &v }
