#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 5 ]]; then
    echo "Usage: $0 VERSION GOOS GOARCH DESTINATION CHECKSUM_MANIFEST" >&2
    exit 2
fi

version=$1
goos=$2
goarch=$3
destination=$4
checksum_manifest=$5

case "${goos}/${goarch}" in
    linux/amd64|linux/arm64|darwin/amd64|darwin/arm64|windows/amd64|windows/arm64)
        ;;
    *)
        echo "ERROR: mcpchecker ${version} has no verified release binary for ${goos}/${goarch}" >&2
        echo "Supported platforms: linux, darwin, and windows on amd64 or arm64" >&2
        exit 1
        ;;
esac

binary_name=mcpchecker
if [[ ${goos} == windows ]]; then
    binary_name=mcpchecker.exe
fi
asset="mcpchecker-${goos}-${goarch}.zip"

if [[ ! -f ${checksum_manifest} ]]; then
    echo "ERROR: no committed checksum manifest for mcpchecker ${version}: ${checksum_manifest}" >&2
    echo "Add and review build/mcpchecker-${version}.sha256 before using this version" >&2
    exit 1
fi

matching_checksum_count=$(awk -v asset="${asset}" '$2 == asset { count++ } END { print count + 0 }' "${checksum_manifest}")
if [[ ${matching_checksum_count} -ne 1 ]]; then
    echo "ERROR: expected exactly one committed checksum for ${asset} in ${checksum_manifest}" >&2
    exit 1
fi
expected_archive_sha=$(awk -v asset="${asset}" '$2 == asset { print $1 }' "${checksum_manifest}")
if [[ ! ${expected_archive_sha} =~ ^[[:xdigit:]]{64}$ ]]; then
    echo "ERROR: invalid SHA-256 for ${asset} in ${checksum_manifest}" >&2
    exit 1
fi
expected_archive_sha=$(printf '%s' "${expected_archive_sha}" | tr '[:upper:]' '[:lower:]')

sha256_file() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{ print $1 }'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$1" | awk '{ print $1 }'
    else
        echo "ERROR: sha256sum or shasum is required to verify mcpchecker" >&2
        return 1
    fi
}

metadata_file="${destination}.metadata"
cache_key="${version}|${goos}/${goarch}|${expected_archive_sha}"

reject_directory_final_paths() {
    if [[ -d ${destination} ]]; then
        echo "ERROR: mcpchecker binary destination must not be a directory: ${destination}" >&2
        return 1
    fi
    if [[ -d ${metadata_file} ]]; then
        echo "ERROR: mcpchecker metadata destination must not be a directory: ${metadata_file}" >&2
        return 1
    fi
}

for command in mkdir sleep; do
    if ! command -v "${command}" >/dev/null 2>&1; then
        echo "ERROR: ${command} is required to lock the mcpchecker installation" >&2
        exit 1
    fi
done

