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

package zitadel

import (
	"context"
	"fmt"
	"time"

	actionv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/action/v2"
	filterv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/filter/v2"
	managementv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
)

// Actions
//
// A Zitadel action is a JavaScript snippet Zitadel runs somewhere in a login
// flow. It belongs to an organization, and the scripts themselves are managed
// through the v1 management API - the v2 action service manages where actions
// are sent, not what they do.

// Action is an organization action.
type Action struct {
	ID            string
	State         string
	Name          string
	Script        string
	Timeout       string
	AllowedToFail bool
}

// ActionInput is the desired action.
type ActionInput struct {
	Name          string
	Script        string
	Timeout       string
	AllowedToFail bool
}

// GetAction returns one action of an organization.
func (c *Client) GetAction(ctx context.Context, orgID, actionID string) (*Action, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	resp, err := mc.GetAction(ctx, &managementv1.GetActionRequest{Id: actionID})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get action %s of organization %s: %w", actionID, orgID, err)
	}

	a := resp.GetAction()
	if a == nil {
		return nil, ErrNotFound
	}

	return &Action{
		ID:            a.GetId(),
		State:         a.GetState().String(),
		Name:          a.GetName(),
		Script:        a.GetScript(),
		Timeout:       durationString(a.GetTimeout()),
		AllowedToFail: a.GetAllowedToFail(),
	}, nil
}

// CreateAction adds an action to an organization.
func (c *Client) CreateAction(ctx context.Context, orgID string, in ActionInput) (string, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return "", err
	}

	if _, err := time.ParseDuration(in.Timeout); err != nil {
		return "", fmt.Errorf("cannot parse the action timeout %q: %w", in.Timeout, err)
	}

	resp, err := mc.CreateAction(ctx, &managementv1.CreateActionRequest{
		Name:          in.Name,
		Script:        in.Script,
		Timeout:       durationProto(in.Timeout),
		AllowedToFail: in.AllowedToFail,
	})
	if err != nil {
		return "", fmt.Errorf("cannot create the action in organization %s: %w", orgID, err)
	}

	return resp.GetId(), nil
}

// UpdateAction changes an action's script, timeout and behaviour.
func (c *Client) UpdateAction(ctx context.Context, orgID, actionID string, in ActionInput) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := time.ParseDuration(in.Timeout); err != nil {
		return fmt.Errorf("cannot parse the action timeout %q: %w", in.Timeout, err)
	}

	if _, err := mc.UpdateAction(ctx, &managementv1.UpdateActionRequest{
		Id:            actionID,
		Name:          in.Name,
		Script:        in.Script,
		Timeout:       durationProto(in.Timeout),
		AllowedToFail: in.AllowedToFail,
	}); err != nil && !IsNotChanged(err) {
		return fmt.Errorf("cannot update action %s of organization %s: %w", actionID, orgID, err)
	}

	return nil
}

// DeleteAction removes an action from an organization.
//
// Zitadel refuses to delete a script that an execution still refers to, which
// is deliberate: an action that something triggers must be unbound from that
// trigger first.
func (c *Client) DeleteAction(ctx context.Context, orgID, actionID string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	if _, err := mc.DeleteAction(ctx, &managementv1.DeleteActionRequest{Id: actionID}); err != nil {
		if IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("cannot delete action %s of organization %s: %w", actionID, orgID, err)
	}

	return nil
}

// SetActionState activates or deactivates an action without changing it.
//
// A deactivated action keeps its script and its bindings but stops running,
// which is what a resource that has been scaled back should look like.
func (c *Client) SetActionState(ctx context.Context, orgID, actionID, state string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	switch state {
	case "Active":
		_, err = mc.ReactivateAction(ctx, &managementv1.ReactivateActionRequest{Id: actionID})

	case "Inactive":
		_, err = mc.DeactivateAction(ctx, &managementv1.DeactivateActionRequest{Id: actionID})

	default:
		return fmt.Errorf("unknown action state %q: must be Active or Inactive", state)
	}

	if err != nil && !IsNotFound(err) && !IsNotChanged(err) {
		return fmt.Errorf("cannot set action %s of organization %s to %s: %w", actionID, orgID, state, err)
	}

	return nil
}

