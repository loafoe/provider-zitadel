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

// Package actionexecution reconciles the four kinds of Zitadel action execution.
//
// They differ only in which condition they bind on - a request, a response, an
// event, or a function another action calls - so they share one package, one
// driver shape and the lifecycle in internal/controller/common/binding.go.
// Writing four controllers for four oneofs of the same API would have been four
// copies of everything except the part that actually differs.
package actionexecution

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/pkg/errors"
	actionv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/action/v2"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
	"github.com/loafoe/provider-zitadel/internal/controller/common"
)

// Setup adds controllers for all four action execution kinds.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	if err := common.SetupBindingController(mgr, o,
		v1alpha1.ActionExecutionRequestGroupKind,
		v1alpha1.ActionExecutionRequestGroupVersionKind,
		&v1alpha1.ActionExecutionRequest{}, &v1alpha1.ActionExecutionRequestList{},
		requestDriver{}); err != nil {
		return err
	}

	if err := common.SetupBindingController(mgr, o,
		v1alpha1.ActionExecutionResponseGroupKind,
		v1alpha1.ActionExecutionResponseGroupVersionKind,
		&v1alpha1.ActionExecutionResponse{}, &v1alpha1.ActionExecutionResponseList{},
		responseDriver{}); err != nil {
		return err
	}

	if err := common.SetupBindingController(mgr, o,
		v1alpha1.ActionExecutionEventGroupKind,
		v1alpha1.ActionExecutionEventGroupVersionKind,
		&v1alpha1.ActionExecutionEvent{}, &v1alpha1.ActionExecutionEventList{},
		eventDriver{}); err != nil {
		return err
	}

	return common.SetupBindingController(mgr, o,
		v1alpha1.ActionExecutionFunctionGroupKind,
		v1alpha1.ActionExecutionFunctionGroupVersionKind,
		&v1alpha1.ActionExecutionFunction{}, &v1alpha1.ActionExecutionFunctionList{},
		functionDriver{})
}

// execution is the behaviour every execution kind shares. A kind supplies only
// the condition it binds on.
type execution[C any] struct{}

// Get finds this binding among all of them.
//
// There is no call for reading one execution, so every binding is listed and the
// one whose condition matches is taken.
func (e execution[C]) Get(ctx context.Context, c *zitadel.Client, condition *actionv2.Condition) ([]string, bool, error) {
	executions, err := c.ListExecutions(ctx)
	if err != nil {
		return nil, false, err
	}

	for _, x := range executions {
		if !common.SameExecutionCondition(x.Condition, condition) {
			continue
		}

		return x.Targets, true, nil
	}

	return nil, false, nil
}

// Set writes the whole binding, because Zitadel has no narrower call.
func (e execution[C]) Set(ctx context.Context, c *zitadel.Client, condition *actionv2.Condition, targets []string) error {
	return c.SetExecutions(ctx, []*actionv2.Condition{condition}, targets)
}

// Equal compares the desired targets with the observed ones, as sets: which
// targets are called, not what order Zitadel lists them in.
func (e execution[C]) Equal(want, got []string) bool { return common.EqualStringSlices(want, got) }

// requestDriver binds targets to an incoming gRPC request.
type requestDriver struct {
	execution[*v1alpha1.ActionExecutionRequest]
}

var _ common.BindingDriver[*v1alpha1.ActionExecutionRequest, *actionv2.Condition] = requestDriver{}

func (requestDriver) Kind() string { return "ActionExecutionRequest" }

func (requestDriver) Condition(cr *v1alpha1.ActionExecutionRequest) (*actionv2.Condition, string, error) {
	fp := cr.Spec.ForProvider

	condition := &actionv2.Condition{}

	switch {
	case fp.Method != nil && *fp.Method != "":
		condition.ConditionType = &actionv2.Condition_Request{
			Request: &actionv2.RequestExecution{
				Condition: &actionv2.RequestExecution_Method{Method: *fp.Method},
			},
		}

		return condition, common.DeriveExecutionID("Method", *fp.Method), nil

	case fp.Service != nil && *fp.Service != "":
		condition.ConditionType = &actionv2.Condition_Request{
			Request: &actionv2.RequestExecution{
				Condition: &actionv2.RequestExecution_Service{Service: *fp.Service},
			},
		}

		return condition, common.DeriveExecutionID("Service", *fp.Service), nil

	case fp.All != nil && *fp.All:
		condition.ConditionType = &actionv2.Condition_Request{
			Request: &actionv2.RequestExecution{
				Condition: &actionv2.RequestExecution_All{All: true},
			},
		}

		return condition, common.DeriveExecutionID("All", ""), nil

	default:
		return nil, "", errors.New("one of method, service or all must be set on the execution request")
	}
}

func (requestDriver) Targets(cr *v1alpha1.ActionExecutionRequest) []string {
	return cr.Spec.ForProvider.TargetIDs
}

func (requestDriver) Report(cr *v1alpha1.ActionExecutionRequest, targets []string) {
	cr.Status.AtProvider.TargetIDs = targets
}

