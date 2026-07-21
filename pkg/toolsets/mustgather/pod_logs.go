package mustgather

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/google/jsonschema-go/jsonschema"
	"k8s.io/utils/ptr"
)

const maxScanLineSize = 1024 * 1024 // 1 MB
const maxOutputSize = 10 * 1024 * 1024

func initPodLogs() []api.ServerTool {
	return []api.ServerTool{
		{
			Tool: api.Tool{
				Name:        "mustgather_pod_logs_get",
				Description: "Get container logs for a specific pod from the must-gather archive. Returns current or previous logs.",
				Annotations: api.ToolAnnotations{
					Title:        "Get Pod Logs",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"path":      {Type: "string", Description: "Path to the must-gather archive directory (optional if mustgather_use was called earlier)"},
						"namespace": {Type: "string", Description: "Pod namespace"},
						"pod":       {Type: "string", Description: "Pod name"},
						"container": {Type: "string", Description: "Container name (uses first container if not specified)"},
						"previous":  {Type: "boolean", Description: "Get previous container logs (from crash/restart)"},
						"tail":      {Type: "integer", Description: "Number of lines from end of logs (0 for all)"},
					},
					Required: []string{"namespace", "pod"},
				},
			},
			Handler:      mustgatherPodLogsGet,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_pod_logs_grep",
				Description: "Filter pod container logs by a search string. Returns only matching lines from the must-gather archive.",
				Annotations: api.ToolAnnotations{
					Title:        "Pod Logs Grep",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"path":            {Type: "string", Description: "Path to the must-gather archive directory (optional if mustgather_use was called earlier)"},
						"namespace":       {Type: "string", Description: "Pod namespace"},
						"pod":             {Type: "string", Description: "Pod name"},
						"container":       {Type: "string", Description: "Container name (uses first container if not specified)"},
						"filter":          {Type: "string", Description: "String to search for in log lines"},
						"previous":        {Type: "boolean", Description: "Search previous container logs (from crash/restart)"},
						"tail":            {Type: "integer", Description: "Maximum number of matching lines to return (0 for all)"},
						"caseInsensitive": {Type: "boolean", Description: "Perform case-insensitive search (default: false)"},
					},
					Required: []string{"namespace", "pod", "filter"},
				},
			},
			Handler:      mustgatherPodLogsGrep,
			ClusterAware: ptr.To(false),
		},
		{
			Tool: api.Tool{
				Name:        "mustgather_pod_logs_by_time",
				Description: "Get pod container logs within a specific time range. Each log line is expected to have an RFC3339Nano timestamp prefix (from kubectl logs --timestamps).",
				Annotations: api.ToolAnnotations{
					Title:        "Pod Logs by Time",
					ReadOnlyHint: ptr.To(true),
				},
				InputSchema: &jsonschema.Schema{
					Type: "object",
					Properties: map[string]*jsonschema.Schema{
						"path":      {Type: "string", Description: "Path to the must-gather archive directory (optional if mustgather_use was called earlier)"},
						"namespace": {Type: "string", Description: "Pod namespace"},
						"pod":       {Type: "string", Description: "Pod name"},
						"container": {Type: "string", Description: "Container name (uses first container if not specified)"},
						"since":     {Type: "string", Description: "Start time in RFC3339 format (e.g. 2026-01-15T10:00:00Z)"},
						"until":     {Type: "string", Description: "End time in RFC3339 format (e.g. 2026-01-15T12:00:00Z)"},
						"previous":  {Type: "boolean", Description: "Search previous container logs (from crash/restart)"},
						"limit":     {Type: "integer", Description: "Maximum number of lines to return (default: 500)"},
					},
					Required: []string{"namespace", "pod", "since"},
				},
			},
			Handler:      mustgatherPodLogsByTime,
			ClusterAware: ptr.To(false),
		},
	}
}

