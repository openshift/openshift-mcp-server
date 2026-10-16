#!/usr/bin/env bash
set -euo pipefail
umask 077

fail() { echo 'ERROR: invalid evaluation status or context binding' >&2; exit 1; }
[[ ${CONTROL_JSON:-} && ${SUITE:-} =~ ^(core|helm|kubevirt|kiali|tekton|netobserv|argocd|all)$ ]] || fail
[[ ${GITHUB_RUN_ID:-} =~ ^[1-9][0-9]{0,15}$ && ${GITHUB_RUN_ATTEMPT:-} =~ ^[1-9][0-9]{0,5}$ && ${HARNESS_SHA:-} =~ ^[a-f0-9]{40}$ ]] || fail
[[ ${CONTEXT_DIRECTORY:-} == /* && ! -e $CONTEXT_DIRECTORY && ! -L $CONTEXT_DIRECTORY && $CONTEXT_DIRECTORY != *$'\n'* && $CONTEXT_DIRECTORY != *$'\r'* ]] || fail
(( ${#CONTROL_JSON} <= 16384 )) || fail
control=$(printf '%s' "$CONTROL_JSON" | jq -ce --arg suite "$SUITE" --arg harness "$HARNESS_SHA" --argjson run "$GITHUB_RUN_ID" --argjson attempt "$GITHUB_RUN_ATTEMPT" '
    select(.version == 1 and .run_id == $run and .run_attempt == $attempt and .harness == $harness and (.suite | index($suite)) != null and
    (.candidate | type == "string" and test("^[a-f0-9]{40}$")) and (.baseline | type == "string" and test("^[a-f0-9]{40}$")) and
    (.is_pr | type == "boolean") and (.pr_number | type == "number" and floor == . and . >= 0) and (.repository | type == "string" and test("^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$")))') || fail
for status in "${CHECK_EXIT:-not-run}" "${SUMMARY_EXIT:-not-run}" "${VERIFY_EXIT:-not-run}" "${DIFF_EXIT:-not-run}"; do
    [[ $status == not-run || $status =~ ^(0|[1-9][0-9]{0,2})$ ]] || fail
done
passed=false
if [[ ${EVAL_OUTCOME:-} == success && ${EVAL_PASSED:-} == true && ${CHECK_EXIT:-} == 0 && ${SUMMARY_EXIT:-} == 0 && ${VERIFY_EXIT:-} == 0 &&
    ${SETUP_VERSION:-} == v0.0.22 && ${SETUP_CHECKER:-} == /* && ${SETUP_AGENT:-} == /* && ${RESULTS_FILE:-} == /* &&
    ${SERVER_OUTCOME:-} == success && ${REVALIDATE_OUTCOME:-} == success && ${BASELINE_OUTCOME:-} == success ]]; then
    if [[ ${HAS_DIFF:-} == true && ${DIFF_OUTCOME:-} == success && ${DIFF_EXIT:-} == 0 && ${DIFF_REASON:-} == baseline-present ]] ||
       [[ ${HAS_DIFF:-} == false && ${DIFF_OUTCOME:-} == skipped && ${DIFF_REASON:-} == baseline-absent ]]; then passed=true; fi
fi
for value in "${SETUP_CHECKER:-}" "${SETUP_AGENT:-}" "${RESULTS_FILE:-}"; do
    [[ $value != *$'\n'* && $value != *$'\r'* ]] || fail
done
metrics='null'
if [[ ${TASKS_TOTAL:-} || ${TASKS_PASSED:-} || ${TASK_PASS_RATE:-} || ${ASSERTIONS_TOTAL:-} || ${ASSERTIONS_PASSED:-} || ${ASSERTION_PASS_RATE:-} ]]; then
    if ! metrics=$(jq -cen --arg total "${TASKS_TOTAL:-}" --arg tasks "${TASKS_PASSED:-}" --arg rate "${TASK_PASS_RATE:-}" --arg assertions "${ASSERTIONS_TOTAL:-}" --arg assertionsPassed "${ASSERTIONS_PASSED:-}" --arg assertionRate "${ASSERTION_PASS_RATE:-}" '
        def count: test("^(0|[1-9][0-9]{0,9})$"); def rate: test("^(0(\\.[0-9]+)?|1(\\.0+)?)$");
        select(($total | count) and ($tasks | count) and ($assertions | count) and ($assertionsPassed | count) and ($rate | rate) and ($assertionRate | rate)) |
        {tasks_total:($total|tonumber),tasks_passed:($tasks|tonumber),task_pass_rate:($rate|tonumber),assertions_total:($assertions|tonumber),assertions_passed:($assertionsPassed|tonumber),assertion_pass_rate:($assertionRate|tonumber)} |
        select(.tasks_total > 0 and .tasks_total <= 2147483647 and .assertions_total <= 2147483647 and .tasks_passed <= .tasks_total and .assertions_passed <= .assertions_total and .task_pass_rate == (.tasks_passed / .tasks_total) and (.assertions_total == 0 or .assertion_pass_rate == (.assertions_passed / .assertions_total)))'); then
        metrics=null
        passed=false
    fi
fi
if [[ $passed == true ]]; then
    [[ $metrics != null && -f $RESULTS_FILE && ! -L $RESULTS_FILE && -s $RESULTS_FILE && ${RESULTS_FILE##*/} == "mcpchecker-$SUITE-e2e-out.json" && -x $SETUP_CHECKER && ! -L $SETUP_CHECKER && -x $SETUP_AGENT && ! -L $SETUP_AGENT ]] || passed=false
