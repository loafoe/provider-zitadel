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
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	actionv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/action/v2"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Bindings
//
// A binding says "when this happens, call these". Zitadel has several of them
// - an action execution for a request, a response, an event or a function, and
// the actions a login flow triggers - and none of them is an object with an
// identifier. There is no "create execution" call, no ID to hold on to, and no
// delete: a binding is removed by writing it with nothing bound.
//
// So all of them are the same shape, and the lifecycle is here once:
//
//   - the condition is the identity, and it is derived rather than read back;
//   - reading means listing everything and finding the condition;
//   - writing is always a full write of the target list;
//   - deleting is a write with no targets.
//
// CR is the managed resource kind and H whatever the SDK's condition type is for
// that kind, which keeps the harness out of the middle of the oneof.

// ManagedBinding is what a binding managed resource adds to resource.Managed.
type ManagedBinding interface {
	resource.ModernManaged
	resource.Conditioned
}

// BindingDriver is everything that differs between one kind of binding and the
// next.
type BindingDriver[CR ManagedBinding, H any] interface {
	// Kind names the binding, for errors and events.
	Kind() string

	// Condition returns the handle the SDK addresses the binding by, and the
	// identity derived from it.
	Condition(cr CR) (H, string, error)

	// Get returns the targets currently bound, and whether the binding is there
	// at all.
	Get(ctx context.Context, c *zitadel.Client, condition H) ([]string, bool, error)

	// Set writes the whole binding.
	Set(ctx context.Context, c *zitadel.Client, condition H, targets []string) error

	// Targets reads the desired targets out of the managed resource.
	Targets(cr CR) []string

	// Report writes the observed targets into the managed resource's status.
	Report(cr CR, targets []string)

	// Equal reports whether the observed targets already match the desired ones.
	Equal(want, got []string) bool
}

// bindingExternal reconciles one kind of binding.
type bindingExternal[CR ManagedBinding, H any] struct {
	kube client.Client
	zc   *zitadel.Client
	d    BindingDriver[CR, H]
}

// Disconnect releases this reconcile's claim on the Zitadel client. The
// client itself is usually shared and outlives the reconcile, so this is a
// no-op unless the client is not owned by a cache.
func (e *bindingExternal[CR, H]) Disconnect(_ context.Context) error {
	e.zc.Release()

	return nil
}

