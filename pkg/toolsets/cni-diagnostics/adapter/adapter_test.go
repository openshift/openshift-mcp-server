package adapter

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type AdapterSuite struct {
	suite.Suite
}

func (s *AdapterSuite) TestNewRunDebugNodeCommand() {
	s.Run("returns function with correct signature", func() {
		fn := NewRunDebugNodeCommand(nil)
		s.NotNil(fn, "adapter should return a non-nil function")
	})
}

func (s *AdapterSuite) TestNewRunPodExecCommand() {
	s.Run("returns function with correct signature", func() {
		fn := NewRunPodExecCommand(nil)
		s.NotNil(fn, "adapter should return a non-nil function")
	})
}

func TestAdapter(t *testing.T) {
	suite.Run(t, new(AdapterSuite))
}