lock_timeout_seconds=${MCPCHECKER_LOCK_TIMEOUT_SECONDS:-30}
if [[ ! ${lock_timeout_seconds} =~ ^[[:digit:]]{1,3}$ ]] || (( 10#${lock_timeout_seconds} > 300 )); then
    echo "ERROR: MCPCHECKER_LOCK_TIMEOUT_SECONDS must be an integer from 0 through 300" >&2
    exit 1
fi
lock_timeout_seconds=$((10#${lock_timeout_seconds}))

destination_dir=$(dirname "${destination}")
mkdir -p "${destination_dir}"
lock_dir="${destination}.lock"
lock_acquired=0
temp_dir=
temporary_binary=
temporary_metadata=
cleanup() {
    if [[ -n ${temp_dir} ]]; then
        rm -rf "${temp_dir}"
    fi
    if [[ -n ${temporary_binary} ]]; then
        rm -f "${temporary_binary}"
    fi
    if [[ -n ${temporary_metadata} ]]; then
        rm -f "${temporary_metadata}"
    fi
    if (( lock_acquired )); then
        rm -rf "${lock_dir}"
    fi
}
trap cleanup EXIT

lock_deadline=$((SECONDS + lock_timeout_seconds))
while ! mkdir "${lock_dir}" 2>/dev/null; do
    if (( SECONDS >= lock_deadline )); then
        echo "ERROR: timed out after ${lock_timeout_seconds}s waiting for mcpchecker installer lock: ${lock_dir}" >&2
        echo "A normally exiting installer removes its lock; an uncatchable termination can leave one behind." >&2
        echo "Inspect active installers before removing a stranded lock directory." >&2
        exit 1
    fi
    sleep 1
done
lock_acquired=1
printf 'pid=%s\ncache_key=%s\n' "$$" "${cache_key}" > "${lock_dir}/owner"

reject_directory_final_paths

if [[ -x ${destination} && -f ${metadata_file} ]]; then
    stored_cache_key=$(awk -F= '$1 == "cache_key" { print substr($0, index($0, "=") + 1) }' "${metadata_file}")
    stored_binary_sha=$(awk -F= '$1 == "binary_sha256" { print $2 }' "${metadata_file}")
    if [[ ${stored_cache_key} == "${cache_key}" && ${stored_binary_sha} =~ ^[[:xdigit:]]{64}$ ]]; then
        current_binary_sha=$(sha256_file "${destination}")
        current_binary_sha=$(printf '%s' "${current_binary_sha}" | tr '[:upper:]' '[:lower:]')
        stored_binary_sha=$(printf '%s' "${stored_binary_sha}" | tr '[:upper:]' '[:lower:]')
        if [[ ${current_binary_sha} == "${stored_binary_sha}" ]]; then
            echo "mcpchecker ${version} (${goos}/${goarch}) matches its cached metadata and binary digest"
            exit 0
        fi
    fi
    echo "Cached mcpchecker metadata is stale or its binary changed; reinstalling"
fi

for command in curl unzip mktemp; do
    if ! command -v "${command}" >/dev/null 2>&1; then
        echo "ERROR: ${command} is required to install mcpchecker" >&2
        exit 1
    fi
done

temp_dir=$(mktemp -d "${destination_dir}/.mcpchecker-install.XXXXXX")

archive="${temp_dir}/${asset}"
release_url="https://github.com/mcpchecker/mcpchecker/releases/download/${version}/${asset}"
echo "Downloading mcpchecker ${version} (${goos}/${goarch}) from the official GitHub release"
# Bound both connection establishment and the complete transfer while the
# per-destination installation lock is held.
curl --proto '=https' --tlsv1.2 --connect-timeout 10 --max-time 300 -fsSL "${release_url}" -o "${archive}"

actual_archive_sha=$(sha256_file "${archive}")
actual_archive_sha=$(printf '%s' "${actual_archive_sha}" | tr '[:upper:]' '[:lower:]')
if [[ ${actual_archive_sha} != "${expected_archive_sha}" ]]; then
    echo "ERROR: checksum mismatch for ${asset}" >&2
    echo "Expected: ${expected_archive_sha}" >&2
    echo "Actual:   ${actual_archive_sha}" >&2
    exit 1
fi
echo "Verified ${asset} SHA-256: ${expected_archive_sha}"

extract_dir="${temp_dir}/extract"
mkdir -p "${extract_dir}"
unzip -q "${archive}" -d "${extract_dir}"
extracted_binary="${extract_dir}/${binary_name}"
if [[ ! -f ${extracted_binary} ]]; then
    echo "ERROR: verified archive ${asset} does not contain ${binary_name}" >&2
    exit 1
fi
chmod 0755 "${extracted_binary}"
binary_sha=$(sha256_file "${extracted_binary}")

temporary_binary=$(mktemp "${destination}.tmp.XXXXXX")
temporary_metadata=$(mktemp "${metadata_file}.tmp.XXXXXX")
cp "${extracted_binary}" "${temporary_binary}"
chmod 0755 "${temporary_binary}"
printf 'cache_key=%s\nbinary_sha256=%s\n' "${cache_key}" "${binary_sha}" > "${temporary_metadata}"
chmod 0644 "${temporary_metadata}"

# The per-destination lock serializes cache checks and both atomic renames.
reject_directory_final_paths
mv -f "${temporary_binary}" "${destination}"
temporary_binary=
mv -f "${temporary_metadata}" "${metadata_file}"
temporary_metadata=

echo "Installed checksum-verified mcpchecker ${version} to ${destination}"
