# ====================================================================================
# Setup Project
PROJECT_NAME := provider-zitadel
PROJECT_REPO := github.com/loafoe/$(PROJECT_NAME)

# The provider package is always published as a multi-arch image index.
PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

# The GitHub repository this provider is published from, derived from the origin
# remote. Used for the registry org, the image labels and the cosign identity.
GITHUB_REPO := $(shell git config --get remote.origin.url | sed -E 's|^git@github.com[:/]||; s|^https://github.com/||; s|\.git$$||')
GITHUB_OWNER := $(word 1,$(subst /, ,$(GITHUB_REPO)))

# The image build runs in a sub-make, which needs these to label the image with
# its source repository.
export PROJECT_NAME
export PROJECT_REPO

# The build submodule only publishes artifacts when BRANCH_NAME matches
# RELEASE_BRANCH_FILTER, which by default only lists branches. This provider
# also releases from version tags, so the filter has to match v* too -
# otherwise `make publish` on a tag-triggered release silently does nothing.
RELEASE_BRANCH_FILTER ?= main master release-% v%

# ====================================================================================
# Setup Output

-include build/makelib/output.mk

# ====================================================================================
# Setup Go

NPROCS ?= 1
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis
GO111MODULE = on
GOLANGCILINT_VERSION = 2.13.2
-include build/makelib/golang.mk

# Fix Go 1.26 covdata issue - the build submodule uses a redundant -cover flag
# We override test.run to use our fixed target instead of go.test.unit
go.test.unit.fixed:
	@$(INFO) go test unit-tests
	@mkdir -p $(GO_TEST_OUTPUT)
	@CGO_ENABLED=$(GO_CGO_ENABLED) $(GOHOST) test -v -covermode=$(GO_COVER_MODE) -coverprofile=$(GO_TEST_OUTPUT)/coverage.txt $(GO_TEST_FLAGS) $(GO_STATIC_FLAGS) $(GO_PACKAGES) 2>&1 | tee $(GO_TEST_OUTPUT)/unit-tests.log || $(FAIL)
	@$(OK) go test unit-tests

test.run: go.test.unit.fixed

# ====================================================================================
# Setup Kubernetes tools

-include build/makelib/k8s_tools.mk

# ====================================================================================
# Setup Images

# Two artifacts come out of a release:
#
#   * ghcr.io/<owner>/provider-zitadel:<tag>            the Crossplane package,
#     a multi-arch image index. This is what you install, and it embeds a
#     controller image per platform, so it doubles as its own runtime.
#   * ghcr.io/<owner>/provider-zitadel:controller-<tag> the standalone
#     multi-arch controller image. The "controller-" prefix matters: a
#     Crossplane v2 package doubles as its own runtime, so an image on the
#     package tag would overwrite the package.
#
# The controller image push is also what associates the GHCR package with this
# repository, which is what makes the workflow token able to push the package.
IMAGES = provider-zitadel
REGISTRY_ORGS ?= ghcr.io/$(GITHUB_OWNER)
-include build/makelib/imagelight.mk

# ====================================================================================
# Setup Cosign

# COSIGN_VERSION is the cosign version used by the `cosign` targets. CI pins the
# same version via sigstore/cosign-installer.
COSIGN_VERSION ?= 2.6.1
COSIGN := $(TOOLS_HOST_DIR)/cosign-v$(COSIGN_VERSION)

# The keyless identity of the release workflow. This is what ends up in the
# Fulcio certificate of the signature, and what Crossplane matches on through
# examples/imageconfig-signature-verification.yaml.
COSIGN_CERT_IDENTITY ?= https://github.com/$(GITHUB_REPO)/.github/workflows/release.yml@refs/tags/$(VERSION)
COSIGN_CERT_ISSUER ?= https://token.actions.githubusercontent.com
COSIGN_OCI_REPO ?= ghcr.io/$(GITHUB_OWNER)/$(PROJECT_NAME)

