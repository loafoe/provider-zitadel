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

package common

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
)

// ExtractOrganizationID extracts the Zitadel organization ID from the status of
// an Organization.
func ExtractOrganizationID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		org, ok := mg.(*v1alpha1.Organization)
		if !ok {
			return ""
		}

		if org.Status.AtProvider.ID == nil {
			return ""
		}

		return *org.Status.AtProvider.ID
	}
}

// ExtractProjectID extracts the Zitadel project ID from the status of a
// Project.
func ExtractProjectID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		p, ok := mg.(*v1alpha1.Project)
		if !ok {
			return ""
		}

		if p.Status.AtProvider.ID == nil {
			return ""
		}

		return *p.Status.AtProvider.ID
	}
}

// ExtractUserID extracts the Zitadel user ID from the status of a HumanUser or
// a ServiceAccount. Both kinds publish their user ID as `status.atProvider.id`.
func ExtractUserID() reference.ExtractValueFn {
	return func(mg resource.Managed) string {
		switch u := mg.(type) {
		case *v1alpha1.HumanUser:
			if u.Status.AtProvider.ID == nil {
				return ""
			}
			return *u.Status.AtProvider.ID
		case *v1alpha1.ServiceAccount:
			if u.Status.AtProvider.ID == nil {
				return ""
			}
			return *u.Status.AtProvider.ID
		default:
			return ""
		}
	}
}

// ResolveOrganizationID resolves the organization a managed resource belongs
// to from the supplied organizationRef, organizationSelector, direct
// organizationID or the ProviderConfig default.
//
// currentValue is the organization ID the resource already resolved to - it is
// passed to the resolver so that an already created resource is not silently
// re-pointed at a different organization by a selector that now matches more
// than one Organization.
func ResolveOrganizationID(ctx context.Context, kube client.Client, mg resource.ModernManaged, ref *xpv1.Reference, selector *xpv1.Selector, direct *string, providerDefault, currentValue string) (string, error) {
	switch {
	case ref != nil || selector != nil:
		rsp, err := reference.NewAPIResolver(kube, mg).Resolve(ctx, reference.ResolutionRequest{
			CurrentValue: currentValue,
			Reference:    ref,
			Selector:     selector,
			To: reference.To{
				Managed: &v1alpha1.Organization{},
				List:    &v1alpha1.OrganizationList{},
			},
			Extract:   ExtractOrganizationID(),
			Namespace: mg.GetNamespace(),
		})

		if err != nil {
			return "", errors.Join(ErrResolveOrganization, err)
		}

		return rsp.ResolvedValue, nil

	case direct != nil:
		return *direct, nil

	default:
		return providerDefault, nil
	}
}

// ResolveProjectID resolves the project a managed resource belongs to from the
// supplied projectRef, projectSelector or direct projectID.
func ResolveProjectID(ctx context.Context, kube client.Client, mg resource.ModernManaged, ref *xpv1.Reference, selector *xpv1.Selector, direct *string, currentValue string) (string, error) {
	if ref == nil && selector == nil {
		if direct == nil {
			return "", errors.Join(ErrNoProjectID, errors.New("either projectID, projectRef or projectSelector must be set"))
		}

		return *direct, nil
	}

	rsp, err := reference.NewAPIResolver(kube, mg).Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: currentValue,
		Reference:    ref,
		Selector:     selector,
		To: reference.To{
			Managed: &v1alpha1.Project{},
			List:    &v1alpha1.ProjectList{},
		},
		Extract:   ExtractProjectID(),
		Namespace: mg.GetNamespace(),
	})
	if err != nil {
		return "", errors.Join(ErrResolveProject, err)
	}

	return rsp.ResolvedValue, nil
}

// userKinds is the list of kinds a user reference can point at, in resolution
// order. A Personal Access Token is commonly issued for a machine user, so
// service accounts are tried first.
var userKinds = []struct {
	managed resource.Managed
	list    resource.ManagedList
}{
	{managed: &v1alpha1.ServiceAccount{}, list: &v1alpha1.ServiceAccountList{}},
	{managed: &v1alpha1.HumanUser{}, list: &v1alpha1.HumanUserList{}},
}

// ResolveUserID resolves the user a managed resource refers to from the supplied
// userRef, userSelector or direct userID.
//
// A reference may point at either a HumanUser or a ServiceAccount, because both
// kinds are Zitadel users and both can own a Personal Access Token. The
// reference is therefore resolved against every supported kind in turn.
func ResolveUserID(ctx context.Context, kube client.Client, mg resource.ModernManaged, ref *xpv1.Reference, selector *xpv1.Selector, direct *string, currentValue string) (string, error) {
	if ref == nil && selector == nil {
		if direct == nil {
			return "", errors.Join(ErrNoUserID, errors.New("either userID, userRef or userSelector must be set"))
		}

		return *direct, nil
	}

	resolver := reference.NewAPIResolver(kube, mg)

	var lastErr error

	for _, kind := range userKinds {
		rsp, err := resolver.Resolve(ctx, reference.ResolutionRequest{
			CurrentValue: currentValue,
			Reference:    ref,
			Selector:     selector,
			To: reference.To{
				Managed: kind.managed,
				List:    kind.list,
			},
			Extract:   ExtractUserID(),
			Namespace: mg.GetNamespace(),
		})
		if err == nil && rsp.ResolvedValue != "" {
			return rsp.ResolvedValue, nil
		}

		if err != nil {
			lastErr = err
		}
	}

	if lastErr != nil {
		return "", errors.Join(ErrResolveUser, lastErr)
	}

	return "", errors.Join(ErrResolveUser, errors.New("the user reference does not match a HumanUser or a ServiceAccount managed by this provider"))
}

