package mustgather

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// underContainerDir cleans relPath, joins it onto the archive container
// directory, and verifies the result stays within that directory. It is the
// shared traversal guard for the generic archive-file accessors, mirroring the
// clean+prefix check used by ReadETCDFile.
func (p *Provider) underContainerDir(relPath string) (string, error) {
	root := filepath.Clean(p.metadata.ContainerDir)
	full := filepath.Clean(filepath.Join(root, relPath))
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid archive path: %s", relPath)
	}
	return full, nil
}

// ReadArchiveFile reads a file at relPath, interpreted relative to the archive
// container directory. Files with a .gz suffix (but not .tar.gz) are
// transparently gzip-decompressed. relPath is guarded against directory
// traversal.
func (p *Provider) ReadArchiveFile(relPath string) ([]byte, error) {
	full, err := p.underContainerDir(relPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(full)
	if err != nil {
		return nil, fmt.Errorf("archive file %s not found: %w", relPath, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not a file", relPath)
	}

	if strings.HasSuffix(full, ".gz") && !strings.HasSuffix(full, ".tar.gz") {
		content, err := readGzipFile(full)
		if err != nil {
			return nil, err
		}
		return []byte(content), nil
	}

	return os.ReadFile(full)
}
