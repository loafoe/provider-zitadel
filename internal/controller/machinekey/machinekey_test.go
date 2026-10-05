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

package machinekey

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// A machine key carries an expiration date because Zitadel requires one, so the
// interesting behaviour is what happens when the spec leaves it out: a year is a
// long enough default not to cause an outage and short enough that rotation
// stays routine.

func TestExpirationOf(t *testing.T) {
	want := time.Date(2027, time.March, 1, 12, 0, 0, 0, time.UTC)

	cr := &v1alpha1.MachineKey{Spec: v1alpha1.MachineKeySpec{
		ForProvider: v1alpha1.MachineKeyParameters{
			ExpirationDate: &metav1.Time{Time: want},
		},
	}}

	if got := expirationOf(cr); !got.Equal(want) {
		t.Errorf("expirationOf(...): want %v, got %v", want, got)
	}

	// An unset date is a year out from now, which is the whole point of the
	// default: Zitadel rejects a key without an expiration.
	unset := &v1alpha1.MachineKey{}
	got := expirationOf(unset)

	delta := time.Until(got)
	if delta <= defaultMachineKeyValidity-time.Hour || delta >= defaultMachineKeyValidity+time.Hour {
		t.Errorf("expirationOf(...): want about %v from now, got %v", defaultMachineKeyValidity, delta)
	}
}

func TestUpdateStatus(t *testing.T) {
	cr := &v1alpha1.MachineKey{}

	updateStatus(cr, &zitadel.MachineKey{
		KeyID:          "k1",
		UserID:         "u1",
		OrganizationID: "org-1",
		ExpirationDate: "2027-03-01T12:00:00Z",
		CreationDate:   "2026-03-01T12:00:00Z",
	})

	ap := cr.Status.AtProvider

	for name, tc := range map[string]struct {
		got  *string
		want string
	}{
		"KeyID":            {ap.KeyID, "k1"},
		"ServiceAccountID": {ap.ServiceAccountID, "u1"},
		"OrganizationID":   {ap.OrganizationID, "org-1"},
	} {
		if tc.got == nil || *tc.got != tc.want {
			t.Errorf("%s: want %q, got %v", name, tc.want, tc.got)
		}
	}

	if ap.ExpirationDate == nil || !ap.ExpirationDate.Time.Equal(time.Date(2027, time.March, 1, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("ExpirationDate: want 2027-03-01T12:00:00Z, got %v", ap.ExpirationDate)
	}

	if ap.CreationDate == nil || !ap.CreationDate.Time.Equal(time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("CreationDate: want 2026-03-01T12:00:00Z, got %v", ap.CreationDate)
	}
}

// TestUpdateStatusWithoutAnOrganization covers a key whose organization is not
// known: the rest of the observation is still recorded.
func TestUpdateStatusWithoutAnOrganization(t *testing.T) {
	cr := &v1alpha1.MachineKey{}

	updateStatus(cr, &zitadel.MachineKey{KeyID: "k1", UserID: "u1"})

	if cr.Status.AtProvider.OrganizationID != nil {
		t.Errorf("OrganizationID: want it left unset, got %v", cr.Status.AtProvider.OrganizationID)
	}
}
