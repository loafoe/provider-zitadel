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
	"encoding/json"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Zitadel policies
//
// Zitadel is full of policies - login, lockout, password age, privacy, and a
// dozen more - and they are all the same shape: a singleton per scope, a fixed
// set of scalar settings, and a read that answers even when nothing has been
// customised for that scope. Every one of them also has an instance wide
// default that the very same fields describe.
//
// Seventeen such kinds would otherwise be seventeen near identical controllers,
// differing only in the handful of fields they carry and the three or four API
// calls they make. Everything around those calls - when a policy exists, what
// counts as drift, what deleting a policy means, and what happens when a
// reference has been replaced - is identical, so it lives here once.

// Policy is the client side representation of a Zitadel policy: what it looks
// like once read back.
//
// Every policy type in the client package satisfies it, which is what lets the
// harness serialise one generically when recording a restore point.
type Policy interface {
	// Input returns the same policy in the shape the write API accepts.
	Input() any
}

// ManagedPolicy is what a policy managed resource adds to resource.Managed so
// that the shared harness can drive it.
//
// A policy has no identity of its own: it is the singleton of a scope, so the
// Kubernetes object *is* the identity. All the harness records is the scope it
// last acted on and, for a scope Zitadel cannot reset, the value to put back
// when the resource is deleted.
type ManagedPolicy interface {
	resource.ModernManaged
	resource.Conditioned

	// SetPolicyScope records the scope the policy was last read from.
	SetPolicyScope(scope string)

	// PolicyScope returns the recorded scope.
	PolicyScope() string

	// SetPolicyRestore records the policy to put back when this resource is
	// deleted.
	SetPolicyRestore(restore []byte)

	// PolicyRestore returns the recorded restore point.
	PolicyRestore() []byte
}

// PolicyDriver is everything that differs between one Zitadel policy and the
// next. A policy that fits in a handful of fields implements this in a handful of
// lines; the harness supplies the lifecycle around it.
//
// P is the shape a policy is written in, and O the shape it is read back in.
// They are different because what Zitadel returns is not what it accepts: it
// adds the identifiers and an "is this the inherited default" flag, spells
// durations and enumerations its own way, and reports which fields it defaulted.
type PolicyDriver[P, O any] interface {
	// Kind names the policy, so errors and events say which policy failed.
	Kind() string

	// Scope resolves the organization the policy applies to. The empty string
	// means Zitadel's instance wide default rather than any organization. A
	// scoped policy that cannot resolve its organization returns an error, so an
	// empty scope never means "unresolved".
	Scope(ctx context.Context, kube client.Client, cr ManagedPolicy) (string, error)

	// Get reads the policy Zitadel currently holds for the scope, and reports
	// whether it is the one the scope inherits rather than its own.
	//
	// Inheritance decides what "gone" means for a policy: there is nothing to
	// delete, so a scope is back to where it started once it is inheriting again.
	// That is also how a terminating resource learns its delete has taken effect,
	// and how an operator can tell "this is what the instance sets for everyone"
	// from "this organization has customised it".
	Get(ctx context.Context, c *zitadel.Client, scope string) (O, bool, error)

	// Apply writes the policy to the scope.
	Apply(ctx context.Context, c *zitadel.Client, scope string, want P) error

	// Resettable reports whether deleting this resource can put the scope back
	// the way it was by resetting it to the instance default.
	//
	// It is a question and not a call, because asking must never change
	// anything: the answer is needed before the first write, and a reset
	// performed to find out would undo the very policy being managed.
	Resettable() bool

	// Reset reverts the scope to the instance default. It is only called for a
	// policy that is resettable.
	Reset(ctx context.Context, c *zitadel.Client, scope string) error

	// Desired reads the desired policy out of the managed resource.
	Desired(cr ManagedPolicy) P

	// Report writes an observed policy into the managed resource's status.
	Report(cr ManagedPolicy, observed O)

	// Equal reports whether the observed policy already matches the desired one.
	//
	// It takes the managed resource as well as the two values, because whether a
	// field counts as drift depends on whether the manifest set it. Desired
	// cannot express that on its own: it has to turn an unset field into
	// something, and the something it picks is the zero value. Every field of
	// every policy here is optional in the CRD, so without the managed resource
	// a manifest that omits a field would be compared against Zitadel's own
	// value for it and reported as drift on every poll, forever - fighting a
	// value the operator never asked to have managed.
	//
	// An unset field is therefore not compared, so Zitadel's own defaults are
	// never fought over.
	Equal(cr ManagedPolicy, want P, got O) bool
}

