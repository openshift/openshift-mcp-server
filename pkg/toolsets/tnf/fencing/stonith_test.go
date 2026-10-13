package fencing

import (
	"testing"

	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/stretchr/testify/suite"
)

type STONITHHandlerSuite struct {
	suite.Suite
}

func (s *STONITHHandlerSuite) TestInvalidNodeType() {
	params := api.ToolHandlerParams{
		ToolCallRequest: staticRequest{args: map[string]any{
			"node": 12345,
		}},
	}
	result, err := checkSTONITHStatus(params)
	s.Require().NoError(err)
	s.Require().NotNil(result.Error)
	s.Contains(result.Error.Error(), "failed to check STONITH status")
}

func (s *STONITHHandlerSuite) TestInvalidTimeoutType() {
	params := api.ToolHandlerParams{
		ToolCallRequest: staticRequest{args: map[string]any{
			"timeout_seconds": "not-a-number",
		}},
	}
	result, err := checkSTONITHStatus(params)
	s.Require().NoError(err)
	s.Require().NotNil(result.Error)
	s.Contains(result.Error.Error(), "timeout_seconds must be a numeric value")
}

func (s *STONITHHandlerSuite) TestInvalidTimeoutValue() {
	params := api.ToolHandlerParams{
		ToolCallRequest: staticRequest{args: map[string]any{
			"timeout_seconds": float64(0),
		}},
	}
	result, err := checkSTONITHStatus(params)
	s.Require().NoError(err)
	s.Require().NotNil(result.Error)
	s.Contains(result.Error.Error(), "timeout_seconds must be an integer >= 1")
}

func (s *STONITHHandlerSuite) TestParseSectionsBasic() {
	raw := `===PCS_STATUS===
Cluster name: ostest
Online: [ master-0 master-1 ]
===PCS_STONITH_CONFIG===
 Resource: fence_master_0 (class=stonith type=fence_ipmilan)
  Attributes: fence_master_0-instance_attributes
              ipaddr=192.168.1.10
              lanplus=1
===PCS_STONITH_STATUS===
  * fence_master_0	(stonith:fence_ipmilan):	 Started master-1
===PCS_STONITH_HISTORY===
No fencing actions recorded.
===PCS_QUORUM_STATUS===
Quorate:          Yes
Expected votes:   2
===PCS_PROPERTY===
stonith-enabled: true
no-quorum-policy: stop
===END===`

	sections := parseSections(raw)
	s.Contains(sections["PCS_STATUS"], "Cluster name: ostest")
	s.Contains(sections["PCS_STONITH_CONFIG"], "fence_master_0")
	s.Contains(sections["PCS_STONITH_STATUS"], "Started master-1")
	s.Contains(sections["PCS_QUORUM_STATUS"], "Quorate")
	s.Contains(sections["PCS_PROPERTY"], "stonith-enabled: true")
}

func (s *STONITHHandlerSuite) TestBuildReportHealthy() {
	raw := `===PCS_STATUS===
Cluster name: ostest
Online: [ master-0 master-1 ]
OFFLINE: [ ]
===PCS_STONITH_CONFIG===
 Resource: fence_master_0 (class=stonith type=fence_ipmilan)
===PCS_STONITH_STATUS===
  * fence_master_0	(stonith:fence_ipmilan):	 Started master-1
  * fence_master_1	(stonith:fence_ipmilan):	 Started master-0
===PCS_STONITH_HISTORY===
No fencing actions recorded.
===PCS_QUORUM_STATUS===
Quorate:          Yes
Expected votes:   2
===PCS_PROPERTY===
stonith-enabled: true
no-quorum-policy: stop
===END===`

	report, issues := buildSTONITHReport("master-0", raw)
	s.Contains(report, "TNF STONITH Status")
	s.Contains(report, "master-0, master-1")
	s.Contains(report, "fence_master_0")
	s.Contains(report, "Quorate")
	s.Empty(issues)
}

func (s *STONITHHandlerSuite) TestBuildReportStonithDisabled() {
	raw := `===PCS_STATUS===
Cluster name: ostest
Online: [ master-0 master-1 ]
===PCS_STONITH_CONFIG===
 Resource: fence_master_0 (class=stonith type=fence_ipmilan)
===PCS_STONITH_STATUS===
  * fence_master_0	(stonith:fence_ipmilan):	 Started master-1
===PCS_STONITH_HISTORY===
===PCS_QUORUM_STATUS===
Quorate:          Yes
===PCS_PROPERTY===
stonith-enabled: false
no-quorum-policy: stop
===END===`

	_, issues := buildSTONITHReport("master-0", raw)
	s.Contains(issues, "STONITH is disabled (stonith-enabled: false)")
}

func (s *STONITHHandlerSuite) TestBuildReportNodeOffline() {
	raw := `===PCS_STATUS===
Cluster name: ostest
Online: [ master-0 ]
OFFLINE: [ master-1 ]
===PCS_STONITH_CONFIG===
 Resource: fence_master_0 (class=stonith type=fence_ipmilan)
===PCS_STONITH_STATUS===
  * fence_master_0	(stonith:fence_ipmilan):	 Stopped
===PCS_STONITH_HISTORY===
===PCS_QUORUM_STATUS===
Quorate:          No
===PCS_PROPERTY===
stonith-enabled: true
===END===`

	_, issues := buildSTONITHReport("master-0", raw)
	s.Contains(issues, "nodes offline: master-1")
	s.Contains(issues, "STONITH device stopped: * fence_master_0\t(stonith:fence_ipmilan):\t Stopped")
	s.Contains(issues, "cluster is NOT quorate")
}

func (s *STONITHHandlerSuite) TestBuildReportNoDevices() {
	raw := `===PCS_STATUS===
Cluster name: ostest
Online: [ master-0 master-1 ]
===PCS_STONITH_CONFIG===
NO stonith devices configured
===PCS_STONITH_STATUS===
===PCS_STONITH_HISTORY===
===PCS_QUORUM_STATUS===
Quorate:          Yes
===PCS_PROPERTY===
stonith-enabled: true
===END===`

	_, issues := buildSTONITHReport("master-0", raw)
	s.Contains(issues, "no STONITH devices configured")
}

func (s *STONITHHandlerSuite) TestExtractBracketedList() {
	nodes := extractBracketedList("Online: [ master-0 master-1 ]")
	s.Equal([]string{"master-0", "master-1"}, nodes)

	empty := extractBracketedList("OFFLINE: [ ]")
	s.Nil(empty)

	none := extractBracketedList("no brackets here")
	s.Nil(none)
}

func TestSTONITHHandler(t *testing.T) {
	suite.Run(t, new(STONITHHandlerSuite))
}
