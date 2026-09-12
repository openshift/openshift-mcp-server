package mustgather

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ArchiveIDSuite struct {
	suite.Suite
}

func (s *ArchiveIDSuite) TestArchiveIDFromPath() {
	s.Run("valid paths", func() {
		s.Run("matches the reference example", func() {
			id, err := ArchiveIDFromPath("path/to/some/directory/must-gather.local.ocp412.20260911.9f2a")
			s.Require().NoError(err)
			s.Equal("mg-3cfd-1d447b93", id)
		})
		s.Run("trailing slash is ignored", func() {
			id, err := ArchiveIDFromPath("path/to/some/directory/must-gather.local.ocp412.20260911.9f2a/")
			s.Require().NoError(err)
			s.Equal("mg-3cfd-1d447b93", id)
		})
		s.Run("absolute path", func() {
			id, err := ArchiveIDFromPath("/data/archives/must-gather.local")
			s.Require().NoError(err)
			s.Regexp(archiveIDPattern, id)
		})
		s.Run("no parent directory hashes empty parent", func() {
			id, err := ArchiveIDFromPath("must-gather.local")
			s.Require().NoError(err)
			// parent is "" -> shortHash("")=811c9dc5 -> "811c"
			s.Regexp(`^mg-811c-[0-9a-f]{8}$`, id)
		})
		s.Run("differing parents yield differing IDs", func() {
			a, err := ArchiveIDFromPath("/rootA/sub/must-gather.x")
			s.Require().NoError(err)
			b, err := ArchiveIDFromPath("/rootB/sub/must-gather.x")
			s.Require().NoError(err)
			s.NotEqual(a, b)
		})
	})

	s.Run("edge cases", func() {
		s.Run("empty path returns error", func() {
			_, err := ArchiveIDFromPath("")
			s.Error(err)
		})
		s.Run("only slashes returns error", func() {
			_, err := ArchiveIDFromPath("///")
			s.Error(err)
		})
	})
}

func (s *ArchiveIDSuite) TestParseArchiveID() {
	s.Run("valid ID", func() {
		parent, leaf, err := ParseArchiveID("mg-3842-26d712f0")
		s.Require().NoError(err)
		s.Equal("3842", parent)
		s.Equal("26d712f0", leaf)
	})
	s.Run("invalid IDs return error", func() {
		for _, id := range []string{"", "mg-384-26d712f0", "mg-3842-26d712f", "3842-26d712f0", "mg-XYZW-26d712f0", "must-gather"} {
			_, _, err := ParseArchiveID(id)
			s.Error(err, "expected error for %q", id)
		}
	})
	s.Run("round-trips with ArchiveIDFromPath", func() {
		id, err := ArchiveIDFromPath("/data/must-gather.local")
		s.Require().NoError(err)
		_, _, err = ParseArchiveID(id)
		s.Require().NoError(err)
	})
}

func TestArchiveID(t *testing.T) {
	suite.Run(t, new(ArchiveIDSuite))
}
