package mustgather

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/stretchr/testify/suite"
)

type LogRetentionSuite struct {
	suite.Suite
}

type logRequest map[string]any

func (r logRequest) GetArguments() map[string]any { return r }

func (s *LogRetentionSuite) logParams(logs string, maxOutputSize int) api.ToolHandlerParams {
	parent := s.T().TempDir()
	archive := filepath.Join(parent, "gather")
	makeArchiveRoot(s.T(), archive)
	path := filepath.Join(archive, "quay-io-openshift-content-sha256-abc123", "namespaces", "ns", "pods", "pod", "app", "app", "logs", "current.log")
	s.Require().NoError(os.MkdirAll(filepath.Dir(path), 0o755))
	s.Require().NoError(os.WriteFile(path, []byte(logs), 0o600))
	id, err := mg.ArchiveIDFromLocalPath(archive)
	s.Require().NoError(err)

	params := paramsWithDirs(s.T(), parent)
	cfg, ok := params.GetToolsetConfig("openshift/mustgather")
	s.Require().True(ok)
	cfg.(*Config).MaxOutputSize = maxOutputSize
	params.Request = logRequest{"archive_id": id, "namespace": "ns", "pod": "pod", "container": "app", "tail": 100}
	return params
}

func (s *LogRetentionSuite) TestTailKeepsRecentLinesWithinByteCap() {
	tail := newBoundedLogTail(1000, 32)
	for range 1000 {
		tail.add("older", 32)
	}
	tail.add("second", 32)
	tail.add("third", 32)

	s.True(tail.omitted)
	s.LessOrEqual(len(tail.text()), 32)
	s.True(strings.HasSuffix(tail.text(), "second\nthird"))
}

func (s *LogRetentionSuite) TestTailDoesNotRetainOversizedLine() {
	tail := newBoundedLogTail(10, 12)
	tail.add(strings.Repeat("x", 100), 12)
	tail.add("last", 12)

	s.True(tail.omitted)
	s.Equal("last", tail.text())
}

func (s *LogRetentionSuite) TestRequestedTailDoesNotCountAsTruncation() {
	tail := newBoundedLogTail(2, 100)
	tail.add("first", 100)
	tail.add("second", 100)
	tail.add("third", 100)

	s.False(tail.omitted)
	s.Equal("second\nthird", tail.text())
}

func (s *LogRetentionSuite) TestTailAndGrepKeepNewestLinesAndNotice() {
	params := s.logParams(strings.Repeat("old match\n", 30)+"recent match\nnewest match\n", 80)

	get, err := mustgatherPodLogsGet(params)
	s.Require().NoError(err)
	s.Contains(get.Content, "newest match"+outputTruncatedNotice(80))
	s.LessOrEqual(len(get.Content)-strings.Index(get.Content, ":\n\n")-3, 80)

	params.Request = logRequest{"archive_id": params.GetArguments()["archive_id"], "namespace": "ns", "pod": "pod", "container": "app", "filter": "match", "tail": 100}
	grep, err := mustgatherPodLogsGrep(params)
	s.Require().NoError(err)
	s.Contains(grep.Content, "newest match"+outputTruncatedNotice(80))
}

func (s *LogRetentionSuite) TestResultBudgetKeepsNewestTailAfterHeader() {
	params := s.logParams(strings.Repeat("older\n", 40)+"newest\n", 200)
	params.Context = api.WithResultBudget(params.Context, 110)

	result, err := mustgatherPodLogsGet(params)
	s.Require().NoError(err)
	s.Contains(result.Content, "newest"+api.ResultTruncationNotice)
	s.LessOrEqual(len(result.Content), 110)
}

func TestLogRetentionSuite(t *testing.T) {
	suite.Run(t, new(LogRetentionSuite))
}
