package prometheus

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
)

// Guardrail name constants for use with ParseGuardrails
const (
	GuardrailDisallowExplicitNameLabel = "disallow-explicit-name-label"
	GuardrailRequireLabelMatcher       = "require-label-matcher"
	GuardrailDisallowBlanketRegex      = "disallow-blanket-regex"
	GuardrailMaxMetricCardinality      = "max-metric-cardinality"
	GuardrailMaxLabelCardinality       = "max-label-cardinality"
)

// Guardrails provides safety checks for PromQL queries based on configurable rules.
type Guardrails struct {
	// DisallowExplicitNameLabel prevents queries using explicit {__name__="..."} syntax
	DisallowExplicitNameLabel bool
	// RequireLabelMatcher ensures all vector selectors have at least one non-name label matcher
	RequireLabelMatcher bool
	// DisallowBlanketRegex prevents expensive regex patterns like .* or .+ on any label
	DisallowBlanketRegex bool
	// MaxMetricCardinality sets the maximum allowed series count per metric (0 = disabled)
	MaxMetricCardinality uint64
	// MaxLabelCardinality sets the maximum allowed label value count for blanket regex
	// (0 = always disallow regex matcher provided DisallowBlanketRegex is true)
	MaxLabelCardinality uint64
}

// DefaultGuardrails returns a Guardrails instance with all safety checks enabled.
func DefaultGuardrails() *Guardrails {
	return &Guardrails{
		DisallowExplicitNameLabel: true,
		RequireLabelMatcher:       true,
		DisallowBlanketRegex:      true,
		MaxMetricCardinality:      20000,
		MaxLabelCardinality:       500,
	}
}

func ParseGuardrails(value string) (*Guardrails, error) {
	value = strings.TrimSpace(value)

	switch strings.ToLower(value) {
	case "none":
		return nil, nil
	case "all", "":
		return DefaultGuardrails(), nil
	}

	g := &Guardrails{}
	names := strings.SplitSeq(value, ",")
	for name := range names {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" {
			continue
		}

		switch name {
		case GuardrailDisallowExplicitNameLabel:
			g.DisallowExplicitNameLabel = true
		case GuardrailRequireLabelMatcher:
			g.RequireLabelMatcher = true
		case GuardrailDisallowBlanketRegex:
			g.DisallowBlanketRegex = true
		default:
			return nil, fmt.Errorf("unknown guardrail: %q (valid options: %s, %s, %s)",
				name, GuardrailDisallowExplicitNameLabel, GuardrailRequireLabelMatcher,
				GuardrailDisallowBlanketRegex)
		}
	}

	return g, nil
}

