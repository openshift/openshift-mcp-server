# OpenShift Service Mesh (OSSM) toolset

Downstream builds expose the upstream [Kiali toolset](KIALI.md) under the
OpenShift Service Mesh (OSSM) name: the toolset is registered as `ossm`, and
every tool is prefixed with `ossm_` (for example `ossm_get_mesh_status`,
`ossm_manage_istio_config`).

Enable it via the server TOML configuration file:

```toml
toolsets = ["core", "ossm"]

[toolset_configs.kiali]
url = "https://kiali.example" # Endpoint/route to reach the Kiali console
# insecure = true  # optional: allow insecure TLS (not recommended in production)
# certificate_authority = "/path/to/ca.crt"  # File path to CA certificate
```

> **Note:** the toolset is *exposed* as `ossm`, but its configuration section
> keeps the upstream parser key `[toolset_configs.kiali]`.

For the full list of tools, configuration options, URL discovery, reachability
checks, and multi-cluster setup, see [docs/KIALI.md](KIALI.md).