// ActionTarget is somewhere an action's payload is sent: a webhook, or Zitadel
// itself over HTTP.
type ActionTarget struct {
	ID          string
	Name        string
	Type        string
	Endpoint    string
	Timeout     string
	PayloadType string

	// SigningKey is generated by Zitadel when the target is created. It is what
	// the target signs its payloads with, and the resource republishes it so an
	// operator can configure the receiving side to verify them.
	SigningKey string
}

// CreateActionTarget adds a target.
func (c *Client) CreateActionTarget(ctx context.Context, in ActionTarget) (string, error) {
	req, err := createTargetRequest(in)
	if err != nil {
		return "", err
	}

	resp, err := c.action.CreateTarget(ctx, req)
	if err != nil {
		return "", fmt.Errorf("cannot create the action target: %w", err)
	}

	return resp.GetId(), nil
}

// GetActionTarget returns one target.
func (c *Client) GetActionTarget(ctx context.Context, targetID string) (*ActionTarget, error) {
	resp, err := c.action.GetTarget(ctx, &actionv2.GetTargetRequest{Id: targetID})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get action target %s: %w", targetID, err)
	}

	t := resp.GetTarget()
	if t == nil {
		return nil, ErrNotFound
	}

	return &ActionTarget{
		ID:          t.GetId(),
		Name:        t.GetName(),
		Type:        actionTargetTypeFromProto(t.GetTargetType()),
		Endpoint:    t.GetEndpoint(),
		Timeout:     durationString(t.GetTimeout()),
		PayloadType: actionPayloadTypeFromProto(t.GetPayloadType()),
		SigningKey:  t.GetSigningKey(),
	}, nil
}

// UpdateActionTarget changes a target's endpoint, timeout and behaviour.
func (c *Client) UpdateActionTarget(ctx context.Context, targetID string, in ActionTarget) error {
	req, err := updateTargetRequest(targetID, in)
	if err != nil {
		return err
	}

	if _, err := c.action.UpdateTarget(ctx, req); err != nil {
		return fmt.Errorf("cannot update action target %s: %w", targetID, err)
	}

	return nil
}

// DeleteActionTarget removes a target.
//
// Zitadel refuses to remove a target that an execution still calls, for the same
// reason it refuses to delete a bound action: the binding has to go first.
func (c *Client) DeleteActionTarget(ctx context.Context, targetID string) error {
	if _, err := c.action.DeleteTarget(ctx, &actionv2.DeleteTargetRequest{Id: targetID}); err != nil {
		if IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("cannot delete action target %s: %w", targetID, err)
	}

	return nil
}

// ActionTargetPublicKey is a key a target's payloads are encrypted with.
type ActionTargetPublicKey struct {
	KeyID        string
	Active       bool
	PublicKey    string
	Fingerprint  string
	CreationDate string
}

// AddActionTargetPublicKey adds a public key to a target.
//
// Zitadel returns only the key's identifier; the rest is read back from the
// target's key list, which is the only place the fingerprint is reported.
func (c *Client) AddActionTargetPublicKey(ctx context.Context, targetID, publicKey string) (string, error) {
	resp, err := c.action.AddPublicKey(ctx, &actionv2.AddPublicKeyRequest{
		TargetId:  targetID,
		PublicKey: []byte(publicKey),
	})
	if err != nil {
		return "", fmt.Errorf("cannot add a public key to action target %s: %w", targetID, err)
	}

	return resp.GetKeyId(), nil
}

// GetActionTargetPublicKey returns one public key of a target.
func (c *Client) GetActionTargetPublicKey(ctx context.Context, targetID, keyID string) (*ActionTargetPublicKey, error) {
	resp, err := c.action.ListPublicKeys(ctx, &actionv2.ListPublicKeysRequest{
		TargetId:   targetID,
		Pagination: &filterv2.PaginationRequest{Limit: 1000},
	})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot list the public keys of action target %s: %w", targetID, err)
	}

	for _, k := range resp.GetPublicKeys() {
		if k.GetKeyId() != keyID {
			continue
		}

		out := &ActionTargetPublicKey{
			KeyID:       k.GetKeyId(),
			Active:      k.GetActive(),
			PublicKey:   string(k.GetPublicKey()),
			Fingerprint: k.GetFingerprint(),
		}

		if d := k.GetCreationDate(); d.IsValid() {
			out.CreationDate = d.AsTime().Format(time.RFC3339)
		}

		return out, nil
	}

	return nil, ErrNotFound
}

