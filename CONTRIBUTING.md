# Contributing

Thanks for wanting to help out!

## Getting started

```console
git submodule update --init --recursive
make generate
make build
make test
make lint
```

## What a good change looks like

* **Keep it native.** The provider talks to the Zitadel gRPC API through
  `github.com/zitadel/zitadel-go`. Do not shell out to Terraform or pull in
  Upjet.
* **Keep it namespaced.** Every managed resource lives in
  `zitadel.m.crossplane.io/v1alpha1` and is namespaced, together with its
  `ProviderConfig` in the namespace it is used from.
* **Regenerate.** After touching anything under `apis/`, run `make generate` and
  commit the generated deepcopy methods and CRDs.
* **Test it.** Prefer table driven tests (see the existing files in
  `internal/controller/*/..._test.go`). Anything that depends on Zitadel quirks
  deserves a comment explaining the quirk, and ideally a test.
* **Document quirks.** If Zitadel behaves surprisingly, say so in a comment next
  to the workaround *and* in the README, with a pointer to the upstream
  behaviour that caused it.

## Adding a new kind

```console
make provider.addtype provider=Zitadel group=zitadel kind=Foo
```

Then register the controller in `internal/controller/register.go` and add an
example under `examples/`.

## Dependencies

Dependency updates are opened by Renovate, running from
`.github/workflows/renovate.yaml` on a daily schedule. It is configured in
`renovate.json`; `.github/renovate-config.js` only holds settings that cannot
live in repository config (identity, platform, no onboarding).

Most versions are managed without any extra work: `go.mod`, the `uses:` pins in
the workflows, and the digest of the distroless base image in
`cluster/images/provider-zitadel/Dockerfile`. Versions that are written as plain
strings are managed by the `customManagers` entries in `renovate.json`:

| Where | What |
| --- | --- |
| `GO_VERSION`, `GOLANGCI_VERSION`, `DOCKER_BUILDX_VERSION`, `COSIGN_VERSION`, `CRANE_VERSION` in `.github/workflows/*.yml` | tool versions the workflows install |
| `GOLANGCILINT_VERSION`, `COSIGN_VERSION`, `GOMPLATE_VERSION` in `Makefile` | binaries the make targets download |
| `RENOVATE_VERSION` in `.github/workflows/renovate.yaml` | Renovate itself |
| `KIND_NODE_IMAGE_TAG`, `CROSSPLANE_CHART_VERSION`, `ZITADEL_CHART_VERSION` in `cluster/local/integration_tests.sh` | what the integration tests stand up |

If you pin a version somewhere new, add a `customManagers` entry for it rather
than leaving it to rot, and run the config through the validator that
`.github/workflows/renovate.yaml` runs on pull requests:

```console
npx --yes --package renovate renovate-config-validator renovate.json
```

Updates are grouped so that the version of a thing that is pinned twice (cosign
in the `Makefile` and in the release workflow, say) moves in one commit. Go
minor and patch updates arrive as one pull request, `k8s.io/*` and `sigs.k8s.io/*`
as another, and a major Go bump gets its own with a `major` label.

## Reporting bugs

Open an issue with the Zitadel version, the Crossplane version, the manifest you
applied and the provider logs at `--debug` level.
