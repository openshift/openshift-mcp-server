package types

import "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/timeout"

// DebugNodeParams is a type that contains the namespace, name, image, command, host path, mount path
// of a node along with  timeout parameters to be used for the command execution.
type DebugNodeParams struct {
	NamespacedNameParams
	Image     string   `json:"image"`
	Command   []string `json:"command"`
	HostPath  string   `json:"host_path,omitempty"`
	MountPath string   `json:"mount_path,omitempty"`
	timeout.TimeoutParams
}

// DebugNodeResult is a type that contains the stdout and stderr of the executed command.
type DebugNodeResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}
