package types

import (
	k8stypes "github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/kubernetes/types"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/headtail"
	"github.com/ovn-kubernetes/ovn-kubernetes-mcp/pkg/utils/pattern"
)

// Database represents an OVN database type.
type Database string

const (
	// NorthboundDB is the OVN Northbound database.
	NorthboundDB Database = "nbdb"
	// SouthboundDB is the OVN Southbound database.
	SouthboundDB Database = "sbdb"
)

// ShowParams are the parameters for ovn-nbctl/ovn-sbctl show command.
type ShowParams struct {
	k8stypes.NamespacedNameParams
	Database Database `json:"database"`
	headtail.HeadTailParams
}

// ShowResult contains the output of ovn-nbctl/ovn-sbctl show command.
type ShowResult struct {
	Database Database `json:"database"`
	Output   string   `json:"output"`
}

// LogicalFlowListParams are the parameters for listing logical flows from SBDB.
type LogicalFlowListParams struct {
	k8stypes.NamespacedNameParams
	Datapath string `json:"datapath,omitempty"`
	pattern.PatternParams
	headtail.HeadTailParams
}

// LogicalFlowListResult contains the list of logical flows.
type LogicalFlowListResult struct {
	Datapath string   `json:"datapath,omitempty"`
	Flows    []string `json:"flows"`
}

// TraceMode represents the output verbosity mode for ovn-trace.
type TraceMode string

const (
	// TraceModeDetailed shows detailed trace output (default).
	TraceModeDetailed TraceMode = "detailed"
	// TraceModeSummary shows summary output only.
	TraceModeSummary TraceMode = "summary"
	// TraceModeMinimal shows minimal output.
	TraceModeMinimal TraceMode = "minimal"
)

// OVNTraceParams are the parameters for ovn-trace command.
type OVNTraceParams struct {
	k8stypes.NamespacedNameParams
	Datapath  string    `json:"datapath"`
	Microflow string    `json:"microflow"`
	Mode      TraceMode `json:"mode,omitempty"` // Output mode: detailed (default), summary, or minimal
	pattern.PatternParams
	headtail.HeadTailParams
}

// OVNTraceResult contains the output of ovn-trace command.
type OVNTraceResult struct {
	Datapath  string `json:"datapath"`
	Microflow string `json:"microflow"`
	Output    string `json:"output"`
}

// GetParams are the parameters for querying records from an OVN table.
// This is a flexible command that supports:
// - Listing all records (when Record is empty)
// - Getting a specific record (when Record is set)
// - Getting specific columns (when Columns is set)
type GetParams struct {
	k8stypes.NamespacedNameParams
	Database Database `json:"database"`
	Table    string   `json:"table"`
	Record   string   `json:"record,omitempty"`  // Optional: if empty, lists all records
	Columns  string   `json:"columns,omitempty"` // Optional: comma-separated columns to retrieve
	pattern.PatternParams
	headtail.HeadTailParams
}

// GetResult contains the output of ovn-nbctl/ovn-sbctl query.
type GetResult struct {
	Database Database `json:"database"`
	Table    string   `json:"table"`
	Record   string   `json:"record,omitempty"`
	Output   string   `json:"output"`
}
