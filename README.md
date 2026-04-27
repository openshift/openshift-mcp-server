# Kubernetes MCP Server

[![GitHub License](https://img.shields.io/github/license/containers/kubernetes-mcp-server)](https://github.com/containers/kubernetes-mcp-server/blob/main/LICENSE)
[![npm](https://img.shields.io/npm/v/kubernetes-mcp-server)](https://www.npmjs.com/package/kubernetes-mcp-server)
[![PyPI - Version](https://img.shields.io/pypi/v/kubernetes-mcp-server)](https://pypi.org/project/kubernetes-mcp-server/)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/containers/kubernetes-mcp-server?sort=semver)](https://github.com/containers/kubernetes-mcp-server/releases/latest)
[![Build](https://github.com/containers/kubernetes-mcp-server/actions/workflows/build.yaml/badge.svg)](https://github.com/containers/kubernetes-mcp-server/actions/workflows/build.yaml)

[✨ Features](#features) | [🚀 Getting Started](#getting-started) | [🎥 Demos](#demos) | [⚙️ Configuration](#configuration) | [🛠️ Tools](#tools-and-functionalities) | [💬 Community](#community) | [🧑‍💻 Development](#development)

https://github.com/user-attachments/assets/be2b67b3-fc1c-4d11-ae46-93deba8ed98e

## ✨ Features <a id="features"></a>

A powerful and flexible Kubernetes [Model Context Protocol (MCP)](https://blog.marcnuri.com/model-context-protocol-mcp-introduction) server implementation with support for **Kubernetes** and **OpenShift**.

- **✅ Configuration**:
  - Automatically detect changes in the Kubernetes configuration and update the MCP server.
  - **View** and manage the current [Kubernetes `.kube/config`](https://blog.marcnuri.com/where-is-my-default-kubeconfig-file) or in-cluster configuration.
- **✅ Generic Kubernetes Resources**: Perform operations on **any** Kubernetes or OpenShift resource.
  - Any CRUD operation (Create or Update, Get, List, Delete).
- **✅ Pods**: Perform Pod-specific operations.
  - **List** pods in all namespaces or in a specific namespace.
  - **Get** a pod by name from the specified namespace.
  - **Delete** a pod by name from the specified namespace.
  - **Show logs** for a pod by name from the specified namespace.
  - **Top** gets resource usage metrics for all pods or a specific pod in the specified namespace.
  - **Exec** into a pod and run a command.
  - **Run** a container image in a pod and optionally expose it.
- **✅ Namespaces**: List Kubernetes Namespaces.
- **✅ Events**: View Kubernetes events in all namespaces or in a specific namespace.
- **✅ Projects**: List OpenShift Projects.
- **☸️ Helm**:
  - **Install** a Helm chart in the current or provided namespace.
  - **List** Helm releases in all namespaces or in a specific namespace.
  - **Uninstall** a Helm release in the current or provided namespace.
- **🔧 Tekton**: Tekton-specific operations that complement generic Kubernetes resource management.
  - **Pipeline**: Start a Tekton Pipeline by creating a PipelineRun.
  - **PipelineRun**: Restart a PipelineRun with the same spec.
  - **Task**: Start a Tekton Task by creating a TaskRun.
  - **TaskRun**: Restart a TaskRun with the same spec, and retrieve TaskRun logs via pod resolution.
- **🔭 Observability**: Optional OpenTelemetry distributed tracing and metrics with custom sampling rates. Includes `/stats` endpoint for real-time statistics. See [OTEL.md](docs/OTEL.md).

Unlike other Kubernetes MCP server implementations, this **IS NOT** just a wrapper around `kubectl` or `helm` command-line tools.
It is a **Go-based native implementation** that interacts directly with the Kubernetes API server.

There is **NO NEED** for external dependencies or tools to be installed on the system.
If you're using the native binaries you don't need to have Node or Python installed on your system.

- **✅ Lightweight**: The server is distributed as a single native binary for Linux, macOS, and Windows.
- **✅ High-Performance / Low-Latency**: Directly interacts with the Kubernetes API server without the overhead of calling and waiting for external commands.
- **✅ Multi-Cluster**: Can interact with multiple Kubernetes clusters simultaneously (as defined in your kubeconfig files).
- **✅ Cross-Platform**: Available as a native binary for Linux, macOS, and Windows, as well as an npm package, a Python package, and container/Docker image.
- **✅ Configurable**: Supports [command-line arguments](#configuration), [TOML configuration files](docs/configuration.md), and environment variables.
- **✅ Well tested**: The server has an extensive test suite to ensure its reliability and correctness across different Kubernetes environments.
- **📚 Documentation**: Comprehensive [user documentation](docs/) including setup guides, configuration reference, and observability.

## 🚀 Getting Started <a id="getting-started"></a>

### Requirements

- Access to a Kubernetes cluster.

<details>
<summary><b>Claude Code</b></summary>

Follow the [dedicated Claude Code getting started guide](docs/getting-started-claude-code.md) in our [user documentation](docs/).

For a secure production setup with dedicated ServiceAccount and read-only access, also review the [Kubernetes setup guide](docs/getting-started-kubernetes.md).

</details>

### Claude Desktop

#### Using npx

If you have npm installed, this is the fastest way to get started with `kubernetes-mcp-server` on Claude Desktop.

Open your `claude_desktop_config.json` and add the mcp server to the list of `mcpServers`:
``` json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": [
        "-y",
        "kubernetes-mcp-server@latest"
      ]
    }
  }
}
```

### VS Code / VS Code Insiders

Install the Kubernetes MCP server extension in VS Code Insiders by pressing the following link:

[<img src="https://img.shields.io/badge/VS_Code-VS_Code?style=flat-square&label=Install%20Server&color=0098FF" alt="Install in VS Code">](https://insiders.vscode.dev/redirect?url=vscode%3Amcp%2Finstall%3F%257B%2522name%2522%253A%2522kubernetes%2522%252C%2522command%2522%253A%2522npx%2522%252C%2522args%2522%253A%255B%2522-y%2522%252C%2522kubernetes-mcp-server%2540latest%2522%255D%257D)
[<img alt="Install in VS Code Insiders" src="https://img.shields.io/badge/VS_Code_Insiders-VS_Code_Insiders?style=flat-square&label=Install%20Server&color=24bfa5">](https://insiders.vscode.dev/redirect?url=vscode-insiders%3Amcp%2Finstall%3F%257B%2522name%2522%253A%2522kubernetes%2522%252C%2522command%2522%253A%2522npx%2522%252C%2522args%2522%253A%255B%2522-y%2522%252C%2522kubernetes-mcp-server%2540latest%2522%255D%257D)

Alternatively, you can install the extension manually by running the following command:

```shell
# For VS Code
code --add-mcp '{"name":"kubernetes","command":"npx","args":["kubernetes-mcp-server@latest"]}'
# For VS Code Insiders
code-insiders --add-mcp '{"name":"kubernetes","command":"npx","args":["kubernetes-mcp-server@latest"]}'
```

### Cursor

Install the Kubernetes MCP server extension in Cursor by pressing the following link:

[![Install MCP Server](https://cursor.com/deeplink/mcp-install-dark.svg)](https://cursor.com/en/install-mcp?name=kubernetes-mcp-server&config=eyJjb21tYW5kIjoibnB4IC15IGt1YmVybmV0ZXMtbWNwLXNlcnZlckBsYXRlc3QifQ%3D%3D)

Alternatively, you can install the extension manually by editing the `mcp.json` file:

```json
{
  "mcpServers": {
    "kubernetes-mcp-server": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest"]
    }
  }
}
```

### Goose CLI

[Goose CLI](https://blog.marcnuri.com/goose-on-machine-ai-agent-cli-introduction) is the easiest (and cheapest) way to get rolling with artificial intelligence (AI) agents.

#### Using npm

If you have npm installed, this is the fastest way to get started with `kubernetes-mcp-server`.

Open your goose `config.yaml` and add the mcp server to the list of `mcpServers`:
```yaml
extensions:
  kubernetes:
    command: npx
    args:
      - -y
      - kubernetes-mcp-server@latest

```

## 🎥 Demos <a id="demos"></a>

### Diagnosing and automatically fixing an OpenShift Deployment

Demo showcasing how Kubernetes MCP server is leveraged by Claude Desktop to automatically diagnose and fix a deployment in OpenShift without any user assistance.

https://github.com/user-attachments/assets/a576176d-a142-4c19-b9aa-a83dc4b8d941

### _Vibe Coding_ a simple game and deploying it to OpenShift

In this demo, I walk you through the process of _Vibe Coding_ a simple game using VS Code and how to leverage [Podman MCP server](https://github.com/manusa/podman-mcp-server) and Kubernetes MCP server to deploy it to OpenShift.

<a href="https://www.youtube.com/watch?v=l05jQDSrzVI" target="_blank">
 <img src="docs/images/vibe-coding.jpg" alt="Vibe Coding: Build & Deploy a Game on Kubernetes" width="240"  />
</a>

### Supercharge GitHub Copilot with Kubernetes MCP Server in VS Code - One-Click Setup!

In this demo, I'll show you how to set up Kubernetes MCP server in VS code just by clicking a link.

<a href="https://youtu.be/AI4ljYMkgtA" target="_blank">
 <img src="docs/images/kubernetes-mcp-server-github-copilot.jpg" alt="Supercharge GitHub Copilot with Kubernetes MCP Server in VS Code - One-Click Setup!" width="240"  />
</a>

## ⚙️ Configuration <a id="configuration"></a>

The Kubernetes MCP server can be configured using command line (CLI) arguments.

You can run the CLI executable either by using `npx`, `uvx`, or by downloading the [latest release binary](https://github.com/containers/kubernetes-mcp-server/releases/latest).

```shell
# Run the Kubernetes MCP server using npx (in case you have npm and node installed)
npx kubernetes-mcp-server@latest --help
```

```shell
# Run the Kubernetes MCP server using uvx (in case you have uv and python installed)
uvx kubernetes-mcp-server@latest --help
```

```shell
# Run the Kubernetes MCP server using the latest release binary
./kubernetes-mcp-server --help
```

### Configuration Options

| Option                    | Description                                                                                                                                                                                                                                                                                   |
|---------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `--port`                  | Starts the MCP server in Streamable HTTP mode (path /mcp) and Server-Sent Event (SSE) (path /sse) mode and listens on the specified port .                                                                                                                                                    |
| `--log-level`             | Sets the logging level (values [from 0-9](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md)). Similar to [kubectl logging levels](https://kubernetes.io/docs/reference/kubectl/quick-reference/#kubectl-output-verbosity-and-debugging). |
| `--config`                | (Optional) Path to the main TOML configuration file. See [Configuration Reference](docs/configuration.md) for details.                                                                                                                                                                        |
| `--config-dir`            | (Optional) Path to drop-in configuration directory. Files are loaded in lexical (alphabetical) order. Defaults to `conf.d` relative to the main config file if `--config` is specified. See [Configuration Reference](docs/configuration.md) for details.                                    |
| `--kubeconfig`            | Path to the Kubernetes configuration file. If not provided, it will try to resolve the configuration (in-cluster, default location, etc.).                                                                                                                                                    |
| `--list-output`           | Output format for resource list operations (one of: yaml, table) (default "table")                                                                                                                                                                                                            |
| `--read-only`             | If set, the MCP server will run in read-only mode, meaning it will not allow any write operations (create, update, delete) on the Kubernetes cluster. This is useful for debugging or inspecting the cluster without making changes.                                                          |
| `--disable-destructive`   | If set, the MCP server will disable all destructive operations (delete, update, etc.) on the Kubernetes cluster. This is useful for debugging or inspecting the cluster without accidentally making changes. This option has no effect when `--read-only` is used.                            |
| `--stateless`             | If set, the MCP server will run in stateless mode, disabling tool and prompt change notifications. This is useful for container deployments, load balancing, and serverless environments where maintaining client state is not desired.                                                       |
| `--toolsets`              | Comma-separated list of toolsets to enable. Check the [🛠️ Tools and Functionalities](#tools-and-functionalities) section for more information.                                                                                                                                                |
| `--disable-multi-cluster` | If set, the MCP server will disable multi-cluster support and will only use the current context from the kubeconfig file. This is useful if you want to restrict the MCP server to a single cluster.                                                                                          |
| `--cluster-provider`      | Cluster provider strategy to use (one of: kubeconfig, in-cluster, kcp, disabled). If not set, the server will auto-detect based on the environment.                                                                                                                                           |

> **Note**: Most CLI options have equivalent TOML configuration fields. The `--disable-multi-cluster` flag is equivalent to setting `cluster_provider_strategy = "disabled"` in TOML. See the [Configuration Reference](docs/configuration.md) for all TOML options.

### TOML Configuration Files

For complex or persistent configurations, use TOML configuration files instead of CLI arguments:

```shell
kubernetes-mcp-server --config /etc/kubernetes-mcp-server/config.toml
```

**Example configuration:**

```toml
log_level = 2
read_only = true
toolsets = ["core", "config", "helm", "kubevirt"]

# Deny access to sensitive resources
[[denied_resources]]
group = ""
version = "v1"
kind = "Secret"

[telemetry]
endpoint = "http://localhost:4317"
```

For comprehensive TOML configuration documentation, including:
- All configuration options and their defaults
- Drop-in configuration files for modular settings
- Dynamic configuration reload via SIGHUP
- Denied resources for restricting access to sensitive resource types
- Server instructions for MCP Tool Search
- [Custom MCP prompts](docs/prompts.md)
- OAuth/OIDC authentication for HTTP mode ([Keycloak](docs/KEYCLOAK_OIDC_SETUP.md), [Microsoft Entra ID](docs/ENTRA_ID_SETUP.md))

See the **[Configuration Reference](docs/configuration.md)**.

## 📊 MCP Logging <a id="mcp-logging"></a>

The server supports the MCP logging capability, allowing clients to receive debugging information via structured log messages.
Kubernetes API errors are automatically categorized and logged to clients with appropriate severity levels.
Sensitive data (tokens, keys, passwords, cloud credentials) is automatically redacted before being sent to clients.

See the **[MCP Logging Guide](docs/logging.md)**.

## 🛠️ Tools and Functionalities <a id="tools-and-functionalities"></a>

The Kubernetes MCP server supports enabling or disabling specific groups of tools and functionalities (tools, resources, prompts, and so on) via the `--toolsets` command-line flag or `toolsets` configuration option.
This allows you to control which Kubernetes functionalities are available to your AI tools.
Enabling only the toolsets you need can help reduce the context size and improve the LLM's tool selection accuracy.

### Available Toolsets

The following sets of tools are available (toolsets marked with ✓ in the Default column are enabled by default):

<!-- AVAILABLE-TOOLSETS-START -->

| Toolset  | Description                                                                                                                                                                     | Default |
|----------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| config   | View and manage the current local Kubernetes configuration (kubeconfig)                                                                                                         | ✓       |
| core     | Most common tools for Kubernetes management (Pods, Generic Resources, Events, etc.)                                                                                             | ✓       |
| helm     | Tools for managing Helm charts and releases                                                                                                                                     |         |
| kcp      | Manage kcp workspaces and multi-tenancy features                                                                                                                                |         |
| kubevirt | KubeVirt virtual machine management tools, check the [KubeVirt documentation](https://github.com/containers/kubernetes-mcp-server/blob/main/docs/kubevirt.md) for more details. |         |
| metrics  | Toolset for querying Prometheus and Alertmanager endpoints in efficient ways.                                                                                                   |         |
| oadp     | OADP (OpenShift API for Data Protection) tools for managing Velero backups, restores, and schedules                                                                             |         |
| ossm     | Most common tools for managing OSSM, check the [OSSM documentation](https://github.com/openshift/openshift-mcp-server/blob/main/docs/OSSM.md) for more details.                 |         |
| tekton   | Tekton pipeline management tools for Pipelines, PipelineRuns, Tasks, and TaskRuns.                                                                                              |         |

<!-- AVAILABLE-TOOLSETS-END -->

### Tools

In case multi-cluster support is enabled (default) and you have access to multiple clusters, all applicable tools will include an additional `context` argument to specify the Kubernetes context (cluster) to use for that operation.

<!-- AVAILABLE-TOOLSETS-TOOLS-START -->

<details>

<summary>config</summary>

- **configuration_contexts_list** - List all available context names and associated server urls from the kubeconfig file

- **targets_list** - List all available targets

- **configuration_view** - Get the current Kubernetes configuration content as a kubeconfig YAML
  - `minified` (`boolean`) - Return a minified version of the configuration. If set to true, keeps only the current-context and the relevant pieces of the configuration for that context. If set to false, all contexts, clusters, auth-infos, and users are returned in the configuration. (Optional, default true)

</details>

<details>

<summary>core</summary>

- **events_list** - List Kubernetes events (warnings, errors, state changes) for debugging and troubleshooting in the current cluster from all namespaces
  - `namespace` (`string`) - Optional Namespace to retrieve the events from. If not provided, will list events from all namespaces

- **namespaces_list** - List all the Kubernetes namespaces in the current cluster

- **projects_list** - List all the OpenShift projects in the current cluster

- **nodes_log** - Get logs from a Kubernetes node (kubelet, kube-proxy, or other system logs). This accesses node logs through the Kubernetes API proxy to the kubelet
  - `name` (`string`) **(required)** - Name of the node to get logs from
  - `query` (`string`) **(required)** - query specifies services(s) or files from which to return logs (required). Example: "kubelet" to fetch kubelet logs, "/<log-file-name>" to fetch a specific log file from the node (e.g., "/var/log/kubelet.log" or "/var/log/kube-proxy.log")
  - `tailLines` (`integer`) - Number of lines to retrieve from the end of the logs (Optional, 0 means all logs)

- **nodes_stats_summary** - Get detailed resource usage statistics from a Kubernetes node via the kubelet's Summary API. Provides comprehensive metrics including CPU, memory, filesystem, and network usage at the node, pod, and container levels. On systems with cgroup v2 and kernel 4.20+, also includes PSI (Pressure Stall Information) metrics that show resource pressure for CPU, memory, and I/O. See https://kubernetes.io/docs/reference/instrumentation/understand-psi-metrics/ for details on PSI metrics
  - `name` (`string`) **(required)** - Name of the node to get stats from

- **nodes_top** - List the resource consumption (CPU and memory) as recorded by the Kubernetes Metrics Server for the specified Kubernetes Nodes or all nodes in the cluster
  - `label_selector` (`string`) - Kubernetes label selector (e.g. 'node-role.kubernetes.io/worker=') to filter nodes by label (Optional, only applicable when name is not provided)
  - `name` (`string`) - Name of the Node to get the resource consumption from (Optional, all Nodes if not provided)

- **pods_list** - List all the Kubernetes pods in the current cluster from all namespaces
  - `fieldSelector` (`string`) - Optional Kubernetes field selector to filter pods by field values (e.g. 'status.phase=Running', 'spec.nodeName=node1'). Supported fields: metadata.name, metadata.namespace, spec.nodeName, spec.restartPolicy, spec.schedulerName, spec.serviceAccountName, status.phase (Pending/Running/Succeeded/Failed/Unknown), status.podIP, status.nominatedNodeName. Note: CrashLoopBackOff is a container state, not a pod phase, so it cannot be filtered directly. See https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/
  - `labelSelector` (`string`) - Optional Kubernetes label selector (e.g. 'app=myapp,env=prod' or 'app in (myapp,yourapp)'), use this option when you want to filter the pods by label

- **pods_list_in_namespace** - List all the Kubernetes pods in the specified namespace in the current cluster
  - `fieldSelector` (`string`) - Optional Kubernetes field selector to filter pods by field values (e.g. 'status.phase=Running', 'spec.nodeName=node1'). Supported fields: metadata.name, metadata.namespace, spec.nodeName, spec.restartPolicy, spec.schedulerName, spec.serviceAccountName, status.phase (Pending/Running/Succeeded/Failed/Unknown), status.podIP, status.nominatedNodeName. Note: CrashLoopBackOff is a container state, not a pod phase, so it cannot be filtered directly. See https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/
  - `labelSelector` (`string`) - Optional Kubernetes label selector (e.g. 'app=myapp,env=prod' or 'app in (myapp,yourapp)'), use this option when you want to filter the pods by label
  - `namespace` (`string`) **(required)** - Namespace to list pods from

- **pods_get** - Get a Kubernetes Pod in the current or provided namespace with the provided name
  - `name` (`string`) **(required)** - Name of the Pod
  - `namespace` (`string`) - Namespace to get the Pod from

- **pods_delete** - Delete a Kubernetes Pod in the current or provided namespace with the provided name
  - `name` (`string`) **(required)** - Name of the Pod to delete
  - `namespace` (`string`) - Namespace to delete the Pod from

- **pods_top** - List the resource consumption (CPU and memory) as recorded by the Kubernetes Metrics Server for the specified Kubernetes Pods in the all namespaces, the provided namespace, or the current namespace
  - `all_namespaces` (`boolean`) - If true, list the resource consumption for all Pods in all namespaces. If false, list the resource consumption for Pods in the provided namespace or the current namespace
  - `label_selector` (`string`) - Kubernetes label selector (e.g. 'app=myapp,env=prod' or 'app in (myapp,yourapp)'), use this option when you want to filter the pods by label (Optional, only applicable when name is not provided)
  - `name` (`string`) - Name of the Pod to get the resource consumption from (Optional, all Pods in the namespace if not provided)
  - `namespace` (`string`) - Namespace to get the Pods resource consumption from (Optional, current namespace if not provided and all_namespaces is false)

- **pods_exec** - Execute a command in a Kubernetes Pod (shell access, run commands in container) in the current or provided namespace with the provided name and command
  - `command` (`array`) **(required)** - Command to execute in the Pod container. The first item is the command to be run, and the rest are the arguments to that command. Example: ["ls", "-l", "/tmp"]
  - `container` (`string`) - Name of the Pod container where the command will be executed (Optional)
  - `name` (`string`) **(required)** - Name of the Pod where the command will be executed
  - `namespace` (`string`) - Namespace of the Pod where the command will be executed

- **pods_log** - Get the logs of a Kubernetes Pod in the current or provided namespace with the provided name
  - `container` (`string`) - Name of the Pod container to get the logs from (Optional)
  - `name` (`string`) **(required)** - Name of the Pod to get the logs from
  - `namespace` (`string`) - Namespace to get the Pod logs from
  - `previous` (`boolean`) - Return previous terminated container logs (Optional)
  - `tail` (`integer`) - Number of lines to retrieve from the end of the logs (Optional, default: 100)

- **pods_run** - Run a Kubernetes Pod in the current or provided namespace with the provided container image and optional name
  - `image` (`string`) **(required)** - Container Image to run in the Pod
  - `name` (`string`) - Name of the Pod (Optional, random name if not provided)
  - `namespace` (`string`) - Namespace to run the Pod in
  - `port` (`number`) - TCP/IP port to expose from the Pod container (Optional, no port exposed if not provided)

- **resources_list** - List Kubernetes resources and objects in the current cluster by providing their apiVersion and kind and optionally the namespace and label selector
(common apiVersion and kind include: v1 Pod, v1 Service, v1 Node, apps/v1 Deployment, networking.k8s.io/v1 Ingress, route.openshift.io/v1 Route)
  - `apiVersion` (`string`) **(required)** - apiVersion of the resources (examples of valid apiVersion are: v1, apps/v1, networking.k8s.io/v1)
  - `fieldSelector` (`string`) - Optional Kubernetes field selector to filter resources by field values (e.g. 'status.phase=Running', 'metadata.name=myresource'). Supported fields vary by resource type. For Pods: metadata.name, metadata.namespace, spec.nodeName, spec.restartPolicy, spec.schedulerName, spec.serviceAccountName, status.phase (Pending/Running/Succeeded/Failed/Unknown), status.podIP, status.nominatedNodeName. See https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/
  - `kind` (`string`) **(required)** - kind of the resources (examples of valid kind are: Pod, Service, Deployment, Ingress)
  - `labelSelector` (`string`) - Optional Kubernetes label selector (e.g. 'app=myapp,env=prod' or 'app in (myapp,yourapp)'), use this option when you want to filter the resources by label
  - `namespace` (`string`) - Optional Namespace to retrieve the namespaced resources from (ignored in case of cluster scoped resources). If not provided, will list resources from all namespaces

- **resources_get** - Get a Kubernetes resource in the current cluster by providing its apiVersion, kind, optionally the namespace, and its name
(common apiVersion and kind include: v1 Pod, v1 Service, v1 Node, apps/v1 Deployment, networking.k8s.io/v1 Ingress, route.openshift.io/v1 Route)
  - `apiVersion` (`string`) **(required)** - apiVersion of the resource (examples of valid apiVersion are: v1, apps/v1, networking.k8s.io/v1)
  - `kind` (`string`) **(required)** - kind of the resource (examples of valid kind are: Pod, Service, Deployment, Ingress)
  - `name` (`string`) **(required)** - Name of the resource
  - `namespace` (`string`) - Optional Namespace to retrieve the namespaced resource from (ignored in case of cluster scoped resources). If not provided, will get resource from configured namespace

- **resources_create_or_update** - Create or update a Kubernetes resource in the current cluster by providing a YAML or JSON representation of the resource
(common apiVersion and kind include: v1 Pod, v1 Service, v1 Node, apps/v1 Deployment, networking.k8s.io/v1 Ingress, route.openshift.io/v1 Route)
  - `resource` (`string`) **(required)** - A JSON or YAML containing a representation of the Kubernetes resource. Should include top-level fields such as apiVersion,kind,metadata, and spec

- **resources_delete** - Delete a Kubernetes resource in the current cluster by providing its apiVersion, kind, optionally the namespace, and its name
(common apiVersion and kind include: v1 Pod, v1 Service, v1 Node, apps/v1 Deployment, networking.k8s.io/v1 Ingress, route.openshift.io/v1 Route)
  - `apiVersion` (`string`) **(required)** - apiVersion of the resource (examples of valid apiVersion are: v1, apps/v1, networking.k8s.io/v1)
  - `gracePeriodSeconds` (`integer`) - Optional duration in seconds before the object should be deleted. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period for the specified type will be used
  - `kind` (`string`) **(required)** - kind of the resource (examples of valid kind are: Pod, Service, Deployment, Ingress)
  - `name` (`string`) **(required)** - Name of the resource
  - `namespace` (`string`) - Optional Namespace to delete the namespaced resource from (ignored in case of cluster scoped resources). If not provided, will delete resource from configured namespace

- **resources_scale** - Get or update the scale of a Kubernetes resource in the current cluster by providing its apiVersion, kind, name, and optionally the namespace. If the scale is set in the tool call, the scale will be updated to that value. Always returns the current scale of the resource
  - `apiVersion` (`string`) **(required)** - apiVersion of the resource (examples of valid apiVersion are apps/v1)
  - `kind` (`string`) **(required)** - kind of the resource (examples of valid kind are: StatefulSet, Deployment)
  - `name` (`string`) **(required)** - Name of the resource
  - `namespace` (`string`) - Optional Namespace to get/update the namespaced resource scale from (ignored in case of cluster scoped resources). If not provided, will get/update resource scale from configured namespace
  - `scale` (`integer`) - Optional scale to update the resources scale to. If not provided, will return the current scale of the resource, and not update it

</details>

<details>

<summary>helm</summary>

- **helm_install** - Install (deploy) a Helm chart to create a release in the current or provided namespace
  - `chart` (`string`) **(required)** - Chart reference to install (for example: stable/grafana, oci://ghcr.io/nginxinc/charts/nginx-ingress)
  - `name` (`string`) - Name of the Helm release (Optional, random name if not provided)
  - `namespace` (`string`) - Namespace to install the Helm chart in (Optional, current namespace if not provided)
  - `values` (`object`) - Values to pass to the Helm chart (Optional)

- **helm_list** - List all the Helm releases in the current or provided namespace (or in all namespaces if specified)
  - `all_namespaces` (`boolean`) - If true, lists all Helm releases in all namespaces ignoring the namespace argument (Optional)
  - `namespace` (`string`) - Namespace to list Helm releases from (Optional, all namespaces if not provided)

- **helm_uninstall** - Uninstall a Helm release in the current or provided namespace
  - `name` (`string`) **(required)** - Name of the Helm release to uninstall
  - `namespace` (`string`) - Namespace to uninstall the Helm release from (Optional, current namespace if not provided)

</details>

<details>

<summary>kcp</summary>

- **kcp_workspaces_list** - List all available kcp workspaces in the current cluster

- **kcp_workspace_describe** - Get detailed information about a specific kcp workspace
  - `workspace` (`string`) **(required)** - Name or path of the workspace to describe

</details>

<details>

<summary>kubevirt</summary>

- **vm_clone** - Clone a KubeVirt VirtualMachine by creating a VirtualMachineClone resource. This creates a copy of the source VM with a new name using the KubeVirt Clone API
  - `name` (`string`) **(required)** - The name of the source virtual machine to clone
  - `namespace` (`string`) **(required)** - The namespace of the source virtual machine
  - `targetName` (`string`) **(required)** - The name for the new cloned virtual machine

- **vm_create** - Create a VirtualMachine in the cluster with the specified configuration, automatically resolving instance types, preferences, and container disk images. VM will be created in Halted state by default; use autostart parameter to start it immediately.
  - `autostart` (`boolean`) - Optional flag to automatically start the VM after creation (sets runStrategy to Always instead of Halted). Defaults to false.
  - `instancetype` (`string`) - Optional instance type name for the VM (e.g., 'u1.small', 'u1.medium', 'u1.large')
  - `name` (`string`) **(required)** - The name of the virtual machine
  - `namespace` (`string`) **(required)** - The namespace for the virtual machine
  - `networks` (`array`) - Optional secondary network interfaces to attach to the VM. Each item specifies a Multus NetworkAttachmentDefinition to attach. Accepts either simple strings (NetworkAttachmentDefinition names) or objects with 'name' (interface name in VM) and 'networkName' (NetworkAttachmentDefinition name) properties. Each network creates a bridge interface on the VM.
  - `performance` (`string`) - Optional performance family hint for the VM instance type (e.g., 'u1' for general-purpose, 'o1' for overcommitted, 'c1' for compute-optimized, 'm1' for memory-optimized). Defaults to 'u1' (general-purpose) if not specified.
  - `preference` (`string`) - Optional preference name for the VM
  - `size` (`string`) - Optional workload size hint for the VM (e.g., 'small', 'medium', 'large', 'xlarge'). Used to auto-select an appropriate instance type if not explicitly specified.
  - `storage` (`string`) - Optional storage size for the VM's root disk when using DataSources (e.g., '30Gi', '50Gi', '100Gi'). Defaults to 30Gi. Ignored when using container disks.
  - `workload` (`string`) - The workload for the VM. Accepts OS names (e.g., 'fedora' (default), 'ubuntu', 'centos', 'centos-stream', 'debian', 'rhel', 'opensuse', 'opensuse-tumbleweed', 'opensuse-leap') or full container disk image URLs

- **vm_lifecycle** - Manage VirtualMachine lifecycle: start, stop, or restart a VM
  - `action` (`string`) **(required)** - The lifecycle action to perform: 'start' (changes runStrategy to Always), 'stop' (changes runStrategy to Halted), or 'restart' (stops then starts the VM)
  - `name` (`string`) **(required)** - The name of the virtual machine
  - `namespace` (`string`) **(required)** - The namespace of the virtual machine

</details>

<details>

<summary>metrics</summary>

- **list_metrics** - MANDATORY FIRST STEP: List all available metric names in Prometheus.

YOU MUST CALL THIS TOOL BEFORE ANY OTHER QUERY TOOL

This tool MUST be called first for EVERY observability question to:
1. Discover what metrics actually exist in this environment
2. Find the EXACT metric name to use in queries
3. Avoid querying non-existent metrics
4. The 'name_regex' parameter should always be provided, and be a best guess of what the metric would be named like.
5. Do not use a blanket regex like .* or .+ in the 'name_regex' parameter. Use specific ones like kube.*, node.*, etc.

REGEX PATTERN GUIDANCE:
- Prometheus metrics are typically prefixed (e.g., 'prometheus_tsdb_head_series', 'kube_pod_status_phase')
- To match metrics CONTAINING a substring, use wildcards: '.*tsdb.*' matches 'prometheus_tsdb_head_series'
- Without wildcards, the pattern matches EXACTLY: 'tsdb' only matches a metric literally named 'tsdb' (which rarely exists)
- Common patterns: 'kube_pod.*' (pods), '.*memory.*' (memory-related), 'node_.*' (node metrics)
- If you get empty results, try adding '.*' before/after your search term

NEVER skip this step. NEVER guess metric names. Metric names vary between environments.

After calling this tool:
1. Search the returned list for relevant metrics
2. Use the EXACT metric name found in subsequent queries
3. If no relevant metric exists, inform the user
  - `name_regex` (`string`) **(required)** - Regex pattern to filter metric names. IMPORTANT: Metric names are typically prefixed (e.g., 'prometheus_tsdb_head_series'). Use wildcards to match substrings: '.*tsdb.*' matches any metric containing 'tsdb', while 'tsdb' only matches the exact string 'tsdb'. Examples: 'http_.*' (starts with http_), '.*memory.*' (contains memory), 'node_.*' (starts with node_). This parameter is required. Don't pass in blanket regex like '.*' or '.+'.

- **execute_instant_query** - Execute a PromQL instant query to get current/point-in-time values.

PREREQUISITE: You MUST call list_metrics first to verify the metric exists

WHEN TO USE:
- Current state questions: "What is the current error rate?"
- Point-in-time snapshots: "How many pods are running?"
- Latest values: "Which pods are in Pending state?"

The 'query' parameter MUST use metric names that were returned by list_metrics.
  - `query` (`string`) **(required)** - PromQL query string using metric names verified via list_metrics
  - `time` (`string`) - Evaluation time as RFC3339 or Unix timestamp. Omit or use 'NOW' for current time.

- **execute_range_query** - Execute a PromQL range query to get time-series data over a period.

PREREQUISITE: You MUST call list_metrics first to verify the metric exists

WHEN TO USE:
- Trends over time: "What was CPU usage over the last hour?"
- Rate calculations: "How many requests per second?"
- Historical analysis: "Were there any restarts in the last 5 minutes?"

TIME PARAMETERS:
- 'duration': Look back from now (e.g., "5m", "1h", "24h")
- 'step': Data point resolution (e.g., "1m" for 1-hour duration, "5m" for 24-hour duration)

The 'query' parameter MUST use metric names that were returned by list_metrics.
  - `duration` (`string`) - Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional)
  - `end` (`string`) - End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time.
  - `query` (`string`) **(required)** - PromQL query string using metric names verified via list_metrics
  - `start` (`string`) - Start time as RFC3339 or Unix timestamp (optional)
  - `step` (`string`) **(required)** - Query resolution step width (e.g., '15s', '1m', '1h'). Choose based on time range: shorter ranges use smaller steps.

- **show_timeseries** - Display the results as an interactive timeseries chart.

This tool works like execute_range_query but renders the results as a visual chart in the UI clients.
Use it when the user wants to see a graph or visualization of time-series data and to use visuals to provide the answer.
Use the show_timeseries as the last tool call after all the other Prometheus tool calls where finalized.

TIME PARAMETERS:
- 'duration': Look back from now (e.g., "5m", "1h", "24h")
- 'step': Data point resolution (e.g., "1m" for 1-hour duration, "5m" for 24-hour duration)
- 'title': A descriptive chart title (e.g., "API Error Rate Over Last Hour")
- 'description': An explanation of the chart's meaning or context (e.g., "Shows the rate of HTTP 5xx errors per second, broken down by pod")

The 'query' parameter MUST be a range query and must use metric names that were returned by list_metrics.
  - `description` (`string`) - Explanation of the chart's meaning or context (e.g., 'Shows the rate of HTTP 5xx errors per second, broken down by pod'). Displayed below the title when provided.
  - `duration` (`string`) - Duration to look back from now (e.g., '1h', '30m', '1d', '2w') (optional)
  - `end` (`string`) - End time as RFC3339 or Unix timestamp (optional). Use `NOW` for current time.
  - `query` (`string`) **(required)** - PromQL query string using metric names verified via list_metrics
  - `start` (`string`) - Start time as RFC3339 or Unix timestamp (optional)
  - `step` (`string`) **(required)** - Query resolution step width (e.g., '15s', '1m', '1h'). Choose based on time range: shorter ranges use smaller steps.
  - `title` (`string`) - Human-readable chart title describing what the query shows (e.g., 'API Error Rate Over Last Hour'). Displayed above the chart when provided.

- **get_label_names** - Get all label names (dimensions) available for filtering a metric.

WHEN TO USE (after calling list_metrics):
- To discover how to filter metrics (by namespace, pod, service, etc.)
- Before constructing label matchers in PromQL queries

The 'metric' parameter should use a metric name from list_metrics output.
  - `end` (`string`) - End time for label discovery as RFC3339 or Unix timestamp (optional, defaults to now)
  - `metric` (`string`) - Metric name (from list_metrics) to get label names for. Leave empty for all metrics.
  - `start` (`string`) - Start time for label discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)

- **get_label_values** - Get all unique values for a specific label.

WHEN TO USE (after calling list_metrics and get_label_names):
- To find exact label values for filtering (namespace names, pod names, etc.)
- To see what values exist before constructing queries

The 'metric' parameter should use a metric name from list_metrics output.
  - `end` (`string`) - End time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to now)
  - `label` (`string`) **(required)** - Label name (from get_label_names) to get values for
  - `metric` (`string`) - Metric name (from list_metrics) to scope the label values to. Leave empty for all metrics.
  - `start` (`string`) - Start time for label value discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)

- **get_series** - Get time series matching selectors and preview cardinality.

WHEN TO USE (optional, after calling list_metrics):
- To verify label filters match expected series before querying
- To check cardinality and avoid slow queries

CARDINALITY GUIDANCE:
- <100 series: Safe
- 100-1000: Usually fine
- >1000: Add more label filters

The selector should use metric names from list_metrics output.
  - `end` (`string`) - End time for series discovery as RFC3339 or Unix timestamp (optional, defaults to now)
  - `matches` (`string`) **(required)** - PromQL series selector using metric names from list_metrics
  - `start` (`string`) - Start time for series discovery as RFC3339 or Unix timestamp (optional, defaults to 1 hour ago)

- **get_alerts** - Get alerts from Alertmanager.

WHEN TO USE:
- START HERE when investigating issues: if the user asks about things breaking, errors, failures, outages, services being down, or anything going wrong in the cluster
- When the user mentions a specific alert name - use this tool to get the alert's full labels (namespace, pod, service, etc.) which are essential for further investigation with other tools
- To see currently firing alerts in the cluster
- To check which alerts are active, silenced, or inhibited
- To understand what's happening before diving into metrics or logs

INVESTIGATION TIP: Alert labels often contain the exact identifiers (pod names, namespaces, job names) needed for targeted queries with prometheus tools.

FILTERING:
- Use 'active' to filter for only active alerts (not resolved)
- Use 'silenced' to filter for silenced alerts
- Use 'inhibited' to filter for inhibited alerts
- Use 'filter' to apply label matchers (e.g., "alertname=HighCPU")
- Use 'receiver' to filter alerts by receiver name

All filter parameters are optional. Without filters, all alerts are returned.
  - `active` (`boolean`) - Filter for active alerts only (true/false, optional)
  - `filter` (`string`) - Label matchers to filter alerts (e.g., 'alertname=HighCPU', optional)
  - `inhibited` (`boolean`) - Filter for inhibited alerts only (true/false, optional)
  - `receiver` (`string`) - Receiver name to filter alerts (optional)
  - `silenced` (`boolean`) - Filter for silenced alerts only (true/false, optional)
  - `unprocessed` (`boolean`) - Filter for unprocessed alerts only (true/false, optional)

- **get_silences** - Get silences from Alertmanager.

WHEN TO USE:
- To see which alerts are currently silenced
- To check active, pending, or expired silences
- To investigate why certain alerts are not firing notifications

FILTERING:
- Use 'filter' to apply label matchers to find specific silences

Silences are used to temporarily mute alerts based on label matchers. This tool helps you understand what is currently silenced in your environment.
  - `filter` (`string`) - Label matchers to filter silences (e.g., 'alertname=HighCPU', optional)

</details>

<details>

<summary>oadp</summary>

- **oadp_backup** - Manage Velero/OADP backups: list, get, create, delete, or get status
  - `action` (`string`) **(required)** - Action to perform: 'list' (list all backups), 'get' (get backup details), 'create' (create new backup), 'delete' (delete backup), 'status' (get detailed backup status)
  - `defaultVolumesToFsBackup` (`boolean`) - Use file system backup for volumes instead of snapshots (for create action)
  - `excludedNamespaces` (`array`) - Namespaces to exclude from the backup (for create action)
  - `excludedResources` (`array`) - Resource types to exclude (for create action)
  - `includedNamespaces` (`array`) - Namespaces to include in the backup (for create action)
  - `includedResources` (`array`) - Resource types to include (for create action)
  - `labelSelector` (`string`) - Label selector to filter backups (for list action) or select resources to back up (for create action). For create, only equality-based selectors are supported (e.g., 'app=myapp,env=prod')
  - `name` (`string`) - Name of the backup (required for get, create, delete, status)
  - `namespace` (`string`) - Namespace containing backups (default: openshift-adp)
  - `snapshotVolumes` (`boolean`) - Whether to snapshot persistent volumes (for create action)
  - `storageLocation` (`string`) - BackupStorageLocation name to use (for create action)
  - `ttl` (`string`) - Backup TTL duration e.g., '720h' for 30 days (for create action)
  - `volumeSnapshotLocations` (`array`) - VolumeSnapshotLocation names to use (for create action)

- **oadp_restore** - Manage Velero/OADP restore operations: list, get, create, delete, or get status
  - `action` (`string`) **(required)** - Action to perform: 'list' (list all restores), 'get' (get restore details), 'create' (create new restore), 'delete' (delete restore), 'status' (get detailed restore status)
  - `backupName` (`string`) - Name of the backup to restore from (required for create action)
  - `excludedNamespaces` (`array`) - Namespaces to exclude from restore (for create action)
  - `excludedResources` (`array`) - Resource types to exclude (for create action)
  - `includedNamespaces` (`array`) - Namespaces to restore (for create action)
  - `includedResources` (`array`) - Resource types to restore (for create action)
  - `labelSelector` (`string`) - Label selector to filter restores (for list action)
  - `name` (`string`) - Name of the restore (required for get, create, delete, status)
  - `namespace` (`string`) - Namespace containing restores (default: openshift-adp)
  - `namespaceMapping` (`object`) - Map source namespaces to target namespaces (for create action)
  - `preserveNodePorts` (`boolean`) - Preserve service node ports during restore (for create action)
  - `restorePVs` (`boolean`) - Whether to restore persistent volumes (for create action)

- **oadp_schedule** - Manage Velero/OADP backup schedules: list, get, create, update, delete, or pause (set paused=true to pause, paused=false to unpause)
  - `action` (`string`) **(required)** - Action to perform: 'list', 'get', 'create', 'update', 'delete', or 'pause' (use with 'paused' param: true to pause, false to unpause)
  - `excludedNamespaces` (`array`) - Namespaces to exclude from scheduled backups (for create action)
  - `excludedResources` (`array`) - Resource types to exclude (for create action)
  - `includedNamespaces` (`array`) - Namespaces to include in scheduled backups (for create action)
  - `includedResources` (`array`) - Resource types to include (for create action)
  - `name` (`string`) - Name of the schedule (required for get, create, update, delete, pause)
  - `namespace` (`string`) - Namespace containing schedules (default: openshift-adp)
  - `paused` (`boolean`) - Set to true to pause, false to unpause (for pause action)
  - `schedule` (`string`) - Cron expression e.g., '0 1 * * *' for daily at 1am (for create/update action)
  - `storageLocation` (`string`) - BackupStorageLocation name (for create action)
  - `ttl` (`string`) - Backup TTL duration e.g., '720h' for 30 days (for create/update action)

- **oadp_dpa** - Manage OADP DataProtectionApplication resources: list, get, create, update, or delete
  - `action` (`string`) **(required)** - Action to perform: 'list', 'get', 'create', 'update', or 'delete'
  - `backupLocationBucket` (`string`) - Bucket name for backup storage (for create)
  - `backupLocationCredentialName` (`string`) - Secret name containing backup storage credentials (for create)
  - `backupLocationProvider` (`string`) - Provider for backup storage e.g., aws, azure, gcp (for create)
  - `backupLocationRegion` (`string`) - Region for backup storage (for create)
  - `enableNodeAgent` (`boolean`) - Enable NodeAgent for file-system backups (for create/update)
  - `name` (`string`) - Name of the DPA (required for get, create, update, delete)
  - `namespace` (`string`) - Namespace containing DPAs (default: openshift-adp)
  - `snapshotLocationProvider` (`string`) - Provider for volume snapshots (for create)
  - `snapshotLocationRegion` (`string`) - Region for volume snapshots (for create)

- **oadp_storage_location** - Manage Velero storage locations (BackupStorageLocation and VolumeSnapshotLocation): list, get, create, update, or delete
  - `accessMode` (`string`) - Access mode: ReadWrite or ReadOnly (for BSL update)
  - `action` (`string`) **(required)** - Action to perform: 'list', 'get', 'create', 'update', or 'delete'
  - `bucket` (`string`) - Bucket name for object storage (for BSL create)
  - `credentialSecretKey` (`string`) - Key in the secret containing credentials (default: cloud)
  - `credentialSecretName` (`string`) - Name of the secret containing credentials (for create)
  - `default` (`boolean`) - Set as the default storage location (for BSL create/update)
  - `name` (`string`) - Name of the storage location (required for get, create, update, delete)
  - `namespace` (`string`) - Namespace containing storage locations (default: openshift-adp)
  - `prefix` (`string`) - Optional prefix within the bucket (for BSL create)
  - `provider` (`string`) - Storage provider e.g., aws, azure, gcp (for create)
  - `region` (`string`) - Region for the storage (for create/update)
  - `type` (`string`) **(required)** - Storage location type: 'bsl' (BackupStorageLocation) or 'vsl' (VolumeSnapshotLocation)

- **oadp_data_mover** - Manage Velero data mover resources (DataUpload and DataDownload for CSI snapshots): list, get, or cancel
  - `action` (`string`) **(required)** - Action to perform: 'list', 'get', or 'cancel'
  - `labelSelector` (`string`) - Label selector to filter resources (for list action, e.g., 'app=myapp,env=prod')
  - `name` (`string`) - Name of the resource (required for get, cancel)
  - `namespace` (`string`) - Namespace containing resources (default: openshift-adp)
  - `type` (`string`) **(required)** - Resource type: 'upload' (DataUpload) or 'download' (DataDownload)

- **oadp_repository** - Manage Velero BackupRepository resources (connections to backup storage): list, get, or delete
  - `action` (`string`) **(required)** - Action to perform: 'list', 'get', or 'delete'
  - `name` (`string`) - Name of the repository (required for get, delete)
  - `namespace` (`string`) - Namespace containing repositories (default: openshift-adp)

- **oadp_data_protection_test** - Manage OADP DataProtectionTest resources for validating storage connectivity: list, get, create, or delete
  - `action` (`string`) **(required)** - Action to perform: 'list', 'get', 'create', or 'delete'
  - `backupLocationName` (`string`) - Name of the BackupStorageLocation to test (for create)
  - `name` (`string`) - Name of the test (required for get, create, delete)
  - `namespace` (`string`) - Namespace containing resources (default: openshift-adp)
  - `skipTLSVerify` (`boolean`) - Skip TLS certificate verification (for create)
  - `uploadTestFileSize` (`string`) - Size of test file for upload speed test e.g., '100MB' (for create)
  - `uploadTestTimeout` (`string`) - Timeout for upload test e.g., '60s' (for create)

</details>

<details>

<summary>ossm</summary>

- **ossm_get_mesh_traffic_graph** - Returns service-to-service traffic topology, dependencies, and network metrics (throughput, response time, mTLS) for the specified namespaces. Use this to diagnose routing issues, latency, or find upstream/downstream dependencies.
  - `clusterName` (`string`) - Optional cluster name to include in the graph. Default is the cluster name in the Kiali configuration (KubeConfig).
  - `graphType` (`string`) - Granularity of the graph. 'app' aggregates by app name, 'versionedApp' separates by versions, 'workload' maps specific pods/deployments. Default: versionedApp.
  - `namespaces` (`string`) **(required)** - Comma-separated list of namespaces to map

- **ossm_get_mesh_status** - Retrieves the high-level health, topology, and environment details of the Istio service mesh. Returns multi-cluster control plane status (istiod), data plane namespace health (including ambient mesh status), observability stack health (Prometheus, Grafana...), and component connectivity. Use this tool as the first step to diagnose mesh-wide issues, verify Istio/Kiali versions, or check overall health before drilling into specific workloads.

- **ossm_manage_istio_config_read** - Read-only Istio config: list or get objects. For action 'list', returns an array of objects with {name, namespace, type, validation}. For create, patch, or delete use manage_istio_config.
  - `action` (`string`) **(required)** - Action to perform (read-only)
  - `clusterName` (`string`) - Optional cluster name. Defaults to the cluster name in the Kiali configuration.
  - `group` (`string`) - API group of the Istio object. Required for 'get' action.
  - `kind` (`string`) - Kind of the Istio object. Required for 'get' action.
  - `namespace` (`string`) - Namespace containing the Istio object. For 'list', if not provided, returns objects across all namespaces. For 'get', required.
  - `object` (`string`) - Name of the Istio object. Required for 'get' action.
  - `serviceName` (`string`) - Filter Istio configurations (VirtualServices, DestinationRules, and their referenced Gateways) that affect a specific service. Only applicable for 'list' action
  - `version` (`string`) - API version. Use 'v1' for VirtualService, DestinationRule, and Gateway. Required for 'get' action.

- **ossm_manage_istio_config** - Create, patch, or delete Istio config. For list and get (read-only) use manage_istio_config_read.
  - `action` (`string`) **(required)** - Action to perform (write)
  - `clusterName` (`string`) - Optional cluster name. Defaults to the cluster name in the Kiali configuration.
  - `data` (`string`) - Complete JSON or YAML data to apply or create the object. Required for create and patch actions. You MUST provide a COMPLETE and VALID manifest with ALL required fields for the resource type. Arrays (like servers, http, etc.) are REPLACED entirely, so you must include ALL required fields within each array element.
  - `group` (`string`) **(required)** - API group of the Istio object
  - `kind` (`string`) **(required)** - Kind of the Istio object (e.g., 'VirtualService', 'DestinationRule').
  - `namespace` (`string`) **(required)** - Namespace containing the Istio object
  - `object` (`string`) **(required)** - Name of the Istio object
  - `version` (`string`) **(required)** - API version. Use 'v1' for VirtualService, DestinationRule, and Gateway.

- **ossm_get_resource_details** - Fetches a list of resources OR retrieves detailed data for a specific resource. If 'resourceName' is omitted, it returns a list. If 'resourceName' is provided, it returns details for that specific resource.
  - `clusterName` (`string`) - Optional. Name of the cluster to get resources from. If not provided, will use the default cluster name in the Kiali KubeConfig
  - `namespaces` (`string`) - Comma-separated list of namespaces to query (e.g., 'bookinfo' or 'bookinfo,default'). If not provided, it will query across all accessible namespaces.
  - `resourceName` (`string`) - Optional. The specific name of the resource. If left empty, the tool returns a list of all resources of the specified type. If provided, the tool returns deep details for this specific resource.
  - `resourceType` (`string`) **(required)** - The type of resource to query.

- **ossm_list_traces** - Lists distributed traces for a service in a namespace. Returns a summary (namespace, service, total_found, avg_duration_ms) and a list of traces with id, duration_ms, spans_count, root_op, slowest_service, has_errors. Use get_trace_details with a trace id to get full hierarchy.
  - `clusterName` (`string`) - Optional cluster name. Defaults to the cluster name in the Kiali configuration.
  - `errorOnly` (`boolean`) - If true, only consider traces that contain errors. Default false.
  - `limit` (`integer`) - Maximum number of traces to return. Default 10.
  - `lookbackSeconds` (`integer`) - How far back to search. Default 600 (10m).
  - `namespace` (`string`) **(required)** - Kubernetes namespace of the service.
  - `serviceName` (`string`) **(required)** - Service name to search traces for (required). Returns multiple traces up to limit.

- **ossm_get_trace_details** - Fetches a single distributed trace by trace_id and returns its call hierarchy (service tree with duration, status, and nested calls). Use this after list_traces to drill into a specific trace.
  - `traceId` (`string`) **(required)** - Trace ID to fetch and summarize. If provided, namespace/service_name are ignored.

- **ossm_get_pod_performance** - Returns a human-readable text summary with current Pod CPU/memory usage (from Prometheus) compared to Kubernetes requests/limits (from the Pod spec). Useful to answer questions like 'Is this workload using too much memory?'
  - `clusterName` (`string`) - Optional. Name of the cluster to get resources from. If not provided, will use the default cluster name in the Kiali KubeConfig
  - `namespace` (`string`) **(required)** - Kubernetes namespace of the Pod.
  - `podName` (`string`) - Kubernetes Pod name. If workloadName is provided, the tool will attempt to resolve a Pod from that workload first.
  - `queryTime` (`string`) - Optional end timestamp (RFC3339) for the query. Defaults to now.
  - `timeRange` (`string`) - Time window used to compute CPU rate (Prometheus duration like '5m', '10m', '1h', '1d'). Defaults to '10m'.
  - `workloadName` (`string`) - Kubernetes Workload name (e.g. Deployment/StatefulSet/etc). Tool will look up the workload and pick one of its Pods. If not found, it will fall back to treating this value as a podName.

- **ossm_get_logs** - Get the logs of a Kubernetes Pod (or workload name that will be resolved to a pod) in a namespace. Output is plain text, matching kubernetes-mcp-server pods_log.
  - `clusterName` (`string`) - Optional. Name of the cluster to get the logs from. If not provided, will use the default cluster name in the Kiali KubeConfig
  - `container` (`string`) - Optional. Name of the Pod container to get the logs from.
  - `format` (`string`) - Output formatting for chat. 'codeblock' wraps logs in ~~~ fences (recommended). 'plain' returns raw text like kubernetes-mcp-server pods_log.
  - `name` (`string`) **(required)** - Name of the Pod to get the logs from. If it does not exist, it will be treated as a workload name and a running pod will be selected.
  - `namespace` (`string`) **(required)** - Namespace to get the Pod logs from
  - `previous` (`boolean`) - Optional. Return previous terminated container logs
  - `severity` (`string`) - Optional severity filter applied client-side. Accepts 'ERROR', 'WARN' or combinations like 'ERROR,WARN'.
  - `tail` (`integer`) - Number of lines to retrieve from the end of the logs (Optional, defaults to 50). Cannot exceed 200 lines.
  - `workload` (`string`) - Optional. Workload name override (used when name lookup fails).

- **ossm_get_metrics** - Returns a compact JSON summary of Istio metrics (latency quantiles, traffic trends, throughput, payload sizes) for the given resource.
  - `byLabels` (`string`) - Comma-separated list of labels to group metrics by (e.g., 'source_workload,destination_service'). Optional
  - `clusterName` (`string`) - Cluster name to get metrics from. Optional, defaults to the cluster name in the Kiali configuration (KubeConfig)
  - `direction` (`string`) - Traffic direction. Optional, defaults to 'outbound'
  - `namespace` (`string`) **(required)** - Namespace to get metrics from
  - `quantiles` (`string`) - Comma-separated list of quantiles for histogram metrics (e.g., '0.5,0.95,0.99'). Optional
  - `rateInterval` (`string`) - Rate interval for metrics (e.g., '1m', '5m'). Optional, defaults to '10m'
  - `reporter` (`string`) - Metrics reporter. Optional, defaults to 'source'
  - `requestProtocol` (`string`) - Filter by request protocol (e.g., 'http', 'grpc', 'tcp'). Optional
  - `resourceName` (`string`) **(required)** - Name of the resource to get metrics for
  - `resourceType` (`string`) **(required)** - Type of resource to get metrics
  - `step` (`string`) - Step between data points in seconds (e.g., '15'). Optional, defaults to 15 seconds

</details>

<details>

<summary>tekton</summary>

- **tekton_pipeline_start** - Start a Tekton Pipeline by creating a PipelineRun that references it
  - `name` (`string`) **(required)** - Name of the Pipeline to start
  - `namespace` (`string`) - Namespace of the Pipeline
  - `params` (`object`) - Parameter values to pass to the Pipeline. Keys are parameter names; values can be a string, an array of strings, or an object (map of string to string) depending on the parameter type defined in the Pipeline spec

- **tekton_pipelinerun_restart** - Restart a Tekton PipelineRun by creating a new PipelineRun with the same spec
  - `name` (`string`) **(required)** - Name of the PipelineRun to restart
  - `namespace` (`string`) - Namespace of the PipelineRun

- **tekton_task_start** - Start a Tekton Task by creating a TaskRun that references it
  - `name` (`string`) **(required)** - Name of the Task to start
  - `namespace` (`string`) - Namespace of the Task
  - `params` (`object`) - Parameter values to pass to the Task. Keys are parameter names; values can be a string, an array of strings, or an object (map of string to string) depending on the parameter type defined in the Task spec

- **tekton_taskrun_restart** - Restart a Tekton TaskRun by creating a new TaskRun with the same spec
  - `name` (`string`) **(required)** - Name of the TaskRun to restart
  - `namespace` (`string`) - Namespace of the TaskRun

- **tekton_taskrun_logs** - Get the logs from a Tekton TaskRun by resolving its underlying pod
  - `name` (`string`) **(required)** - Name of the TaskRun to get logs from
  - `namespace` (`string`) - Namespace of the TaskRun
  - `tail` (`integer`) - Number of lines to retrieve from the end of the logs (Optional, default: 100)

</details>


<!-- AVAILABLE-TOOLSETS-TOOLS-END -->

### Prompts

<!-- AVAILABLE-TOOLSETS-PROMPTS-START -->

<details>

<summary>core</summary>

- **cluster-health-check** - Perform comprehensive health assessment of Kubernetes/OpenShift cluster
  - `namespace` (`string`) - Optional namespace to limit health check scope (default: all namespaces)
  - `check_events` (`string`) - Include recent warning/error events (true/false, default: true)

</details>

<details>

<summary>kubevirt</summary>

- **vm-troubleshoot** - Generate a step-by-step troubleshooting guide for diagnosing VirtualMachine issues
  - `namespace` (`string`) **(required)** - The namespace of the VirtualMachine to troubleshoot
  - `name` (`string`) **(required)** - The name of the VirtualMachine to troubleshoot

</details>

<details>

<summary>oadp</summary>

- **oadp-troubleshoot** - Generate a step-by-step troubleshooting guide for diagnosing OADP backup and restore issues
  - `namespace` (`string`) - The OADP namespace (default: openshift-adp)
  - `backup` (`string`) - The name of a specific backup to troubleshoot
  - `restore` (`string`) - The name of a specific restore to troubleshoot

</details>


<!-- AVAILABLE-TOOLSETS-PROMPTS-END -->

## Helm Chart

A [Helm Chart](https://helm.sh) is available to simplify the deployment of the Kubernetes MCP server.

```shell
helm install kubernetes-mcp-server oci://ghcr.io/containers/charts/kubernetes-mcp-server
```

For configuration options including OAuth, telemetry, and resource limits, see the [chart README](./charts/kubernetes-mcp-server/README.md) and [values.yaml](./charts/kubernetes-mcp-server/values.yaml).

## 💬 Community <a id="community"></a>

Join the conversation and connect with other users and contributors:

- [Slack](https://cloud-native.slack.com/archives/C0AHQJVR725) - Ask questions, share feedback, and discuss the Kubernetes MCP server in the `#kubernetes-mcp-server` channel on the CNCF Slack workspace. If you're not already a member, you can [request an invitation](https://slack.cncf.io).

## 🧑‍💻 Development <a id="development"></a>

### Running with mcp-inspector

Compile the project and run the Kubernetes MCP server with [mcp-inspector](https://modelcontextprotocol.io/docs/tools/inspector) to inspect the MCP server.

```shell
# Compile the project
make build
# Run the Kubernetes MCP server with mcp-inspector
npx @modelcontextprotocol/inspector@latest $(pwd)/kubernetes-mcp-server
```

---

mcp-name: io.github.containers/kubernetes-mcp-server
