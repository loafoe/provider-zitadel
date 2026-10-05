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
	"errors"
	"math"
	"strings"
	"testing"

	policyv1 "github.com/zitadel/zitadel-go/v3/pkg/client/zitadel/policy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Values the cases share, named so that a case reads as a statement about
// behaviour rather than about a repeated literal.
const (
	themeAuto = "Auto"
)

// setPolicy is the only piece of real logic in the policy layer, and it is what
// keeps a managed policy from getting stuck. Zitadel exposes "add" and "update"
// as separate calls and refuses either one in a situation the other accepts, so
// the write has to be able to fall back from one to the other in both
// directions.
//
// The fallback matters because of the read that picks the first call. An
// organization can be reported as still on the instance default while already
// holding a custom policy, and an add then fails with AlreadyExists. If the
// write did not correct itself, that organization would fail forever.

// recorded is one call a policy write made.
type recorded struct {
	add    bool
	update bool
}

// calls builds the add and update closures setPolicy is handed, recording which
// one ran and returning the supplied error for it.
func calls(log *recorded, addErr, updateErr error) (func() error, func() error) {
	add := func() error {
		log.add = true

		return addErr
	}

	update := func() error {
		log.update = true

		return updateErr
	}

	return add, update
}

func TestSetPolicy(t *testing.T) {
	// The errors Zitadel uses to say the other call is needed.
	alreadyExists := status.Error(codes.AlreadyExists, "already exists")
	alreadyExistsSensed := status.Error(codes.FailedPrecondition, "Error AlreadyExists: custom policy exists")
	notFound := status.Error(codes.NotFound, "policy not found")
	noChanges := status.Error(codes.FailedPrecondition, "No changes")

	other := errors.New("something else went wrong")

	cases := map[string]struct {
		reason string
		// inherited is what the read said about the policy.
		inherited bool
		addErr    error
		updateErr error
		// want is the call the write is expected to end up making.
		want recorded
		// wantErr is the error the write is expected to report.
		wantErr error
	}{
		"InheritedAddSucceeds": {
			reason:    "An organization on the instance default gets an add, and it works",
			inherited: true,
			want:      recorded{add: true},
		},
		"InheritedAddAlreadyExists": {
			reason:    "An add refused because the policy already exists falls back to an update",
			inherited: true,
			addErr:    alreadyExists,
			want:      recorded{add: true, update: true},
		},
		"InheritedAddAlreadyExistsSensed": {
			reason:    "The AlreadyExists sense is honoured even when the code is a precondition",
			inherited: true,
			addErr:    alreadyExistsSensed,
			want:      recorded{add: true, update: true},
		},
		"InheritedAddThenUpdateFails": {
			reason:    "A failure of the fallback is reported",
			inherited: true,
			addErr:    alreadyExists,
			updateErr: other,
			want:      recorded{add: true, update: true},
			wantErr:   other,
		},
		"InheritedNoChange": {
			reason:    "A write Zitadel refused because it changes nothing is success",
			inherited: true,
			addErr:    noChanges,
			want:      recorded{add: true},
		},
		"InheritedOtherError": {
			reason:    "An unrelated add failure is reported without falling back",
			inherited: true,
			addErr:    other,
			want:      recorded{add: true},
			wantErr:   other,
		},
		"CustomUpdateSucceeds": {
			reason: "An organization holding a custom policy gets an update, and it works",
			want:   recorded{update: true},
		},
		"CustomUpdateNotFound": {
			reason:    "An update refused because there is nothing to update falls back to an add",
			addErr:    nil,
			updateErr: notFound,
			want:      recorded{update: true, add: true},
		},
		"CustomUpdateNoChange": {
			reason:    "An update Zitadel refused because it changes nothing is success",
			updateErr: noChanges,
			want:      recorded{update: true},
		},
		"CustomOtherError": {
			reason:    "An unrelated update failure is reported without falling back",
			updateErr: other,
			want:      recorded{update: true},
			wantErr:   other,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var log recorded

			add, update := calls(&log, tc.addErr, tc.updateErr)

			err := setPolicy(tc.inherited, add, update)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("\n%s\nsetPolicy(...): want error %v, got %v", tc.reason, tc.wantErr, err)
			}

			if log != tc.want {
				t.Errorf("\n%s\nsetPolicy(...): want %+v, got %+v", tc.reason, tc.want, log)
			}
		})
	}
}

