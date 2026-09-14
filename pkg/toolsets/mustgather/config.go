package mustgather

import (
	"context"
	"errors"
	"sync/atomic"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
)

// Default limits applied when the toolset config does not override them.
const (
	// defaultTailLimit caps the number of lines a log tool keeps when a
	// caller requests tailing (the "tail" parameter). It bounds the slice
	// capacity allocated for the tail ring so a large tool-provided value
	// cannot trigger an excessive allocation.
	defaultTailLimit = 1000
	// defaultMaxOutputSize caps the aggregate size, in bytes, of assembled
	// log output returned by a single tool call.
	defaultMaxOutputSize = 1024 * 1024 // 1 MiB, roughly 200K tokens!
)

// Config holds the openshift/mustgather toolset configuration.
type Config struct {
	// MustGatherDirs is the list of directories scanned for must-gather
	// archives. Each entry may be a directory containing one or more
	// archives, or a directory that is itself an archive.
	MustGatherDirs []string `toml:"mustgather_dirs,omitempty"`
	// TailLimit caps the number of lines a log tool keeps when a caller
	// requests tailing. A tool-provided "tail" larger than this is capped to
	// this value, bounding the allocation for the tail ring. When unset (0),
	// defaultTailLimit is used.
	TailLimit int `toml:"tail_limit,omitempty"`
	// MaxOutputSize caps the aggregate size, in bytes, of assembled log
	// output returned by a single tool call. When unset (0),
	// defaultMaxOutputSize is used.
	MaxOutputSize int `toml:"max_output_size,omitempty"`
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

// toolsetTailLimit returns the configured cap on tail line counts, falling
// back to defaultTailLimit when the toolset is unconfigured or the value is
// unset/non-positive.
func toolsetTailLimit() int {
	if c := current.Load(); c != nil && c.TailLimit > 0 {
		return c.TailLimit
	}
	return defaultTailLimit
}

// toolsetMaxOutputSize returns the configured cap on aggregate log output
// size in bytes, falling back to defaultMaxOutputSize when the toolset is
// unconfigured or the value is unset/non-positive.
func toolsetMaxOutputSize() int {
	if c := current.Load(); c != nil && c.MaxOutputSize > 0 {
		return c.MaxOutputSize
	}
	return defaultMaxOutputSize
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