// IsSafeQuery analyzes a PromQL query string and returns false if it's
// deemed unsafe or too expensive based on the configured rules.
// If client is provided and MaxMetricCardinality is set, it checks TSDB metric cardinality.
// If client is provided and MaxLabelCardinality is set, it checks TSDB label cardinality for blanket regex.
//
// Returns (false, error) if the query is invalid or violates a guardrail rule.
// The error message explains which rule was violated.
// Returns (true, nil) if the query is valid and passes all rules.
//
//nolint:gocyclo // complex validation logic, refactoring would reduce readability
func (g *Guardrails) IsSafeQuery(ctx context.Context, query string, client v1.API) (bool, error) {
	if ((g.DisallowBlanketRegex && g.MaxLabelCardinality > 0) || (g.MaxMetricCardinality > 0)) && (client == nil || ctx == nil) {
		return false, fmt.Errorf("cannot verify cardinality without TSDB client")
	}

	expr, err := parser.ParseExpr(query)
	if err != nil {
		return false, fmt.Errorf("failed to parse query: %w", err)
	}

	var unsafeReason error

	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		vs, ok := node.(*parser.VectorSelector)
		if !ok {
			return nil
		}

		// Check for explicit __name__ label query
		if g.DisallowExplicitNameLabel && vs.Name == "" {
			for _, m := range vs.LabelMatchers {
				if m.Name == labels.MetricName {
					unsafeReason = fmt.Errorf("query uses explicit __name__ label matcher, which is disallowed")
					return unsafeReason
				}
			}
		}

		// All vector selectors must have at least one non-name label matcher
		if g.RequireLabelMatcher {
			hasNonNameMatcher := false
			for _, m := range vs.LabelMatchers {
				if m.Name != labels.MetricName {
					hasNonNameMatcher = true
					break
				}
			}
			if !hasNonNameMatcher {
				unsafeReason = fmt.Errorf("query for metric %q does not have any label matchers, which is required", vs.Name)
				return unsafeReason
			}
		}

		return nil
	})

	if unsafeReason != nil {
		return false, unsafeReason
	}

	// Check metric cardinality
	if g.MaxMetricCardinality > 0 {
		metricNames, err := ExtractMetricNames(query)
		if err != nil {
			return false, fmt.Errorf("failed to extract metric names: %w", err)
		}

		if len(metricNames) > 0 {
			tsdbResult, err := client.TSDB(ctx)
			if err != nil {
				return false, fmt.Errorf("failed to get TSDB stats: %w", err)
			}

			seriesCountByMetric := make(map[string]uint64)
			for _, stat := range tsdbResult.SeriesCountByMetricName {
				seriesCountByMetric[stat.Name] = stat.Value
			}

			for _, metricName := range metricNames {
				if count, exists := seriesCountByMetric[metricName]; exists {
					if count > g.MaxMetricCardinality {
						return false, fmt.Errorf("metric %q has cardinality %d, which exceeds maximum allowed %d", metricName, count, g.MaxMetricCardinality)
					}
				}
			}
		}
	}

	// Check blanket regex patterns
	if g.DisallowBlanketRegex {
		blanketRegexLabels, err := ExtractBlanketRegexLabels(query)
		if err != nil {
			return false, fmt.Errorf("failed to extract blanket regex labels: %w", err)
		}

		if len(blanketRegexLabels) > 0 {
			// If MaxLabelCardinality is 0, always disallow blanket regex
			if g.MaxLabelCardinality == 0 {
				return false, fmt.Errorf("query uses blanket regex on label %q, which is disallowed", blanketRegexLabels[0])
			}

			// Check TSDB label cardinality for blanket regex
			tsdbResult, err := client.TSDB(ctx)
			if err != nil {
				return false, fmt.Errorf("failed to get TSDB stats: %w", err)
			}

			labelValueCountByLabel := make(map[string]uint64)
			for _, stat := range tsdbResult.LabelValueCountByLabelName {
				labelValueCountByLabel[stat.Name] = stat.Value
			}

			for _, labelName := range blanketRegexLabels {
				if count, exists := labelValueCountByLabel[labelName]; exists {
					if count > g.MaxLabelCardinality {
						return false, fmt.Errorf("label %q has cardinality %d, which exceeds maximum allowed %d for blanket regex", labelName, count, g.MaxLabelCardinality)
					}
				}
			}
		}
	}

	return true, nil
}

func ExtractMetricNames(query string) ([]string, error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	metricNames := make(map[string]bool)
	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if vs, ok := node.(*parser.VectorSelector); ok {
			if vs.Name != "" {
				metricNames[vs.Name] = true
			}
			// Also check for __name__ label matchers
			for _, m := range vs.LabelMatchers {
				if m.Name == labels.MetricName && m.Type == labels.MatchEqual {
					metricNames[m.Value] = true
				}
			}
		}
		return nil
	})

	result := make([]string, 0, len(metricNames))
	for name := range metricNames {
		result = append(result, name)
	}
	return result, nil
}

// ExtractBlanketRegexLabels extracts label names that use blanket regex patterns (.* or .+).
func ExtractBlanketRegexLabels(query string) ([]string, error) {
	expr, err := parser.ParseExpr(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	labelNames := make(map[string]bool)
	parser.Inspect(expr, func(node parser.Node, path []parser.Node) error {
		if vs, ok := node.(*parser.VectorSelector); ok {
			for _, m := range vs.LabelMatchers {
				isRegex := m.Type == labels.MatchRegexp || m.Type == labels.MatchNotRegexp
				if isRegex && (m.Value == ".*" || m.Value == ".+") {
					labelNames[m.Name] = true
				}
			}
		}
		return nil
	})

	result := make([]string, 0, len(labelNames))
	for name := range labelNames {
		result = append(result, name)
	}
	return result, nil
}
