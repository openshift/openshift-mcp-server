package mustgather

import (
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/stretchr/testify/suite"
)

type ConfigSuite struct {
	suite.Suite
}

func (s *ConfigSuite) SetupTest() {
	current.Store(nil)
}

func (s *ConfigSuite) TearDownTest() {
	current.Store(nil)
}

func (s *ConfigSuite) TestConfigParser_ParsesDirs() {
	cfg := test.Must(config.ReadToml([]byte(`
		[toolset_configs."openshift/mustgather"]
		mustgather_dirs = ["/var/data/must-gather", "/home/user/downloads/must-gather.local.123"]
	`)))

	mgCfg, ok := cfg.GetToolsetConfig("openshift/mustgather")
	s.Require().True(ok, "mustgather config should be present")
	mgc, ok := mgCfg.(*Config)
	s.Require().True(ok, "mustgather config should be of type *Config")
	s.Equal([]string{"/var/data/must-gather", "/home/user/downloads/must-gather.local.123"}, mgc.MustGatherDirs)

	s.Equal(mgc.MustGatherDirs, toolsetDirs(), "live accessor should reflect the parsed config")
}

func (s *ConfigSuite) TestConfigParser_EmptySection() {
	test.Must(config.ReadToml([]byte(`
		[toolset_configs."openshift/mustgather"]
	`)))

	s.Empty(toolsetDirs(), "no directories should be configured for an empty section")
}

func (s *ConfigSuite) TestToolsetDirs_Unconfigured() {
	s.Nil(toolsetDirs(), "toolsetDirs should be nil when no toolset config was loaded")
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
