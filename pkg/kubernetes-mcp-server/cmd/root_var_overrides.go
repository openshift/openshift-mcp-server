package cmd

// IMPORTANT: this file is used to override default variable values in downstream builds.

// varOverrides is invoked early in the command bootstrapping process, allowing
// downstreams to rename things like env var keys.
func varOverrides() {
	config_path_env_var = "OCP_MCP_CONFIG_PATH"
}
