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

	filterv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/filter/v2"
	objectv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/object/v2"
	userv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateHumanUserInput describes a human user that should be created.
type CreateHumanUserInput struct {
	// OrganizationID is the organization the user is created in.
	OrganizationID string

	// UserID optionally pins the user ID.
	UserID string

	// UserName is the username. Zitadel defaults it to the email address when
	// empty.
	UserName string

	// Profile holds the name related attributes of the user.
	Profile SetHumanProfile

	// Email is the email address of the user.
	Email string

	// EmailVerification determines how Zitadel handles the email address.
	// Supported values are SendCode, ReturnCode and Verified. Defaults to
	// SendCode.
	EmailVerification string

	// Phone is the phone number of the user.
	Phone string

	// PhoneVerification determines how Zitadel handles the phone number.
	PhoneVerification string

	// Password is the initial password of the user. Empty means no password is
	// set, which leaves the user in the Initial state.
	Password string

	// PasswordChangeRequired forces the user to change its password on the next
	// login.
	PasswordChangeRequired bool

	// IDPLinks are identity provider links added at creation time.
	IDPLinks []IDPLink

	// Metadata entries set at creation time.
	Metadata []MetadataEntry
}

// SetHumanProfile holds the name related attributes of a human user.
type SetHumanProfile struct {
	GivenName         string
	FamilyName        string
	NickName          string
	HasNickName       bool
	DisplayName       string
	HasDisplayName    bool
	PreferredLanguage string
	HasLanguage       bool
	Gender            userv2.Gender
	HasGender         bool
}

// IDPLink references a user at an upstream identity provider.
type IDPLink struct {
	IDPID    string
	UserID   string
	UserName string
}

// MetadataEntry is a single metadata key/value pair.
type MetadataEntry struct {
	Key   string
	Value string
}

// CreateHumanUserResult is returned after creating a human user.
type CreateHumanUserResult struct {
	// UserID is the ID of the created user.
	UserID string

	// EmailCode is the email verification code, when ReturnCode was requested.
	EmailCode string

	// PhoneCode is the phone verification code, when ReturnCode was requested.
	PhoneCode string
}

// CreateServiceAccountInput describes a service account (machine user) that
// should be created.
type CreateServiceAccountInput struct {
	// OrganizationID is the organization the account is created in.
	OrganizationID string

	// UserID optionally pins the user ID.
	UserID string

	// UserName is the username. Zitadel defaults it to the user ID when empty.
	UserName string

	// Name is the display name of the service account.
	Name string

	// Description of the service account.
	Description string

	// AccessTokenType is Bearer (default) or Jwt.
	AccessTokenType userv2.AccessTokenType

	// Metadata entries set at creation time.
	Metadata []MetadataEntry
}

// User describes a Zitadel user (human or machine).
type User struct {
	UserID             string
	UserName           string
	PreferredLoginName string
	LoginNames         []string
	State              string
	CreationDate       string
	ChangeDate         string

	// Human is set when the user is an interactive user.
	Human *HumanUserDetails

	// Machine is set when the user is a service account.
	Machine *MachineUserDetails
}

// HumanUserDetails holds the attributes of an interactive user.
type HumanUserDetails struct {
	GivenName             string
	FamilyName            string
	NickName              string
	DisplayName           string
	PreferredLanguage     string
	Gender                string
	Email                 string
	EmailVerified         bool
	Phone                 string
	PhoneVerified         bool
	PasswordChangeRequire bool
}

// MachineUserDetails holds the attributes of a service account.
type MachineUserDetails struct {
	Name            string
	Description     string
	HasSecret       bool
	AccessTokenType string
}

