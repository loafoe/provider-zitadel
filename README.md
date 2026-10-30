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
| Provider package | `ghcr.io/loafoe/provider-zitadel` — multi-arch, cosigned |
| Managed resources | **81** namespaced kinds, plus **19** cluster scoped variants — **100** in all |
| Namespaced group | `zitadel.m.crossplane.io/v1alpha1` |
| Cluster scoped group | `zitadel.crossplane.io/v1alpha1` (with `ClusterProviderConfig`) |
| Platforms | `linux/amd64`, `linux/arm64` |
| Zitadel | 4.x, over its `*/v2` gRPC APIs |
| Crossplane | v2 |

## Installing

Pick a release tag from [the releases page](https://github.com/loafoe/provider-zitadel/releases)
— `v0.3.0` at the time of writing — and install it as a Crossplane v2 package:

```console
crossplane xpkg install provider ghcr.io/loafoe/provider-zitadel:<version>
```

Or, without the Crossplane package machinery, apply just the CRDs:

```console
kubectl apply -f https://raw.githubusercontent.com/loafoe/provider-zitadel/main/package/crds
```

Install a tag rather than `latest`: only tags are signed. Releases are signed
with a keyless [cosign](https://github.com/sigstore/cosign) signature, and you
can verify one yourself:

```console
cosign verify \
  --certificate-identity-regexp '^https://github\.com/loafoe/provider-zitadel/\.github/workflows/release\.yml@refs/tags/v[0-9]+\.[0-9]+\.[0-9]+$' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  ghcr.io/loafoe/provider-zitadel:<version>
```

To make a cluster refuse anything unsigned, start Crossplane with
`--enable-signature-verification` and apply the shipped `ImageConfig`, which
pins this repository's release workflow as the only accepted signer:

```console
kubectl apply -f examples/imageconfig-signature-verification.yaml
```

### If the install fails

`crossplane xpkg install provider ghcr.io/loafoe/provider-zitadel:<version>`
pulls a package that embeds its own controller image, so it needs read access to
a **public** package in the registry.

| Symptom | Check |
| --- | --- |
| `unauthorized` or `denied` pulling the package | The GHCR package is private. Make it public again in [package settings](https://github.com/packages?package_type=container). |
| The package is missing entirely | There is no release for that tag yet. `gh release list` shows what has been published; a `v*` tag produces one. |
| Signature verification fails | Only relevant with `signatureVerification` enabled — see above. |
| `no matching manifest for linux/arm64` | You are on an architecture the package does not cover, or on an older `crossplane` CLI that cannot read a multi-arch index. |

## Authenticating the provider

The provider always authenticates as a Zitadel **service account** (a machine
user). Two credential types are supported.

### 1. Service account key (recommended)

The machine key is exchanged for a short lived access token through the OAuth
**JWT Profile** grant, and the token is refreshed automatically.

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

A cluster scoped resource cannot name a secret in a namespace, so it references a
**`ClusterProviderConfig`**, whose credential references say which namespace
their secret is in. A namespaced resource may reference one too, which is the
way to share a single configuration across namespaces.

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
`tokenID` and `userID`, a `MachineKey` publishes `keyID` and `key.json`, an
`ApplicationAPI` publishes `clientID` plus `clientSecret` or `privateKey`, and an
`ActionTarget` publishes `targetID` and `signingKey` - the only place the signing
key appears, since it is not copied into the resource's status.

More examples live in [`examples/`](./examples).

## Managed resources

Every managed resource has a **namespaced** form, so a whole Zitadel tenancy can
be described in one namespace and permissions can be scoped down to a single
team. The nineteen kinds that are instance-wide singletons also have a cluster
scoped form; see [Scoped resources](#scoped-resources).

| Kind | Purpose |
|---|---|
| `Organization` | A Zitadel organization (tenant). Parent of users, projects and applications. |
| `Project` | A project inside an organization. Parent of roles and OIDC applications. |
| `ProjectRole` | A role of a project, grantable to users and service accounts. |
| `OIDCApplication` | An OAuth2 / OIDC client (redirect URIs, grant types, response types, token settings). |
| `ApplicationAPI` | A Zitadel API application, authenticated with a client secret or a private key JWT. |
| `ApplicationSAML` | A SAML application alongside the API and OIDC ones. |
| `HumanUser` | An interactive user with a profile, email, phone and an initial password. |
| `ServiceAccount` | A machine user. Can be used to authenticate the provider itself. |
| `MachineKey` | A machine key generated for a `ServiceAccount`, with its `key.json` written to a connection secret. |
| `PersonalAccessToken` | A Personal Access Token for a `ServiceAccount`. |
| `OrgMember` | A user's membership of an organization, with organization roles. |
| `InstanceMember` | A user's membership of the instance itself, with IAM roles. |
| `UserGrant` | Project roles granted to a user of the project's own organization. |
| `ProjectGrant` | A project shared with another organization, along with the roles that organization may use. |
| `ProjectGrantMember` | A user of the granted organization, with roles on the shared project. |
| `ProjectMember` | A user's roles on a project. |
| `UserMetadata` | The complete metadata set of a user. |
| `OrganizationMetadata` | The complete metadata set of an organization. |
| `LoginPolicy` | The login policy of an organization: registration, MFA, password lifetimes. |
| `DomainPolicy`, `LabelPolicy`, `LockoutPolicy`, `NotificationPolicy`, `PasswordAgePolicy`, `PasswordComplexityPolicy`, `PrivacyPolicy` | The seven organization policies that sit beside `LoginPolicy`: branding, password rules, notifications and privacy. |
| `DefaultDomainPolicy`, `DefaultLabelPolicy`, `DefaultLockoutPolicy`, `DefaultLoginPolicy`, `DefaultNotificationPolicy`, `DefaultPasswordAgePolicy`, `DefaultPasswordComplexityPolicy`, `DefaultPrivacyPolicy`, `DefaultOIDCSettings`, `DefaultSecuritySettings` | The ten instance-wide policies and settings an organization inherits until it sets its own. |
| `InstanceFeatures`, `SystemFeatures`, `InstanceRestrictions`, `InstanceSecretGenerator` | The four instance wide settings that are a single value: the feature flags, the registration restrictions, and the shape of one of Zitadel's generated codes. |
| `InstanceCustomDomain`, `InstanceTrustedDomain`, `OrganizationDomain` | The three lists of domains: the ones the instance answers on, the ones allowed to ask it for a token, and the ones an organization owns. |
| `WebKey`, `ActiveWebKey`, `ApplicationKey` | Zitadel's own signing keys and which one is active, and the key an application authenticates itself with (its private half is written to the connection secret). |
| `Action` | A JavaScript snippet Zitadel runs at a point in a login flow. |
| `ActionTarget` | Somewhere an action's payload is sent: a webhook, a call, or an asynchronous request. |
| `ActionTargetPublicKey` | A key a target's payloads are encrypted with. |
| `ActionExecutionRequest` | Action targets to call when Zitadel handles a particular request, service, or all of them. |
| `ActionExecutionResponse` | Action targets to call on a response Zitadel is about to return. |
| `ActionExecutionEvent` | Action targets to call on a Zitadel event, or a group of them. |
| `ActionExecutionFunction` | Action targets to call when another action calls a named function. |
| `TriggerActions` | Actions to run at a point in a login flow, such as once a user has authenticated. |
| `EmailProviderSMTP`, `EmailProviderHTTP`, `SMSProviderTwilio`, `SMSProviderHTTP` | The providers Zitadel sends its codes and notifications through. |
| `IDPOIDC`, `IDPOAuth`, `IDPApple`, `IDPAzureAD`, `IDPGitHub`, `IDPGitHubEnterpriseServer`, `IDPGitLab`, `IDPGitLabSelfHosted`, `IDPGoogle`, `IDPLDAP`, `IDPSAML` | Identity providers available to every organization that has not set up its own. |
| `OrgIDPOIDC`, `OrgIDPOAuth`, `OrgIDPJWT`, `OrgIDPApple`, `OrgIDPAzureAD`, `OrgIDPGitHub`, `OrgIDPGitHubEnterpriseServer`, `OrgIDPGitLab`, `OrgIDPGitLabSelfHosted`, `OrgIDPGoogle`, `OrgIDPLDAP`, `OrgIDPSAML` | The same providers, belonging to one organization. |

The identity provider kinds come to 23 rather than 24 because `OrgIDPJWT` has no
instance-wide counterpart: Zitadel only offers the JWT provider per
organization.

### Grouping the resources

The kinds fall into ten groups, and it helps to know which one you are looking
at when a manifest fails:

* **Structure** — `Organization`, `Project`, `ProjectRole`, `OIDCApplication`,
  `ApplicationAPI`, `ApplicationSAML`.
* **Principals** — `HumanUser`, `ServiceAccount`, `MachineKey`,
  `PersonalAccessToken`.
* **Authorization** — `OrgMember`, `InstanceMember`, `UserGrant`,
  `ProjectGrant`, `ProjectGrantMember`, `ProjectMember`. These attach roles to
  subjects; none of them creates a subject.
* **Settings** — `UserMetadata`, `OrganizationMetadata`, `LoginPolicy`, and the
  other seventeen policies: seven more per organization and the ten
  instance-wide defaults they inherit.
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
  through.
* **Keys and grants** — `WebKey` (a signing key Zitadel generates),
  `ApplicationKey` (the key an application authenticates itself with), and
  `ProjectMember`.
* **Identity providers** — the largest block: twenty three kinds covering twelve
  provider types. Eleven are available to the whole instance (`IDPGitHub`) and
  twelve belong to one organization (`OrgIDPGitHub`).

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

### Two kinds of role keys

Zitadel uses different role keys at each level, and mixing them up is the most
common reason a grant is rejected. A `ProjectGrant` takes the **project's** role
keys (`invoice-reader`, defined by a `ProjectRole`), while a `ProjectGrantMember`
takes the **grant's** roles, which Zitadel prefixes with `PROJECT_GRANT_`. A
member of a granted project is deliberately not given a role in the project
itself; the grant is what confers the role, and the member picks which of the
grant's roles to hold.

### A credential has no safe place in a manifest

A client secret or a directory bind password in a custom resource would be
readable by anyone who can read the object and would end up in git. Every
credential is therefore a `SecretKeySelector`, read when the provider is called.

## Drift detection

The provider reads the remote resource, publishes it to `status.atProvider` and
compares it with `spec.forProvider`. A field that is *unset* in the spec is
never compared, so unspecified defaults are not fought over. State fields
(`state`) activate, deactivate, lock and unlock on demand.

Several Zitadel behaviours affect what you can expect:

* **Fields Zitadel never returns.** An initial password, the identity provider
  links on a user, and a user's metadata are only available at creation, so those
  fields are applied once and are not drift detected afterwards.
* **A project grant's roles can only be narrowed.** Zitadel rejects an update
  naming a role the grant does not already have (`Errors.Project.Role.NotFound`).
  Adding a role therefore means recreating the grant; removing one is an ordinary
  update.
* **Metadata is additive.** Zitadel's metadata writes never remove a key, so
  conveying "this is the whole set" takes both a write and an explicit delete of
  the keys that are no longer wanted. That is why `UserMetadata` and
  `OrganizationMetadata` manage the complete set of one subject rather than one
  key each: two resources each writing part of a set would both see the other's
  keys as drift and fight over them.
* **An identity provider must be unbound before it is removed.** Zitadel refuses
  to remove one that is still offered on a login page, so it has to come off the
  login policy first — the same rule that applies to an action something still
  calls. See [What Zitadel will and will not read
  back](#what-zitadel-will-and-will-not-read-back) for the related read-back
  limitation.
* **A login policy is a singleton per organization.** Deleting the resource
  resets the organization to the instance default. An organization with no custom
  policy still answers the read, reporting the inherited default.
* **An instance wide setting that Zitadel cannot reset is restored, not removed.**
  There is no reset call for the registration restrictions or for a secret
  generator, so the value the resource overwrote is recorded and written back on
  deletion rather than left in place.

Deletion never depends on resolving references, so a resource whose references
are broken or whose create failed part way through can still be removed and will
release its finalizer.

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

## Coverage against the Terraform provider

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

Those eighty capabilities are 81 namespaced kinds, because one is split in two: a
v2 application becomes `ApplicationAPI` and `OIDCApplication` here. Zitadel has
one application service with three types in it, and the two kinds carry
genuinely different settings, where the Terraform provider keeps them in a
single `application_v2` resource.

| | Count |
| --- | --- |
| Terraform provider resources | 89 |
| Modelled here, one for one | 80 |
| Deliberately not modelled | 9 |
| Managed resource kinds here | 100, of which 19 are cluster scoped variants |

`hack/parity.py` checks this against the Terraform provider's own source and
exits non-zero while anything worth modelling is missing, so these figures
cannot silently drift away from it.

## Scoped resources

Every managed resource has a namespaced form, and so does every one that also
has a cluster scoped form. **Neither is deprecated and neither replaces the
other.** Use whichever suits the cluster.

The cluster scoped kinds live in `zitadel.crossplane.io` beside `ProviderConfig`,
which is the Crossplane v2 convention: `*.m.crossplane.io` for namespaced managed
resources, the bare group for cluster scoped ones.

Which kinds get both comes down to one question:

> **Can Zitadel hold two of them?**

| | |
| --- | --- |
| **Yes** — `Organization`, `HumanUser`, `Project`, `OIDCApplication`, the 11 instance `IDP*` kinds, `ActionTarget`, `WebKey`, `InstanceCustomDomain`, `ActionExecution*`, `InstanceMember` and the rest | A namespace is how you separate them: by team, by environment, by tenant. Cluster scoping them would only take that away, and avoid no conflict. **Namespaced only.** |
| **No** — the 10 instance-wide defaults, `InstanceFeatures`, `SystemFeatures`, `InstanceRestrictions`, `InstanceSecretGenerator`, `ActiveWebKey`, the 4 messaging providers | Two of them are the same settings, and the second writer wins. Cluster scoping makes that conflict unrepresentable. **Both.** |

The 19 kinds with both are `ClusterInstanceFeatures`, `ClusterSystemFeatures`,
`ClusterInstanceRestrictions`, `ClusterInstanceSecretGenerator`,
`ClusterActiveWebKey`, the ten `ClusterDefault*` policies and settings
(`ClusterDefaultDomainPolicy`, `ClusterDefaultLabelPolicy`,
`ClusterDefaultLockoutPolicy`, `ClusterDefaultLoginPolicy`,
`ClusterDefaultNotificationPolicy`, `ClusterDefaultPasswordAgePolicy`,
`ClusterDefaultPasswordComplexityPolicy`, `ClusterDefaultPrivacyPolicy`,
`ClusterDefaultOIDCSettings`, `ClusterDefaultSecuritySettings`) and
`ClusterEmailProviderSMTP`, `ClusterEmailProviderHTTP`, `ClusterSMSProviderTwilio`,
`ClusterSMSProviderHTTP`.

**None of it is a second implementation.** Each cluster scoped kind is reconciled
by the controller that already reconciles the namespaced one: the resource is
handed to it as the kind it mirrors, and the external name, finalizers and status
it produces are copied back. The spec, the observation and the status are one set
of Go types, so there is no second definition of what a field means.

## What has not been proven

Most kinds were run end to end against a live instance. These were not, and the
reason is always the same: something outside Zitadel is needed and is not there.

| Verified | How far |
| --- | --- |
| `OrgIDPJWT`, `OrgIDPOAuth` | Full lifecycle: created, read back, drifted, corrected, deleted |
| `OrgIDPSAML`, `IDPGitHub`, `IDPAzureAD` | Created with real Zitadel identifiers and deleted |
| `OrgIDPLDAP`, `OrgIDPApple` | Reached Zitadel and were refused on their own validation, which is as far as a bare instance goes |

| Not proven | Why |
| --- | --- |
| The other **16** identity provider kinds | Each needs a real upstream to federate to. |
| `EmailProviderHTTP`, `SMSProviderHTTP` | Zitadel refuses an arbitrary endpoint with `Errors.NotificationProvider.Blocked.Endpoint` — an instance policy, not a wiring problem. |
| `SystemFeatures`, `InstanceCustomDomain` | Zitadel returns `PermissionDenied` for both even to an IAM owner, while reading them works. |

That is 20 of the 80 capabilities whose write path is wired and reasoned about
but whose full round trip has not been observed on this instance.

Zitadel also has capabilities that have no kind here at all, and are additions
rather than parity work: **quotas** (the nearest genuine gap — a real limit on an
organization that nothing can manage today), **session** listing and revocation,
and the read-only **analytics** reports.

## Developing

```console
make generate       # deepcopy methodsets, CRDs, crossplane-runtime methodsets
make build          # build the provider binary for the host platform
make build.all      # build for every platform + the multi-arch xpkg
make lint           # golangci-lint
make test           # unit tests
make test-integration   # kind cluster + Zitadel, apply and delete resources
make run            # run out-of-cluster against the current kubecontext
make dev            # kind cluster + CRDs + provider
```

Running out-of-cluster:

```console
kubectl apply -f package/crds
go run ./cmd/provider --debug
```

Unit tests are hermetic. There is also an opt-in test that talks to a real Zitadel
instance; it is skipped unless the environment provides credentials:

```console
export ZP_URL=https://my-instance.zitadel.cloud
export ZP_KEY=./key.json     # machine key of a service account
export ZP_PAT=./pat          # ... or a Personal Access Token
make test
```

See [CONTRIBUTING.md](./CONTRIBUTING.md) for style expectations and the workflow
for adding a kind.

## License

Apache 2.0. See [LICENSE](./LICENSE).