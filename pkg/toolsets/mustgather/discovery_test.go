package mustgather

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
	"github.com/stretchr/testify/suite"
)

type DiscoverySuite struct {
	suite.Suite
}

func (s *DiscoverySuite) SetupTest() {
	// Reset the resolution cache so tests don't leak state into each other.
	scanCache.mu.Lock()
	scanCache.byID = make(map[string]string)
	scanCache.scanAt = time.Time{}
	scanCache.mu.Unlock()
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

		archives := discoverArchives([]string{root})
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
			id, err := mg.ArchiveIDFromPath(a.Path)
			s.Require().NoError(err)
			s.Equal(id, a.ID)
		}
	})

	s.Run("root that is itself an archive", func() {
		root := s.T().TempDir()
		containerDir := filepath.Join(root, "quay-io-content-sha256-xyz")
		s.Require().NoError(os.MkdirAll(containerDir, 0o755))
		s.Require().NoError(os.WriteFile(filepath.Join(containerDir, "version"), []byte("4.15"), 0o644))

		archives := discoverArchives([]string{root})
		s.Require().Len(archives, 1)
		s.Equal("4.15", archives[0].Version)
	})

	s.Run("empty root returns nothing", func() {
		archives := discoverArchives([]string{s.T().TempDir()})
		s.Empty(archives)
	})

	s.Run("missing root is skipped", func() {
		archives := discoverArchives([]string{filepath.Join(s.T().TempDir(), "does-not-exist")})
		s.Empty(archives)
	})

	s.Run("dedupes identical archive IDs across roots", func() {
		root1 := s.T().TempDir()
		root2 := s.T().TempDir()
		// Same parent-dir-name + leaf-name in two roots would only collide if the
		// absolute paths hash equal; here the absolute paths differ, so both show.
		s.makeArchive(root1, "must-gather.x", "4.1", "")
		s.makeArchive(root2, "must-gather.x", "4.2", "")
		archives := discoverArchives([]string{root1, root2})
		s.Len(archives, 2)
	})
}

func (s *DiscoverySuite) TestResolveArchivePath() {
	root := s.T().TempDir()
	a1 := s.makeArchive(root, "must-gather.aaa", "4.12", "")
	id1, err := mg.ArchiveIDFromPath(a1)
	s.Require().NoError(err)

	s.Run("resolves a known ID", func() {
		path, err := resolveArchivePath([]string{root}, id1)
		s.Require().NoError(err)
		s.Equal(a1, path)
	})

	s.Run("unknown ID lists known IDs", func() {
		_, err := resolveArchivePath([]string{root}, "mg-0000-00000000")
		s.Require().Error(err)
		s.Contains(err.Error(), id1)
	})

	s.Run("invalid ID format errors", func() {
		_, err := resolveArchivePath([]string{root}, "not-an-id")
		s.Error(err)
	})

	s.Run("no dirs configured errors", func() {
		_, err := resolveArchivePath(nil, id1)
		s.Require().Error(err)
		s.Contains(err.Error(), "no must-gather directories configured")
	})

	s.Run("re-scans when a cached archive is deleted", func() {
		tmp := s.T().TempDir()
		a := s.makeArchive(tmp, "must-gather.tmp", "4.9", "")
		id, err := mg.ArchiveIDFromPath(a)
		s.Require().NoError(err)

		path, err := resolveArchivePath([]string{tmp}, id)
		s.Require().NoError(err)
		s.Equal(a, path)

		// Delete the archive; the stale cache entry must not be returned.
		s.Require().NoError(os.RemoveAll(a))
		_, err = resolveArchivePath([]string{tmp}, id)
		s.Error(err)
	})
}

func TestDiscovery(t *testing.T) {
	suite.Run(t, new(DiscoverySuite))
}
