package mustgather

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// MustGatherArchiveIDPattern matches the canonical must-gather archive ID form:
// "mg-" + 12 hex digits (truncated SHA-256 of the archive URI).
var MustGatherArchiveIDPattern = regexp.MustCompile(`^mg-[0-9a-f]{12}$`)

// LocalURIPrefix is the source-URI prefix for local-filesystem archives.
// Archive IDs are derived from the full source URI so that remote sources
// (e.g. gs://, s3://) can be added without changing the ID scheme.
const LocalURIPrefix = "local://"

// ArchiveIDFromURI derives a compact, deterministic archive ID from an
// archive source URI (e.g. local:///data/archives/mg..., gs://bucket/mg...)
// by truncating the SHA-256 of the full URI:
//
//	ID = "mg-" + hex(sha256(uri))[:12]
//
// The 12 hex digits (48 bits) give a collision probability of ~2^-48 between
// any two distinct URIs. The URI is hashed as-is (only trailing slashes are
// trimmed); no normalization is performed. An error is returned for an empty
// URI.
func ArchiveIDFromURI(uri string) (string, error) {
	trimmed := strings.TrimRight(uri, "/")
	if trimmed == "" {
		return "", fmt.Errorf("cannot derive archive ID from empty URI")
	}

	sum := sha256.Sum256([]byte(trimmed))
	h := hex.EncodeToString(sum[:6])
	return fmt.Sprintf("mg-%s", h), nil
}

// ArchiveIDFromLocalPath derives an archive ID from a local filesystem path by
// hashing its local:// source URI, i.e. ArchiveIDFromURI(LocalURIPrefix+path).
// An error is returned for an empty path.
func ArchiveIDFromLocalPath(path string) (string, error) {
	trimmed := strings.TrimRight(path, "/")
	if trimmed == "" {
		return "", fmt.Errorf("cannot derive archive ID from empty path")
	}
	return ArchiveIDFromURI(LocalURIPrefix + trimmed)
}

// IsValidArchiveID validates an archive ID. It returns an error if id is not
// of the form mg-XXXX-YYYYYYYY (12 hex digits).
func IsValidArchiveID(id string) error {
	if !MustGatherArchiveIDPattern.MatchString(id) {
		return fmt.Errorf("invalid must-gather archive ID %q: expected format mg-XXXXYYYYYYYY (e.g. mg-384226d712f0)", id)
	}
	return nil
}