// Disconnect releases this reconcile's claim on the Zitadel client. The
// client itself is usually shared and outlives the reconcile, so this is a
// no-op unless the client is not owned by a cache.
func (e *policyExternal[P, O, CR]) Disconnect(_ context.Context) error {
	e.zc.Release()

	return nil
}

// policyExternal reconciles one Zitadel policy.
//
// P is the policy as the driver represents it and CR the managed resource kind
// that carries it.
type policyExternal[P, O any, CR ManagedPolicy] struct {
	kube client.Client
	zc   *zitadel.Client
	d    PolicyDriver[P, O]
}

// newPolicyExternal builds the external client of a policy managed resource.
func newPolicyExternal[P, O any, CR ManagedPolicy](kube client.Client, zc *zitadel.Client, mg resource.ModernManaged, d PolicyDriver[P, O]) (managed.ExternalClient, error) {
	if _, ok := mg.(CR); !ok {
		return nil, errors.Errorf("managed resource is not a %s", d.Kind())
	}

	return &policyExternal[P, O, CR]{kube: kube, zc: zc, d: d}, nil
}

func (e *policyExternal[P, O, CR]) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalObservation{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	scope, err := e.d.Scope(ctx, e.kube, cr)
	if err != nil {
		if meta.WasDeleted(cr) {
			// Nothing resolves while an object is terminating, so report the
			// policy as gone. That lets the reconciler run Delete, which lets the
			// finalizer go, instead of retrying an observation that can never
			// succeed.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	observed, inherited, err := e.d.Get(ctx, e.zc, scope)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	cr.SetPolicyScope(scope)
	e.d.Report(cr, observed)
	cr.SetConditions(xpv1.Available())

	// A scope always has a policy - its own, or the one it inherits - so an
	// inherited policy is reported as existing. Reporting it as absent would
	// route the write through Create, which Crossplane answers by discarding
	// the observation and with it the status saying what the policy currently is.
	exists := true

	if meta.WasDeleted(cr) {
		switch {
		case e.d.Resettable():
			// A policy Zitadel can reset is gone once the scope is inheriting the
			// default again. Without this the finalizer is never released:
			// Crossplane waits for the external resource to disappear, and a
			// reset policy never does.
			exists = !inherited

		default:
			// One Zitadel cannot reset is restored rather than removed, and that
			// happens in Delete. The recorded restore point outlives the delete
			// that used it, so its presence both asks for that delete and reports
			// that it has already happened.
			exists = cr.PolicyRestore() != nil
		}
	}

	return managed.ExternalObservation{
		ResourceExists:   exists,
		ResourceUpToDate: e.d.Equal(cr, e.d.Desired(cr), observed),
	}, nil
}

func (e *policyExternal[P, O, CR]) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalCreation{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	cr.SetConditions(xpv1.Creating())

	scope, err := e.d.Scope(ctx, e.kube, cr)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	// The value this is about to overwrite is recorded before the write, not only
	// on the way through Update.
	//
	// A resource that is created once and never edited otherwise reaches no
	// Update at all, so recording there alone would leave a policy Zitadel
	// cannot reset with no restore point: deleting it would then leave the scope
	// configured by a manifest that no longer exists.
	if err := e.recordRestorePoint(ctx, cr, scope); err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, e.d.Apply(ctx, e.zc, scope, e.d.Desired(cr))
}

func (e *policyExternal[P, O, CR]) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalUpdate{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	scope, err := e.d.Scope(ctx, e.kube, cr)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	if err := e.recordRestorePoint(ctx, cr, scope); err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, e.d.Apply(ctx, e.zc, scope, e.d.Desired(cr))
}

func (e *policyExternal[P, O, CR]) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalDelete{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	cr.SetConditions(xpv1.Deleting())

	// Deleting a policy means putting the scope back the way it was. A
	// terminating object must not follow a reference that has since moved: it
	// exists to undo what it did, so it acts on the scope it recorded.
	scope := cr.PolicyScope()
	if scope == "" {
		resolved, err := e.d.Scope(ctx, e.kube, cr)
		switch {
		case err == nil:
			scope = resolved

		case IsReferenceGone(err):
			// The organization is gone, and with it any policy it customised.

		default:
			return managed.ExternalDelete{}, err
		}
	}

	if e.d.Resettable() {
		cr.SetPolicyRestore(nil)

		return managed.ExternalDelete{}, e.d.Reset(ctx, e.zc, scope)
	}

	// Zitadel has no way to reset an instance wide policy, so the value this
	// resource overwrote is the only thing deletion can restore. Without it,
	// deleting the resource would leave the instance configured by a manifest
	// that no longer exists.
	snapshot := cr.PolicyRestore()
	if snapshot == nil {
		return managed.ExternalDelete{}, nil
	}

	// Only once the recorded value is safely in hand: the restore point outlives
	// this call, so a failed restore can be retried with it still recorded.
	cr.SetPolicyRestore(nil)

	var recorded O
	if err := json.Unmarshal(snapshot, &recorded); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot read the policy to restore on delete")
	}

	restore, ok := any(&recorded).(Policy)
	if !ok {
		return managed.ExternalDelete{}, errors.Errorf("the %s policy cannot be restored from a recorded value", e.d.Kind())
	}

	return managed.ExternalDelete{}, e.d.Apply(ctx, e.zc, scope, restore.Input().(P))
}

// recordRestorePoint captures the policy this resource is about to overwrite, so
// that deleting it can put the previous value back.
//
// It is a no-op for a policy Zitadel can reset - deleting it resets instead -
// and after the first write, so that the restore point is what the scope looked
// like before this resource ever touched it.
func (e *policyExternal[P, O, CR]) recordRestorePoint(ctx context.Context, cr CR, scope string) error {
	if e.d.Resettable() || cr.PolicyRestore() != nil {
		return nil
	}

	current, _, err := e.d.Get(ctx, e.zc, scope)
	if err != nil {
		return err
	}

	snapshot, err := json.Marshal(current)
	if err != nil {
		return errors.Wrap(err, "cannot record the policy to restore on delete")
	}

	cr.SetPolicyRestore(snapshot)

	return nil
}

// SetupPolicyController wires up the managed resource reconciler for one policy
// kind, driven by d.
func SetupPolicyController[P, O any, CR ManagedPolicy](mgr ctrl.Manager, o controller.Options,
	groupKind string, gvk schema.GroupVersionKind, obj CR, list resource.ManagedList,
	d PolicyDriver[P, O],
) error {
	return SetupManagedResourceController(mgr, o, groupKind, gvk, obj, list,
		PolicyExternalFactory[P, O, CR](d))
}

// PolicyExternalFactory returns the client factory for a policy driver.
//
// It is exported so that the namespaced and the cluster scoped setup of one kind
// are built from the same thing rather than from two that could drift apart.
func PolicyExternalFactory[P, O any, CR ManagedPolicy](d PolicyDriver[P, O]) NewExternalClientFn {
	return func(_ context.Context, kube client.Client, mg resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
		return newPolicyExternal[P, O, CR](kube, zc, mg, d)
	}
}
