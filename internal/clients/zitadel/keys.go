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

	"google.golang.org/protobuf/types/known/timestamppb"

	apiv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/application/v2"
	permissionv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/internal_permission/v2"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/webkey/v2"
)

// neverExpires is how far ahead a key with no expiry asked for is set.
//
// Zitadel refuses an application key whose expiry is the zero time, so a key
// that is not meant to expire has to be given a real date. This is the furthest
// one it accepts, so the key behaves the way a manifest asking for no expiry
// means: it does not expire within any lifetime anyone will live.
var neverExpires = time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)

// Keys and grants
//
// Three of the last resources left are about keys: the signing keys an
// application verifies a Zitadel token with, the one application key it
// authenticates itself with, and which of the signing keys is the active one.
// The fourth grants a user a role on a project.

// WebKey is one of Zitadel's own signing keys.
//
// The private half never leaves Zitadel, so a key is an identity and a state and
// nothing more: there is no material to read back, and rotating one means
// replacing it.
type WebKey struct {
	ID           string
	Algorithm    string
	State        string
	CreationDate string
	ChangeDate   string
}

// WebKeyAlgorithm says what shape of key to generate.
type WebKeyAlgorithm struct {
	// Type is `rsa`, `ecdsa` or `ed25519`.
	Type string

	// RSABits is the key size, `RSA_BITS_2048`, `RSA_BITS_3072` or
	// `RSA_BITS_4096`. Ignored for the other types.
	RSABits string

	// RSAHasher is the signing algorithm, `RSA_HASHER_SHA256`,
	// `RSA_HASHER_SHA384` or `RSA_HASHER_SHA512`. Ignored for the other types.
	RSAHasher string

	// ECDSACurve is the curve for an ECDSA key, `ECDSA_CURVE_P256`,
	// `ECDSA_CURVE_P384` or `ECDSA_CURVE_P512`. Ignored for the other types.
	//
	// It has to be said: Zitadel documents P-256 as the default but rejects an
	// unset curve as invalid rather than choosing one.
	ECDSACurve string
}

// CreateWebKey makes Zitadel generate a signing key.
//
// Zitadel makes it inactive: an inactive key verifies nothing, so a new key has
// to be activated before it is trusted, and ActivatedWebKey is what does that.
func (c *Client) CreateWebKey(ctx context.Context, alg WebKeyAlgorithm) (string, error) {
	req := &webkey.CreateWebKeyRequest{}

	switch alg.Type {
	case "rsa", "":
		bits, hasher, err := rsaSettings(alg)
		if err != nil {
			return "", err
		}

		req.Key = &webkey.CreateWebKeyRequest_Rsa{Rsa: &webkey.RSA{Bits: bits, Hasher: hasher}}

	case "ecdsa":
		curve, err := ecdsaCurve(alg)
		if err != nil {
			return "", err
		}

		req.Key = &webkey.CreateWebKeyRequest_Ecdsa{Ecdsa: &webkey.ECDSA{Curve: curve}}

	case "ed25519":
		req.Key = &webkey.CreateWebKeyRequest_Ed25519{Ed25519: &webkey.ED25519{}}

	default:
		return "", fmt.Errorf("zitadel cannot generate a web key of type %q: it must be rsa, ecdsa or ed25519", alg.Type)
	}

	resp, err := c.webkey.CreateWebKey(ctx, req)
	if err != nil {
		return "", err
	}

	return resp.GetId(), nil
}

// rsaSettings reads the key size and signing algorithm, defaulting both.
//
// Zitadel's own defaults are 2048 and SHA256, and are spelled as the
// unspecified value rather than the named one, so an unset field is filled in
// here rather than being left for Zitadel to interpret.
func rsaSettings(alg WebKeyAlgorithm) (webkey.RSABits, webkey.RSAHasher, error) {
	bits := webkey.RSABits(1)
	if alg.RSABits != "" {
		v, ok := webkey.RSABits_value[alg.RSABits]
		if !ok {
			return 0, 0, fmt.Errorf("zitadel has no key size %q: it must be one of %v",
				alg.RSABits, webkey.RSABits_name)
		}

		bits = webkey.RSABits(v)
	}

	hasher := webkey.RSAHasher(1)
	if alg.RSAHasher != "" {
		v, ok := webkey.RSAHasher_value[alg.RSAHasher]
		if !ok {
			return 0, 0, fmt.Errorf("zitadel has no RSA hasher %q: it must be one of %v",
				alg.RSAHasher, webkey.RSAHasher_name)
		}

		hasher = webkey.RSAHasher(v)
	}

	return bits, hasher, nil
}

