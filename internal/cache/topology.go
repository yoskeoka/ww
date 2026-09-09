// Package cache stores optional workspace-discovery topology hints.
package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

// SchemaVersion identifies the on-disk topology format.
const SchemaVersion = 1

// Repo is the topology-only identity of one discovered repository.
type Repo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// MarkerFingerprint describes the immediate child .git marker.
type MarkerFingerprint struct {
	Kind     string `json:"kind"`
	Size     int64  `json:"size,omitempty"`
	ModTime  int64  `json:"mod_time,omitempty"`
	Contents string `json:"contents,omitempty"`
}

// EntryFingerprint describes one immediate child of a discovery candidate.
type EntryFingerprint struct {
	Name string            `json:"name"`
	Kind string            `json:"kind"`
	Git  MarkerFingerprint `json:"git"`
}

// CandidateFingerprint records the bounded directory scan used by discovery.
type CandidateFingerprint struct {
	Path    string             `json:"path"`
	Entries []EntryFingerprint `json:"entries"`
}

// Topology contains only data that can be safely revalidated before use.
type Topology struct {
	SchemaVersion int                    `json:"schema_version"`
	StartPath     string                 `json:"start_path"`
	Sandbox       bool                   `json:"sandbox"`
	Root          string                 `json:"root"`
	MainRoot      string                 `json:"main_root,omitempty"`
	Mode          string                 `json:"mode"`
	Repos         []Repo                 `json:"repos"`
	Candidates    []CandidateFingerprint `json:"candidates"`
}

// DiscoveryCache is the optional seam used by workspace discovery.
type DiscoveryCache interface {
	Load(startDir string, sandbox bool) (Topology, bool)
	Save(topology Topology)
}

