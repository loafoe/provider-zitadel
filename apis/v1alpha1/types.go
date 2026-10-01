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

// Package v1alpha1 contains the core resources of the Zitadel provider.
// +kubebuilder:object:generate=true
// +groupName=zitadel.crossplane.io
// +versionName=v1alpha1
package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AuthType specifies how the provider authenticates against a Zitadel
// instance. Both supported types are Zitadel credentials belonging to a
// machine user (also referred to as a service account).
// +kubebuilder:validation:Enum=ServiceAccount;Token
type AuthType string

const (
	// AuthTypeServiceAccount uses a machine user's JSON key ("key file") as
	// downloaded from the Zitadel console. The key is exchanged for a short
	// lived access token using the JWT Profile grant.
	//
	// The machine user must be configured with the "Jwt" access token type,
	// because the Zitadel API only accepts signed JWTs. A machine user with
	// the default "Bearer" token type issues encrypted tokens which the API
	// rejects with `Unauthenticated: Errors.Token.Invalid`.
	AuthTypeServiceAccount AuthType = "ServiceAccount"

	// AuthTypeToken uses a Personal Access Token belonging to a service
	// account (machine user) as a static bearer token. The token never
	// expires, so it should be scoped to a dedicated, least privileged service
	// account.
	AuthTypeToken AuthType = "Token"
)

// ServiceAccountAuth holds the credentials of a Zitadel service account,
// i.e. a machine user together with one of its machine keys.
type ServiceAccountAuth struct {
	// KeySecretRef references the secret key holding the JSON encoded
	// machine key of a Zitadel service account, i.e. the key file downloaded
	// from the Zitadel console:
	// `{"type":"serviceaccount","keyId":"...","key":"-----BEGIN RSA PRIVATE KEY-----..."}`.
	// The service account must use the "Jwt" access token type.
	// +kubebuilder:validation:Required
	KeySecretRef xpv1.LocalSecretKeySelector `json:"keySecretRef"`
}

// TokenAuth holds a Zitadel Personal Access Token (PAT).
type TokenAuth struct {
	// TokenSecretRef references the secret key holding the Personal Access
	// Token of a Zitadel service account.
	// +kubebuilder:validation:Required
	TokenSecretRef xpv1.LocalSecretKeySelector `json:"tokenSecretRef"`
}

// ZitadelCredentials contains the authentication configuration used to talk to
// a Zitadel instance.
type ZitadelCredentials struct {
	// Source of the provider credentials.
	// +kubebuilder:validation:Enum=None;Secret
	// +kubebuilder:default=Secret
	Source xpv1.CredentialsSource `json:"source"`

	// AuthType specifies which Zitadel service account credential is used.
	// +kubebuilder:validation:Required
	AuthType AuthType `json:"authType"`

	// ServiceAccount holds the machine key of a Zitadel service account.
	// Required when authType is "ServiceAccount".
	// +optional
	ServiceAccount *ServiceAccountAuth `json:"serviceAccount,omitempty"`

	// Token holds the Personal Access Token of a Zitadel service account.
	// Required when authType is "Token".
	// +optional
	Token *TokenAuth `json:"token,omitempty"`
}

// ProviderConfigSpec defines the configuration for connecting to Zitadel.
type ProviderConfigSpec struct {
	// URL is the base URL of the Zitadel instance, i.e. the issuer of the
	// instance. For example `https://my-instance.zitadel.cloud`. The gRPC API
	// endpoint is derived from it (`<host>:443`, or `<host>:80` when insecure
	// is enabled).
	// +kubebuilder:validation:Required
	URL string `json:"url"`
	// Insecure disables TLS when talking to the Zitadel API. Only useful for
	// local development instances without TLS.
	// +optional
	// +default=false
	Insecure *bool `json:"insecure,omitempty"`

	// InsecureSkipTLSVerify skips verification of the Zitadel server
	// certificate. Only useful for instances using a self signed certificate.
	// +optional
	// +default=false
	InsecureSkipTLSVerify *bool `json:"insecureSkipTLSVerify,omitempty"`

	// OrganizationID sets the organization context (`x-zitadel-orgid`) used for
	// all API calls that do not carry an explicit organization ID. Resources
	// that set `spec.forProvider.organizationID` (or a reference to an
	// Organization) always win.
	// +optional
	OrganizationID *string `json:"organizationID,omitempty"`

	// Credentials contains the authentication configuration for Zitadel.
	// +kubebuilder:validation:Required
	Credentials ZitadelCredentials `json:"credentials"`
}

// A ProviderConfigStatus defines the status of a Provider.
type ProviderConfigStatus struct {
	xpv1.ProviderConfigStatus `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="URL",type="string",JSONPath=".spec.url"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,zitadel}
// A ProviderConfig configures a Zitadel provider.
type ProviderConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProviderConfigSpec   `json:"spec"`
	Status ProviderConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProviderConfigList contains a list of ProviderConfig
type ProviderConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfig `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="CONFIG-NAME",type="string",JSONPath=".providerConfigRef.name"
// +kubebuilder:printcolumn:name="RESOURCE-KIND",type="string",JSONPath=".resourceRef.kind"
// +kubebuilder:printcolumn:name="RESOURCE-NAME",type="string",JSONPath=".resourceRef.name"
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,provider,zitadel}
// A ProviderConfigUsage indicates that a resource is using a ProviderConfig.
type ProviderConfigUsage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	xpv2.TypedProviderConfigUsage `json:",inline"`
}

// +kubebuilder:object:root=true

// ProviderConfigUsageList contains a list of ProviderConfigUsage
type ProviderConfigUsageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ProviderConfigUsage `json:"items"`
}
