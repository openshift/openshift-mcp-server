package config

// defaultOverrides is invoked from New after Option metadata is installed and
// before values are reset to Default. Downstream builds replace this file to
// change Option defaults and spellings (TOML / env names) plus
// ConfigPathEnvName.
func defaultOverrides(c *Config) {
	// IMPORTANT: this file is used to override default config values in downstream builds.
	// For current release we want to just expose the settings below:
	c.ReadOnly.Default = true
	c.DisableDestructive.Default = true
	c.Port.Default = "8080"
	c.DeniedResources.Default = []GroupVersionKind{
		{
			Group:   "",
			Version: "v1",
			Kind:    "ServiceAccount",
		},
		{
			Group:   "",
			Version: "v1",
			Kind:    "Secret",
		},
		{
			Group:   "rbac.authorization.k8s.io",
			Version: "v1",
		},
	}
	c.DisabledTools.Default = []string{"configuration_view"}
	c.Toolsets.Default = []string{"core", "config"}
	c.ToolOverrides.Default = map[string]ToolOverride{
		"resources_create_or_update": {
			Description: "Create or update a Kubernetes resource in the current cluster by providing a YAML or JSON representation of the resource.\n" +
				"IMPORTANT: For Pod resources, you MUST set spec.securityContext and spec.containers[*].securityContext. " +
				"For workload resources (Deployment, StatefulSet, Job), you MUST set spec.template.spec.securityContext and spec.template.spec.containers[*].securityContext. " +
				"Use a non-root security context: set runAsNonRoot: true, allowPrivilegeEscalation: false, and drop ALL capabilities (capabilities: {drop: [\"ALL\"]}). " +
				"Omitting the SecurityContext will cause pod scheduling failures on OpenShift and other clusters that enforce restricted security policies.",
		},
	}
}
