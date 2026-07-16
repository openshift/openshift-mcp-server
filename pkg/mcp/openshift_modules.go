package mcp

import (
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/cluster-diagnostics"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/mustgather"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/netedge"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/oadp"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/openshift"
	_ "github.com/rhobs/obs-mcp/pkg/toolset"
)
