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

package serviceaccount

import (
	"testing"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

const (
	testServiceAccountName = "billing-worker"
)

func TestIsUpToDate(t *testing.T) {
	remote := zitadel.User{
		UserID:             "u1",
		UserName:           testServiceAccountName,
		PreferredLoginName: testServiceAccountName,
		State:              "Active",
		Machine: &zitadel.MachineUserDetails{
			Name:            "Billing Worker",
			Description:     "Background worker",
			AccessTokenType: "Jwt",
			HasSecret:       false,
		},
	}

	cases := map[string]struct {
		reason string
		fp     v1alpha1.ServiceAccountParameters
		want   bool
	}{
		"Unchanged": {
			reason: "A service account matching the desired state is up to date",
			fp:     v1alpha1.ServiceAccountParameters{UserName: ptr(testServiceAccountName)},
			want:   true,
		},
		"NameDrift": {
			reason: "A differing name is drift",
			fp:     v1alpha1.ServiceAccountParameters{Name: ptr("Other")},
			want:   false,
		},
		"DescriptionDrift": {
			reason: "A differing description is drift",
			fp:     v1alpha1.ServiceAccountParameters{Description: ptr("Something else")},
			want:   false,
		},
		"AccessTokenTypeDrift": {
			reason: "A differing access token type is drift",
			fp:     v1alpha1.ServiceAccountParameters{AccessTokenType: ptr(v1alpha1.AccessTokenTypeBearer)},
			want:   false,
		},
		"StateDrift": {
			reason: "A differing state is drift",
			fp:     v1alpha1.ServiceAccountParameters{State: ptr(v1alpha1.UserStateInactive)},
			want:   false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.ServiceAccount{Spec: v1alpha1.ServiceAccountSpec{ForProvider: tc.fp}}
			if got := isUpToDate(cr, &remote); got != tc.want {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestIsUpToDateHumanUser(t *testing.T) {
	cr := &v1alpha1.ServiceAccount{Spec: v1alpha1.ServiceAccountSpec{}}

	if isUpToDate(cr, &zitadel.User{Human: &zitadel.HumanUserDetails{GivenName: "Alice"}}) {
		t.Error("isUpToDate(...): want false for a human user, got true")
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.ServiceAccount{}

	updateStatus(cr, &zitadel.User{
		UserID:             "u1",
		UserName:           testServiceAccountName,
		PreferredLoginName: testServiceAccountName,
		State:              "Active",
		Machine: &zitadel.MachineUserDetails{
			Name:            "Billing Worker",
			Description:     "Background worker",
			AccessTokenType: "Jwt",
			HasSecret:       true,
		},
	})

	if got := common.Deref(cr.Status.AtProvider.ID); got != "u1" {
		t.Errorf("id: want %q, got %q", "u1", got)
	}
	if got := common.Deref(cr.Status.AtProvider.Name); got != "Billing Worker" {
		t.Errorf("name: want %q, got %q", "Billing Worker", got)
	}
	if cr.Status.AtProvider.AccessTokenType == nil || *cr.Status.AtProvider.AccessTokenType != v1alpha1.AccessTokenTypeJwt {
		t.Errorf("accessTokenType: want %q, got %v", v1alpha1.AccessTokenTypeJwt, cr.Status.AtProvider.AccessTokenType)
	}
	if !common.Bool(cr.Status.AtProvider.HasSecret) {
		t.Error("hasSecret: want true, got false")
	}
}

func TestConnectionDetails(t *testing.T) {
	details := connectionDetails(&zitadel.User{UserID: "u1", PreferredLoginName: testServiceAccountName})
	if got := string(details[v1alpha1.ConnectionKeyUserID]); got != "u1" {
		t.Errorf("userID: want %q, got %q", "u1", got)
	}
	if got := string(details[v1alpha1.ConnectionKeyUsername]); got != testServiceAccountName {
		t.Errorf("username: want %q, got %q", testServiceAccountName, got)
	}
}

func ptr[T any](v T) *T { return &v }
