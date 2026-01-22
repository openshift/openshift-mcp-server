# Observability Toolset

This toolset provides tools for querying OpenShift cluster observability data including Prometheus metrics and Alertmanager alerts.

## Tools

### prometheus_query

Execute instant PromQL queries against the cluster's Thanos Querier.

**Parameters:**
- `query` (required) - PromQL query string
- `time` (optional) - Evaluation timestamp (RFC3339, Unix timestamp, or relative like `-5m`, `now`)

**Example:**
```
Query: up{job="apiserver"}
```

### prometheus_query_range

Execute range PromQL queries for time-series data.

**Parameters:**
- `query` (required) - PromQL query string
- `start` (required) - Start time (RFC3339, Unix timestamp, or relative like `-1h`)
- `end` (required) - End time (RFC3339, Unix timestamp, or relative like `now`)
- `step` (optional) - Query resolution step (default: `1m`)

**Example:**
```
Query: rate(container_cpu_usage_seconds_total[5m])
Start: -1h
End: now
Step: 1m
```

### alertmanager_alerts

Query alerts from the cluster's Alertmanager.

**Parameters:**
- `active` (optional) - Include active alerts (default: true)
- `silenced` (optional) - Include silenced alerts (default: false)
- `inhibited` (optional) - Include inhibited alerts (default: false)
- `filter` (optional) - Label filter in PromQL format (e.g., `alertname="Watchdog"`)

**Example:**
```
Active: true
Filter: severity="critical"
```

## Enable the Observability Toolset

### Option 1: Command Line

```bash
kubernetes-mcp-server --toolsets core,config,helm,observability
```

### Option 2: Configuration File

```toml
toolsets = ["core", "config", "helm", "observability"]
```

### Option 3: MCP Client Configuration

```json
{
  "mcpServers": {
    "kubernetes": {
      "command": "npx",
      "args": ["-y", "kubernetes-mcp-server@latest", "--toolsets", "core,config,helm,observability"]
    }
  }
}
```

## Configuration

The observability toolset supports optional configuration via the config file:

```toml
[observability]
# Custom monitoring namespace (default: "openshift-monitoring")
monitoring_namespace = "custom-monitoring"
```

| Option | Default | Description |
|--------|---------|-------------|
| `monitoring_namespace` | `openshift-monitoring` | Namespace where Prometheus and Alertmanager routes are located |

## Prerequisites

The observability tools require:

1. **OpenShift cluster** - These tools are designed for OpenShift and rely on OpenShift-specific routes
2. **Monitoring stack enabled** - The cluster must have the monitoring stack deployed (default in OpenShift)
3. **Proper RBAC** - The user/service account must have permissions to:
   - Read routes in `openshift-monitoring` namespace
   - Access the Thanos Querier and Alertmanager APIs

## How It Works

### Route Discovery

The tools automatically discover the Prometheus (Thanos Querier) and Alertmanager endpoints by reading OpenShift routes:

- **Thanos Querier**: `thanos-querier` route in `openshift-monitoring` namespace
- **Alertmanager**: `alertmanager-main` route in `openshift-monitoring` namespace

### Authentication

The tools use the bearer token from your Kubernetes configuration to authenticate with the monitoring endpoints. This is the same credential used to access the cluster.

### Relative Time Support

Time parameters support multiple formats:

| Format | Example | Description |
|--------|---------|-------------|
| RFC3339 | `2024-01-15T10:00:00Z` | Absolute timestamp |
| Unix | `1705312800` | Unix timestamp in seconds |
| Relative | `-10m`, `-1h`, `-1d` | Relative to current time |
| Keyword | `now` | Current time |

## Security Considerations

### Allowed Prometheus Endpoints

Only read-only Prometheus API endpoints are allowed:
- `/api/v1/query` - Instant queries
- `/api/v1/query_range` - Range queries
- `/api/v1/series` - Series metadata
- `/api/v1/labels` - Label names
- `/api/v1/label/<name>/values` - Label values

Administrative endpoints (like `/api/v1/admin/*`) are blocked.

### Allowed Alertmanager Endpoints

Only alert query endpoints are allowed:
- `/api/v2/alerts` - List alerts
- `/api/v2/silences` - List silences
- `/api/v1/alerts` - Legacy alert endpoint

### Query Limits

- Maximum query length: 10,000 characters
- Maximum response size: 10MB

## Common Use Cases

### Cluster Health

**Check if all API servers are up:**
```
Query: up{job="apiserver"}
```

**API server request latency (99th percentile):**
```
Query: histogram_quantile(0.99, sum(rate(apiserver_request_duration_seconds_bucket[5m])) by (le, verb))
```

### Node and Pod Metrics

**Node CPU usage percentage:**
```
Query: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
```

**Pods in CrashLoopBackOff:**
```
Query: kube_pod_container_status_waiting_reason{reason="CrashLoopBackOff"} > 0
```

**Container memory usage by namespace:**
```
Query: sum by(namespace) (container_memory_working_set_bytes{container!=""})
```

### Alerting

**Get all firing critical alerts:**
```
Tool: alertmanager_alerts
Active: true
Filter: severity="critical"
```

**Count alerts by severity:**
```
Query: count by(severity) (ALERTS{alertstate="firing"})
```

### Network

**Network receive rate by pod:**
```
Query: rate(container_network_receive_bytes_total[5m])
Start: -1h
End: now
Step: 1m
```

### etcd Health

**etcd leader changes:**
```
Query: changes(etcd_server_leader_changes_seen_total[1h])
```

**etcd disk sync duration:**
```
Query: histogram_quantile(0.99, rate(etcd_disk_wal_fsync_duration_seconds_bucket[5m]))
```

## Troubleshooting

### "failed to get route" Error

The monitoring routes may not exist or the user lacks permissions:
```bash
oc get routes -n openshift-monitoring
```

### "no bearer token available" Error

Ensure your kubeconfig has a valid token:
```bash
oc whoami
oc get pods -n openshift-monitoring
```

### Empty Results from Prometheus

Verify the query works in the OpenShift console:
1. Go to **Observe** > **Metrics**
2. Enter your PromQL query
3. Check for results

### TLS Certificate Errors

The tools use `InsecureSkipVerify` for route access. If you need strict TLS verification, this would require additional configuration.
