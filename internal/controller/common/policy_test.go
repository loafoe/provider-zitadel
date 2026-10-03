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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// The tests drive the harness with a real managed resource kind, so that what
// is exercised is the same thing a controller uses rather than a stand-in that
// could satisfy the interface while behaving differently.
type testCR = v1alpha1.LockoutPolicy

func testResource() *testCR {
	cr := &testCR{}
	cr.Name = "test-policy"

	return cr
}

// testDriver is a policy whose scope is always resolved and whose state is held
// in the test rather than in Zitadel.
type testDriver struct {
	// observed is what Get reports, and inherited whether it is the default.
	observed  zitadel.LockoutPolicy
	inherited bool

	// applied records every write, so a test can see what Delete tried to do.
	applied []zitadel.LockoutPolicyInput

	// resettable mirrors what a driver reports about being able to reset.
	resettable bool

	// resetErr is what Reset returns.
	resetErr error
}

func (d *testDriver) Kind() string { return "TestPolicy" }

func (d *testDriver) Scope(_ context.Context, _ client.Client, _ ManagedPolicy) (string, error) {
	return "org-1", nil
}

func (d *testDriver) Get(_ context.Context, _ *zitadel.Client, _ string) (zitadel.LockoutPolicy, bool, error) {
	return d.observed, d.inherited, nil
}

func (d *testDriver) Apply(_ context.Context, _ *zitadel.Client, _ string, want zitadel.LockoutPolicyInput) error {
	d.applied = append(d.applied, want)

	return d.resetErr
}

func (d *testDriver) Resettable() bool { return d.resettable }

func (d *testDriver) Reset(_ context.Context, _ *zitadel.Client, _ string) error { return d.resetErr }

func (d *testDriver) Desired(_ ManagedPolicy) zitadel.LockoutPolicyInput {
	return zitadel.LockoutPolicyInput{MaxPasswordAttempts: 5}
}

func (d *testDriver) Report(_ ManagedPolicy, _ zitadel.LockoutPolicy) {}

func (d *testDriver) Equal(want zitadel.LockoutPolicyInput, got zitadel.LockoutPolicy) bool {
	return want.MaxPasswordAttempts == got.MaxPasswordAttempts
}

func newTestExternal(d *testDriver) *policyExternal[zitadel.LockoutPolicyInput, zitadel.LockoutPolicy, *testCR] {
	return &policyExternal[zitadel.LockoutPolicyInput, zitadel.LockoutPolicy, *testCR]{d: d}
}

// An inherited policy still exists. Reporting it as absent would send the write
// through Create, and Crossplane answers that by throwing the observation away -
// so the status would never say what the policy currently is.
func TestObserveTreatsAnInheritedPolicyAsExisting(t *testing.T) {
	d := &testDriver{inherited: true, observed: zitadel.LockoutPolicy{MaxPasswordAttempts: 1}}
	e := newTestExternal(d)

	got, err := e.Observe(context.Background(), testResource())
	if err != nil {
		t.Fatal(err)
	}

	if !got.ResourceExists {
		t.Error("an inherited policy reported as absent, which routes writes through Create and discards the status")
	}

	if got.ResourceUpToDate {
		t.Error("a policy that differs from the spec reported as up to date")
	}
}

// A policy is deleted by resetting it, so a terminating resource is finished
// once the scope is inheriting the default again.
//
// Reporting it as still existing is the bug this exists for: Crossplane waits
// for the external resource to disappear before releasing the finalizer, so a
// reset policy that always reports itself present leaves the object stuck in
// Terminating forever.
func TestObserveReportsAResetPolicyAsGone(t *testing.T) {
	d := &testDriver{inherited: true, observed: zitadel.LockoutPolicy{MaxPasswordAttempts: 1}}
	e := newTestExternal(d)

	cr := testResource()
	now := metav1.Now()
	cr.SetDeletionTimestamp(&now)

	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	if got.ResourceExists {
		t.Error("a terminating policy that is back on the instance default still reports itself existing, so the finalizer is never released")
	}
}

// A policy the organization still owns is not gone, however long the object has
// been terminating: resetting it is what makes it gone, and that has not
// happened yet.
func TestObserveReportsAnOwnedPolicyAsStillThere(t *testing.T) {
	d := &testDriver{resettable: true, inherited: false, observed: zitadel.LockoutPolicy{MaxPasswordAttempts: 5}}
	e := newTestExternal(d)

	cr := testResource()
	now := metav1.Now()
	cr.SetDeletionTimestamp(&now)

	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	if !got.ResourceExists {
		t.Error("a policy the organization still owns reported as gone, which would release the finalizer without resetting it")
	}
}

// Deleting a resettable policy resets it and records nothing to restore.
func TestDeleteResetsAResettablePolicy(t *testing.T) {
	d := &testDriver{resettable: true}
	e := newTestExternal(d)

	cr := testResource()
	cr.SetPolicyScope("org-1")
	if _, err := e.Delete(context.Background(), cr); err != nil {
		t.Fatal(err)
	}

	if len(d.applied) != 0 {
		t.Errorf("a resettable policy was written rather than reset: %v", d.applied)
	}
}

