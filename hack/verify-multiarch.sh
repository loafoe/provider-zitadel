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

# Verify that every reference given on the command line is a multi-arch OCI
# image index covering the platforms this provider promises to support.
#
# The provider is only ever published as a multi-arch image: one entry per
# platform in the package index, each embedding that platform's controller
# image. A single-arch release would be silently un-pullable on the other
# architecture, so this runs after every publish and promote.
#
# The manifest is read as raw bytes and parsed with jq, never with a buildx
# `--format` template: the template context differs between buildx versions
# (older ones have no `.Manifests`) and fails at evaluation time.
#
# crane is preferred when installed because it can read a public registry
# anonymously. `docker buildx imagetools inspect` needs a docker login even for a
# public image, and the main consumer of this script is a person checking a
# public release without a registry credential.
#
# Usage: hack/verify-multiarch.sh <image-reference>...

set -euo pipefail

# The platforms this provider is published for.
readonly PLATFORMS=(linux/amd64 linux/arm64)

# read_manifest writes the raw manifest of a reference to stdout, and on failure
# writes the reason to stderr, so an unreadable reference is actionable instead
# of a bare "could not be inspected".
read_manifest() {
	local ref="$1"

	if command -v crane >/dev/null 2>&1 && crane manifest "${ref}" 2>/dev/null; then
		return 0
	fi

	if docker buildx imagetools inspect "${ref}" --raw 2>/dev/null; then
		return 0
	fi

	# Say which of the possible causes it is: this is the difference between
	# "the release is broken" and "you have no tooling".
	if ! command -v docker >/dev/null 2>&1; then
		echo "neither crane nor docker is installed" >&2
	elif ! docker buildx version >/dev/null 2>&1; then
		echo "docker buildx is unavailable; install the buildx CLI plugin" >&2
	else
		echo "the registry refused the request; the reference may not exist, or the registry may require a login even for a public image" >&2
	fi

	return 1
}

# platforms_of prints the linux platforms an index covers.
#
# Attestation manifests (SBOM, provenance) are excluded: buildx gives them an
# "unknown/unknown" platform by design, and they are not something anyone pulls.
platforms_of() {
	jq -r '
		[ (.manifests // [])[]
		  | select(.platform.os != "unknown" and .platform.architecture != "unknown")
		  | "\(.platform.os)/\(.platform.architecture)" ]
		| unique | .[]
	'
}

manifest_file="$(mktemp)"
reason_file="$(mktemp)"
# shellcheck disable=SC2064 # expand now, the variables outlive the trap body
trap "rm -f '${manifest_file}' '${reason_file}'" EXIT

failed=0

for ref in "$@"; do
	echo "==> ${ref}"

	if ! read_manifest "${ref}" >"${manifest_file}" 2>"${reason_file}"; then
		echo "::error::${ref} could not be read: $(<"${reason_file}")"
		failed=1
		continue
	fi

	# An image index is either an OCI image index or a Docker manifest list.
	# Both are multi-arch and both are valid to install from; anything else is a
	# single platform image, which is not publishable here.
	media_type="$(jq -r '.mediaType // ""' <"${manifest_file}")"
	case "${media_type}" in
	*index* | *manifest.list*) ;;
	*)
		echo "::error::${ref} is a ${media_type:-plain image manifest}, not a multi-arch image index. A single-platform image is not publishable."
		failed=1
		continue
		;;
	esac

	platforms="$(platforms_of <"${manifest_file}")"
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
