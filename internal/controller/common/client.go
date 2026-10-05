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

// Package common contains helpers shared by the Zitadel controllers: building
// an API client from a ProviderConfig, and resolving cross-resource
// references.
package common

import (
	"context"
	stderrors "errors"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apisv1alpha1 "github.com/loafoe/provider-zitadel/apis/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

const (
	// ErrGetProviderConfig is returned when the ProviderConfig of a managed
	// resource cannot be read.
	ErrGetProviderConfig = "cannot get ProviderConfig"
)

// Errors returned while building a Zitadel client.
var (
	// ErrUnsupportedProviderConfigKind is returned when a managed resource
	// references a ProviderConfig kind this provider does not support.
	ErrUnsupportedProviderConfigKind = stderrors.New("unsupported provider config kind")

	// ErrGetCredentials is returned when the credentials of a ProviderConfig
	// cannot be read.
	ErrGetCredentials = stderrors.New("cannot get ProviderConfig credentials")

	// ErrNewClient is returned when the Zitadel client cannot be created.
	ErrNewClient = stderrors.New("cannot create Zitadel client")
)

// Errors returned while resolving cross-resource references.
var (
	// ErrResolveOrganization is returned when a reference to an Organization
	// cannot be resolved.
	ErrResolveOrganization = errors.New("cannot resolve organization reference")

	// ErrResolveProject is returned when a reference to a Project cannot be
	// resolved.
	ErrResolveProject = errors.New("cannot resolve project reference")

	// ErrResolveUser is returned when a reference to a user cannot be resolved.
	ErrResolveUser = errors.New("cannot resolve user reference")

	// ErrNoProjectID is returned when the project a resource belongs to cannot
	// be determined.
	ErrNoProjectID = errors.New("cannot determine the project of the resource")

	// ErrNoOrganizationID is returned when the organization a resource belongs
	// to cannot be determined.
	ErrNoOrganizationID = errors.New("cannot determine the organization of the resource")

	// ErrNoUserID is returned when the user a resource belongs to cannot be
	// determined.
	ErrNoUserID = errors.New("cannot determine the user of the resource")

	// ErrNoApplicationID is returned when the application a resource belongs to
	// cannot be determined.
	ErrNoApplicationID = errors.New("cannot determine the application of the resource")

	// ErrNoWebKeyID is returned when the signing key a resource points at cannot
	// be determined.
	ErrNoWebKeyID = errors.New("cannot determine the signing key of the resource")
)

// NewClientFromProviderConfig reads the ProviderConfig referenced by mg from
// the cluster, extracts its credentials and returns a ready to use Zitadel
// client.
//
// The caller is responsible for closing the returned client. A reconcile that
// runs more than once for the same configuration should use a ClientCache
// instead, which shares one client between every caller that needs it.
func NewClientFromProviderConfig(ctx context.Context, kube client.Client, mg resource.ModernManaged) (*zitadel.Client, error) {
	cfg, err := ConfigForProviderConfig(ctx, kube, mg)
	if err != nil {
		return nil, err
	}

	client, err := zitadel.NewClient(ctx, cfg)
	if err != nil {
		return nil, errors.Wrap(err, ErrNewClient.Error())
	}

	return client, nil
}

// providerResolution is the outcome of resolving a provider configuration
// reference: the settings, and a way to read the credentials of whichever kind
// the reference named.
//
// The credentials are behind a closure because the two kinds store them
// differently - a namespaced configuration reads its secret from the
// resource's own namespace, a cluster scoped one from a namespace the secret
// reference names - while a caller that only needs the default organization has
// no reason to read a secret at all.
type providerResolution struct {
	settings *providerSettings
	creds    func(ctx context.Context, kube client.Client) (zitadel.Credentials, error)
}

// resolveProviderConfig reads whichever provider configuration a resource points
// at, and returns its settings together with a reader for its credentials.
//
// This is the only place that decides what a providerConfigRef may name, so
// every reader of a ProviderConfig accepts the same references. The two
// readers used to disagree: building a client accepted a ClusterProviderConfig
// while reading the default organization from it rejected the very same
// reference, so a namespaced resource pointing at one authenticated fine and
// then failed on the next call. Keeping the kind switch here is what stops that
// from coming back.
//
// The two kinds are the same shape on purpose, so a cluster scoped resource and
// a namespaced one authenticate through exactly the same rules and there is one
// definition of what a Zitadel credential looks like.
func resolveProviderConfig(ctx context.Context, kube client.Client, mg resource.ModernManaged,
	ref *xpv1.ProviderConfigReference,
) (*providerResolution, error) {
	switch ref.Kind {
	case "ClusterProviderConfig":
		// A cluster scoped resource cannot name a namespaced ProviderConfig, so it
		// refers to a ClusterProviderConfig instead. A namespaced one may do
		// either: a ProviderConfig of its own namespace, or a ClusterProviderConfig
		// to share one configuration across namespaces.
		pc := &apisv1alpha1.ClusterProviderConfig{}
		if err := kube.Get(ctx, types.NamespacedName{Name: ref.Name}, pc); err != nil {
			return nil, errors.Wrap(err, ErrGetProviderConfig)
		}

		return &providerResolution{
			settings: &providerSettings{
				URL:                   pc.Spec.URL,
				OrganizationID:        pc.Spec.OrganizationID,
				Insecure:              pc.Spec.Insecure,
				InsecureSkipTLSVerify: pc.Spec.InsecureSkipTLSVerify,
			},
			// The secret names its own namespace: a cluster scoped configuration
			// has no namespace of its own to look one up in.
			creds: func(ctx context.Context, kube client.Client) (zitadel.Credentials, error) {
				return extractClusterCredentials(ctx, kube, pc.Spec.Credentials)
			},
		}, nil

	case "", "ProviderConfig":
		pc := &apisv1alpha1.ProviderConfig{}
		if err := kube.Get(ctx, types.NamespacedName{Name: ref.Name, Namespace: mg.GetNamespace()}, pc); err != nil {
			return nil, errors.Wrap(err, ErrGetProviderConfig)
		}

		return &providerResolution{
			settings: &providerSettings{
				URL:                   pc.Spec.URL,
				OrganizationID:        pc.Spec.OrganizationID,
				Insecure:              pc.Spec.Insecure,
				InsecureSkipTLSVerify: pc.Spec.InsecureSkipTLSVerify,
			},
			creds: func(ctx context.Context, kube client.Client) (zitadel.Credentials, error) {
				return extractCredentials(ctx, kube, mg.GetNamespace(), pc.Spec.Credentials)
			},
		}, nil

	default:
		return nil, errors.Wrapf(ErrUnsupportedProviderConfigKind, "%s", ref.Kind)
	}
}

// providerSettings is the part of either configuration that is not a credential,
// gathered so that building the client does not care which kind it came from.
type providerSettings struct {
	URL                   string
	OrganizationID        *string
	Insecure              *bool
	InsecureSkipTLSVerify *bool
}

// ProviderConfigOrganizationID returns the default organization ID configured
// on the ProviderConfig of mg, or an empty string when unset.
//
// It accepts the same references as NewClientFromProviderConfig, including a
// ClusterProviderConfig named by a namespaced resource. It used to reject those,
// which meant a resource could build its client and then fail on this call: the
// two readers disagreed about what a valid reference is.
func ProviderConfigOrganizationID(ctx context.Context, kube client.Client, mg resource.ModernManaged) (string, error) {
	if mg.GetProviderConfigReference() == nil {
		return "", nil
	}

	res, err := resolveProviderConfig(ctx, kube, mg, mg.GetProviderConfigReference())
	if err != nil {
		return "", err
	}

	if res.settings.OrganizationID == nil {
		return "", nil
	}

	return *res.settings.OrganizationID, nil
}

// extractCredentials resolves the credentials of a ProviderConfig into the
// material needed by the Zitadel client.
// extractClusterCredentials reads the credentials of a cluster scoped
// configuration, whose secret references name their own namespace.
func extractClusterCredentials(ctx context.Context, kube client.Client, creds apisv1alpha1.ClusterZitadelCredentials) (zitadel.Credentials, error) {
	if creds.Source != xpv1.CredentialsSourceSecret {
		return zitadel.Credentials{}, errors.New("only Secret credentials are supported")
	}

	switch creds.AuthType {
	case apisv1alpha1.AuthTypeServiceAccount:
		if creds.ServiceAccount == nil {
			return zitadel.Credentials{}, errors.New("spec.credentials.serviceAccount is required when authType is ServiceAccount")
		}

		ref := creds.ServiceAccount.KeySecretRef
		key, err := secretValue(ctx, kube, ref.Namespace, ref.Name, ref.Key)
		if err != nil {
			return zitadel.Credentials{}, errors.Wrap(err, ErrGetCredentials.Error())
		}

		return zitadel.Credentials{ServiceAccountKey: []byte(key)}, nil

	case apisv1alpha1.AuthTypeToken:
		if creds.Token == nil {
			return zitadel.Credentials{}, errors.New("spec.credentials.token is required when authType is Token")
		}

		ref := creds.Token.TokenSecretRef
		token, err := secretValue(ctx, kube, ref.Namespace, ref.Name, ref.Key)
		if err != nil {
			return zitadel.Credentials{}, errors.Wrap(err, ErrGetCredentials.Error())
		}

		return zitadel.Credentials{Token: token}, nil

	default:
		return zitadel.Credentials{}, errors.Errorf("authType %q is not supported", creds.AuthType)
	}
}

func extractCredentials(ctx context.Context, kube client.Client, namespace string, creds apisv1alpha1.ZitadelCredentials) (zitadel.Credentials, error) {
	if creds.Source != xpv1.CredentialsSourceSecret {
		return zitadel.Credentials{}, errors.New("only Secret credentials are supported")
	}

	switch creds.AuthType {
	case apisv1alpha1.AuthTypeServiceAccount:
		if creds.ServiceAccount == nil {
			return zitadel.Credentials{}, errors.New("spec.credentials.serviceAccount is required when authType is ServiceAccount")
		}

		key, err := secretValue(ctx, kube, namespace, creds.ServiceAccount.KeySecretRef.Name, creds.ServiceAccount.KeySecretRef.Key)
		if err != nil {
			return zitadel.Credentials{}, errors.Wrap(err, ErrGetCredentials.Error())
		}

		return zitadel.Credentials{ServiceAccountKey: []byte(key)}, nil

	case apisv1alpha1.AuthTypeToken:
		if creds.Token == nil {
			return zitadel.Credentials{}, errors.New("spec.credentials.token is required when authType is Token")
		}

		token, err := secretValue(ctx, kube, namespace, creds.Token.TokenSecretRef.Name, creds.Token.TokenSecretRef.Key)
		if err != nil {
			return zitadel.Credentials{}, errors.Wrap(err, ErrGetCredentials.Error())
		}

		return zitadel.Credentials{Token: token}, nil

	default:
		return zitadel.Credentials{}, errors.Errorf("unsupported auth type %q", creds.AuthType)
	}
}

// secretValue reads a single key out of a secret in the namespace of the
// managed resource.
func secretValue(ctx context.Context, kube client.Client, namespace, name, key string) (string, error) {
	ref := xpv1.SecretKeySelector{
		SecretReference: xpv1.SecretReference{Name: name, Namespace: namespace},
		Key:             key,
	}

	data, err := resource.CommonCredentialExtractor(ctx, xpv1.CredentialsSourceSecret, kube,
		xpv1.CommonCredentialSelectors{SecretRef: &ref})
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func derefBool(b *bool) bool {
	return b != nil && *b
}

// Join combines several errors into a single one. It exists so that the
// controllers do not have to import the standard errors package alongside the
// github.com/pkg/errors helpers.
func Join(errs ...error) error { return stderrors.Join(errs...) }
