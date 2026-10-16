# Upstream synchronization

`hack/sync-upstream.sh` merges upstream `main` into downstream `main` through an `upstream-sync` branch and PR. The Make targets are defined in `build/openshift/sync-upstream.mk`.

## Setup

- Configure `origin` to point to the downstream repository. The script adds `upstream` if missing and checks existing remote URLs against the configured repository names.
- Install Bash 4.4 or newer, Git with `merge-tree --write-tree` support, and Make. The script rejects older Bash versions before doing any work. Publishing also requires Go and an authenticated GitHub CLI (`gh`) with permission to push and manage PRs in the downstream repository.
- Publish from a clean working tree.

On macOS, put a newer Bash on `PATH`; the system `/bin/bash` is too old. The Make targets invoke `bash` from `PATH`.

The defaults are `upstream/main`, `origin/main`, and the `upstream-sync` branch. Only the `main` base branch is supported. Override `UPSTREAM_REPO`, `UPSTREAM_REMOTE`, `ORIGIN_REMOTE`, `SYNC_BRANCH_NAME`, or `OWNER_REPO` through Make variables. `OWNER_REPO` defaults to the repository identified by `ORIGIN_REMOTE`.

If `origin` is your personal fork and `downstream` is the OpenShift repository, specify the destination:

```sh
make sync-upstream-check ORIGIN_REMOTE=downstream
make sync-upstream-pr ORIGIN_REMOTE=downstream
```

## Preview

```sh
make sync-upstream-check
```

Fetch the base branches, record their commit SHAs, and report how many upstream commits downstream lacks. If there are incoming commits, show overlapping paths and merge-conflict previews. This does not run generators, push branches, or create/update PRs. It updates local fetch metadata and adds the upstream remote if missing.

## Publish

```sh
make sync-upstream-pr
```

Each attempt builds from the recorded downstream `main` commit in a temporary worktree. It merges upstream, runs generators, and verifies that the result contains the upstream commit and merges cleanly with downstream. The caller's checkout and index are left unchanged.

```text
Fetch and record base SHAs
          │
          ├── No incoming commits ──> stop
          ▼
Merge upstream into temporary downstream worktree
          │
          ├── Unlisted conflict ────> stop; no push or PR update
          ▼
Regenerate outputs and verify candidate
          │
          ├── Failure ─────────────> stop; no push or PR update
          ▼
Recheck base refs; push with lease
          ▼
Wait for remote branch; recheck refs and mergeability
          │
          ├── Failure ─────────────> stop; no PR update
          ▼
Create or update sync PR
```

The push uses `--force-with-lease` against the inspected sync-branch SHA to reject concurrent branch changes. It replaces the previous integration; changes made only on the old sync branch are not retained.

If checks fail after the push, the branch may already be updated, but the script does not create or update the PR body. An existing PR on that branch will still reflect the pushed commits.

PR lookup is limited to the downstream repository's own sync branch targeting `main`; same-named branches in personal forks are excluded.

## Generated conflicts

Every conflicted path must match `hack/upstream-sync.allowlist`. Entries use `GENERATED|<regex>`, with Bash extended regular expressions anchored by `^` and `$`. Rules match the complete repository-relative path; absolute paths and parent traversal are rejected. The file is parsed once as data, never executed. Invalid or unanchored rules stop the sync.

For example, `GENERATED|^vendor/.*$` includes nested vendor files, while `GENERATED|^pkg/mcp/testdata/[^/]*\.json$` matches JSON files directly in that directory. Escape literal dots with `\.`.

The current rules cover `go.sum`, vendor files, snapshot JSON files, `README.md`, and `docs/configuration.md`. Conflicts in `go.mod` or other unlisted paths stop the sync.

`make downstream-autogenerate-content` runs module tidy/vendor, documentation generation, and snapshot generation. Add regeneration steps to that target. It does not stage or commit files; the sync script does that in the temporary worktree.

For allowlisted conflicts, generators run before the files are staged. The script rejects remaining conflict markers or unmerged index entries. README and configuration generators replace only their marked sections, so handwritten conflicts outside those sections remain unresolved. An allowlist match permits regeneration; it does not guarantee resolution.

Keep normal required CI checks on sync PRs. A clean merge does not establish that the changes work correctly. The planner does not inspect other open downstream PRs; their authors may need to update their branches after a sync lands.
