# Konflux Release Artifacts

This directory contains the Konflux release artifacts used to promote builds of
this repository.

## What these resources mean

- `Snapshot`: a point-in-time capture of the component image, source revision,
  and related build inputs that should be released together.
- `ReleasePlan`: the developer-side configuration that declares where the
  application can be promoted and which managed-side admission should process
  it.
- `Release`: the concrete release request that promotes one snapshot through one
  release plan.
- `ReleasePlanAdmission`: the managed-side policy and pipeline configuration.
  It is managed outside this repository in the Konflux configuration repo, so
  it is not stored under `.konflux_release/`.

## Layout and authority

- `<version>/releasePlan/`: developer-side `ReleasePlan` manifests for a
  supported release stream. For example, `1.2/releasePlan/` contains the stage
  and production plans for the 1.2 application.
- `snapshot/`: `Snapshot` manifests that pin the exact image digest and source
  revision to release.
- `releases/`: `Release` manifests that request promotion of a snapshot through
  a release plan.

`main` is the authoritative source for these shared definitions. The release
procedure must explicitly read the selected versioned `ReleasePlan` from the
`main` revision; a copy that happens to exist in a release branch is not applied
automatically and must not become an independent per-branch configuration
source. The Pipeline-as-Code build files are separate from these manifests and
continue to use their event and target-branch filters.

## High-Level Flow

1. Build pipelines produce an image that should be promoted.
2. A `Snapshot` captures the exact image digest, component name, and source
   revision for that build.
3. A `ReleasePlan` defines the developer-side promotion intent.
4. A `Release` references both the `Snapshot` and the `ReleasePlan` to request
   promotion.
5. Konflux matches that `ReleasePlan` with the corresponding
   `ReleasePlanAdmission` from the Konflux config repo and executes the managed
   release process.

## Typical usage

1. Choose the version directory that matches the snapshot's application and
   component, then choose its `stage` or `prod` `ReleasePlan` as appropriate.
2. Update or create a `Snapshot` with the image digest and source revision you
   want to promote.
3. Apply the selected developer-side `ReleasePlan` from `main`.
4. Create a new `Release` that references that `Snapshot` and `ReleasePlan`.

```bash
git show main:.konflux_release/<version>/releasePlan/<release-plan>.yaml | kubectl apply -f -
kubectl apply -f .konflux_release/snapshot/<snapshot>.yaml
kubectl create -f .konflux_release/releases/<release>.yaml
```

## Notes

- Snapshots should reference container images by digest, not by tag.
- `ReleasePlanAdmission` is owned in the Konflux config repo, not in this
  application repo.
- Create a new manifest under `releases/` for each manual release request
  instead of reusing an old `Release` object.
- No helper in this repository automatically applies a `ReleasePlan`, creates a
  `Release`, or promotes a snapshot. Those remain deliberate release-procedure
  actions.
