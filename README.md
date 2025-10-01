# Kiali MCP Server

Kiali MCP Server is a thin extension of the upstream Kubernetes MCP Server. It adds Kiali-specific tooling while keeping the same core behavior and configuration.

- Based on `kubernetes-mcp-server` (native Go MCP server for Kubernetes/OpenShift)
- For the full set of tools and behavior adopted from upstream, see the upstream README: [openshift/openshift-mcp-server README](https://github.com/openshift/openshift-mcp-server/blob/main/README.md)

## Features

- Kiali integrations:
  - `validations_list`: Lists Istio object validations aggregated by namespace and cluster from a Kiali instance.

## Requirements

- Access to a Kubernetes or OpenShift cluster (kubeconfig or in-cluster service account)
- A reachable Kiali server URL

## Configuration

Kiali MCP Server reuses the same configuration and flags as the upstream Kubernetes MCP Server. In addition, it adds the following Kiali-specific flags:

- `--kiali-server-url` string: URL of the Kiali server (e.g. "https://kiali-istio-system.apps-crc.testing/")
- `--kiali-insecure`: Skip TLS verification when connecting to the Kiali server

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



## 🧑‍💻 Development <a id="development"></a>

### Running with mcp-inspector

Compile the project and run the Kiali MCP server with [mcp-inspector](https://modelcontextprotocol.io/docs/tools/inspector) to inspect the MCP server.

```shell
# Compile the project
make build
# Run the Kubernetes MCP server with mcp-inspector
npx @modelcontextprotocol/inspector@latest $(pwd)/kiali-mcp-server --kiali-server-url "https://kiali-istio-system.apps-crc.testing/" --kiali-insecure
```
