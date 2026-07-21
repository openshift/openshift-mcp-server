"""End-to-end tests for Kuadrant MCP Gateway integration.

These tests verify that the kubernetes-mcp-server is accessible through the
Kuadrant MCP Gateway, including tool federation with prefixes and
user-specific tool listing.

Prerequisites (installed via ``make kuadrant-setup``):
  - Gateway API CRDs
  - Istio (base + istiod)
  - Kuadrant MCP Gateway (controller + broker)
"""

from __future__ import annotations

import asyncio
import time

import pytest
from kubernetes_asyncio.client import CustomObjectsApi

# Prefix applied to all tools federated through the MCPServerRegistration.
TOOL_PREFIX = "k8s_"


async def _wait_for_registration_ready(
    custom_objects: CustomObjectsApi,
    namespace: str,
    name: str,
    timeout: float = 120.0,
) -> dict:
    """Poll until the MCPServerRegistration has a Ready=True condition."""
    deadline = time.monotonic() + timeout
    obj: dict = {}
    while time.monotonic() < deadline:
        obj = await custom_objects.get_namespaced_custom_object(
            group="mcp.kuadrant.io",
            version="v1alpha1",
            namespace=namespace,
            plural="mcpserverregistrations",
            name=name,
        )
        for cond in obj.get("status", {}).get("conditions", []):
            if cond.get("type") == "Ready" and cond.get("status") == "True":
                return obj
        await asyncio.sleep(2)
    raise TimeoutError(
        f"MCPServerRegistration {namespace}/{name} not ready within {timeout}s. "
        f"Last status: {obj.get('status', {})}"
    )


@pytest.mark.kuadrant
async def test_kuadrant_gateway_tools(
    deploy_server, k8s_core_v1, k8s_custom_objects, kuadrant_gateway,
):
    """Tools should be accessible and callable through the Kuadrant MCP Gateway."""
    # 1. Deploy the MCP server into the cluster with read-only config,
    #    RBAC granting cluster-wide read access, and an HTTPRoute to the
    #    Kuadrant MCP Gateway.  Using the chart's httpRoute template (instead
    #    of creating the route by hand) gives that template e2e coverage.
    #    The hostname uses the internal wildcard pattern (*.mcp.local) that
    #    the gateway's "mcps" listener accepts for broker-to-server hairpin
    #    routing.
    server = await deploy_server("kuadrant-gw", """
        read_only = true
    """, extra_values={
        "rbac": {
            "create": True,
            "extraClusterRoles": [{
                "name": "cluster-reader",
                "rules": [{
                    "apiGroups": [""],
                    "resources": ["namespaces", "nodes"],
                    "verbs": ["get", "list", "watch"],
                }],
            }],
            "extraClusterRoleBindings": [
                {
                    "name": "view",
                    "roleRef": {
                        "name": "view",
                        "external": True,
                    },
                },
                {
                    "name": "cluster-reader",
                    "roleRef": {
                        "name": "cluster-reader",
                    },
                },
            ],
        },
        "httpRoute": {
            "enabled": True,
            "parentRefs": [{
                "name": "mcp-gateway",
                "namespace": "gateway-system",
            }],
            "hostnames": ["{{ .Release.Name }}.mcp.local"],
            "rules": [{
                "matches": [
                    {"path": {"type": "PathPrefix", "value": "/"}},
                ],
            }],
        },
    })

    # 2. Create an MCPServerRegistration that registers our MCP server with
    #    the gateway.  userSpecificList is Enabled so the broker forwards
    #    per-user credentials when listing tools (required for servers that
    #    enforce auth, which we will layer on in a follow-up).
    await k8s_custom_objects.create_namespaced_custom_object(
        group="mcp.kuadrant.io",
        version="v1alpha1",
        namespace=server.namespace,
        plural="mcpserverregistrations",
        body={
            "apiVersion": "mcp.kuadrant.io/v1alpha1",
            "kind": "MCPServerRegistration",
            "metadata": {
                "name": f"{server.name}-reg",
                "namespace": server.namespace,
            },
            "spec": {
                "prefix": TOOL_PREFIX,
                "userSpecificList": "Enabled",
                "targetRef": {
                    "group": "gateway.networking.k8s.io",
                    "kind": "HTTPRoute",
                    "name": server.name,
                },
            },
        },
    )

    # 3. Wait for the registration to become Ready (controller has connected
    #    the broker to our server and discovered its tools).
    await _wait_for_registration_ready(
        k8s_custom_objects, server.namespace, f"{server.name}-reg",
    )

    # 4. Connect to the MCP Gateway and verify tools.
    async with kuadrant_gateway.connect_mcp() as session:
        # Verify that tools from our server appear with the prefix.
        result = await session.list_tools()
        tool_names = [t.name for t in result.tools]
        prefixed = [n for n in tool_names if n.startswith(TOOL_PREFIX)]
        assert len(prefixed) > 0, (
            f"expected tools with prefix '{TOOL_PREFIX}', "
            f"got: {tool_names}"
        )

        # 5. Call namespaces_list through the gateway and verify the
        #    response against a direct Kubernetes API call.
        namespaces_tool = f"{TOOL_PREFIX}namespaces_list"
        assert namespaces_tool in tool_names, (
            f"expected '{namespaces_tool}' in tool list, got: {tool_names}"
        )
        call_result = await session.call_tool(namespaces_tool, {})
        assert len(call_result.content) > 0, (
            f"expected content from {namespaces_tool}, got empty response"
        )
        tool_output = call_result.content[0].text

        ns_list = await k8s_core_v1.list_namespace()
        ns_names = [ns.metadata.name for ns in ns_list.items]
        for ns in ns_names:
            assert ns in tool_output, (
                f"expected server namespace '{server.namespace}' in tool output, "
                f"got: {tool_output[:500]}"
            )