func mustgatherPodLogsGet(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	path := getString(args, "path", "")
	p, err := InitProvider(params.Context, path)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := getString(args, "namespace", "")
	pod := getString(args, "pod", "")
	container := getString(args, "container", "")
	previous := getBool(args, "previous", false)
	tail := getInt(args, "tail", 0)

	if namespace == "" || pod == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace and pod are required")), nil
	}

	logType := mg.LogTypeCurrent
	if previous {
		logType = mg.LogTypePrevious
	}

	logPath, err := p.GetPodLogPath(mg.PodLogOptions{
		Namespace: namespace,
		Pod:       pod,
		Container: container,
		LogType:   logType,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get pod logs: %w", err)), nil
	}

	f, err := os.Open(logPath)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to open log file: %w", err)), nil
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxScanLineSize)

	var logs string
	if tail > 0 {
		ring := make([]string, 0, tail)
		var ringIdx int
		for scanner.Scan() {
			if len(ring) < tail {
				ring = append(ring, scanner.Text())
			} else {
				ring[ringIdx] = scanner.Text()
				ringIdx = (ringIdx + 1) % tail
			}
		}
		if len(ring) == tail {
			ordered := make([]string, tail)
			for i := range tail {
				ordered[i] = ring[(ringIdx+i)%tail]
			}
			ring = ordered
		}
		logs = strings.Join(ring, "\n")
	} else {
		var sb strings.Builder
		lineCount := 0
		truncated := false
		for scanner.Scan() {
			line := scanner.Text()
			if sb.Len()+len(line)+1 > maxOutputSize {
				sb.WriteString("\n... [output truncated at 10 MB, use 'tail' parameter to limit]")
				truncated = true
				break
			}
			if lineCount > 0 {
				sb.WriteByte('\n')
			}
			sb.WriteString(line)
			lineCount++
		}
		_ = truncated
		logs = sb.String()
	}

	if err := scanner.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to read log file: %w", err)), nil
	}

	header := fmt.Sprintf("Logs for pod %s/%s", namespace, pod)
	if container != "" {
		header += fmt.Sprintf(", container %s", container)
	}
	if previous {
		header += " (previous)"
	}
	if tail > 0 {
		header += fmt.Sprintf(" (last %d lines)", tail)
	}
	header += ":\n\n"

	return api.NewToolCallResult(header+logs, nil), nil
}

