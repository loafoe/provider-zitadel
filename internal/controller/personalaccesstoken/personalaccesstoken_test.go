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

package personalaccesstoken

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

func TestIsUpToDate(t *testing.T) {
	expires := metav1.NewTime(metav1.Now().Rfc3339Copy().Time)

	cases := map[string]struct {
		reason string
		fp     v1alpha1.PersonalAccessTokenParameters
		remote zitadel.PersonalAccessToken
		want   bool
	}{
		"NoExpirationRequested": {
			reason: "A token without a desired expiration is always up to date",
			remote: zitadel.PersonalAccessToken{ExpirationDate: "2024-01-02T03:04:05Z"},
			want:   true,
		},
		"MatchingExpiration": {
			reason: "A matching expiration is up to date",
			fp:     v1alpha1.PersonalAccessTokenParameters{ExpirationDate: &expires},
			remote: zitadel.PersonalAccessToken{ExpirationDate: expires.UTC().Format("2006-01-02T15:04:05Z07:00")},
			want:   true,
		},
		"DifferentExpiration": {
			reason: "A differing expiration is drift",
			fp:     v1alpha1.PersonalAccessTokenParameters{ExpirationDate: &expires},
			remote: zitadel.PersonalAccessToken{ExpirationDate: "1999-01-02T03:04:05Z"},
			want:   false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := &v1alpha1.PersonalAccessToken{Spec: v1alpha1.PersonalAccessTokenSpec{ForProvider: tc.fp}}
			if got := isUpToDate(cr, &tc.remote); got != tc.want {
				t.Errorf("\n%s\nisUpToDate(...): want %v, got %v", tc.reason, tc.want, got)
			}
		})
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.PersonalAccessToken{}

	updateStatus(cr, &zitadel.PersonalAccessToken{
		TokenID:        "t1",
		UserID:         "u1",
		OrganizationID: "o1",
		ExpirationDate: "2024-01-02T03:04:05Z",
	})

	if got := common.Deref(cr.Status.AtProvider.TokenID); got != "t1" {
		t.Errorf("tokenID: want %q, got %q", "t1", got)
	}
	if got := common.Deref(cr.Status.AtProvider.UserID); got != "u1" {
		t.Errorf("userID: want %q, got %q", "u1", got)
	}
	if cr.Status.AtProvider.ExpirationDate == nil {
		t.Error("expirationDate: want a timestamp, got nil")
	}
}
