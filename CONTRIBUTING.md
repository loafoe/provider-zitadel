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

## Reporting bugs

Open an issue with the Zitadel version, the Crossplane version, the manifest you
applied and the provider logs at `--debug` level.
