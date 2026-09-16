package mustgather

import (
	"context"
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
	s.Require().NotNil(mgc.registry, "the parser should initialize an empty registry for every Config")
}

func (s *ConfigSuite) TestConfigParser_EmptySection() {
	cfg := test.Must(config.ReadToml([]byte(`
		[toolset_configs."openshift/mustgather"]
	`)))

	mgCfg, ok := cfg.GetToolsetConfig("openshift/mustgather")
	s.Require().True(ok, "mustgather config should be present")
	mgc, ok := mgCfg.(*Config)
	s.Require().True(ok, "mustgather config should be of type *Config")
	s.Empty(mgc.MustGatherDirs, "no directories should be configured for an empty section")
	s.Require().NotNil(mgc.registry, "the parser should initialize an empty registry even for an empty section")
}

func (s *ConfigSuite) TestConfigParser_FreshRegistryPerParse() {
	// Every parse (e.g. a server reload) must yield a distinct registry so a
	// reload starts from a clean cache instead of reusing the previous one.
	first := test.Must(config.ReadToml([]byte(`
		[toolset_configs."openshift/mustgather"]
		mustgather_dirs = ["/var/data/must-gather"]
	`)))
	second := test.Must(config.ReadToml([]byte(`
		[toolset_configs."openshift/mustgather"]
		mustgather_dirs = ["/var/data/must-gather"]
	`)))

	c1, _ := first.GetToolsetConfig("openshift/mustgather")
	c2, _ := second.GetToolsetConfig("openshift/mustgather")
	s.NotSame(c1.(*Config).registry, c2.(*Config).registry, "each parsed config should own a fresh registry")
}

func (s *ConfigSuite) TestLimitDefaults() {
	s.Run("nil config falls back to defaults", func() {
		var c *Config
		s.Equal(defaultTailLimit, c.tailLimit())
		s.Equal(defaultMaxOutputSize, c.maxOutputSize())
	})

	s.Run("unset values fall back to defaults", func() {
		c := &Config{}
		s.Equal(defaultTailLimit, c.tailLimit())
		s.Equal(defaultMaxOutputSize, c.maxOutputSize())
	})

	s.Run("configured values are honored", func() {
		c := &Config{TailLimit: 42, MaxOutputSize: 4096}
		s.Equal(42, c.tailLimit())
		s.Equal(4096, c.maxOutputSize())
	})
}

// TestCommit_PublishesConfigForResourceHandlers verifies that commit publishes
// the parsed toolset config to the package-global read by MCP resource handlers
// (which receive only a context), and that the same *Config instance the tools
// see is exposed — so resource handlers share the per-config archive registry.
func (s *ConfigSuite) TestCommit_PublishesConfigForResourceHandlers() {
	s.Nil(configFromContext(context.Background()), "no config should be published before commit")

	cfg := test.Must(config.ReadToml([]byte(`
		[toolset_configs."openshift/mustgather"]
		mustgather_dirs = ["/var/data/must-gather"]
	`)))
	commit(cfg)

	published := configFromContext(context.Background())
	s.Require().NotNil(published, "commit should publish the parsed config for resource handlers")
	s.Equal([]string{"/var/data/must-gather"}, published.MustGatherDirs)

	toolCfg, _ := cfg.GetToolsetConfig("openshift/mustgather")
	s.Same(toolCfg.(*Config), published, "resource handlers must observe the same *Config (and registry) as the tools")
}

// TestCommit_AbsentSectionClearsConfig verifies that a committed reload dropping
// the mustgather section stops resource handlers from serving stale config.
func (s *ConfigSuite) TestCommit_AbsentSectionClearsConfig() {
	commit(test.Must(config.ReadToml([]byte(`
		[toolset_configs."openshift/mustgather"]
		mustgather_dirs = ["/var/data/must-gather"]
	`))))
	s.Require().NotNil(configFromContext(context.Background()))

	commit(test.Must(config.ReadToml([]byte(`
		[toolset_configs."core"]
	`))))
	s.Nil(configFromContext(context.Background()), "dropping the section must clear the published config")
}

func TestConfig(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}
