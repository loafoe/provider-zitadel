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

## Managed resources

Everything is a **namespaced** managed resource, so a whole Zitadel tenancy can
be described in one namespace and permissions can be scoped down to a single
team.

| Kind | Purpose |
|---|---|
| `Organization` | A Zitadel organization (tenant). Parent of users, projects and applications. |
| `Project` | A project inside an organization. Parent of roles and OIDC applications. |
| `ProjectRole` | A role of a project, grantable to users and service accounts. |
| `OIDCApplication` | An OAuth2 / OIDC client (redirect URIs, grant types, response types, token settings). |
| `ApplicationAPI` | A Zitadel API application, authenticated with a client secret or a private key JWT. |
| `HumanUser` | An interactive user with a profile, email, phone and an initial password. |
| `ServiceAccount` | A machine user. Can be used to authenticate the provider itself. |
| `MachineKey` | A machine key generated for a `ServiceAccount`, with its `key.json` written to a connection secret. |
| `PersonalAccessToken` | A Personal Access Token for a `ServiceAccount`. |
| `OrgMember` | A user's membership of an organization, with organization roles. |
| `InstanceMember` | A user's membership of the instance itself, with IAM roles. |
| `UserGrant` | Project roles granted to a user of the project's own organization. |
| `ProjectGrant` | A project shared with another organization, along with the roles that organization may use. |
| `ProjectGrantMember` | A user of the granted organization, with roles on the shared project. |
| `UserMetadata` | The complete metadata set of a user. |
| `OrganizationMetadata` | The complete metadata set of an organization. |
| `LoginPolicy` | The login policy of an organization: registration, MFA, password lifetimes. |
| `Action` | A JavaScript snippet Zitadel runs at a point in a login flow. |
| `ActionTarget` | Somewhere an action's payload is sent: a webhook, a call, or an asynchronous request. |
| `ActionTargetPublicKey` | A key a target's payloads are encrypted with. |
| `ActionExecutionRequest` | Action targets to call when Zitadel handles a particular request, service, or all of them. |
| `ActionExecutionResponse` | Action targets to call on a response Zitadel is about to return. |
| `ActionExecutionEvent` | Action targets to call on a Zitadel event, or a group of them. |
| `ActionExecutionFunction` | Action targets to call when another action calls a named function. |
| `TriggerActions` | Actions to run at a point in a login flow, such as once a user has authenticated. |
| `LockoutPolicy` … `DefaultSecuritySettings` | The seven organization policies, and the ten instance-wide policies they inherit from. |
| `InstanceFeatures`, `SystemFeatures`, `InstanceRestrictions`, `InstanceSecretGenerator` | The four instance wide settings that are a single value: the feature flags, the registration restrictions, and the shape of one of Zitadel's generated codes. |
| `InstanceCustomDomain`, `InstanceTrustedDomain`, `OrganizationDomain` | The three lists of domains: the ones the instance answers on, the ones allowed to ask it for a token, and the ones an organization owns. |
| `WebKey`, `ActiveWebKey`, `ApplicationKey`, `ProjectMember` | Zitadel's own signing keys and which one is active, the key an application authenticates itself with, and a user's roles on a project. |
| `ApplicationSAML` | A SAML application alongside the API and OIDC ones. |
| `EmailProviderSMTP`, `EmailProviderHTTP`, `SMSProviderTwilio`, `SMSProviderHTTP` | The providers Zitadel sends its codes and notifications through. |
| `IDPOIDC`, `IDPOAuth`, `IDPApple`, `IDPAzureAD`, `IDPGitHub`, `IDPGitHubEnterpriseServer`, `IDPGitLab`, `IDPGitLabSelfHosted`, `IDPGoogle`, `IDPLDAP`, `IDPSAML` | Identity providers available to every organization that has not set up its own. |
| `OrgIDPOIDC`, `OrgIDPOAuth`, `OrgIDPJWT`, `OrgIDPApple`, `OrgIDPAzureAD`, `OrgIDPGitHub`, `OrgIDPGitHubEnterpriseServer`, `OrgIDPGitLab`, `OrgIDPGitLabSelfHosted`, `OrgIDPGoogle`, `OrgIDPLDAP`, `OrgIDPSAML` | The same providers, belonging to one organization. |

