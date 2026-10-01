#!/usr/bin/env bash

# Copyright 2025 The Crossplane Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Verify that every reference given on the command line is an OCI image index
# that covers the platforms this provider promises to support.
#
# The provider is only ever published as a multi-arch image: one entry per
# platform in the package index, each embedding that platform's controller
# image. A single-arch release would be silently un-pullable on the other
# architecture, so this is checked after every publish and promote.
#
# The manifest is read with `imagetools inspect --raw` and parsed with jq rather
# than with a `--format` template: the template context differs between buildx
# versions (older ones have no `.Manifests`), and `--raw` is the raw manifest
# bytes, so it does not change.
#
# Usage: hack/verify-multiarch.sh <image-reference>...

set -euo pipefail

# The platforms this provider is published for.
readonly PLATFORMS=(linux/amd64 linux/arm64)

# A digest or a reference is never inspected for its attestation manifests: those
# carry an "unknown/unknown" platform by design.
filter_platform() {
	jq -r '
		[ (.manifests // [])[]
		  | select(.platform.os != "unknown" and .platform.architecture != "unknown")
		  | "\(.platform.os)/\(.platform.architecture)" ]
		| unique | .[]
	'
}

failed=0

for ref in "$@"; do
	echo "==> ${ref}"

	if ! manifest="$(docker buildx imagetools inspect "${ref}" --raw 2>/dev/null)"; then
		echo "::error::${ref} could not be inspected. If it was just published, the registry may still be propagating it."
		failed=1
		continue
	fi

	# An image index is an OCI image index or a Docker manifest list. Both are
	# multi-arch, and both are valid to install from; anything else is a single
	# platform image, which is not publishable here.
	media_type="$(jq -r '.mediaType // ""' <<<"${manifest}")"
	case "${media_type}" in
	*index* | *manifest.list*) ;;
	*)
		echo "::error::${ref} is a ${media_type:-plain image manifest}, not a multi-arch image index. A single-platform image is not publishable."
		failed=1
		continue
		;;
	esac

	platforms="$(filter_platform <<<"${manifest}")"
	echo "    mediaType: ${media_type}"
	echo "    platforms: ${platforms//$'\n'/, }"

	for want in "${PLATFORMS[@]}"; do
		if ! grep -qx "${want}" <<<"${platforms}"; then
			echo "::error::${ref} is missing the ${want} platform"
			failed=1
		fi
	done
done

if [[ "${failed}" -ne 0 ]]; then
	exit 1
fi

echo "All references cover: ${PLATFORMS[*]}"
