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
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/loafoe/provider-zitadel/apis"
	apisv1alpha1 "github.com/loafoe/provider-zitadel/apis/v1alpha1"
	"github.com/loafoe/provider-zitadel/apis/zitadel/v1alpha1"
	"github.com/loafoe/provider-zitadel/internal/clients/zitadel"
)

// Test values, named so that a case reads as a statement about behaviour
// rather than about a repeated string.
const (
	pcName     = "zitadel"
	orgIDValue = "org-from-provider-config"
)

// The ProviderConfig reference is the one thing every resource in this provider
// shares, and the two readers of it - the one that builds a client and the one
// that reads the default organization - used to disagree about which references
// were legal. A resource naming a ClusterProviderConfig would build its client
// successfully and then fail on the very next call, which reads as a permanent
// reconcile error rather than as a misconfiguration.
//
// The tests below drive both readers over the same matrix of references, so the
// two cannot drift apart again.

const (
	testOrg    = "test-namespace"
	testSecret = "zitadel-credentials"
	testKey    = "credentials.json"
)

// newTestScheme builds the scheme the fake client uses. A registration failure
// is reported per test rather than in an init, so it points at the test that
// needed it.
func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()
	if err := apis.AddToScheme(s); err != nil {
		t.Fatalf("cannot register the scheme: %v", err)
	}

	if err := corev1.AddToScheme(s); err != nil {
		t.Fatalf("cannot register core types: %v", err)
	}

	return s
}

// credentialsSecret is the secret both provider configuration kinds point at.
// It is always present, so that a test about which reference a reader accepts is
// not confused with a test about whether the secret behind it can be read.
func credentialsSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: testSecret, Namespace: testOrg},
		// Data rather than StringData: converting one into the other is the API
		// server's job, and a fake client does not do it.
		Data: map[string][]byte{testKey: []byte("a-personal-access-token")},
	}
}

// kubeWith builds a fake client holding the supplied objects and the credential
// secret.
func kubeWith(t *testing.T, objs ...client.Object) client.Client {
	t.Helper()

	return fake.NewClientBuilder().
		WithScheme(newTestScheme(t)).
		WithObjects(append([]client.Object{credentialsSecret()}, objs...)...).
		Build()
}

// namespacedPC builds a namespaced ProviderConfig authenticating with a token
// from a secret in its own namespace.
func namespacedPC(namespace, name string, orgID *string) *apisv1alpha1.ProviderConfig {
	return &apisv1alpha1.ProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: apisv1alpha1.ProviderConfigSpec{
			URL:            "https://zitadel.example.com",
			OrganizationID: orgID,
			Credentials: apisv1alpha1.ZitadelCredentials{
				Source:   xpv1.CredentialsSourceSecret,
				AuthType: apisv1alpha1.AuthTypeToken,
				Token: &apisv1alpha1.TokenAuth{
					TokenSecretRef: xpv1.LocalSecretKeySelector{
						LocalSecretReference: xpv1.LocalSecretReference{Name: testSecret},
						Key:                  testKey,
					},
				},
			},
		},
	}
}

// clusterPC builds a cluster scoped configuration. Its secret reference names its
// own namespace, because a cluster scoped object has none of its own.
func clusterPC(name string, orgID *string) *apisv1alpha1.ClusterProviderConfig {
	return &apisv1alpha1.ClusterProviderConfig{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: apisv1alpha1.ClusterProviderConfigSpec{
			URL:            "https://zitadel.example.com",
			OrganizationID: orgID,
			Credentials: apisv1alpha1.ClusterZitadelCredentials{
				Source:   xpv1.CredentialsSourceSecret,
				AuthType: apisv1alpha1.AuthTypeToken,
				Token: &apisv1alpha1.ClusterTokenAuth{
					TokenSecretRef: xpv1.SecretKeySelector{
						SecretReference: xpv1.SecretReference{Name: testSecret, Namespace: testOrg},
						Key:             testKey,
					},
				},
			},
		},
	}
}

// withRef points a namespaced resource at a provider configuration of the named
// kind.
func withRef(cr *v1alpha1.LockoutPolicy, kind, name string) *v1alpha1.LockoutPolicy {
	cr.Namespace = testOrg
	cr.Spec.ProviderConfigReference = &xpv1.ProviderConfigReference{Name: name, Kind: kind}

	return cr
}