// Deleting a policy Zitadel cannot reset puts back what was there before this
// resource overwrote it, which is the only thing that makes deleting one
// meaningful.
func TestDeleteRestoresWhatWasOverwritten(t *testing.T) {
	d := &testDriver{resettable: false, observed: zitadel.LockoutPolicy{MaxPasswordAttempts: 9}}
	e := newTestExternal(d)

	cr := testResource()
	cr.SetPolicyScope("org-1")

	// The first update records the policy it is about to overwrite.
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatal(err)
	}

	if cr.PolicyRestore() == nil {
		t.Fatal("the policy that was overwritten was not recorded, so deleting could not put it back")
	}

	// The second update must not overwrite the restore point with the policy
	// this resource itself wrote.
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatal(err)
	}

	if _, err := e.Delete(context.Background(), cr); err != nil {
		t.Fatal(err)
	}

	if len(d.applied) != 3 {
		t.Fatalf("want two writes and one restore, got %d: %v", len(d.applied), d.applied)
	}

	if d.applied[2].MaxPasswordAttempts != 9 {
		t.Errorf("delete restored %d, want the 9 that was there before", d.applied[2].MaxPasswordAttempts)
	}

	if cr.PolicyRestore() != nil {
		t.Error("the restore point outlived the delete")
	}
}

// A resource that is created once and never edited reaches no Update at all, so
// the value it is about to overwrite has to be recorded on the way through
// Create as well.
//
// Recording it only on Update is the bug this covers: the restore point stayed
// nil, deleting the resource succeeded without restoring anything, and the
// instance was left configured by a manifest that no longer existed. Verified
// against a live instance, where the restrictions and the init code generator
// both survived their own deletion.
func TestCreateRecordsTheValueItIsAboutToOverwrite(t *testing.T) {
	d := &testDriver{resettable: false, observed: zitadel.LockoutPolicy{MaxPasswordAttempts: 9}}
	e := newTestExternal(d)

	cr := testResource()
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatal(err)
	}

	if cr.PolicyRestore() == nil {
		t.Fatal("nothing was recorded, so deleting could not put the scope back")
	}

	if _, err := e.Delete(context.Background(), cr); err != nil {
		t.Fatal(err)
	}

	if len(d.applied) != 2 {
		t.Fatalf("want one write and one restore, got %d: %v", len(d.applied), d.applied)
	}

	if d.applied[1].MaxPasswordAttempts != 9 {
		t.Errorf("delete restored %d, want the 9 that was there before", d.applied[1].MaxPasswordAttempts)
	}
}

// A policy that Zitadel cannot be reset has to be writable from a recorded
// snapshot, which the harness finds through the Policy interface.
//
// This is the bug this covers: the observed type of the new instance settings
// had no Input method, so the restore was refused with "cannot be restored from
// a recorded value" and deleting the resource left the instance holding the
// values its manifest had written. Verified against a live instance, where the
// recorded restore point was right and was still never applied.
func TestAnObservedPolicyCanBeWrittenBackFromARecord(t *testing.T) {
	for name, observed := range map[string]any{
		"a lockout policy": zitadel.LockoutPolicy{MaxPasswordAttempts: 5},
		"the flags":        zitadel.FeatureFlags{LoginDefaultOrg: true},
		"the restrictions": zitadel.Restrictions{DisallowPublicOrgRegistration: true},
		"a generator":      zitadel.SecretGenerator{Length: 8},
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := observed.(Policy); !ok {
				t.Errorf("%T has no Input method, so a recorded restore point can never be written back",
					observed)
			}
		})
	}
}

// A resettable policy must not record a restore point: deleting it resets, so
// there is nothing to put back and reading the scope must not change anything.
func TestUpdateDoesNotRecordARestorePointForAResettablePolicy(t *testing.T) {
	d := &testDriver{resettable: true}
	e := newTestExternal(d)

	cr := testResource()
	if _, err := e.Update(context.Background(), cr); err != nil {
		t.Fatal(err)
	}

	if cr.PolicyRestore() != nil {
		t.Error("a policy that can be reset recorded a restore point")
	}

	// Creating one must not record it either, or deleting it would restore a
	// value over the reset that is supposed to happen instead.
	cr = testResource()
	if _, err := e.Create(context.Background(), cr); err != nil {
		t.Fatal(err)
	}

	if cr.PolicyRestore() != nil {
		t.Error("creating a policy that can be reset recorded a restore point")
	}
}

// A policy Zitadel cannot reset is restored rather than removed, so it never
// becomes "absent" the way a reset one does. The recorded restore point is what
// says whether the delete still has work to do.
//
// Reporting it as still there forever would leave the finalizer in place; the
// opposite would let the finalizer go before the restore was ever written.
func TestObserveTracksARestorePointThatStillHasToBeWritten(t *testing.T) {
	d := &testDriver{resettable: false, observed: zitadel.LockoutPolicy{MaxPasswordAttempts: 9}}
	e := newTestExternal(d)

	cr := testResource()
	cr.SetDeletionTimestamp(&metav1.Time{Time: time.Now()})
	cr.SetPolicyRestore([]byte(`{"MaxPasswordAttempts":9}`))

	got, err := e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	if !got.ResourceExists {
		t.Error("a policy with a restore point still to write reported itself gone, so Delete would never run")
	}

	// Delete writes the recorded value back and clears the point, which is what
	// tells Observe that there is nothing left to do.
	if _, err := e.Delete(context.Background(), cr); err != nil {
		t.Fatal(err)
	}

	if cr.PolicyRestore() != nil {
		t.Fatal("delete left the restore point behind, so the policy would never be seen as gone")
	}

	got, err = e.Observe(context.Background(), cr)
	if err != nil {
		t.Fatal(err)
	}

	if got.ResourceExists {
		t.Error("a policy whose recorded value has been written back still reports itself existing, so the finalizer is never released")
	}
}
