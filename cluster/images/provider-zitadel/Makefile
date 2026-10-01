# ====================================================================================
# Setup Project

include ../../../build/makelib/common.mk

# ====================================================================================
#  Options

include ../../../build/makelib/imagelight.mk

# ====================================================================================
# Targets

img.build:
	@$(INFO) docker build $(IMAGE)
	@$(MAKE) BUILD_ARGS="--load" img.build.shared
	@$(OK) docker build $(IMAGE)

# The controller image is not published on its own.
#
# A Crossplane v2 provider package embeds its controller image: the xpkg is
# built with `--embed-runtime-image` and the package image doubles as the
# runtime image, so `spec.controller.image` stays empty. The published
# `ghcr.io/loafoe/provider-zitadel:<tag>` reference is therefore the multi-arch
# *package* index - publishing the controller image under the same tag would
# overwrite the package.
#
# The per-platform controller images are still built here (that is what
# `--embed-runtime-image` reads from the local daemon) and they are always
# multi-arch in the sense that matters: the package index carries one
# controller image per platform.
img.publish:
	@$(INFO) Skipping image publish for $(IMAGE)
	@echo The controller image is embedded in the Crossplane package, which is published as $(IMAGE) by the xpkg machinery.
	@$(OK) Image publish skipped

img.build.shared:
	@cp Dockerfile $(IMAGE_TEMP_DIR) || $(FAIL)
	@cp -r $(OUTPUT_DIR)/bin/ $(IMAGE_TEMP_DIR)/bin || $(FAIL)
	@docker buildx build $(BUILD_ARGS) \
		--platform $(IMAGE_PLATFORMS) \
		-t $(IMAGE) \
		$(IMAGE_TEMP_DIR) || $(FAIL)

img.promote:
	@$(INFO) Skipping image promotion for $(IMAGE)
	@echo The controller image is embedded in the Crossplane package, which is promoted by the xpkg machinery.
	@$(OK) Image promotion skipped