// responseDriver binds targets to a gRPC response Zitadel is about to return.
type responseDriver struct {
	execution[*v1alpha1.ActionExecutionResponse]
}

var _ common.BindingDriver[*v1alpha1.ActionExecutionResponse, *actionv2.Condition] = responseDriver{}

func (responseDriver) Kind() string { return "ActionExecutionResponse" }

func (responseDriver) Condition(cr *v1alpha1.ActionExecutionResponse) (*actionv2.Condition, string, error) {
	fp := cr.Spec.ForProvider

	condition := &actionv2.Condition{}

	switch {
	case fp.Method != nil && *fp.Method != "":
		condition.ConditionType = &actionv2.Condition_Response{
			Response: &actionv2.ResponseExecution{
				Condition: &actionv2.ResponseExecution_Method{Method: *fp.Method},
			},
		}

		return condition, common.DeriveExecutionID("Method", *fp.Method), nil

	case fp.Service != nil && *fp.Service != "":
		condition.ConditionType = &actionv2.Condition_Response{
			Response: &actionv2.ResponseExecution{
				Condition: &actionv2.ResponseExecution_Service{Service: *fp.Service},
			},
		}

		return condition, common.DeriveExecutionID("Service", *fp.Service), nil

	case fp.All != nil && *fp.All:
		condition.ConditionType = &actionv2.Condition_Response{
			Response: &actionv2.ResponseExecution{
				Condition: &actionv2.ResponseExecution_All{All: true},
			},
		}

		return condition, common.DeriveExecutionID("All", ""), nil

	default:
		return nil, "", errors.New("one of method, service or all must be set on the execution response")
	}
}

func (responseDriver) Targets(cr *v1alpha1.ActionExecutionResponse) []string {
	return cr.Spec.ForProvider.TargetIDs
}

func (responseDriver) Report(cr *v1alpha1.ActionExecutionResponse, targets []string) {
	cr.Status.AtProvider.TargetIDs = targets
}

// eventDriver binds targets to a Zitadel event.
type eventDriver struct {
	execution[*v1alpha1.ActionExecutionEvent]
}

var _ common.BindingDriver[*v1alpha1.ActionExecutionEvent, *actionv2.Condition] = eventDriver{}

func (eventDriver) Kind() string { return "ActionExecutionEvent" }

func (eventDriver) Condition(cr *v1alpha1.ActionExecutionEvent) (*actionv2.Condition, string, error) {
	fp := cr.Spec.ForProvider

	condition := &actionv2.Condition{}

	switch {
	case fp.Event != nil && *fp.Event != "":
		condition.ConditionType = &actionv2.Condition_Event{
			Event: &actionv2.EventExecution{
				Condition: &actionv2.EventExecution_Event{Event: *fp.Event},
			},
		}

		return condition, common.DeriveExecutionID("Event", *fp.Event), nil

	case fp.Group != nil && *fp.Group != "":
		condition.ConditionType = &actionv2.Condition_Event{
			Event: &actionv2.EventExecution{
				Condition: &actionv2.EventExecution_Group{Group: *fp.Group},
			},
		}

		return condition, common.DeriveExecutionID("Group", *fp.Group), nil

	case fp.All != nil && *fp.All:
		condition.ConditionType = &actionv2.Condition_Event{
			Event: &actionv2.EventExecution{
				Condition: &actionv2.EventExecution_All{All: true},
			},
		}

		return condition, common.DeriveExecutionID("All", ""), nil

	default:
		return nil, "", errors.New("one of event, group or all must be set on the execution event")
	}
}

func (eventDriver) Targets(cr *v1alpha1.ActionExecutionEvent) []string {
	return cr.Spec.ForProvider.TargetIDs
}

func (eventDriver) Report(cr *v1alpha1.ActionExecutionEvent, targets []string) {
	cr.Status.AtProvider.TargetIDs = targets
}

// functionDriver binds targets to a function another action calls.
type functionDriver struct {
	execution[*v1alpha1.ActionExecutionFunction]
}

var _ common.BindingDriver[*v1alpha1.ActionExecutionFunction, *actionv2.Condition] = functionDriver{}

func (functionDriver) Kind() string { return "ActionExecutionFunction" }

func (functionDriver) Condition(cr *v1alpha1.ActionExecutionFunction) (*actionv2.Condition, string, error) {
	if cr.Spec.ForProvider.Name == nil || *cr.Spec.ForProvider.Name == "" {
		return nil, "", errors.New("name must be set on the execution function")
	}

	condition := &actionv2.Condition{
		ConditionType: &actionv2.Condition_Function{
			Function: &actionv2.FunctionExecution{Name: *cr.Spec.ForProvider.Name},
		},
	}

	return condition, common.DeriveExecutionID("Name", *cr.Spec.ForProvider.Name), nil
}

func (functionDriver) Targets(cr *v1alpha1.ActionExecutionFunction) []string {
	return cr.Spec.ForProvider.TargetIDs
}

func (functionDriver) Report(cr *v1alpha1.ActionExecutionFunction, targets []string) {
	cr.Status.AtProvider.TargetIDs = targets
}
