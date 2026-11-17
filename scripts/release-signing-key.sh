#!/usr/bin/env bash
# Release signing with the Cloud KMS key, through github.com/coder/gpgkms.
#
#   release-signing-key.sh check                 derive the public key and compare it with
#                                                $GPGKMS_PUBLIC_KEY (the key the Terraform
#                                                Registry has)
#   release-signing-key.sh sign FILE SIGNATURE   write a binary detached signature (goreleaser
#                                                calls this for the SHA256SUMS file)
#
# Runs in release.yml's goreleaser job, which sets GPGKMS_* and Google credentials. The private
# key never leaves Cloud KMS.
#
# $GPGKMS_PUBLIC_KEY was derived with the same gpgkms version and GPGKMS_* values by someone with
# cloudkms.cryptoKeyVersions.useToSign on the key (the self-signature needs it). The derivation is
# deterministic, so `check` reproduces it byte for byte.
set -euo pipefail

identity=(
	-key "${GPGKMS_KEY:?}"
	-name "${GPGKMS_NAME:?}"
	-email "${GPGKMS_EMAIL:?}"
	-created "${GPGKMS_CREATED:?}"
)

# A keyring holding only the committed key, so a signature can only verify against it.
keyring() {
	local home
	home="$(mktemp -d)"
	gpg --homedir "$home" --batch --quiet --import "${GPGKMS_PUBLIC_KEY:?}"
	echo "$home"
}

fingerprint() {
	gpg --batch --with-colons --import-options show-only --import "${GPGKMS_PUBLIC_KEY:?}" |
		awk -F: '/^fpr/ { print $10; exit }'
}

case "${1:-}" in
check)
	derived="${RUNNER_TEMP:-/tmp}/release-signing-key.asc"
	gpgkms public-key "${identity[@]}" >"$derived"
	if [[ ! -f "${GPGKMS_PUBLIC_KEY}" ]]; then
		echo "::error::${GPGKMS_PUBLIC_KEY} does not exist."
		exit 1
	fi
	if ! cmp -s "$derived" "${GPGKMS_PUBLIC_KEY}"; then
		echo "::error::The key derived from Cloud KMS differs from ${GPGKMS_PUBLIC_KEY}. Check GPGKMS_KEY and GPGKMS_CREATED: the Terraform Registry only accepts the committed key."
		diff <(gpg --batch --with-colons --import-options show-only --import "$derived") \
			<(gpg --batch --with-colons --import-options show-only --import "${GPGKMS_PUBLIC_KEY}") || true
		exit 1
	fi
	echo "Signing key $(fingerprint) matches ${GPGKMS_PUBLIC_KEY}."
	;;
sign)
	file="${2:?file to sign}"
	signature="${3:?signature path}"
	gpgkms sign "${identity[@]}" -fingerprint "$(fingerprint)" -armor=false -out "$signature" "$file"
	home="$(keyring)"
	trap 'rm -rf "$home"' EXIT
	gpg --homedir "$home" --batch --verify "$signature" "$file"
	;;
*)
	echo "usage: $0 check | sign FILE SIGNATURE" >&2
	exit 2
	;;
esac
