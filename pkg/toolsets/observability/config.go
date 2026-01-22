package observability

import (
	"context"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
)

// Config holds observability toolset configuration
type Config struct {
	// MonitoringNamespace is the namespace where monitoring components are deployed.
	// Defaults to "openshift-monitoring" if not specified.
	MonitoringNamespace string `toml:"monitoring_namespace,omitempty"`
}

var _ api.ExtendedConfig = (*Config)(nil)

// Validate checks that the configuration values are valid.
func (c *Config) Validate() error {
	// All fields are optional with sensible defaults, no validation required
	return nil
}

func observabilityToolsetParser(_ context.Context, primitive toml.Primitive, md toml.MetaData) (api.ExtendedConfig, error) {
	var cfg Config
	if err := md.PrimitiveDecode(primitive, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func init() {
	config.RegisterToolsetConfig("observability", observabilityToolsetParser)
}
