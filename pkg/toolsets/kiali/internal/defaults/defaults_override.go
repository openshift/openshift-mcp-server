package defaults

const (
	toolsetNameOverride        = "ossm"
	toolsetDescriptionOverride = "Most common tools for managing OSSM, check the [OSSM documentation](https://github.com/openshift/openshift-mcp-server/blob/main/docs/OSSM.md) for more details."
)

func ToolsetNameOverride() string {
	return toolsetNameOverride
}

func ToolsetDescriptionOverride() string {
	return toolsetDescriptionOverride
}
