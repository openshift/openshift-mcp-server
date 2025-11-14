---
description: Sync downstream repo with upstream main, preserving downstream-specific files
---

Sync downstream repository with upstream changes while preserving downstream-specific files.

# Important Instructions

- **ALWAYS** start from a detached HEAD: `git checkout --detach`
- **DO NOT** use TodoWrite tool
- **DO NOT** ask for confirmations - just execute
- **BE CONCISE** - minimal output, only show final summary

# Execution Steps

Execute these commands in sequence:

```bash
# 1. Start from detached HEAD (REQUIRED)
git checkout --detach

# 2. Delete old sync branch if it exists
git branch -D sync-downstream 2>/dev/null || true

# 3. Fetch and create sync branch
git fetch downstream
git fetch upstream
git checkout -b sync-downstream upstream/main
git merge -s ours --no-commit downstream/main

# 4. Restore downstream files (read from .downstream-preserve or use defaults)
git checkout downstream/main -- .ci-operator.yaml .snyk .tekton/ Dockerfile.ci Dockerfile.ocp Makefile-ocp.mk OWNERS mcp_config.toml renovate.json

# 5. Update dependencies
go mod tidy
go mod vendor
git add vendor/

# 6. Commit everything in ONE commit
git commit -am "sync: merge upstream main with downstream config"
```

# Output

After completion, show ONLY:
- Commit hash
- File change summary: `git diff --stat downstream/main..HEAD | tail -1`
- Next steps:
  ```
  git push origin sync-downstream
  gh pr create --base downstream/main
  ```

# Error Handling

If any command fails, stop immediately and show the error. No rollback needed - user can investigate.