All managed resources are namespaced, in `zitadel.m.crossplane.io/v1alpha1`.
`ProviderConfig` stays cluster wide in `zitadel.crossplane.io/v1alpha1`, so one
ProviderConfig can serve every namespace that needs it. See [Roadmap](#roadmap)
for what is next.

### Grouping the resources

The kinds fall into four groups, and it helps to know which one you are looking
at when a manifest fails:

* **Structure** — `Organization`, `Project`, `ProjectRole`, `OIDCApplication`,
  `ApplicationAPI`.
* **Principals** — `HumanUser`, `ServiceAccount`, `MachineKey`,
  `PersonalAccessToken`.
* **Authorization** — `OrgMember`, `InstanceMember`, `UserGrant`,
  `ProjectGrant`, `ProjectGrantMember`. These attach roles to subjects; none of
  them creates a subject.
* **Settings** — `UserMetadata`, `OrganizationMetadata`, `LoginPolicy`, and the
  other sixteen policies: seven per organization and the instance-wide defaults
  they inherit.
* **Actions** — `Action`, `ActionTarget`, `ActionTargetPublicKey`, and the five
  bindings that say when a target is called: four execution kinds plus
  `TriggerActions`.
* **Instance settings** — `InstanceFeatures`, `SystemFeatures`,
  `InstanceRestrictions` and `InstanceSecretGenerator`. The first three are the
  singleton of the instance; the last is one per code type, so the generator
  type is what tells two of them apart.
* **Domains** — the three domain lists, `InstanceCustomDomain`,
  `InstanceTrustedDomain` and `OrganizationDomain`. A domain is a name in a list
  and nothing else, so they share a harness with no equality function at all: the
  name is the identity, and once it is in the list there is nothing to compare.
* **Messaging** — the four providers Zitadel sends codes and notifications
  through, and `ApplicationSAML`.
* **Keys and grants** — `WebKey` (a signing key Zitadel generates),
  `ApplicationKey` (the key an application authenticates itself with, whose
  private half is written to the connection secret), and `ProjectMember`.
* **Identity providers** — twelve provider types, each in two forms: one
  available to the whole instance (`IDPGitHub`) and one belonging to an
  organization (`OrgIDPGitHub`). Twenty three kinds in all, and the last block
  of the Terraform provider's surface.

### What Zitadel will and will not read back

This is the single most important thing to know about identity providers, and it
shapes every one of the twenty three kinds.

Zitadel returns an identity provider's **name, state, and whether it registers
users automatically** — and its configuration **only for an OIDC or a JWT
provider**. For the other twenty one, the by-ID read answers `Identity Provider
Configuration doesn't exist` and the list leaves them out, even after they have
been created.

So for those twenty one:

- their settings are applied once, at creation, and are **not** drift detected;
- changing one means deleting the provider and making it again;
- their existence is taken from the identifier they were created with, because
  there is nothing in Zitadel to read it back from.

`OrgIDPOIDC` and `OrgIDPJWT` are the exception and get real drift detection on
their issuer, endpoints, scopes, client ID and header. A credential — a client
secret, a bind password, a signing key — is never returned by Zitadel either, so
one is applied and then left alone, and is read from a secret rather than
written into the manifest.

### A credential has no safe place in a manifest

A client secret or a directory bind password in a custom resource would be
readable by anyone who can read the object and would end up in git. Every
credential is therefore a `SecretKeySelector`, read when the provider is called.

### Two kinds of role keys

Zitadel uses different role keys at each level, and mixing them up is the most
common reason a grant is rejected. A `ProjectGrant` takes the **project's** role
keys (`invoice-reader`, defined by a `ProjectRole`), while a `ProjectGrantMember`
takes the **grant's** roles, which Zitadel prefixes with `PROJECT_GRANT_`. A
member of a granted project is deliberately not given a role in the project
itself; the grant is what confers the role, and the member picks which of the
grant's roles to hold.

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
publishes `userID` and `username`, a `PersonalAccessToken` publishes `token`,
`tokenID` and `userID`, a `MachineKey` publishes `keyID` and `key.json`, and an
`ApplicationAPI` publishes `clientID` plus `clientSecret` or `privateKey`.

More examples live in [`examples/`](./examples), one file per kind.

## Sharing a project with another organization

The SaaS case, and the one place the two kinds of role key matter, is worth
walking through. A platform organization owns a billing project and offers it to
a customer organization. Three steps, and the order is not negotiable:

```yaml
# 1. The project and a role on it, in the owning organization.
apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: ProjectRole
metadata: {name: invoice-reader, namespace: identity}
spec:
  providerConfigRef: {name: zitadel, kind: ProviderConfig}
  forProvider:
    projectRef: {name: billing}
    displayName: Invoice Reader
    key: invoice-reader
---
# 2. Share the project with the customer organization. Note the role keys: these
#    are the project's own roles, not PROJECT_GRANT_* ones.
apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: ProjectGrant
metadata: {name: billing-to-customer, namespace: identity}
spec:
  providerConfigRef: {name: zitadel, kind: ProviderConfig}
  forProvider:
    organizationRef: {name: platform}          # the project's owner
    projectRef: {name: billing}
    grantedOrganizationRef: {name: customer}   # who it is shared with
    roleKeys: [invoice-reader]
---
# 3. Let a user of the customer organization use it. Note the role keys again:
#    these are the grant's roles.
apiVersion: zitadel.m.crossplane.io/v1alpha1
kind: ProjectGrantMember
metadata: {name: bob-invoice-reader, namespace: identity}
spec:
  providerConfigRef: {name: zitadel, kind: ProviderConfig}
  forProvider:
    projectRef: {name: billing}
    grantedOrganizationRef: {name: customer}
    userRef: {name: bob}
    roleKeys: [PROJECT_GRANT_OWNER]
```

`ProjectGrantMember` needs no `organizationRef`: `grantedOrganizationRef` is both
who the project is shared with and whose members are being added.

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

`PersonalAccessToken.userRef` resolves against a `ServiceAccount`, because
Zitadel issues personal access tokens for machine users only.

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
* **A machine key is looked up per owning user.** Zitadel's v2 `ListKeys`
  answers for the authenticated user only, so a service account's key is
  invisible to it: the lookup would always come back empty and the controller
  would create a new key on every reconcile. The provider reads keys through the
  v1 `ListMachineKeys`, which takes the owning user. The SDK marks that endpoint
  deprecated, and the `//nolint:staticcheck` above the call says why.
* **A project grant's roles can only be narrowed.** Zitadel rejects an update
  naming a role the grant does not already have, and reports it as
  `Errors.Project.Role.NotFound`. Adding a role therefore means recreating the
  grant; removing one is an ordinary update. The provider reports this as
  itself rather than leaving a misleading "unknown project role".
* **Personal access tokens are for machine users.** Zitadel refuses to issue
  one for an interactive user (`Errors.User.WrongType`), so `PersonalAccessToken`
  resolves its reference against a `ServiceAccount`.
* **Metadata is additive.** Zitadel's metadata writes never remove a key, so
  conveying "this is the whole set" takes both a write and an explicit delete of
  the keys that are no longer wanted. That is why `UserMetadata` and
  `OrganizationMetadata` manage the complete set of one subject rather than one
  key each: two resources each writing part of a set would both see the other's
  keys as drift and fight over them.
* **An identity provider Zitadel cannot read back.** Only an OIDC and a JWT
  provider are: for every other kind the by-ID read answers "Identity Provider
  Configuration doesn't exist" and the list omits it, even after it has been
  created. Those kinds are created and removed successfully but cannot be
  observed, so their existence is taken from the identifier they were created
  with, and no drift detection is claimed for them. See "What Zitadel will and
  will not read back" above.
* **An identity provider is unbound before it is removed.** Zitadel refuses to
  remove one that is still offered on a login page, so it has to come off the
  login policy first - the same rule that applies to an action something still
  calls.
* **An instance wide setting that Zitadel cannot reset is restored, not
  removed.** There is no reset call for the registration restrictions or for a
  secret generator, so the value the resource overwrote is recorded in the status
  and written back on deletion. The recording has to happen *before* the first
  write rather than on the first update: a resource created once and never
  edited otherwise reaches no update at all, and would be deleted without putting
  anything back.
* **A recorded restore point has to be writable.** The harness writes one back
  by reading it into the shape the write API accepts, so the observed type of
  every non-resettable setting carries an `Input` method. Without it the restore
  is refused and the instance keeps the values a deleted manifest wrote.
* **Login policy factor lists are diffed.** `UpdateCustomLoginPolicy` does not
  accept second or multi factor lists, so the provider adds and removes them
  through the dedicated endpoints.
* **A login policy is a singleton per organization.** Deleting the resource
  resets the organization to the instance default. An organization with no
  custom policy still answers the read, reporting the inherited default, and
  rejects the update; that is read as "nothing customised yet" and the policy is
  created rather than updated.

### Deletion

Deleting a managed resource never depends on resolving its references, because
Zitadel and Crossplane between them make that unreliable:

* While an object is terminating, the Crossplane reference resolver deliberately
  stops resolving and returns the caller's current value. Resolution therefore
  starts from the spec, so a dependent notices when a referenced resource is
  replaced and follows the new one — and a terminating object instead acts on
  the identity recorded in its own status.
* `Observe` reports the external resource as gone while an object is terminating
  if it cannot resolve anything, so the reconciler reaches `Delete` instead of
  retrying an observation that can never succeed.
* `Delete` uses the identifiers already observed, and returns success when there
  are none: a resource whose Create never got far enough has nothing in Zitadel
  to detach from. Without this a single failed Create leaves an object stuck in
  `Terminating` forever, because the delete that would clean it up depends on the
  very resolution that keeps failing.

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
multi-arch index at all, when a platform is missing, or when the reference cannot
be read — and it says which of those it was.

It reads manifests with `crane` when installed, falling back to
`imagetools inspect --raw`, and parses them with `jq` rather than a buildx
`--format` template: the template context changed between buildx versions and an
older runner fails to resolve `.Manifests`. `crane` is preferred because it can
read a public registry anonymously, while `docker buildx imagetools inspect`
wants a docker login even for a public image — and verifying a public release is
exactly what you want to do without a credential.

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

Phase 2 added the authorization, credential and settings kinds listed above.
Natural next steps, roughly in the order they tend to be needed:

* `OrgIDP` / `IDP` — upstream identity providers.
* `Action` / `ActionTarget` — login customisation.
* `NotificationProvider` and `PasswordComplexity` — the rest of the organization
  settings.

### Scoped resources

Every managed resource has a namespaced form, and so does every one that also
has a cluster scoped form. **Neither is deprecated and neither replaces the
other.** Use whichever suits the cluster.

The cluster scoped kinds live in `zitadel.crossplane.io` beside `ProviderConfig`,
which is the Crossplane v2 convention: `*.m.crossplane.io` for namespaced managed
resources, the bare group for cluster scoped ones.

There is one test for which kinds get both, and it is worth stating because it is
what stops the answer from being "all of them":

> **Can Zitadel hold two of them?**

| | |
| --- | --- |
| **Yes** — `Organization`, `HumanUser`, `Project`, `OIDCApplication`, the 11 instance `IDP*` kinds, `ActionTarget`, `WebKey`, `InstanceCustomDomain`, `ActionExecution*`, `InstanceMember` and the rest | A namespace is how you separate them: by team, by environment, by tenant. Cluster scoping them would only take that away, and avoid no conflict. **Namespaced only.** |
| **No** — the 10 `Default*Policy` defaults, `InstanceFeatures`, `SystemFeatures`, `InstanceRestrictions`, `ActiveWebKey`, the 4 messaging providers | Two of them are the same settings, and the second writer wins. Cluster scoping makes that conflict unrepresentable. **Both.** |

The 19 kinds with both are `ClusterInstanceFeatures`, `ClusterSystemFeatures`,
`ClusterInstanceRestrictions`, `ClusterInstanceSecretGenerator`,
`ClusterActiveWebKey`, `ClusterDefault{Lockout,Notification,PasswordAge,PasswordComplexity,Privacy,Domain,Label,Login,OIDCSettings,SecuritySettings}Policy`,
`ClusterDefaultOIDCSettings` and `Cluster{EmailProviderSMTP,EmailProviderHTTP,SMSProviderTwilio,SMSProviderHTTP}`.

**None of it is a second implementation.** Each cluster scoped kind is reconciled
by the controller that already reconciles the namespaced one: the resource is
handed to it as the kind it mirrors, and the external name, finalizers and status
it produces are copied back. The spec, the observation and the status are one set
of Go types, so there is no second definition of what a field means.

A cluster scoped resource cannot name a secret in a namespace, so it references a
**`ClusterProviderConfig`**, whose credential references say which namespace their
secret is in. A namespaced resource may reference one too, which is the way to
share a single configuration across namespaces.

### Coverage against the Terraform provider

The official Zitadel Terraform provider registers **89** managed resources. This
provider currently models **80** of them, one for one.

**Everything worth modelling is modelled.** What is left is the nine that should
not be, and the reasons are written down rather than left implicit:

| Not modelled | Why |
| --- | --- |
| `hosted_login_translation`, `default_hosted_login_translation` | Static translation strings, not managed state. |
| `application_api`, `application_oidc` | The v1 application API, superseded by the v2 API this provider uses. |
| `org`, `project` | The v1 organization and project APIs, superseded by `organization` and `project_v2`. |
| `domain`, `org_metadata`, `smtp_config` | Deprecated by the Terraform provider in favour of resources that are modelled. |

`./hack/parity.py <path to terraform-provider-zitadel>` checks all of this against
the Terraform provider's own source and exits non-zero while anything worth
modelling is missing, so the figures cannot drift away from it. It reports
**80 of 89 raw, and 80 of 80 adjusted**:

| | Count |
| --- | --- |
| Terraform provider resources | 89 |
| Modelled here, one for one | 80 |
| Deliberately not modelled | 9 |
| Managed resource kinds here | 100, of which 19 are cluster scoped variants |

Two of the eighty are ours in a way the Terraform provider's are not: a v2
application is split here into `ApplicationAPI` and `OIDCApplication`, because
Zitadel has one application service with three types in it and the two kinds
carry genuinely different settings.

The Terraform provider registers 89 resources, which the following reads straight
out of its own source:

```console
git clone --depth 1 --filter=blob:none --sparse \
  https://github.com/zitadel/terraform-provider-zitadel.git
cd tf-zitadel && git sparse-checkout set zitadel
grep -oE '"zitadel_[a-z0-9_]+":\s*\w+\.GetResource\(\)' zitadel/provider.go | sort -u | wc -l
```

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md) style expectations inherited from the
Crossplane provider template: table driven tests, `make generate && make lint &&
make test` must be clean, and every change needs a short note in the PR
description.

## License

Apache 2.0. See [LICENSE](./LICENSE).
