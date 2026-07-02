package mustgather

import (
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
)

// archiveIDPattern matches the canonical must-gather archive ID form:
// "mg-" + 4 hex digits (parent hash) + "-" + 8 hex digits (leaf hash).
var archiveIDPattern = regexp.MustCompile(`^mg-([0-9a-f]{4})-([0-9a-f]{8})$`)

// ArchiveIDFromPath derives a compact, deterministic archive ID from a
// filesystem path. Both the path's parent directory and its leaf name are
// hashed with FNV-1a (32-bit); the parent contributes 16 bits (disambiguates
// the containing directory) and the leaf 32 bits (identifies the archive):
//
//	ID = "mg-" + shortHash(parent)[:4hex] + "-" + shortHash(leaf)
//
// The raw path string is hashed as-is (only trailing slashes are trimmed); no
// scheme normalization is performed, so callers must strip any "local://"-style
// scheme before calling. An error is returned for an empty path.
func ArchiveIDFromPath(path string) (string, error) {
	trimmed := strings.TrimRight(path, "/")
	if trimmed == "" {
		return "", fmt.Errorf("cannot derive archive ID from empty path")
	}

	parent, leaf := trimmed, ""
	if i := strings.LastIndex(trimmed, "/"); i >= 0 {
		parent, leaf = trimmed[:i], trimmed[i+1:]
	} else {
		// No separator: the whole string is the leaf, parent is empty.
		parent, leaf = "", trimmed
	}

	return fmt.Sprintf("mg-%s-%s", shortHash(parent)[:4], shortHash(leaf)), nil
}

// ParseArchiveID validates an archive ID and returns its parent and leaf hash
// components. It returns an error if id is not of the form mg-XXXX-YYYYYYYY.
func ParseArchiveID(id string) (parentHash, leafHash string, err error) {
	m := archiveIDPattern.FindStringSubmatch(id)
	if m == nil {
		return "", "", fmt.Errorf("invalid must-gather archive ID %q: expected format mg-XXXX-YYYYYYYY (e.g. mg-3842-26d712f0)", id)
	}
	return m[1], m[2], nil
}

// shortHash returns the 8-hex-digit FNV-1a 32-bit hash of s.
func shortHash(s string) string {
	hash := fnv.New32a()
	hash.Write([]byte(s))
	return fmt.Sprintf("%08x", hash.Sum32())
}