// ParseTime parses an RFC3339 timestamp as returned by the Zitadel client. It
// returns nil for empty strings.
func ParseTime(s string) *metav1.Time {
	if s == "" {
		return nil
	}

	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil
	}

	return &metav1.Time{Time: t}
}

// FormatTime renders a Kubernetes timestamp as an RFC3339 string. It returns an
// empty string for nil timestamps.
func FormatTime(t *metav1.Time) string {
	if t == nil {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}

// StringPtr returns a pointer to the supplied string. It is handy when
// building the desired state for a status field.
func StringPtr(s string) *string { return &s }

// BoolPtr returns a pointer to the supplied bool.
func BoolPtr(b bool) *bool { return &b }

// Bool dereferences a bool pointer, returning false for nil.
func Bool(b *bool) bool {
	if b == nil {
		return false
	}

	return *b
}

// Ptr dereferences a string pointer, returning an empty string for nil.
func Deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// Value dereferences a pointer to a string-like type, returning an empty
// string for nil. It is used for the typed string enums of the API.
func Value[T ~string](p *T) string {
	if p == nil {
		return ""
	}

	return string(*p)
}

// EqualStringSlices reports whether two string slices contain the same
// elements in the same order. A nil slice and an empty slice are considered
// equal, which mirrors how Zitadel reports unset list fields.
func EqualStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// IsReferenceGone reports whether err means a referenced resource no longer
// exists. Deletion has to tolerate this: by the time a dependent resource is
// removed its organization, project or user may already be gone, and then
// there is nothing left to detach from.
func IsReferenceGone(err error) bool {
	if err == nil {
		return false
	}

	if kerrors.IsNotFound(err) {
		return true
	}

	// The crossplane reference package wraps a NotFound from the API server
	// rather than returning it directly.
	return strings.Contains(err.Error(), "cannot get referenced resource") ||
		strings.Contains(err.Error(), "referenced field was empty")
}

// ResolveServiceAccountID resolves a reference that must point at a
// ServiceAccount specifically, as opposed to ResolveUserID which also accepts a
// HumanUser. A machine key, for instance, can only belong to a machine user.
func ResolveServiceAccountID(ctx context.Context, kube client.Client, mg resource.ModernManaged, ref *xpv1.Reference, selector *xpv1.Selector, direct *string, currentValue string) (string, error) {
	if ref == nil && selector == nil {
		if direct == nil {
			return "", errors.Join(ErrNoUserID, errors.New("either serviceAccountID, serviceAccountRef or serviceAccountSelector must be set"))
		}

		return *direct, nil
	}

	rsp, err := reference.NewAPIResolver(kube, mg).Resolve(ctx, reference.ResolutionRequest{
		CurrentValue: currentValue,
		Reference:    ref,
		Selector:     selector,
		To: reference.To{
			Managed: &v1alpha1.ServiceAccount{},
			List:    &v1alpha1.ServiceAccountList{},
		},
		Extract:   ExtractUserID(),
		Namespace: mg.GetNamespace(),
	})
	if err != nil {
		return "", errors.Join(ErrNoUserID, err)
	}

	return rsp.ResolvedValue, nil
}

// Resolution and deletion
//
// Two rules keep a reference from turning into a stuck object:
//
//   - Resolution never starts from the observed status. Re-resolving from the
//     spec is what lets a dependent notice that its referenced resource was
//     deleted and recreated, which gives it a new identity. Handing the resolver
//     the previously observed ID would short-circuit it, leaving the dependent
//     pointing at a resource that no longer exists.
//   - Deletion never resolves at all. While an object is terminating the
//     crossplane reference resolver refuses to re-resolve, and a resource whose
//     Create never succeeded has nothing to resolve from. Deletion therefore uses
//     the identifiers already recorded in the status, and lets the finalizer go
//     when there are none.

// ResolveServiceAccount resolves a reference to a ServiceAccount and returns the
// service account's ID together with the organization that owns it.
//
// Zitadel lists a user's machine keys through an organization scoped API, so a
// caller that has to read a service account's keys needs both identifiers, and
// resolving twice would be wasteful.
func ResolveServiceAccount(ctx context.Context, kube client.Client, mg resource.ModernManaged,
	ref *xpv1.Reference, selector *xpv1.Selector, direct *string,
) (serviceAccountID, orgID string, err error) {
	if ref == nil && selector == nil {
		if direct == nil {
			return "", "", errors.Join(ErrNoUserID, errors.New("either serviceAccountID, serviceAccountRef or serviceAccountSelector must be set"))
		}

		return *direct, "", nil
	}

	// The resolver hands back a single value, so the organization is captured
	// as a side effect of extracting the ID.
	rsp, err := reference.NewAPIResolver(kube, mg).Resolve(ctx, reference.ResolutionRequest{
		Reference: ref,
		Selector:  selector,
		To: reference.To{
			Managed: &v1alpha1.ServiceAccount{},
			List:    &v1alpha1.ServiceAccountList{},
		},
		Extract: func(mg resource.Managed) string {
			sa, ok := mg.(*v1alpha1.ServiceAccount)
			if !ok {
				return ""
			}

			orgID = Deref(sa.Status.AtProvider.OrganizationID)

			return Deref(sa.Status.AtProvider.ID)
		},
		Namespace: mg.GetNamespace(),
	})
	if err != nil {
		return "", "", errors.Join(ErrResolveUser, err)
	}

	return rsp.ResolvedValue, orgID, nil
}

// CurrentIfDeleting returns the observed identifier when the object is
// terminating, and nothing otherwise.
//
// Resolution normally starts from the spec, so that a dependent notices when
// its referenced resource was replaced and has to follow the new one. A
// terminating object is the exception: it exists to remove what was created, so
// it has to act on the identity it recorded. Following a reference that has
// moved on would either delete the wrong thing, or - when the new subject
// happens to hold state that looks like this resource's - delete forever without
// ever letting the finalizer go.
func CurrentIfDeleting(deleting bool, observed *string) string {
	if !deleting {
		return ""
	}

	return Deref(observed)
}

// DerefBool returns the value p points at, or false when p is nil.
func DerefBool(p *bool) bool {
	if p == nil {
		return false
	}

	return *p
}

// DerefInt64 returns the value p points at, or zero when p is nil.
func DerefInt64(p *int64) int64 {
	if p == nil {
		return 0
	}

	return *p
}

// Int64Ptr returns a pointer to i.
func Int64Ptr(i int64) *int64 { return &i }

// ToUint32 narrows a count from a spec into the uint32 the Zitadel API accepts.
//
// Every such field is an attempt count, a day count or a minimum length, which
// no real configuration approaches the limit. Saturating rather than wrapping is
// what makes a nonsensical value visible as an extreme setting rather than as a
// small plausible one, and the CRD constrains the field to a non-negative value
// so the ordinary range is unaffected.
func ToUint32(v int64) uint32 {
	if v < 0 {
		return 0
	}

	if v > math.MaxUint32 {
		return math.MaxUint32
	}

	return uint32(v) //nolint:gosec // Bounded by the checks above.
}

// enumNames projects a slice of a named string type onto its plain values, so
// that a spec field can be compared with the API spelling of the same setting
// without the comparison depending on the generated type.
//
// A nil or empty slice becomes an empty, non-nil one, so that "not configured"
// and "configured with nothing" compare the same way.
func EnumNames[T ~string](values []T) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, string(v))
	}

	return out
}