func (e *bindingExternal[CR, H]) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalObservation{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	condition, id, err := e.d.Condition(cr)
	if err != nil {
		if meta.WasDeleted(cr) {
			// Nothing resolves while an object is terminating, so report the
			// binding as gone. That lets the reconciler run Delete, which lets the
			// finalizer go.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	// The identity is derived from the condition rather than read back, so it is
	// set on every observe and not only when it is missing.
	meta.SetExternalName(cr, id)

	targets, _, err := e.d.Get(ctx, e.zc, condition)
	if err != nil {
		if zitadel.IsNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}

		return managed.ExternalObservation{}, err
	}

	e.d.Report(cr, targets)
	cr.SetConditions(xpv1.Available())

	exists := true

	if meta.WasDeleted(cr) {
		// A binding is deleted by clearing it, so a terminating resource is
		// finished once nothing is bound any more. Without this the finalizer is
		// never released: Crossplane waits for the external resource to
		// disappear, and a cleared binding never does.
		exists = len(targets) > 0
	}

	return managed.ExternalObservation{
		ResourceExists:   exists,
		ResourceUpToDate: e.d.Equal(e.d.Targets(cr), targets),
	}, nil
}

func (e *bindingExternal[CR, H]) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalCreation{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	cr.SetConditions(xpv1.Creating())

	if err := e.write(ctx, cr); err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

func (e *bindingExternal[CR, H]) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalUpdate{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	return managed.ExternalUpdate{}, e.write(ctx, cr)
}

// write applies the desired targets to the binding.
func (e *bindingExternal[CR, H]) write(ctx context.Context, cr CR) error {
	condition, id, err := e.d.Condition(cr)
	if err != nil {
		return err
	}

	meta.SetExternalName(cr, id)

	return e.d.Set(ctx, e.zc, condition, e.d.Targets(cr))
}

func (e *bindingExternal[CR, H]) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(CR)
	if !ok {
		return managed.ExternalDelete{}, errors.Errorf("managed resource is not a %s", e.d.Kind())
	}

	cr.SetConditions(xpv1.Deleting())

	h, id, err := e.d.Condition(cr)
	if err != nil {
		// The condition lives in the spec, so a resource whose condition cannot
		// be built was never created and so has nothing bound.
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot read the binding to clear")
	}

	meta.SetExternalName(cr, id)

	// Clearing the binding is what deleting one means: Zitadel has no call that
	// removes an execution or a trigger, only one that writes it.
	return managed.ExternalDelete{}, e.d.Set(ctx, e.zc, h, nil)
}

// SetupBindingController wires up the managed resource reconciler for one kind
// of binding, driven by d.
func SetupBindingController[CR ManagedBinding, H any](mgr ctrl.Manager, o controller.Options,
	groupKind string, gvk schema.GroupVersionKind, obj CR, list resource.ManagedList,
	d BindingDriver[CR, H],
) error {
	return SetupManagedResourceController(mgr, o, groupKind, gvk, obj, list,
		func(_ context.Context, kube client.Client, mg resource.ModernManaged, zc *zitadel.Client) (managed.ExternalClient, error) {
			return &bindingExternal[CR, H]{kube: kube, zc: zc, d: d}, nil
		})
}

// DeriveExecutionID derives the identity of an action execution from its
// condition.
//
// Zitadel gives an execution no identifier of its own, so one has to be derived
// if the managed resource is to have an external name at all. The spelling
// follows Zitadel's own: a method keeps its leading slash, a service is set
// after one, and a catch-all condition is just the kind.
//
// The identity is not an ID Zitadel recognises. It is a name for the object, and
// it changes when the condition does, so it also says what a resource is bound
// to.
func DeriveExecutionID(field, value string) string {
	switch {
	case field == "All":
		return "execution"
	case strings.HasPrefix(value, "/"):
		return "execution" + value
	default:
		return "execution/" + value
	}
}

// SameExecutionCondition reports whether two Zitadel execution conditions are the
// same binding.
//
// Zitadel has no call for reading one execution, so the only way to find it is to
// list them all and match. The comparison therefore has to be structural: the
// SDK models a condition as a oneof, so only the arm that is set can match.
func SameExecutionCondition(a, b *actionv2.Condition) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	// The arm that is set decides: a request execution and a response execution
	// can name the same method and still be different bindings.
	arms := []func() bool{
		func() bool { return sameRequestCondition(a.GetRequest(), b.GetRequest()) },
		func() bool { return sameResponseCondition(a.GetResponse(), b.GetResponse()) },
		func() bool { return sameEventCondition(a.GetEvent(), b.GetEvent()) },
		func() bool { return a.GetFunction().GetName() == b.GetFunction().GetName() },
		// Neither condition is set: nothing is bound either way.
		func() bool { return true },
	}

	for _, arm := range arms {
		if !arm() {
			return false
		}
	}

	return true
}

func sameRequestCondition(a, b *actionv2.RequestExecution) bool {
	switch {
	case a == nil || b == nil:
		return a == nil && b == nil
	case a.GetMethod() != "" || b.GetMethod() != "":
		return a.GetMethod() == b.GetMethod()
	case a.GetService() != "" || b.GetService() != "":
		return a.GetService() == b.GetService()
	default:
		return a.GetAll() == b.GetAll()
	}
}

func sameResponseCondition(a, b *actionv2.ResponseExecution) bool {
	switch {
	case a == nil || b == nil:
		return a == nil && b == nil
	case a.GetMethod() != "" || b.GetMethod() != "":
		return a.GetMethod() == b.GetMethod()
	case a.GetService() != "" || b.GetService() != "":
		return a.GetService() == b.GetService()
	default:
		return a.GetAll() == b.GetAll()
	}
}

func sameEventCondition(a, b *actionv2.EventExecution) bool {
	switch {
	case a == nil || b == nil:
		return a == nil && b == nil
	case a.GetEvent() != "" || b.GetEvent() != "":
		return a.GetEvent() == b.GetEvent()
	case a.GetGroup() != "" || b.GetGroup() != "":
		return a.GetGroup() == b.GetGroup()
	default:
		return a.GetAll() == b.GetAll()
	}
}

// DeriveTriggerID derives the identity of a login flow trigger from where it
// runs.
//
// Zitadel gives a trigger no identifier of its own, so one has to be derived.
// The name is not an ID Zitadel recognises: it says where the binding applies,
// and changes when the flow or the trigger does.
func DeriveTriggerID(flow, trigger string) string {
	return "trigger/" + flow + "/" + trigger
}
