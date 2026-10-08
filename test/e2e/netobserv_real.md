# NetObserv real-plugin E2E test

`TestNetObservReal` runs against a live NetObserv operator, FlowCollector, and Loki installation on an OpenShift cluster. It is skipped unless `NETOBSERV_OPERATOR=deploy` is set. The test reuses an existing ready FlowCollector when available; otherwise, it installs the required operator and creates a test FlowCollector.

Prerequisites: OpenShift cluster access, `kubectl`, `helm`, and a pullable MCP server image. The FlowCollector must have Loki enabled in Monolithic mode with demo Loki installed. The runner needs cluster-admin permissions when the test must install the operator, because that setup creates a cluster-scoped ImageDigestMirrorSet.

Run the test with:

```sh
NETOBSERV_OPERATOR=deploy \\
MCP_SERVER_IMAGE=registry.example.com/project/kubernetes-mcp-server:tag \\
go test -tags e2e -run '^TestNetObservReal$' -v -count=1 ./test/e2e/
```

The test deploys the MCP server in `e2e-netobserv-real`. When it installs the operator, cleanup removes the test FlowCollector, subscription, operator group and namespaces, CatalogSource, and ImageDigestMirrorSet.
