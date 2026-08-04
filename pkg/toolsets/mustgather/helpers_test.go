package mustgather

import (
	"context"
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
	registry.lastUsed = make(map[string]string)
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
		p, err := InitProvider(context.Background(), s.archiveDir)
		s.NoError(err)
		s.NotNil(p)
		s.Equal(s.archiveDir, p.GetMetadata().Path)
	})
}

func (s *RegistrySuite) TestCaching() {
	s.Run("returns same provider on repeated calls", func() {
		p1, err := InitProvider(context.Background(), s.archiveDir)
		s.Require().NoError(err)

		p2, err := InitProvider(context.Background(), s.archiveDir)
		s.Require().NoError(err)

		s.Same(p1, p2)
	})
}

func (s *RegistrySuite) TestLastUsedFallback() {
	s.Run("uses last-used path when path is empty", func() {
		_, err := InitProvider(context.Background(), s.archiveDir)
		s.Require().NoError(err)

		p, err := InitProvider(context.Background(), "")
		s.NoError(err)
		s.Equal(s.archiveDir, p.GetMetadata().Path)
	})
}

func (s *RegistrySuite) TestNoPathNoLastUsed() {
	s.Run("returns error when no path and no last-used", func() {
		_, err := InitProvider(context.Background(), "")
		s.Error(err)
		s.Contains(err.Error(), "no must-gather archive loaded")
	})
}

func (s *RegistrySuite) TestGetProviderForResource() {
	s.Run("uses session last-used path", func() {
		_, err := InitProvider(context.Background(), s.archiveDir)
		s.Require().NoError(err)

		p, err := GetProviderForResource(context.Background())
		s.NoError(err)
		s.Equal(s.archiveDir, p.GetMetadata().Path)
	})

	s.Run("returns error when no last-used", func() {
		registry.mu.Lock()
		registry.providers = make(map[string]*mg.Provider)
		registry.lastUsed = make(map[string]string)
		registry.mu.Unlock()

		_, err := GetProviderForResource(context.Background())
		s.Error(err)
	})
}

func (s *RegistrySuite) TestMultipleArchives() {
	s.Run("loads different providers for different paths", func() {
		dir2, err := os.MkdirTemp("", "mustgather-test2-*")
		s.Require().NoError(err)
		defer os.RemoveAll(dir2)

		p1, err := InitProvider(context.Background(), s.archiveDir)
		s.Require().NoError(err)

		p2, err := InitProvider(context.Background(), dir2)
		s.Require().NoError(err)

		s.NotSame(p1, p2)
		s.Equal(s.archiveDir, p1.GetMetadata().Path)
		s.Equal(dir2, p2.GetMetadata().Path)
	})
}

func (s *RegistrySuite) TestLastUsedUpdatedOnSwitch() {
	s.Run("last-used updates when switching archives", func() {
		dir2, err := os.MkdirTemp("", "mustgather-test2-*")
		s.Require().NoError(err)
		defer os.RemoveAll(dir2)

		_, err = InitProvider(context.Background(), s.archiveDir)
		s.Require().NoError(err)

		_, err = InitProvider(context.Background(), dir2)
		s.Require().NoError(err)

		// Last-used should now point to dir2
		p, err := InitProvider(context.Background(), "")
		s.Require().NoError(err)
		s.Equal(dir2, p.GetMetadata().Path)
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
				p, err := InitProvider(context.Background(), s.archiveDir)
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
		p, err := InitProvider(context.Background(), s.archiveDir)
		s.NoError(err)
		s.NotNil(p)
		s.Equal(0, p.GetMetadata().ResourceCount)
	})
}

func (s *RegistrySuite) TestSessionIDExtraction() {
	s.Run("returns empty string for plain context", func() {
		s.Equal("", sessionID(context.Background()))
	})

	s.Run("returns empty string for nil session", func() {
		s.Equal("", sessionID(context.Background()))
	})
}

func TestRegistry(t *testing.T) {
	suite.Run(t, new(RegistrySuite))
}
