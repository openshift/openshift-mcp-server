package netedge

import (
	"fmt"
	"strings"
)

func (s *NetEdgeTestSuite) TestTruncateConfigOutput() {
	smallConfig := strings.Join([]string{
		"global",
		"  maxconn 20000",
		"defaults",
		"  timeout connect 5s",
		"frontend public",
		"  bind *:80",
		"frontend public_ssl",
		"  bind *:443",
		"backend be_http:myapp:ns",
		"  server pod1 10.0.0.1:8080",
		"backend be_http:otherapp:ns",
		"  server pod2 10.0.0.2:8080",
	}, "\n")

	s.Run("no truncation needed", func() {
		result, total, shown, truncated := truncateConfigOutput(smallConfig, 200, "", "")
		s.Equal(smallConfig, result)
		s.Equal(12, total)
		s.Equal(12, shown)
		s.False(truncated)
	})

	s.Run("tail_lines truncation", func() {
		result, total, shown, truncated := truncateConfigOutput(smallConfig, 5, "", "")
		lines := strings.Split(result, "\n")
		s.Equal(5, len(lines))
		s.Equal(12, total)
		s.Equal(5, shown)
		s.True(truncated)
		s.Contains(result, "backend be_http:otherapp:ns")
	})

	s.Run("section filter - global", func() {
		result, total, shown, truncated := truncateConfigOutput(smallConfig, 200, "global", "")
		s.Equal(12, total)
		s.Equal(2, shown)
		s.False(truncated)
		s.Contains(result, "global")
		s.Contains(result, "maxconn")
		s.NotContains(result, "defaults")
		s.NotContains(result, "frontend")
	})

	s.Run("section filter - backend returns all backends", func() {
		result, total, shown, truncated := truncateConfigOutput(smallConfig, 200, "backend", "")
		s.Equal(12, total)
		s.Equal(4, shown)
		s.False(truncated)
		s.Contains(result, "be_http:myapp:ns")
		s.Contains(result, "be_http:otherapp:ns")
		s.NotContains(result, "frontend")
	})

	s.Run("section filter - frontend", func() {
		result, _, shown, _ := truncateConfigOutput(smallConfig, 200, "frontend", "")
		s.Equal(4, shown)
		s.Contains(result, "frontend public")
		s.Contains(result, "frontend public_ssl")
	})

	s.Run("substring filter on section header", func() {
		result, total, shown, truncated := truncateConfigOutput(smallConfig, 200, "", "myapp")
		s.Equal(12, total)
		s.Equal(2, shown)
		s.False(truncated)
		s.Contains(result, "be_http:myapp:ns")
		s.NotContains(result, "otherapp")
	})

	s.Run("section plus filter combined", func() {
		result, _, shown, _ := truncateConfigOutput(smallConfig, 200, "backend", "myapp")
		s.Equal(2, shown)
		s.Contains(result, "be_http:myapp:ns")
		s.NotContains(result, "otherapp")
	})

	s.Run("filter is case insensitive", func() {
		result, _, shown, _ := truncateConfigOutput(smallConfig, 200, "", "MYAPP")
		s.Equal(2, shown)
		s.Contains(result, "be_http:myapp:ns")
	})

	s.Run("filtered result still truncated by tail_lines", func() {
		_, _, shown, truncated := truncateConfigOutput(smallConfig, 1, "backend", "")
		s.Equal(1, shown)
		s.True(truncated)
	})

	s.Run("empty output", func() {
		result, total, shown, truncated := truncateConfigOutput("", 200, "", "")
		s.Equal("", result)
		s.Equal(0, total)
		s.Equal(0, shown)
		s.False(truncated)
	})

	s.Run("no matching section", func() {
		result, total, shown, truncated := truncateConfigOutput(smallConfig, 200, "frontend", "nonexistent")
		s.Equal(12, total)
		s.Equal(0, shown)
		s.False(truncated)
		s.Equal("", result)
	})

	s.Run("large config with many backends", func() {
		var lines []string
		lines = append(lines, "global", "  maxconn 20000")
		lines = append(lines, "defaults", "  timeout connect 5s")
		for i := 0; i < 100; i++ {
			lines = append(lines, fmt.Sprintf("backend be_http:app%d:ns", i))
			lines = append(lines, fmt.Sprintf("  server pod%d 10.0.%d.1:8080", i, i))
		}
		largeConfig := strings.Join(lines, "\n")

		result, total, shown, truncated := truncateConfigOutput(largeConfig, 50, "", "")
		s.Equal(204, total)
		s.Equal(50, shown)
		s.True(truncated)
		resultLines := strings.Split(result, "\n")
		s.Equal(50, len(resultLines))
	})
}

