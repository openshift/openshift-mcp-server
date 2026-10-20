package mustgather

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type LogRetentionSuite struct {
	suite.Suite
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

func TestLogRetentionSuite(t *testing.T) {
	suite.Run(t, new(LogRetentionSuite))
}