// ecdsaCurve reads the curve for an ECDSA key.
//
// Zitadel's own documentation calls P-256 the default, but an unset curve is
// refused as invalid rather than being filled in, so the default is applied here.
func ecdsaCurve(alg WebKeyAlgorithm) (webkey.ECDSACurve, error) {
	if alg.ECDSACurve == "" {
		return webkey.ECDSACurve_ECDSA_CURVE_P256, nil
	}

	v, ok := webkey.ECDSACurve_value[alg.ECDSACurve]
	if !ok {
		return 0, fmt.Errorf("zitadel has no ECDSA curve %q: it must be one of %v",
			alg.ECDSACurve, webkey.ECDSACurve_name)
	}

	return webkey.ECDSACurve(v), nil
}

// GetWebKey reads one signing key, or nil when Zitadel does not have it.
func (c *Client) GetWebKey(ctx context.Context, id string) (*WebKey, error) {
	keys, err := c.ListWebKeys(ctx)
	if err != nil {
		return nil, err
	}

	for i := range keys {
		if keys[i].ID == id {
			return &keys[i], nil
		}
	}

	return nil, nil //nolint:nilnil // an absent key reads as absent, which is what it is
}

// ListWebKeys reads Zitadel's signing keys.
func (c *Client) ListWebKeys(ctx context.Context) ([]WebKey, error) {
	resp, err := c.webkey.ListWebKeys(ctx, &webkey.ListWebKeysRequest{})
	if err != nil {
		return nil, err
	}

	out := make([]WebKey, 0, len(resp.GetWebKeys()))
	for _, k := range resp.GetWebKeys() {
		// The state is an enumeration sent as a number, so it is read through
		// its name map rather than converted to a string, which would yield the
		// character with that code point.
		state, ok := webkey.State_name[int32(k.GetState())]
		if !ok {
			return nil, fmt.Errorf("zitadel reported the unknown web key state %d", int32(k.GetState()))
		}

		out = append(out, WebKey{
			ID:           k.GetId(),
			Algorithm:    webKeyAlgorithm(k),
			State:        state,
			CreationDate: formatTime(k.GetCreationDate()),
			ChangeDate:   formatTime(k.GetChangeDate()),
		})
	}

	return out, nil
}

// webKeyAlgorithm names the kind of key without reading its material.
//
// Only the size and the hasher come back, never the key itself, and the key type
// is the one thing worth reporting about it.
func webKeyAlgorithm(k *webkey.WebKey) string {
	switch key := k.GetKey().(type) {
	case *webkey.WebKey_Rsa:
		return fmt.Sprintf("rsa %d %s", key.Rsa.GetBits(), key.Rsa.GetHasher())
	case *webkey.WebKey_Ecdsa:
		return "ecdsa"
	case *webkey.WebKey_Ed25519:
		return "ed25519"
	default:
		return ""
	}
}

// ActivateWebKey makes a signing key the one Zitadel signs with.
func (c *Client) ActivateWebKey(ctx context.Context, id string) error {
	_, err := c.webkey.ActivateWebKey(ctx, &webkey.ActivateWebKeyRequest{Id: id})
	return err
}

// DeleteWebKey removes a signing key.
//
// Zitadel will not remove the active one, so it has to be activated away first -
// the same rule that applies to an action something still calls.
func (c *Client) DeleteWebKey(ctx context.Context, id string) error {
	_, err := c.webkey.DeleteWebKey(ctx, &webkey.DeleteWebKeyRequest{Id: id})
	return err
}

// AddAppKey makes Zitadel generate a key an application authenticates itself
// with, and returns the private half.
//
// The private material is returned exactly once, at creation, and is never
// readable afterwards: Zitadel keeps only the public half.
func (c *Client) AddAppKey(ctx context.Context, projectID, appID string, expiry string) (string, []byte, error) {
	req := &apiv2.CreateApplicationKeyRequest{ProjectId: projectID, ApplicationId: appID}

	// Zitadel has no way to say "never expires" here: an absent expiry is read as
	// the zero time and refused with ExpireBeforeNow. A key that should not
	// expire is therefore given the furthest date it will take, which is what the
	// request has to say to mean what the manifest means.
	until := neverExpires
	if expiry != "" {
		d, err := time.ParseDuration(expiry)
		if err != nil {
			return "", nil, fmt.Errorf("cannot read the expiry %q: %w", expiry, err)
		}

		until = time.Now().Add(d)
	}

	req.ExpirationDate = timestamppb.New(until)

	resp, err := c.application.CreateApplicationKey(ctx, req)
	if err != nil {
		return "", nil, err
	}

	return resp.GetKeyId(), resp.GetKeyDetails(), nil
}

