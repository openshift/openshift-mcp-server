# Korrel8r Toolset (`observability/korrel8r`)

This toolset correlates Kubernetes resources with logs, metrics, alerts and traces using
[Korrel8r](https://korrel8r.github.io/korrel8r/).

Tools, schemas and handlers come from the upstream [`korrel8r/korrel8r/pkg/mcp`](https://github.com/korrel8r/korrel8r)
module; this toolset only adapts them, prefixing tool names with `korrel8r_`.

For configuration options see the [configuration reference](../configuration.md#korrel8r-configuration).

## Two independent use cases

**Correlation search (no console).** Most of the toolset is a correlation engine.
Given a starting point — pod, deployment, alert — `korrel8r_create_neighbors_graph`
and `korrel8r_create_goals_graph` follow correlation rules across domains and return a graph of
related classes with queries and result counts. `korrel8r_get_objects` retrieves the matching
objects. This needs nothing but a reachable Korrel8r endpoint: no OpenShift console, browser,
or UIPlugin. It works from any MCP client, including a terminal agent.

```
list_domains -> help -> create_neighbors_graph / create_goals_graph -> get_objects
```

**Console navigation (optional).** The tools `korrel8r_get_console` and `korrel8r_show_in_console`
require an OpenShift console. They let an agent see what the user is looking at and draw results
back into the troubleshooting panel. They require the extra setup described below and report
"no console is connected" if it is absent; every other tool is unaffected.

## Architecture

Korrel8r is a REST service. This MCP server is a REST client, and holds no correlation
configuration or state of its own: stores, rules and the CPU/memory cost of traversal all live in
the Korrel8r service. An agent makes an MCP call which is forwarded as a REST call to the server,
the original caller's bearer-token is forwarded for authentication:

```
agent  --MCP-->  kubernetes-mcp-server  --REST-->  korrel8r  -->  Prometheus, Loki, Tempo, K8s API
```

## Testing an agent using correlation services

The test is an agent answering a troubleshooting question by requesting a correlation and reporting
the result as text in the session. The queries and class names in that answer are the evidence that
the agent really called Korrel8r rather than guessing or using other observability tools.

Any reachable Korrel8r will do: one deployed by the Cluster Observability Operator, your own
deployment (`make deploy` in the Korrel8r repo installs to the `korrel8r` namespace), or a local
`korrel8r web --http :8090` using your `oc login`.
`make run-korrel8r-evals` creates a dedicated, temporary re-encrypt Route
(`korrel8r-mcp-eval`) to the COO `svc/korrel8r`, waits for HTTPS to be reachable,
and starts the host-local MCP server with a generated drop-in pointing at the
Route. It does not modify `dev/config/mcp-configs/korrel8r.toml` or require a
pre-existing Route. On exit, it stops the MCP server and deletes only the Route
it created. For manual runs, use `make expose-korrel8r`, then
`make run-server TOOLSETS=core,observability/korrel8r MCP_CONFIG_DIR=_output/korrel8r-eval-configs`,
and finally `make stop-server stop-korrel8r`. The host must trust the ingress
certificate CA; see [TLS when running locally](#3-configure-the-mcp-server).

### Example agent session

Start the server with the toolset enabled and point an MCP client at it (Claude Code, Goose,
`npx @modelcontextprotocol/inspector`, ...):

```bash
make run-server TOOLSETS=core,observability/korrel8r
claude mcp add --transport http kubernetes http://localhost:8080/mcp
```

Prompt the agent, asking for the queries explicitly so the answer can be checked. This is the
`pod-correlation` eval task:

> First find an existing OpenShift pod in a namespace beginning with `openshift-`, preferably one
> showing signs of failure. Use correlation to find its related infrastructure logs and alerts, then
> explain which correlated signals might explain any failure. List every correlated class you found
> in DOMAIN:CLASS form together with the queries you used.

The agent should chain several tools, for example `korrel8r_list_domains` and `korrel8r_help` to
learn the query syntax, `korrel8r_create_goals_graph` from the pod to logs and alerts, then
`korrel8r_get_objects` to read the data. An answer then looks roughly like this (illustrative, the
exact classes and selectors depend on the cluster and the Korrel8r rules):

```
Start:  k8s:Pod:{"namespace":"openshift-kube-apiserver","name":"installer-2-node"}
Correlated classes found:
  k8s:Pod             k8s:Pod:{"namespace":"openshift-kube-apiserver","name":"installer-2-node"}
  log:infrastructure  log:infrastructure:{"namespace":"openshift-kube-apiserver","name":"installer-2-node"}

The infrastructure logs show the installer timed out while copying revision files.
No alert was correlated with this pod.
```

Other prompts that force a chain of calls:

- *"What domains does Korrel8r know about, and what query syntax does the log domain use?"*
  exercises `korrel8r_list_domains` then `korrel8r_help`.
- *"What is related to pod X in namespace Y? Show me everything within 2 steps."*
  exercises `korrel8r_create_neighbors_graph`.

### What a good run looks like

- `korrel8r_help` is called before queries are written, not after a failed guess.
- Queries are in `domain:class:selector` form and are quoted back in the answer.
- A `constraint` limit is used on high-volume domains (logs, metrics, traces).
- Conclusions cite objects actually retrieved with `korrel8r_get_objects`. An agent that reports
  graph node counts and stops is describing the correlation without reading the data.

`korrel8r_instructions` is static and answers even with no backend configured, which makes it a
quick check that the toolset is enabled.

### Automated version

The same scenario runs unattended as the `korrel8r` eval suite
(`evals/tasks/korrel8r/`, see [evals/README.md](../../evals/README.md)):

```bash
make run-korrel8r-evals AGENT=acp-anthropic MODEL=sonnet EVAL_TASK_FILTER="Find signals correlated with a failing pod"
```

The suite currently runs only pod correlation; console navigation needs a connected browser and is tested manually below.

The eval config asserts `toolsUsed` with `toolPattern: "korrel8r_.*"`, which is exactly the check
you make by eye in an interactive session: the agent must have called the tools, not recalled an
answer.

## Testing console navigation

This section is only needed for `korrel8r_get_console` and `korrel8r_show_in_console`. It requires
an OpenShift cluster with the Cluster Observability Operator (COO) and the observability stores you
want to correlate (Prometheus, Loki, Tempo).

### How the session works

These two tools work through a session that Korrel8r shares between the browser and the agent:

```
browser console plugin  --REST-->  korrel8r  <--REST--  kubernetes-mcp-server  <--MCP--  agent
   PUT /console                              GET /console           korrel8r_get_console
   GET /console/events (SSE)                 PUT /console/events    korrel8r_show_in_console
```

Two consequences:

- **Same user on both sides.** Korrel8r keys sessions by the username resolved from the bearer token
  via `TokenReview`. The console login and the token this server forwards must belong to the same
  user; the tokens themselves need not be identical.
- **Same Korrel8r instance.** The troubleshooting panel talks to the Korrel8r deployed by COO in
  `openshift-cluster-observability-operator`. Point this toolset at that service; a separately
  deployed Korrel8r has its own sessions and the console will never see the agent's updates.

Only one console SSE listener is allowed per session.

### 1. Enable agent navigation

A custom `ConsolePlugin` is **not** needed: the troubleshooting panel supports agent navigation
behind a COO feature gate.

```yaml
apiVersion: observability.openshift.io/v1alpha1
kind: UIPlugin
metadata:
  name: troubleshooting-panel
spec:
  type: TroubleshootingPanel
  troubleshootingPanel:
    enableAgentNavigation: true
```

Agent navigation needs Korrel8r v0.13.x or later. If COO ships an older image, patch the deployment
and stop the operator from reconciling it back:

```bash
oc scale -n openshift-cluster-observability-operator deploy/observability-operator --replicas=0
oc set image -n openshift-cluster-observability-operator deploy/korrel8r korrel8r=quay.io/korrel8r/korrel8r:0.13.2
```

A custom plugin build is only required if the console plugin itself predates agent navigation. In
that case, in order of increasing effort: patch the image of the COO-created plugin deployment; run
the `openshift/console` bridge locally with `-plugins monitoring-console-plugin=http://localhost:9001`
against a local plugin build; or deploy your own `ConsolePlugin` and add its name to
`consoles.operator.openshift.io/cluster` `spec.plugins`.

### 2. Expose Korrel8r to the MCP server

```bash
oc create route reencrypt --service=korrel8r -n openshift-cluster-observability-operator
export KORREL8R_URL=$(oc get route/korrel8r -n openshift-cluster-observability-operator -o template='https://{{.spec.host}}')
```

### 3. Configure the MCP server

Running locally over stdio, as the same user you log into the console with:

```toml
toolsets = ["core", "observability/korrel8r"]

[toolset_configs."observability/korrel8r"]
url = "https://korrel8r-openshift-cluster-observability-operator.apps.example.com"
auth_mode = "kubeconfig"   # use the `oc login` identity
timeout = "2m"             # optional backstop on a single REST call
```

`timeout` is a backstop against a hung connection, not a traversal budget. Korrel8r bounds its own
work with `tuning.requestTimeout` (`1m` in the configs shipped for OpenShift, unset — meaning no
limit — by default) and returns truncated traversals as successful partial results. The default of
`2m` is longer than that, so the server's own timeout or partial result is what reaches the agent.
Raise `timeout` to stay above `requestTimeout` if the backend is tuned for longer traversals.

Running in HTTP mode, keep the default `auth_mode = "header"` and have the MCP client send the
user's token:

```json
{
  "mcpServers": {
    "kubernetes": {
      "type": "streamable-http",
      "url": "http://localhost:8080/mcp",
      "headers": { "Authorization": "Bearer <oc whoami -t>" }
    }
  }
}
```

In-cluster, use the service DNS name instead of the route:
`url = "https://korrel8r.openshift-cluster-observability-operator.svc:8443"`.

**TLS when running locally.** Backend connections trust the system pool, the Kubernetes CA from the
REST config, and the mounted OpenShift service CA; insecure TLS is not supported. A default route
certificate is signed by the ingress CA, which is in none of those, so add it to the system pool:

```bash
oc extract secret/router-ca -n openshift-ingress-operator --keys=tls.crt --to=- > /tmp/router-ca.pem
cat /etc/pki/tls/certs/ca-bundle.crt /tmp/router-ca.pem > /tmp/ca.pem
SSL_CERT_FILE=/tmp/ca.pem ./kubernetes-mcp-server --config config.toml
```

### 4. Run the scenario

1. Log into the console as the same user, open a workload, and open the troubleshooting panel.
2. Click the AI icon in the panel toolbar and toggle it on. Green means connected to Korrel8r; red
   shows the error in the AI menu.
3. Ask the agent:
   *"What am I looking at in my console? Report the console view query you read in
   DOMAIN:CLASS:SELECTOR form, find related signals that help explain it, then display the useful
   correlation in my console's troubleshooting panel and confirm that you updated it."*
4. Expect `korrel8r_get_console` to return the view query you are looking at, and
   `korrel8r_show_in_console` to redraw the panel's correlation graph.

## Troubleshooting

| Symptom | Likely cause |
|---------|--------------|
| `korrel8r endpoint not configured` | The toolset is enabled but `[toolset_configs."observability/korrel8r"].url` is unset. |
| `no console is connected` | No browser is attached to the session for this user; expected if you are testing without a console. |
| `another console is already connected` | A second listener for the same user; only one is allowed. |
| Console tools read an empty view | The console and the MCP server authenticated as different users, or are using different Korrel8r instances. |
| `x509: certificate signed by unknown authority` | Route certificate CA not trusted: see the `SSL_CERT_FILE` note above. |
| `require_tls` error | Global `require_tls = true` with an `http://` Korrel8r URL. |

Server-side detail is available with `-v=3` on the Korrel8r deployment, which logs each MCP and
REST request.
