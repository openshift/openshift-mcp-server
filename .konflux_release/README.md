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

## Layout

- `releasePlan/`: developer-side `ReleasePlan` manifests
- `snapshot/`: `Snapshot` manifests that pin the exact image digest and source
  revision to release
- `releases/`: `Release` manifests that request promotion of a snapshot through
  a release plan

The exact filenames may change over time as new release requests are created.

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

1. Update or create a `Snapshot` with the image digest and source revision you
   want to promote.
2. Apply or update the developer-side `ReleasePlan` if needed.
3. Create a new `Release` that references the chosen `Snapshot` and
   `ReleasePlan`.

```bash
kubectl apply -f .konflux_release/releasePlan/<release-plan>.yaml
kubectl apply -f .konflux_release/snapshot/<snapshot>.yaml
kubectl create -f .konflux_release/releases/<release>.yaml
```

## Notes

- Snapshots should reference container images by digest, not by tag.
- `ReleasePlanAdmission` is owned in the Konflux config repo, not in this
  application repo.
- Create a new manifest under `releases/` for each manual release request
  instead of reusing an old `Release` object.
