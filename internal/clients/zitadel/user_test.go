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
	"testing"

	userv2 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/user/v2"
	"google.golang.org/grpc"
)

// The initial password and an already hashed one are two shapes of the same
// one-of in the create request. Picking the wrong arm would either send the
// hash as a plaintext password or ignore the migration value entirely.

type fakeUserService struct {
	userv2.UnimplementedUserServiceServer

	created *userv2.CreateUserRequest
}

func (f *fakeUserService) CreateUser(_ context.Context, in *userv2.CreateUserRequest) (*userv2.CreateUserResponse, error) {
	f.created = in

	return &userv2.CreateUserResponse{Id: "u1"}, nil
}

func newUserClient(t *testing.T) (*Client, *fakeUserService) {
	t.Helper()

	u := &fakeUserService{}

	fake := newFakeZitadel(t, func(s *grpc.Server) {
		userv2.RegisterUserServiceServer(s, u)
	})

	return fake.client(t), u
}

func TestCreateHumanUserPasswordType(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     CreateHumanUserInput
		check  func(*testing.T, *userv2.CreateUserRequest)
	}{
		"PlainPassword": {
			reason: "A password is sent as the plaintext arm",
			in: CreateHumanUserInput{
				Password:               "Str0ng-Passw0rd!",
				PasswordChangeRequired: true,
			},
			check: func(t *testing.T, req *userv2.CreateUserRequest) {
				t.Helper()

				got, ok := req.GetHuman().GetPasswordType().(*userv2.CreateUserRequest_Human_Password)
				if !ok {
					t.Fatalf("PasswordType: want the plaintext arm, got %T", req.GetHuman().GetPasswordType())
				}

				if got.Password.GetPassword() != "Str0ng-Passw0rd!" {
					t.Errorf("password: want the configured value")
				}

				if !got.Password.GetChangeRequired() {
					t.Error("changeRequired: want true")
				}
			},
		},
		"HashedPassword": {
			reason: "A hashed password is sent as the hashed arm and never as plaintext",
			in: CreateHumanUserInput{
				HashedPassword: "JDJhJDEwJCR...",
			},
			check: func(t *testing.T, req *userv2.CreateUserRequest) {
				t.Helper()

				got, ok := req.GetHuman().GetPasswordType().(*userv2.CreateUserRequest_Human_HashedPassword)
				if !ok {
					t.Fatalf("PasswordType: want the hashed arm, got %T", req.GetHuman().GetPasswordType())
				}

				if got.HashedPassword.GetHash() != "JDJhJDEwJCR..." {
					t.Errorf("hash: want the configured value")
				}

				if got.HashedPassword.GetChangeRequired() {
					t.Error("changeRequired: want false")
				}
			},
		},
		"Neither": {
			reason: "No password leaves the user in the Initial state",
			in:     CreateHumanUserInput{},
			check: func(t *testing.T, req *userv2.CreateUserRequest) {
				t.Helper()

				if req.GetHuman().GetPasswordType() != nil {
					t.Errorf("PasswordType: want nil, got %T", req.GetHuman().GetPasswordType())
				}
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			zc, u := newUserClient(t)

			tc.in.OrganizationID = "org-1"
			tc.in.Email = "alice@example.com"
			tc.in.EmailVerification = "Verified"

			if _, err := zc.CreateHumanUser(testContext(t), tc.in); err != nil {
				t.Fatalf("\n%s\nCreateHumanUser(...): unexpected error: %v", tc.reason, err)
			}

			if u.created == nil {
				t.Fatalf("\n%s\nCreateHumanUser(...): no request reached Zitadel", tc.reason)
			}

			tc.check(t, u.created)
		})
	}
}

func TestCreateHumanUserRejectsBothPasswords(t *testing.T) {
	zc, u := newUserClient(t)

	_, err := zc.CreateHumanUser(testContext(t), CreateHumanUserInput{
		OrganizationID:    "org-1",
		Email:             "alice@example.com",
		EmailVerification: "Verified",
		Password:          "Str0ng-Passw0rd!",
		HashedPassword:    "JDJhJDEwJCR...",
	})
	if err == nil {
		t.Fatal("CreateHumanUser(...): want an error when both passwords are set, got nil")
	}

	if u.created != nil {
		t.Error("CreateHumanUser(...): a request with both passwords must not reach Zitadel")
	}
}
