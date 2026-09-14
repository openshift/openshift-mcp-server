package mustgather

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ArchiveIDSuite struct {
	suite.Suite
}

func (s *ArchiveIDSuite) TestArchiveIDFromURI() {
	s.Run("valid URIs", func() {
		s.Run("local path", func() {
			id, err := ArchiveIDFromURI("local://path/to/some/directory/must-gather.local.ocp412.20260911.9f2a")
			s.Require().NoError(err)
			s.Equal("mg-400d1ad63e0f", id)
		})
		s.Run("remote sources hash like any other URI", func() {
			id, err := ArchiveIDFromURI("gs://must-gather-bucket/archives/must-gather.local.ocp412")
			s.Require().NoError(err)
			s.Regexp(MustGatherArchiveIDPattern, id)
		})
		s.Run("trailing slash is ignored", func() {
			a, err := ArchiveIDFromURI("local:///data/archives/must-gather.local")
			s.Require().NoError(err)
			b, err := ArchiveIDFromURI("local:///data/archives/must-gather.local/")
			s.Require().NoError(err)
			s.Equal(a, b)
		})
		s.Run("deterministic", func() {
			a, err := ArchiveIDFromURI("local:///data/archives/must-gather.local")
			s.Require().NoError(err)
			b, err := ArchiveIDFromURI("local:///data/archives/must-gather.local")
			s.Require().NoError(err)
			s.Equal(a, b)
		})
	})

	s.Run("edge cases", func() {
		s.Run("empty URI returns error", func() {
			_, err := ArchiveIDFromURI("")
			s.Error(err)
		})
		s.Run("only slashes returns error", func() {
			_, err := ArchiveIDFromURI("///")
			s.Error(err)
		})
	})
}

func (s *ArchiveIDSuite) TestArchiveIDFromPath() {
	s.Run("valid paths", func() {
		s.Run("matches the reference example", func() {
			id, err := ArchiveIDFromLocalPath("path/to/some/directory/must-gather.local.ocp412.20260911.9f2a")
			s.Require().NoError(err)
			s.Equal("mg-400d1ad63e0f", id)
		})
		s.Run("trailing slash is ignored", func() {
			id, err := ArchiveIDFromLocalPath("path/to/some/directory/must-gather.local.ocp412.20260911.9f2a/")
			s.Require().NoError(err)
			s.Equal("mg-400d1ad63e0f", id)
		})
		s.Run("absolute path", func() {
			id, err := ArchiveIDFromLocalPath("/data/archives/must-gather.local")
			s.Require().NoError(err)
			s.Regexp(MustGatherArchiveIDPattern, id)
		})
		s.Run("differing parents yield differing IDs", func() {
			a, err := ArchiveIDFromLocalPath("/rootA/sub/must-gather.x")
			s.Require().NoError(err)
			b, err := ArchiveIDFromLocalPath("/rootB/sub/must-gather.x")
			s.Require().NoError(err)
			s.NotEqual(a, b)
		})
	})

	s.Run("edge cases", func() {
		s.Run("empty path returns error", func() {
			_, err := ArchiveIDFromLocalPath("")
			s.Error(err)
		})
		s.Run("only slashes returns error", func() {
			_, err := ArchiveIDFromLocalPath("///")
			s.Error(err)
		})
	})

	s.Run("equivalent to the local:// source URI", func() {
		id, err := ArchiveIDFromLocalPath("/data/archives/must-gather.local")
		s.Require().NoError(err)
		uriID, err := ArchiveIDFromURI("local:///data/archives/must-gather.local")
		s.Require().NoError(err)
		s.Equal(uriID, id)
	})
}

func (s *ArchiveIDSuite) TestIsValidArchiveID() {
	s.Run("valid ID", func() {
		s.NoError(IsValidArchiveID("mg-384226d712f0"))
	})
	s.Run("invalid IDs return error", func() {
		for _, id := range []string{"", "mg-38426d712f0", "mg-384226d712f", "384226d712f0", "mg-XYZW26d712f0", "must-gather"} {
			s.Error(IsValidArchiveID(id), "expected error for %q", id)
		}
	})
	s.Run("round-trips with ArchiveIDFromPath", func() {
		id, err := ArchiveIDFromLocalPath("/data/must-gather.local")
		s.Require().NoError(err)
		s.NoError(IsValidArchiveID(id))
	})
}

func TestArchiveID(t *testing.T) {
	suite.Run(t, new(ArchiveIDSuite))
}
