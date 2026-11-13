#!/usr/bin/env bash
set -euo pipefail

version=${MCPCHECKER_VERSION:-v0.0.22}
if [[ $version != v0.0.22 ]]; then
    echo "ERROR: action setup supports only v0.0.22" >&2
    exit 1
fi
: "${GITHUB_ACTION_PATH:?GITHUB_ACTION_PATH is required}"
: "${RUNNER_TEMP:?RUNNER_TEMP is required}"
: "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required}"
: "${GITHUB_PATH:?GITHUB_PATH is required}"
for directory in "$GITHUB_ACTION_PATH" "$RUNNER_TEMP"; do
    if [[ $directory == *$'\n'* || $directory == *$'\r'* ]]; then
        echo "ERROR: action source and runner temp cannot contain newlines" >&2
        exit 1
    fi
    if [[ $directory != /* && ! ( ${RUNNER_OS:-} == Windows && $directory =~ ^[[:alpha:]]:[/\\] ) ]]; then
        echo "ERROR: action source and runner temp must be absolute paths" >&2
        exit 1
    fi
done
source_root=$(cd -- "$GITHUB_ACTION_PATH/../../.." && pwd -P)
script_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
if [[ $source_root != "$script_root" ]]; then
    echo "ERROR: action source must match the trusted setup source" >&2
    exit 1
fi
runner_temp=$(cd -- "$RUNNER_TEMP" && pwd -P)
if [[ $runner_temp == *$'\n'* || $runner_temp == *$'\r'* ]]; then
    echo "ERROR: runner temp cannot contain newlines" >&2
    exit 1
fi
for output_file in "$GITHUB_OUTPUT" "$GITHUB_PATH"; do
    if [[ ! -f $output_file || -L $output_file || ! -w $output_file ]]; then
        echo "ERROR: GitHub output files must be writable regular files" >&2
        exit 1
    fi
done
case "${RUNNER_OS:-}/${RUNNER_ARCH:-}" in
    Linux/X64) goos=linux; goarch=amd64 ;;
    Linux/ARM64) goos=linux; goarch=arm64 ;;
    macOS/X64) goos=darwin; goarch=amd64 ;;
    macOS/ARM64) goos=darwin; goarch=arm64 ;;
    Windows/X64) goos=windows; goarch=amd64 ;;
    Windows/ARM64) goos=windows; goarch=arm64 ;;
    *) echo "ERROR: unsupported runner platform" >&2; exit 1 ;;
esac
install_dir=$(mktemp -d "$runner_temp/mcpchecker-action.XXXXXXXX")
chmod 0700 "$install_dir"
complete=false
cleanup() {
    if [[ $complete != true ]]; then rm -rf -- "$install_dir"; fi
}
trap cleanup EXIT
mkdir -m 0700 "$install_dir/bin"
extension=
if [[ $goos == windows ]]; then extension=.exe; fi
for component in mcpchecker agent; do
    bash "$source_root/build/install-mcpchecker.sh" "$version" "$goos" "$goarch" \
        "$install_dir/bin/$component$extension" "$source_root/build/mcpchecker-v0.0.22.sha256" "$component"
done
printf 'mcpchecker-path=%s\nagent-path=%s\nversion=%s\n' \
    "$install_dir/bin/mcpchecker$extension" "$install_dir/bin/agent$extension" "$version" >> "$GITHUB_OUTPUT"
printf '%s\n' "$install_dir/bin" >> "$GITHUB_PATH"
complete=true
