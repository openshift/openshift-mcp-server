# Red Hat OpenShift Router

This document provides guidance on using the OpenShift router with the Kubernetes MCP Server.

## Overview

The router toolset enables AI assistants to inspect and monitor OpenShift router pods directly through the Kubernetes MCP Server. This includes viewing router configuration, checking runtime statistics, and monitoring active connections without requiring manual pod access or HAProxy command-line knowledge.

## Prerequisites

- OpenShift Container Platform 4.x cluster

## Enabling the Toolset

The `router` toolset is not enabled by default. Enable it using the `--toolsets` flag:

```bash
kubernetes-mcp-server --toolsets core,config,helm,router
```

Or in your MCP client configuration:

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": [
        "-y",
        "kubernetes-mcp-server@latest",
        "--toolsets", "core,config,helm,router"
      ]
    }
  }
}
```

## Available Tools

The router toolset provides tools to inspect and monitor OpenShift router pods.

### Configuration and Runtime Information

| Tool | Description | Parameters |
|------|-------------|------------|
| `router_show_config` | View the router's configuration | `pod` (optional) - Router pod name |
| `router_show_info` | Get router runtime information and statistics | `pod` (optional) - Router pod name |
| `router_show_sessions` | View all active sessions in the router | `pod` (optional) - Router pod name |

**Notes**:
- All tools have an optional `pod` parameter. If not provided, the tools will automatically select any available router pod from the default ingress controller.
- Currently, the default ingress controller name (`default`) is hard-coded. Support for custom ingress controller names may be added in future versions.

## Examples

### View Router Configuration

View the router configuration file from any router pod:

```
Show me the router configuration

● I'll get the router configuration for you.
  ⎿  # Router configuration (pod: router-default-6fbcf6cb84-vqthh)
     ```
     global
     … +333 lines (ctrl+o to expand)

● Here's the router configuration from pod router-default-6fbcf6cb84-vqthh. This is an HAProxy configuration for the OpenShift router with the following key settings:

  Global Settings:
  - Max connections: 50,000
  - 4 worker threads
  - TLS: Minimum version TLSv1.2
  - Cipher suites: Modern secure ciphers (ECDHE-ECDSA, ECDHE-RSA, etc.)
  - DH parameter size: 2048 bits

  Timeout Settings:
  - Connect: 5s
  - Client/Server: 30s
  - HTTP request: 10s
  - HTTP keep-alive: 300s
  - WebSocket tunnel: 1h
  - Console: 5 minutes
```

Or specify a particular router pod:

```
Show me the router configuration from pod router-default-6fbcf6cb84-vqthh
```

### View Router Runtime Information

Get router runtime statistics and information:

```
Show me the router info
  ⎿  # Router HAProxy info (pod: router-default-6fbcf6cb84-vqthh)
     ```
     Name: HAProxy
     … +76 lines (ctrl+o to expand)

● Here's the HAProxy runtime information for the router (pod: router-default-6fbcf6cb84-vqthh):

  Version Information:
  - HAProxy Version: 2.8.10-f28885f
  - Release Date: 2024/06/14
  - Uptime: 1h 45m 7s

  Process Information:
  - PID: 138
  - Threads: 4
  - Processes: 1

  Connection Statistics:
  - Current Connections: 10
  - Max Connections: 50,000
  - Cumulative Connections: 2,253
  - Cumulative Requests: 4,668
  - Max Connection Rate: 12/sec

  SSL/TLS Statistics:
  - Current SSL Connections: 1
  - Cumulative SSL Connections: 862
  - SSL Frontend Max Key Rate: 4/sec
  - SSL Backend Max Key Rate: 1/sec
```

This displays:
- Router version and uptime
- Process information
- Connection statistics
- Request rates
- Memory usage

### View Active Sessions

View all active sessions in the router:

```
Show me active router sessions
  ⎿  # Router active sessions (pod: router-default-6fbcf6cb84-vqthh)
     ```
     0x7f223405ae50: [12/Jan/2026:22:35:39.393119] id=4697 proto=unix_stream source=unix:1
     … +17 lines (ctrl+o to expand)

● Here are the active sessions in the router (pod: router-default-6fbcf6cb84-vqthh):

  There is currently 1 active session:

  Session ID 4697 (CLI Connection)
  - Protocol: Unix stream socket
  - Frontend: GLOBAL (HAProxy admin socket)
  - State: ESTABLISHED
  - Age: 0s (just created)
  - Type: CLI applet connection

  Details:
  - Connection flags: 0x8
  - Task state: running with high priority (nice=-64)
  - File descriptor: 23
  - Buffers: 32KB request/response buffers allocated
  - Request data: 14 bytes total (the command itself)
  - Timeouts: 2 minutes for client, no write timeout
```

This displays detailed information about each active connection including:
- Session ID and protocol
- Connection state
- Frontend and backend information
- Task and file descriptor details

## Quick Start

1. Enable the router toolset when starting the server:

```bash
kubernetes-mcp-server --toolsets core,config,router
```

2. View router configuration:

```
Show me the router configuration
```

3. Query router information:

```
Show me the router info
```

## Troubleshooting

### No Router Pod Found

If you get an error that no router pod was found, verify that:
- You're connected to an OpenShift cluster (not plain Kubernetes)
- The default ingress controller exists in the `openshift-ingress` namespace
- Router pods are running

Check router pods:
```
List pods in namespace openshift-ingress
```