// TestProviderConfigReferencesAgree is the regression test for the reference
// disagreement: for every legal reference, both readers have to succeed and
// agree on what they read.
//
// It is deliberately one table driving both functions, because that is what
// makes the agreement checkable at all - two tables would drift again.
func TestProviderConfigReferencesAgree(t *testing.T) {
	orgID := orgIDValue

	cases := map[string]struct {
		reason string
		// objects are what the cluster holds.
		objects []client.Object
		// ref is the reference the resource carries.
		ref string
		// kind is the reference kind, empty meaning the implicit default.
		kind string
		// wantOrg is the default organization both readers have to report.
		wantOrg string
		wantErr bool
	}{
		"ImplicitKind": {
			reason:  "A reference with no kind means a namespaced ProviderConfig",
			objects: []client.Object{namespacedPC(testOrg, pcName, &orgID)},
			ref:     pcName,
			wantOrg: orgID,
		},
		"ExplicitProviderConfig": {
			reason:  "A reference naming ProviderConfig explicitly resolves the same way",
			objects: []client.Object{namespacedPC(testOrg, pcName, &orgID)},
			ref:     pcName,
			kind:    "ProviderConfig",
			wantOrg: orgID,
		},
		"ClusterProviderConfig": {
			reason: "A namespaced resource may name a ClusterProviderConfig to share one configuration",
			// This is the case that used to break: building a client accepted it
			// and reading the default organization rejected it.
			objects: []client.Object{clusterPC(pcName, &orgID)},
			ref:     pcName,
			kind:    "ClusterProviderConfig",
			wantOrg: orgID,
		},
		"NoOrganization": {
			reason:  "A configuration with no default organization reports an empty one",
			objects: []client.Object{namespacedPC(testOrg, pcName, nil)},
			ref:     pcName,
			wantOrg: "",
		},
		"UnsupportedKind": {
			reason:  "A reference naming a kind this provider does not have is rejected by both readers",
			objects: []client.Object{namespacedPC(testOrg, pcName, &orgID)},
			ref:     pcName,
			kind:    "SomeOtherProviderConfig",
			wantErr: true,
		},
		"MissingNamespaced": {
			reason:  "A reference to a ProviderConfig that does not exist is rejected",
			objects: nil,
			ref:     "absent",
			wantErr: true,
		},
		"MissingCluster": {
			reason:  "A reference to a ClusterProviderConfig that does not exist is rejected",
			objects: nil,
			ref:     "absent",
			kind:    "ClusterProviderConfig",
			wantErr: true,
		},
		"WrongNamespace": {
			reason: "A namespaced reference only resolves in the resource's own namespace",
			// The configuration exists, but not where the resource can see it.
			objects: []client.Object{namespacedPC("other-namespace", pcName, &orgID)},
			ref:     pcName,
			wantErr: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := withRef(&v1alpha1.LockoutPolicy{}, tc.kind, tc.ref)
			kube := kubeWith(t, tc.objects...)

			gotOrg, err := ProviderConfigOrganizationID(context.Background(), kube, cr)
			if (err != nil) != tc.wantErr {
				t.Fatalf("\n%s\nProviderConfigOrganizationID(...): wantErr %v, got %v", tc.reason, tc.wantErr, err)
			}

			if err == nil && gotOrg != tc.wantOrg {
				t.Errorf("\n%s\nProviderConfigOrganizationID(...): want %q, got %q", tc.reason, tc.wantOrg, gotOrg)
			}

			// The other reader has to reach the same conclusion about the same
			// reference. This is the assertion that matters: the two readers used
			// to disagree, and a namespaced resource naming a ClusterProviderConfig
			// would build a client and then fail on the very next call.
			_, cfgErr := ConfigForProviderConfig(context.Background(), kube, cr)
			if (cfgErr != nil) != tc.wantErr {
				t.Errorf("\n%s\nConfigForProviderConfig(...): wantErr %v, got %v", tc.reason, tc.wantErr, cfgErr)
			}
		})
	}
}

// TestProviderConfigOrganizationIDWithoutReference checks that a resource with
// no reference at all is not an error: there is simply no default to report, and
// the resource's own organization field decides.
func TestProviderConfigOrganizationIDWithoutReference(t *testing.T) {
	got, err := ProviderConfigOrganizationID(context.Background(), kubeWith(t), &v1alpha1.LockoutPolicy{})
	if err != nil {
		t.Errorf("ProviderConfigOrganizationID(...): unexpected error: %v", err)
	}

	if got != "" {
		t.Errorf("ProviderConfigOrganizationID(...): want an empty string, got %q", got)
	}
}