# ====================================================================================
# Setup XPKG

XPKG_REG_ORGS ?= ghcr.io/$(GITHUB_OWNER)
# Channel tags (":stable") only exist on registries that do not infer vanity
# tags from the tag, so nothing is excluded here. Leaving this empty matters:
# a stale value silently turns `make promote` into a no-op.
XPKG_REG_ORGS_NO_PROMOTE ?=
XPKGS = provider-zitadel
-include build/makelib/xpkg.mk

# NOTE(hasheddan): we force image building to happen prior to xpkg build so that
# we ensure image is present in daemon.
xpkg.build.provider-zitadel: do.build.images

fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# integration tests
e2e.run: test-integration

# Run integration tests.
test-integration: $(KIND) $(KUBECTL) $(CROSSPLANE_CLI) $(HELM3)
	@$(INFO) running integration tests using kind $(KIND_VERSION)
	@KIND_NODE_IMAGE_TAG=${KIND_NODE_IMAGE_TAG} $(ROOT_DIR)/cluster/local/integration_tests.sh || $(FAIL)
	@$(OK) integration tests passed

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# NOTE(hasheddan): the build submodule currently overrides XDG_CACHE_HOME in
# order to force the Helm 3 to use the .work/helm directory. This causes Go on
# Linux machines to use that directory as the build cache as well. We should
# adjust this behavior in the build submodule because it is also causing Linux
# users to duplicate their build cache, but for now we just make it easier to
# identify its location in CI so that we cache between builds.
go.cachedir:
	@go env GOCACHE

go.mod.cachedir:
	@go env GOMODCACHE

# NOTE(hasheddan): we must ensure up is installed in tool cache prior to build
# as including the k8s_tools machinery prior to the xpkg machinery sets UP to
# point to tool cache.
build.init: $(CROSSPLANE_CLI)

# This is for running out-of-cluster locally, and is for convenience. Running
# this make target will print out the command which was used. For more control,
# try running the binary directly with different arguments.
run: go.build
	@$(INFO) Installing Provider Zitadel CRDs . . .
	@kubectl apply -R -f package/crds
	@$(INFO) Running Crossplane locally out-of-cluster . . .
	@# To see other arguments that can be provided, run the command with --help instead
	$(GO_OUT_DIR)/provider --debug

dev: $(KIND) $(KUBECTL)
	@$(INFO) Creating kind cluster
	@$(KIND) create cluster --name=$(PROJECT_NAME)-dev
	@$(KUBECTL) cluster-info --context kind-$(PROJECT_NAME)-dev
	@$(INFO) Installing Provider Zitadel CRDs
	@$(KUBECTL) apply -R -f package/crds
	@$(INFO) Starting Provider GF controllers
	@$(GO) run ./cmd/provider --debug

dev-clean: $(KIND) $(KUBECTL)
	@$(INFO) Deleting kind cluster
	@$(KIND) delete cluster --name=$(PROJECT_NAME)-dev

.PHONY: submodules fallthrough test-integration run dev dev-clean

# ====================================================================================
# Special Targets

# Sign the published package with a keyless cosign signature, exactly like the
# release workflow does. Requires a GitHub OIDC token, so it only works inside
# GitHub Actions (or with COSIGN_EXPERIMENTAL and a suitable ambient token).
sign:
	@[ "${VERSION}" ] || ( echo "argument \"VERSION\" is not set, e.g. make sign VERSION=v0.1.0"; exit 1 )
	@$(INFO) cosign sign $(COSIGN_OCI_REPO):$(VERSION)
	@cosign sign --yes \
		--certificate-identity "$(COSIGN_CERT_IDENTITY)" \
		--certificate-oidc-issuer "$(COSIGN_CERT_ISSUER)" \
		$(COSIGN_OCI_REPO):$(VERSION)
	@$(OK) signed $(COSIGN_OCI_REPO):$(VERSION)

