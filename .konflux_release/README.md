# Konflux Release Artifacts

This directory contains developer-side Konflux release templates for the existing
`mcp-server-02` application in the `ocp-mcp-server-tenant` workspace.

## Layout

- `releasePlan/` contains developer-side `ReleasePlan` objects.
- `releasePlanAdmission/` contains managed-side `ReleasePlanAdmission` examples.
- `releases/` contains `Release` objects.
- `snapshot/` contains manual `Snapshot` templates.

## How to use these manifests

1. Update placeholders in the YAML files:
   - `TODO-managed-tenant-namespace`
   - `TODO-your-userid`
   - the image digest and git revision in the manual snapshot
2. Apply the release plan:

   ```bash
   kubectl apply -f .konflux_release/releasePlan/openshift-mcp-server-release-020.yaml
   ```

3. Ask the managed environment owners to create or update the matching
   `ReleasePlanAdmission`:

   ```bash
   kubectl apply -f .konflux_release/releasePlanAdmission/openshift-mcp-server-release-020-managed.yaml
   ```

4. Either:
   - use a push-generated Snapshot created by Konflux, then update
     `.konflux_release/releases/openshift-mcp-server-release-020-manual.yaml`
     to point at it, or
   - create the manual Snapshot from `snapshot/openshift-mcp-server-release-020-manual.yaml`

5. Create the release:

   ```bash
   kubectl create -f .konflux_release/releases/openshift-mcp-server-release-020-manual.yaml
   ```

## Notes

- The manual snapshot must reference an image by digest, not by tag.
- A matching `ReleasePlanAdmission` must exist in the managed namespace before
  the `Release` can proceed.
- `releasePlan/openshift-mcp-server-release-020.yaml` defaults to manual releases by setting
  `release.appstudio.openshift.io/auto-release: "false"`. If you want automatic
  releases after tests pass, change that value to a Konflux-supported CEL
  expression such as `"true"`.
