package mustgather

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/stretchr/testify/suite"
	"golang.org/x/sync/singleflight"
)

type DiscoverySuite struct {
	suite.Suite
}

func (s *DiscoverySuite) SetupTest() {
	// Reset the registry so tests don't leak state into each other.
	registry.mu.Lock()
	registry.dirs = nil
	registry.byID = make(map[string]string)
	registry.providers = make(map[string]*mg.Provider)
	registry.flight = singleflight.Group{}
	registry.mu.Unlock()
}

// makeArchive creates a minimal must-gather archive under parent/name and
// returns its absolute path. The archive contains a container dir (recognized
// via the "sha256" marker) with version and timestamp metadata files.
func (s *DiscoverySuite) makeArchive(parent, name, version, timestamp string) string {
	archive := filepath.Join(parent, name)
	containerDir := filepath.Join(archive, "quay-io-openshift-content-sha256-abc123")
	s.Require().NoError(os.MkdirAll(containerDir, 0o755))
	s.Require().NoError(os.WriteFile(filepath.Join(containerDir, "version"), []byte(version), 0o644))
	s.Require().NoError(os.WriteFile(filepath.Join(containerDir, "timestamp"), []byte(timestamp), 0o644))
	abs, err := filepath.Abs(archive)
	s.Require().NoError(err)
	return abs
}

func (s *DiscoverySuite) TestDiscoverArchives() {
	s.Run("finds multiple archives and skips junk", func() {
		root := s.T().TempDir()
		a1 := s.makeArchive(root, "must-gather.aaa", "4.12", "2026-09-11T09:01:02Z")
		a2 := s.makeArchive(root, "must-gather.bbb", "4.14", "2026-09-12T09:01:02Z")
		// A junk directory that is not an archive.
		s.Require().NoError(os.MkdirAll(filepath.Join(root, "not-an-archive"), 0o755))

		archives := discoverArchives(context.Background(), []string{root})
		s.Require().Len(archives, 2)

		byPath := map[string]ArchiveInfo{}
		for _, a := range archives {
			byPath[a.Path] = a
		}
		s.Contains(byPath, a1)
		s.Contains(byPath, a2)
		s.Equal("4.12", byPath[a1].Version)
		s.Equal("2026-09-12T09:01:02Z", byPath[a2].Timestamp)
		for _, a := range archives {
			id, err := mg.ArchiveIDFromLocalPath(a.Path)
			s.Require().NoError(err)
			s.Equal(id, a.ID)
		}
	})

	s.Run("root that is itself an archive", func() {
		root := s.T().TempDir()
		containerDir := filepath.Join(root, "quay-io-content-sha256-xyz")
		s.Require().NoError(os.MkdirAll(containerDir, 0o755))
		s.Require().NoError(os.WriteFile(filepath.Join(containerDir, "version"), []byte("4.15"), 0o644))

		archives := discoverArchives(context.Background(), []string{root})
		s.Require().Len(archives, 1)
		s.Equal("4.15", archives[0].Version)
	})

	s.Run("empty root returns nothing", func() {
		archives := discoverArchives(context.Background(), []string{s.T().TempDir()})
		s.Empty(archives)
	})

	s.Run("missing root is skipped", func() {
		archives := discoverArchives(context.Background(), []string{filepath.Join(s.T().TempDir(), "does-not-exist")})
		s.Empty(archives)
	})

	s.Run("dedupes identical archive IDs across roots", func() {
		root1 := s.T().TempDir()
		root2 := s.T().TempDir()
		// Same parent-dir-name + leaf-name in two roots would only collide if the
		// absolute paths hash equal; here the absolute paths differ, so both show.
		s.makeArchive(root1, "must-gather.x", "4.1", "")
		s.makeArchive(root2, "must-gather.x", "4.2", "")
		archives := discoverArchives(context.Background(), []string{root1, root2})
		s.Len(archives, 2)
	})
}

