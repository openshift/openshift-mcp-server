#!/usr/bin/env bash

set -Eeuo pipefail

UPSTREAM_SYNC_ROOT=${BASH_SOURCE[0]%/*}
[[ "$UPSTREAM_SYNC_ROOT" != "${BASH_SOURCE[0]}" ]] || UPSTREAM_SYNC_ROOT=.
UPSTREAM_SYNC_ROOT=$(cd -- "$UPSTREAM_SYNC_ROOT" && pwd)
ALLOWLIST_FILE=${UPSTREAM_SYNC_ALLOWLIST:-"$UPSTREAM_SYNC_ROOT/upstream-sync-generated.allowlist"}

# Shared validation, configuration, and process lifecycle helpers.
fail() {
	printf 'upstream-sync: %s\n' "$*" >&2
	exit 1
}

require_command() {
	local tool=$1
	command -v "$tool" >/dev/null 2>&1 || fail "required command not found: $tool"
}

validate_required_tools() {
	local mode=$1 tool
	local -a required=(cat mktemp rm)
	case "$mode" in
		plan) required+=(git) ;;
		publish) required+=(git go gh make seq sleep) ;;
		*) fail "cannot validate tools for unknown mode: $mode" ;;
	esac
	for tool in "${required[@]}"; do require_command "$tool"; done
}

valid_relative_path() {
	local path=$1
	[[ -n "$path" && "$path" != /* && "$path" =~ ^[A-Za-z0-9_./-]+$ ]] || return 1
	case "/$path/" in *"/../"*) return 1 ;; esac
	return 0
}

valid_repo_name() {
	[[ "$1" =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]]
}

valid_remote_name() {
	[[ "$1" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]
}

github_repo_from_url() {
	local url=$1 actual
	case "$url" in
		https://github.com/*) actual=${url#https://github.com/} ;;
		https://*@github.com/*) actual=${url#*github.com/} ;;
		ssh://github.com/*) actual=${url#ssh://github.com/} ;;
		ssh://*@github.com/*) actual=${url#*github.com/} ;;
		git@github.com:*) actual=${url#git@github.com:} ;;
		*) return 1 ;;
	esac
	actual=${actual%.git}
	valid_repo_name "$actual" || return 1
	printf '%s\n' "$actual"
}

initialize_config() {
	local origin_url
	REPO_DIR=${REPO_DIR:-$(git rev-parse --show-toplevel)} || fail "not inside a Git repository"
	UPSTREAM_REPO=${UPSTREAM_REPO:-containers/kubernetes-mcp-server}
	UPSTREAM_REMOTE=${UPSTREAM_REMOTE:-upstream}
	ORIGIN_REMOTE=${ORIGIN_REMOTE:-origin}
	SYNC_BRANCH_NAME=${SYNC_BRANCH_NAME:-upstream-sync}
	SYNC_BASE_BRANCH=${SYNC_BASE_BRANCH:-main}
	OWNER_REPO=${OWNER_REPO:-}
	ALLOWLIST_FILE=${UPSTREAM_SYNC_ALLOWLIST:-${ALLOWLIST_FILE:-$UPSTREAM_SYNC_ROOT/upstream-sync-generated.allowlist}}
	valid_remote_name "$UPSTREAM_REMOTE" || fail "invalid upstream remote name: $UPSTREAM_REMOTE"
	valid_remote_name "$ORIGIN_REMOTE" || fail "invalid downstream remote name: $ORIGIN_REMOTE"
	if [[ -z "$OWNER_REPO" ]]; then
		origin_url=$(git -C "$REPO_DIR" remote get-url "$ORIGIN_REMOTE") || fail "unknown downstream remote: $ORIGIN_REMOTE"
		OWNER_REPO=$(github_repo_from_url "$origin_url") || fail "could not derive a GitHub repository from remote $ORIGIN_REMOTE"
	fi
	valid_repo_name "$UPSTREAM_REPO" || fail "invalid upstream repository name: $UPSTREAM_REPO"
	valid_repo_name "$OWNER_REPO" || fail "invalid downstream repository name: $OWNER_REPO"
	[[ "$SYNC_BASE_BRANCH" == main ]] || fail "upstream synchronization only supports the main base branch"
	[[ "$UPSTREAM_REMOTE" != "$ORIGIN_REMOTE" ]] || fail "upstream and downstream remotes must differ"
	git -C "$REPO_DIR" check-ref-format --branch "$SYNC_BRANCH_NAME" >/dev/null || fail "invalid sync branch name: $SYNC_BRANCH_NAME"
}

validate_remote() {
	local remote=$1 expected=$2 url actual
	url=$(git -C "$REPO_DIR" remote get-url "$remote") || fail "unknown remote: $remote"
	if [[ -n "$expected" ]]; then
		actual=$(github_repo_from_url "$url") || fail "remote $remote does not point to expected GitHub repository $expected"
		[[ "$actual" == "$expected" ]] || fail "remote $remote does not point to expected GitHub repository $expected"
	fi
}

initialize_runtime() {
	TMP_DIR=$(mktemp -d)
	WORKTREE=""
	trap cleanup EXIT
	trap 'exit 1' HUP INT TERM
}

validate_publish_preconditions() {
	[[ -z "$(git -C "$REPO_DIR" status --porcelain)" ]] || fail "refusing to publish from a dirty worktree"
}

cleanup() {
	if [[ -n "${WORKTREE:-}" ]]; then git -C "$REPO_DIR" worktree remove --force "$WORKTREE" >/dev/null 2>&1 || true; fi
	if [[ -n "${TMP_DIR:-}" ]]; then rm -rf "$TMP_DIR"; fi
}

# Plaintext allowlist parsing and generated-conflict resolution.
validate_allowlist() {
	local file=${1:-$ALLOWLIST_FILE}
	local line kind path start end extra line_number=0
	[[ -r "$file" ]] || fail "generated-content allowlist is missing: $file"
	while IFS= read -r line || [[ -n "$line" ]]; do
		line_number=$((line_number + 1))
		line=${line%$'\r'}
		[[ -z "$line" || "$line" == \#* ]] && continue
		kind="" path="" start="" end="" extra=""
		IFS='|' read -r kind path start end extra <<< "$line"
		case "$kind" in
			GENERATED)
				[[ -z "$start$end$extra" ]] || fail "allowlist line $line_number has unexpected fields"
				[[ -n "$path" && "$path" != /* && "$path" =~ ^[A-Za-z0-9_./*-]+$ ]] || fail "invalid generated path pattern on allowlist line $line_number"
				case "/$path/" in *"/../"*) fail "parent traversal in allowlist line $line_number" ;; esac
				;;
			*) fail "unknown allowlist rule '$kind' on line $line_number" ;;
		esac
	done < "$file"
}

file_rule_matches() {
	local path=$1 pattern=$2 regex='^' index=0 char next
	while (( index < ${#pattern} )); do
		char=${pattern:index:1}
		if [[ "$char" == '*' ]]; then
			next=${pattern:index+1:1}
			if [[ "$next" == '*' ]]; then
				regex+='.*'
				index=$((index + 2))
			else
				regex+='[^/]*'
				index=$((index + 1))
			fi
		else
			[[ "$char" == '.' ]] && regex+='\.' || regex+="$char"
			index=$((index + 1))
		fi
	done
	regex+='$'
	[[ "$path" =~ $regex ]]
}

allowlisted_conflict() {
	local path=$1 line kind pattern start end extra
	valid_relative_path "$path" || return 1
	while IFS= read -r line || [[ -n "$line" ]]; do
		line=${line%$'\r'}
		[[ -z "$line" || "$line" == \#* ]] && continue
		kind="" pattern="" start="" end="" extra=""
		IFS='|' read -r kind pattern start end extra <<< "$line"
		if [[ "$kind" == GENERATED ]] && file_rule_matches "$path" "$pattern"; then
			return 0
		fi
	done < "$ALLOWLIST_FILE"
	return 1
}

all_conflicts_allowlisted() {
	local path
	[[ ${#CONFLICT_PATHS[@]} -gt 0 ]] || return 1
	for path in "${CONFLICT_PATHS[@]}"; do
		if ! allowlisted_conflict "$path"; then return 1; fi
	done
	return 0
}

load_unmerged_paths() {
	local path
	CONFLICT_PATHS=()
	git -C "$WORKTREE" diff --name-only --diff-filter=U -z > "$TMP_DIR/unmerged.paths" || return 1
	while IFS= read -r -d '' path; do
		valid_relative_path "$path" || return 1
		CONFLICT_PATHS+=("$path")
	done < "$TMP_DIR/unmerged.paths"
	return 0
}

conflict_markers_remain() {
	local path file line
	for path in "${CONFLICT_PATHS[@]}"; do
		file="$WORKTREE/$path"
		[[ -f "$file" ]] || continue
		[[ -r "$file" ]] || return 0
		while IFS= read -r line || [[ -n "$line" ]]; do
			case "$line" in
				'<<<<<<<'*|'======='*|'>>>>>>>'*) return 0 ;;
			esac
		done < "$file"
	done
	return 1
}

stage_generated() {
	git -C "$WORKTREE" add -- go.mod go.sum vendor/ README.md docs/configuration.md pkg/mcp/testdata/
}

# Git remotes, pinned refs, plan data, and mergeability operations.
fetch_sha() {
	local remote=$1 branch=$2 output_name=$3
	git -C "$REPO_DIR" fetch --no-tags "$remote" "refs/heads/$branch" >/dev/null
	printf -v "$output_name" '%s' "$(git -C "$REPO_DIR" rev-parse FETCH_HEAD)"
}

collect_overlaps() {
	local path other
	UPSTREAM_PATHS=()
	DOWNSTREAM_PATHS=()
	OVERLAPS=()
	git -C "$REPO_DIR" diff --name-only -z "$BASE_SHA" "$UPSTREAM_SHA" > "$TMP_DIR/upstream.paths"
	git -C "$REPO_DIR" diff --name-only -z "$BASE_SHA" "$DOWNSTREAM_SHA" > "$TMP_DIR/downstream.paths"
	while IFS= read -r -d '' path; do UPSTREAM_PATHS+=("$path"); done < "$TMP_DIR/upstream.paths"
	while IFS= read -r -d '' path; do DOWNSTREAM_PATHS+=("$path"); done < "$TMP_DIR/downstream.paths"
	for path in "${UPSTREAM_PATHS[@]}"; do
		for other in "${DOWNSTREAM_PATHS[@]}"; do
			if [[ "$path" == "$other" ]]; then OVERLAPS+=("$path"); break; fi
		done
	done
}

preview_merge() {
	local left=$1 right=$2 output=$3 result_name=$4 status
	if git -C "$REPO_DIR" merge-tree --write-tree "$left" "$right" > "$output" 2>&1; then
		printf -v "$result_name" '%s' 0
	else
		status=$?
		if [[ $status -eq 1 ]]; then
			printf -v "$result_name" '%s' 1
		else
			fail "merge-tree failed (exit $status): $(<"$output")"
		fi
	fi
}

remote_branch_sha() {
	local refs sha fetched
	refs=$(git -C "$REPO_DIR" ls-remote --heads "$ORIGIN_REMOTE" "refs/heads/$SYNC_BRANCH_NAME") || fail "could not inspect remote sync branch"
	sha=${refs%%$'\t'*}
	[[ "$sha" == "$refs" ]] && sha=""
	if [[ -n "$sha" ]]; then
		git -C "$REPO_DIR" fetch --no-tags "$ORIGIN_REMOTE" "refs/heads/$SYNC_BRANCH_NAME" >/dev/null || fail "could not fetch existing sync branch"
		fetched=$(git -C "$REPO_DIR" rev-parse FETCH_HEAD) || fail "could not resolve fetched sync branch"
		[[ "$fetched" == "$sha" ]] || fail "remote sync branch moved during inspection; retry"
	fi
	printf '%s' "$sha"
}

print_plan() {
	printf 'Upstream:   %s/%s (%s)\n' "$UPSTREAM_REMOTE" "$SYNC_BASE_BRANCH" "$UPSTREAM_SHA"
	printf 'Downstream: %s/%s (%s)\n' "$ORIGIN_REMOTE" "$SYNC_BASE_BRANCH" "$DOWNSTREAM_SHA"
	printf 'Behind:     %s commit(s)\n' "$BEHIND_COUNT"
	if [[ ${#OVERLAPS[@]} -gt 0 ]]; then
		printf 'Overlapping paths (review semantics):\n'
		printf '  %q\n' "${OVERLAPS[@]}"
	fi
	if [[ "$TARGET_CONFLICT" == 1 ]]; then
		printf 'Merge conflict detected between downstream and upstream.\n'
		cat "$TMP_DIR/target-merge.out"
	fi
	if [[ "$BRANCH_CONFLICT" == 1 ]]; then
		printf 'Merge conflict detected updating existing sync branch.\n'
		cat "$TMP_DIR/branch-merge.out"
	fi
}

verify_pinned_refs() {
	pinned_refs_match || fail "upstream or downstream main moved since plan; rerun sync"
}

pinned_refs_match() {
	local upstream_refs downstream_refs upstream_now downstream_now
	upstream_refs=$(git -C "$REPO_DIR" ls-remote --heads "$UPSTREAM_REMOTE" "refs/heads/$SYNC_BASE_BRANCH") || return 1
	downstream_refs=$(git -C "$REPO_DIR" ls-remote --heads "$ORIGIN_REMOTE" "refs/heads/$SYNC_BASE_BRANCH") || return 1
	upstream_now=${upstream_refs%%$'\t'*}
	downstream_now=${downstream_refs%%$'\t'*}
	[[ "$upstream_now" == "$UPSTREAM_SHA" && "$downstream_now" == "$DOWNSTREAM_SHA" ]]
}

remote_ref_sha() {
	local remote=$1 ref=$2 refs
	refs=$(git -C "$REPO_DIR" ls-remote --heads "$remote" "$ref") || fail "could not inspect $remote $ref"
	printf '%s\n' "${refs%%$'\t'*}"
}

push_branch() {
	local head_sha=$1 expected_sha=$2
	git -C "$REPO_DIR" push --force-with-lease="refs/heads/$SYNC_BRANCH_NAME:$expected_sha" \
		"$ORIGIN_REMOTE" "$head_sha:refs/heads/$SYNC_BRANCH_NAME" || fail "lease-protected push failed; remote branch was not overwritten"
}

wait_for_branch() {
	local expected=$1 attempt actual
	for attempt in $(seq 1 30); do
		actual=$(remote_ref_sha "$ORIGIN_REMOTE" "refs/heads/$SYNC_BRANCH_NAME")
		[[ "$actual" == "$expected" ]] && return 0
		sleep 5
	done
	fail "remote branch was not indexed at $expected"
}

mergeable_head() {
	local head=$1
	git -C "$REPO_DIR" merge-base --is-ancestor "$UPSTREAM_SHA" "$head" || return 1
	if git -C "$REPO_DIR" merge-tree --write-tree "$DOWNSTREAM_SHA" "$head" > "$TMP_DIR/mergeability.out" 2>&1; then return 0; fi
	return 1
}

# Temporary-worktree merge, generation, and candidate verification.
merge_in_worktree() {
	local base_sha=$1 preview_conflicts candidate has_conflicts=0 status
	WORKTREE="$TMP_DIR/worktree"
	if ! git -C "$REPO_DIR" worktree add --detach "$WORKTREE" "$base_sha" >/dev/null; then
		INTEGRATION_NOTE="Could not create isolated worktree."
		WORKTREE=""
		return 1
	fi
	preview_merge "$base_sha" "$UPSTREAM_SHA" "$TMP_DIR/worktree-merge.out" preview_conflicts
	if git -C "$WORKTREE" -c merge.conflictStyle=merge merge --no-ff "$UPSTREAM_SHA" -m "chore: merge upstream changes" > "$TMP_DIR/actual-merge.out" 2>&1; then
		if [[ "$preview_conflicts" == 1 ]]; then
			INTEGRATION_NOTE="Merge preflight and actual merge disagreed; integration stopped."
			git -C "$WORKTREE" reset --merge >/dev/null 2>&1 || true
			remove_worktree
			return 2
		fi
	else
		if [[ "$preview_conflicts" != 1 ]]; then
			INTEGRATION_NOTE="Actual merge conflicted although preflight was clean."
			git -C "$WORKTREE" merge --abort >/dev/null 2>&1 || true
			remove_worktree
			return 2
		fi
		if ! load_unmerged_paths; then
			INTEGRATION_NOTE="Could not enumerate unmerged paths."
			git -C "$WORKTREE" merge --abort >/dev/null 2>&1 || true
			remove_worktree
			return 2
		fi
		if ! all_conflicts_allowlisted; then
			INTEGRATION_NOTE="Merge conflicts include paths without regeneration rules."
			git -C "$WORKTREE" merge --abort >/dev/null 2>&1 || true
			remove_worktree
			return 2
		fi
		has_conflicts=1
	fi
	# Let the generators replace conflicted outputs before staging any resolution.
	if ! run_generators; then
		INTEGRATION_NOTE="A generator failed; the isolated candidate was discarded."
		remove_worktree
		return 2
	fi
	if [[ "$has_conflicts" == 1 ]]; then
		# A conflict outside generated regions must remain visible and block publishing.
		if conflict_markers_remain; then
			INTEGRATION_NOTE="Regeneration left conflict markers in an allowlisted file."
			git -C "$WORKTREE" merge --abort >/dev/null 2>&1 || true
			remove_worktree
			return 2
		fi
		if ! stage_generated; then
			INTEGRATION_NOTE="Could not stage regenerated conflict outputs."
			git -C "$WORKTREE" merge --abort >/dev/null 2>&1 || true
			remove_worktree
			return 2
		fi
		if ! git -C "$WORKTREE" diff --name-only --diff-filter=U -z > "$TMP_DIR/remaining-unmerged" || [[ -s "$TMP_DIR/remaining-unmerged" ]]; then
			INTEGRATION_NOTE="Unmerged paths remain after regeneration."
			git -C "$WORKTREE" merge --abort >/dev/null 2>&1 || true
			remove_worktree
			return 2
		fi
		if ! git -C "$WORKTREE" -c core.editor=true commit --no-edit > "$TMP_DIR/conflict-commit.out" 2>&1; then
			INTEGRATION_NOTE="Could not commit the regenerated conflict resolution."
			remove_worktree
			return 2
		fi
		status=$(git -C "$WORKTREE" status --porcelain) || { INTEGRATION_NOTE="Could not inspect the regenerated worktree."; remove_worktree; return 2; }
		if [[ -n "$status" ]]; then
			INTEGRATION_NOTE="Generator changed paths outside staged generated outputs."
			remove_worktree
			return 2
		fi
	elif ! commit_generated; then
		INTEGRATION_NOTE="Generated outputs could not be committed in the temporary worktree."
		remove_worktree
		return 2
	fi
	if ! candidate=$(git -C "$WORKTREE" rev-parse HEAD); then
		INTEGRATION_NOTE="Could not resolve candidate commit."
		remove_worktree
		return 2
	fi
	if ! git -C "$WORKTREE" merge-base --is-ancestor "$UPSTREAM_SHA" "$candidate"; then
		INTEGRATION_NOTE="Candidate does not contain the pinned upstream commit."
		remove_worktree
		return 2
	fi
	if ! git -C "$WORKTREE" merge-tree --write-tree "$DOWNSTREAM_SHA" "$candidate" > "$TMP_DIR/candidate-merge.out" 2>&1; then
		INTEGRATION_NOTE="Candidate still conflicts with downstream after regeneration."
		remove_worktree
		return 2
	fi
	CANDIDATE_SHA=$candidate
	remove_worktree
	return 0
}

remove_worktree() {
	if [[ -n "${WORKTREE:-}" ]]; then
		git -C "$REPO_DIR" worktree remove --force "$WORKTREE" >/dev/null 2>&1 || true
		WORKTREE=""
	fi
}

run_generators() {
	(
		cd "$WORKTREE"
		go mod tidy
		go mod vendor
		make update-readme-tools
		make test-update-snapshots
	)
}

commit_generated() {
	local status
	status=$(git -C "$WORKTREE" status --porcelain) || return 1
	[[ -n "$status" ]] || return 0
	stage_generated || return 1
	if git -C "$WORKTREE" diff --cached --quiet; then return 1; fi
	git -C "$WORKTREE" commit -m "chore: regenerate files for upstream sync" || return 1
	status=$(git -C "$WORKTREE" status --porcelain) || return 1
	[[ -z "$status" ]]
}

# GitHub commit-status and pull-request publishing operations.
set_merge_status() {
	local head=$1 state_code=$2 description=$3 state
	state=failure
	[[ "$state_code" == 0 ]] && state=success
	gh api --method POST "repos/$OWNER_REPO/statuses/$head" \
		--field "state=$state" \
		--field "context=upstream-sync/mergeable" \
		--field "description=$description" >/dev/null || fail "could not publish mergeability status"
}

upsert_pr() {
	local head=$1 body=$2 number is_draft
	number=$(gh pr list --repo "$OWNER_REPO" --head "$SYNC_BRANCH_NAME" --base "$SYNC_BASE_BRANCH" --state open --json number --jq '.[0].number') || fail "could not inspect existing sync PR"
	if [[ -n "$number" && "$number" != null ]]; then
		is_draft=$(gh pr view "$number" --repo "$OWNER_REPO" --json isDraft --jq '.isDraft') || fail "could not inspect sync PR draft state"
		[[ "$is_draft" == true || "$is_draft" == false ]] || fail "unexpected sync PR draft state: $is_draft"
		gh api --method PATCH "repos/$OWNER_REPO/pulls/$number" --raw-field "body=$body" >/dev/null || fail "could not update sync PR body"
		if [[ "$is_draft" == true ]]; then
			gh pr ready "$number" --repo "$OWNER_REPO" || fail "could not mark sync PR ready"
		fi
		return 0
	fi
	gh api --method POST "repos/$OWNER_REPO/pulls" \
		--raw-field "title=NO-JIRA: sync with upstream ${UPSTREAM_SHA:0:12}" \
		--raw-field "head=$SYNC_BRANCH_NAME" \
		--raw-field "base=$SYNC_BASE_BRANCH" \
		--raw-field "body=$body" >/dev/null || fail "could not create sync PR"
}

clean_pr_body() {
	local head=$1
	{
		printf '### 🔄 Upstream Sync\n\nThis PR syncs the downstream fork with upstream.\n\n'
		printf -- '- Upstream SHA: `%s`\n- Downstream SHA: `%s`\n- Published head: `%s`\n- Mergeable: `true`\n\n' "$UPSTREAM_SHA" "$DOWNSTREAM_SHA" "$head"
		printf '**Changes:**\n'
		git -C "$REPO_DIR" log --pretty=format:'- %h %s (%an)' "$DOWNSTREAM_SHA..$UPSTREAM_SHA"
		printf '\n'
	}
}

# Command orchestration and clean/conflict publication workflows.
usage() {
	cat <<'USAGE'
Usage: hack/sync-upstream.sh [plan|publish]

  plan     Fetch and describe the pinned sync plan without publishing (default).
  publish  Integrate, regenerate, push with a lease, and upsert the sync PR.

The sibling upstream-sync-generated.allowlist is parsed as data; it is never sourced.
USAGE
}

build_sync_plan() {
	validate_allowlist
	if [[ "$MODE" == publish ]]; then validate_publish_preconditions; fi
	validate_remote "$UPSTREAM_REMOTE" "$UPSTREAM_REPO"
	validate_remote "$ORIGIN_REMOTE" "$OWNER_REPO"
	fetch_sha "$UPSTREAM_REMOTE" "$SYNC_BASE_BRANCH" UPSTREAM_SHA
	fetch_sha "$ORIGIN_REMOTE" "$SYNC_BASE_BRANCH" DOWNSTREAM_SHA
	BASE_SHA=$(git -C "$REPO_DIR" merge-base "$DOWNSTREAM_SHA" "$UPSTREAM_SHA") || fail "upstream and downstream have no merge base"
	BEHIND_COUNT=$(git -C "$REPO_DIR" rev-list --count "$DOWNSTREAM_SHA..$UPSTREAM_SHA")
	if [[ "$BEHIND_COUNT" == 0 ]]; then return 0; fi
	collect_overlaps
	preview_merge "$DOWNSTREAM_SHA" "$UPSTREAM_SHA" "$TMP_DIR/target-merge.out" TARGET_CONFLICT
	SYNC_REMOTE_SHA=$(remote_branch_sha)
	BRANCH_CONFLICT=0
	if [[ -n "$SYNC_REMOTE_SHA" ]]; then
		preview_merge "$SYNC_REMOTE_SHA" "$UPSTREAM_SHA" "$TMP_DIR/branch-merge.out" BRANCH_CONFLICT
	fi
}

main() {
	MODE=${1:-plan}
	case "$MODE" in plan|publish) ;; *) usage; fail "unknown mode: $MODE" ;; esac
	validate_required_tools "$MODE"
	initialize_config
	initialize_runtime
	build_sync_plan
	if [[ "$BEHIND_COUNT" == 0 ]]; then
		printf 'Up to date: %s and %s\n' "$UPSTREAM_SHA" "$DOWNSTREAM_SHA"
		return 0
	fi
	print_plan
	[[ "$MODE" == publish ]] || return 0
	publish_sync
}

publish_sync() {
	local base_sha="$DOWNSTREAM_SHA" head_sha body
	if [[ -n "$SYNC_REMOTE_SHA" ]]; then base_sha="$SYNC_REMOTE_SHA"; fi
	INTEGRATION_NOTE=""
	if ! merge_in_worktree "$base_sha"; then
		fail "could not produce a conflict-free candidate; no branch or PR was published: ${INTEGRATION_NOTE:-see command error output}"
	fi
	head_sha="$CANDIDATE_SHA"
	verify_pinned_refs
	push_branch "$head_sha" "$SYNC_REMOTE_SHA"
	wait_for_branch "$head_sha"
	if ! pinned_refs_match; then
		set_merge_status "$head_sha" 1 "source refs moved after planning; rerun sync before creating a PR"
		fail "source refs moved after pushing the clean candidate; no PR was created"
	fi
	if ! mergeable_head "$head_sha"; then
		set_merge_status "$head_sha" 1 "candidate is not mergeable with the pinned downstream ref"
		fail "published candidate failed mergeability verification; no PR was created"
	fi
	body=$(clean_pr_body "$head_sha")
	set_merge_status "$head_sha" 0 "candidate contains pinned upstream and merges cleanly with downstream"
	upsert_pr "$head_sha" "$body"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then main "$@"; fi
