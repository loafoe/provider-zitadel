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

package v1alpha1

// ActionState is whether an action runs.
//
// +kubebuilder:validation:Enum=Active;Inactive
type ActionState string

const (
	// ActionStateActive means the action runs wherever it is bound.
	ActionStateActive ActionState = "Active"

	// ActionStateInactive keeps the action's script and its bindings but stops it
	// from running.
	ActionStateInactive ActionState = "Inactive"
)

// ActionTargetType is how Zitadel calls a target.
//
// +kubebuilder:validation:Enum=Webhook;Call;Async
type ActionTargetType string

const (
	// ActionTargetTypeWebhook POSTs the payload and checks only the status code.
	ActionTargetTypeWebhook ActionTargetType = "Webhook"

	// ActionTargetTypeCall sends the request and uses the response as the next
	// step of the flow.
	ActionTargetTypeCall ActionTargetType = "Call"

	// ActionTargetTypeAsync sends the request and does not wait for it.
	ActionTargetTypeAsync ActionTargetType = "Async"
)

// ActionPayloadType is how a target's payload is encoded.
//
// +kubebuilder:validation:Enum=Json;Jwt;Jwe
type ActionPayloadType string

const (
	// ActionPayloadTypeJson sends the payload as JSON in the request body.
	ActionPayloadTypeJson ActionPayloadType = "Json"

	// ActionPayloadTypeJwt sends the payload as a signed JSON web token.
	ActionPayloadTypeJwt ActionPayloadType = "Jwt"

	// ActionPayloadTypeJwe encrypts the payload to the target's active public
	// key, so the target needs an ActionTargetPublicKey first.
	ActionPayloadTypeJwe ActionPayloadType = "Jwe"
)

// TriggerFlowType is the login flow a trigger belongs to.
//
// +kubebuilder:validation:Enum=ExternalAuthentication;CustomiseToken;InternalAuthentication;SAMLResponse
type TriggerFlowType string

const (
	// TriggerFlowTypeExternalAuthentication is a user logging in through an
	// upstream identity provider.
	TriggerFlowTypeExternalAuthentication TriggerFlowType = "ExternalAuthentication"

	// TriggerFlowTypeCustomiseToken runs while a token is being issued.
	TriggerFlowTypeCustomiseToken TriggerFlowType = "CustomiseToken"

	// TriggerFlowTypeInternalAuthentication is a user logging in against
	// Zitadel's own user store.
	TriggerFlowTypeInternalAuthentication TriggerFlowType = "InternalAuthentication"

	// TriggerFlowTypeSAMLResponse runs while a SAML response is being built.
	TriggerFlowTypeSAMLResponse TriggerFlowType = "SAMLResponse"
)

// TriggerType is where in a login flow a trigger runs.
//
// +kubebuilder:validation:Enum=PostAuthentication;PreCreation;PostCreation;PreUserinfoCreation;PreAccessTokenCreation;PreSAMLResponseCreation
type TriggerType string

const (
	// TriggerTypePostAuthentication runs once a user is authenticated.
	TriggerTypePostAuthentication TriggerType = "PostAuthentication"

	// TriggerTypePreCreation runs before a user is created.
	TriggerTypePreCreation TriggerType = "PreCreation"

	// TriggerTypePostCreation runs after a user is created.
	TriggerTypePostCreation TriggerType = "PostCreation"

	// TriggerTypePreUserinfoCreation runs before a userinfo is built.
	TriggerTypePreUserinfoCreation TriggerType = "PreUserinfoCreation"

	// TriggerTypePreAccessTokenCreation runs before an access token is issued.
	TriggerTypePreAccessTokenCreation TriggerType = "PreAccessTokenCreation"

	// TriggerTypePreSAMLResponseCreation runs before a SAML response is built.
	TriggerTypePreSAMLResponseCreation TriggerType = "PreSAMLResponseCreation"
)