// SetActionTargetPublicKeyActive activates or deactivates a target's key.
//
// An inactive key stays on the target but is no longer used to encrypt, which is
// what rotating a key looks like before the old one is removed.
func (c *Client) SetActionTargetPublicKeyActive(ctx context.Context, targetID, keyID string, active bool) error {
	var err error

	if active {
		_, err = c.action.ActivatePublicKey(ctx, &actionv2.ActivatePublicKeyRequest{TargetId: targetID, KeyId: keyID})
	} else {
		_, err = c.action.DeactivatePublicKey(ctx, &actionv2.DeactivatePublicKeyRequest{TargetId: targetID, KeyId: keyID})
	}

	if err != nil && !IsNotFound(err) && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the public key %s of action target %s active=%t: %w", keyID, targetID, active, err)
	}

	return nil
}

// RemoveActionTargetPublicKey removes a key from a target.
func (c *Client) RemoveActionTargetPublicKey(ctx context.Context, targetID, keyID string) error {
	if _, err := c.action.RemovePublicKey(ctx, &actionv2.RemovePublicKeyRequest{
		TargetId: targetID,
		KeyId:    keyID,
	}); err != nil {
		if IsNotFound(err) {
			return nil
		}

		return fmt.Errorf("cannot remove the public key %s of action target %s: %w", keyID, targetID, err)
	}

	return nil
}

// Execution is a binding: when a condition holds, call these targets.
//
// Zitadel gives an execution no identifier of its own. The only way to find one
// is to list them all and match the condition, and the only way to change one is
// to write the whole binding again. So an execution managed resource carries its
// condition in the spec and derives its own identity from it.
type Execution struct {
	Condition    *actionv2.Condition
	Targets      []string
	CreationDate string
	ChangeDate   string
}

// SetExecutions writes the bindings of the given conditions.
//
// Writing is unconditional: a condition that already has the same targets is
// written again, because there is no other way to express "these targets, and
// nothing else".
func (c *Client) SetExecutions(ctx context.Context, executions []*actionv2.Condition, targets []string) error {
	if _, err := c.action.SetExecution(ctx, &actionv2.SetExecutionRequest{
		Condition: conditionOrNil(executions),
		Targets:   targets,
	}); err != nil {
		return fmt.Errorf("cannot set the action execution: %w", err)
	}

	return nil
}

// ListExecutions returns every binding in the instance.
func (c *Client) ListExecutions(ctx context.Context) ([]Execution, error) {
	resp, err := c.action.ListExecutions(ctx, &actionv2.ListExecutionsRequest{
		Pagination: &filterv2.PaginationRequest{Limit: 1000},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the action executions: %w", err)
	}

	out := make([]Execution, 0, len(resp.GetExecutions()))
	for _, e := range resp.GetExecutions() {
		x := Execution{Condition: e.GetCondition(), Targets: e.GetTargets()}

		if d := e.GetCreationDate(); d.IsValid() {
			x.CreationDate = d.AsTime().Format(time.RFC3339)
		}

		if d := e.GetChangeDate(); d.IsValid() {
			x.ChangeDate = d.AsTime().Format(time.RFC3339)
		}

		out = append(out, x)
	}

	return out, nil
}

// SetTriggerActions binds actions to a point in a login flow.
func (c *Client) SetTriggerActions(ctx context.Context, orgID, flowType, triggerType string, actionIDs []string) error {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return err
	}

	// Zitadel names these enums by their constant names, so the API spelling is
	// translated here rather than by every caller.
	flow, err := FlowTypeToProto(flowType)
	if err != nil {
		return err
	}

	point, err := TriggerTypeToProto(triggerType)
	if err != nil {
		return err
	}

	if _, err := mc.SetTriggerActions(ctx, &managementv1.SetTriggerActionsRequest{
		FlowType:    flow,
		TriggerType: point,
		ActionIds:   actionIDs,
	}); err != nil && !IsNotFound(err) && !IsNotChanged(err) {
		return fmt.Errorf("cannot set the %s trigger actions of organization %s: %w", triggerType, orgID, err)
	}

	return nil
}

