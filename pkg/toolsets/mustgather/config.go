package mustgather

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
)

// Config holds the openshift/mustgather toolset configuration.
type Config struct {
	// MustGatherDirs is the list of directories scanned for must-gather
	// archives. Each entry may be a directory containing one or more
	// archives, or a directory that is itself an archive.
	MustGatherDirs []string `toml:"mustgather_dirs,omitempty"`
}

var _ api.ExtendedConfig = (*Config)(nil)

// Validate validates the openshift/mustgather toolset configuration.
func (c *Config) Validate() error {
	if c == nil {
		return errors.New("mustgather config is nil")
	}
	return nil
}

// current holds the last parsed toolset configuration. mustgatherToolsetParser
// updates it every time the server config is loaded or reloaded (SIGHUP), so
// tool and resource handlers observe the live value. If the
// [toolset_configs."openshift/mustgather"] section is removed on a reload, the
// last parsed value remains until the section is loaded again.
var current atomic.Pointer[Config]

// toolsetDirs returns the directories scanned for must-gather archives, or
// nil if the toolset is not configured.
func toolsetDirs() []string {
	if c := current.Load(); c != nil {
		return c.MustGatherDirs
	}
	return nil
}

// mustgatherToolsetParser parses the openshift/mustgather toolset
// configuration from TOML.
func mustgatherToolsetParser(_ context.Context, primitive toml.Primitive, md toml.MetaData) (api.ExtendedConfig, error) {
	var cfg Config
	if err := md.PrimitiveDecode(primitive, &cfg); err != nil {
		return nil, err
	}
	current.Store(&cfg)
	return &cfg, nil
}

func init() {
	config.RegisterToolsetConfig("openshift/mustgather", mustgatherToolsetParser)
}
