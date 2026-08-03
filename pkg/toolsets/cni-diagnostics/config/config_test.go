package config

import (
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

func (s *ConfigSuite) TestConfigDefaults() {
	s.Run("empty config has empty fields", func() {
		cfg := Config{}
		s.Equal("", cfg.KernelDebugImage)
		s.Equal("", cfg.TcpdumpImage)
		s.Equal("", cfg.PwruImage)
	})
}

func (s *ConfigSuite) TestConfigParsing() {
	s.Run("parses kernel_debug_image", func() {
		tomlStr := `kernel_debug_image = "registry.example.com/toolbox:1.0"`
		var cfg Config
		err := toml.Unmarshal([]byte(tomlStr), &cfg)
		s.NoError(err)
		s.Equal("registry.example.com/toolbox:1.0", cfg.KernelDebugImage)
	})

	s.Run("parses tcpdump_image", func() {
		tomlStr := `tcpdump_image = "nicolaka/netshoot:v0.14"`
		var cfg Config
		err := toml.Unmarshal([]byte(tomlStr), &cfg)
		s.NoError(err)
		s.Equal("nicolaka/netshoot:v0.14", cfg.TcpdumpImage)
	})

	s.Run("parses pwru_image", func() {
		tomlStr := `pwru_image = "cilium/pwru:v1.1.0"`
		var cfg Config
		err := toml.Unmarshal([]byte(tomlStr), &cfg)
		s.NoError(err)
		s.Equal("cilium/pwru:v1.1.0", cfg.PwruImage)
	})

	s.Run("parses all images together", func() {
		tomlStr := `
kernel_debug_image = "registry.example.com/toolbox:1.0"
tcpdump_image = "nicolaka/netshoot:v0.14"
pwru_image = "cilium/pwru:v1.1.0"
`
		var cfg Config
		err := toml.Unmarshal([]byte(tomlStr), &cfg)
		s.NoError(err)
		s.Equal("registry.example.com/toolbox:1.0", cfg.KernelDebugImage)
		s.Equal("nicolaka/netshoot:v0.14", cfg.TcpdumpImage)
		s.Equal("cilium/pwru:v1.1.0", cfg.PwruImage)
	})

	s.Run("handles partial config", func() {
		tomlStr := `kernel_debug_image = "registry.example.com/toolbox:1.0"`
		var cfg Config
		err := toml.Unmarshal([]byte(tomlStr), &cfg)
		s.NoError(err)
		s.Equal("registry.example.com/toolbox:1.0", cfg.KernelDebugImage)
		s.Equal("", cfg.TcpdumpImage)
		s.Equal("", cfg.PwruImage)
	})
}

func (s *ConfigSuite) TestConfigValidation() {
	s.Run("validates all fields present", func() {
		cfg := Config{
			KernelDebugImage: "toolbox:latest",
			TcpdumpImage:     "netshoot:latest",
			PwruImage:        "pwru:latest",
		}
		err := cfg.Validate()
		s.NoError(err)
	})

	s.Run("validates partial config", func() {
		cfg := Config{
			KernelDebugImage: "toolbox:latest",
		}
		err := cfg.Validate()
		s.NoError(err)
	})

	s.Run("validates empty config", func() {
		cfg := Config{}
		err := cfg.Validate()
		s.NoError(err)
	})
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
