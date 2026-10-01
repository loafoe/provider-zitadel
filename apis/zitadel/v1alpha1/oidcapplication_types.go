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

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ApplicationState is the lifecycle state of an Application.
// +kubebuilder:validation:Enum=Active;Inactive;Removed
type ApplicationState string

const (
	ApplicationStateActive   ApplicationState = "Active"
	ApplicationStateInactive ApplicationState = "Inactive"
	ApplicationStateRemoved  ApplicationState = "Removed"
)

// OIDCApplicationType describes the kind of client an OIDC application is
// used by.
// +kubebuilder:validation:Enum=Web;UserAgent;Native
type OIDCApplicationType string

const (
	OIDCApplicationTypeWeb       OIDCApplicationType = "Web"
	OIDCApplicationTypeUserAgent OIDCApplicationType = "UserAgent"
	OIDCApplicationTypeNative    OIDCApplicationType = "Native"
)

// OIDCAuthMethodType is the client authentication method of an OIDC
// application.
// +kubebuilder:validation:Enum=Basic;Post;None;PrivateKeyJwt
type OIDCAuthMethodType string

const (
	OIDCAuthMethodTypeBasic         OIDCAuthMethodType = "Basic"
	OIDCAuthMethodTypePost          OIDCAuthMethodType = "Post"
	OIDCAuthMethodTypeNone          OIDCAuthMethodType = "None"
	OIDCAuthMethodTypePrivateKeyJwt OIDCAuthMethodType = "PrivateKeyJwt"
)

// OIDCResponseType is a response type returned by an OIDC application.
// +kubebuilder:validation:Enum=Code;IdToken;IdTokenToken
type OIDCResponseType string

const (
	OIDCResponseTypeCode         OIDCResponseType = "Code"
	OIDCResponseTypeIdToken      OIDCResponseType = "IdToken"
	OIDCResponseTypeIdTokenToken OIDCResponseType = "IdTokenToken"
)

// OIDCGrantType is an OAuth2 grant supported by an OIDC application.
// +kubebuilder:validation:Enum=AuthorizationCode;Implicit;RefreshToken;DeviceCode;TokenExchange
type OIDCGrantType string

const (
	OIDCGrantTypeAuthorizationCode OIDCGrantType = "AuthorizationCode"
	OIDCGrantTypeImplicit          OIDCGrantType = "Implicit"
	OIDCGrantTypeRefreshToken      OIDCGrantType = "RefreshToken"
	OIDCGrantTypeDeviceCode        OIDCGrantType = "DeviceCode"
	OIDCGrantTypeTokenExchange     OIDCGrantType = "TokenExchange"
)

// OIDCTokenType is the format of the access tokens issued by an application.
// +kubebuilder:validation:Enum=Bearer;Jwt
type OIDCTokenType string

const (
	OIDCTokenTypeBearer OIDCTokenType = "Bearer"
	OIDCTokenTypeJwt    OIDCTokenType = "Jwt"
)

// OIDCVersion is the OIDC specification version implemented by the
// application.
// +kubebuilder:validation:Enum="1.0"
type OIDCVersion string

const OIDCVersion10 OIDCVersion = "1.0"

// OIDCApplicationParameters are the configurable fields of an OIDCApplication.
type OIDCApplicationParameters struct {
	// Name is the display name of the application.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// ProjectID is the ID of the project the application belongs to. Either
	// `projectID`, `projectRef` or `projectSelector` must be set.
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// ProjectRef references a Project managed by this provider and uses its ID.
	// +optional
	ProjectRef *xpv1.Reference `json:"projectRef,omitempty"`

	// ProjectSelector selects a Project managed by this provider and uses its
	// ID.
	// +optional
	ProjectSelector *xpv1.Selector `json:"projectSelector,omitempty"`

	// ID allows setting a custom application ID. If omitted Zitadel generates
	// one. It cannot be changed after creation.
	// +optional
	ID *string `json:"id,omitempty"`

	// RedirectURIs are the URIs the authorization server may redirect to.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	RedirectURIs []string `json:"redirectURIs"`

	// PostLogoutRedirectURIs are the URIs the user may be redirected to after
	// logging out.
	// +optional
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs,omitempty"`

	// ResponseTypes the application may request. Defaults to [Code].
	// +optional
	ResponseTypes []OIDCResponseType `json:"responseTypes,omitempty"`

	// GrantTypes supported by the application. Defaults to
	// [AuthorizationCode, RefreshToken].
	// +optional
	GrantTypes []OIDCGrantType `json:"grantTypes,omitempty"`

	// ApplicationType describes the kind of client. Defaults to Web.
	// +optional
	ApplicationType *OIDCApplicationType `json:"applicationType,omitempty"`

	// AuthMethodType is the client authentication method. Defaults to Basic.
	// +optional
	AuthMethodType *OIDCAuthMethodType `json:"authMethodType,omitempty"`

	// Version is the OIDC version. Defaults to "1.0".
	// +optional
	Version *OIDCVersion `json:"version,omitempty"`

	// DevelopmentMode relaxes some security checks and allows, for example,
	// http redirect URIs on web applications. Never enable it in production.
	// +optional
	DevelopmentMode *bool `json:"developmentMode,omitempty"`

	// AccessTokenType is the format of the issued access tokens. Defaults to
	// Bearer.
	// +optional
	AccessTokenType *OIDCTokenType `json:"accessTokenType,omitempty"`

	// AccessTokenRoleAssertion adds the roles of the user to the access token.
	// +optional
	AccessTokenRoleAssertion *bool `json:"accessTokenRoleAssertion,omitempty"`

	// IDTokenRoleAssertion adds the roles of the user to the ID token.
	// +optional
	IDTokenRoleAssertion *bool `json:"idTokenRoleAssertion,omitempty"`

	// IDTokenUserinfoAssertion adds the userinfo claims to the ID token.
	// +optional
	IDTokenUserinfoAssertion *bool `json:"idTokenUserinfoAssertion,omitempty"`

	// ClockSkew is the amount of leeway granted when validating timestamps,
	// e.g. "5s".
	// +optional
	ClockSkew *string `json:"clockSkew,omitempty"`

	// AdditionalOrigins are additional allowed CORS origins.
	// +optional
	AdditionalOrigins []string `json:"additionalOrigins,omitempty"`

	// SkipNativeAppSuccessPage skips the success page shown after native app
	// logins.
	// +optional
	SkipNativeAppSuccessPage *bool `json:"skipNativeAppSuccessPage,omitempty"`

	// BackChannelLogoutURI is the URI notified on back-channel logout.
	// +optional
	BackChannelLogoutURI *string `json:"backChannelLogoutURI,omitempty"`

	// State of the application. Defaults to Active. Setting it to Inactive
	// deactivates the application, setting it back to Active reactivates it.
	// +optional
	State *ApplicationState `json:"state,omitempty"`

	// RegenerateClientSecret requests a new client secret on every reconcile
	// where the application is (re)created. The new secret is published to the
	// connection secret. Note that this is a destructive operation - the
	// previously issued secret stops working immediately.
	// +optional
	RegenerateClientSecret *bool `json:"regenerateClientSecret,omitempty"`
}

