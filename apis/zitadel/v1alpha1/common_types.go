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

// UserState is the lifecycle state of a Zitadel user or service account.
// +kubebuilder:validation:Enum=Active;Inactive;Locked;Initial
type UserState string

const (
	// UserStateActive is an active user.
	UserStateActive UserState = "Active"
	// UserStateInactive is a deactivated user.
	UserStateInactive UserState = "Inactive"
	// UserStateLocked is a locked user.
	UserStateLocked UserState = "Locked"
	// UserStateInitial is a user that has never logged in.
	UserStateInitial UserState = "Initial"
)

// Gender of a human user.
// +kubebuilder:validation:Enum=Female;Male;Diverse;Unspecified
type Gender string

const (
	GenderFemale      Gender = "Female"
	GenderMale        Gender = "Male"
	GenderDiverse     Gender = "Diverse"
	GenderUnspecified Gender = "Unspecified"
)

// AccessTokenType of a service account.
// +kubebuilder:validation:Enum=Bearer;Jwt
type AccessTokenType string

const (
	// AccessTokenTypeBearer issues opaque bearer tokens.
	AccessTokenTypeBearer AccessTokenType = "Bearer"
	// AccessTokenTypeJwt issues signed JWT access tokens.
	AccessTokenTypeJwt AccessTokenType = "Jwt"
)

// EmailVerificationType determines how Zitadel handles the email address of a
// human user on creation.
// +kubebuilder:validation:Enum=SendCode;ReturnCode;Verified
type EmailVerificationType string

const (
	// EmailVerificationTypeSendCode sends a verification email and keeps the
	// address unverified.
	EmailVerificationTypeSendCode EmailVerificationType = "SendCode"
	// EmailVerificationTypeReturnCode returns the verification code to the
	// caller and keeps the address unverified.
	EmailVerificationTypeReturnCode EmailVerificationType = "ReturnCode"
	// EmailVerificationTypeVerified marks the address as verified. Only
	// possible when an initial password is supplied.
	EmailVerificationTypeVerified EmailVerificationType = "Verified"
)

// PhoneVerificationType determines how Zitadel handles the phone number of a
// human user on creation.
// +kubebuilder:validation:Enum=SendCode;ReturnCode;Verified
type PhoneVerificationType string

const (
	PhoneVerificationTypeSendCode   PhoneVerificationType = "SendCode"
	PhoneVerificationTypeReturnCode PhoneVerificationType = "ReturnCode"
	PhoneVerificationTypeVerified   PhoneVerificationType = "Verified"
)

// IDPLink references a user at an upstream identity provider.
type IDPLink struct {
	// IDPID is the ID of the identity provider.
	// +kubebuilder:validation:Required
	IDPID string `json:"idpID"`

	// UserID is the ID of the user at the identity provider.
	// +kubebuilder:validation:Required
	UserID string `json:"userID"`

	// UserName is the user name at the identity provider.
	// +kubebuilder:validation:Required
	UserName string `json:"userName"`
}

// MetadataEntry is a single metadata key/value pair.
type MetadataEntry struct {
	// Key of the metadata entry.
	// +kubebuilder:validation:Required
	Key string `json:"key"`

	// Value of the metadata entry.
	// +kubebuilder:validation:Required
	Value string `json:"value"`
}

// MetadataList is a list of metadata entries.
type MetadataList []MetadataEntry

// Connection secret keys published by the managed resources of this provider.
const (
	// ConnectionKeyUserID is the Zitadel user ID.
	ConnectionKeyUserID = "userID"
	// ConnectionKeyUsername is the (preferred) login name of a user.
	ConnectionKeyUsername = "username"
	// ConnectionKeyClientID is the OIDC client ID of an application.
	ConnectionKeyClientID = "clientID"
	// ConnectionKeyClientSecret is the OIDC client secret of an application.
	ConnectionKeyClientSecret = "clientSecret"
	// ConnectionKeyToken is a Personal Access Token.
	ConnectionKeyToken = "token"
	// ConnectionKeyTokenID is the ID of a Personal Access Token.
	ConnectionKeyTokenID = "tokenID"
	// ConnectionKeyEmailCode is an email verification code returned by Zitadel.
	ConnectionKeyEmailCode = "emailCode"
	// ConnectionKeyPhoneCode is a phone verification code returned by Zitadel.
	ConnectionKeyPhoneCode = "phoneCode"
	// ConnectionKeyKeyID is the ID of a machine key.
	ConnectionKeyKeyID = "keyID"
)