func (s *DiscoverySuite) TestResolveArchivePath() {
	root := s.T().TempDir()
	a1 := s.makeArchive(root, "must-gather.aaa", "4.12", "")
	id1, err := mg.ArchiveIDFromLocalPath(a1)
	s.Require().NoError(err)

	s.Run("resolves a known ID", func() {
		path, err := resolveArchivePath(context.Background(), []string{root}, id1)
		s.Require().NoError(err)
		s.Equal(a1, path)
	})

	s.Run("unknown ID lists known IDs", func() {
		_, err := resolveArchivePath(context.Background(), []string{root}, "mg-000000000000")
		s.Require().Error(err)
		s.Contains(err.Error(), id1)
	})

	s.Run("invalid ID format errors", func() {
		_, err := resolveArchivePath(context.Background(), []string{root}, "not-an-id")
		s.Error(err)
	})

	s.Run("no dirs configured errors", func() {
		_, err := resolveArchivePath(context.Background(), nil, id1)
		s.Require().Error(err)
		s.Contains(err.Error(), "no must-gather directories configured")
	})

	s.Run("re-scans when a cached archive is deleted", func() {
		tmp := s.T().TempDir()
		a := s.makeArchive(tmp, "must-gather.tmp", "4.9", "")
		id, err := mg.ArchiveIDFromLocalPath(a)
		s.Require().NoError(err)

		path, err := resolveArchivePath(context.Background(), []string{tmp}, id)
		s.Require().NoError(err)
		s.Equal(a, path)

		// Delete the archive; the stale cache entry must not be returned.
		s.Require().NoError(os.RemoveAll(a))
		_, err = resolveArchivePath(context.Background(), []string{tmp}, id)
		s.Error(err)
	})

	s.Run("resolves without archive metadata files", func() {
		// An archive whose container dir has neither version nor timestamp still
		// resolves: ID resolution derives the ID from the path alone.
		tmp := s.T().TempDir()
		archive := filepath.Join(tmp, "must-gather.nometa")
		s.Require().NoError(os.MkdirAll(filepath.Join(archive, "quay-io-content-sha256-nometa"), 0o755))
		abs, err := filepath.Abs(archive)
		s.Require().NoError(err)
		id, err := mg.ArchiveIDFromLocalPath(abs)
		s.Require().NoError(err)

		path, err := resolveArchivePath(context.Background(), []string{tmp}, id)
		s.Require().NoError(err)
		s.Equal(abs, path)
	})

	s.Run("discovers a newly added archive on next resolve", func() {
		tmp := s.T().TempDir()
		a1 := s.makeArchive(tmp, "must-gather.first", "4.1", "")
		id1, err := mg.ArchiveIDFromLocalPath(a1)
		s.Require().NoError(err)
		_, err = resolveArchivePath(context.Background(), []string{tmp}, id1)
		s.Require().NoError(err)

		// A brand-new archive appears after the first scan; a miss triggers a
		// rescan that discovers it without any restart.
		a2 := s.makeArchive(tmp, "must-gather.second", "4.2", "")
		id2, err := mg.ArchiveIDFromLocalPath(a2)
		s.Require().NoError(err)
		path, err := resolveArchivePath(context.Background(), []string{tmp}, id2)
		s.Require().NoError(err)
		s.Equal(a2, path)
	})
}

func (s *DiscoverySuite) TestRescanInvalidatesProviders() {
	s.Run("a rescan clears every cached provider", func() {
		root := s.T().TempDir()
		a := s.makeArchive(root, "must-gather.aaa", "4.12", "")
		id, err := mg.ArchiveIDFromLocalPath(a)
		s.Require().NoError(err)

		// Warm the provider cache via the full resolve+load path.
		p1, err := providerForArchiveIn(context.Background(), []string{root}, id)
		s.Require().NoError(err)

		registry.mu.RLock()
		_, cached := registry.providers[a]
		registry.mu.RUnlock()
		s.True(cached, "provider should be cached after load")

		// Changing the dir-set forces a rescan, which must invalidate providers.
		other := s.T().TempDir()
		_, _ = resolveArchivePath(context.Background(), []string{other, root}, id)

		registry.mu.RLock()
		_, stillCached := registry.providers[a]
		registry.mu.RUnlock()
		s.False(stillCached, "rescan should have invalidated the cached provider")

		// A subsequent load produces a fresh provider instance.
		p2, err := providerForArchiveIn(context.Background(), []string{other, root}, id)
		s.Require().NoError(err)
		s.NotSame(p1, p2)
	})
}

// providerForArchiveIn mirrors providerForArchive but with an explicit dir-set,
// so tests need not touch the global toolset config.
func providerForArchiveIn(ctx context.Context, dirs []string, id string) (*mg.Provider, error) {
	path, err := resolveArchivePath(ctx, dirs, id)
	if err != nil {
		return nil, err
	}
	return loadProvider(path)
}

func TestDiscovery(t *testing.T) {
	suite.Run(t, new(DiscoverySuite))
}
