package ovs

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/suite"
)

type OVSRetentionSuite struct {
	suite.Suite
}

func (s *OVSRetentionSuite) TestStructuredResultKeepsCompleteJSON() {
	value := map[string]string{"output": strings.Repeat("x", 80)}
	encoded, err := json.Marshal(value)
	s.Require().NoError(err)

	textOnly := api.WithResultBudget(context.Background(), int64(len(encoded)))
	result := budgetOVSResult(textOnly, value, nil)
	s.Equal(string(encoded), result.Content)
	s.Nil(result.StructuredContent)
	s.NoError(result.Error)

	tooSmall := api.WithResultBudget(context.Background(), int64(len(encoded)-1))
	result = budgetOVSResult(tooSmall, value, nil)
	s.ErrorContains(result.Error, "max_result_bytes")
	s.Empty(result.Content)
}

func (s *OVSRetentionSuite) TestArrayWrapperCountsTowardBudget() {
	value := []string{"one", "two"}
	encoded, err := json.Marshal(value)
	s.Require().NoError(err)

	budget := int64(len(encoded) * 2)
	result := budgetOVSResult(api.WithResultBudget(context.Background(), budget), value, nil)
	s.Equal(string(encoded), result.Content)
	s.Nil(result.StructuredContent)
}

func TestOVSRetentionSuite(t *testing.T) {
	suite.Run(t, new(OVSRetentionSuite))
}