// OIDCComplianceProblem is a single reason why Zitadel considers an OIDC
// application non compliant.
type OIDCComplianceProblem struct {
	// Key of the compliance problem.
	// +optional
	Key *string `json:"key,omitempty"`

	// LocalizedMessage describing the compliance problem.
	// +optional
	LocalizedMessage *string `json:"localizedMessage,omitempty"`
}

// OIDCApplicationObservation are the observable fields of an OIDCApplication.
type OIDCApplicationObservation struct {
	// ID is the Zitadel application ID.
	// +optional
	ID *string `json:"id,omitempty"`

	// ProjectID is the ID of the owning project.
	// +optional
	ProjectID *string `json:"projectID,omitempty"`

	// Name is the display name of the application.
	// +optional
	Name *string `json:"name,omitempty"`

	// State of the application.
	// +optional
	State *ApplicationState `json:"state,omitempty"`

	// ClientID is the OAuth2 client ID of the application.
	// +optional
	ClientID *string `json:"clientID,omitempty"`

	// RedirectURIs currently configured on the application.
	// +optional
	RedirectURIs []string `json:"redirectURIs,omitempty"`

	// PostLogoutRedirectURIs currently configured on the application.
	// +optional
	PostLogoutRedirectURIs []string `json:"postLogoutRedirectURIs,omitempty"`

	// ResponseTypes currently configured on the application.
	// +optional
	ResponseTypes []OIDCResponseType `json:"responseTypes,omitempty"`

	// GrantTypes currently configured on the application.
	// +optional
	GrantTypes []OIDCGrantType `json:"grantTypes,omitempty"`

	// ApplicationType of the application.
	// +optional
	ApplicationType *OIDCApplicationType `json:"applicationType,omitempty"`

	// AuthMethodType of the application.
	// +optional
	AuthMethodType *OIDCAuthMethodType `json:"authMethodType,omitempty"`

	// AccessTokenType of the application.
	// +optional
	AccessTokenType *OIDCTokenType `json:"accessTokenType,omitempty"`

	// NonCompliant is true when Zitadel flagged the application as non
	// compliant with the OIDC specification.
	// +optional
	NonCompliant *bool `json:"nonCompliant,omitempty"`

	// ComplianceProblems lists the reasons for a non compliant configuration.
	// +optional
	ComplianceProblems []OIDCComplianceProblem `json:"complianceProblems,omitempty"`

	// AllowedOrigins are the CORS origins Zitadel allows for this client.
	// +optional
	AllowedOrigins []string `json:"allowedOrigins,omitempty"`

	// CreationDate is the timestamp the application was created at.
	// +optional
	CreationDate *metav1.Time `json:"creationDate,omitempty"`

	// ChangeDate is the timestamp the application was last modified at.
	// +optional
	ChangeDate *metav1.Time `json:"changeDate,omitempty"`
}

// An OIDCApplicationSpec defines the desired state of an OIDCApplication.
type OIDCApplicationSpec struct {
	xpv2.ManagedResourceSpec `json:",inline"`
	ForProvider              OIDCApplicationParameters `json:"forProvider"`
}

// An OIDCApplicationStatus represents the observed state of an OIDCApplication.
type OIDCApplicationStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OIDCApplicationObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion

// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="STATE",type="string",JSONPath=".status.atProvider.state"
// +kubebuilder:printcolumn:name="CLIENT-ID",type="string",JSONPath=".status.atProvider.clientID"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,zitadel}
// An OIDCApplication is an OAuth2/OIDC client registered with Zitadel.
type OIDCApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OIDCApplicationSpec   `json:"spec"`
	Status OIDCApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OIDCApplicationList contains a list of OIDCApplication.
type OIDCApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OIDCApplication `json:"items"`
}

// OIDCApplication type metadata.
var (
	OIDCApplicationKind             = reflect.TypeOf(OIDCApplication{}).Name()
	OIDCApplicationGroupKind        = schema.GroupKind{Group: Group, Kind: OIDCApplicationKind}.String()
	OIDCApplicationKindAPIVersion   = OIDCApplicationKind + "." + SchemeGroupVersion.String()
	OIDCApplicationGroupVersionKind = SchemeGroupVersion.WithKind(OIDCApplicationKind)
)

func init() {
	SchemeBuilder.Register(&OIDCApplication{}, &OIDCApplicationList{})
}