// TestSetPolicyNeverLoops checks that a fallback is attempted at most once. A
// policy write that retried both ways could not terminate against an instance
// that fails every call, and would turn one bad request into a reconcile loop.
func TestSetPolicyNeverLoops(t *testing.T) {
	notFound := status.Error(codes.NotFound, "gone")
	alreadyExists := status.Error(codes.AlreadyExists, "exists")

	t.Run("Add", func(t *testing.T) {
		var adds, updates int

		err := setPolicy(true,
			func() error { adds++; return alreadyExists },
			func() error { updates++; return alreadyExists },
		)
		if err == nil {
			t.Error("setPolicy(...): want the second failure reported, got nil")
		}

		if adds != 1 || updates != 1 {
			t.Errorf("setPolicy(...): want one add and one update, got %d and %d", adds, updates)
		}
	})

	t.Run("Update", func(t *testing.T) {
		var adds, updates int

		err := setPolicy(false,
			func() error { adds++; return notFound },
			func() error { updates++; return notFound },
		)
		if err == nil {
			t.Error("setPolicy(...): want the second failure reported, got nil")
		}

		if adds != 1 || updates != 1 {
			t.Errorf("setPolicy(...): want one add and one update, got %d and %d", adds, updates)
		}
	})
}

// TestBoundedUint32 checks the saturation of the attempt counts read back from
// Zitadel.
//
// Every one of these numbers is an attempt count or a number of days, so a value
// beyond uint32 only ever means the server said something unexpected. Saturating
// keeps that visible as an extreme setting; wrapping would turn it into a small
// plausible one, and the controller would report permanent drift against a value
// the user never wrote.
func TestBoundedUint32(t *testing.T) {
	cases := map[string]struct {
		reason string
		in     uint64
		want   uint32
	}{
		"Zero":  {reason: "Zero is zero", in: 0, want: 0},
		"Small": {reason: "A plausible value is unchanged", in: 5, want: 5},
		"MaxUint32": {
			reason: "The largest representable value is unchanged",
			in:     math.MaxUint32,
			want:   math.MaxUint32,
		},
		"MaxUint32PlusOne": {
			reason: "One past the maximum saturates rather than wrapping to zero",
			in:     math.MaxUint32 + 1,
			want:   math.MaxUint32,
		},
		"MaxUint64": {
			reason: "The largest possible value saturates",
			in:     math.MaxUint64,
			want:   math.MaxUint32,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := boundedUint32(tc.in); got != tc.want {
				t.Errorf("\n%s\nboundedUint32(%d): want %d, got %d", tc.reason, tc.in, tc.want, got)
			}
		})
	}
}

// TestLabelThemeMode covers the label policy theme mapping.
//
// The generated enum's String method returns the protobuf constant name rather
// than the API spelling, so using it would make a label policy report drift on
// every reconcile against a value the operator never changed.
func TestLabelThemeMode(t *testing.T) {
	fromCases := map[string]struct {
		reason string
		in     policyv1.ThemeMode
		want   string
	}{
		"Light":        {reason: "Light round trips", in: policyv1.ThemeMode_THEME_MODE_LIGHT, want: "Light"},
		"Dark":         {reason: "Dark round trips", in: policyv1.ThemeMode_THEME_MODE_DARK, want: "Dark"},
		themeAuto:      {reason: "Auto round trips", in: policyv1.ThemeMode_THEME_MODE_AUTO, want: themeAuto},
		"Unspecified":  {reason: "An unset mode follows the system preference", in: policyv1.ThemeMode_THEME_MODE_UNSPECIFIED, want: themeAuto},
		"UnknownValue": {reason: "An unknown mode follows the system preference", in: policyv1.ThemeMode(99), want: themeAuto},
	}

	for name, tc := range fromCases {
		t.Run("From/"+name, func(t *testing.T) {
			if got := LabelThemeModeFromProto(tc.in); got != tc.want {
				t.Errorf("\n%s\nLabelThemeModeFromProto(...): want %q, got %q", tc.reason, tc.want, got)
			}
		})
	}

	toCases := map[string]struct {
		reason  string
		in      string
		want    policyv1.ThemeMode
		wantErr bool
	}{
		"Light":   {reason: "Light maps to light", in: "Light", want: policyv1.ThemeMode_THEME_MODE_LIGHT},
		"Dark":    {reason: "Dark maps to dark", in: "Dark", want: policyv1.ThemeMode_THEME_MODE_DARK},
		themeAuto: {reason: "Auto maps to auto", in: themeAuto, want: policyv1.ThemeMode_THEME_MODE_AUTO},
		"Empty": {
			// Auto is what Zitadel applies for an unset mode, and the reverse
			// mapping reports Auto for an unspecified enum, so defaulting here
			// is what keeps a policy that omits the field from reporting drift.
			reason: "An empty mode defaults to Auto rather than being rejected",
			in:     "",
			want:   policyv1.ThemeMode_THEME_MODE_AUTO,
		},
		"Unknown": {reason: "An unknown mode is rejected", in: "Neon", want: policyv1.ThemeMode_THEME_MODE_UNSPECIFIED, wantErr: true},
	}

	for name, tc := range toCases {
		t.Run("To/"+name, func(t *testing.T) {
			got, err := LabelThemeModeToProto(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\nLabelThemeModeToProto(%q): wantErr %v, got %v", tc.reason, tc.in, tc.wantErr, err)
			}

			if got != tc.want {
				t.Errorf("\n%s\nLabelThemeModeToProto(%q): want %v, got %v", tc.reason, tc.in, tc.want, got)
			}
		})
	}
}