// GetUser returns the user with the supplied ID. It returns ErrNotFound when
// the user does not exist.
func (c *Client) GetUser(ctx context.Context, userID string) (*User, error) { //nolint:gocyclo // a flat mapping of the protobuf response
	if userID == "" {
		return nil, fmt.Errorf("%w: empty user ID", ErrNotFound)
	}

	resp, err := c.user.GetUserByID(ctx, &userv2.GetUserByIDRequest{UserId: userID})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get user %s: %w", userID, err)
	}

	if resp.GetUser() == nil {
		return nil, ErrNotFound
	}

	u := resp.GetUser()
	out := &User{
		UserID:             u.GetUserId(),
		UserName:           u.GetUsername(),
		PreferredLoginName: u.GetPreferredLoginName(),
		LoginNames:         u.GetLoginNames(),
		State:              UserStateFromProto(u.GetState()),
	}

	switch t := u.GetType().(type) {
	case *userv2.User_Human:
		if h := t.Human; h != nil {
			details := &HumanUserDetails{
				PasswordChangeRequire: h.GetPasswordChangeRequired(),
			}
			if p := h.GetProfile(); p != nil {
				details.GivenName = p.GetGivenName()
				details.FamilyName = p.GetFamilyName()
				details.NickName = p.GetNickName()
				details.DisplayName = p.GetDisplayName()
				details.PreferredLanguage = p.GetPreferredLanguage()
				details.Gender = GenderFromProto(p.GetGender())
			}
			if e := h.GetEmail(); e != nil {
				details.Email = e.GetEmail()
				details.EmailVerified = e.GetIsVerified()
			}
			if ph := h.GetPhone(); ph != nil {
				details.Phone = ph.GetPhone()
				details.PhoneVerified = ph.GetIsVerified()
			}
			out.Human = details
		}
	case *userv2.User_Machine:
		if m := t.Machine; m != nil {
			out.Machine = &MachineUserDetails{
				Name:            m.GetName(),
				Description:     m.GetDescription(),
				HasSecret:       m.GetHasSecret(),
				AccessTokenType: AccessTokenTypeFromProto(m.GetAccessTokenType()),
			}
		}
	}

	return out, nil
}