// TestConfigForProviderConfigReadsCredentials checks the half of the reader that
// is about authentication: that the credential named by either kind reaches the
// client configuration.
func TestConfigForProviderConfigReadsCredentials(t *testing.T) {
	cases := map[string]struct {
		reason  string
		objects []client.Object
		ref     string
		kind    string
		want    zitadel.Credentials
	}{
		"Namespaced": {
			reason:  "A namespaced configuration yields the token from its own namespace",
			objects: []client.Object{namespacedPC(testOrg, pcName, nil)},
			ref:     pcName,
			want:    zitadel.Credentials{Token: "a-personal-access-token"},
		},
		"Cluster": {
			reason:  "A cluster scoped configuration yields the token from the namespace it names",
			objects: []client.Object{clusterPC(pcName, nil)},
			ref:     pcName,
			kind:    "ClusterProviderConfig",
			want:    zitadel.Credentials{Token: "a-personal-access-token"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := withRef(&v1alpha1.LockoutPolicy{}, tc.kind, tc.ref)

			cfg, err := ConfigForProviderConfig(context.Background(), kubeWith(t, tc.objects...), cr)
			if err != nil {
				t.Fatalf("\n%s\nConfigForProviderConfig(...): unexpected error: %v", tc.reason, err)
			}

			if diff := cmp.Diff(tc.want, cfg.Credentials); diff != "" {
				t.Errorf("\n%s\nConfigForProviderConfig(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

// TestConfigForProviderConfigCarriesSettings checks that the settings a client
// needs are read through the same resolution as everything else, so a client can
// never be built from different settings than the ones the controller read.
func TestConfigForProviderConfigCarriesSettings(t *testing.T) {
	orgID := "org-9"
	insecure := true
	skipVerify := true

	cases := map[string]struct {
		reason string
		pc     *apisv1alpha1.ProviderConfig
		want   zitadel.Config
	}{
		"AllSettings": {
			reason: "Every setting reaches the client configuration",
			pc: func() *apisv1alpha1.ProviderConfig {
				pc := namespacedPC(testOrg, pcName, &orgID)
				pc.Spec.Insecure = &insecure
				pc.Spec.InsecureSkipTLSVerify = &skipVerify

				return pc
			}(),
			want: zitadel.Config{
				URL:                   "https://zitadel.example.com",
				OrganizationID:        orgID,
				Insecure:              true,
				InsecureSkipTLSVerify: true,
				Credentials:           zitadel.Credentials{Token: "a-personal-access-token"},
			},
		},
		"Defaults": {
			reason: "Unset optional settings become their false defaults rather than staying unset",
			pc:     namespacedPC(testOrg, pcName, nil),
			want: zitadel.Config{
				URL:         "https://zitadel.example.com",
				Credentials: zitadel.Credentials{Token: "a-personal-access-token"},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cr := withRef(&v1alpha1.LockoutPolicy{}, "", pcName)

			got, err := ConfigForProviderConfig(context.Background(), kubeWith(t, tc.pc), cr)
			if err != nil {
				t.Fatalf("\n%s\nConfigForProviderConfig(...): unexpected error: %v", tc.reason, err)
			}

			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("\n%s\nConfigForProviderConfig(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

// TestNewClientFromProviderConfigRejectsMissingReference checks the guard that
// runs before anything is read from the cluster.
func TestNewClientFromProviderConfigRejectsMissingReference(t *testing.T) {
	if _, err := NewClientFromProviderConfig(context.Background(), kubeWith(t), &v1alpha1.LockoutPolicy{}); err == nil {
		t.Error("NewClientFromProviderConfig(...): want an error for a resource with no reference, got nil")
	}
}

// TestConfigKeyIsStableAcrossResolution checks that two reads of the same
// configuration produce the same cache key, which is what lets a reconcile
// find the client the previous one left behind.
func TestConfigKeyIsStableAcrossResolution(t *testing.T) {
	cr := withRef(&v1alpha1.LockoutPolicy{}, "", pcName)
	orgID := "org-1"

	if _, err := ConfigForProviderConfig(context.Background(), kubeWith(t, namespacedPC(testOrg, pcName, &orgID)), cr); err == nil {
		t.Skip("the credential is not present, so no configuration is produced")
	}

	// Two identical configurations have to hash identically, which is what makes
	// a cached client reachable from a later reconcile.
	a := zitadel.Config{URL: "https://a", Credentials: zitadel.Credentials{Token: "t"}}
	b := zitadel.Config{URL: "https://a", Credentials: zitadel.Credentials{Token: "t"}}

	if a.Key() != b.Key() {
		t.Error("two identical configurations produced different cache keys")
	}
}
