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
  - **PipelineRun**: Restart, cancel, troubleshoot, and retrieve PipelineRun logs.
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

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest"]
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
| ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `--port`                  | Starts the MCP server in Streamable HTTP mode (path /mcp) and Server-Sent Event (SSE) (path /sse) mode and listens on the specified port .                                                                                                                                                    |
| `--log-level`             | Sets the logging level (values [from 0-9](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md)). Similar to [kubectl logging levels](https://kubernetes.io/docs/reference/kubectl/quick-reference/#kubectl-output-verbosity-and-debugging). |
| `--config`                | (Optional) Path to the main TOML configuration file. See [Configuration Reference](docs/configuration.md) for details.                                                                                                                                                                        |
| `--config-dir`            | (Optional) Path to drop-in configuration directory. Files are loaded in lexical (alphabetical) order. Defaults to `conf.d` relative to the main config file if `--config` is specified. See [Configuration Reference](docs/configuration.md) for details.                                     |
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

### Validated Kubernetes Ecosystem Projects

The following CNCF and Kubernetes ecosystem projects are covered by
automated evaluation scenarios in [`evals/tasks`](evals/tasks). Most scenarios
work with just the `core` toolset. The dedicated toolsets below are optional
and only needed for the project-specific scenarios noted.

<!-- VALIDATED-PROJECTS-START -->

| Project | Optional toolset(s) | Eval scenarios |
|---------|---------------------|----------------|
| [Helm](https://helm.sh) | `helm` | 3 |
| [Istio](https://istio.io) | `kiali` | 5 |
| [Kiali](https://kiali.io) | `kiali` | 16 |
| [Kubernetes](https://kubernetes.io) | - | 32 |
| [KubeVirt](https://kubevirt.io) | `kubevirt`, `tekton` | 19 |
| [NetObserv](https://netobserv.io) | `netobserv` | 4 |
| [Tekton](https://tekton.dev) | `tekton` | 9 |

<!-- VALIDATED-PROJECTS-END -->

### Available Toolsets

The following sets of tools are available (toolsets marked with ✓ in the Default column are enabled by default):

<!-- AVAILABLE-TOOLSETS-START -->

| Toolset               | Description                                                                                                                                                                                                                             | Default |
|-----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|
| cluster-diagnostics   | Tools for cluster diagnostics and troubleshooting                                                                                                                                                                                       |         |
| cni-diagnostics       | Tools for Container Network Interface (CNI) diagnostics and troubleshooting                                                                                                                                                             |         |
| config                | View and manage the current local Kubernetes configuration (kubeconfig)                                                                                                                                                                 | ✓       |
| core                  | Most common tools for Kubernetes management (Pods, Generic Resources, Events, etc.)                                                                                                                                                     | ✓       |
| helm                  | Tools for managing Helm charts and releases                                                                                                                                                                                             |         |
| kcp                   | Manage kcp workspaces and multi-tenancy features                                                                                                                                                                                        |         |
| kubevirt              | OpenShift Virtualization tools for managing virtual machines, check the [OpenShift Virtualization documentation](https://github.com/openshift/openshift-mcp-server/blob/main/docs/kubevirt.md) for more details.                        |         |
| netedge               | NetEdge troubleshooting tools for OpenShift                                                                                                                                                                                             |         |
| netobserv             | Network observability tools backed by the NetObserv console plugin API (flows, metrics, export). Check the [NetObserv documentation](https://github.com/containers/kubernetes-mcp-server/blob/main/docs/NETOBSERV.md) for more details. |         |
| oadp                  | OADP (OpenShift API for Data Protection) tools for managing Velero backups, restores, and schedules                                                                                                                                     |         |
| observability/logs    | Toolset for querying Loki logs                                                                                                                                                                                                          |         |
| observability/metrics | Toolset for querying Prometheus and Alertmanager endpoints in efficient ways.                                                                                                                                                           |         |
| observability/otelcol | Toolset for OpenTelemetry Collector configuration assistance including schema validation, component documentation, and version management.                                                                                              |         |
| observability/traces  | Distributed tracing tools for discovering Tempo instances, searching and retrieving traces, and exploring trace attributes.                                                                                                             |         |
| openshift             | OpenShift-specific tools for cluster management and troubleshooting                                                                                                                                                                     |         |
| openshift/mustgather  | Analyze OpenShift must-gather archives offline without a live cluster connection                                                                                                                                                        |         |
| ossm                  | Most common tools for managing OSSM, check the [OSSM documentation](https://github.com/openshift/openshift-mcp-server/blob/main/docs/OSSM.md) for more details.                                                                         |         |
| ovn-kubernetes        | OVN-Kubernetes CNI network troubleshooting tools                                                                                                                                                                                        |         |
| tekton                | Tekton pipeline management tools for Pipelines, PipelineRuns, Tasks, TaskRuns, and troubleshooting.                                                                                                                                     |         |

<!-- AVAILABLE-TOOLSETS-END -->

### Tools

In case multi-cluster support is enabled (default) and you have access to multiple clusters, all applicable tools will include an additional `context` argument to specify the Kubernetes context (cluster) to use for that operation.

<!-- AVAILABLE-TOOLSETS-TOOLS-START -->

<details>

<summary>cluster-diagnostics</summary>

- **nodes_debug_exec** - Run commands on an OpenShift node using a privileged debug pod with comprehensive troubleshooting utilities. The debug pod uses the UBI9 toolbox image which includes: systemd tools (systemctl, journalctl), networking tools (ss, ip, ping, traceroute, nmap), process tools (ps, top, lsof, strace), file system tools (find, tar, rsync), and debugging tools (gdb). The host filesystem is mounted at /host, allowing commands to chroot /host if needed to access node-level resources.
  - `command` (`array`) **(required)** - Command to execute on the node. All standard debugging utilities from the UBI9 toolbox are available. The host filesystem is mounted at /host - use 'chroot /host <command>' to access node-level resources, or run commands directly in the toolbox environment. Provide each argument as a separate array item (e.g. ['chroot', '/host', 'systemctl', 'status', 'kubelet'] or ['journalctl', '-u', 'kubelet', '--since', '1 hour ago']).
  - `image` (`string`) - Container image to use for the debug pod (optional). Defaults to registry.access.redhat.com/ubi9/toolbox:latest which provides comprehensive debugging and troubleshooting utilities.
  - `namespace` (`string`) - Namespace to create the temporary debug pod in (optional, defaults to the current namespace or 'default').
  - `node` (`string`) **(required)** - Name of the node to debug (e.g. worker-0).
  - `timeout_seconds` (`integer`) - Maximum time to wait for the command to complete before timing out (optional, defaults to 60 seconds).

</details>

<details>

<summary>cni-diagnostics</summary>

- **get-conntrack** - Interact with the connection tracking system on a Kubernetes node. Lists, counts, or shows statistics for tracked connections. Connection tracking shows active network connections and their state (ESTABLISHED, TIME_WAIT, etc.).
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
  - `command` (`string`) - These options specify the particular operation to perform. These options can only be used if configured image has 'conntrack' utility available.
							-L, --dump : List connection tracking table.
							-C, --count: Show the table counter.
							-S, --stats: Show the in-kernel connection tracking system statistics.
  - `filter_parameters` (`string`) - These parameters are useful to filter certain entries from the whole table:
							-s, --src, --orig-src IP_ADDRESS : Match only entries whose source address in the original direction equals to mentioned IP.
							-d, --dst, --orig-dst IP_ADDRESS : Match only entries whose destination address in the original direction equals to mentioned IP.
							-p, --proto PROTO                : Specify layer four (TCP, UDP, ...) protocol.
							--sport, --orig-port-src PORT    : Source port in original direction.
							--dport, --orig-port-dst PORT    : Destination port in original direction.
  - `head` (`integer`) - Return only first N lines. Default: 100 lines if tail is not specified
  - `namespace` (`string`) - Namespace of the debug pod from where conntrack entries are expected to be extracted (optional, defaults to 'default')
  - `node` (`string`) **(required)** - Name of the node from where conntrack entries are expected to be extracted
  - `tail` (`integer`) - Return only last N lines
  - `timeout_seconds` (`integer`) - Timeout in seconds for the command execution. If not specified, server default timeout is used. The maximum value is 300 seconds.

- **get-iptables** - List packet filter rules using iptables or ip6tables on a Kubernetes node. Shows rules for specific tables (filter, nat, mangle, raw, security). Use this to inspect firewall rules, NAT configuration, and packet filtering on nodes.
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
  - `command` (`string`) - These options specify the desired action to perform. Only one of them can be specified on the command line unless otherwise stated below.
							-L, --list [chain]       : List all rules in the selected chain. If no chain is selected, all chains are listed.
							-S, --list-rules [chain] : Print all rules in the selected chain. If no chain is selected, all chains are printed like iptables-save.
  - `filter_parameters` (`string`) - These parameters are useful to filter certain entries from the whole table:
							-s, --source address[/mask]      : Source specification. Address can be either a network name, a hostname, a network IP address (with /mask), or a  plain  IP  address.
							-d, --destination address[/mask] : Destination  specification.
							-v, --verbose					 : Verbose output.
							-n, --numeric                    : Numeric  output.   IP  addresses  and port numbers will be printed in numeric format.
							-p, --protocol protocol          : The protocol of the rule or of the packet to check.
							-4, --ipv4                       : IPv4
							-6, --ipv6                       : IPv6
  - `head` (`integer`) - Return only first N lines. Default: 100 lines if tail is not specified
  - `namespace` (`string`) - Namespace of the debug pod from where packet filter rules are expected to be extracted (optional, defaults to 'default')
  - `node` (`string`) **(required)** - Name of the node from where packet filter rules are expected to be extracted
  - `table` (`string`) - There are currently five independent tables (which tables are present at any time depends on the kernel configuration options and which modules are present).
							filter	: This is the default table
							nat   	: This  table is consulted when a packet that creates a new connection is encountered.
							mangle	: This table is used for specialized packet alteration.
							raw   	: This table is used mainly for configuring exemptions from connection tracking in combination with the NOTRACK target.
							security: This table is used for Mandatory Access Control (MAC) networking rules.
  - `tail` (`integer`) - Return only last N lines
  - `timeout_seconds` (`integer`) - Timeout in seconds for the command execution. If not specified, server default timeout is used. The maximum value is 300 seconds.

- **get-nft** - List nftables packet filtering and classification rules on a Kubernetes node. nftables is the modern replacement for iptables. Use this to inspect firewall rules, packet filtering, and network address translation.
  - `address_families` (`string`) - Address families determine the type of packets which are processed. For each address family, the kernel contains so called hooks at specific stages of
       						   the packet processing paths, which invoke nftables if rules for these hooks exist.
							   - ip       IPv4 address family.
                               - ip6      IPv6 address family.
                               - inet     Internet (IPv4/IPv6) address family.
                               - arp      ARP address family, handling IPv4 ARP packets.
                               - bridge   Bridge address family, handling packets which traverse a bridge device.
                               - netdev   Netdev address family, handling packets on ingress and egress.
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
  - `command` (`string`) **(required)** - These options specify the desired action to perform. Only one of them can be specified on the command line unless otherwise stated below.
                    					- list ruleset   : The ruleset keyword is used to identify the whole set of tables, chains, etc. Print the ruleset in human-readable format.
										- list tables    : List all chains and rules of the specified table.
										- list chains    : List all rules of the specified chain.
										- list sets      : Display the elements in the specified set.
										- list maps      : Display the elements in the specified map.
										- list flowtables: List all flowtables.
  - `head` (`integer`) - Return only first N lines. Default: 100 lines if tail is not specified
  - `namespace` (`string`) - Namespace of the debug pod from where packet filtering and classification rules are expected to be extracted (optional, defaults to 'default')
  - `node` (`string`) **(required)** - Name of the node from where packet filtering and classification rules are expected to be extracted
  - `tail` (`integer`) - Return only last N lines
  - `timeout_seconds` (`integer`) - Timeout in seconds for the command execution. If not specified, server default timeout is used. The maximum value is 300 seconds.

- **get-ip** - Execute ip commands on a Kubernetes node to show routing, network devices, interfaces, and network namespaces. Part of the iproute2 suite for network configuration inspection.
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
  - `command` (`string`) **(required)** - These options specify the desired action to perform. Only one of them can be specified on the command line unless otherwise stated below.
                      - address show     : protocol (IP or IPv6) address on a device.
					  - link show        : network device.
					  - neighbour show   : manage ARP or NDISC cache entries.
					  - netns show       : manage network namespaces.
					  - route show       : routing table entry.
					  - rule show        : rule in routing policy database.
					  - vrf show         : manage virtual routing and forwarding devices. 
					  - xfrm state list  : show Security Association Database.
					  - xfrm policy list : show Security Policy Database.
  - `filter_parameters` (`string`) - This allows to mention sub command to get more filtered data. Available sub command varies and supportability depends on what is 
                          already supported with 'ip' utility.
  - `head` (`integer`) - Return only first N lines. Default: 100 lines if tail is not specified
  - `namespace` (`string`) - Namespace of the debug pod on which ip command is expected to be executed (optional, defaults to 'default')
  - `node` (`string`) **(required)** - Name of the node on which ip command is expected to be executed
  - `options` (`string`) - These options helps in providing more details or formattig output data.
                      					-d, -details        : Output more detailed information.
					  					-4                  : shortcut for -family inet.
					  					-6                  : shortcut for -family inet6.
					  					-r, -resolve        : use the system's name resolver to print DNS names instead of host addresses.
					  					-n, -netns <NETNS>  : switches ip to the specified network namespace NETNS.
					  -a, -all            : executes specified command over all objects, it depends if command supports this option.
  - `tail` (`integer`) - Return only last N lines
  - `timeout_seconds` (`integer`) - Timeout in seconds for the command execution. If not specified, server default timeout is used. The maximum value is 300 seconds.

- **tcpdump** - Capture network packets on a node or inside a pod with BPF filtering. Creates a specialized debug pod for node-level captures. IMPORTANT: Use restrictive BPF filters and low packet counts to avoid performance impact. Maximum 1000 packets.
  - `bpf_filter` (`string`) - BPF filter expression (optional, e.g., 'tcp and dst port 8080', 'host 10.0.0.1')
  - `container_name` (`string`) - Name of the container in the pod when target_type is 'pod' (optional, uses default container if not specified)
  - `interface` (`string`) - Network interface name or 'any' (optional, captures on all interfaces if not specified)
  - `name` (`string`) **(required)** - Name of the target (node or pod)
  - `namespace` (`string`) - Namespace of the target (node or pod). Required when target_type is 'pod'. Optional when target_type is 'node' and defaults to 'default'.
  - `packet_count` (`integer`) - Number of packets to capture (default: 100, max: 1000)
  - `snaplen` (`integer`) - Snapshot length in bytes (default: 96, max: 1500). Use 96 for headers only, 1500 for full packets.
  - `target_type` (`string`) **(required)** - Capture target: 'node' (node-level) or 'pod' (pod network namespace)
  - `timeout_seconds` (`integer`) - Timeout in seconds for the command execution. If not specified, server default timeout is used. The maximum value is 300 seconds.

- **pwru** - Trace packets through the Linux kernel networking stack using eBPF. pwru (packet, where are you?) shows which kernel functions process a packet, helping debug packet drops and routing issues. Creates a specialized debug pod with eBPF capabilities.
  - `bpf_filter` (`string`) - BPF filter expression to match packets (optional, e.g., 'tcp and dst port 8080', 'host 10.0.0.1')
  - `node_name` (`string`) **(required)** - Name of the node to run pwru on
  - `node_pod_namespace` (`string`) - Namespace of the debug pod on which the command is expected to be executed (optional, defaults to 'default')
  - `output_limit_lines` (`integer`) - Maximum number of trace events to capture (default: 100, max: 1000)
  - `timeout_seconds` (`integer`) - Timeout in seconds for the command execution. If not specified, server default timeout is used. The maximum value is 300 seconds.

</details>

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
  - `fieldSelector` (`string`) - Optional Kubernetes field selector to filter events by field values (e.g. 'type=Warning', 'involvedObject.name=my-pod'). Supported fields: involvedObject.kind, involvedObject.name, involvedObject.namespace, involvedObject.uid, involvedObject.apiVersion, involvedObject.resourceVersion, involvedObject.fieldPath, reason, reportingComponent, source, type. See https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/
  - `namespace` (`string`) - Optional Namespace to retrieve the events from. If not provided, will list events from all namespaces

- **namespaces_list** - List all the Kubernetes namespaces in the current cluster
  - `fieldSelector` (`string`) - Optional Kubernetes field selector to filter namespaces by field values (e.g. 'metadata.name=default', 'status.phase=Active'). Supported fields: metadata.name, status.phase. See https://kubernetes.io/docs/concepts/overview/working-with-objects/field-selectors/

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

- **resources_create_or_update** - Create or update a Kubernetes resource via Server-Side Apply. The manifest is the complete desired state: any field this tool previously set and the new manifest omits is removed. To edit an existing resource, fetch it with resources_get, modify it, then re-apply the full resource.
(common apiVersion and kind include: v1 Pod, v1 Service, v1 Node, apps/v1 Deployment, networking.k8s.io/v1 Ingress, route.openshift.io/v1 Route)
  - `resource` (`string`) **(required)** - Complete YAML or JSON representation of the Kubernetes resource (full desired state, not a partial patch). Include apiVersion, kind, metadata, and the full spec.

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

- **vm_clone** - Clone a VirtualMachine on OpenShift Virtualization by creating a VirtualMachineClone resource. This creates a copy of the source VM with a new name using the OpenShift Virtualization Clone API
  - `name` (`string`) **(required)** - The name of the source virtual machine to clone
  - `namespace` (`string`) **(required)** - The namespace of the source virtual machine
  - `targetName` (`string`) **(required)** - The name for the new cloned virtual machine

- **vm_create** - Create a VirtualMachine on OpenShift Virtualization with the specified configuration, automatically resolving instance types, preferences, and container disk images. VM will be created in Halted state by default; use autostart parameter to start it immediately.
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

- **vm_guest_info** - Get guest operating system information from a VirtualMachine's QEMU guest agent. Requires the guest agent to be installed and running inside the VM. Provides detailed information about the OS, filesystems, network interfaces, and logged-in users.
  - `info_type` (`string`) - Type of information to retrieve: 'all' (default - all available info), 'os' (operating system details), 'filesystem' (disk and filesystem info), 'users' (logged-in users), 'network' (network interfaces and IPs)
  - `name` (`string`) **(required)** - The name of the virtual machine
  - `namespace` (`string`) **(required)** - The namespace of the virtual machine

- **vm_lifecycle** - Manage OpenShift Virtualization VirtualMachine lifecycle: start, stop, or restart a VM
  - `action` (`string`) **(required)** - The lifecycle action to perform: 'start' (changes runStrategy to Always), 'stop' (changes runStrategy to Halted), or 'restart' (stops then starts the VM)
  - `name` (`string`) **(required)** - The name of the virtual machine
  - `namespace` (`string`) **(required)** - The namespace of the virtual machine

</details>

<details>

<summary>netedge</summary>

- **netedge_query_prometheus** - Executes specialized diagnostic queries for specific NetEdge components (ingress, dns).
  - `diagnostic_target` (`string`) **(required)** - Run specialized diagnostics for a specific component.

- **get_coredns_config** - Retrieve the current CoreDNS configuration (Corefile) from the cluster.

- **get_service_endpoints** - Return EndpointSlice objects for a Service to verify backend pod availability.
  - `namespace` (`string`) **(required)** - Service namespace
  - `service` (`string`) **(required)** - Service name

- **probe_dns_local** - Run a DNS query using local libraries on the MCP server host to verify connectivity and resolution.
  - `name` (`string`) **(required)** - FQDN to query
  - `server` (`string`) **(required)** - DNS server IP (e.g. 8.8.8.8, 10.0.0.10)
  - `type` (`string`) - Record type (A, AAAA, CNAME, TXT, SRV, etc.). Defaults to A.

- **probe_http** - Send an HTTP(S) request from the MCP server host to verify reachability and inspect the response status code and headers.
  - `method` (`string`) - HTTP method to use. Defaults to GET.
  - `timeout_seconds` (`integer`) - Request timeout in seconds. Defaults to 5.
  - `url` (`string`) **(required)** - The URL to probe (e.g. https://example.com/path).

- **inspect_route** - Inspect an OpenShift Route to view its full configuration and status.
  - `namespace` (`string`) **(required)** - Route namespace
  - `route` (`string`) **(required)** - Route name

- **exec_dns_in_pod** - Spin up a temporary pod in the cluster to execute a DNS lookup using dig, verifying internal cluster networking and DNS path.
  - `namespace` (`string`) **(required)** - Namespace to run the ephemeral pod in.
  - `record_type` (`string`) - DNS record type (A, AAAA, etc.). Defaults to A.
  - `target_name` (`string`) **(required)** - DNS name to query (e.g. kubernetes.default.svc.cluster.local).
  - `target_server` (`string`) **(required)** - DNS server IP to query (e.g. 172.30.0.10).

- **get_router_config** - Retrieve the current router's HAProxy configuration from the cluster. Supports filtering by section type (global/defaults/frontend/backend), substring filter on section headers, and line-count limiting via tail_lines.
  - `filter` (`string`) - Substring filter applied to section headers (e.g. a route or backend name). Only sections whose header contains this string are returned.
  - `pod` (`string`) - Router pod name (optional, chooses any existing if not provided)
  - `section` (`string`) - Filter to a specific HAProxy config section type
  - `tail_lines` (`integer`) - Maximum number of lines to return from the end of the config output (default: 200)

- **get_router_info** - Retrieve HAProxy runtime information from the router.
  - `pod` (`string`) - Router pod name (optional, chooses any existing if not provided)

- **get_router_sessions** - Retrieve active sessions from the router. Supports limiting the number of sessions returned and filtering by substring (e.g. backend name or source IP).
  - `filter` (`string`) - Substring filter applied to each session block. Only sessions containing this string are returned (e.g. a backend name or source IP).
  - `limit` (`integer`) - Maximum number of session blocks to return (default: 50)
  - `pod` (`string`) - Router pod name (optional, chooses any existing if not provided)

</details>

<details>

<summary>netobserv</summary>

- **netobserv_list_flows** - Lists NetObserv network flow records from Loki. Use when investigating traffic between workloads, IPs, ports, or protocols in a namespace or time window.
  - `endTime` (`integer`) - End of time range as Unix epoch seconds. Defaults to now.
  - `filters` (`string`) - NetObserv filter expression passed to the console plugin (plain text; the client URL-encodes it).

Syntax:
- key=value — exact match; key=a,b — OR multiple values for the same key
- key~pattern — regex / contains match; key!~pattern — NOT regex
- key!=value — not equal; key>number — numeric greater-or-equal (e.g. Bytes>1000)
- AND within a group: & (e.g. SrcK8S_Namespace=default&Proto=6)
- OR between groups: | (e.g. SrcK8S_Name=pod-a|SrcK8S_Name=pod-b)

Prefer the dedicated "namespace" parameter for namespace scope when possible.
Use Kubernetes list tools (namespaces, pods, deployments, etc.) to discover filter values.

Common Kubernetes fields (Src/Dst prefixes mirror each other):
- SrcK8S_Namespace, DstK8S_Namespace, SrcK8S_Name, DstK8S_Name
- SrcK8S_Type, DstK8S_Type (e.g. Pod, Service, Node)
- SrcK8S_OwnerName, DstK8S_OwnerName, SrcK8S_OwnerType, DstK8S_OwnerType (for Deployment, StatefulSet, etc.)
- SrcK8S_HostName, DstK8S_HostName, SrcK8S_Zone, DstK8S_Zone, K8S_ClusterName, UDN

Network & flow:
- SrcAddr, DstAddr (IPs), SrcPort, DstPort, Proto (IANA number, e.g. 6=TCP, 17=UDP)
- FlowDirection (0=Ingress, 1=Egress, 2=Inner), Bytes, Packets, Dscp, Flags

Packet drops (often with recordType flowLog and packetLoss dropped/hasDrops):
- PktDropPackets, PktDropBytes, PktDropLatestState, PktDropLatestDropCause

DNS:
- DnsName, DnsId, DnsLatencyMs, DnsErrno, DnsFlagsResponseCode

Examples:
- SrcK8S_Namespace=openshift-netobserv&SrcK8S_Name~my-app
- Proto=6&DstPort=443
- SrcK8S_Name=pod-a|SrcK8S_Name=pod-b
  - `limit` (`integer`) - Maximum number of flow records to return. Default 100.
  - `namespace` (`string`) - Restrict results to flows where source or destination namespace matches (dev-scoped Loki tenant).
  - `packetLoss` (`string`) - Packet loss filter.
  - `recordType` (`string`) - Flow record type filter.
  - `startTime` (`integer`) - Start of time range as Unix epoch seconds. Overrides timeRange when set.
  - `timeRange` (`integer`) - Lookback window in seconds when startTime is omitted. Default 300.

- **netobserv_get_flow_metrics** - Returns aggregated NetObserv flow metrics as topology or time-series data. Use for throughput, TLS/DNS/drop breakdowns, and namespace or workload traffic analysis; see aggregateBy and groups for grouping options.
  - `aggregateBy` (`string`) **(required)** - Primary dimension for netobserv_get_flow_metrics (console plugin /api/flow/metrics).

Two forms (use exact spelling):

1) Topology scopes — aggregate endpoints for graph/topology views:
- app — application workloads (pods/services), excluding infrastructure traffic
- namespace — Kubernetes namespace (default)
- owner — controller owner (Deployment, StatefulSet, …)
- resource — pod, service, or node (finest workload granularity)
- host — node name
- zone — availability zone
- cluster — cluster name (multi-cluster)
- network — user-defined / secondary network name

2) Flow record fields — group by a single flow attribute (PascalCase field name).
Use for breakdown charts (TLS, DNS, drops, protocol). Field names match filters / flow logs.

TLS (requires TLS tracking on the FlowCollector):
- TLSVersion, TLSCipherSuite, TLSGroup, TLSTypes

DNS:
- DnsName, DnsFlagsResponseCode, DnsErrno

Packet drops:
- PktDropLatestState, PktDropLatestDropCause

Network / K8s (single-sided breakdown; pair with filters for src/dst):
- Proto, SrcPort, DstPort, FlowDirection, Dscp
- SrcK8S_Namespace, DstK8S_Namespace, SrcK8S_Name, DstK8S_Name
- SrcK8S_Type, DstK8S_Type, SrcK8S_OwnerName, DstK8S_OwnerName
- SrcK8S_HostName, DstK8S_HostName, SrcK8S_Zone, DstK8S_Zone
- K8S_ClusterName, SrcK8S_NetworkName, DstK8S_NetworkName

Pair aggregateBy with type and function:
- Throughput: type=Bytes or Packets, function=rate
- Flow count: type=Flows, function=count or rate
- DNS volume: type=DnsFlows, function=count
- DNS latency: type=DnsLatencyMs, function=avg, p90, or max
- RTT: type=TimeFlowRttNs, function=avg, min, or p90
- Drops: type=PktDropPackets or PktDropBytes, function=rate

Examples:
- aggregateBy=namespace, type=Bytes, function=rate
- aggregateBy=TLSVersion, type=Bytes, function=rate, filters=TLSTypes!~""
- aggregateBy=TLSGroup, type=Flows, function=count
- aggregateBy=DnsFlagsResponseCode, type=DnsFlows, function=count
- aggregateBy=PktDropLatestState, type=PktDropPackets, function=rate, packetLoss=dropped
- aggregateBy=resource, type=Bytes, function=rate, namespace=netobserv
  - `dataSource` (`string`) - Metrics backend: auto (prefer Prometheus, fallback to Loki), prom, or loki.
  - `endTime` (`integer`) - End of time range as Unix epoch seconds. Defaults to now.
  - `filters` (`string`) - NetObserv filter expression passed to the console plugin (plain text; the client URL-encodes it).

Syntax:
- key=value — exact match; key=a,b — OR multiple values for the same key
- key~pattern — regex / contains match; key!~pattern — NOT regex
- key!=value — not equal; key>number — numeric greater-or-equal (e.g. Bytes>1000)
- AND within a group: & (e.g. SrcK8S_Namespace=default&Proto=6)
- OR between groups: | (e.g. SrcK8S_Name=pod-a|SrcK8S_Name=pod-b)

Prefer the dedicated "namespace" parameter for namespace scope when possible.
Use Kubernetes list tools (namespaces, pods, deployments, etc.) to discover filter values.

Common Kubernetes fields (Src/Dst prefixes mirror each other):
- SrcK8S_Namespace, DstK8S_Namespace, SrcK8S_Name, DstK8S_Name
- SrcK8S_Type, DstK8S_Type (e.g. Pod, Service, Node)
- SrcK8S_OwnerName, DstK8S_OwnerName, SrcK8S_OwnerType, DstK8S_OwnerType (for Deployment, StatefulSet, etc.)
- SrcK8S_HostName, DstK8S_HostName, SrcK8S_Zone, DstK8S_Zone, K8S_ClusterName, UDN

Network & flow:
- SrcAddr, DstAddr (IPs), SrcPort, DstPort, Proto (IANA number, e.g. 6=TCP, 17=UDP)
- FlowDirection (0=Ingress, 1=Egress, 2=Inner), Bytes, Packets, Dscp, Flags

Packet drops (often with recordType flowLog and packetLoss dropped/hasDrops):
- PktDropPackets, PktDropBytes, PktDropLatestState, PktDropLatestDropCause

DNS:
- DnsName, DnsId, DnsLatencyMs, DnsErrno, DnsFlagsResponseCode

Examples:
- SrcK8S_Namespace=openshift-netobserv&SrcK8S_Name~my-app
- Proto=6&DstPort=443
- SrcK8S_Name=pod-a|SrcK8S_Name=pod-b
  - `function` (`string`) - Aggregation function.
  - `groups` (`string`) - Optional comma-separated parent scopes when aggregateBy is a topology scope.
Adds extra label dimensions (e.g. break namespace results down by cluster or zone).
Ignored or less useful when aggregateBy is already a raw flow field (e.g. TLSVersion); use filters instead.

Single scopes:
- clusters, networks, zones, hosts, namespaces, owners

Combined scopes (use +, no spaces):
- clusters+zones, clusters+hosts, clusters+namespaces, clusters+owners
- zones+hosts, zones+namespaces, zones+owners
- hosts+namespaces, hosts+owners
- namespaces+owners
- networks+zones, networks+hosts, networks+namespaces, networks+owners

Examples:
- aggregateBy=namespace, groups=clusters
- aggregateBy=resource, groups=namespaces
- aggregateBy=owner, groups=zones,hosts
  - `limit` (`integer`) - Maximum number of flow records to return. Default 100.
  - `namespace` (`string`) - Restrict results to flows where source or destination namespace matches (dev-scoped Loki tenant).
  - `packetLoss` (`string`) - Packet loss filter.
  - `rateInterval` (`string`) - Prometheus rate interval (e.g. 1m, 5m).
  - `recordType` (`string`) - Flow record type filter.
  - `startTime` (`integer`) - Start of time range as Unix epoch seconds. Overrides timeRange when set.
  - `step` (`string`) - Query resolution step (e.g. 30s, 1m).
  - `timeRange` (`integer`) - Lookback window in seconds when startTime is omitted. Default 300.
  - `type` (`string`) - Metric type to aggregate.

- **netobserv_export_flows** - Exports NetObserv flow records as CSV with the same filters as list_flows. Use when the user needs downloadable flow data for audits or offline analysis.
  - `columns` (`string`) - Optional comma-separated column names to include (e.g. SrcK8S_Namespace,DstK8S_Namespace,Bytes). Omit to export all columns present in the result.
  - `endTime` (`integer`) - End of time range as Unix epoch seconds. Defaults to now.
  - `filters` (`string`) - NetObserv filter expression passed to the console plugin (plain text; the client URL-encodes it).

Syntax:
- key=value — exact match; key=a,b — OR multiple values for the same key
- key~pattern — regex / contains match; key!~pattern — NOT regex
- key!=value — not equal; key>number — numeric greater-or-equal (e.g. Bytes>1000)
- AND within a group: & (e.g. SrcK8S_Namespace=default&Proto=6)
- OR between groups: | (e.g. SrcK8S_Name=pod-a|SrcK8S_Name=pod-b)

Prefer the dedicated "namespace" parameter for namespace scope when possible.
Use Kubernetes list tools (namespaces, pods, deployments, etc.) to discover filter values.

Common Kubernetes fields (Src/Dst prefixes mirror each other):
- SrcK8S_Namespace, DstK8S_Namespace, SrcK8S_Name, DstK8S_Name
- SrcK8S_Type, DstK8S_Type (e.g. Pod, Service, Node)
- SrcK8S_OwnerName, DstK8S_OwnerName, SrcK8S_OwnerType, DstK8S_OwnerType (for Deployment, StatefulSet, etc.)
- SrcK8S_HostName, DstK8S_HostName, SrcK8S_Zone, DstK8S_Zone, K8S_ClusterName, UDN

Network & flow:
- SrcAddr, DstAddr (IPs), SrcPort, DstPort, Proto (IANA number, e.g. 6=TCP, 17=UDP)
- FlowDirection (0=Ingress, 1=Egress, 2=Inner), Bytes, Packets, Dscp, Flags

Packet drops (often with recordType flowLog and packetLoss dropped/hasDrops):
- PktDropPackets, PktDropBytes, PktDropLatestState, PktDropLatestDropCause

DNS:
- DnsName, DnsId, DnsLatencyMs, DnsErrno, DnsFlagsResponseCode

Examples:
- SrcK8S_Namespace=openshift-netobserv&SrcK8S_Name~my-app
- Proto=6&DstPort=443
- SrcK8S_Name=pod-a|SrcK8S_Name=pod-b
  - `format` (`string`) - Export format. Only csv is supported.
  - `limit` (`integer`) - Maximum number of flow records to return. Default 100.
  - `namespace` (`string`) - Restrict results to flows where source or destination namespace matches (dev-scoped Loki tenant).
  - `packetLoss` (`string`) - Packet loss filter.
  - `recordType` (`string`) - Flow record type filter.
  - `startTime` (`integer`) - Start of time range as Unix epoch seconds. Overrides timeRange when set.
  - `timeRange` (`integer`) - Lookback window in seconds when startTime is omitted. Default 300.

</details>

<details>

<summary>oadp</summary>

</details>

<details>

<summary>observability/logs</summary>

- **loki_list_instances** - List LokiStack instances available in the Kubernetes cluster.
Call this first when using Loki Operator managed stacks so you can pass lokiNamespace and lokiName to other Loki tools.

- **loki_label_names** - List available Loki label names for a time range. Use this before writing LogQL queries.
  - `end` (`string`) - End time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional).
  - `lokiName` (`string`) - Name of the LokiStack. Use loki_list_instances to discover valid values.
  - `lokiNamespace` (`string`) - Kubernetes namespace of the LokiStack. Use loki_list_instances to discover valid values.
  - `start` (`string`) - Start time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional).
  - `tenant` (`string`) - Loki tenant ID (X-Scope-OrgID). For LokiStack gateway modes (e.g. openshift-network) this selects the `/api/logs/v1/<tenant>` path; use `network` for openshift-network.

- **loki_label_values** - List possible values for a Loki label key. Use this to build precise label matchers in LogQL.
  - `end` (`string`) - End time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional).
  - `label` (`string`) **(required)** - Label key to inspect (for example namespace, pod, container).
  - `lokiName` (`string`) - Name of the LokiStack. Use loki_list_instances to discover valid values.
  - `lokiNamespace` (`string`) - Kubernetes namespace of the LokiStack. Use loki_list_instances to discover valid values.
  - `start` (`string`) - Start time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional).
  - `tenant` (`string`) - Loki tenant ID (X-Scope-OrgID). For LokiStack gateway modes (e.g. openshift-network) this selects the `/api/logs/v1/<tenant>` path; use `network` for openshift-network.

- **loki_query_range** - Execute a Loki LogQL range query and return matching log streams and lines.

Use precise label matchers and a short time window first.
  - `direction` (`string`) - Search direction: backward (default) or forward.
  - `duration` (`string`) - Lookback duration from now when start/end are omitted (for example 5m, 1h). Defaults to 15m.
  - `end` (`string`) - End time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional).
  - `limit` (`integer`) - Maximum number of log lines to return. Defaults to 100, max 1000.
  - `lokiName` (`string`) - Name of the LokiStack. Use loki_list_instances to discover valid values.
  - `lokiNamespace` (`string`) - Kubernetes namespace of the LokiStack. Use loki_list_instances to discover valid values.
  - `query` (`string`) **(required)** - LogQL query string.
  - `start` (`string`) - Start time as RFC3339, Unix timestamp, NOW, or NOW-relative expression (optional).
  - `tenant` (`string`) - Loki tenant ID (X-Scope-OrgID). For LokiStack gateway modes (e.g. openshift-network) this selects the `/api/logs/v1/<tenant>` path; use `network` for openshift-network.

</details>

<details>

<summary>observability/metrics</summary>

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

<summary>observability/otelcol</summary>

- **otelcol_list_components** - List available OpenTelemetry Collector components (receivers, processors, exporters, extensions, connectors) for a given version.
  - `version` (`string`) - Collector version (e.g., 'v0.100.0'). Defaults to latest available.

- **otelcol_get_component_schema** - Get the JSON schema for an OpenTelemetry Collector component's configuration options.
  - `component_name` (`string`) **(required)** - Component name from otelcol_list_components (e.g., 'otlp', 'batch', 'debug')
  - `component_type` (`string`) **(required)** - Component type: receiver, processor, exporter, extension, connector
  - `version` (`string`) - Collector version (e.g., 'v0.100.0'). Defaults to latest available.

- **otelcol_validate_config** - Validate an OpenTelemetry Collector component configuration against its JSON schema.
  - `component_name` (`string`) **(required)** - Component name from otelcol_list_components (e.g., 'otlp', 'batch', 'debug')
  - `component_type` (`string`) **(required)** - Component type: receiver, processor, exporter, extension, connector
  - `config` (`string`) **(required)** - Configuration to validate as YAML or JSON string
  - `format` (`string`) - Config format: 'yaml' (default) or 'json'
  - `version` (`string`) - Collector version (e.g., 'v0.100.0'). Defaults to latest available.

- **otelcol_get_versions** - List available OpenTelemetry Collector versions and identify the latest.

</details>

<details>

<summary>observability/traces</summary>

- **tempo_list_instances** - List all Tempo instances available in the Kubernetes cluster.
Call this tool first to discover available Tempo instances before using other Tempo tools,
as the returned namespace, name, and tenant values are required parameters for all other Tempo tools.
Always print the output of this tool in a table.

- **tempo_get_trace_by_id** - Retrieve a single distributed trace by its trace ID from Tempo.
Returns the full trace with all its spans, including service names, operation names, durations, and attributes.
Use this tool when you already have a specific trace ID, e.g. from search results or logs.
  - `end` (`string`) - Optional end of the time range in RFC 3339 format, e.g. "2025-01-02T00:00:00Z".
Narrows the time range to improve query performance.
  - `start` (`string`) - Optional start of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".
Narrows the time range to improve query performance.
  - `tempoName` (`string`) **(required)** - The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names.
  - `tempoNamespace` (`string`) **(required)** - The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces.
  - `tenant` (`string`) - The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance.
  - `traceid` (`string`) **(required)** - The trace ID to retrieve, e.g. "26dad4a0e2b0dd9a440dd5ff203a24a4".

- **tempo_search_traces** - Search for distributed traces in Tempo using TraceQL.
Use this tool to find traces matching specific criteria such as service name, HTTP status code, duration, or other span or resource attributes.

IMPORTANT — "slow" or "long" trace requests: Do NOT guess a duration threshold.
First call this tool WITHOUT a duration filter to establish a latency baseline, then use that baseline to set a sensible threshold.
Both steps are required — do NOT skip the second search with the duration filter.
Skip this two-step process only when the user provides an explicit duration (e.g. "find traces slower than 2s").

  - `end` (`string`) - End of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".
Use "NOW" for current time.
Both start and end should be provided to search the full time range; if omitted, only a small window of recent data is searched.
  - `limit` (`integer`) - Maximum number of traces to return. Defaults to the server-side limit if not specified.
  - `query` (`string`) **(required)** - A TraceQL query expression. Format:
query: "{ <filters joined by &&> }"

Filters:
- service name:     resource.service.name="<value>" (string, use quotes)
- HTTP status code: span.http.response.status_code=<code> (number, no quotes)
- duration:         duration><value like 100ms, 2s, 5m> (no quotes)
- error status:     status=error (keyword, NO quotes — do NOT write status="error")

IMPORTANT: status values (error, ok, unset) are keywords, NOT strings. Write status=error, NEVER status="error".

Operators: =, !=, >, <, >=, <=

Common attributes:
- resource.service.name (service name)
- resource.k8s.namespace.name (Kubernetes namespace)
- resource.k8s.deployment.name (Kubernetes deployment)
- resource.k8s.statefulset.name (Kubernetes statefulset)
- resource.k8s.daemonset.name (Kubernetes daemonset)
- resource.k8s.replicaset.name (Kubernetes replicaset)
- resource.k8s.pod.name (Kubernetes pod)
- resource.k8s.container.name (Kubernetes container)
- resource.k8s.job.name (Kubernetes job)
- resource.k8s.cronjob.name (Kubernetes cronjob)
- resource.k8s.node.name (Kubernetes node)
- resource.k8s.cluster.name (Kubernetes cluster)
- span.http.response.status_code (HTTP response code)
- span.http.request.method (HTTP method like GET, POST)
- span.url.full (request URL)
- name (span name / operation name, e.g. "GET /api/users")
- duration (trace duration, e.g. 100ms, 2s)
- status (trace status: ok, error, unset)

Note: older instrumentation may use legacy HTTP attribute names (e.g. span.http.status_code instead of span.http.response.status_code).
If a query returns no results, try tempo_search_tags to check which attributes exist.

IMPORTANT:
- Always wrap filters in curly braces { }.
- Do NOT use SQL, PromQL, or Lucene syntax.
- Do NOT omit the "resource." or "span." prefix from attribute names
- When the user refers to a Kubernetes resource type (deployment, pod, namespace, etc.), use the matching resource.k8s.* attribute, NOT resource.service.name.

Examples:
- { resource.service.name="frontend" }
- { resource.k8s.deployment.name="checkout" && span.http.response.status_code>=500 }
- { status=error && duration>2s }

If unsure which attributes to filter on, use tempo_search_tags to discover available attributes before building a query.

  - `spss` (`integer`) - Maximum number of matching spans to return per trace.
  - `start` (`string`) - Start of the time range in RFC 3339 format, e.g. "2025-01-01T00:00:00Z".
Use "NOW" for current time.
Both start and end should be provided to search the full time range; if omitted, only a small window of recent data is searched.
  - `tempoName` (`string`) **(required)** - The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names.
  - `tempoNamespace` (`string`) **(required)** - The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces.
  - `tenant` (`string`) - The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance.

- **tempo_search_tags** - List available tag names (attribute keys) in Tempo, grouped by scope.
Use this tool to discover which attributes are available for building TraceQL queries with tempo_search_traces.
For example, this tool may reveal tag names like "service.name" (in the "resource" scope) or "http.response.status_code" (in the "span" scope).
To use these in TraceQL queries, prefix them with their scope, e.g. "resource.service.name" or "span.http.response.status_code".
  - `end` (`string`) - Optional end of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing tags.
  - `limit` (`integer`) - Maximum number of tag names to return per scope.
  - `maxStaleValues` (`integer`) - Maximum number of consecutive blocks without new tag names before the search stops early. Higher values are more thorough but slower.
  - `query` (`string`) - Optional TraceQL query to filter which traces are considered when listing tags,
e.g. '{ resource.service.name="payment-service" }' to only show tags present in traces from the 'payment-service' service.
  - `scope` (`string`) - Filter tags to a specific scope. One of:
"resource" (service-level attributes like service.name),
"span" (individual span attributes like http.response.status_code),
"intrinsic" (built-in fields like duration, status, name).
If omitted, tags from all scopes are returned.
  - `start` (`string`) - Optional start of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing tags.
  - `tempoName` (`string`) **(required)** - The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names.
  - `tempoNamespace` (`string`) **(required)** - The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces.
  - `tenant` (`string`) - The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance.

- **tempo_search_tag_values** - List the known values for a specific tag (attribute key) in Tempo.
Use this tool to discover what values exist for a given tag, e.g. to find all service names (values of "resource.service.name") or all HTTP methods (values of "span.http.request.method").
This is useful for building accurate TraceQL queries with tempo_search_traces.
  - `end` (`string`) - Optional end of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing values.
  - `limit` (`integer`) - Maximum number of tag values to return.
  - `maxStaleValues` (`integer`) - Maximum number of consecutive blocks without new values before the search stops early. Higher values are more thorough but slower.
  - `query` (`string`) - Optional TraceQL query to filter which traces are considered when listing values,
e.g. '{ resource.service.name="payment-service" }' to only show tag values from the 'payment-service' service.
  - `start` (`string`) - Optional start of the time range (in RFC 3339 format, e.g. "2025-01-01T00:00:00Z") to filter which traces are considered when listing values.
  - `tag` (`string`) **(required)** - The fully qualified tag name to get values for, including its scope prefix, e.g. "resource.service.name" or "span.http.response.status_code".
Use tempo_search_tags to discover available tag names.
  - `tempoName` (`string`) **(required)** - The name of the Tempo instance to query. Use tempo_list_instances to discover available instance names.
  - `tempoNamespace` (`string`) **(required)** - The Kubernetes namespace where the Tempo instance is deployed. Use tempo_list_instances to discover available namespaces.
  - `tenant` (`string`) - The tenant to query. This parameter is required for multi-tenant instances. Use tempo_list_instances to discover available tenants for each instance.

</details>

<details>

<summary>openshift</summary>

</details>

<details>

<summary>openshift/mustgather</summary>

- **mustgather_use** - Load a must-gather archive from a given filesystem path for analysis. Must be called before any other mustgather_* tools.
  - `path` (`string`) **(required)** - Absolute path to the must-gather archive directory

- **mustgather_resources_list** - List Kubernetes resources from the must-gather archive with optional filtering by namespace, labels, and fields
  - `apiVersion` (`string`) - API version (default: v1)
  - `fieldSelector` (`string`) - Field selector (e.g., metadata.name=foo)
  - `kind` (`string`) **(required)** - Resource kind (e.g., Pod, Deployment, Service)
  - `labelSelector` (`string`) - Label selector (e.g., app=nginx,tier=frontend)
  - `limit` (`integer`) - Maximum number of resources to return (0 for all)
  - `namespace` (`string`) - Filter by namespace

- **mustgather_events_list** - List Kubernetes events from the must-gather archive with optional filtering by type, namespace, resource, and reason
  - `limit` (`integer`) - Maximum number of events to return (default: 100)
  - `namespace` (`string`) - Filter by namespace
  - `reason` (`string`) - Filter by event reason (partial match)
  - `resource` (`string`) - Filter by involved resource name (partial match)
  - `type` (`string`) - Event type filter: all, Warning, Normal

- **mustgather_events_by_resource** - Get all events related to a specific Kubernetes resource from the must-gather archive
  - `kind` (`string`) - Resource kind (optional, narrows search)
  - `name` (`string`) **(required)** - Resource name
  - `namespace` (`string`) - Resource namespace

- **mustgather_events_by_time** - List Kubernetes events from the must-gather archive within a specific time range, sorted chronologically
  - `limit` (`integer`) - Maximum number of events to return (default: 200)
  - `namespace` (`string`) - Filter by namespace
  - `since` (`string`) **(required)** - Start time in RFC3339 format (e.g. 2026-01-15T10:00:00Z)
  - `type` (`string`) - Event type filter: all, Warning, Normal
  - `until` (`string`) - End time in RFC3339 format (e.g. 2026-01-15T12:00:00Z)

- **mustgather_pod_logs_get** - Get container logs for a specific pod from the must-gather archive. Returns current or previous logs.
  - `container` (`string`) - Container name (uses first container if not specified)
  - `namespace` (`string`) **(required)** - Pod namespace
  - `pod` (`string`) **(required)** - Pod name
  - `previous` (`boolean`) - Get previous container logs (from crash/restart)
  - `tail` (`integer`) - Number of lines from end of logs (0 for all)

- **mustgather_pod_logs_grep** - Filter pod container logs by a search string. Returns only matching lines from the must-gather archive.
  - `caseInsensitive` (`boolean`) - Perform case-insensitive search (default: false)
  - `container` (`string`) - Container name (uses first container if not specified)
  - `filter` (`string`) **(required)** - String to search for in log lines
  - `namespace` (`string`) **(required)** - Pod namespace
  - `pod` (`string`) **(required)** - Pod name
  - `previous` (`boolean`) - Search previous container logs (from crash/restart)
  - `tail` (`integer`) - Maximum number of matching lines to return (0 for all)

- **mustgather_pod_logs_by_time** - Get pod container logs within a specific time range. Each log line is expected to have an RFC3339Nano timestamp prefix (from kubectl logs --timestamps).
  - `container` (`string`) - Container name (uses first container if not specified)
  - `limit` (`integer`) - Maximum number of lines to return (default: 500)
  - `namespace` (`string`) **(required)** - Pod namespace
  - `pod` (`string`) **(required)** - Pod name
  - `previous` (`boolean`) - Search previous container logs (from crash/restart)
  - `since` (`string`) **(required)** - Start time in RFC3339 format (e.g. 2026-01-15T10:00:00Z)
  - `until` (`string`) - End time in RFC3339 format (e.g. 2026-01-15T12:00:00Z)

- **mustgather_node_diagnostics_get** - Get comprehensive diagnostic information for a specific node including kubelet logs, system info, CPU/IRQ affinities, and hardware details
  - `include` (`string`) - Comma-separated diagnostics to include: kubelet,sysinfo,cpu,irq,pods,podresources,lscpu,lspci,dmesg,cmdline (default: all)
  - `kubeletTail` (`integer`) - Number of lines from end of kubelet log (0 for all, default: 100)
  - `node` (`string`) **(required)** - Node name

- **mustgather_node_kubelet_logs** - Get kubelet logs for a specific node (decompressed from .gz file)
  - `node` (`string`) **(required)** - Node name
  - `tail` (`integer`) - Number of lines from end (0 for all)

- **mustgather_node_kubelet_logs_grep** - Filter kubelet logs for a specific node by a search string. Returns only matching lines.
  - `caseInsensitive` (`boolean`) - Perform case-insensitive search (default: false)
  - `filter` (`string`) **(required)** - String to search for in log lines
  - `node` (`string`) **(required)** - Node name
  - `tail` (`integer`) - Maximum number of matching lines to return (0 for all)

- **mustgather_etcd_health** - Get ETCD cluster health status including endpoint health and active alarms from the must-gather archive

- **mustgather_etcd_object_count** - Get ETCD object counts by resource type from the must-gather archive
  - `limit` (`integer`) - Maximum number of resource types to show (default: 50, sorted by count descending)

- **mustgather_monitoring_prometheus_status** - Get Prometheus TSDB and runtime status from the must-gather archive
  - `replica` (`string`) - Prometheus replica (0, 1, or all). Default: all

- **mustgather_monitoring_prometheus_targets** - Get Prometheus scrape targets and their health status from the must-gather archive
  - `health` (`string`) - Filter by health status: up, down, unknown (default: all)
  - `replica` (`string`) - Prometheus replica (0, 1, or all). Default: 0

- **mustgather_monitoring_prometheus_tsdb** - Get detailed Prometheus TSDB statistics including top metrics by series count and label cardinality
  - `limit` (`integer`) - Number of top entries to show per category (default: 10)
  - `replica` (`string`) - Prometheus replica (0, 1, or all). Default: 0

- **mustgather_monitoring_prometheus_alerts** - Get active Prometheus alerts from the must-gather archive
  - `state` (`string`) - Filter by alert state: firing, pending (default: all)

- **mustgather_monitoring_prometheus_rules** - Get Prometheus alerting and recording rules from the must-gather archive
  - `type` (`string`) - Filter by rule type: alerting, recording (default: all)

</details>

<details>

<summary>ossm</summary>

- **ossm_get_mesh_traffic_graph** - Returns service-to-service traffic topology, dependencies, and network metrics (throughput, response time, mTLS) for the specified namespaces. Use this to diagnose routing issues, latency, or find upstream/downstream dependencies.
  - `graphType` (`string`) - Granularity of the graph. 'app' aggregates by app name, 'versionedApp' separates by versions, 'workload' maps specific pods/deployments. Default: versionedApp.
  - `meshCluster` (`string`) - Optional Istio mesh cluster name from ossm_list_mesh_clusters (e.g. west). When omitted, Kiali defaults to its home cluster.
  - `namespaces` (`string`) **(required)** - Comma-separated list of namespaces to map

- **ossm_get_mesh_status** - Retrieves the high-level health, topology, and environment details of the Istio service mesh. Returns multi-cluster control plane status (istiod), data plane namespace health (including ambient mesh status), observability stack health (Prometheus, Grafana...), and component connectivity. Use this tool as the first step to diagnose mesh-wide issues, verify Istio/Kiali versions, or check overall health before drilling into specific workloads.

- **ossm_manage_istio_config_read** - Read Istio, Gateway API, and Inference API config. 'list' groups by namespace→'group/version/kind'→{valid:[...],invalid:[...]} where valid/invalid arrays contain resource names; omit group/kind to retrieve ALL config types in a single call. Supports Istio (networking.istio.io, security.istio.io), Gateway API (gateway.networking.k8s.io), and Inference API (inference.networking.k8s.io) when installed. 'get' returns full YAML. For writes use manage_istio_config.
  - `action` (`string`) **(required)** - Action to perform (read-only)
  - `group` (`string`) - API group of the Istio object. Required ONLY for 'get' action. For 'list', OMIT group and kind to retrieve ALL config types in a single call. Use 'gateway.networking.k8s.io' for Gateway API resources. Use 'inference.networking.k8s.io' for Inference API resources.
  - `kind` (`string`) - Kind of the Istio object. Required ONLY for 'get' action. For 'list', OMIT to return all kinds at once — do NOT call separately for each kind.
  - `meshCluster` (`string`) - Optional Istio mesh cluster name from ossm_list_mesh_clusters (e.g. west). When omitted, Kiali defaults to its home cluster.
  - `namespace` (`string`) - Namespace containing the Istio object. For 'list', if not provided, returns objects across all namespaces. For 'get', required.
  - `object` (`string`) - Name of the Istio object. Required for 'get' action.
  - `serviceName` (`string`) - Filter Istio configurations (VirtualServices, DestinationRules, and their referenced Gateways) that affect a specific service. Only applicable for 'list' action
  - `version` (`string`) - API version. Use 'v1' for all resource types. Required for 'get' action.

- **ossm_manage_istio_config** - Create, patch, or delete Istio, Gateway API, and Inference API config. Supports Istio resources (networking.istio.io, security.istio.io), Gateway API resources (gateway.networking.k8s.io), and Inference API resources (inference.networking.k8s.io) when installed on the cluster. For list and get (read-only) use manage_istio_config_read.
  - `action` (`string`) **(required)** - Action to perform (write)
  - `data` (`string`) - JSON or YAML data for the resource. Required for create and patch actions. For create, you can provide partial content (e.g. only spec) and it will be merged onto a valid template with defaults. Arrays (like servers, http, etc.) are REPLACED entirely, so include ALL elements you want.
  - `group` (`string`) **(required)** - API group of the Istio object. Use 'gateway.networking.k8s.io' for Gateway API resources. Use 'inference.networking.k8s.io' for Inference API resources.
  - `kind` (`string`) **(required)** - Kind of the Istio object (e.g., 'VirtualService', 'DestinationRule').
  - `meshCluster` (`string`) - Optional Istio mesh cluster name from ossm_list_mesh_clusters (e.g. west). When omitted, Kiali defaults to its home cluster.
  - `namespace` (`string`) **(required)** - Namespace containing the Istio object.
  - `object` (`string`) **(required)** - Name of the Istio object.
  - `version` (`string`) **(required)** - API version. Use 'v1' for all resource types.

- **ossm_list_mesh_clusters** - Returns the list of Istio mesh clusters that Kiali can access. Each entry includes its name and whether it is the home cluster (where Kiali is deployed). Call this tool before using meshCluster on other Kiali tools when the target cluster is unknown.

- **ossm_get_resource_details** - Fetches a list of resources OR retrieves detailed data for a specific resource. If 'resourceName' is omitted, it returns a list. If 'resourceName' is provided, it returns details for that specific resource.
  - `meshCluster` (`string`) - Optional Istio mesh cluster name from ossm_list_mesh_clusters (e.g. west). When omitted, Kiali defaults to its home cluster.
  - `namespaces` (`string`) - Comma-separated list of namespaces to query (e.g., 'bookinfo' or 'bookinfo,default'). If not provided, it will query across all accessible namespaces.
  - `resourceName` (`string`) - Optional. The specific name of the resource. If left empty, the tool returns a list of all resources of the specified type. If provided, the tool returns deep details for this specific resource.
  - `resourceType` (`string`) **(required)** - The type of resource to query. Use 'app' for Kiali applications (grouped by the Kubernetes 'app' label). Use 'argoapp' for ArgoCD Application CRDs (requires ArgoCD installed and the Kiali service account must have read permissions on applications.argoproj.io).

- **ossm_list_traces** - Lists distributed traces for a service in a namespace. Returns a summary (namespace, service, total_found, avg_duration_ms) and a list of traces with id, duration_ms, spans_count, root_op, slowest_service, has_errors. Use get_trace_details with a trace id to get full hierarchy.
  - `errorOnly` (`boolean`) - If true, only consider traces that contain errors. Default false.
  - `limit` (`integer`) - Maximum number of traces to return. Default 10.
  - `lookbackSeconds` (`integer`) - How far back to search. Default 600 (10m).
  - `meshCluster` (`string`) - Optional Istio mesh cluster name from ossm_list_mesh_clusters (e.g. west). When omitted, Kiali defaults to its home cluster.
  - `namespace` (`string`) **(required)** - Kubernetes namespace of the service.
  - `serviceName` (`string`) **(required)** - Service name to search traces for (required). Returns multiple traces up to limit.

- **ossm_get_trace_details** - Fetches a single distributed trace by trace_id and returns its call hierarchy (service tree with duration, status, and nested calls). Use this after list_traces to drill into a specific trace.
  - `traceId` (`string`) **(required)** - Trace ID to fetch and summarize. If provided, namespace/service_name are ignored.

- **ossm_get_pod_performance** - Returns a human-readable text summary with current Pod CPU/memory usage (from Prometheus) compared to Kubernetes requests/limits (from the Pod spec). Useful to answer questions like 'Is this workload using too much memory?'
  - `meshCluster` (`string`) - Optional Istio mesh cluster name from ossm_list_mesh_clusters (e.g. west). When omitted, Kiali defaults to its home cluster.
  - `namespace` (`string`) **(required)** - Kubernetes namespace of the Pod.
  - `podName` (`string`) - Kubernetes Pod name. If workloadName is provided, the tool will attempt to resolve a Pod from that workload first.
  - `queryTime` (`string`) - Optional end timestamp (RFC3339) for the query. Defaults to now.
  - `timeRange` (`string`) - Time window used to compute CPU rate (Prometheus duration like '5m', '10m', '1h', '1d'). Defaults to '10m'.
  - `workloadName` (`string`) - Kubernetes Workload name (e.g. Deployment/StatefulSet/etc). Tool will look up the workload and pick one of its Pods. If not found, it will fall back to treating this value as a podName.

- **ossm_get_logs** - Get the logs of a Kubernetes Pod (or workload name that will be resolved to a pod) in a namespace. Output is plain text, matching kubernetes-mcp-server pods_log. The line_count field tells you the total number of log lines returned. Analyze ALL of them, but summarize the results unless the user explicitly asks for the raw output. Do not omit any error or warning lines.
  - `container` (`string`) - Optional. Name of the Pod container to get the logs from.
  - `format` (`string`) - Output formatting for chat. 'codeblock' wraps logs in ~~~ fences (recommended). 'plain' returns raw text like kubernetes-mcp-server pods_log.
  - `meshCluster` (`string`) - Optional Istio mesh cluster name from ossm_list_mesh_clusters (e.g. west). When omitted, Kiali defaults to its home cluster.
  - `name` (`string`) **(required)** - Name of the Pod to get the logs from. If it does not exist, it will be treated as a workload name and a running pod will be selected.
  - `namespace` (`string`) **(required)** - Namespace to get the Pod logs from
  - `previous` (`boolean`) - Optional. Return previous terminated container logs
  - `severity` (`string`) - Optional severity filter applied client-side. Accepts 'ERROR', 'WARN' or combinations like 'ERROR,WARN'.
  - `tail` (`integer`) - Number of lines to retrieve from the end of the logs (Optional, defaults to 50). Cannot exceed 200 lines.
  - `workload` (`string`) - Optional. Workload name override (used when name lookup fails).

- **ossm_get_metrics** - Returns a compact JSON summary of Istio metrics (latency quantiles, traffic trends, throughput, payload sizes) for the given resource.
  - `byLabels` (`string`) - Comma-separated list of labels to group metrics by (e.g., 'source_workload,destination_service'). Optional
  - `direction` (`string`) - Traffic direction. Optional, defaults to 'outbound'
  - `meshCluster` (`string`) - Optional Istio mesh cluster name from ossm_list_mesh_clusters (e.g. west). When omitted, Kiali defaults to its home cluster.
  - `namespace` (`string`) **(required)** - Namespace to get metrics from
  - `quantiles` (`string`) - Comma-separated list of quantiles for histogram metrics (e.g., '0.5,0.95,0.99'). Optional
  - `rateInterval` (`string`) - Rate interval for metrics (e.g., '1m', '5m'). Optional, defaults to '10m'
  - `reporter` (`string`) - Metrics reporter(s). Comma-separated list of: 'source', 'destination', 'waypoint', or the special value 'both' (no reporter filter). Optional, defaults to 'source'. Example: 'source,waypoint'
  - `requestProtocol` (`string`) - Filter by request protocol (e.g., 'http', 'grpc', 'tcp'). Optional
  - `resourceName` (`string`) **(required)** - Name of the resource to get metrics for
  - `resourceType` (`string`) **(required)** - Type of resource to get metrics
  - `step` (`string`) - Step between data points in seconds (e.g., '15'). Optional, defaults to 15 seconds

</details>

<details>

<summary>ovn-kubernetes</summary>

- **ovn_show** - Display a comprehensive overview of OVN configuration from either the Northbound or Southbound database.

For Northbound (nbdb): Runs 'ovn-nbctl show' and displays logical switches, logical routers,
their ports, and connections between them.

For Southbound (sbdb): Runs 'ovn-sbctl show' and displays chassis information, port bindings,
and their relationships. Returns 100 lines by default; use head/tail to adjust.

Example output for nbdb:
{
  "database": "nbdb",
  "output": "switch 1234-5678 (node1)\n    port node1-k8s\n        addresses: [\"00:00:00:00:00:01\"]\n..."
}
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
  - `database` (`string`) **(required)** - OVN database to query - "nbdb" for Northbound or "sbdb" for Southbound
  - `head` (`integer`) - Return only first N lines. Default: 100 lines if tail is not specified
  - `name` (`string`) **(required)** - Name of the pod running OVN (e.g., "ovnkube-node-xxxxx")
  - `namespace` (`string`) - Kubernetes namespace of the OVN pod (e.g., "openshift-ovn-kubernetes")
  - `tail` (`integer`) - Return only last N lines

- **ovn_get** - Query records from an OVN database table with flexible filtering.

This is a versatile command that can:
1. List all records in a table (when no record specified)
2. Get a specific record (when record specified)

Common Northbound tables: Logical_Switch, Logical_Router, Logical_Switch_Port, 
Logical_Router_Port, ACL, Address_Set, Port_Group, Load_Balancer, NAT

Common Southbound tables: Chassis, Port_Binding, Datapath_Binding, Logical_Flow,
MAC_Binding, Multicast_Group, SB_Global

Returns 100 lines by default; use head/tail to adjust.

Example listing all records:
{
  "database": "nbdb",
  "table": "Port_Group",
  "output": "_uuid: 1234-5678\nname: \"pg_default\"\nports: [...]\n\n_uuid: abcd-efgh\n..."
}

Example getting a specific record:
{
  "database": "nbdb",
  "table": "Logical_Router",
  "record": "ovn_cluster_router",
  "output": "_uuid: 4c4a0a35-348c-41cc-8417-53a618e0c383\nname: ovn_cluster_router\nports: [...]"
}

Example getting specific columns:
{
  "database": "nbdb",
  "table": "Logical_Switch",
  "columns": "name,ports",
  "output": "name: ovn-worker\nports: [uuid1, uuid2]\n\nname: join\nports: [uuid3]"
}
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
  - `columns` (`string`) - Comma-separated list of columns to display (e.g., "name,_uuid,ports")
  - `database` (`string`) **(required)** - OVN database to query - "nbdb" for Northbound or "sbdb" for Southbound
  - `head` (`integer`) - Return only first N lines. Default: 100 lines if tail is not specified
  - `name` (`string`) **(required)** - Name of the pod running OVN
  - `namespace` (`string`) - Kubernetes namespace of the OVN pod
  - `pattern` (`string`) - Regex pattern to filter results. Only applies when listing all records.
  - `record` (`string`) - Record identifier (UUID or name). If not specified, lists all records
  - `table` (`string`) **(required)** - Name of the table (e.g., "Logical_Switch", "Port_Binding")
  - `tail` (`integer`) - Return only last N lines

- **ovn_lflow_list** - List logical flows from the OVN Southbound database.

Runs 'ovn-sbctl lflow-list' to retrieve logical flows which represent the compiled
logical network pipeline. This is essential for debugging packet forwarding.
Returns 100 lines by default; use head/tail to adjust.

Example output:
{
  "datapath": "node1",
  "flows": [
    "table=0 (ls_in_port_sec_l2), priority=100, match=(inport == \"pod1\"), action=(next;)",
    "table=1 (ls_in_port_sec_ip), priority=90, match=(ip4), action=(next;)"
  ]
}
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
  - `datapath` (`string`) - Datapath name or UUID to filter flows for a specific logical switch/router
  - `head` (`integer`) - Return only first N lines. Default: 100 lines if tail is not specified
  - `name` (`string`) **(required)** - Name of the pod running OVN
  - `namespace` (`string`) - Kubernetes namespace of the OVN pod
  - `pattern` (`string`) - Regex pattern to filter flows
  - `tail` (`integer`) - Return only last N lines

- **ovn_trace** - Trace a packet through the OVN logical network.

Runs 'ovn-trace' to simulate packet processing through the logical network pipeline.
This shows which logical flows match, what actions are taken, and the final disposition.

The trace is essential for debugging connectivity issues and understanding how traffic
flows through the OVN logical network. Returns 100 lines by default; use head/tail to adjust.

Microflow specification examples:
- inport=="pod1" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5
- inport=="pod1" && eth.src==00:00:00:00:00:01 && icmp && ip4.src==10.244.0.5 && ip4.dst==8.8.8.8

Example output:
{
  "datapath": "node1",
  "microflow": "inport==\"pod1\" && ...",
  "output": "ingress(dp=\"node1\", inport=\"pod1\")\n  0. ls_in_port_sec_l2: inport == \"pod1\", priority 50, uuid 1234\n     next;\n..."
}
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
  - `datapath` (`string`) **(required)** - Name of the logical switch or router to start the trace
  - `head` (`integer`) - Return only first N lines. Default: 100 lines if tail is not specified
  - `microflow` (`string`) **(required)** - Microflow specification describing the packet (e.g., "inport==\"pod1\" && eth.src==00:00:00:00:00:01 && ip4.src==10.244.0.5 && ip4.dst==10.244.1.5")
  - `mode` (`string`) - Output verbosity mode - "detailed" (default), "summary", or "minimal"
  - `name` (`string`) **(required)** - Name of the pod running OVN
  - `namespace` (`string`) - Kubernetes namespace of the OVN pod
  - `pattern` (`string`) - Regex pattern to filter trace output
  - `tail` (`integer`) - Return only last N lines

- **ovs_vsctl** - Run an ovs-vsctl command against an ovnkube-node pod.

The 'action' parameter selects the ovs-vsctl subcommand to run.

--- action: "show" ---
Display a comprehensive overview of OVS configuration.

Runs 'ovs-vsctl show' command and returns detailed information about bridges, ports, interfaces,
controllers, and their configurations in a hierarchical format.

This command is useful for getting a complete view of the OVS switch configuration including:
- All bridges and their configurations
- Ports and interfaces attached to each bridge
- Controller connections and status
- Interface types and options
- Port configurations and tags

Example output:
{
  "output": "a1b2c3d4-5678-90ab-cdef-1234567890ab\n    Bridge br-int\n        Port ovn-k8s-mp0\n            Interface ovn-k8s-mp0\n                type: internal\n        Port br-int\n            Interface br-int\n                type: internal\n    ovs_version: \"2.17.0\""
}

--- action: "list-br" ---
List all OVS bridges on a specific pod.

Runs 'ovs-vsctl list-br' command and returns the names of all configured bridges.

Example output:
{
  "bridges": [
    "br-int",
    "br-ex",
    "br-local"
  ]
}

--- action: "list-ports" ---
List all ports on a specific OVS bridge.

Runs 'ovs-vsctl list-ports' command and returns the names of all ports attached to the specified bridge.

Example output:
{
  "ports": [
    "patch-br-int-to-br-ex",
    "veth1234",
    "ovn-k8s-mp0"
  ]
}

--- action: "list-ifaces" ---
List all interfaces on a specific OVS bridge.

Runs 'ovs-vsctl list-ifaces' command and returns the names of all interfaces attached to the specified bridge.

Example output:
{
  "interfaces": [
    "patch-br-int-to-br-ex",
    "veth1234",
    "ovn-k8s-mp0"
  ]
}
  - `action` (`string`) **(required)** - The ovs-vsctl subcommand to run: "show", "list-br", "list-ports", or "list-ifaces"
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head (only used when action is "show"). Default: false
  - `bridge` (`string`) - Name of the OVS bridge (required for "list-ports" and "list-ifaces"; e.g., "br-int")
  - `head` (`integer`) - Return only first N lines (only used when action is "show"). Default: 100 lines if tail is not specified
  - `name` (`string`) **(required)** - Name of the ovnkube-node pod (e.g., "ovnkube-node-xxxxx")
  - `namespace` (`string`) **(required)** - Kubernetes namespace of the ovnkube-node pod (e.g., "openshift-ovn-kubernetes")
  - `tail` (`integer`) - Return only last N lines (only used when action is "show")

- **ovs_ofctl** - Run an ovs-ofctl command against an ovnkube-node pod.

The 'action' parameter selects the ovs-ofctl subcommand to run.

--- action: "dump-flows" ---
Dump OpenFlow flows from a specific OVS bridge.

Runs 'ovs-ofctl dump-flows' command on the specified bridge and returns the flow entries.

Example output:
{
  "bridge": "br-int",
  "flows": [
    "cookie=0x0, duration=123.456s, table=0, n_packets=100, n_bytes=10000, priority=100,in_port=1 actions=output:2",
    "cookie=0x0, duration=123.456s, table=0, n_packets=50, n_bytes=5000, priority=90,in_port=2 actions=output:1"
  ]
}
  - `action` (`string`) **(required)** - The ovs-ofctl subcommand to run: "dump-flows"
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
  - `bridge` (`string`) **(required)** - Name of the OVS bridge (e.g., "br-int")
  - `head` (`integer`) - Return only first N lines. Default: 100 lines if tail is not specified
  - `name` (`string`) **(required)** - Name of the ovnkube-node pod (e.g., "ovnkube-node-xxxxx")
  - `namespace` (`string`) **(required)** - Kubernetes namespace of the ovnkube-node pod (e.g., "openshift-ovn-kubernetes")
  - `pattern` (`string`) - Regex pattern to filter output lines
  - `tail` (`integer`) - Return only last N lines

- **ovs_appctl** - Run an ovs-appctl command against an ovnkube-node pod.

The 'action' parameter selects the ovs-appctl subcommand to run.

--- action: "dpctl/dump-conntrack" ---
Dump connection tracking entries from OVS datapath.

Runs 'ovs-appctl dpctl/dump-conntrack' command and returns the conntrack entries.

Connection tracking (conntrack) maintains state for stateful firewall rules and NAT.
Each entry shows source/destination IPs, ports, protocol, connection state, and more.

Example output:
{
  "entries": [
    "tcp,orig=(src=10.244.0.5,dst=10.96.0.1,sport=45678,dport=443),reply=(src=10.96.0.1,dst=10.244.0.5,sport=443,dport=45678)",
    "udp,orig=(src=10.244.0.3,dst=8.8.8.8,sport=53214,dport=53),reply=(src=8.8.8.8,dst=10.244.0.3,sport=53,dport=53214)"
  ]
}

--- action: "ofproto/trace" ---
Trace a packet through the OpenFlow pipeline.

Runs 'ovs-appctl ofproto/trace' command to simulate packet processing through OpenFlow tables.
This shows which flows match, what actions are taken, and the final disposition of the packet.

The trace output is essential for debugging flow rules, understanding packet forwarding decisions,
and troubleshooting connectivity issues.

Flow specification examples:
- "in_port=1,icmp"
- "in_port=2,ip,nw_src=192.168.1.10,nw_dst=192.168.1.20"
- "in_port=3,tcp,nw_src=10.0.0.1,nw_dst=10.0.0.2,tp_src=12345,tp_dst=80"

Example output:
{
  "bridge": "br-int",
  "flow": "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1",
  "output": "Flow: ip,in_port=1,nw_src=10.244.0.5,nw_dst=10.96.0.1\n\nbridge(\"br-int\")\n-------------\n 0. priority 100\n    resubmit(,10)\n10. ip,nw_dst=10.96.0.1, priority 200\n    load:0x1->NXM_NX_REG0[]\n    resubmit(,20)\n...\nFinal flow: ...\nDatapath actions: ..."
}
  - `action` (`string`) **(required)** - The ovs-appctl subcommand to run: "dpctl/dump-conntrack" or "ofproto/trace"
  - `additional_params` (`array`) - Additional CLI arguments (only used when action is "dpctl/dump-conntrack"; e.g., ["zone=5"])
  - `apply_tail_first` (`boolean`) - If both head and tail are set and apply_tail_first is true, apply tail before head. Default: false
  - `bridge` (`string`) - Name of the OVS bridge (required for "ofproto/trace"; e.g., "br-int")
  - `flow` (`string`) - Flow specification (required for "ofproto/trace"; e.g., "in_port=1,ip,nw_src=10.244.0.5,nw_dst=10.96.0.1")
  - `head` (`integer`) - Return only first N lines. Default: 100 lines if tail is not specified
  - `name` (`string`) **(required)** - Name of the ovnkube-node pod (e.g., "ovnkube-node-xxxxx")
  - `namespace` (`string`) **(required)** - Kubernetes namespace of the ovnkube-node pod (e.g., "openshift-ovn-kubernetes")
  - `pattern` (`string`) - Regex pattern to filter output lines
  - `tail` (`integer`) - Return only last N lines

</details>

<details>

<summary>tekton</summary>

- **tekton_pipeline_start** - Start a Tekton Pipeline by creating a PipelineRun that references it
  - `name` (`string`) **(required)** - Name of the Pipeline to start
  - `namespace` (`string`) - Namespace of the Pipeline
  - `params` (`object`) - Parameter values to pass to the Pipeline. Keys are parameter names; values can be a string, an array of strings, or an object (map of string to string) depending on the parameter type defined in the Pipeline spec

- **tekton_pipelinerun_lifecycle** - Manage a Tekton PipelineRun lifecycle by restarting it with the same spec or cancelling it by setting spec.status to Cancelled.
  - `action` (`string`) **(required)** - Lifecycle action to perform: 'restart' creates a new PipelineRun with the same spec; 'cancel' sets spec.status to Cancelled.
  - `name` (`string`) **(required)** - Name of the PipelineRun to manage
  - `namespace` (`string`) - Namespace of the PipelineRun

- **tekton_pipelinerun_logs** - Get logs for all TaskRuns owned by a Tekton PipelineRun. Use this to inspect PipelineRun execution output without locating pods manually.
  - `name` (`string`) **(required)** - Name of the PipelineRun to get logs from
  - `namespace` (`string`) - Namespace of the PipelineRun
  - `tail` (`integer`) - Number of lines to retrieve from the end of each container log (default: 100)

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

- **vm-troubleshoot** - Generate a step-by-step troubleshooting guide for diagnosing OpenShift Virtualization VirtualMachine issues
  - `namespace` (`string`) **(required)** - The namespace of the VirtualMachine to troubleshoot
  - `name` (`string`) **(required)** - The name of the VirtualMachine to troubleshoot

- **windows-golden-image** - Guides creation of a Windows golden image via the OpenShift Virtualization windows-efi-installer Tekton pipeline
  - `winImageDownloadURL` (`string`) **(required)** - Microsoft Windows ISO download URL (must be https://)
  - `namespace` (`string`) - Target namespace for the PipelineRun
  - `windowsVersion` (`string`) - Windows version: 10, 11, 2k22 (default), or 2k25
  - `pipelineVersion` (`string`) - Pipeline version (default: latest). Use specific version like 0.25.0 if needed

</details>

<details>

<summary>oadp</summary>

- **oadp-troubleshoot** - Generate a step-by-step troubleshooting guide for diagnosing OADP backup and restore issues
  - `namespace` (`string`) - The OADP namespace (default: openshift-adp)
  - `backup` (`string`) - The name of a specific backup to troubleshoot
  - `restore` (`string`) - The name of a specific restore to troubleshoot

</details>

<details>

<summary>openshift</summary>

- **plan_mustgather** - Plan for collecting a must-gather archive from an OpenShift cluster. Must-gather is a tool for collecting cluster data related to debugging and troubleshooting like logs, kubernetes resources, etc.
  - `node_name` (`string`) - Specific node name to run must-gather pod on
  - `node_selector` (`string`) - Node selector in key=value,key2=value2 format to filter nodes for the pod
  - `source_dir` (`string`) - Custom gather directory inside pod (default: /must-gather)
  - `namespace` (`string`) - Privileged namespace to use for must-gather (auto-generated if not specified)
  - `gather_command` (`string`) - Custom gather command eg. /usr/bin/gather_audit_logs (default: /usr/bin/gather)
  - `timeout` (`string`) - Timeout duration for gather command (eg. 30m, 1h)
  - `since` (`string`) - Only gather data newer than this duration (eg. 5s, 2m5s, or 3h6m10s) defaults to all data.
  - `host_network` (`string`) - Use host network for must-gather pod (true/false)
  - `keep_resources` (`string`) - Keep pod resources after collection (true/false, default: false)
  - `all_component_images` (`string`) - Include must-gather images from all installed operators (true/false)
  - `images` (`string`) - Comma-separated list of custom must-gather container images

</details>

<details>

<summary>ossm</summary>

- **mesh-list-applications** - List applications in the mesh namespaces
  - `namespace` (`string`) - Optional namespace to filter applications (default: all namespaces)

- **list-istio-config** - List Istio configuration resources in the mesh namespaces
  - `namespace` (`string`) - Optional namespace to filter Istio configuration (default: all namespaces)

- **mesh-list-namespaces** - List all namespaces with their sidecar injection status and Istio labels

- **mesh-list-services** - List services in the mesh namespaces
  - `namespace` (`string`) - Optional namespace to filter services (default: all namespaces)

- **mesh-list-workloads** - List workloads in the mesh namespaces
  - `namespace` (`string`) - Optional namespace to filter workloads (default: all namespaces)

- **mesh-health-check** - Perform a comprehensive health assessment of the Istio service mesh including control plane and data plane status
  - `namespace` (`string`) - Optional namespace to focus the health check on (default: all namespaces)

- **mesh-topology** - Show the mesh topology including control plane components and cluster connectivity

- **traffic-topology** - Analyze the service mesh traffic topology showing service dependencies, traffic flow, and communication patterns
  - `namespaces` (`string`) **(required)** - Comma-separated list of namespaces to include in the graph, or 'all' to include all accessible mesh namespaces

- **service-troubleshoot** - Investigate service errors using logs, traces, and Istio configuration to identify root causes
  - `namespace` (`string`) **(required)** - Namespace where the service is deployed
  - `service` (`string`) **(required)** - Name of the service to troubleshoot
  - `workload` (`string`) - Optional workload or pod name to fetch logs from (if omitted, uses the service name)

- **trace-analysis** - Investigate distributed traces for a service to identify latency bottlenecks, error sources, and slow spans
  - `namespace` (`string`) **(required)** - Namespace where the service is deployed
  - `service` (`string`) **(required)** - Name of the service to investigate traces for

- **istio-config-review** - Review and validate Istio configuration in a namespace, checking for misconfigurations and best practice violations
  - `namespace` (`string`) **(required)** - Namespace to review Istio configuration for

</details>

<details>

<summary>tekton</summary>

- **pipeline-troubleshoot** - Gather PipelineRun status, TaskRuns, logs, events, Pipeline-as-Code Repository, and TektonConfig context for Tekton troubleshooting
  - `namespace` (`string`) **(required)** - Namespace of the PipelineRun to troubleshoot
  - `name` (`string`) **(required)** - Name of the PipelineRun to troubleshoot

</details>


<!-- AVAILABLE-TOOLSETS-PROMPTS-END -->

### Resources

<!-- AVAILABLE-TOOLSETS-RESOURCES-START -->

<details>

<summary>openshift/mustgather</summary>

- **must-gather** - Loaded must-gather archive metadata
  - URI: `must-gather://current`
  - MIME Type: `text/plain`
- **must-gather-namespaces** - List of all namespaces in the must-gather archive
  - URI: `must-gather://current/namespaces`
  - MIME Type: `text/plain`
- **must-gather-etcd-members** - ETCD cluster member list from the must-gather archive
  - URI: `must-gather://current/etcd/members`
  - MIME Type: `application/json`
- **must-gather-etcd-endpoint-status** - ETCD endpoint status from the must-gather archive
  - URI: `must-gather://current/etcd/endpoint-status`
  - MIME Type: `application/json`
- **must-gather-prometheus-config** - Prometheus configuration summary from the must-gather archive
  - URI: `must-gather://current/prometheus/config`
  - MIME Type: `text/plain`
- **must-gather-alertmanager-status** - AlertManager status from the must-gather archive
  - URI: `must-gather://current/alertmanager/status`
  - MIME Type: `text/plain`
</details>


<!-- AVAILABLE-TOOLSETS-RESOURCES-END -->

### Resource Templates

<!-- AVAILABLE-TOOLSETS-RESOURCES-TEMPLATES-START -->

<details>

<summary>openshift/mustgather</summary>

- **must-gather-resource** - A specific Kubernetes resource from the must-gather archive as YAML. Use '-' for empty group (core API) or cluster-scoped namespace.
  - URI Template: `must-gather://current/resources/{group}/{version}/{kind}/{namespace}/{name}`
  - MIME Type: `text/yaml`
</details>


<!-- AVAILABLE-TOOLSETS-RESOURCES-TEMPLATES-END -->

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
