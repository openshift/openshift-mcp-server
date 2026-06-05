package types

import (
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/headtail"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/pattern"
)

// GetPodLogsParams is a type that contains the name, namespace and container of a pod.
type GetPodLogsParams struct {
	NamespacedNameParams
	Container string `json:"container,omitempty"`
	Previous  bool   `json:"previous,omitempty"`
	pattern.PatternParams
	headtail.HeadTailParams
}

// GetPodLogsResult is a type that contains the logs of a pod where each log line
// is a separate element in the string slice.
type GetPodLogsResult struct {
	Logs []string `json:"logs"`
}

// ExecPodParams is a type that contains the name, namespace and container of a pod.
type ExecPodParams struct {
	NamespacedNameParams
	Container string   `json:"container,omitempty"`
	Command   []string `json:"command"`
}

// ExecPodResult is a type that contains the stdout and stderr of the executed command.
type ExecPodResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
}