// TestErrorClassification covers the predicates every controller branches on.
//
// Getting any of these wrong is what puts a resource into a permanent error
// loop: a not-found read that is not recognised as one means Create is never
// called, and a no-change write that is not recognised means Update fails
// forever against a resource that is already correct.
func TestErrorClassification(t *testing.T) {
	// Zitadel spells "nothing changed" differently per endpoint, and each
	// spelling has to be recognised.
	noChangeMessages := []string{
		"No changes",
		"NotChanged",
		"Error NotChanged: no changes",
		"custom policy has not been changed",
		"action is not inactive",
	}

	cases := map[string]struct {
		reason     string
		err        error
		wantFound  bool
		wantUnch   bool
		wantExist  bool
		wantUnauth bool
	}{
		"Nil": {
			reason: "A nil error is none of them",
		},
		"NotFound": {
			reason:    "A NotFound status is a missing resource",
			err:       status.Error(codes.NotFound, "project not found"),
			wantFound: true,
		},
		"SentinelNotFound": {
			reason:    "The package sentinel is a missing resource",
			err:       ErrNotFound,
			wantFound: true,
		},
		"WrappedNotFound": {
			reason:    "A wrapped sentinel is a missing resource",
			err:       errors.Join(errors.New("while getting"), ErrNotFound),
			wantFound: true,
		},
		"PermissionDenied": {
			reason: "A denial is not a missing resource",
			err:    status.Error(codes.PermissionDenied, "nope"),
		},
		"AlreadyExists": {
			reason:    "An AlreadyExists status is a conflict",
			err:       status.Error(codes.AlreadyExists, "exists"),
			wantExist: true,
		},
		"AlreadyExistsSensed": {
			reason:    "The AlreadyExists sense is recognised behind another code",
			err:       status.Error(codes.FailedPrecondition, "Error AlreadyExists"),
			wantExist: true,
		},
		"Unauthenticated": {
			reason:     "An Unauthenticated status is a rejected credential",
			err:        status.Error(codes.Unauthenticated, "token invalid"),
			wantUnauth: true,
		},
		"PlainError": {
			reason: "A plain error is none of them",
			err:    errors.New("boom"),
		},
	}

	for _, msg := range noChangeMessages {
		cases["NoChange/"+msg] = struct {
			reason     string
			err        error
			wantFound  bool
			wantUnch   bool
			wantExist  bool
			wantUnauth bool
		}{
			reason:   "Zitadel's spelling of a no-op write is recognised: " + msg,
			err:      status.Error(codes.FailedPrecondition, msg),
			wantUnch: true,
		}
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := IsNotFound(tc.err); got != tc.wantFound {
				t.Errorf("\n%s\nIsNotFound(...): want %v, got %v", tc.reason, tc.wantFound, got)
			}

			if got := IsNotChanged(tc.err); got != tc.wantUnch {
				t.Errorf("\n%s\nIsNotChanged(...): want %v, got %v", tc.reason, tc.wantUnch, got)
			}

			if got := IsAlreadyExists(tc.err); got != tc.wantExist {
				t.Errorf("\n%s\nIsAlreadyExists(...): want %v, got %v", tc.reason, tc.wantExist, got)
			}

			if got := IsInvalidToken(tc.err); got != tc.wantUnauth {
				t.Errorf("\n%s\nIsInvalidToken(...): want %v, got %v", tc.reason, tc.wantUnauth, got)
			}
		})
	}
}

// TestWrapError checks that the one authentication mistake worth naming gets
// named, and that nothing else is disturbed on its way to the reconcile error.
func TestWrapError(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		if err := WrapError(nil); err != nil {
			t.Errorf("WrapError(nil): want nil, got %v", err)
		}
	})

	t.Run("Rejection", func(t *testing.T) {
		original := status.Error(codes.Unauthenticated, "Errors.Token.Invalid")

		got := WrapError(original)
		if got == nil {
			t.Fatal("WrapError(...): want an error, got nil")
		}

		if !strings.Contains(got.Error(), "Jwt") {
			t.Errorf("WrapError(...): want the hint to name the access token type, got %q", got.Error())
		}

		// The original has to survive, or the status code that callers branch on
		// is lost.
		if !IsInvalidToken(got) {
			t.Error("WrapError(...): the wrapped error is no longer recognised as a credential rejection")
		}
	})

	t.Run("Other", func(t *testing.T) {
		original := errors.New("something else")

		got := WrapError(original)

		// Anything that is not a credential rejection is handed back exactly as
		// it came in, so its type and its message are both preserved.
		if got != original { //nolint:errorlint // identity is the assertion here.
			t.Errorf("WrapError(...): want the error unchanged, got %v", got)
		}
	})
}
