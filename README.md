# Kiali MCP Server

[![GitHub License](https://img.shields.io/github/license/kiali/kiali-mcp-server)](https://github.com/kiali/kiali-mcp-server/blob/main/LICENSE)
[![npm](https://img.shields.io/npm/v/kiali-mcp-server)](https://www.npmjs.com/package/kiali-mcp-server)
[![PyPI - Version](https://img.shields.io/pypi/v/kiali-mcp-server)](https://pypi.org/project/kiali-mcp-server/)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/kiali/kiali-mcp-server?sort=semver)](https://github.com/kiali/kiali-mcp-server/releases/latest)
[![Build](https://github.com/kiali/kiali-mcp-server/actions/workflows/build.yaml/badge.svg)](https://github.com/kiali/kiali-mcp-server/actions/workflows/build.yaml)

https://github.com/user-attachments/assets/d88a3b72-980c-43db-a69a-a19ad564cf49

Kiali MCP Server is a thin extension of the upstream Kubernetes MCP Server. It adds Kiali-specific tooling while keeping the same core behavior and configuration.

- Based on `kubernetes-mcp-server` (native Go MCP server for Kubernetes/OpenShift)
- For the full set of tools and behavior adopted from upstream, see the upstream README: [openshift/openshift-mcp-server README](https://github.com/openshift/openshift-mcp-server/blob/main/README.md)

[✨ Features](#features) | [🚀 Getting Started](#getting-started) | [🎥 Demos](#demos) | [⚙️ Configuration](#configuration) | [🛠️ Tools](#tools-and-functionalities) | [🧑‍💻 Development](#development)

## ✨ Features <a id="features"></a>

- **✅ Istio Objects**:
  - `validations_list`: Lists Istio object validations aggregated by namespace and cluster from a Kiali instance.

## 🚀 Getting Started <a id="getting-started"></a>

### Requirements

- Access to a Kubernetes or OpenShift cluster (kubeconfig or in-cluster service account)
- A reachable Kiali server URL

### Cursor

Install the Kubernetes MCP server extension in Cursor by pressing the following link:

[![Install MCP Server](https://cursor.com/deeplink/mcp-install-dark.svg)](https://cursor.com/install-mcp?name=kiali-mcp-server&config=eyJjb21tYW5kIjoibnB4IC15IGtpYWxpLW1jcC1zZXJ2ZXJAbGF0ZXN0In0%3D)


Alternatively, you can install the extension manually by editing the `mcp.json` file:

```json
{
  "mcpServers": {
    "kiali-mcp-server": {
      "command": "npx",
      "args": ["-y", "kiali-mcp-server@latest"]
    }
  }
}
```
*Note:* You must specify the Kiali endpoint if the MCP cannot detect it.You must also specify whether it should skip TLS. 
```json
{
  "mcpServers": {
    "kiali-mcp-server": {
      "command": "npx",
      "args": [
        "-y",
        "kiali-mcp-server@latest",
        "--kiali-server-url",
        "https://kiali-istio-system.apps-crc.testing/",
        "--kiali-insecure"
      ]
    }
  }
}
```

## Configuration <a id="configuration"></a>

Kiali MCP Server reuses the same configuration and flags as the upstream Kubernetes MCP Server. In addition, it adds the following Kiali-specific flags:

- `--kiali-server-url` string: URL of the Kiali server (e.g. "https://kiali-istio-system.apps-crc.testing/")
- `--kiali-insecure`: Skip TLS verification when connecting to the Kiali server

*By default, Kubernetes and Kiali tools are exposed. If you want to only Kiali use this configuration*

- `--toolsets`:  kiali

You can run the server via npx, uvx, or the compiled binary. Example using npx:

```sh
npx -y kiali-mcp-server@latest \
  --kiali-server-url "https://kiali-istio-system.apps-crc.testing/" \
  --kiali-insecure
```

Or using the binary after building:

```sh
./kiali-mcp-server \
  --kiali-server-url "https://kiali-istio-system.apps-crc.testing/" \
  --kiali-insecure
```

Refer to the upstream README for the rest of the flags and features (ports, auth, read-only, list output, etc.): [openshift/openshift-mcp-server README](https://github.com/openshift/openshift-mcp-server/blob/main/README.md)

## 🛠️ Tools and Functionalities <a id="tools-and-functionalities"></a>

The Kiali MCP server supports enabling or disabling specific groups of tools and functionalities (tools, resources, prompts, and so on) via the `--toolsets` command-line flag or `toolsets` configuration option.
This allows you to control which Kubernetes functionalities are available to your AI tools.
Enabling only the toolsets you need can help reduce the context size and improve the LLM's tool selection accuracy.

### Available Toolsets

The following sets of tools are available (only Kiali by default):

<!-- AVAILABLE-TOOLSETS-START -->

| Toolset | Description                          |
|---------|--------------------------------------|
| kiali   | Most common tools for managing Kiali |

<!-- AVAILABLE-TOOLSETS-END -->

### Tools

<!-- AVAILABLE-TOOLSETS-TOOLS-START -->

<details>

<summary>kiali</summary>

- **validations_list** - List all the validations in the current cluster from all namespaces
  - `namespace` (`string`) - Optional single namespace to retrieve validations from (alternative to namespaces)
  - `namespaces` (`string`) - Optional comma-separated list of namespaces to retrieve validations from

- **graph** - Check the status of my mesh by querying Kiali graph
  - `namespace` (`string`) - Optional single namespace to include in the graph (alternative to namespaces)
  - `namespaces` (`string`) - Optional comma-separated list of namespaces to include in the graph

</details>


<!-- AVAILABLE-TOOLSETS-TOOLS-END -->
## 🎥 Demos <a id="demos"></a>

In this video, we explore how the Mesh Control Plane (MCP) in Kubernetes/OpenShift works together with Kiali to validate Istio configuration objects directly in your editor (_Cursor_).

<a href="https://youtu.be/1l9m1B5uEPw" target="_blank">
 <img src="docs/images/kiali_mcp_cursor.png" alt="Cursor: Kiali-mcp-server running" width="240"  />
</a>


## 🧑‍💻 Development <a id="development"></a>

### Running with mcp-inspector

Compile the project and run the Kiali MCP server with [mcp-inspector](https://modelcontextprotocol.io/docs/tools/inspector) to inspect the MCP server.

```shell
# Compile the project
make build
# Run the Kubernetes MCP server with mcp-inspector
npx @modelcontextprotocol/inspector@latest $(pwd)/kiali-mcp-server --kiali-server-url "https://kiali-istio-system.apps-crc.testing/" --kiali-insecure
```
