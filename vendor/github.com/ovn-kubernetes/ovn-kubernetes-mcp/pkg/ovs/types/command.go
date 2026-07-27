package types

import (
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/headtail"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/pattern"
)

// VsctlAction selects which ovs-vsctl subcommand the consolidated tool runs.
type VsctlAction string

const (
	VsctlShow       VsctlAction = "show"
	VsctlListBr     VsctlAction = "list-br"
	VsctlListPorts  VsctlAction = "list-ports"
	VsctlListIfaces VsctlAction = "list-ifaces"
)

// OfctlAction selects which ovs-ofctl subcommand the consolidated tool runs.
type OfctlAction string

const (
	OfctlDumpFlows OfctlAction = "dump-flows"
)

// AppctlAction selects which ovs-appctl subcommand the consolidated tool runs.
type AppctlAction string

const (
	AppctlDumpConntrack AppctlAction = "dpctl/dump-conntrack"
	AppctlOfprotoTrace  AppctlAction = "ofproto/trace"
)

// VsctlParams are the parameters for the consolidated ovs-vsctl tool. The
// Action field selects the subcommand to run. Bridge is required when Action
// is "list-ports" or "list-ifaces". HeadTailParams is only applied when Action
// is "show".
type VsctlParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Action    string `json:"action"`
	Bridge    string `json:"bridge,omitempty"`
	headtail.HeadTailParams
}

// VsctlResult holds the response of the consolidated ovs-vsctl tool. Only the
// field(s) relevant to the invoked action are populated.
type VsctlResult struct {
	Output     string   `json:"output,omitempty"`     // populated for action="show"
	Bridges    []string `json:"bridges,omitempty"`    // populated for action="list-br"
	Ports      []string `json:"ports,omitempty"`      // populated for action="list-ports"
	Interfaces []string `json:"interfaces,omitempty"` // populated for action="list-ifaces"
}

// OfctlParams are the parameters for the consolidated ovs-ofctl tool. The
// Action field selects the subcommand to run.
type OfctlParams struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Action    string `json:"action"`
	Bridge    string `json:"bridge"`
	pattern.PatternParams
	headtail.HeadTailParams
}

// OfctlResult holds the response of the consolidated ovs-ofctl tool.
type OfctlResult struct {
	Bridge string   `json:"bridge,omitempty"`
	Flows  []string `json:"flows,omitempty"` // populated for action="dump-flows"
}

// AppctlParams are the parameters for the consolidated ovs-appctl tool. The
// Action field selects the subcommand to run. Bridge and Flow are required
// when Action is "ofproto/trace". AdditionalParams is only used when Action is
// "dpctl/dump-conntrack".
type AppctlParams struct {
	Namespace        string   `json:"namespace"`
	Name             string   `json:"name"`
	Action           string   `json:"action"`
	Bridge           string   `json:"bridge,omitempty"`
	Flow             string   `json:"flow,omitempty"`
	AdditionalParams []string `json:"additional_params,omitempty"`
	pattern.PatternParams
	headtail.HeadTailParams
}

// AppctlResult holds the response of the consolidated ovs-appctl tool. Only
// the field(s) relevant to the invoked action are populated.
type AppctlResult struct {
	Entries []string `json:"entries,omitempty"` // populated for action="dpctl/dump-conntrack"
	Bridge  string   `json:"bridge,omitempty"`  // populated for action="ofproto/trace"
	Flow    string   `json:"flow,omitempty"`    // populated for action="ofproto/trace"
	Output  string   `json:"output,omitempty"`  // populated for action="ofproto/trace"
}
