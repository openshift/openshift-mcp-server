#!/usr/bin/env bash
# Fixture checks for hack/upstream-sync.rules.
#
# Sources sync-upstream.sh (whose main guard keeps sourcing side-effect free)
# and asserts that every fixture path matches exactly its expected set of
# rule kinds, and that malformed rules files are rejected. Run through
# make check-upstream-sync-rules.

set -Eeuo pipefail

TEST_ROOT=${BASH_SOURCE[0]%/*}
[[ "$TEST_ROOT" != "${BASH_SOURCE[0]}" ]] || TEST_ROOT=.
TEST_ROOT=$(cd -- "$TEST_ROOT" && pwd)
FIXTURES_FILE=$TEST_ROOT/upstream-sync.rules.fixtures

# shellcheck source=sync-upstream.sh
source "$TEST_ROOT/sync-upstream.sh"

pass=0
failed=0

note_failure() {
	printf '%s: FAIL: %s\n' "${0##*/}" "$*" >&2
	failed=$((failed + 1))
}

kind_in_list() {
	local needle=$1 item
	shift
	for item in "$@"; do
		if [[ "$item" == "$needle" ]]; then
			return 0
		fi
	done
	return 1
}

check_fixture() {
	local path=$1 expected=$2 kind ok=0
	local -a matched=() expected_kinds=()
	for kind in "${KNOWN_RULE_KINDS[@]}"; do
		if rule_matches "$path" "$kind"; then
			matched+=("$kind")
		fi
	done
	if [[ "$expected" == "-" ]]; then
		if (( ${#matched[@]} == 0 )); then
			ok=1
		fi
	else
		ok=1
		IFS=',' read -r -a expected_kinds <<< "$expected"
		for kind in "${expected_kinds[@]}"; do
			kind_in_list "$kind" "${matched[@]}" || ok=0
		done
		for kind in "${matched[@]}"; do
			kind_in_list "$kind" "${expected_kinds[@]}" || ok=0
		done
	fi
	if (( ok )); then
		pass=$((pass + 1))
	else
		note_failure "$path: expected [$expected], matched [$(IFS=,; printf '%s' "${matched[*]}")]"
	fi
}

# The real rules file must parse before any fixture is checked.
load_rules

line_number=0
while IFS= read -r line || [[ -n "$line" ]]; do
	line_number=$((line_number + 1))
	line=${line%$'\r'}
	[[ -z "$line" || "$line" == \#* ]] && continue
	if [[ "$line" != *'|'* ]]; then
		note_failure "fixture line $line_number: missing separator"
		continue
	fi
	check_fixture "${line%%|*}" "${line#*|}"
done < "$FIXTURES_FILE"

# Malformed rules files must be rejected, not silently ignored.
rejection_tmp=$(mktemp)
trap 'rm -f "$rejection_tmp"' EXIT

expect_rejection() {
	local label=$1 content=$2
	printf '%s\n' "$content" > "$rejection_tmp"
	if ( load_rules "$rejection_tmp" ) >/dev/null 2>&1; then
		note_failure "parser accepted $label"
	else
		pass=$((pass + 1))
	fi
}

expect_rejection "an unknown rule kind" $'OTHER|^go\\.sum$'
expect_rejection "an empty pattern" $'GENERATED|'
expect_rejection "an unanchored pattern" $'GENERATED|go\\.sum'
expect_rejection "a missing separator" $'GENERATED ^go\\.sum$'
expect_rejection "an invalid regex" $'GENERATED|^[unclosed$'

printf '%s: %d passed, %d failed\n' "${0##*/}" "$pass" "$failed"
(( failed == 0 ))
