package incidents

import (
	"context"

	"github.com/BurntSushi/toml"
	serverconfig "github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets/metrics"
)

func toolsetConfigParser(_ context.Context, primitive toml.Primitive, md toml.MetaData) (serverconfig.ExtendedConfig, error) {
	var cfg metrics.Config
	if err := md.PrimitiveDecode(primitive, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func init() {
	serverconfig.RegisterToolsetConfig(ToolsetName, toolsetConfigParser)
}
