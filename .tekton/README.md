# .tekton/ Multi-Branch PipelineRun Setup

## Overview

This directory contains Pipelines-as-Code (PaC) configs for **all** branches (main, release-1.0, release-1.1, release-1.2), not just the current branch. This is intentional.

## Why all branches live together

Having identical `.tekton/` contents across every branch enables **clean fast-forward merges** between branches. If each branch carried only its own PipelineRun files, merging would produce conflicts or require cherry-picks.

## How PaC selects the right PipelineRun

Each PipelineRun file contains a CEL expression that gates it to a specific branch and event:

```yaml
pipelinesascode.tekton.dev/on-cel-expression: event == "push" && target_branch == "release-1.0"
```

When a push or pull request arrives, PaC evaluates the CEL expression on **every** PipelineRun file in `.tekton/`. Only the one whose expression matches the target branch and event type is triggered -- the rest are silently skipped.

## File layout

| File | Purpose |
|------|---------|
| `build-pipeline.yaml` | Shared Pipeline definition used by all PipelineRuns |
| `openshift-mcp-server-<branch>-push.yaml` | PipelineRun triggered on push to `<branch>` |
| `openshift-mcp-server-<branch>-pull-request.yaml` | PipelineRun triggered on PRs targeting `<branch>` |

## Branch-specific values in PipelineRun files

Each pair of PipelineRun files (push + pull-request) carries branch-specific values:

- **CEL expression** (`on-cel-expression`): gates to the target branch
- **Application label** (`appstudio.openshift.io/application`): e.g. `mcp-server-main`, `mcp-server-10`
- **Component label** (`appstudio.openshift.io/component`): e.g. `openshift-mcp-server-main`, `openshift-mcp-server-release-10`
- **Service account** (`serviceAccountName`): e.g. `build-pipeline-openshift-mcp-server-main`
- **Output image** (`output-image`): branch-specific image tag path

## Adding a new release branch

1. Create two new PipelineRun files (push + pull-request) with the branch-specific component, application, service account, and output-image values.
2. **Commit them to ALL branches** (main and every release branch) to keep fast-forward merge compatibility.

## MintMaker updates

MintMaker only updates `build-pipeline.yaml` (task bundle references). It does not modify the PipelineRun files.
