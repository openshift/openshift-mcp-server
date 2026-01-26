package observability

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type HelpersSuite struct {
	suite.Suite
}

func (s *HelpersSuite) TestValidatePrometheusEndpoint() {
	s.Run("allows valid query endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/query")
		s.NoError(err)
	})

	s.Run("allows valid query_range endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/query_range")
		s.NoError(err)
	})

	s.Run("allows valid series endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/series")
		s.NoError(err)
	})

	s.Run("allows valid labels endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/labels")
		s.NoError(err)
	})

	s.Run("allows valid label values endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/label/job/values")
		s.NoError(err)
	})

	s.Run("rejects invalid endpoint", func() {
		err := validatePrometheusEndpoint("/api/v1/admin/tsdb/delete_series")
		s.Error(err)
		s.Contains(err.Error(), "not allowed")
	})

	s.Run("rejects arbitrary path", func() {
		err := validatePrometheusEndpoint("/some/random/path")
		s.Error(err)
	})
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
