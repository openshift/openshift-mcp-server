# Upstream synchronization

The downstream repository can periodically integrate `containers/kubernetes-mcp-server`'s `main` branch through a dedicated `upstream-sync` branch and pull request. The workflow is implemented by `hack/sync-upstream.sh`; Make targets in `build/openshift/sync-upstream.mk` provide its entrypoints.

## Prerequisites

- Configure an `upstream` Git remote pointing to `containers/kubernetes-mcp-server`.
- Configure an `origin` remote pointing to the downstream fork.
- Install Git and Bash. Publishing also requires the GitHub CLI (`gh`) authenticated for the downstream repository.
- Start publishing from a clean working tree. `sync-upstream-pr` refuses a dirty tree.

The script preflights mode-specific commands before inspecting remotes. Publish mode also requires Go, Make, `seq`, and `sleep` for generators and branch-index polling.

Defaults are `upstream/main`, `origin/main`, and the `upstream-sync` branch. Synchronization is intentionally limited to the `main` base branch. Make variables can override `UPSTREAM_REPO`, `UPSTREAM_REMOTE`, `ORIGIN_REMOTE`, `SYNC_BRANCH_NAME`, and `OWNER_REPO`.

In a developer clone where `origin` is a personal fork and `downstream` points to the downstream repository, select the downstream remote explicitly:

```sh
make sync-upstream-check ORIGIN_REMOTE=downstream
make sync-upstream-pr ORIGIN_REMOTE=downstream
```

## Workflow at a glance

```text
make sync-upstream-check ──> plan mode ──> fetch/pin refs ──> report plan ──> stop

make sync-upstream-pr ─────> publish mode ──> plan ──> temporary worktree merge
                                                            │
                                              ┌─────────────┴─────────────┐
                                              │                           │
                                         no conflict                  conflict
                                              │                           │
                                              │                 all paths allowlisted?
                                              │                    │ yes       │ no
                                              │                    │           └──> stop
                                              └──────────┬─────────┘               no push/PR
                                                         ▼
                                             run generators and verify
                                                         │
                                                candidate clean?
                                                  │ yes       │ no
                                                  ▼           └──> stop; no PR
                                          lease-push branch
                                                  │
                                                  ▼
                                        verify refs and mergeability
                                                  │
                                     ┌────────────┴─────────────┐
                                     │                          │
                                   pass                       fail
                                     │                          │
                                     ▼                          ▼
                          success status, PR upsert      failure status; no PR
```

## Inspect the plan

```sh
make sync-upstream-check
```

This fetches the configured base refs, pins their commit SHAs, reports upstream commits and overlapping paths, and previews merge conflicts. It does not push a branch or create/update a pull request. As with any Git fetch, it updates local fetch metadata.

Planning considers only the two repositories' merged `main` refs. It does not inspect open downstream pull requests. After a sync PR lands, maintainers of affected downstream PRs should rebase or update those branches against downstream `main` and resolve any resulting conflicts.

## Publish or update the sync pull request

```sh
make sync-upstream-pr
```

Publishing integrates the pinned upstream commit in a temporary Git worktree, runs the repository's generators, and verifies that the candidate includes the pinned upstream SHA and merges cleanly with the pinned downstream SHA. Immediately before pushing, it checks that both remote base refs still match the pinned SHAs. The sync branch is pushed with an explicit `--force-with-lease` expectation so concurrent changes are not silently overwritten. The script then sets the `upstream-sync/mergeable` commit status and creates or updates the pull request.

The regeneration sequence is centralized in `make downstream-autogenerate-content` in `build/openshift/sync-upstream.mk`. It runs module tidy/vendor, README/configuration generation, and test snapshot generation; it does not stage or commit files. The sync script invokes this target from its temporary worktree, so new regeneration steps can be added there without expanding the script's command list.

When a conflict exists, every conflicted path must match a rule in `hack/upstream-sync.allowlist`. Rules use `GENERATED|<glob>`; a single `*` matches within one path segment, while `**` may span directories. The current rules cover vendor files, test snapshots, and the generated sections in README/configuration docs.

The allowlist is parsed as data; it is never sourced or evaluated. The script runs the existing generators in a temporary worktree before staging any conflicted file. This lets the documentation generator replace its marker-bounded sections directly, while the vendor and snapshot generators rewrite their outputs. It then checks that conflict markers are gone and the Git index has no unmerged paths.

```text
unmerged paths ──> every path matches GENERATED allowlist?
                            │
                 ┌──────────┴──────────┐
                 │                     │
                no                    yes
                 │                     │
       discard worktree          run generators
       stop; no push/PR                │
                                      ▼
                            conflict markers remain?
                              │                 │
                             yes                no
                              │                 │
                     discard; no PR       stage generated outputs
                                                │
                                                ▼
                                     verify no unmerged paths
```

If integration cannot produce a clean candidate, publishing stops without pushing a branch or creating/updating a PR. If a post-push pinned-ref or mergeability check fails, the script publishes a failing status and does not create/update a PR. The mergeability status and PR are updated only after the candidate passes the pinned checks; conflict-only draft PRs are not created.

For CI, protect the downstream base branch with the `upstream-sync/mergeable` status check. A failed status blocks the sync PR until a maintainer resolves the conflict or generation problem and reruns the workflow. Keep the repository's normal CI tests required as well: this status verifies upstream coverage and mergeability, not semantic correctness.