fi
mkdir -m 0700 -- "$CONTEXT_DIRECTORY"
jq -n --argjson control "$control" --arg suite "$SUITE" --argjson metrics "$metrics" --argjson passed "$passed" --arg check "${CHECK_EXIT:-not-run}" --arg summary "${SUMMARY_EXIT:-not-run}" --arg verify "${VERIFY_EXIT:-not-run}" --arg diff "${DIFF_EXIT:-not-run}" --arg hasDiff "${HAS_DIFF:-false}" --arg reason "${DIFF_REASON:-not-run}" '
    def exitCode: if . == "not-run" then null else tonumber end;
    {schema_version:1,repository:$control.repository,pr_number:$control.pr_number,pr_sha:$control.candidate,candidate:$control.candidate,harness:$control.harness,baseline:$control.baseline,run_id:$control.run_id,run_attempt:$control.run_attempt,suite:$suite,passed:$passed,has_diff:($hasDiff == "true" and $diff == "0"),diff_applicable:($hasDiff == "true"),diff_reason:$reason,check_exit:($check|exitCode),summary_exit:($summary|exitCode),verify_exit:($verify|exitCode),diff_exit:($diff|exitCode)} + ($metrics // {tasks_passed:null,tasks_total:null,task_pass_rate:null,assertions_passed:null,assertions_total:null,assertion_pass_rate:null})
    ' > "$CONTEXT_DIRECTORY/context.json"
if [[ -n ${DIFF_DIRECTORY:-} ]]; then
    [[ $DIFF_DIRECTORY == "${RUNNER_TEMP:?}"/mcpchecker-diff.* && ! -L $DIFF_DIRECTORY && -O $DIFF_DIRECTORY && -f $DIFF_DIRECTORY/diff.txt && ! -L $DIFF_DIRECTORY/diff.txt && -O $DIFF_DIRECTORY/diff.txt ]] || fail
    [[ $(cd -- "$DIFF_DIRECTORY" && pwd -P) == "$DIFF_DIRECTORY" ]] || fail
    head -c 65536 -- "$DIFF_DIRECTORY/diff.txt" > "$CONTEXT_DIRECTORY/diff-output.txt"
elif [[ -n ${DIFF_OUTPUT:-} ]]; then
    (( ${#DIFF_OUTPUT} <= 65536 )) || fail
    printf '%s' "$DIFF_OUTPUT" > "$CONTEXT_DIRECTORY/diff-output.txt"
fi
[[ $passed == true ]]
