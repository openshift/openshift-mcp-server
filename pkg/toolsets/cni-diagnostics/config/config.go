package config

import (
	"context"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
)

const (
	DefaultNetshootImage = "nicolaka/netshoot:v0.16"
	DefaultPwruImage     = "docker.io/cilium/pwru:v1.0.10"
)

// Config holds CNI Diagnostics toolset configuration
type Config struct {
	// KernelDebugImage is the container image used for kernel tools (conntrack, iptables, nft, ip)
	KernelDebugImage string `toml:"kernel_debug_image,omitempty"`

	// TcpdumpImage is the container image used for tcpdump packet capture
	TcpdumpImage string `toml:"tcpdump_image,omitempty"`

	// PwruImage is the container image used for pwru eBPF packet tracing
	PwruImage string `toml:"pwru_image,omitempty"`
}

var _ api.ExtendedConfig = (*Config)(nil)

// Validate validates the CNI Diagnostics toolset configuration
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("cni-diagnostics config is nil")
	}

	// Image names are validated at runtime when creating debug pods
	// No specific validation needed here
	return nil
}

// GetConfig returns the CNI Diagnostics toolset configuration from the tool handler parameters
func GetConfig(params api.ToolHandlerParams) *Config {
	if toolsetCfg, ok := params.GetToolsetConfig("cni-diagnostics"); ok {
		if cniCfg, ok := toolsetCfg.(*Config); ok {
			return cniCfg
		}
	}
	return nil
}

// cnidiagnosticsToolsetParser parses the CNI Diagnostics toolset configuration from TOML
func cnidiagnosticsToolsetParser(_ context.Context, primitive toml.Primitive, md toml.MetaData) (api.ExtendedConfig, error) {
	var cfg Config
	if err := md.PrimitiveDecode(primitive, &cfg); err != nil {
		return nil, err
	}
	// Set default images if not provided
	if cfg.KernelDebugImage == "" {
		cfg.KernelDebugImage = DefaultNetshootImage
	}
	if cfg.TcpdumpImage == "" {
		cfg.TcpdumpImage = DefaultNetshootImage
	}
	if cfg.PwruImage == "" {
		cfg.PwruImage = DefaultPwruImage
	}
	return &cfg, nil
}

func init() {
	config.RegisterToolsetConfig("cni-diagnostics", cnidiagnosticsToolsetParser)
}
