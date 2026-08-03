package defaults

const (
	toolsetDescriptionOverride               = "OpenShift Virtualization tools for managing virtual machines, check the [OpenShift Virtualization documentation](https://github.com/openshift/openshift-mcp-server/blob/main/docs/kubevirt.md) for more details."
	productNameOverride                      = "OpenShift Virtualization"
	windowsEFIInstallerTektonCatalogOverride = ""
)

func ToolsetDescriptionOverride() string {
	return toolsetDescriptionOverride
}

func ProductNameOverride() string {
	return productNameOverride
}

func WindowsEFIInstallerTektonCatalogOverride() string {
	return windowsEFIInstallerTektonCatalogOverride
}
