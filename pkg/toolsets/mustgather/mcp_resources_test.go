package mustgather

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/stretchr/testify/suite"
)

type CompletionArchiveDirSuite struct {
	suite.Suite
	root string
	id   string
}

func (s *CompletionArchiveDirSuite) SetupTest() {
	s.root = s.T().TempDir()
	archive := filepath.Join(s.root, "must-gather.local.test")
	containerDir := filepath.Join(archive, "quay-io-openshift-content-sha256-abc123")
	logsDir := filepath.Join(containerDir, "host_service_logs", "masters")
	s.Require().NoError(os.MkdirAll(logsDir, 0o755))
	s.Require().NoError(os.WriteFile(filepath.Join(containerDir, "version"), []byte("4.19"), 0o644))
	s.Require().NoError(os.WriteFile(filepath.Join(logsDir, "kubelet_service.log"), []byte("kubelet\n"), 0o644))
	s.Require().NoError(os.WriteFile(filepath.Join(logsDir, "crio_service.log"), []byte("crio\n"), 0o644))

	abs, err := filepath.Abs(archive)
	s.Require().NoError(err)
	id, err := mg.ArchiveIDFromLocalPath(abs)
	s.Require().NoError(err)
	s.id = id

	// Publish a committed config so providerForArchiveContext can resolve the
	// archive from the package-global current pointer.
	current.Store(&Config{MustGatherDirs: []string{s.root}, registry: newRegistry()})
}

func (s *CompletionArchiveDirSuite) TearDownTest() {
	current.Store(nil)
}

func (s *CompletionArchiveDirSuite) TestCompletionArchiveDir() {
	complete := completionArchiveDir("host_service_logs")
	ctx := context.Background()
	args := map[string]string{"archive_id": s.id}

	s.Run("lists files and subdirs under the directory", func() {
		values, err := complete(ctx, "path", "", args)
		s.Require().NoError(err)
		s.Contains(values, "masters")
		s.Contains(values, "masters/kubelet_service.log")
		s.Contains(values, "masters/crio_service.log")
		s.IsIncreasing(values, "values should be sorted")
	})

	s.Run("filters by the partial value", func() {
		values, err := complete(ctx, "path", "masters/ku", args)
		s.Require().NoError(err)
		s.Equal([]string{"masters/kubelet_service.log"}, values)
	})

	s.Run("edge cases", func() {
		s.Run("non-path argument yields no suggestions", func() {
			values, err := complete(ctx, "archive_id", "", args)
			s.Require().NoError(err)
			s.Nil(values)
		})
		s.Run("missing archive_id yields no suggestions", func() {
			values, err := complete(ctx, "path", "", map[string]string{})
			s.Require().NoError(err)
			s.Nil(values)
		})
		s.Run("unknown archive yields no suggestions, not an error", func() {
			values, err := complete(ctx, "path", "", map[string]string{"archive_id": "mg-000000000000"})
			s.Require().NoError(err)
			s.Nil(values)
		})
	})
}

func TestCompletionArchiveDir(t *testing.T) {
	suite.Run(t, new(CompletionArchiveDirSuite))
}
