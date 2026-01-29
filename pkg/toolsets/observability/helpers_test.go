package observability

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type HelpersSuite struct {
	suite.Suite
}

func (s *HelpersSuite) TestValidateAlertmanagerEndpoint() {
	s.Run("allows valid v2 alerts endpoint", func() {
		err := validateAlertmanagerEndpoint("/api/v2/alerts")
		s.NoError(err)
	})

	s.Run("allows valid v2 silences endpoint", func() {
		err := validateAlertmanagerEndpoint("/api/v2/silences")
		s.NoError(err)
	})

	s.Run("allows valid v1 alerts endpoint", func() {
		err := validateAlertmanagerEndpoint("/api/v1/alerts")
		s.NoError(err)
	})

	s.Run("rejects invalid endpoint", func() {
		err := validateAlertmanagerEndpoint("/api/v2/status")
		s.Error(err)
		s.Contains(err.Error(), "not allowed")
	})
}

func (s *HelpersSuite) TestPrettyJSON() {
	s.Run("formats valid JSON", func() {
		input := []byte(`{"key":"value","number":123}`)
		result, err := prettyJSON(input)

		s.NoError(err)
		s.Contains(result, "\"key\"")
		s.Contains(result, "\"value\"")
		s.Contains(result, "\n") // Should have newlines for pretty printing
	})

	s.Run("returns original for invalid JSON", func() {
		input := []byte("not valid json")
		result, err := prettyJSON(input)

		s.NoError(err)
		s.Equal("not valid json", result)
	})
}

func TestHelpersSuite(t *testing.T) {
	suite.Run(t, new(HelpersSuite))
}