# Verify the signature of a published package the way a consumer would.
verify-signature:
	@[ "${VERSION}" ] || ( echo "argument \"VERSION\" is not set, e.g. make verify-signature VERSION=v0.1.0"; exit 1 )
	@$(INFO) cosign verify $(COSIGN_OCI_REPO):$(VERSION)
	@cosign verify \
		--certificate-identity-regexp '^https://github\.com/$(GITHUB_OWNER)/$(PROJECT_NAME)/\.github/workflows/release\.yml@refs/tags/$(VERSION)$$' \
		--certificate-oidc-issuer "$(COSIGN_CERT_ISSUER)" \
		$(COSIGN_OCI_REPO):$(VERSION)
	@$(OK) signature of $(COSIGN_OCI_REPO):$(VERSION) is valid

$(COSIGN):
	@$(INFO) installing cosign-v$(COSIGN_VERSION) $(SAFEHOSTPLATFORM)
	@mkdir -p $(TOOLS_HOST_DIR)
	@curl -fsSLo $(TOOLS_HOST_DIR)/cosign.tgz "https://github.com/sigstore/cosign/releases/download/v$(COSIGN_VERSION)/cosign-linux-amd64.tgz" || $(FAIL)
	@tar -xzf $(TOOLS_HOST_DIR)/cosign.tgz -C $(TOOLS_HOST_DIR) cosign || $(FAIL)
	@mv $(TOOLS_HOST_DIR)/cosign $(COSIGN) || $(FAIL)
	@rm -fr $(TOOLS_HOST_DIR)/cosign.tgz
	@$(OK) installing cosign-v$(COSIGN_VERSION) $(SAFEHOSTPLATFORM)

# Install gomplate
GOMPLATE_VERSION := 3.10.0
GOMPLATE := $(TOOLS_HOST_DIR)/gomplate-$(GOMPLATE_VERSION)

$(GOMPLATE):
	@$(INFO) installing gomplate $(SAFEHOSTPLATFORM)
	@mkdir -p $(TOOLS_HOST_DIR)
	@curl -fsSLo $(GOMPLATE) https://github.com/hairyhenderson/gomplate/releases/download/v$(GOMPLATE_VERSION)/gomplate_$(SAFEHOSTPLATFORM) || $(FAIL)
	@chmod +x $(GOMPLATE)
	@$(OK) installing gomplate $(SAFEHOSTPLATFORM)

export GOMPLATE

# This target adds a new api type and its controller. You still need to
# register the controller in "internal/controller/register.go".
# Arguments:
#   provider: Camel case name of your provider, i.e. Zitadel
#   group: API group for the type, i.e. the part before ".m.crossplane.io"
#   kind: Kind of the type you want to add
#	apiversion: API version of the type you want to add. Optional and defaults to "v1alpha1"
provider.addtype: $(GOMPLATE)
	@[ "${provider}" ] || ( echo "argument \"provider\" is not set"; exit 1 )
	@[ "${group}" ] || ( echo "argument \"group\" is not set"; exit 1 )
	@[ "${kind}" ] || ( echo "argument \"kind\" is not set"; exit 1 )
	@PROVIDER=$(provider) GROUP=$(group) KIND=$(kind) APIVERSION=$(apiversion) PROJECT_REPO=$(PROJECT_REPO) ./hack/helpers/addtype.sh

define CROSSPLANE_MAKE_HELP
Crossplane Targets:
    submodules            Update the submodules, such as the common build scripts.
    run                   Run crossplane locally, out-of-cluster. Useful for development.

endef
# The reason CROSSPLANE_MAKE_HELP is used instead of CROSSPLANE_HELP is because the crossplane
# binary will try to use CROSSPLANE_HELP if it is set, and this is for something different.
export CROSSPLANE_MAKE_HELP

crossplane.help:
	@echo "$$CROSSPLANE_MAKE_HELP"

help-special: crossplane.help

.PHONY: crossplane.help help-special
