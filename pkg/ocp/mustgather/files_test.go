package mustgather

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type FilesSuite struct {
	suite.Suite
	archiveRoot  string
	containerDir string
	provider     *Provider
}

func (s *FilesSuite) SetupTest() {
	s.archiveRoot = s.T().TempDir()
	s.containerDir = filepath.Join(s.archiveRoot, "quay-io-openshift-release-dev-sha256-deadbeef")
	s.Require().NoError(os.MkdirAll(filepath.Join(s.containerDir, "host_service_logs", "masters"), 0o755))
	s.Require().NoError(os.MkdirAll(filepath.Join(s.containerDir, "static-pods", "kube-apiserver"), 0o755))

	// Plain text log.
	s.Require().NoError(os.WriteFile(
		filepath.Join(s.containerDir, "host_service_logs", "masters", "kubelet_service.log"),
		[]byte("kubelet started\nkubelet ready\n"), 0o644))

	// Gzipped termination log.
	s.writeGzip(
		filepath.Join(s.containerDir, "static-pods", "kube-apiserver", "node-a-termination.log.gz"),
		"terminated gracefully\n")

	// Metadata so version/timestamp load is exercised (optional).
	s.Require().NoError(os.WriteFile(filepath.Join(s.containerDir, "version"), []byte("4.19.0\n"), 0o644))

	p, err := NewProvider(s.archiveRoot)
	s.Require().NoError(err)
	s.provider = p
}

func (s *FilesSuite) writeGzip(path, content string) {
	f, err := os.Create(path)
	s.Require().NoError(err)
	defer func() { _ = f.Close() }()
	w := gzip.NewWriter(f)
	_, err = w.Write([]byte(content))
	s.Require().NoError(err)
	s.Require().NoError(w.Close())
}

func (s *FilesSuite) TestReadArchiveFile() {
	s.Run("reads a plain text file", func() {
		data, err := s.provider.ReadArchiveFile("host_service_logs/masters/kubelet_service.log")
		s.Require().NoError(err)
		s.Equal("kubelet started\nkubelet ready\n", string(data))
	})

	s.Run("decompresses a .gz file", func() {
		data, err := s.provider.ReadArchiveFile("static-pods/kube-apiserver/node-a-termination.log.gz")
		s.Require().NoError(err)
		s.Equal("terminated gracefully\n", string(data))
	})

	s.Run("edge cases", func() {
		s.Run("returns error for a missing file", func() {
			_, err := s.provider.ReadArchiveFile("host_service_logs/masters/nope.log")
			s.Error(err)
		})
		s.Run("returns error when the path is a directory", func() {
			_, err := s.provider.ReadArchiveFile("host_service_logs")
			s.Error(err)
		})
		s.Run("rejects directory traversal", func() {
			_, err := s.provider.ReadArchiveFile("../../../../etc/passwd")
			s.Error(err)
		})
	})
}

func (s *FilesSuite) TestListArchiveDir() {
	s.Run("lists nested entries relative to the directory", func() {
		entries, err := s.provider.ListArchiveDir("host_service_logs")
		s.Require().NoError(err)

		byPath := map[string]ArchiveEntry{}
		for _, e := range entries {
			byPath[e.Path] = e
		}
		s.Contains(byPath, "masters")
		s.True(byPath["masters"].IsDir, "masters should be reported as a directory")
		s.Contains(byPath, "masters/kubelet_service.log")
		s.False(byPath["masters/kubelet_service.log"].IsDir)
		s.Equal(int64(len("kubelet started\nkubelet ready\n")), byPath["masters/kubelet_service.log"].Size)
	})

	s.Run("edge cases", func() {
		s.Run("returns error for a missing directory", func() {
			_, err := s.provider.ListArchiveDir("does_not_exist")
			s.Error(err)
		})
		s.Run("returns error when the path is a file", func() {
			_, err := s.provider.ListArchiveDir("version")
			s.Error(err)
		})
		s.Run("rejects directory traversal", func() {
			_, err := s.provider.ListArchiveDir("../..")
			s.Error(err)
		})
	})
}

func TestFiles(t *testing.T) {
	suite.Run(t, new(FilesSuite))
}