// ListUsersByOrgID returns all users of an organization.
func (c *Client) ListUsersByOrgID(ctx context.Context, organizationID string) ([]*User, error) {
	resp, err := c.user.ListUsers(ctx, &userv2.ListUsersRequest{
		Query: &objectv2.ListQuery{
			Offset: 0,
			Limit:  1000,
		},
		Queries: []*userv2.SearchQuery{
			{
				Query: &userv2.SearchQuery_OrganizationIdQuery{
					OrganizationIdQuery: &userv2.OrganizationIdQuery{OrganizationId: organizationID},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list users of organization %s: %w", organizationID, err)
	}

	out := make([]*User, 0, len(resp.GetResult()))
	for _, u := range resp.GetResult() {
		user := &User{
			UserID:             u.GetUserId(),
			UserName:           u.GetUsername(),
			PreferredLoginName: u.GetPreferredLoginName(),
			LoginNames:         u.GetLoginNames(),
			State:              UserStateFromProto(u.GetState()),
		}
		switch t := u.GetType().(type) {
		case *userv2.User_Human:
			if h := t.Human; h != nil {
				details := &HumanUserDetails{}
				if e := h.GetEmail(); e != nil {
					details.Email = e.GetEmail()
					details.EmailVerified = e.GetIsVerified()
				}
				user.Human = details
			}
		case *userv2.User_Machine:
			if m := t.Machine; m != nil {
				user.Machine = &MachineUserDetails{
					Name:            m.GetName(),
					Description:     m.GetDescription(),
					AccessTokenType: AccessTokenTypeFromProto(m.GetAccessTokenType()),
				}
			}
		}
		out = append(out, user)
	}

	return out, nil
}

// CreateHumanUser creates an interactive user.
func (c *Client) CreateHumanUser(ctx context.Context, in CreateHumanUserInput) (*CreateHumanUserResult, error) { //nolint:gocyclo // flat request construction
	human := &userv2.CreateUserRequest_Human{
		Profile: &userv2.SetHumanProfile{
			GivenName:  in.Profile.GivenName,
			FamilyName: in.Profile.FamilyName,
		},
		Email: &userv2.SetHumanEmail{Email: in.Email},
	}

	if in.Profile.HasNickName {
		human.Profile.NickName = &in.Profile.NickName
	}
	if in.Profile.HasDisplayName {
		human.Profile.DisplayName = &in.Profile.DisplayName
	}
	if in.Profile.HasLanguage {
		human.Profile.PreferredLanguage = &in.Profile.PreferredLanguage
	}
	if in.Profile.HasGender {
		g := in.Profile.Gender
		human.Profile.Gender = &g
	}

	switch in.EmailVerification {
	case "", "SendCode":
		human.Email.Verification = &userv2.SetHumanEmail_SendCode{SendCode: &userv2.SendEmailVerificationCode{}}
	case "ReturnCode":
		human.Email.Verification = &userv2.SetHumanEmail_ReturnCode{ReturnCode: &userv2.ReturnEmailVerificationCode{}}
	case "Verified":
		human.Email.Verification = &userv2.SetHumanEmail_IsVerified{IsVerified: true}
	default:
		return nil, fmt.Errorf("unsupported email verification %q", in.EmailVerification)
	}

	if in.Phone != "" {
		phone := &userv2.SetHumanPhone{Phone: in.Phone}
		switch in.PhoneVerification {
		case "", "SendCode":
			phone.Verification = &userv2.SetHumanPhone_SendCode{SendCode: &userv2.SendPhoneVerificationCode{}}
		case "ReturnCode":
			phone.Verification = &userv2.SetHumanPhone_ReturnCode{ReturnCode: &userv2.ReturnPhoneVerificationCode{}}
		case "Verified":
			phone.Verification = &userv2.SetHumanPhone_IsVerified{IsVerified: true}
		default:
			return nil, fmt.Errorf("unsupported phone verification %q", in.PhoneVerification)
		}
		human.Phone = phone
	}

	if in.Password != "" {
		human.PasswordType = &userv2.CreateUserRequest_Human_Password{
			Password: &userv2.Password{
				Password:       in.Password,
				ChangeRequired: in.PasswordChangeRequired,
			},
		}
	}

	for _, l := range in.IDPLinks {
		human.IdpLinks = append(human.IdpLinks, &userv2.IDPLink{
			IdpId:    l.IDPID,
			UserId:   l.UserID,
			UserName: l.UserName,
		})
	}

	req := &userv2.CreateUserRequest{
		OrganizationId: in.OrganizationID,
		UserType:       &userv2.CreateUserRequest_Human_{Human: human},
	}
	if len(in.Metadata) > 0 {
		req.Metadata = toProtoMetadata(in.Metadata)
	}
	if in.UserID != "" {
		id := in.UserID
		req.UserId = &id
	}
	if in.UserName != "" {
		name := in.UserName
		req.Username = &name
	}

	resp, err := c.user.CreateUser(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("cannot create human user in organization %s: %w", in.OrganizationID, err)
	}

	out := &CreateHumanUserResult{UserID: resp.GetId()}
	if resp.EmailCode != nil {
		out.EmailCode = resp.GetEmailCode()
	}
	if resp.PhoneCode != nil {
		out.PhoneCode = resp.GetPhoneCode()
	}

	return out, nil
}

// CreateServiceAccount creates a machine user.
func (c *Client) CreateServiceAccount(ctx context.Context, in CreateServiceAccountInput) (string, error) {
	machine := &userv2.CreateUserRequest_Machine{
		Name:            in.Name,
		AccessTokenType: in.AccessTokenType,
	}
	if in.Description != "" {
		d := in.Description
		machine.Description = &d
	}

	req := &userv2.CreateUserRequest{
		OrganizationId: in.OrganizationID,
		UserType:       &userv2.CreateUserRequest_Machine_{Machine: machine},
	}
	if in.UserID != "" {
		id := in.UserID
		req.UserId = &id
	}
	if in.UserName != "" {
		name := in.UserName
		req.Username = &name
	}
	if len(in.Metadata) > 0 {
		req.Metadata = toProtoMetadata(in.Metadata)
	}

	resp, err := c.user.CreateUser(ctx, req)
	if err != nil {
		return "", fmt.Errorf("cannot create service account in organization %s: %w", in.OrganizationID, err)
	}

	return resp.GetId(), nil
}

// UpdateHumanUser updates the mutable attributes of an interactive user.
// A nil profile (or a profile without any field set) is a no-op.
func (c *Client) UpdateHumanUser(ctx context.Context, userID string, profile *SetHumanProfile) error {
	if profile == nil {
		return nil
	}

	p := &userv2.UpdateUserRequest_Human_Profile{}
	set := false

	if profile.GivenName != "" {
		p.GivenName = &profile.GivenName
		set = true
	}
	if profile.FamilyName != "" {
		p.FamilyName = &profile.FamilyName
		set = true
	}
	if profile.HasNickName {
		p.NickName = &profile.NickName
		set = true
	}
	if profile.HasDisplayName {
		p.DisplayName = &profile.DisplayName
		set = true
	}
	if profile.HasLanguage {
		p.PreferredLanguage = &profile.PreferredLanguage
		set = true
	}
	if profile.HasGender {
		g := profile.Gender
		p.Gender = &g
		set = true
	}

	if !set {
		return nil
	}

	req := &userv2.UpdateUserRequest{
		UserId:   userID,
		UserType: &userv2.UpdateUserRequest_Human_{Human: &userv2.UpdateUserRequest_Human{Profile: p}},
	}

	if _, err := c.user.UpdateUser(ctx, req); err != nil {
		return fmt.Errorf("cannot update user %s: %w", userID, err)
	}

	return nil
}

// UpdateHumanEmail changes the email address of an interactive user.
func (c *Client) UpdateHumanEmail(ctx context.Context, userID, email, verification string) error {
	set := &userv2.SetHumanEmail{Email: email}
	switch verification {
	case "", "SendCode":
		set.Verification = &userv2.SetHumanEmail_SendCode{SendCode: &userv2.SendEmailVerificationCode{}}
	case "ReturnCode":
		set.Verification = &userv2.SetHumanEmail_ReturnCode{ReturnCode: &userv2.ReturnEmailVerificationCode{}}
	case "Verified":
		set.Verification = &userv2.SetHumanEmail_IsVerified{IsVerified: true}
	default:
		return fmt.Errorf("unsupported email verification %q", verification)
	}

	req := &userv2.UpdateUserRequest{UserId: userID}
	human := &userv2.UpdateUserRequest_Human{Email: set}
	req.UserType = &userv2.UpdateUserRequest_Human_{Human: human}

	if _, err := c.user.UpdateUser(ctx, req); err != nil {
		return fmt.Errorf("cannot update email of user %s: %w", userID, err)
	}

	return nil
}

// UpdateHumanPhone changes the phone number of an interactive user. Passing an
// empty phone removes it.
func (c *Client) UpdateHumanPhone(ctx context.Context, userID, phone, verification string) error {
	if phone == "" {
		// Zitadel removes a phone number by setting the phone field to the
		// empty string through UpdateUser. RemovePhone is deprecated.
		req := &userv2.UpdateUserRequest{
			UserId:   userID,
			UserType: &userv2.UpdateUserRequest_Human_{Human: &userv2.UpdateUserRequest_Human{Phone: &userv2.SetHumanPhone{}}},
		}
		if _, err := c.user.UpdateUser(ctx, req); err != nil {
			return fmt.Errorf("cannot remove phone of user %s: %w", userID, err)
		}

		return nil
	}

	req := &userv2.UpdateUserRequest{UserId: userID}

	set := &userv2.SetHumanPhone{Phone: phone}
	switch verification {
	case "", "SendCode":
		set.Verification = &userv2.SetHumanPhone_SendCode{SendCode: &userv2.SendPhoneVerificationCode{}}
	case "ReturnCode":
		set.Verification = &userv2.SetHumanPhone_ReturnCode{ReturnCode: &userv2.ReturnPhoneVerificationCode{}}
	case "Verified":
		set.Verification = &userv2.SetHumanPhone_IsVerified{IsVerified: true}
	default:
		return fmt.Errorf("unsupported phone verification %q", verification)
	}

	human := &userv2.UpdateUserRequest_Human{Phone: set}
	req.UserType = &userv2.UpdateUserRequest_Human_{Human: human}

	if _, err := c.user.UpdateUser(ctx, req); err != nil {
		return fmt.Errorf("cannot update phone of user %s: %w", userID, err)
	}

	return nil
}

// UpdateServiceAccount updates the mutable attributes of a service account.
// Empty arguments leave the corresponding attribute untouched; an update
// without any field set is a no-op. accessTokenType is the protobuf enum, or
// zero (Bearer) to leave it unchanged - which is why callers that want to
// change it must pass the desired value explicitly.
func (c *Client) UpdateServiceAccount(ctx context.Context, userID, name string, description *string, accessTokenType *userv2.AccessTokenType) error {
	machine := &userv2.UpdateUserRequest_Machine{}
	set := false

	if name != "" {
		n := name
		machine.Name = &n
		set = true
	}
	if description != nil {
		d := *description
		machine.Description = &d
		set = true
	}
	if accessTokenType != nil {
		machine.AccessTokenType = accessTokenType
		set = true
	}

	if !set {
		return nil
	}

	req := &userv2.UpdateUserRequest{
		UserId:   userID,
		UserType: &userv2.UpdateUserRequest_Machine_{Machine: machine},
	}

	if _, err := c.user.UpdateUser(ctx, req); err != nil {
		return fmt.Errorf("cannot update service account %s: %w", userID, err)
	}

	return nil
}

// UpdateUsername changes the username of a user.
//
// Zitadel's UpdateUser requires a user type to be present even when only the
// username changes, so an empty human user payload is sent along. machine is
// false for human users and true for service accounts.
func (c *Client) UpdateUsername(ctx context.Context, userID, userName string, machine bool) error {
	if userName == "" {
		return nil
	}

	name := userName
	req := &userv2.UpdateUserRequest{UserId: userID, Username: &name}
	if machine {
		req.UserType = &userv2.UpdateUserRequest_Machine_{Machine: &userv2.UpdateUserRequest_Machine{}}
	} else {
		req.UserType = &userv2.UpdateUserRequest_Human_{Human: &userv2.UpdateUserRequest_Human{}}
	}

	if _, err := c.user.UpdateUser(ctx, req); err != nil {
		return fmt.Errorf("cannot update the username of user %s: %w", userID, err)
	}

	return nil
}

// SetUserState activates, deactivates, locks or unlocks a user. Only changes
// that actually alter the state are sent to Zitadel.
//
//nolint:gocyclo // the state machine is a flat switch over Zitadel operations
func (c *Client) SetUserState(ctx context.Context, userID, current, desired string) error {
	if desired == "" || desired == current {
		return nil
	}

	var err error

	switch desired {
	case StateActive:
		switch current {
		case StateInactive:
			_, err = c.user.ReactivateUser(ctx, &userv2.ReactivateUserRequest{UserId: userID})
		case StateLocked:
			_, err = c.user.UnlockUser(ctx, &userv2.UnlockUserRequest{UserId: userID})
		}
	case StateInactive:
		_, err = c.user.DeactivateUser(ctx, &userv2.DeactivateUserRequest{UserId: userID})
	case StateLocked:
		_, err = c.user.LockUser(ctx, &userv2.LockUserRequest{UserId: userID})
	default:
		// StateInitial is a state a user is in before its first login. It
		// cannot be requested explicitly.
		return nil
	}

	if err != nil {
		return fmt.Errorf("cannot set the state of user %s to %s: %w", userID, desired, err)
	}

	return nil
}

// DeleteUser removes a user from Zitadel.
func (c *Client) DeleteUser(ctx context.Context, userID string) error {
	if _, err := c.user.DeleteUser(ctx, &userv2.DeleteUserRequest{UserId: userID}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot delete user %s: %w", userID, err)
	}

	return nil
}

// PersonalAccessToken describes a Personal Access Token.
type PersonalAccessToken struct {
	TokenID        string
	UserID         string
	OrganizationID string
	ExpirationDate string
}

// CreatePersonalAccessToken issues a new Personal Access Token for a user. The
// token value itself is only returned by Zitadel once. expirationDate must be
// an RFC3339 timestamp, or empty to let Zitadel pick its maximum.
func (c *Client) CreatePersonalAccessToken(ctx context.Context, userID, expirationDate string) (string, string, error) {
	req := &userv2.AddPersonalAccessTokenRequest{UserId: userID}
	if expirationDate != "" {
		t, err := time.Parse(time.RFC3339, expirationDate)
		if err != nil {
			return "", "", fmt.Errorf("cannot parse PAT expiration date %q: %w", expirationDate, err)
		}
		req.ExpirationDate = timestamppb.New(t)
	}

	resp, err := c.user.AddPersonalAccessToken(ctx, req)
	if err != nil {
		return "", "", fmt.Errorf("cannot create personal access token for user %s: %w", userID, err)
	}

	return resp.GetTokenId(), resp.GetToken(), nil
}

// GetPersonalAccessToken returns a single Personal Access Token by ID. It
// returns ErrNotFound when the token does not exist.
func (c *Client) GetPersonalAccessToken(ctx context.Context, tokenID string) (*PersonalAccessToken, error) {
	resp, err := c.user.ListPersonalAccessTokens(ctx, &userv2.ListPersonalAccessTokensRequest{
		Pagination: &filterv2.PaginationRequest{Offset: 0, Limit: 1},
		Filters: []*userv2.PersonalAccessTokensSearchFilter{
			{
				Filter: &userv2.PersonalAccessTokensSearchFilter_TokenIdFilter{
					TokenIdFilter: &filterv2.IDFilter{Id: tokenID},
				},
			},
		},
	})
	if err != nil {
		if IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("cannot get personal access token %s: %w", tokenID, err)
	}

	if len(resp.GetResult()) == 0 {
		return nil, ErrNotFound
	}

	t := resp.GetResult()[0]

	return &PersonalAccessToken{
		TokenID:        t.GetId(),
		UserID:         t.GetUserId(),
		OrganizationID: t.GetOrganizationId(),
		ExpirationDate: t.GetExpirationDate().AsTime().Format(time.RFC3339),
	}, nil
}

// ListPersonalAccessTokensByUser returns all Personal Access Tokens of a user.
func (c *Client) ListPersonalAccessTokensByUser(ctx context.Context, userID string) ([]*PersonalAccessToken, error) {
	resp, err := c.user.ListPersonalAccessTokens(ctx, &userv2.ListPersonalAccessTokensRequest{
		Pagination: &filterv2.PaginationRequest{Offset: 0, Limit: 1000},
		Filters: []*userv2.PersonalAccessTokensSearchFilter{
			{
				Filter: &userv2.PersonalAccessTokensSearchFilter_UserIdFilter{
					UserIdFilter: &filterv2.IDFilter{Id: userID},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list personal access tokens of user %s: %w", userID, err)
	}

	out := make([]*PersonalAccessToken, 0, len(resp.GetResult()))
	for _, t := range resp.GetResult() {
		out = append(out, &PersonalAccessToken{
			TokenID:        t.GetId(),
			UserID:         t.GetUserId(),
			OrganizationID: t.GetOrganizationId(),
		})
	}

	return out, nil
}

// DeletePersonalAccessToken revokes a Personal Access Token.
func (c *Client) DeletePersonalAccessToken(ctx context.Context, userID, tokenID string) error {
	if _, err := c.user.RemovePersonalAccessToken(ctx, &userv2.RemovePersonalAccessTokenRequest{
		UserId:  userID,
		TokenId: tokenID,
	}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot remove personal access token %s: %w", tokenID, err)
	}

	return nil
}

func toProtoMetadata(entries []MetadataEntry) []*userv2.Metadata {
	out := make([]*userv2.Metadata, 0, len(entries))
	for _, e := range entries {
		out = append(out, &userv2.Metadata{Key: e.Key, Value: []byte(e.Value)})
	}

	return out
}
