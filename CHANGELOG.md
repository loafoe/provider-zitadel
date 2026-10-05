# Changelog

All notable changes to this provider are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Changed

- **The provider now reuses one Zitadel connection instead of building twelve
  per reconcile.** A `zitadel.Client` previously opened a separate gRPC
  connection for each of the ZITADEL services it wrapped - twelve in all, each
  with its own copy of the authentication interceptor - and closed all of them
  again at the end of every reconcile. All twelve are gRPC services on one
  endpoint, so they now share one connection and one token source. No manifest
  changes.
- **A client is now cached per ProviderConfig and shared between the resources
  that use it.** Previously every reconcile of every managed resource rebuilt its
  client, paying a TLS handshake and a token exchange each time. Resources
  pointing at the same ProviderConfig now share one client, and therefore one
  connection, one token source and one set of lazily created per-organization
  management clients. A rotated secret still produces a new client, because the
  cache key covers both credentials. Clients for configurations that have gone
  away are released after 30 idle minutes, and the cache is bounded at 32
  clients.
- **`make test-integration` now has a script to run.** The target existed and
  referenced `cluster/local/integration_tests.sh`, which was not in the
  repository. It stands a Zitadel up in a kind cluster, deploys the provider
  built from the working tree, applies a set of resources, waits for them to
  become ready and then deletes them again to prove the finalizers are released.
- **A namespaced resource may now name a `ClusterProviderConfig`.** The two
  readers of a `providerConfigRef` disagreed: building a client accepted a
  `ClusterProviderConfig`, while reading the default organization from it
  rejected the same reference. A resource in that configuration authenticated
  successfully and then failed on the very next call, which reads as a permanent
  reconcile error rather than as a misconfiguration. Both now go through one
  resolution, so they cannot drift apart again.

### Fixed

- **A policy with an omitted field no longer reports drift forever.** Every field
  of every policy is optional in the CRD, but each driver's drift check compared
  every field against what Zitadel holds. `Desired` has to turn an unset field
  into something, and it turned it into the zero value, so a manifest that
  mentioned only `maxPasswordAttempts` was reported as out of date on every poll
  because Zitadel's `maxOTPAttempts` was not zero - and was rewritten forever.
  A field the manifest left unset is now left alone, which is what the doc
  comment above each of those methods already promised. Affects all seventeen
  organization and instance wide policy drivers plus the three instance settings
  drivers; the login policy was already correct, because it defaults an unset
  field to Zitadel's documented default rather than declining to manage it, and
  that is still how it works.
- The instance wide OIDC settings are now reported as missing when Zitadel
  returns a response carrying no settings. Reading the fields off that nil
  payload yielded an empty lifetime for each of them, which the controller would
  then have reported as drift against the lifetimes the manifest asked for, on
  every poll. The other seven instance policies already guarded for this.
- A negative duration in a login policy lifetime is now dropped rather than sent
  to Zitadel, which rejects it. The helper used for this disagreed with the
  package's own exported duration parser about what counts as a duration; both
  now go through the same one.
- A rejected credential no longer loses its gRPC status on the way to the
  reconcile error. `WrapError` flattened the error into a new message, so every
  credential rejection reached the user as an error nothing could be classified
  from. It now wraps rather than replaces.
- `Client.Close` is idempotent. Closing a client that had already been closed
  returned an error, which a reconcile that was still holding a retired client
  would surface as a failure that said nothing useful.

### Added

- An in-process fake ZITADEL gRPC server for tests, used to drive the client
  layer against real requests and to assert that a client dials exactly one
  connection. A fake admin API and a fake management API on top of it cover the
  instance wide policies, the registration restrictions and the secret
  generators.
- `common.AllSetMatch` and `common.AllSetMatchLists`, which is where the
  "an unset field is not drift" rule now lives for every policy driver, so the
  twenty-one drivers cannot each spell it slightly differently.
- `codecov.yml`, which pins a coverage floor for the project and a target for
  new code. Coverage was previously published but never gated, so a drop to
  nothing was invisible.
- Unit tests for all seventeen policy drivers and all three instance settings
  drivers, each covering every field set and unset.
- Unit tests for the client layer's pure mappings: the login policy's five
  lifetimes and factor enums, the OIDC enums translated into Zitadel's v1
  application API, the metadata and project grant helpers, and the ten instance
  wide policies read and written through a real gRPC round trip.
- Unit tests for cross-resource reference resolution: every resolver, the
  extractors they share, and the rule that keeps a terminating resource from
  acting on a reference that has moved on.
- Unit tests for memberships and grants at all three levels - organization,
  instance and project grant - read, added, updated and removed, through the v1
  management and admin APIs.

## [0.1.0] - Unreleased

- Initial release: the tenancy, project, application, user, policy, identity
  provider, action and messaging resources described in the README, plus a
  cluster scoped form of the nineteen instance wide singletons.