// GetTriggerActions returns the actions bound to a point in a login flow.
func (c *Client) GetTriggerActions(ctx context.Context, orgID, flowType, triggerType string) ([]string, error) {
	mc, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	flow, err := FlowTypeToProto(flowType)
	if err != nil {
		return nil, err
	}

	resp, err := mc.GetFlow(ctx, &managementv1.GetFlowRequest{Type: flow})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("cannot get the %s flow of organization %s: %w", flowType, orgID, err)
	}

	// Zitadel has no call for the trigger actions on their own, so they are read
	// out of the flow and the wanted trigger picked from it. A flow with no such
	// trigger has nothing bound, which is not an error.
	for _, t := range resp.GetFlow().GetTriggerActions() {
		if TriggerTypeFromProto(t.GetTriggerType().String()) != triggerType {
			continue
		}

		out := make([]string, 0, len(t.GetActions()))
		for _, a := range t.GetActions() {
			out = append(out, a.GetId())
		}

		return out, nil
	}

	return nil, nil
}

// actionState and the rest of this file's enum helpers keep the generated enum
// names out of the rest of the package.

// The kind of target - a webhook, a call, or an asynchronous call - is a oneof
// whose type is unexported by the generated SDK, so these helpers build the
// whole request rather than returning the oneof value.

func createTargetRequest(in ActionTarget) (*actionv2.CreateTargetRequest, error) {
	payload, err := actionPayloadTypeToProto(in.PayloadType)
	if err != nil {
		return nil, err
	}

	if _, err := time.ParseDuration(in.Timeout); err != nil {
		return nil, fmt.Errorf("cannot parse the target timeout %q: %w", in.Timeout, err)
	}

	req := &actionv2.CreateTargetRequest{
		Name:        in.Name,
		Endpoint:    in.Endpoint,
		Timeout:     durationProto(in.Timeout),
		PayloadType: payload,
	}

	switch in.Type {
	case "Webhook":
		req.TargetType = &actionv2.CreateTargetRequest_RestWebhook{RestWebhook: &actionv2.RESTWebhook{}}

	case "Call":
		req.TargetType = &actionv2.CreateTargetRequest_RestCall{RestCall: &actionv2.RESTCall{}}

	case "Async":
		req.TargetType = &actionv2.CreateTargetRequest_RestAsync{RestAsync: &actionv2.RESTAsync{}}

	default:
		return nil, fmt.Errorf("unknown target type %q: must be Webhook, Call or Async", in.Type)
	}

	return req, nil
}

func updateTargetRequest(targetID string, in ActionTarget) (*actionv2.UpdateTargetRequest, error) {
	payload, err := actionPayloadTypeToProto(in.PayloadType)
	if err != nil {
		return nil, err
	}

	if _, err := time.ParseDuration(in.Timeout); err != nil {
		return nil, fmt.Errorf("cannot parse the target timeout %q: %w", in.Timeout, err)
	}

	name, endpoint := in.Name, in.Endpoint

	// A target signs its payloads, and Zitadel replaces its signing key only
	// when told to expire the current one. Leaving the expiry unset keeps the
	// key, which is what an ordinary change should do.
	req := &actionv2.UpdateTargetRequest{
		Id:                   targetID,
		Name:                 &name,
		Endpoint:             &endpoint,
		Timeout:              durationProto(in.Timeout),
		PayloadType:          payload,
		ExpirationSigningKey: nil,
	}

	switch in.Type {
	case "Webhook":
		req.TargetType = &actionv2.UpdateTargetRequest_RestWebhook{RestWebhook: &actionv2.RESTWebhook{}}

	case "Call":
		req.TargetType = &actionv2.UpdateTargetRequest_RestCall{RestCall: &actionv2.RESTCall{}}

	case "Async":
		req.TargetType = &actionv2.UpdateTargetRequest_RestAsync{RestAsync: &actionv2.RESTAsync{}}

	default:
		return nil, fmt.Errorf("unknown target type %q: must be Webhook, Call or Async", in.Type)
	}

	return req, nil
}