// DeleteAppKey removes a key an application authenticated itself with.
//
// Zitadel wants all three identifiers and validates each: a request carrying
// only the key ID is refused as invalid rather than being resolved.
func (c *Client) DeleteAppKey(ctx context.Context, projectID, appID, keyID string) error {
	_, err := c.application.DeleteApplicationKey(ctx, &apiv2.DeleteApplicationKeyRequest{
		ProjectId:     projectID,
		ApplicationId: appID,
		KeyId:         keyID,
	})
	return err
}

// GetAppKey reads one application key, or nil when Zitadel does not have it.
//
// Only the public half and the dates come back. A key whose private half was
// lost is therefore unreadable but present, and can only be replaced.
func (c *Client) GetAppKey(ctx context.Context, keyID string) (*AppKey, error) {
	resp, err := c.application.GetApplicationKey(ctx, &apiv2.GetApplicationKeyRequest{KeyId: keyID})
	if err != nil {
		if IsNotFound(err) {
			return nil, nil //nolint:nilnil // an absent key reads as absent
		}

		return nil, err
	}

	// Zitadel answers a key it does not have with an empty body rather than a
	// not found, so an empty identifier is what absence looks like here.
	if resp.GetKeyId() == "" {
		return nil, nil //nolint:nilnil // an absent key reads as absent
	}

	return &AppKey{
		ID:             resp.GetKeyId(),
		CreationDate:   formatTime(resp.GetCreationDate()),
		ExpirationDate: formatTime(resp.GetExpirationDate()),
	}, nil
}

// AppKey is the readable half of an application key.
//
// There is no type to report: the v2 API creates one kind of key and does not
// say which algorithm it used. Only the dates come back, and never the private
// half, which Zitadel returns exactly once at creation.
type AppKey struct {
	ID             string
	CreationDate   string
	ExpirationDate string
}

// Input returns the key in the shape the write API accepts.
func (k AppKey) Input() any { return k }

// ProjectMember is a user's membership of a project, with the roles they hold.
type ProjectMember struct {
	UserID string
	Roles  []string

	// The user and project details Zitadel reports alongside the membership,
	// which is what makes a member legible without a second lookup.
	UserName    string
	DisplayName string
	ProjectName string
}

// Input returns the membership in the shape the write API accepts.
func (m ProjectMember) Input() any { return m }

// AddProjectMember grants a user roles on a project.
//
// The membership is an upsert in Zitadel, so granting a role the user already
// has is not an error.
func (c *Client) AddProjectMember(ctx context.Context, projectID, userID string, roles []string) error {
	_, err := c.permission.CreateAdministrator(ctx, &permissionv2.CreateAdministratorRequest{
		UserId:   userID,
		Resource: projectResource(projectID),
		Roles:    roles,
	})
	return err
}

// projectResource names a project to the permission service, which grants roles
// against any resource through one call.
func projectResource(projectID string) *permissionv2.ResourceType {
	return &permissionv2.ResourceType{Resource: &permissionv2.ResourceType_ProjectId{ProjectId: projectID}}
}

// GetProjectMember reads one membership, or nil when the user is not a member.
func (c *Client) GetProjectMember(ctx context.Context, projectID, userID string) (*ProjectMember, error) {
	resp, err := c.permission.ListAdministrators(ctx, &permissionv2.ListAdministratorsRequest{})
	if err != nil {
		return nil, err
	}

	for _, a := range resp.GetAdministrators() {
		u, p := a.GetUser(), a.GetProject()
		if u == nil || p == nil || u.GetId() != userID || p.GetId() != projectID {
			continue
		}

		return &ProjectMember{
			UserID:      u.GetId(),
			Roles:       a.GetRoles(),
			UserName:    u.GetPreferredLoginName(),
			DisplayName: u.GetDisplayName(),
			ProjectName: p.GetName(),
		}, nil
	}

	return nil, nil //nolint:nilnil // a user who is not a member reads as absent
}

// UpdateProjectMember replaces the roles a user holds on a project.
func (c *Client) UpdateProjectMember(ctx context.Context, projectID, userID string, roles []string) error {
	_, err := c.permission.UpdateAdministrator(ctx, &permissionv2.UpdateAdministratorRequest{
		UserId:   userID,
		Resource: projectResource(projectID),
		Roles:    roles,
	})
	return err
}

// RemoveProjectMember takes a user off a project.
func (c *Client) RemoveProjectMember(ctx context.Context, projectID, userID string) error {
	_, err := c.permission.DeleteAdministrator(ctx, &permissionv2.DeleteAdministratorRequest{
		UserId:   userID,
		Resource: projectResource(projectID),
	})
	return err
}
