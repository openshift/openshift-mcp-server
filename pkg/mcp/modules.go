package mcp

import (
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/cluster-diagnostics"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/config"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/core"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/helm"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/kcp"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/kiali"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/kubevirt"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/netobserv"
	_ "github.com/containers/kubernetes-mcp-server/pkg/toolsets/tekton"
	_ "github.com/rhobs/obs-mcp/pkg/toolset"
)
