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

// IDPState is whether an identity provider is offered on the login page.
//
// +kubebuilder:validation:Enum=Active;Inactive
type IDPState string

const (
	// IDPStateActive means the provider is offered.
	IDPStateActive IDPState = "Active"

	// IDPStateInactive keeps the provider's settings and its user links but
	// stops it being offered, which is what scaling one back looks like.
	IDPStateInactive IDPState = "Inactive"
)

// IDPAutoLinking is how an account is matched to a Zitadel one without asking
// the user.
//
// +kubebuilder:validation:Enum=Email;Username
type IDPAutoLinking string

const (
	// IDPAutoLinkingEmail matches on the email address.
	IDPAutoLinkingEmail IDPAutoLinking = "Email"

	// IDPAutoLinkingUsername matches on the username.
	IDPAutoLinkingUsername IDPAutoLinking = "Username"
)

// AzureADTenant is which Microsoft Entra ID tenants may log in.
//
// +kubebuilder:validation:Enum=Common;Consumers;Organisations
type AzureADTenant string

const (
	// AzureADTenantCommon lets any Entra organization log in, which is what most
	// installations use.
	AzureADTenantCommon AzureADTenant = "Common"

	// AzureADTenantConsumers lets personal Microsoft accounts log in.
	AzureADTenantConsumers AzureADTenant = "Consumers"

	// AzureADTenantOrganisations is one tenant's directory. Set
	// `tenant` to a tenant ID instead to name it exactly.
	AzureADTenantOrganisations AzureADTenant = "Organisations"
)

// SAMLBinding is how a SAML authentication request is sent.
//
// +kubebuilder:validation:Enum=Redirect;POST
type SAMLBinding string

const (
	// SAMLBindingRedirect sends the request in the URL, which is the default and
	// what most identity providers expect.
	SAMLBindingRedirect SAMLBinding = "Redirect"

	// SAMLBindingPOST sends it in the body, for a provider that requires it.
	SAMLBindingPOST SAMLBinding = "POST"
)

// SAMLNameIDFormat is how a SAML identity provider identifies a user.
//
// +kubebuilder:validation:Enum=EmailAddress;Persistent;Transient
type SAMLNameIDFormat string

const (
	// SAMLNameIDFormatEmailAddress identifies a user by their email address.
	SAMLNameIDFormatEmailAddress SAMLNameIDFormat = "EmailAddress"

	// SAMLNameIDFormatPersistent identifies a user by an identifier that survives
	// the session.
	SAMLNameIDFormatPersistent SAMLNameIDFormat = "Persistent"

	// SAMLNameIDFormatTransient identifies a user only within the session.
	SAMLNameIDFormatTransient SAMLNameIDFormat = "Transient"
)

// SAMLSignatureAlgorithm is how a SAML assertion is signed.
//
// +kubebuilder:validation:Enum=RSA-SHA1;RSA-SHA256;RSA-SHA512
type SAMLSignatureAlgorithm string

const (
	// SAMLSignatureAlgorithmRSASHA1 is the weakest, and only for an identity
	// provider that offers nothing better.
	SAMLSignatureAlgorithmRSASHA1 SAMLSignatureAlgorithm = "RSA-SHA1"

	// SAMLSignatureAlgorithmRSASHA256 is the usual choice.
	SAMLSignatureAlgorithmRSASHA256 SAMLSignatureAlgorithm = "RSA-SHA256"

	// SAMLSignatureAlgorithmRSASHA512 is the strongest.
	SAMLSignatureAlgorithmRSASHA512 SAMLSignatureAlgorithm = "RSA-SHA512"
)

// SecretKeySelector is a reference to a key held in a secret.
//
// Secrets are read rather than written: a client secret, an LDAP bind password
// or a SAML signing key has no safe place in a custom resource, and Zitadel
// never returns one either, so nothing here is ever compared.
//
// +kubebuilder:validation:Required
type SecretKeySelector struct {
	// Name of the secret.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +required
	Name string `json:"name"`

	// Key inside the secret.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Required
	// +required
	Key string `json:"key"`
}

// IDPObservation is what Zitadel reports about an identity provider.
//
// Every provider kind reports the same things, and reports a configuration only
// for an OIDC or a JWT provider, so all twenty three kinds share this one type.
// The fields that stay empty for the other kinds are exactly the ones Zitadel
// does not read back - which is also why their spec fields are applied once and
// never compared.
type IDPObservation struct {
	// ID is Zitadel's identifier of the provider.
	// +optional
	ID string `json:"id,omitempty"`

	// Name of the provider.
	// +optional
	Name *string `json:"name,omitempty"`

	// State is whether the provider is offered on the login page.
	// +optional
	State *string `json:"state,omitempty"`

	// IsAutoCreation is whether a user is created without asking on first login.
	// +optional
	IsAutoCreation *bool `json:"isAutoCreation,omitempty"`

	// OrganizationID is the organization the provider belongs to. Empty for an
	// instance wide provider.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// Scope is the organization the provider was last read from.
	// +optional
	Scope *string `json:"scope,omitempty"`

	// Issuer, ClientID and Scopes are reported for an OIDC provider.
	// +optional
	Issuer *string `json:"issuer,omitempty"`

	// +optional
	ClientID *string `json:"clientID,omitempty"`

	// +optional
	Scopes []string `json:"scopes,omitempty"`

	// JWTEndpoint, KeysEndpoint and HeaderName are reported for a JWT provider.
	// +optional
	JWTEndpoint *string `json:"jwtEndpoint,omitempty"`

	// +optional
	KeysEndpoint *string `json:"keysEndpoint,omitempty"`

	// +optional
	HeaderName *string `json:"headerName,omitempty"`
}
