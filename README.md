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
| Provider package | `ghcr.io/loafoe/provider-zitadel` (multi-arch, cosigned) |
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

As a Crossplane v2 package:

```console
crossplane xpkg install provider ghcr.io/loafoe/provider-zitadel:v0.1.0
```

Or, without the Crossplane package machinery, apply just the CRDs:

```console
kubectl apply -f https://raw.githubusercontent.com/loafoe/provider-zitadel/main/package/crds
```

## Releases, signing and multi-arch

Pushing a `v*` tag runs
[`.github/workflows/release.yml`](./.github/workflows/release.yml), which:

1. runs `make reviewable` and `make build.all` — every platform is built and the
   multi-arch package is assembled;
2. publishes two artifacts, both **multi-arch OCI image indexes** covering
   `linux/amd64` and `linux/arm64`:

   | Reference | What it is |
   |---|---|
   | `ghcr.io/loafoe/provider-zitadel:<tag>` | The Crossplane package. This is what you install. |
   | `ghcr.io/loafoe/provider-zitadel:controller-<tag>` | The standalone controller image, for running the provider out of cluster. |

3. signs both with a **keyless cosign signature** (Fulcio certificate + Rekor
   transparency log entry) using the GitHub Actions OIDC identity, and emits a
   SLSA provenance attestation;
4. verifies the signatures, verifies that both platforms are present, then
   promotes to the `:stable` channel — and signs every channel tag too.

A second job then re-verifies the published signature from scratch, using only
the workflow identity — i.e. it proves a consumer can verify the package without
any repository-local knowledge.

Verify a release yourself:

```console
cosign verify \
  --certificate-identity-regexp '^https://github\.com/loafoe/provider-zitadel/\.github/workflows/release\.yml@refs/tags/v[0-9]+\.[0-9]+\.[0-9]+$' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  ghcr.io/loafoe/provider-zitadel:v0.1.0
```

or `make verify-signature VERSION=v0.1.0`. `make sign VERSION=v0.1.0` is the
signing half, for when you need to re-sign from a different runner.

### The package is the runtime image

A Crossplane v2 provider package embeds its controller image: the xpkg is built
with `--embed-runtime-image`, so `spec.controller.image` stays empty and the
package image doubles as the runtime image. That is why the standalone
controller image is tagged `controller-<tag>` rather than `<tag>` — a package and
an image cannot share a tag. The multi-arch property holds end to end: the package
index has one entry per platform, each embedding that platform's controller
image.

### The GHCR package must belong to the repository

A workflow's `GITHUB_TOKEN` may only push to a GHCR package that is **associated
with the repository the workflow runs in**. The association is established by the
`org.opencontainers.image.source` label on an image pushed from that repository,
which `make publish` does first by publishing the labelled controller image.

If the package exists but was created from somewhere else — for example pushed
from a laptop with a personal token — the token is refused with
`denied: permission_denied: read_package` before the label is ever looked at.
The fix is one action, in
[GitHub's package settings](https://github.com/packages?package_type=container):
**delete `provider-zitadel`**, and let the next run recreate it. The first push
from the repository associates the package with it and, because this account
defaults to public package visibility, makes it public in one go.

While the package is in that state, `ci.yml` reports the failed publish as a
warning annotation and the run stays green — a branch build is not the place to
enforce a one time owner action. `release.yml` keeps the step strict, so a tagged
release fails loudly until the package is set up.

### Checking the multi-arch property yourself

Every publish and promote runs `hack/verify-multiarch.sh`, which is also useful
by hand:

```console
hack/verify-multiarch.sh ghcr.io/loafoe/provider-zitadel:v0.1.0
```

```
==> ghcr.io/loafoe/provider-zitadel:v0.1.0
    mediaType: application/vnd.oci.image.index.v1+json
    platforms: linux/amd64, linux/arm64
All references cover: linux/amd64 linux/arm64
```

It fails, listing every problem rather than the first, when a reference is not a
multi-arch index at all, when a platform is missing, or when the reference
cannot be read. The manifest is parsed with `imagetools inspect --raw` and `jq`
rather than a buildx `--format` template, because the template context changed
between buildx versions and an older runner silently fails to resolve
`.Manifests`.

Branch builds pushed by `ci.yml` are deliberately **not** signed. A cluster with
package signature verification enabled must install a `v*` tag, and signing a
mutable branch ref would let it stand in for a release.

### Enforcing signature verification in a cluster

Crossplane can refuse to install a package that is not signed by a trusted
authority. It is an alpha feature, so it has to be switched on first:

```console
crossplane core start --enable-signature-verification
```

Then apply the shipped `ImageConfig`, which pins this repository's release
workflow as the only accepted signer:

```console
kubectl apply -f examples/imageconfig-signature-verification.yaml
crossplane xpkg install provider ghcr.io/loafoe/provider-zitadel:v0.1.0
```

Note that only release tags are signed. Branch builds pushed by CI are
deliberately unsigned, so a cluster with verification enabled must install a
`v*` tag.

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
