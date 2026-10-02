package config

import "github.com/containers/kubernetes-mcp-server/pkg/api"

var toolsetConfigRegistry = newExtendedConfigRegistry()

func RegisterToolsetConfig(name string, parser ExtendedConfigParser) {
	toolsetConfigRegistry.register(name, parser)
}

// ToolsetConfigCommitter publishes a toolset's committed configuration to the
// live state its handlers read. It is invoked with an accepted server
// configuration (initial load and every successful reload), never with a
// candidate config that may still be rejected. Implementations must handle the
// absent case (their section missing from cfg) by clearing any previously
// published state so a present-to-absent reload stops serving stale values.
//
// It exists for handlers that cannot reach the request-scoped toolset config
// (e.g. MCP resource handlers, whose signature carries only a context), so they
// read a package-global instead.
type ToolsetConfigCommitter func(cfg api.BaseConfig)

var toolsetConfigCommitters []ToolsetConfigCommitter

// CommitToolsetConfig registers a committer invoked by
// CommitToolsetConfigs after the server accepts a configuration. Toolsets that
// keep live state derived from their config (e.g. via an atomic pointer read by
// handlers) register here so the state is published on commit rather than as a
// side effect of parsing, which parses candidate configs that may be rejected.
func CommitToolsetConfig(committer ToolsetConfigCommitter) {
	toolsetConfigCommitters = append(toolsetConfigCommitters, committer)
}

// CommitToolsetConfigs invokes every registered committer with the committed
// configuration. Callers must invoke it only after the configuration has been
// accepted and installed, so that rejected reloads never publish new state.
func CommitToolsetConfigs(cfg api.BaseConfig) {
	for _, committer := range toolsetConfigCommitters {
		committer(cfg)
	}
}