func actionTargetTypeFromProto(t any) string {
	switch t.(type) {
	case *actionv2.Target_RestCall:
		return "Call"

	case *actionv2.Target_RestAsync:
		return "Async"

	default:
		return "Webhook"
	}
}

func actionPayloadTypeToProto(p string) (actionv2.PayloadType, error) {
	switch p {
	case "", "Json":
		return actionv2.PayloadType_PAYLOAD_TYPE_JSON, nil

	case "Jwt":
		return actionv2.PayloadType_PAYLOAD_TYPE_JWT, nil

	case "Jwe":
		// An encrypted payload is signed with the target's active public key, so
		// the key has to exist before a target can use this.
		return actionv2.PayloadType_PAYLOAD_TYPE_JWE, nil

	default:
		return 0, fmt.Errorf("unknown payload type %q: must be Json, Jwt or Jwe", p)
	}
}

func actionPayloadTypeFromProto(p actionv2.PayloadType) string {
	switch p {
	case actionv2.PayloadType_PAYLOAD_TYPE_JWT:
		return "Jwt"

	case actionv2.PayloadType_PAYLOAD_TYPE_JWE:
		return "Jwe"

	// An unset payload type is sent as JSON, which is what Zitadel does with
	// it, so it reads back as Json rather than as a value of its own.
	case actionv2.PayloadType_PAYLOAD_TYPE_JSON, actionv2.PayloadType_PAYLOAD_TYPE_UNSPECIFIED:
		return "Json"

	default:
		return "Json"
	}
}

func conditionOrNil(executions []*actionv2.Condition) *actionv2.Condition {
	if len(executions) == 0 {
		return nil
	}

	return executions[0]
}

// The login flow and trigger types are protobuf enums.
//
// Zitadel's trigger API takes them as the *number* rendered as a string, which
// is neither the Go constant name nor the API spelling, so it is translated in
// both directions here rather than by every caller.

// FlowTypeToProto maps a login flow onto the constant name Zitadel expects.
func FlowTypeToProto(flow string) (string, error) {
	switch flow {
	case "ExternalAuthentication":
		return "1", nil

	case "CustomiseToken":
		return "2", nil

	case "InternalAuthentication":
		return "3", nil

	case "SAMLResponse":
		return "4", nil

	default:
		return "", fmt.Errorf("unknown login flow %q", flow)
	}
}

// TriggerTypeToProto maps a trigger point onto the constant name Zitadel
// expects.
func TriggerTypeToProto(trigger string) (string, error) {
	switch trigger {
	case "PostAuthentication":
		return "1", nil

	case "PreCreation":
		return "2", nil

	case "PostCreation":
		return "3", nil

	case "PreUserinfoCreation":
		return "4", nil

	case "PreAccessTokenCreation":
		return "5", nil

	case "PreSAMLResponseCreation":
		return "6", nil

	default:
		return "", fmt.Errorf("unknown trigger %q", trigger)
	}
}

// TriggerTypeFromProto maps a trigger constant name back onto the API spelling.
func TriggerTypeFromProto(trigger string) string {
	switch trigger {
	case "TRIGGER_TYPE_PRE_CREATION":
		return "PreCreation"

	case "TRIGGER_TYPE_POST_CREATION":
		return "PostCreation"

	case "TRIGGER_TYPE_PRE_USERINFO_CREATION":
		return "PreUserinfoCreation"

	case "TRIGGER_TYPE_PRE_ACCESS_TOKEN_CREATION":
		return "PreAccessTokenCreation"

	case "TRIGGER_TYPE_PRE_SAML_RESPONSE_CREATION":
		return "PreSAMLResponseCreation"

	default:
		return "PostAuthentication"
	}
}
