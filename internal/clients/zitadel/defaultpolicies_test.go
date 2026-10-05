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
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	policyv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/policy"
	settingsv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/settings"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

// The ten instance wide policies are the same shape repeated: a read that has to
// turn a missing policy into the sentinel the controller turns into a Create, and
// an update that has to treat "this would change nothing" as success.
//
// The second one is the reason this is tested at all. ZITADEL refuses a write
// that changes nothing, and a resource that reconciles the same value every poll
// is exactly that write - so treating the refusal as an error would put every
// instance policy into a permanent error loop.

// policyCase drives one policy through a read.
type policyCase struct {
	reason string

	// seed puts a policy on the fake, which is what the read then returns.
	seed func(*fakeAdmin)
	// get reads it back and returns the client representation.
	get func(context.Context, *Client) (any, error)
	// want is what the read must return for the seeded value.
	want any
}

// cmpPolicy compares the client representations, which are plain structs, and
// reports the difference for a case to print.
func cmpPolicy(want, got any) string {
	return cmp.Diff(want, got)
}

// isNilPointer reports whether a value returned as an any is a nil pointer.
//
// A read that reports a missing policy returns a typed nil, which is not equal
// to a nil any, so a plain comparison against nil would pass a result that is
// not there.
func isNilPointer(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)

	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