func (s *NetEdgeTestSuite) TestTruncateSessionsOutput() {
	makeSession := func(id int, backend string) string {
		return fmt.Sprintf("0x%08x: proto=tcpv4 src=10.0.0.%d:12345\n  backend=%s state=active\n  flags=0x0", id, id%256, backend)
	}

	smallOutput := strings.Join([]string{
		makeSession(1, "be_myapp"),
		makeSession(2, "be_otherapp"),
		makeSession(3, "be_myapp"),
	}, "\n")

	s.Run("no truncation needed", func() {
		result, total, shown, truncated := truncateSessionsOutput(smallOutput, 50, "")
		s.Equal(3, total)
		s.Equal(3, shown)
		s.False(truncated)
		s.Equal(smallOutput, result)
	})

	s.Run("limit truncation", func() {
		result, total, shown, truncated := truncateSessionsOutput(smallOutput, 2, "")
		s.Equal(3, total)
		s.Equal(2, shown)
		s.True(truncated)
		s.Contains(result, "0x00000001:")
		s.Contains(result, "0x00000002:")
		s.NotContains(result, "0x00000003:")
	})

	s.Run("filter by backend", func() {
		result, total, shown, truncated := truncateSessionsOutput(smallOutput, 50, "be_myapp")
		s.Equal(3, total)
		s.Equal(2, shown)
		s.False(truncated)
		s.Contains(result, "0x00000001:")
		s.Contains(result, "0x00000003:")
		s.NotContains(result, "be_otherapp")
	})

	s.Run("filter is case insensitive", func() {
		_, _, shown, _ := truncateSessionsOutput(smallOutput, 50, "BE_MYAPP")
		s.Equal(2, shown)
	})

	s.Run("filter plus limit", func() {
		result, total, shown, truncated := truncateSessionsOutput(smallOutput, 1, "be_myapp")
		s.Equal(3, total)
		s.Equal(1, shown)
		s.True(truncated)
		s.Contains(result, "0x00000001:")
		s.NotContains(result, "0x00000003:")
	})

	s.Run("empty output", func() {
		result, total, shown, truncated := truncateSessionsOutput("", 50, "")
		s.Equal("", result)
		s.Equal(0, total)
		s.Equal(0, shown)
		s.False(truncated)
	})

	s.Run("no matching filter", func() {
		result, total, shown, truncated := truncateSessionsOutput(smallOutput, 50, "nonexistent")
		s.Equal(3, total)
		s.Equal(0, shown)
		s.False(truncated)
		s.Equal("", result)
	})

	s.Run("large session output", func() {
		var sessions []string
		for i := 0; i < 100; i++ {
			sessions = append(sessions, makeSession(i, fmt.Sprintf("be_app%d", i%10)))
		}
		largeOutput := strings.Join(sessions, "\n")

		result, total, shown, truncated := truncateSessionsOutput(largeOutput, 10, "")
		s.Equal(100, total)
		s.Equal(10, shown)
		s.True(truncated)
		s.Contains(result, "0x00000000:")
		s.NotContains(result, "0x0000000a:")
	})
}
