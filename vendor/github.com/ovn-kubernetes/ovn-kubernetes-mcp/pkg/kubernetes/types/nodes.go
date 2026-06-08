package types

// DebugNodeParams is a type that contains the name, image and command of a node.
type DebugNodeParams struct {
	Name      string   `json:"name"`
	Image     string   `json:"image"`
	Command   []string `json:"command"`
	HostPath  string   `json:"host_path,omitempty"`
	MountPath string   `json:"mount_path,omitempty"`
}

// DebugNodeResult is a type that contains the stdout and stderr of the executed command.
type DebugNodeResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}
