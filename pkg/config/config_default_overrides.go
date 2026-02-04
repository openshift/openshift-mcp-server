package config

func defaultOverrides() StaticConfig {
	return StaticConfig{
		// IMPORTANT: this file is used to override default config values in downstream builds.
		// For current release we want to just expose the settings below:
		ReadOnly: true,
		Toolsets: []string{"core", "config"},
	}
}
