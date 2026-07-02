package mustgather

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	mg "github.com/containers/kubernetes-mcp-server/pkg/ocp/mustgather"
)

// ArchiveInfo describes a discovered must-gather archive.
type ArchiveInfo struct {
	// ID is the stable, compact identifier (mg-XXXX-YYYYYYYY) derived from Path.
	ID string
	// Path is the absolute filesystem path to the archive directory.
	Path string
	// Version is the OpenShift version recorded in the archive (may be empty).
	Version string
	// Timestamp is the collection timestamp recorded in the archive (may be empty).
	Timestamp string
}

// discoverArchives scans the configured directories and returns the archives
// found, in a stable order (directories in config order, entries lexically).
// A directory is treated as an archive if it contains a recognizable container
// directory; otherwise its immediate children are inspected. Non-existent or
// unreadable directories are skipped.
func discoverArchives(dirs []string) []ArchiveInfo {
	var archives []ArchiveInfo
	seen := make(map[string]bool) // dedupe by ID (first-in-scan-order wins)

	add := func(path string) {
		abs, err := filepath.Abs(path)
		if err != nil {
			abs = path
		}
		id, err := mg.ArchiveIDFromPath(abs)
		if err != nil || seen[id] {
			return
		}
		seen[id] = true
		version, timestamp := mg.ReadArchiveMetadata(abs)
		archives = append(archives, ArchiveInfo{
			ID:        id,
			Path:      abs,
			Version:   version,
			Timestamp: timestamp,
		})
	}

	for _, root := range dirs {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		// A root that is itself an archive is not descended into.
		if mg.IsArchive(root) {
			add(root)
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
				add(candidate)
			}
		}
	}
	return archives
}

// discoveryCache caches ID→path resolutions for a short window so that repeated
// tool calls don't re-scan the filesystem on every request. The cache is keyed
// by the directory set it was built from, so a different --mustgather-dirs never
// yields stale results.
type discoveryCache struct {
	mu     sync.Mutex
	scanAt time.Time
	dirs   []string          // directory set the cache was built from
	byID   map[string]string // id -> absolute path
}

const scanTTL = 30 * time.Second

var scanCache = &discoveryCache{byID: make(map[string]string)}

// rescan refreshes the cache for dirs. Caller must hold scanCache.mu.
func (c *discoveryCache) rescan(dirs []string) {
	archives := discoverArchives(dirs)
	byID := make(map[string]string, len(archives))
	for _, a := range archives {
		byID[a.ID] = a.Path
	}
	c.byID = byID
	c.dirs = append([]string(nil), dirs...)
	c.scanAt = time.Now()
}

// resolveArchivePath resolves an archive ID to its absolute filesystem path,
// scanning dirs when the cache is cold, stale, built from a different directory
// set, or missing the requested ID. A fresh-cache hit whose path no longer
// exists on disk triggers a re-scan. When the ID still cannot be found, the
// returned error lists the currently known IDs so the caller (LLM) can
// self-correct.
func resolveArchivePath(dirs []string, id string) (string, error) {
	if _, _, err := mg.ParseArchiveID(id); err != nil {
		return "", err
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("no must-gather directories configured; set --mustgather-dirs (or mustgather_dirs in config) to a directory containing must-gather archives")
	}

	scanCache.mu.Lock()
	defer scanCache.mu.Unlock()

	fresh := time.Since(scanCache.scanAt) < scanTTL && slices.Equal(scanCache.dirs, dirs)
	if fresh {
		if path, ok := scanCache.byID[id]; ok {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
			// Cached path vanished (archive removed): re-scan below.
		}
	}

	// Cold/stale cache, different dirs, or an ID miss: re-scan and retry once.
	scanCache.rescan(dirs)
	if path, ok := scanCache.byID[id]; ok {
		return path, nil
	}

	known := make([]string, 0, len(scanCache.byID))
	for k := range scanCache.byID {
		known = append(known, k)
	}
	sort.Strings(known)
	if len(known) == 0 {
		return "", fmt.Errorf("must-gather archive %q not found; no archives discovered under the configured directories. Call mustgather_list to see available archives", id)
	}
	return "", fmt.Errorf("must-gather archive %q not found. Known archive IDs: %s. Call mustgather_list to see available archives", id, strings.Join(known, ", "))
}
