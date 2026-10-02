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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"

	mgmtv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/management"
	"github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/object"
	userv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user/v2"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DefaultMachineKeyBits is the RSA modulus size of a generated machine key.
// Zitadel requires an RSA key, and 2048 bits is the size the console issues.
const DefaultMachineKeyBits = 2048

// MachineKey is a machine key belonging to a service account (machine user).
type MachineKey struct {
	KeyID          string
	UserID         string
	OrganizationID string
	ExpirationDate string
	CreationDate   string
}

// MachineKeyJSON is the content of the key.json file that Zitadel clients
// expect. It is what gets written to the connection secret, so that a workload
// can authenticate with `authType: ServiceAccount` without anything having to
// leave the cluster.
type MachineKeyJSON struct {
	// Type is always "serviceaccount" for a machine key.
	Type string `json:"type"`
	// KeyID is the Zitadel machine key ID.
	KeyID string `json:"keyId"`
	// UserID is the ID of the machine user the key belongs to.
	UserID string `json:"userId"`
	// Key is the PEM encoded PKCS#1 RSA private key.
	Key string `json:"key"`
	// ExpirationDate is when the key stops working.
	ExpirationDate string `json:"expirationDate,omitempty"`
}

// GetMachineKey returns a single machine key of one user by its ID.
func (c *Client) GetMachineKey(ctx context.Context, orgID, userID, keyID string) (*MachineKey, error) {
	keys, err := c.ListMachineKeys(ctx, orgID, userID)
	if err != nil {
		return nil, err
	}

	for i := range keys {
		if keys[i].KeyID == keyID {
			return &keys[i], nil
		}
	}

	return nil, ErrNotFound
}

// ListMachineKeys lists the machine keys of one user.
//
// This uses the v1 management API on purpose. The v2 user service answers
// ListKeys for the authenticated user only, so a key belonging to a service
// account is invisible to it: the lookup would always come back empty, and a
// controller that could not see the key it had just created would keep creating
// new ones. The v1 endpoint takes the owning user instead. See management.go for
// why the v1 API is used at all.
func (c *Client) ListMachineKeys(ctx context.Context, orgID, userID string) ([]MachineKey, error) {
	mgmt, err := c.managementClient(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// The SDK marks this endpoint deprecated in favour of the v2 user service,
	// but the v2 endpoint only answers for the authenticated user, which cannot
	// see a service account's keys. This is the only call that can. See
	// management.go for why the v1 API is used at all.
	//nolint:staticcheck // A machine key has to be read back per owning user.
	resp, err := mgmt.ListMachineKeys(ctx, &mgmtv1.ListMachineKeysRequest{
		UserId: userID,
		Query:  &object.ListQuery{Limit: 1000},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list the machine keys of user %s: %w", userID, err)
	}

	out := make([]MachineKey, 0, len(resp.GetResult()))
	for _, k := range resp.GetResult() {
		// The response describes the key, not the user it was requested for:
		// the key's own resource owner is its organization, which says nothing
		// about which user holds it. The request is what carries that.
		key := MachineKey{
			KeyID:          k.GetId(),
			UserID:         userID,
			OrganizationID: orgID,
		}
		if d := k.GetExpirationDate(); d.IsValid() {
			key.ExpirationDate = d.AsTime().Format(time.RFC3339)
		}
		if d := k.GetDetails().GetCreationDate(); d.IsValid() {
			key.CreationDate = d.AsTime().Format(time.RFC3339)
		}
		out = append(out, key)
	}

	return out, nil
}

// CreateMachineKeyInput describes a machine key to add.
type CreateMachineKeyInput struct {
	// UserID is the machine user the key belongs to.
	UserID string

	// ExpirationDate is when the key stops working. Zitadel requires one.
	ExpirationDate time.Time

	// Bits is the RSA modulus size of the generated key.
	Bits int
}

// CreateMachineKey generates an RSA key pair, registers the public half with
// Zitadel and returns the key.json content together with the key ID.
//
// The private key is generated here rather than handed in because Zitadel only
// ever stores the public half: without generating it, a declarative machine key
// would be impossible and the provider's own service account credentials could
// never be bootstrapped from inside the cluster.
func (c *Client) CreateMachineKey(ctx context.Context, in CreateMachineKeyInput) (string, MachineKeyJSON, error) {
	bits := in.Bits
	if bits == 0 {
		bits = DefaultMachineKeyBits
	}

	key, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", MachineKeyJSON{}, fmt.Errorf("cannot generate an RSA machine key: %w", err)
	}

	// Zitadel expects the public key as PEM encoded PKIX.
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", MachineKeyJSON{}, fmt.Errorf("cannot encode the machine public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})

	resp, err := c.user.AddKey(ctx, &userv2.AddKeyRequest{
		UserId:         in.UserID,
		PublicKey:      pubPEM,
		ExpirationDate: timestamppb.New(in.ExpirationDate),
	})
	if err != nil {
		return "", MachineKeyJSON{}, fmt.Errorf("cannot add a machine key to user %s: %w", in.UserID, err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	return resp.GetKeyId(), MachineKeyJSON{
		Type:           "serviceaccount",
		KeyID:          resp.GetKeyId(),
		UserID:         in.UserID,
		Key:            string(privPEM),
		ExpirationDate: in.ExpirationDate.Format(time.RFC3339),
	}, nil
}

// EncodeMachineKey renders a machine key as the JSON document Zitadel clients
// read. The output is what goes into a connection secret.
func EncodeMachineKey(k MachineKeyJSON) ([]byte, error) {
	b, err := json.Marshal(k)
	if err != nil {
		return nil, fmt.Errorf("cannot encode the machine key: %w", err)
	}

	return b, nil
}

// DecodeMachineKey parses a key.json document.
func DecodeMachineKey(data []byte) (MachineKeyJSON, error) {
	out := MachineKeyJSON{}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, fmt.Errorf("cannot decode the machine key: %w", err)
	}

	if out.KeyID == "" || out.Key == "" || out.UserID == "" {
		return out, fmt.Errorf("the machine key is missing keyId, userId or key")
	}

	return out, nil
}

// DeleteMachineKey removes a machine key from the machine user.
func (c *Client) DeleteMachineKey(ctx context.Context, userID, keyID string) error {
	if _, err := c.user.RemoveKey(ctx, &userv2.RemoveKeyRequest{UserId: userID, KeyId: keyID}); err != nil {
		if IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("cannot remove machine key %s: %w", keyID, err)
	}

	return nil
}
