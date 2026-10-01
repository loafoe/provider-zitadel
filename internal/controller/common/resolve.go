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
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reference"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
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