func TestDefaultPoliciesRead(t *testing.T) {
	cases := map[string]policyCase{
		"Lockout": {
			reason: "The lockout policy is read with both attempt counts",
			seed: func(a *fakeAdmin) {
				a.lockout = &policyv1.LockoutPolicy{MaxPasswordAttempts: 5, MaxOtpAttempts: 3}
			},
			get:  func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultLockoutPolicy(ctx) },
			want: &DefaultLockoutPolicy{MaxPasswordAttempts: 5, MaxOTPAttempts: 3},
		},
		"Notification": {
			reason: "The notification policy is read",
			seed: func(a *fakeAdmin) {
				a.notification = &policyv1.NotificationPolicy{PasswordChange: true}
			},
			get:  func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultNotificationPolicy(ctx) },
			want: &DefaultNotificationPolicy{PasswordChange: true},
		},
		"PasswordAge": {
			reason: "The password expiry policy is read with both day counts",
			seed: func(a *fakeAdmin) {
				a.passwordAge = &policyv1.PasswordAgePolicy{MaxAgeDays: 90, ExpireWarnDays: 14}
			},
			get:  func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultPasswordAgePolicy(ctx) },
			want: &DefaultPasswordAgePolicy{MaxAgeDays: 90, ExpireWarnDays: 14},
		},
		"PasswordComplexity": {
			reason: "The complexity policy is read with its length and every character class",
			seed: func(a *fakeAdmin) {
				a.passwordComplexity = &policyv1.PasswordComplexityPolicy{
					MinLength: 12, HasUppercase: true, HasLowercase: true, HasNumber: true, HasSymbol: true,
				}
			},
			get: func(ctx context.Context, c *Client) (any, error) {
				return c.GetDefaultPasswordComplexityPolicy(ctx)
			},
			want: &DefaultPasswordComplexityPolicy{
				MinLength: 12, HasUppercase: true, HasLowercase: true, HasNumber: true, HasSymbol: true,
			},
		},
		"Privacy": {
			reason: "The privacy policy is read with all of its links",
			seed: func(a *fakeAdmin) {
				a.privacy = &policyv1.PrivacyPolicy{
					TosLink: "https://example.com/tos", PrivacyLink: "https://example.com/privacy",
					SupportEmail: "help@example.com", CustomLink: "https://example.com/other",
					CustomLinkText: "Other",
				}
			},
			get: func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultPrivacyPolicy(ctx) },
			want: &DefaultPrivacyPolicy{
				TOSLink: "https://example.com/tos", PrivacyLink: "https://example.com/privacy",
				SupportEmail: "help@example.com", CustomLink: "https://example.com/other",
				CustomLinkText: "Other",
			},
		},
		"Domain": {
			reason: "The domain policy is read with all three rules",
			seed: func(a *fakeAdmin) {
				a.domain = &policyv1.DomainPolicy{
					UserLoginMustBeDomain: true, ValidateOrgDomains: true,
					SmtpSenderAddressMatchesInstanceDomain: true,
				}
			},
			get: func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultDomainPolicy(ctx) },
			want: &DefaultDomainPolicy{
				UserLoginMustBeDomain: true, ValidateOrgDomains: true,
				SMTPSenderAddressMatchesInstanceDomain: true,
			},
		},
		"Label": {
			reason: "The label policy is read with its colours and flags",
			seed: func(a *fakeAdmin) {
				a.label = &policyv1.LabelPolicy{
					PrimaryColor: "#fff", WarnColor: "#f00", HideLoginNameSuffix: true,
					ThemeMode: policyv1.ThemeMode_THEME_MODE_DARK,
				}
			},
			get: func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultLabelPolicy(ctx) },
			want: &DefaultLabelPolicy{
				PrimaryColor: "#fff", WarnColor: "#f00", HideLoginNameSuffix: true, ThemeMode: "Dark",
			},
		},
		"OIDCSettings": {
			reason: "The OIDC settings are read with all four lifetimes",
			seed: func(a *fakeAdmin) {
				a.oidcSettings = &settingsv1.OIDCSettings{
					AccessTokenLifetime:        durationpb.New(time.Hour),
					IdTokenLifetime:            durationpb.New(2 * time.Hour),
					RefreshTokenExpiration:     durationpb.New(3 * time.Hour),
					RefreshTokenIdleExpiration: durationpb.New(4 * time.Hour),
				}
			},
			get: func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultOIDCSettings(ctx) },
			want: &DefaultOIDCSettings{
				AccessTokenLifetime: "1h0m0s", IDTokenLifetime: "2h0m0s",
				RefreshTokenExpiration: "3h0m0s", RefreshTokenIdleExpiration: "4h0m0s",
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			zc, a := newAdminClient(t)
			tc.seed(a)

			got, err := tc.get(testContext(t), zc)
			if err != nil {
				t.Fatalf("\n%s\nget: unexpected error: %v", tc.reason, err)
			}

			if diff := cmpPolicy(tc.want, got); diff != "" {
				t.Errorf("\n%s\nget: -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

// TestDefaultPoliciesReadFailures covers the read side for every policy at once,
// because they all answer the same way: a missing policy is the sentinel, and
// anything else is its own error.
func TestDefaultPoliciesReadFailures(t *testing.T) {
	reads := map[string]func(context.Context, *Client) (any, error){
		"Lockout":            func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultLockoutPolicy(ctx) },
		"Notification":       func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultNotificationPolicy(ctx) },
		"PasswordAge":        func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultPasswordAgePolicy(ctx) },
		"PasswordComplexity": func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultPasswordComplexityPolicy(ctx) },
		"Privacy":            func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultPrivacyPolicy(ctx) },
		"Domain":             func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultDomainPolicy(ctx) },
		"Label":              func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultLabelPolicy(ctx) },
		"OIDCSettings":       func(ctx context.Context, c *Client) (any, error) { return c.GetDefaultOIDCSettings(ctx) },
	}

	for name, read := range reads {
		t.Run(name+"/Missing", func(t *testing.T) {
			zc, _ := newAdminClient(t)

			// Nothing seeded: the fake answers with a response carrying no policy,
			// which is how a ZITADEL without the setting reports it.
			got, err := read(testContext(t), zc)
			if !isNilPointer(got) {
				t.Errorf("a missing policy returned %v, want nil", got)
			}

			if !IsNotFound(err) {
				t.Errorf("a missing policy returned %v, want the not-found sentinel", err)
			}
		})

		t.Run(name+"/Failure", func(t *testing.T) {
			zc, a := newAdminClient(t)
			a.getErr = status.Error(codes.PermissionDenied, "nope")

			if _, err := read(testContext(t), zc); err == nil {
				t.Error("an unrelated read failure was swallowed")
			} else if IsNotFound(err) {
				t.Error("a permission failure was reported as a missing policy, which would trigger a Create")
			}
		})
	}
}

