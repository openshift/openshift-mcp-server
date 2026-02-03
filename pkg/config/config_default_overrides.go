package config

func defaultOverrides() StaticConfig {
	return StaticConfig{
		// IMPORTANT: this file is used to override default config values in downstream builds.
		// OpenShift-specific defaults: add openshift-core toolset
		Toolsets: []string{"core", "config", "helm", "openshift-core"},
	}
}
