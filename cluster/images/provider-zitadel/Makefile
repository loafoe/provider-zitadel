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

# The provider is shipped as a self contained, multi-arch Crossplane package: the
# xpkg embeds the controller image of every platform it was built for, and the
# published OCI image is an image index covering all of them. We still publish
# the standalone multi-arch controller image so that the provider can also be run
# outside of Crossplane (e.g. out of cluster against a kubeconfig).
img.publish:
	@$(INFO) Publishing multi-arch image $(IMAGE)
	@$(MAKE) BUILD_ARGS="--push" img.build.shared
	@$(OK) Published multi-arch image $(IMAGE)

img.build.shared:
	@cp Dockerfile $(IMAGE_TEMP_DIR) || $(FAIL)
	@cp -r $(OUTPUT_DIR)/bin/ $(IMAGE_TEMP_DIR)/bin || $(FAIL)
	@docker buildx build $(BUILD_ARGS) \
		--platform $(IMAGE_PLATFORMS) \
		-t $(IMAGE) \
		$(IMAGE_TEMP_DIR) || $(FAIL)

img.promote:
	@$(INFO) Promoting multi-arch image from $(FROM_IMAGE) to $(TO_IMAGE)
	@docker buildx imagetools create -t $(TO_IMAGE) $(FROM_IMAGE) || $(FAIL)
	@$(OK) Promoted multi-arch image from $(FROM_IMAGE) to $(TO_IMAGE)
