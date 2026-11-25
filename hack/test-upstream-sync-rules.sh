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

gomod_case_dir=$(mktemp -d)
trap 'rm -f "$rejection_tmp"; rm -rf "$gomod_case_dir"' EXIT

write_gomod() {
	local path=$1
	shift
	{
		printf 'module example.com/mod\n\ngo 1.22.0\n\nrequire (\n'
		printf '\t%s\n' "$@"
		printf ')\n'
	} > "$path"
}

run_syncgomod() {
	go -C "$TEST_ROOT/.." run -mod=readonly ./hack/syncgomod.go "$gomod_case_dir/down" "$gomod_case_dir/up" "$gomod_case_dir/out" 2>"$gomod_case_dir/err"
}

write_gomod "$gomod_case_dir/down" "example.com/common v1.2.0" "example.com/down-only v1.0.0"
write_gomod "$gomod_case_dir/up" "example.com/common v1.3.0" "example.com/up-only v0.4.0 // indirect"
if run_syncgomod \
	&& grep -q 'example.com/common v1.3.0' "$gomod_case_dir/out" \
	&& grep -q 'example.com/down-only v1.0.0' "$gomod_case_dir/out" \
	&& ! grep -q 'up-only' "$gomod_case_dir/out"; then
	pass=$((pass + 1))
else
	note_failure "resolve failed: $(cat "$gomod_case_dir/out" "$gomod_case_dir/err" 2>/dev/null || true)"
fi

write_gomod "$gomod_case_dir/down" "example.com/common v1.5.0"
write_gomod "$gomod_case_dir/up" "example.com/common v1.4.0"
if run_syncgomod; then
	note_failure "downstream-newer module should stop the sync"
else
	case "$(cat "$gomod_case_dir/err")" in
		*"example.com/common"*) pass=$((pass + 1)) ;;
		*) note_failure "stop note was: $(cat "$gomod_case_dir/err" 2>/dev/null || true)" ;;
	esac
fi

if command -v go >/dev/null 2>&1; then
	repo=$(mktemp -d)
	git -C "$repo" init -q -b main
	write_gomod "$repo/go.mod" "example.com/common v1.0.0"
	git -C "$repo" add go.mod
	git -C "$repo" -c user.email=t@example.com -c user.name=t -c commit.gpgsign=false commit -qm base
	write_gomod "$repo/go.mod" "example.com/common v1.2.0" "example.com/down-only v1.0.0"
	git -C "$repo" add go.mod
	git -C "$repo" -c user.email=t@example.com -c user.name=t -c commit.gpgsign=false commit -qm down
	git -C "$repo" checkout -q -b upstream HEAD^
	write_gomod "$repo/go.mod" "example.com/common v1.3.0" "example.com/up-only v0.4.0"
	git -C "$repo" add go.mod
	git -C "$repo" -c user.email=t@example.com -c user.name=t -c commit.gpgsign=false commit -qm up
	git -C "$repo" checkout -q main
	git -C "$repo" merge -q upstream >/dev/null 2>&1 || true
	WORKTREE=$repo
	TMP_DIR=$(mktemp -d)
	if resolve_conflicted_go_mod && ! grep -q 'up-only' "$repo/go.mod" \
		&& grep -q 'example.com/common v1.3.0' "$repo/go.mod" \
		&& grep -q 'example.com/down-only v1.0.0' "$repo/go.mod"; then
		pass=$((pass + 1))
	else
		note_failure "resolve_conflicted_go_mod produced: $(cat "$repo/go.mod" 2>/dev/null || true) ${INTEGRATION_NOTE:-}"
	fi
	rm -rf "$repo" "$TMP_DIR"
fi

printf '%s: %d passed, %d failed\n' "${0##*/}" "$pass" "$failed"
(( failed == 0 ))
