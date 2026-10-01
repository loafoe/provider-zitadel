# provider-zitadel

A **native** [Crossplane](https://crossplane.io) provider for
[Zitadel](https://zitadel.com), the open source identity platform.

The provider is written directly against the
[Zitadel Go SDK](https://github.com/zitadel/zitadel-go) — there is no Upjet, no
Terraform binary and no generated code from a plugin schema. It is a plain,
readable Go controller per managed resource, and it uses the modern Zitadel
**v2 APIs** (`user.v2`, `project.v2`, `application.v2`, `org.v2`) over gRPC.

| | |
|---|---|
| Provider package | `ghcr.io/loafoe/provider-zitadel` |
| Managed resource group | `zitadel.m.crossplane.io/v1alpha1` |
| Provider config group | `zitadel.crossplane.io/v1alpha1` |
| Platforms | `linux/amd64`, `linux/arm64` (multi-arch OCI image) |
| Zitadel versions | 4.x (the `*/v2` gRPC APIs) |
| Crossplane | v2 (namespaced managed resources) |

## Managed resources (phase 1)

Everything is a **namespaced** managed resource, so a whole Zitadel tenancy can
be described in one namespace and permissions can be scoped down to a single
team.

| Kind | Purpose |
|---|---|
| `Organization` | A Zitadel organization (tenant). Parent of users, projects and applications. |
| `Project` | A project inside an organization. Parent of roles and OIDC applications. |
| `ProjectRole` | A role of a project, grantable to users and service accounts. |
| `OIDCApplication` | An OAuth2 / OIDC client (redirect URIs, grant types, response types, token settings). |
| `HumanUser` | An interactive user with a profile, email, phone and an initial password. |
| `ServiceAccount` | A machine user. Can be used to authenticate the provider itself. |
| `PersonalAccessToken` | A Personal Access Token for a `HumanUser` or a `ServiceAccount`. |

Phase 1 deliberately covers "enough to create users, service accounts, projects
and OAuth2 clients", plus the two containers (`Organization`, `ProjectRole`) and
the credentials helper (`PersonalAccessToken`) you need to make that useful.
See [Roadmap](#roadmap) for what is next.

## Authenticating the provider

The provider always authenticates as a Zitadel **service account** (a machine
user). Two credential types are supported.

### 1. Service account key (recommended)

This is the same mechanism the official Zitadel Terraform provider uses for
`jwt_profile_json`: the machine key is exchanged for a short lived access token
through the OAuth **JWT Profile** grant, and the token is refreshed
automatically.

1. Create a machine user in the Zitadel console under **Service accounts** (or
   with a `ServiceAccount` managed resource) and give it the roles it needs
   (e.g. `ORG_OWNER` on the organization it manages, or `IAM_OWNER`).
2. Set its **Access token type** to **`Jwt`**. The Zitadel API only accepts
   signed JWTs; the default `Bearer` type issues encrypted tokens that are
   rejected with `Unauthenticated: Errors.Token.Invalid`.
3. Add a machine key to it and download the `key.json` file.
4. Store the file in a secret:

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

### 2. Personal Access Token

Handy for quick tests and for bootstrapping the provider against a new
instance. A PAT never expires, so scope it to a dedicated, least privileged
service account.

```yaml
apiVersion: zitadel.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: zitadel
  namespace: crossplane-system
spec:
  url: https://my-instance.zitadel.cloud
  # Optional default organization for resources that do not set one.
  organizationID: "1234567890"
  credentials:
    source: Secret
    authType: Token
    token:
      tokenSecretRef:
        name: zitadel-sa-pat
        key: token
```

### ProviderConfig reference

| Field | Description |
|---|---|
| `spec.url` | The base URL (issuer) of the instance. The gRPC endpoint is derived as `<host>:443`, or `<host>:80` when `insecure` is set. |
| `spec.insecure` | Plaintext gRPC. For local development only. |
| `spec.insecureSkipTLSVerify` | Skip verification of the server certificate. For instances with a self signed certificate. |
| `spec.organizationID` | Default organization context (`x-zitadel-orgid`) for calls that do not carry an explicit organization. |
| `spec.credentials` | `authType: ServiceAccount` + `serviceAccount.keySecretRef`, or `authType: Token` + `token.tokenSecretRef`. |

## A complete example

```yaml
apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: Organization
metadata:
  name: platform
  namespace: identity
spec:
  providerConfigRef: {name: zitadel, kind: ProviderConfig}
  forProvider:
    name: Platform
---
apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: Project
metadata:
  name: billing
  namespace: identity
spec:
  providerConfigRef: {name: zitadel, kind: ProviderConfig}
  forProvider:
    name: Billing
    organizationRef: {name: platform}   # resolved to the organization ID
    projectRoleAssertion: true
---
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
    projectRef: {name: billing}         # resolved to the project ID
    redirectURIs:
      - https://billing.example.com/auth/callback
    grantTypes: [AuthorizationCode, RefreshToken]
    responseTypes: [Code]
    accessTokenType: Jwt
    accessTokenRoleAssertion: true
```

The `billing-web-oidc` secret is written with:

| Key | Value |
|---|---|
| `clientID` | The OAuth2 client ID. |
| `clientSecret` | The client secret. Zitadel only returns it once, at creation (or when `regenerateClientSecret` is set). |

A `HumanUser` additionally publishes `userID` and `username`, a `ServiceAccount`
publishes `userID` and `username`, and a `PersonalAccessToken` publishes
`token`, `tokenID` and `userID`.

More examples live in [`examples/`](./examples).

## Cross-resource references

Wherever a resource belongs to an `Organization` or a `Project` you can use one
of three forms, so nothing has to be hard coded:

| Field | Meaning |
|---|---|
| `organizationID` / `projectID` / `userID` | The raw Zitadel ID. |
| `organizationRef` / `projectRef` / `userRef` | A reference to a resource of this provider, resolved to its ID. |
| `organizationSelector` / `projectSelector` / `userSelector` | A label selector over resources of this provider. |

Resolution honours Crossplane's reference policies (`ResolveIfNotPresent` by
default) and an already resolved value is kept, so a selector that later
matches more than one resource cannot silently re-point an existing resource.

`PersonalAccessToken.userRef` resolves against **both** `ServiceAccount` and
`HumanUser`, in that order, because tokens are usually issued for machine users.

## Drift detection

`Observe` reads the remote resource, publishes it to `status.atProvider` and
compares it with `spec.forProvider`. A field that is *unset* in the spec is
never compared, so unspecified defaults are not fought over. State fields
(`state`) activate, deactivate, lock and unlock on demand.

A few Zitadel specifics the provider works around:

* **OIDC configuration updates go through the v1 management API.** Zitadel 4.15
  answers `UpdateApplication` on the v2 API with `FailedPrecondition: No
  changes` and never persists the change. The v1 path is the one the official
  Terraform provider uses, and it works. The v2 API is still used for creating,
  reading, deleting and (de)activating applications.
* **API audience scope.** Access tokens are requested with
  `urn:zitadel:iam:org:project:id:zitadel:aud` so the issued token carries the
  ZITADEL project ID in its `aud` claim. Without it the audience is the user
  name and every API call is rejected. See `internal/clients/zitadel/zitadel.go`.
* **Create-only fields.** Zitadel never returns an initial password, the IDP
  links or the metadata of a user after creation, so those fields are applied at
  creation and are not drift detected.
* **Zitadel "no changes".** An update that would change nothing is reported by
  Zitadel as a precondition failure; the provider treats that as success.
* **External name recovery.** If the external name annotation is lost, a
  resource is found again by a pinned `id`, then by name within its
  organization or project. A user supplied (non numeric) external name is never
  resolved, because Crossplane cannot tell whether it is an ID.

## Installing

```console
kubectl apply -f https://raw.githubusercontent.com/loafoe/provider-zitadel/main/package/crds
```

or as a Crossplane package:

```console
crossplane xpkg install provider ghcr.io/loafoe/provider-zitadel:v0.1.0
```

The published image is a multi-arch OCI index covering `linux/amd64` and
`linux/arm64`; the Crossplane package embeds the controller image for every
platform it was built for. CI verifies both platforms after publishing and fails
the build if either is missing.

## Developing

```console
make generate       # deepcopy methodsets, CRDs, crossplane-runtime methodsets
make build          # build the provider binary for the host platform
make build.all      # build for every platform + the multi-arch xpkg
make lint           # golangci-lint
make test           # unit tests
make run            # run out-of-cluster against the current kubecontext
make dev            # kind cluster + CRDs + provider
```

Running out-of-cluster:

```console
kubectl apply -f package/crds
go run ./cmd/provider --debug
```

### Tests

Unit tests are hermetic. There is also an opt-in test that talks to a real
Zitadel instance; it is skipped unless the environment provides credentials:

```console
export ZP_URL=https://my-instance.zitadel.cloud
export ZP_KEY=./key.json     # machine key of a service account
export ZP_PAT=./pat          # ... or a Personal Access Token
make test
```

`ZP_KEY` is the regression test for the API audience scope described above: it
asserts that Zitadel accepts a token obtained from a machine key.

## Roadmap

Phase 1 covers the resources above. Natural next steps, roughly in the order
they tend to be needed:

* `ProjectGrant` and `UserGrant` — sharing a project with another organization
  and granting project roles to users and service accounts.
* `MachineKey` — declaratively add a machine key to a `ServiceAccount` and
  write the `key.json` to a connection secret, so the provider's own service
  account can be bootstrapped without leaving the cluster.
* `OrgIDP` / `IDP` — upstream identity providers.
* `ApplicationAPI` — Zitadel API applications.
* `Action` / `ActionTarget` — login customisation.

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) style expectations inherited from the
Crossplane provider template: table driven tests, `make generate && make lint &&
make test` must be clean, and every change needs a short note in the PR
description.

## License

Apache 2.0. See [LICENSE](./LICENSE).
