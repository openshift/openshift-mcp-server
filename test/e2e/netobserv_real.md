# NetObserv real-plugin E2E test

`TestNetObservReal` runs against a live NetObserv operator, FlowCollector, and Loki installation on an OpenShift cluster. It is skipped unless `NETOBSERV_OPERATOR=deploy` is set. The test reuses an existing ready FlowCollector when available. If no FlowCollector exists, it creates one; it installs the operator only when a ready operator is not already present.

Prerequisites: OpenShift cluster access, `kubectl`, `helm`, and a pullable MCP server image. The FlowCollector must have Loki enabled in Monolithic mode with demo Loki installed. The runner needs cluster-admin permissions when the test must install the operator, because that setup creates a cluster-scoped ImageDigestMirrorSet.

Run the test with:

```sh
NETOBSERV_OPERATOR=deploy \\
MCP_SERVER_IMAGE=registry.example.com/project/kubernetes-mcp-server:tag \\
go test -tags e2e -run '^TestNetObservReal$' -v -count=1 ./test/e2e/
```

The test deploys the MCP server in `e2e-netobserv-real`. When it installs the operator, cleanup attempts to remove the test FlowCollector, subscription, operator group, any namespaces created by the test, the CatalogSource, and the ImageDigestMirrorSet. Cleanup is best-effort.
