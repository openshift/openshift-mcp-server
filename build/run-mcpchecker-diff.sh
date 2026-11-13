#!/usr/bin/env bash
set -euo pipefail
umask 077

: "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required}"
[[ -f $GITHUB_OUTPUT && ! -L $GITHUB_OUTPUT && -O $GITHUB_OUTPUT && -w $GITHUB_OUTPUT ]] || exit 1
printf 'exit-code=1\ndiff-output=\n' >> "$GITHUB_OUTPUT"
fail() { echo 'ERROR: invalid diff input or untrusted path' >&2; exit 1; }
for value in "${GITHUB_ACTION_PATH:-}" "${RUNNER_TEMP:-}" "${MCPCHECKER:-}"; do
    [[ ( $value == /* || ( ${RUNNER_OS:-} == Windows && $value =~ ^[[:alpha:]]:[/\\] ) ) && $value != *$'\n'* && $value != *$'\r'* ]] || fail
done
source_root=$(cd -- "$GITHUB_ACTION_PATH/../../.." && pwd -P)
script_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
[[ $source_root == "$script_root" && ${MCPCHECKER_VERSION:-} == v0.0.22 ]] || fail
runner_temp=$(cd -- "$RUNNER_TEMP" && pwd -P)
[[ -O $runner_temp && ! -L $RUNNER_TEMP && $runner_temp != *$'\n'* && $runner_temp != *$'\r'* ]] || fail
[[ $MCPCHECKER == "$runner_temp"/mcpchecker-action.*/bin/mcpchecker || $MCPCHECKER == "$runner_temp"/mcpchecker-action.*/bin/mcpchecker.exe ]] || fail
binary_dir=$(cd -- "$(dirname -- "$MCPCHECKER")" && pwd -P)
[[ $binary_dir == "$(dirname -- "$MCPCHECKER")" && -O $binary_dir && -O $binary_dir/.. ]] || fail
[[ -f $MCPCHECKER && -x $MCPCHECKER && ! -L $MCPCHECKER && -O $MCPCHECKER && -f $MCPCHECKER.metadata && ! -L $MCPCHECKER.metadata && -O $MCPCHECKER.metadata ]] || fail
digest=$(awk -F= '$1 == "binary_sha256" { print $2 }' "$MCPCHECKER.metadata")
if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum < "$MCPCHECKER"); else actual=$(shasum -a 256 < "$MCPCHECKER"); fi
[[ $digest =~ ^[a-f0-9]{64}$ && ${actual%% *} == "$digest" ]] || fail
working_dir=$(cd -- "${WORKING_DIRECTORY:-.}" && pwd -P)
[[ -O $working_dir && ! -L ${WORKING_DIRECTORY:-.} && $working_dir != / ]] || fail
cd -- "$working_dir"
base=${BASE:-}
current=${CURRENT:-}
[[ $base == /* ]] || base="$working_dir/$base"
[[ $current == /* ]] || current="$working_dir/$current"
for result in "$base" "$current"; do
    [[ -f $result && ! -L $result && -O $result && -s $result ]] || fail
    (( $(wc -c < "$result") <= 67108864 )) || fail
    jq -e 'def rows: if type == "array" then . else .results end; rows | type == "array" and length > 0 and all(.[]; type == "object" and (.taskPassed | type == "boolean"))' "$result" >/dev/null 2>&1 || fail
done
diagnostics=$(mktemp -d "$runner_temp/mcpchecker-diff.XXXXXXXX")
chmod 0700 "$diagnostics"
set +e
(ulimit -f 2048; "$MCPCHECKER" result diff --base "$base" --current "$current" --output markdown) > "$diagnostics/diff.txt" 2> "$diagnostics/error.txt"
diff_exit=$?
set -e
delimiter="MCPCHECKER_DIFF_${diagnostics##*.}"
while tr -d '\r' < "$diagnostics/diff.txt" | grep -Fx -- "$delimiter" >/dev/null; do delimiter="${delimiter}_X"; done
{
    printf 'exit-code=%s\ndiff-output<<%s\n' "$diff_exit" "$delimiter"
    cat -- "$diagnostics/diff.txt"
    printf '\n%s\ndiagnostics-path=%s\n' "$delimiter" "$diagnostics"
} >> "$GITHUB_OUTPUT"
exit "$diff_exit"
