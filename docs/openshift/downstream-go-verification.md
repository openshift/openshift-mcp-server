# Downstream Go Version Verification

When a PR in the upstream repository changes the `go` directive in `go.mod`, the
[Go Version Check](./../.github/workflows/check-go-version.yaml) workflow will
fail until the `downstream-go-verified` label is applied.

Before adding that label, verify that the required Go version is available in the
downstream builder image used by `openshift/openshift-mcp-server`.

## Prerequisites

- `oc` CLI installed
- `podman` installed
- A Red Hat SSO account with access to the OpenShift CI cluster

## Steps

### 1. Log in to the OpenShift CI cluster

Open the OpenShift CI console in your browser:

```
https://console-openshift-console.apps.ci.l2s4.p1.openshiftapps.com/k8s/cluster/projects
```

Click the **Red Hat SSO** button to authenticate.

### 2. Copy the login command

In the console UI, click your username in the top-right corner and select
**Copy login command**. Authenticate again via the Red Hat SSO link, then copy
the `oc login` command and run it in your terminal:

```bash
oc login --token=<token> --server=https://api.ci.l2s4.p1.openshiftapps.com:6443
```

### 3. Log in to the container registry

```bash
oc registry login
```

This configures `podman` (and `docker`) to authenticate against
`registry.ci.openshift.org`.

### 4. Verify the Go version in the builder image

Run the builder image with the expected Go version tag. Use `--pull=always` to
ensure you are testing the latest image and not a stale local cache:

```bash
podman run --pull=always registry.ci.openshift.org/ocp/builder:rhel-9-golang-1.26-openshift-5.0 go version
```

Replace the tag (`rhel-9-golang-1.26-openshift-5.0`) with the version matching
the new `go` directive in the PR.

The output should confirm the expected Go version, for example:

```
go version go1.26.0 linux/amd64
```

### 5. Update the CI builder image tag

If the Go version bump requires a new builder image, update the tag in
`.ci-operator.yaml` at the root of the downstream repository:

```yaml
build_root_image:
  name: builder
  namespace: ocp
  tag: rhel-9-golang-1.26-openshift-5.0
```

Change the `tag` value to match the verified image tag from the previous step.

### 6. Apply the label

Once verified, add the `downstream-go-verified` label to the upstream PR. The
Go Version Check workflow will re-evaluate and pass.
