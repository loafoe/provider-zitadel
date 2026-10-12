# provider-zitadel

A **native** [Crossplane](https://crossplane.io) provider for
[Zitadel](https://zitadel.com), the open source identity platform. Manage
organizations, users, projects, applications, policies, actions and identity
providers as Crossplane managed resources.

## Written by hand, not generated

This provider is **not** built with [Upjet](https://github.com/crossplane/upjet).
There is no Terraform binary, no Terraform provider and no generated code from
a plugin schema.

Every managed resource is a plain, readable Go controller written directly
against the [Zitadel Go SDK](https://github.com/zitadel/zitadel-go), calling
Zitadel's modern **v2 gRPC APIs** (`user.v2`, `project.v2`, `application.v2`,
`org.v2`). That means:

* **No Terraform in the loop.** No bundled binary, no plugin protocol, no
  `terraform plan` on every reconcile.
* **Full v2 API coverage.** It speaks the current APIs rather than the v1
  surfaces the Terraform provider is still built on.
* **Readable drift detection.** When a resource drifts, the logic that decided
  it is a few lines of Go you can read, not generated scaffolding.

## Coverage

| | |
|---|---|
| Managed resources | **100** kinds — 81 namespaced, plus 19 cluster scoped variants |
| Namespaced group | `zitadel.m.crossplane.io/v1alpha1` |
| Cluster scoped group | `zitadel.crossplane.io/v1alpha1` |
| Zitadel | 4.x, over its `*/v2` gRPC APIs |
| Crossplane | v2 |
| Platforms | `linux/amd64`, `linux/arm64` |

Every resource the Zitadel Terraform provider registers is modelled here, one
for one. The handful it does not model — static translation strings, and v1 API
resources superseded by v2 — are deliberately left out, with the reasons written
down in the [project README](https://github.com/loafoe/provider-zitadel#coverage-against-the-terraform-provider).

Includes organizations, projects, project roles, OIDC/API/SAML applications,
human users, service accounts, machine keys, personal access tokens,
memberships and grants, user and organization metadata, the login and security
policies, actions and their trigger bindings, domains, email and SMS providers,
signing keys, and all 23 identity provider kinds.

Every resource is **namespaced**, so an entire Zitadel tenancy can live in one
namespace and permissions can be scoped to a single team. The 19 kinds that are
instance-wide singletons also have a `Cluster*` form in `zitadel.crossplane.io`.
Neither form replaces the other; use whichever suits your cluster.

## Install

```console
crossplane xpkg install provider ghcr.io/loafoe/provider-zitadel:<version>
```

## Configure

The provider authenticates as a Zitadel **service account**. Give it a machine
user with the roles it needs (e.g. `ORG_OWNER`), set that user's access token
type to `Jwt`, and store its downloaded `key.json` in a secret:

```console
kubectl -n crossplane-system create secret generic zitadel-sa \
  --from-file=key.json=./key.json
```

```yaml
apiVersion: zitadel.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: zitadel
  namespace: crossplane-system
spec:
  url: https://my-instance.zitadel.cloud
  credentials:
    source: Secret
    authType: ServiceAccount
    serviceAccount:
      keySecretRef:
        name: zitadel-sa
        key: key.json
```

A Personal Access Token is also supported (`authType: Token` with
`token.tokenSecretRef`) for bootstrapping and quick tests.

## Use

```yaml
apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: OIDCApplication
metadata:
  name: billing-web
  namespace: identity
spec:
  providerConfigRef: {name: zitadel, kind: ProviderConfig}
  writeConnectionSecretToRef: {name: billing-web-oidc}
  forProvider:
    name: Billing Web
    projectRef: {name: billing}
    redirectURIs:
      - https://billing.example.com/auth/callback
    grantTypes: [AuthorizationCode, RefreshToken]
    responseTypes: [Code]
    accessTokenType: Jwt
```

References such as `projectRef` resolve to real Zitadel IDs, so nothing has to
be hard coded. The `billing-web-oidc` secret receives the `clientID` and
`clientSecret`.

More examples live in the
[`examples/`](https://github.com/loafoe/provider-zitadel/tree/main/examples)
directory, and the full project README covers authentication, drift detection,
cross-resource references and the Zitadel API quirks the provider works around.

## Report a Bug

Open an [issue](https://github.com/loafoe/provider-zitadel/issues).