// enumValues is enumNames in the other direction, for writing what Zitadel
// reports back into a status field.
func EnumValues[T ~string](values []string) []T {
	out := make([]T, 0, len(values))
	for _, v := range values {
		out = append(out, T(v))
	}

	return out
}

// ResolveTargetID resolves a reference that must point at an ActionTarget.
//
// A target is an instance wide object, so this is the one place where a
// reference is resolved against a kind rather than by pinning an ID: the
// reference is deliberately narrow, so a key cannot be attached to a target the
// provider does not manage.
func ResolveTargetID(ctx context.Context, kube client.Client, mg resource.ModernManaged, ref *xpv1.Reference, selector *xpv1.Selector, direct *string) (string, error) {
	if ref == nil && selector == nil {
		if direct == nil {
			return "", errors.Join(ErrNoUserID, errors.New("either targetID, targetRef or targetSelector must be set"))
		}

		return *direct, nil
	}

	rsp, err := reference.NewAPIResolver(kube, mg).Resolve(ctx, reference.ResolutionRequest{
		Reference: ref,
		Selector:  selector,
		To: reference.To{
			Managed: &v1alpha1.ActionTarget{},
			List:    &v1alpha1.ActionTargetList{},
		},
		Extract: func(mg resource.Managed) string {
			t, ok := mg.(*v1alpha1.ActionTarget)
			if !ok {
				return ""
			}

			return t.Status.AtProvider.ID
		},
		Namespace: mg.GetNamespace(),
	})
	if err != nil {
		return "", errors.Join(ErrResolveUser, err)
	}

	return rsp.ResolvedValue, nil
}