func mustgatherPodLogsGrep(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	path := getString(args, "path", "")
	p, err := InitProvider(params.Context, path)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := getString(args, "namespace", "")
	pod := getString(args, "pod", "")
	container := getString(args, "container", "")
	filter := getString(args, "filter", "")
	previous := getBool(args, "previous", false)
	tail := getInt(args, "tail", 0)
	caseInsensitive := getBool(args, "caseInsensitive", false)

	if namespace == "" || pod == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace and pod are required")), nil
	}
	if filter == "" {
		return api.NewToolCallResult("", fmt.Errorf("filter string is required")), nil
	}

	logType := mg.LogTypeCurrent
	if previous {
		logType = mg.LogTypePrevious
	}

	logPath, err := p.GetPodLogPath(mg.PodLogOptions{
		Namespace: namespace,
		Pod:       pod,
		Container: container,
		LogType:   logType,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get pod logs: %w", err)), nil
	}

	f, err := os.Open(logPath)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to open log file: %w", err)), nil
	}
	defer func() { _ = f.Close() }()

	searchFilter := filter
	if caseInsensitive {
		searchFilter = strings.ToLower(filter)
	}

	var matchingLines []string
	var ringIdx int
	totalMatches := 0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxScanLineSize)
	for scanner.Scan() {
		line := scanner.Text()
		compareLine := line
		if caseInsensitive {
			compareLine = strings.ToLower(line)
		}
		if !strings.Contains(compareLine, searchFilter) {
			continue
		}
		totalMatches++
		if tail > 0 {
			if len(matchingLines) < tail {
				matchingLines = append(matchingLines, line)
			} else {
				matchingLines[ringIdx] = line
				ringIdx = (ringIdx + 1) % tail
			}
		} else {
			matchingLines = append(matchingLines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to read log file: %w", err)), nil
	}

	if tail > 0 && len(matchingLines) == tail {
		ordered := make([]string, tail)
		for i := range tail {
			ordered[i] = matchingLines[(ringIdx+i)%tail]
		}
		matchingLines = ordered
	}

	header := fmt.Sprintf("Logs for pod %s/%s filtered by '%s'", namespace, pod, filter)
	if container != "" {
		header += fmt.Sprintf(", container %s", container)
	}
	if previous {
		header += " (previous)"
	}
	if caseInsensitive {
		header += " (case-insensitive)"
	}
	if tail > 0 {
		header += fmt.Sprintf(" (last %d matches)", tail)
	}
	header += fmt.Sprintf(":\n\nFound %d matching line(s)\n\n", totalMatches)

	if totalMatches == 0 {
		return api.NewToolCallResult(header+"No matching lines found.", nil), nil
	}

	return api.NewToolCallResult(header+strings.Join(matchingLines, "\n"), nil), nil
}

func mustgatherPodLogsByTime(params api.ToolHandlerParams) (*api.ToolCallResult, error) {
	args := params.GetArguments()
	path := getString(args, "path", "")
	p, err := InitProvider(params.Context, path)
	if err != nil {
		return api.NewToolCallResult("", err), nil
	}

	namespace := getString(args, "namespace", "")
	pod := getString(args, "pod", "")
	container := getString(args, "container", "")
	sinceStr := getString(args, "since", "")
	untilStr := getString(args, "until", "")
	previous := getBool(args, "previous", false)
	limit := getInt(args, "limit", 500)

	if namespace == "" || pod == "" {
		return api.NewToolCallResult("", fmt.Errorf("namespace and pod are required")), nil
	}
	if sinceStr == "" {
		return api.NewToolCallResult("", fmt.Errorf("since is required (RFC3339 format)")), nil
	}

	sinceTime, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("invalid since time format, expected RFC3339 (e.g. 2026-01-15T10:00:00Z): %w", err)), nil
	}

	var untilTime time.Time
	hasUntil := untilStr != ""
	if hasUntil {
		untilTime, err = time.Parse(time.RFC3339, untilStr)
		if err != nil {
			return api.NewToolCallResult("", fmt.Errorf("invalid until time format, expected RFC3339 (e.g. 2026-01-15T12:00:00Z): %w", err)), nil
		}
	}

	logType := mg.LogTypeCurrent
	if previous {
		logType = mg.LogTypePrevious
	}

	logPath, err := p.GetPodLogPath(mg.PodLogOptions{
		Namespace: namespace,
		Pod:       pod,
		Container: container,
		LogType:   logType,
	})
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to get pod logs: %w", err)), nil
	}

	f, err := os.Open(logPath)
	if err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to open log file: %w", err)), nil
	}
	defer func() { _ = f.Close() }()

	var matchingLines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, bufio.MaxScanTokenSize), maxScanLineSize)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		tsStr, _, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		lineTime, err := time.Parse(time.RFC3339Nano, tsStr)
		if err != nil {
			continue
		}
		if lineTime.Before(sinceTime) {
			continue
		}
		if hasUntil && lineTime.After(untilTime) {
			continue
		}
		matchingLines = append(matchingLines, line)
		if limit > 0 && len(matchingLines) >= limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return api.NewToolCallResult("", fmt.Errorf("failed to read log file: %w", err)), nil
	}

	header := fmt.Sprintf("Logs for pod %s/%s between %s and ", namespace, pod, sinceStr)
	if hasUntil {
		header += untilStr
	} else {
		header += "latest"
	}
	if container != "" {
		header += fmt.Sprintf(", container %s", container)
	}
	if previous {
		header += " (previous)"
	}
	header += fmt.Sprintf(":\n\nFound %d matching line(s)\n\n", len(matchingLines))

	if len(matchingLines) == 0 {
		return api.NewToolCallResult(header+"No matching lines found.", nil), nil
	}

	return api.NewToolCallResult(header+strings.Join(matchingLines, "\n"), nil), nil
}
