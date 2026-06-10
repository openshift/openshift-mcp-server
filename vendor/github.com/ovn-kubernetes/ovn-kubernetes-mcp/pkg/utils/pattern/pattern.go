package pattern

import (
	"fmt"
	"regexp"
)

type PatternParams struct {
	Pattern string `json:"pattern,omitempty"`
}

// ExecuteWithMatch executes a function and matches the output to the pattern.
// If checkMatch is false, a non-empty Pattern is rejected with an error because
// pattern matching is not supported in that mode; an empty Pattern returns f()
// unchanged. If checkMatch is true, it returns f() when Pattern is empty, or the
// regex-matched subset of f()'s lines when Pattern is set. An error is returned
// if the pattern is invalid or if f() fails.
func (p *PatternParams) ExecuteWithMatch(f func() ([]string, error), checkMatch bool) ([]string, error) {
	if !checkMatch {
		if p.Pattern != "" {
			return nil, fmt.Errorf("pattern matching is not supported in this mode (got pattern %q)", p.Pattern)
		}
		return f()
	}
	if p.Pattern == "" {
		return f()
	}
	searchPattern, err := regexp.Compile(p.Pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid search pattern %q: %w", p.Pattern, err)
	}

	lines, err := f()
	if err != nil {
		return nil, err
	}

	matchedLines := []string{}
	for _, line := range lines {
		if searchPattern.MatchString(line) {
			matchedLines = append(matchedLines, line)
		}
	}
	return matchedLines, nil
}
