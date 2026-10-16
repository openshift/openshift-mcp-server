#!/usr/bin/env bash
set -euo pipefail
umask 077

: "${GITHUB_OUTPUT:?GITHUB_OUTPUT is required}"
[[ -f $GITHUB_OUTPUT && ! -L $GITHUB_OUTPUT && -O $GITHUB_OUTPUT && -w $GITHUB_OUTPUT ]] || exit 1
printf 'passed=false\nresults-file=\ncheck-exit=not-run\nsummary-exit=not-run\nverify-exit=not-run\nartifact-paths=\n' >> "$GITHUB_OUTPUT"
fail() { echo 'ERROR: invalid action input or untrusted path' >&2; exit 1; }
if ! command -v node >/dev/null || ! command -v python3 >/dev/null || [[ $(uname -s) != Linux ]]; then
    echo 'ERROR: evaluation requires Linux, Node.js and Python 3 on PATH' >&2
    exit 1
fi
for value in "${GITHUB_ACTION_PATH:-}" "${RUNNER_TEMP:-}" "${MCPCHECKER:-}" "${AGENT:-}"; do
    [[ ( $value == /* || ( ${RUNNER_OS:-} == Windows && $value =~ ^[[:alpha:]]:[/\\] ) ) && $value != *$'\n'* && $value != *$'\r'* ]] || fail
done
source_root=$(cd -- "$GITHUB_ACTION_PATH/../../.." && pwd -P)
script_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd -P)
[[ $source_root == "$script_root" ]] || fail
runner_temp=$(cd -- "$RUNNER_TEMP" && pwd -P)
[[ -O $runner_temp && ! -L $RUNNER_TEMP && $runner_temp != *$'\n'* && $runner_temp != *$'\r'* ]] || fail
[[ ${MCPCHECKER_VERSION:-} == v0.0.22 ]] || fail
[[ $MCPCHECKER == "$runner_temp"/mcpchecker-action.*/bin/mcpchecker || $MCPCHECKER == "$runner_temp"/mcpchecker-action.*/bin/mcpchecker.exe ]] || fail
binary_dir=$(cd -- "$(dirname -- "$MCPCHECKER")" && pwd -P)
[[ $binary_dir == "$(dirname -- "$MCPCHECKER")" && -O $binary_dir && -O $binary_dir/.. ]] || fail
extension=${MCPCHECKER##*/mcpchecker}
[[ $AGENT == "$binary_dir/agent$extension" ]] || fail
for binary in "$MCPCHECKER" "$AGENT"; do
    [[ -f $binary && -x $binary && ! -L $binary && -O $binary && -f $binary.metadata && ! -L $binary.metadata && -O $binary.metadata ]] || fail
    digest=$(awk -F= '$1 == "binary_sha256" { print $2 }' "$binary.metadata")
    [[ $digest =~ ^[a-f0-9]{64}$ ]] || fail
    if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum < "$binary"); else actual=$(shasum -a 256 < "$binary"); fi
    [[ ${actual%% *} == "$digest" ]] || fail
done
for boolean in "${VERBOSE-false}" "${FAIL_ON_ERROR-true}" "${UPLOAD_ARTIFACTS-true}"; do
    [[ $boolean == true || $boolean == false ]] || fail
done
[[ ${OUTPUT_FORMAT-json} == json || ${OUTPUT_FORMAT-json} == text ]] || fail
[[ ${PARALLEL-1} =~ ^[1-9][0-9]{0,2}$ ]] || fail
for threshold in "${TASK_THRESHOLD-1.0}" "${ASSERTION_THRESHOLD-1.0}"; do
    [[ $threshold =~ ^(0(\.[0-9]+)?|1(\.0+)?)$ ]] || fail
done
working_dir=$(cd -- "${WORKING_DIRECTORY:-.}" && pwd -P)
[[ -O $working_dir && $working_dir != / && $working_dir != *$'\n'* && $working_dir != *$'\r'* && ! -L ${WORKING_DIRECTORY:-.} ]] || fail
cd -- "$working_dir"
config=${EVAL_CONFIG:-}
[[ $config != /* ]] && config="$working_dir/$config"
[[ -f $config && ! -L $config && -O $config ]] || fail
expected=${EXPECTED_RESULTS_FILE:-}
[[ -z $expected || $expected =~ ^mcpchecker-[a-zA-Z0-9][a-zA-Z0-9._-]*-out\.json$ ]] || fail
shopt -s nullglob dotglob
existing=(mcpchecker-*-out.json *-error.txt)
(( ${#existing[@]} == 0 )) || fail
diagnostics=$(mktemp -d "$runner_temp/mcpchecker-diagnostics.XXXXXXXX")
chmod 0700 "$diagnostics"
trap 'rm -f -- "$diagnostics/raw-summary.json"' EXIT
capture() {
    local destination=$1
    shift
    local statuses checker_exit helper_exit expired supervisor_exit
    local summary=()
    [[ $destination != "$diagnostics/summary.txt" ]] || summary=("$diagnostics/raw-summary.json")
    set +e
    statuses=$(python3 "$script_root/build/mcpchecker-capture.py" "$destination" "${summary[0]:-}" "$@")
    supervisor_exit=$?
    set -e
    read -r checker_exit helper_exit expired <<< "$statuses"
    printf 'capture checker-exit=%s diagnostic-exit=%s expired=%s\n' "$checker_exit" "$helper_exit" "$expired" >&2
    [[ $supervisor_exit == 0 && $helper_exit == 0 && $checker_exit =~ ^[0-9]+$ && $expired =~ ^[01]$ ]] || { rm -f -- "$destination" "$diagnostics/raw-summary.json"; fail; }
    captured_exit=$checker_exit
    [[ $expired == 0 ]] || captured_exit=124
}
arguments=("$MCPCHECKER" check "--output=${OUTPUT_FORMAT:-json}" "--parallel=${PARALLEL:-1}")
[[ -z ${TASK_FILTER:-} ]] || arguments+=("--run=$TASK_FILTER")
[[ -z ${LABEL_SELECTOR:-} ]] || arguments+=("--label-selector=$LABEL_SELECTOR")
[[ ${VERBOSE:-false} != true ]] || arguments+=(--verbose)
arguments+=(-- "$config")
capture "$diagnostics/check.txt" "${arguments[@]}"
check_exit=$captured_exit
printf 'check-exit=%s\n' "$check_exit" >> "$GITHUB_OUTPUT"
summary_exit=not-run
verify_exit=not-run
valid=false
results=(mcpchecker-*-out.json)
if (( ${#results[@]} == 1 )); then
    result=${results[0]}
    if [[ ( -z $expected || $result == "$expected" ) && $result =~ ^mcpchecker-[a-zA-Z0-9][a-zA-Z0-9._-]*-out\.json$ && -f $result && ! -L $result && -O $result && -s $result ]] && (( $(wc -c < "$result") <= 67108864 )); then
        if jq -e 'def rows: if type == "array" then . else .results end; rows | type == "array" and length > 0 and all(.[]; type == "object" and (.taskPassed | type == "boolean"))' "$result" >/dev/null 2>&1; then
            if jq -e 'def secrets: [env.MODEL_KEY, env.JUDGE_API_KEY, env.OPENAI_API_KEY, env.ANTHROPIC_API_KEY, env.GOOGLE_API_KEY, env.GEMINI_API_KEY, env.AZURE_OPENAI_API_KEY] | map(select(type == "string" and length > 0)); secrets as $keys | if any(.. | objects | keys[]; . as $name | any($keys[]; . as $key | $name | contains($key))) then error("secret in object key") else walk(if type == "string" then reduce $keys[] as $key (. ; split($key) | join("[REDACTED]")) else . end) | if any(.. | strings; . as $text | any($keys[]; . as $key | $text | contains($key))) then error("redaction failed") else . end end' "$result" > "$diagnostics/result.json" 2>/dev/null; then
                mv -- "$diagnostics/result.json" "$result"
                valid=true
            else
                rm -f -- "$diagnostics/result.json"
            fi
        fi
    fi
fi
for error_file in *-error.txt; do
    [[ -f $error_file && ! -L $error_file && -O $error_file ]] || { valid=false; continue; }
    owned_error=$(mktemp "$diagnostics/task-error.XXXXXXXX")
    node "$script_root/build/mcpchecker-diagnostics.cjs" "$owned_error" < "$error_file" || { rm -f -- "$owned_error"; valid=false; }
done
if [[ $valid == true ]]; then
    capture "$diagnostics/summary.txt" "$MCPCHECKER" result summary "$working_dir/$result" --output=json
    summary_exit=$captured_exit
    if [[ $summary_exit == 0 ]] && jq -e '
        def count: type == "number" and . >= 0 and . <= 2147483647 and floor == .;
        def rate: type == "number" and . >= 0 and . <= 1;
        (.tasksTotal | count) and .tasksTotal > 0 and (.tasksPassed | count) and .tasksPassed <= .tasksTotal and
        (.assertionsTotal | count) and (.assertionsPassed | count) and .assertionsPassed <= .assertionsTotal and
        (.taskPassRate | rate) and (.assertionPassRate | rate) and
        .taskPassRate == (.tasksPassed / .tasksTotal) and
        (.assertionsTotal == 0 or .assertionPassRate == (.assertionsPassed / .assertionsTotal))
        ' "$diagnostics/raw-summary.json" >/dev/null 2>&1 && [[ $(jq -r .tasksTotal "$diagnostics/raw-summary.json") == $(jq 'if type == "array" then length else .results | length end' "$result") ]]; then
        jq -r '"tasks-total=\(.tasksTotal)\ntasks-passed=\(.tasksPassed)\ntask-pass-rate=\(.taskPassRate)\nassertions-total=\(.assertionsTotal)\nassertions-passed=\(.assertionsPassed)\nassertion-pass-rate=\(.assertionPassRate)"' "$diagnostics/raw-summary.json" >> "$GITHUB_OUTPUT"
        capture "$diagnostics/verify.txt" "$MCPCHECKER" result verify "$working_dir/$result" "--task=${TASK_THRESHOLD:-1.0}" "--assertion=${ASSERTION_THRESHOLD:-1.0}"
        verify_exit=$captured_exit
    else
        [[ $summary_exit != 0 ]] || summary_exit=1
    fi
    printf 'results-file=%s\n' "$working_dir/$result" >> "$GITHUB_OUTPUT"
fi
if [[ -f $diagnostics/raw-summary.json ]]; then
    node "$script_root/build/mcpchecker-diagnostics.cjs" "$diagnostics/summary.txt" < "$diagnostics/raw-summary.json" || fail
fi
rm -f -- "$diagnostics/raw-summary.json"
printf 'summary-exit=%s\nverify-exit=%s\n' "$summary_exit" "$verify_exit" >> "$GITHUB_OUTPUT"
[[ $valid != true ]] || cp -- "$result" "$diagnostics/$result"
printf 'artifact-paths=%s\n' "$diagnostics" >> "$GITHUB_OUTPUT"
if [[ $check_exit == 0 && $valid == true && $summary_exit == 0 && $verify_exit == 0 ]]; then
    printf 'passed=true\n' >> "$GITHUB_OUTPUT"
elif [[ ${FAIL_ON_ERROR:-true} == true ]]; then
    exit 1
fi
