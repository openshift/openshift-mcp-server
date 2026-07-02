package mustgather

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"

	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/singleflight"
)

type RegistrySuite struct {
	suite.Suite
	archiveDir string
}

func (s *RegistrySuite) SetupTest() {
	registry.mu.Lock()
	registry.providers = make(map[string]*mg.Provider)
	registry.flight = singleflight.Group{}
	registry.mu.Unlock()

	dir, err := os.MkdirTemp("", "mustgather-test-*")
	s.Require().NoError(err)
	s.archiveDir = dir
}

func (s *RegistrySuite) TearDownTest() {
	os.RemoveAll(s.archiveDir)
}

func (s *RegistrySuite) TestLazyInit() {
	s.Run("loads provider on first call", func() {
		p, err := loadProvider(s.archiveDir)
		s.NoError(err)
		s.NotNil(p)
		s.Equal(s.archiveDir, p.GetMetadata().Path)
	})
}

func (s *RegistrySuite) TestCaching() {
	s.Run("returns same provider on repeated calls", func() {
		p1, err := loadProvider(s.archiveDir)
		s.Require().NoError(err)

		p2, err := loadProvider(s.archiveDir)
		s.Require().NoError(err)

		s.Same(p1, p2)
	})
}

func (s *RegistrySuite) TestMultipleArchives() {
	s.Run("loads different providers for different paths", func() {
		dir2, err := os.MkdirTemp("", "mustgather-test2-*")
		s.Require().NoError(err)
		defer os.RemoveAll(dir2)

		p1, err := loadProvider(s.archiveDir)
		s.Require().NoError(err)

		p2, err := loadProvider(dir2)
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
				p, err := loadProvider(s.archiveDir)
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
		p, err := loadProvider(s.archiveDir)
		s.NoError(err)
		s.NotNil(p)
		s.Equal(0, p.GetMetadata().ResourceCount)
	})
}

func (s *RegistrySuite) TestProviderForArchive() {
	s.Run("returns error when id is empty", func() {
		_, err := providerForArchive(&stubDirsConfig{dirs: []string{s.archiveDir}}, "")
		s.Error(err)
		s.Contains(err.Error(), "must_gather_archive_id is required")
	})
}

// stubDirsConfig is a minimal api.MustGatherDirsProvider for unit tests.
type stubDirsConfig struct {
	dirs []string
}

func (s *stubDirsConfig) GetMustGatherDirs() []string { return s.dirs }

func TestRegistry(t *testing.T) {
	suite.Run(t, new(RegistrySuite))
}