// TestDefaultPoliciesWrite covers the write side for every policy, including the
// refusal that means "this would change nothing".
func TestDefaultPoliciesWrite(t *testing.T) {
	writes := map[string]struct {
		write string
		do    func(context.Context, *Client) error
	}{
		"Lockout": {
			write: "update-lockout",
			do: func(ctx context.Context, c *Client) error {
				return c.SetDefaultLockoutPolicy(ctx, DefaultLockoutPolicyInput{MaxPasswordAttempts: 5, MaxOTPAttempts: 3})
			},
		},
		"Notification": {
			write: "update-notification",
			do: func(ctx context.Context, c *Client) error {
				return c.SetDefaultNotificationPolicy(ctx, DefaultNotificationPolicyInput{PasswordChange: true})
			},
		},
		"PasswordAge": {
			write: "update-password-age",
			do: func(ctx context.Context, c *Client) error {
				return c.SetDefaultPasswordAgePolicy(ctx, DefaultPasswordAgePolicyInput{MaxAgeDays: 90, ExpireWarnDays: 14})
			},
		},
		"PasswordComplexity": {
			write: "update-password-complexity",
			do: func(ctx context.Context, c *Client) error {
				return c.SetDefaultPasswordComplexityPolicy(ctx, DefaultPasswordComplexityPolicyInput{
					MinLength: 12, HasUppercase: true,
				})
			},
		},
		"Privacy": {
			write: "update-privacy",
			do: func(ctx context.Context, c *Client) error {
				return c.SetDefaultPrivacyPolicy(ctx, DefaultPrivacyPolicyInput{TOSLink: "https://example.com/tos"})
			},
		},
		"Domain": {
			write: "update-domain",
			do: func(ctx context.Context, c *Client) error {
				return c.SetDefaultDomainPolicy(ctx, DefaultDomainPolicyInput{ValidateOrgDomains: true})
			},
		},
		"Label": {
			write: "update-label",
			do: func(ctx context.Context, c *Client) error {
				return c.SetDefaultLabelPolicy(ctx, DefaultLabelPolicyInput{PrimaryColor: "#fff"})
			},
		},
		"OIDCSettings": {
			write: "update-oidc-settings",
			do: func(ctx context.Context, c *Client) error {
				return c.SetDefaultOIDCSettings(ctx, DefaultOIDCSettingsInput{AccessTokenLifetime: "1h"})
			},
		},
	}

	for name, tc := range writes {
		t.Run(name+"/Succeeds", func(t *testing.T) {
			zc, a := newAdminClient(t)

			if err := tc.do(testContext(t), zc); err != nil {
				t.Fatalf("set: unexpected error: %v", err)
			}

			if got := a.lastWrite(); got != tc.write {
				t.Errorf("set: want the %q call, got %q", tc.write, got)
			}
		})

		t.Run(name+"/NoChangeIsSuccess", func(t *testing.T) {
			// The case that keeps these resources from looping: ZITADEL refuses
			// a write that changes nothing, and a resource reconciling the same
			// value every poll is exactly that write.
			zc, a := newAdminClient(t)
			a.updateErr = noChanges

			if err := tc.do(testContext(t), zc); err != nil {
				t.Errorf("set: a write ZITADEL refused as a no-op was reported as an error: %v", err)
			}
		})

		t.Run(name+"/FailureIsReported", func(t *testing.T) {
			zc, a := newAdminClient(t)
			a.updateErr = status.Error(codes.PermissionDenied, "nope")

			if err := tc.do(testContext(t), zc); err == nil {
				t.Error("set: a permission failure was swallowed")
			}
		})
	}
}

// TestDefaultPolicyRoundTrip checks that what a write sends is what a read gets
// back, which is the only way to know the two sides agree on the field names.
func TestDefaultPolicyRoundTrip(t *testing.T) {
	zc, _ := newAdminClient(t)
	ctx := testContext(t)

	in := DefaultLockoutPolicyInput{MaxPasswordAttempts: 7, MaxOTPAttempts: 2}

	if err := zc.SetDefaultLockoutPolicy(ctx, in); err != nil {
		t.Fatalf("SetDefaultLockoutPolicy(...): unexpected error: %v", err)
	}

	got, err := zc.GetDefaultLockoutPolicy(ctx)
	if err != nil {
		t.Fatalf("GetDefaultLockoutPolicy(...): unexpected error: %v", err)
	}

	// The read returns the observed policy rather than the input type; the two
	// carry the same fields, which is the point of the round trip.
	want := &DefaultLockoutPolicy{MaxPasswordAttempts: in.MaxPasswordAttempts, MaxOTPAttempts: in.MaxOTPAttempts}

	if diff := cmpPolicy(want, got); diff != "" {
		t.Errorf("round trip: -want, +got:\n%s", diff)
	}
}
