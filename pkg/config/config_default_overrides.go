package config

import "github.com/containers/kubernetes-mcp-server/pkg/api"

func defaultOverrides() StaticConfig {
	return StaticConfig{
		// IMPORTANT: this file is used to override default config values in downstream builds.
		// For current release we want to just expose the settings below:
		ReadOnly:           true,
		DisableDestructive: true,
		Port:               "8080",
		DeniedResources: []api.GroupVersionKind{
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
		},
		DisabledTools: []string{"configuration_view"},
		Toolsets:      []string{"core", "config"},
		ToolOverrides: map[string]ToolOverride{
			"resources_create_or_update": {
				Description: "Create or update a Kubernetes resource in the current cluster by providing a YAML or JSON representation of the resource.\n" +
					"IMPORTANT: For Pod resources, you MUST set spec.securityContext and spec.containers[*].securityContext. " +
					"For workload resources (Deployment, StatefulSet, Job), you MUST set spec.template.spec.securityContext and spec.template.spec.containers[*].securityContext. " +
					"Use a non-root security context: set runAsNonRoot: true, allowPrivilegeEscalation: false, and drop ALL capabilities (capabilities: {drop: [\"ALL\"]}). " +
					"Omitting the SecurityContext will cause pod scheduling failures on OpenShift and other clusters that enforce restricted security policies.",
			},
		},
	}
}
