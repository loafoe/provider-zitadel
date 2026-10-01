# ====================================================================================
# Setup Project

include ../../../build/makelib/common.mk

# ====================================================================================
#  Options

include ../../../build/makelib/imagelight.mk

# ====================================================================================
# Targets

# The published controller image carries the label that associates this GHCR
# package with the repository. GitHub only lets the workflow's GITHUB_TOKEN push
# to a package that is associated with the repository the workflow runs in, and
# the association is established by the org.opencontainers.image.source label.
# Without this step, a first push from CI to a package that was created outside
# of CI fails with "403 Forbidden" on the existing blobs.
# PROJECT_REPO is already the fully qualified "github.com/<owner>/<repo>".
SOURCE_LABEL ?= org.opencontainers.image.source=https://$(PROJECT_REPO)
REVISION_LABEL ?= org.opencontainers.image.revision=$(VERSION)
VERSION_LABEL ?= org.opencontainers.image.version=$(VERSION)
TITLE_LABEL ?= org.opencontainers.image.title=$(PROJECT_NAME)
DESCRIPTION_LABEL ?= org.opencontainers.image.description=A native Crossplane provider for Zitadel.

img.build:
	@$(INFO) docker build $(IMAGE)
	@$(MAKE) BUILD_ARGS="--load" img.build.shared
	@$(OK) docker build $(IMAGE)

# The Crossplane package embeds a controller image, so the package is its own
# runtime and Crossplane needs no separate image reference. The controller image
# is still published, under a "controller-" prefixed tag, because:
#
#   * it is the image the package embeds, so publishing it lets someone run the
#     provider out of cluster, e.g. `make run` against a kubeconfig;
#   * it is the push that associates the GHCR package with this repository,
#     without which the workflow token cannot push the package itself;
#   * it gives the release pipeline a second, independently verifiable artifact.
#
# The tag must not be the package tag: a Crossplane v2 package doubles as its
# own runtime image, so pushing an image on the package tag would overwrite the
# package.
CONTROLLER_IMAGE = $(subst :,:controller-,$(IMAGE))

img.publish:
	@$(INFO) Publishing multi-arch controller image $(CONTROLLER_IMAGE)
	@$(MAKE) BUILD_ARGS="--push" IMAGE=$(CONTROLLER_IMAGE) img.build.shared
	@$(OK) Published multi-arch controller image $(CONTROLLER_IMAGE)

img.build.shared:
	@cp Dockerfile $(IMAGE_TEMP_DIR) || $(FAIL)
	@cp -r $(OUTPUT_DIR)/bin/ $(IMAGE_TEMP_DIR)/bin || $(FAIL)
	@docker buildx build $(BUILD_ARGS) \
		--platform $(IMAGE_PLATFORMS) \
		--label "$(SOURCE_LABEL)" \
		--label "$(REVISION_LABEL)" \
		--label "$(VERSION_LABEL)" \
		--label "$(TITLE_LABEL)" \
		--label "$(DESCRIPTION_LABEL)" \
		-t $(IMAGE) \
		$(IMAGE_TEMP_DIR) || $(FAIL)

# The build machinery hands us the package tags; the controller image lives under
# the same tags with a "controller-" prefix (see img.publish).
CONTROLLER_FROM_IMAGE = $(subst :,:controller-,$(FROM_IMAGE))
CONTROLLER_TO_IMAGE = $(subst :,:controller-,$(TO_IMAGE))

img.promote:
	@$(INFO) Promoting multi-arch controller image from $(CONTROLLER_FROM_IMAGE) to $(CONTROLLER_TO_IMAGE)
	@docker buildx imagetools create -t $(CONTROLLER_TO_IMAGE) $(CONTROLLER_FROM_IMAGE) || $(FAIL)
	@$(OK) Promoted multi-arch controller image from $(CONTROLLER_FROM_IMAGE) to $(CONTROLLER_TO_IMAGE)
