package toolset

import (
	"slices"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/toolsets"
	"github.com/rhobs/obs-mcp/pkg/toolset/tools"
)

// Toolset implements the observability toolset for advanced Prometheus monitoring.
type Toolset struct{}

var _ api.Toolset = (*Toolset)(nil)

// GetName returns the name of the toolset.
func (t *Toolset) GetName() string {
	return "obs-mcp"
}

// GetDescription returns a human-readable description of the toolset.
func (t *Toolset) GetDescription() string {
	return `Advanced observability tools for comprehensive Prometheus metrics querying with guardrails and discovery features.

## MANDATORY WORKFLOW - ALWAYS FOLLOW THIS ORDER

**STEP 1: ALWAYS call list_metrics FIRST**
- This is NON-NEGOTIABLE for EVERY question
- NEVER skip this step, even if you think you know the metric name
- NEVER guess metric names - they vary between environments
- Search the returned list to find the exact metric name that exists

**STEP 2: Call get_label_names for the metric you found**
- Discover available labels for filtering (namespace, pod, service, etc.)

**STEP 3: Call get_label_values if you need specific filter values**
- Find exact label values (e.g., actual namespace names, pod names)

**STEP 4: Execute your query using the EXACT metric name from Step 1**
- Use execute_instant_query for current state questions
- Use execute_range_query for trends/historical analysis

## CRITICAL RULES

1. **NEVER query a metric without first calling list_metrics** - You must verify the metric exists
2. **Use EXACT metric names from list_metrics output** - Do not modify or guess metric names
3. **If list_metrics doesn't return a relevant metric, tell the user** - Don't fabricate queries
4. **BE PROACTIVE** - Complete all steps automatically without asking for confirmation. When you find a relevant metric, proceed to query.
5. **UNDERSTAND TIME FRAMES** - Use the start and end parameters to specify the time frame for your queries. You can use NOW for current time liberally across parameters, and NOW±duration for relative time frames.

## Query Type Selection

- **execute_instant_query**: Current values, point-in-time snapshots, "right now" questions
- **execute_range_query**: Trends over time, rate calculations, historical analysis`
}

// GetTools returns all tools provided by this toolset.
func (t *Toolset) GetTools(_ api.Openshift) []api.ServerTool {
	return slices.Concat(
		tools.InitListMetrics(),
		tools.InitExecuteInstantQuery(),
		tools.InitExecuteRangeQuery(),
		tools.InitGetLabelNames(),
		tools.InitGetLabelValues(),
		tools.InitGetSeries(),
	)
}

// GetPrompts returns prompts provided by this toolset.
func (t *Toolset) GetPrompts() []api.ServerPrompt {
	// Currently, prompts are not supported through this toolset
	// The workflow instructions are embedded in the tool descriptions
	return nil
}

func init() {
	toolsets.Register(&Toolset{})
}
