package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/suite"

	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
)

type MustGatherSuite struct {
	BaseMcpSuite
	archivePath string
	archiveID   string
}

// SetupTest builds a minimal must-gather archive on disk, points the
// openshift/mustgather toolset at its parent directory, and records the
// server-computed archive ID the tools will address it by.
func (s *MustGatherSuite) SetupTest() {
	s.BaseMcpSuite.SetupTest()

	root := s.T().TempDir()
	archive := filepath.Join(root, "must-gather.local.test.20260911.abcd")
	container := filepath.Join(archive, "quay-io-openshift-content-sha256-deadbeef")
	s.Require().NoError(os.MkdirAll(container, 0o755))
	s.Require().NoError(os.WriteFile(filepath.Join(container, "version"), []byte("4.16.0"), 0o644))
	s.Require().NoError(os.WriteFile(filepath.Join(container, "timestamp"), []byte("2026-09-11T09:01:02Z"), 0o644))

	// A namespaced ConfigMap so mustgather_resources_list has something to return.
	cmDir := filepath.Join(container, "namespaces", "openshift-config", "core")
	s.Require().NoError(os.MkdirAll(cmDir, 0o755))
	s.Require().NoError(os.WriteFile(filepath.Join(cmDir, "configmaps.yaml"), []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-config-v1
  namespace: openshift-config
data:
  install-config: "{}"
`), 0o644))

	// A namespaced Event so mustgather_events_list has something to return.
	evDir := filepath.Join(container, "namespaces", "openshift-config", "core")
	s.Require().NoError(os.WriteFile(filepath.Join(evDir, "events.yaml"), []byte(`
apiVersion: v1
kind: Event
metadata:
  name: cluster-config-v1.abc
  namespace: openshift-config
type: Warning
reason: FailedMount
involvedObject:
  kind: ConfigMap
  name: cluster-config-v1
message: sample event
`), 0o644))

	abs, err := filepath.Abs(archive)
	s.Require().NoError(err)
	s.archivePath = abs
	s.archiveID, err = mg.ArchiveIDFromPath(abs)
	s.Require().NoError(err)

	s.Require().NoError(toml.Unmarshal([]byte(`
		toolsets = [ "openshift/mustgather" ]
	`), s.Cfg), "Expected to parse toolsets config")
	s.Cfg.MustGatherDirs = []string{root}
}

func (s *MustGatherSuite) TestList() {
	s.InitMcpClient()
	s.Run("mustgather_list returns the archive ID", func() {
		result, err := s.CallTool("mustgather_list", map[string]interface{}{})
		s.Require().NoError(err, "expected mustgather_list to succeed")
		s.Require().False(result.IsError, "expected mustgather_list not to be an error")
		text := result.Content[0].(*mcp.TextContent).Text
		s.Contains(text, s.archiveID, "expected archive ID in listing")
		s.Contains(text, "4.16.0", "expected archive version in listing")
	})
}

func (s *MustGatherSuite) TestResourcesList() {
	s.InitMcpClient()
	s.Run("mustgather_resources_list resolves archive by ID", func() {
		result, err := s.CallTool("mustgather_resources_list", map[string]interface{}{
			"must_gather_archive_id": s.archiveID,
			"kind":                   "ConfigMap",
		})
		s.Require().NoError(err)
		s.Require().False(result.IsError, "expected resources_list not to be an error")
		text := result.Content[0].(*mcp.TextContent).Text
		s.Contains(text, "cluster-config-v1", "expected the ConfigMap in the listing")
	})
	s.Run("mustgather_resources_list errors on unknown archive ID", func() {
		result, err := s.CallTool("mustgather_resources_list", map[string]interface{}{
			"must_gather_archive_id": "mg-0000-00000000",
			"kind":                   "ConfigMap",
		})
		s.Require().NoError(err)
		s.True(result.IsError, "expected an error result for an unknown archive ID")
		s.Contains(result.Content[0].(*mcp.TextContent).Text, "not found")
	})
}

func (s *MustGatherSuite) TestEventsList() {
	s.InitMcpClient()
	s.Run("mustgather_events_list resolves archive by ID", func() {
		result, err := s.CallTool("mustgather_events_list", map[string]interface{}{
			"must_gather_archive_id": s.archiveID,
		})
		s.Require().NoError(err)
		s.Require().False(result.IsError, "expected events_list not to be an error")
		s.Contains(result.Content[0].(*mcp.TextContent).Text, "FailedMount", "expected the event reason in the listing")
	})
}

func (s *MustGatherSuite) TestResourceTemplates() {
	s.InitMcpClient()
	s.Run("reads namespaces via resource template", func() {
		uri := "must-gather://local/" + s.archiveID + "/namespaces"
		result, err := s.ReadResource(uri)
		s.Require().NoError(err)
		s.Require().NotEmpty(result.Contents)
		s.Contains(result.Contents[0].Text, "openshift-config", "expected the namespace listed")
	})
	s.Run("reads a specific resource via resource template", func() {
		uri := "must-gather://local/" + s.archiveID + "/resources/-/v1/ConfigMap/openshift-config/cluster-config-v1"
		result, err := s.ReadResource(uri)
		s.Require().NoError(err)
		s.Require().NotEmpty(result.Contents)
		s.True(strings.Contains(result.Contents[0].Text, "cluster-config-v1"), "expected the ConfigMap YAML")
	})
}

func TestMustGather(t *testing.T) {
	suite.Run(t, new(MustGatherSuite))
}
