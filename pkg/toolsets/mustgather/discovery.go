package mustgather

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/containers/kubernetes-mcp-server/pkg/klogutil"
	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"

	"golang.org/x/sync/singleflight"
)

// ArchiveInfo describes a discovered must-gather archive.
type ArchiveInfo struct {
	// ID is the stable, compact identifier (mg-XXXXYYYYYYYY) derived from Path.
	ID string
	// Path is the absolute filesystem path to the archive directory.
	Path string
	// Version is the OpenShift version recorded in the archive (may be empty).
	Version string
	// Timestamp is the collection timestamp recorded in the archive (may be empty).
	Timestamp string
}

// walkArchives scans the configured directories and returns the absolute paths
// of the must-gather archives found, in a stable order (directories in config
// order, entries lexical). A directory is treated as an archive if it contains a
// recognizable container directory; otherwise its immediate children are
// inspected. Non-existent or unreadable directories are skipped.
//
// This is the minimal traversal shared by ID resolution and listing: it does no
// metadata reads and computes no IDs, so it is cheap enough to run on demand.
func walkArchives(dirs []string) []string {
	var paths []string
	for _, root := range dirs {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		// A root that is itself an archive is not descended into.
		if mg.IsArchive(root) {
			paths = append(paths, absOr(root))
			continue
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, name := range names {
			candidate := filepath.Join(root, name)
			if mg.IsArchive(candidate) {
				paths = append(paths, absOr(candidate))
			}
		}
	}
	return paths
}

// absOr returns the absolute form of path, falling back to path itself if it
// cannot be resolved.
func absOr(path string) string {
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// discoverArchives returns the archives found under dirs, enriched with the
// version/timestamp metadata used by mustgather_list. Archives are deduped by ID
// (first-in-scan-order wins). Only this listing path pays for the metadata
// reads; ID resolution uses the cheaper registry scan.
func discoverArchives(ctx context.Context, dirs []string) []ArchiveInfo {
	logger := klogutil.FromContext(ctx)
	var archives []ArchiveInfo
	seen := make(map[string]bool)
	for _, path := range walkArchives(dirs) {
		id, err := mg.ArchiveIDFromLocalPath(path)
		if err != nil {
			klogutil.LogWarn(logger, "skipping must-gather archive with undecidable ID", klogutil.Field("path", path), klogutil.Err(err))
			continue
		}
		if seen[id] {
			klogutil.LogWarn(logger, "skipping must-gather archive with duplicate ID (first match wins)", klogutil.Field("path", path), klogutil.Field("archive_id", id))
			continue
		}
		seen[id] = true
		version, timestamp := mg.ReadArchiveMetadata(path)
		archives = append(archives, ArchiveInfo{
			ID:        id,
			Path:      path,
			Version:   version,
			Timestamp: timestamp,
		})
	}
	return archives
}

// mgRegistry is the archive cache for a single parsed toolset configuration
// (see Config.registry). It holds both the ID→path map produced by the last
// directory scan and the lazily-loaded providers keyed by path. Because a fresh
// registry is created per Config, a server reload starts from an empty cache;
// within a registry the configured dir-set is fixed for its lifetime.
//
// Immutability contract: a must-gather archive is a point-in-time snapshot and
// is treated as immutable between scans. A loaded provider is therefore reused
// until the next rescan, which clears every provider. The scan is the sole
// cache-invalidation boundary, so a provider is never staler than the last scan.
// Content changes to an existing archive at an unchanged path are not observed
// until a rescan is triggered (see resolvePath).
type mgRegistry struct {
	mu        sync.RWMutex
	gen       uint64                  // scan generation, bumped on every rescan
	byID      map[string]string       // id -> absolute path (from the last scan)
	providers map[string]*mg.Provider // path -> loaded provider
	flight    singleflight.Group      // coalesces concurrent expensive loads
}

// newRegistry returns an empty registry ready for use.
func newRegistry() *mgRegistry {
	return &mgRegistry{
		byID:      make(map[string]string),
		providers: make(map[string]*mg.Provider),
	}
}

// rescan rebuilds byID from a fresh directory scan and invalidates every cached
// provider. The caller must hold registry.mu for writing.
func (r *mgRegistry) rescan(ctx context.Context, dirs []string) {
	logger := klogutil.FromContext(ctx)
	byID := make(map[string]string)
	for _, path := range walkArchives(dirs) {
		id, err := mg.ArchiveIDFromLocalPath(path)
		if err != nil {
			klogutil.LogWarn(logger, "skipping must-gather archive with undecidable ID", klogutil.Field("path", path), klogutil.Err(err))
			continue
		}
		if _, dup := byID[id]; dup {
			klogutil.LogWarn(logger, "skipping must-gather archive with duplicate ID (first match wins)", klogutil.Field("path", path), klogutil.Field("archive_id", id))
			continue
		}
		byID[id] = path
	}
	r.byID = byID
	// Bump the generation and invalidate all cached providers: the scan is the
	// sole cache-invalidation boundary (see the mgRegistry immutability
	// contract). The generation lets an in-flight provider load detect that it
	// raced a rescan and must not repopulate the freshly cleared map.
	r.gen++
	r.providers = make(map[string]*mg.Provider)
}

// resolvePath resolves an archive ID to its absolute filesystem path. It rescans
// (which also invalidates all cached providers) when the ID is not in the
// current map or when a cached path no longer exists on disk. Between scans an
// ID→path hit is trusted without re-reading the archive. When the ID still
// cannot be found, the returned error lists the currently known IDs so the
// caller (LLM) can self-correct.
func (r *mgRegistry) resolvePath(ctx context.Context, dirs []string, id string) (string, error) {
	if err := mg.IsValidArchiveID(id); err != nil {
		return "", err
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("no must-gather directories configured; set mustgather_dirs in the [toolset_configs.\"openshift/mustgather\"] section of the config file to a directory containing must-gather archives")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if path, ok := r.byID[id]; ok {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
		// Cached path vanished (archive removed/replaced): rescan below.
	}

	// ID miss or vanished path: rescan once and retry.
	r.rescan(ctx, dirs)
	if path, ok := r.byID[id]; ok {
		return path, nil
	}

	known := make([]string, 0, len(r.byID))
	for k := range r.byID {
		known = append(known, k)
	}
	sort.Strings(known)
	if len(known) == 0 {
		return "", fmt.Errorf("must-gather archive %q not found; no archives discovered under the configured directories. Call mustgather_list to see available archives", id)
	}
	return "", fmt.Errorf("must-gather archive %q not found. Known archive IDs: %s. Call mustgather_list to see available archives", id, strings.Join(known, ", "))
}

// loadProvider returns a provider for the given absolute path, lazily
// initializing and caching it. Concurrent loads of the same path are coalesced
// via singleflight. The cache is invalidated wholesale on the next rescan (see
// the mgRegistry immutability contract).
//
// The scan generation is captured before the load and re-checked before the
// built provider is committed: if a rescan cleared the providers map while the
// load was in flight, the provider was built against a now-invalidated scan and
// must not repopulate the fresh map. It is still returned to this caller (the
// data is valid); it simply is not cached across the invalidation boundary. The
// generation is also part of the singleflight key so loads from before and after
// a rescan are never coalesced.
func (r *mgRegistry) loadProvider(path string) (*mg.Provider, error) {
	r.mu.RLock()
	if p, ok := r.providers[path]; ok {
		r.mu.RUnlock()
		return p, nil
	}
	gen := r.gen
	r.mu.RUnlock()

	key := fmt.Sprintf("%d\x00%s", gen, path)
	result, err, _ := r.flight.Do(key, func() (interface{}, error) {
		p, err := mg.NewProvider(path)
		if err != nil {
			return nil, fmt.Errorf("failed to load must-gather archive: %w", err)
		}
		r.mu.Lock()
		if r.gen == gen {
			r.providers[path] = p
		}
		r.mu.Unlock()
		return p, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*mg.Provider), nil
}
