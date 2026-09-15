package mustgather

import (
	"context"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/containers/kubernetes-mcp-server/internal/test"
	"github.com/containers/kubernetes-mcp-server/pkg/api"
	"github.com/containers/kubernetes-mcp-server/pkg/config"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/stretchr/testify/suite"
)

type RegistrySuite struct {
	suite.Suite
	registry   *mgRegistry
	archiveDir string
}

func (s *RegistrySuite) SetupTest() {
	// A fresh registry per test, matching the per-config lifecycle.
	s.registry = newRegistry()

	dir, err := os.MkdirTemp("", "mustgather-test-*")
	s.Require().NoError(err)
	s.archiveDir = dir
}

func (s *RegistrySuite) TearDownTest() {
	_ = os.RemoveAll(s.archiveDir)
}

func (s *RegistrySuite) TestLazyInit() {
	s.Run("loads provider on first call", func() {
		p, err := s.registry.loadProvider(s.archiveDir)
		s.NoError(err)
		s.NotNil(p)
		s.Equal(s.archiveDir, p.GetMetadata().Path)
	})
}

func (s *RegistrySuite) TestCaching() {
	s.Run("returns same provider on repeated calls", func() {
		p1, err := s.registry.loadProvider(s.archiveDir)
		s.Require().NoError(err)

		p2, err := s.registry.loadProvider(s.archiveDir)
		s.Require().NoError(err)

		s.Same(p1, p2)
	})
}

func (s *RegistrySuite) TestMultipleArchives() {
	s.Run("loads different providers for different paths", func() {
		dir2, err := os.MkdirTemp("", "mustgather-test2-*")
		s.Require().NoError(err)
		defer func() { _ = os.RemoveAll(dir2) }()

		p1, err := s.registry.loadProvider(s.archiveDir)
		s.Require().NoError(err)

		p2, err := s.registry.loadProvider(dir2)
		s.Require().NoError(err)

		s.NotSame(p1, p2)
		s.Equal(s.archiveDir, p1.GetMetadata().Path)
		s.Equal(dir2, p2.GetMetadata().Path)
	})
}

func (s *RegistrySuite) TestConcurrentSamePath() {
	s.Run("deduplicates concurrent loads for same path", func() {
		var wg sync.WaitGroup
		var count int32
		results := make([]*mg.Provider, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				p, err := s.registry.loadProvider(s.archiveDir)
				if err == nil {
					results[idx] = p
					atomic.AddInt32(&count, 1)
				}
			}(i)
		}
		wg.Wait()

		s.Equal(int32(10), atomic.LoadInt32(&count))
		for i := 1; i < 10; i++ {
			s.Same(results[0], results[i])
		}
	})
}

func (s *RegistrySuite) TestEmptyArchive() {
	s.Run("loads provider with zero resources for empty directory", func() {
		p, err := s.registry.loadProvider(s.archiveDir)
		s.NoError(err)
		s.NotNil(p)
		s.Equal(0, p.GetMetadata().ResourceCount)
	})
}

func (s *RegistrySuite) TestProviderForArchive() {
	s.Run("returns error when id is empty", func() {
		params := paramsWithDirs(s.T(), s.archiveDir)
		_, err := providerForArchive(params, "")
		s.Error(err)
		s.Contains(err.Error(), "archive_id is required")
	})

	s.Run("returns error when toolset is not configured", func() {
		params := api.ToolHandlerParams{
			Context:    context.Background(),
			BaseConfig: test.Must(config.ReadToml([]byte(``))),
		}
		_, err := providerForArchive(params, "mg-000000000000")
		s.Error(err)
		s.Contains(err.Error(), "not configured")
	})
}

// paramsWithDirs builds ToolHandlerParams whose toolset config points at the
// given must-gather directories, so tests can exercise the config-driven
// resolution path.
func paramsWithDirs(t *testing.T, dirs ...string) api.ToolHandlerParams {
	t.Helper()
	quoted := make([]string, 0, len(dirs))
	for _, d := range dirs {
		quoted = append(quoted, `"`+d+`"`)
	}
	toml := "[toolset_configs.\"openshift/mustgather\"]\nmustgather_dirs = [" +
		strings.Join(quoted, ", ") + "]\n"
	return api.ToolHandlerParams{
		Context:    context.Background(),
		BaseConfig: test.Must(config.ReadToml([]byte(toml))),
	}
}

func TestRegistry(t *testing.T) {
	suite.Run(t, new(RegistrySuite))
}