// NewTopology captures a validated topology hint from a completed discovery.
// It returns false when any candidate cannot be fingerprinted safely.
func NewTopology(startDir string, sandbox bool, root, mainRoot, mode string, repos []Repo, candidatePaths []string) (Topology, bool) {
	startPath, ok := canonicalPath(startDir)
	if !ok {
		return Topology{}, false
	}
	rootPath, ok := canonicalPath(root)
	if !ok {
		return Topology{}, false
	}
	if rootPath == "" || mode == "" {
		return Topology{}, false
	}

	topology := Topology{
		SchemaVersion: SchemaVersion,
		StartPath:     startPath,
		Sandbox:       sandbox,
		Root:          rootPath,
		Mode:          mode,
		Repos:         make([]Repo, 0, len(repos)),
	}
	if mainRoot != "" {
		topology.MainRoot, ok = canonicalPath(mainRoot)
		if !ok {
			return Topology{}, false
		}
	}

	for _, repo := range repos {
		path, pathOK := canonicalPath(repo.Path)
		if !pathOK || path == "" {
			return Topology{}, false
		}
		name := repo.Name
		if name == "" {
			name = filepath.Base(path)
		}
		topology.Repos = append(topology.Repos, Repo{Name: name, Path: path})
	}
	sort.Slice(topology.Repos, func(i, j int) bool {
		if topology.Repos[i].Name == topology.Repos[j].Name {
			return topology.Repos[i].Path < topology.Repos[j].Path
		}
		return topology.Repos[i].Name < topology.Repos[j].Name
	})

	seen := make(map[string]struct{}, len(candidatePaths))
	for _, candidate := range candidatePaths {
		path, pathOK := canonicalPath(candidate)
		if !pathOK || path == "" {
			return Topology{}, false
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		fingerprint, fingerprintOK := fingerprintDirectory(path)
		if !fingerprintOK {
			return Topology{}, false
		}
		topology.Candidates = append(topology.Candidates, CandidateFingerprint{
			Path:    path,
			Entries: fingerprint,
		})
	}
	if len(topology.Candidates) == 0 {
		return Topology{}, false
	}
	sort.Slice(topology.Candidates, func(i, j int) bool {
		return topology.Candidates[i].Path < topology.Candidates[j].Path
	})
	return topology, true
}

// ValidFor reports whether the hint still describes the current bounded tree.
func (topology Topology) ValidFor(startDir string, sandbox bool) bool {
	if topology.SchemaVersion != SchemaVersion || topology.Sandbox != sandbox || topology.Mode == "" {
		return false
	}
	startPath, ok := canonicalPath(startDir)
	if !ok || startPath != topology.StartPath {
		return false
	}
	if !existingCanonicalDirectory(topology.Root) {
		return false
	}
	if topology.MainRoot != "" && !existingCanonicalDirectory(topology.MainRoot) {
		return false
	}
	if sandbox && topology.MainRoot != "" && topology.Root != topology.StartPath && topology.Root != topology.MainRoot {
		return false
	}

	candidatePaths := expectedCandidatePaths(topology.StartPath, topology.MainRoot, sandbox)
	if len(topology.Candidates) == 0 {
		return false
	}
	candidates := make(map[string]CandidateFingerprint, len(topology.Candidates))
	for _, candidate := range topology.Candidates {
		if _, exists := candidates[candidate.Path]; exists {
			return false
		}
		if !containsString(candidatePaths, candidate.Path) || !existingCanonicalDirectory(candidate.Path) {
			return false
		}
		current, currentOK := fingerprintDirectory(candidate.Path)
		if !currentOK || !equalEntries(current, candidate.Entries) {
			return false
		}
		candidates[candidate.Path] = candidate
	}

	if !containsString(candidatePaths, topology.Root) {
		return false
	}
	if len(topology.Repos) == 0 {
		return false
	}
	for i, repo := range topology.Repos {
		if repo.Name == "" || repo.Path == "" || filepath.Base(repo.Path) != repo.Name || !existingCanonicalDirectory(repo.Path) {
			return false
		}
		if i > 0 && (topology.Repos[i-1].Name > repo.Name || (topology.Repos[i-1].Name == repo.Name && topology.Repos[i-1].Path >= repo.Path)) {
			return false
		}
		if repo.Path != topology.Root && !directChild(topology.Root, repo.Path) {
			return false
		}
		info, err := os.Lstat(repo.Path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return false
		}
	}
	return true
}

func expectedCandidatePaths(startPath, mainRoot string, sandbox bool) []string {
	paths := []string{startPath}
	if mainRoot == "" {
		return paths
	}
	add := func(path string) {
		if path == "" || containsString(paths, path) {
			return
		}
		paths = append(paths, path)
	}
	add(mainRoot)
	if sandbox {
		return paths
	}
	parent := filepath.Dir(mainRoot)
	grandparent := filepath.Dir(parent)
	add(parent)
	add(grandparent)
	sort.Strings(paths)
	return paths
}

func directChild(parent, child string) bool {
	return parent != child && filepath.Dir(child) == parent
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func existingCanonicalDirectory(path string) bool {
	canonical, ok := canonicalPath(path)
	if !ok || canonical != path {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func canonicalPath(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	evaluated, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", false
	}
	return filepath.Clean(evaluated), true
}

func fingerprintDirectory(path string) ([]EntryFingerprint, bool) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, false
	}
	fingerprints := make([]EntryFingerprint, 0, len(entries))
	for _, entry := range entries {
		kind := entryKind(entry)
		if kind != "dir" && kind != "symlink" {
			continue
		}
		fingerprint := EntryFingerprint{Name: entry.Name(), Kind: kind}
		if kind == "dir" {
			marker, markerOK := fingerprintMarker(filepath.Join(path, entry.Name(), ".git"))
			if !markerOK {
				return nil, false
			}
			fingerprint.Git = marker
		}
		fingerprints = append(fingerprints, fingerprint)
	}
	sort.Slice(fingerprints, func(i, j int) bool { return fingerprints[i].Name < fingerprints[j].Name })
	return fingerprints, true
}

func entryKind(entry os.DirEntry) string {
	if entry.Type()&os.ModeSymlink != 0 {
		return "symlink"
	}
	if entry.IsDir() {
		return "dir"
	}
	if entry.Type().IsRegular() {
		return "file"
	}
	return "other"
}

func fingerprintMarker(path string) (MarkerFingerprint, bool) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return MarkerFingerprint{Kind: "missing"}, true
	}
	if err != nil {
		return MarkerFingerprint{}, false
	}
	marker := MarkerFingerprint{Size: info.Size(), ModTime: info.ModTime().UnixNano()}
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		marker.Kind = "symlink"
	case info.IsDir():
		marker.Kind = "dir"
	case info.Mode().IsRegular():
		marker.Kind = "file"
		contents, err := os.ReadFile(path)
		if err != nil {
			return MarkerFingerprint{}, false
		}
		hash := sha256.Sum256(contents)
		marker.Contents = hex.EncodeToString(hash[:])
	default:
		marker.Kind = "other"
	}
	return marker, true
}

func equalEntries(left, right []EntryFingerprint) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
