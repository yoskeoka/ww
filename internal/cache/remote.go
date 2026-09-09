package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RemoteCacheSchemaVersion identifies the on-disk positive remote evidence format.
const RemoteCacheSchemaVersion = 1

// RemotePositiveTTL bounds how long a successful remote presence observation
// may be used without another live query.
const RemotePositiveTTL = 30 * time.Second

// RemoteEntry contains only positive evidence from one complete remote query.
// Its identity is a digest so raw URLs and credentials never reach disk.
type RemoteEntry struct {
	SchemaVersion int       `json:"schema_version"`
	Identity      string    `json:"identity"`
	ObservedAt    time.Time `json:"observed_at"`
	Branches      []string  `json:"branches"`
}

// NewAtWithClock creates a cache store whose remote evidence uses now. It is
// primarily a deterministic test seam; normal callers should use NewAt or New.
func NewAtWithClock(root string, now func() time.Time) *Store {
	store := NewAt(root)
	if now != nil {
		store.now = now
	}
	return store
}

// LoadRemote returns positive remote evidence when it is fresh, matches the
// repository/remote identity, and covers every requested candidate branch.
func (s *Store) LoadRemote(commonDir, remote string, urls, candidates []string) (map[string]struct{}, bool) {
	identity, ok := remoteIdentity(commonDir, remote, urls)
	if !ok || s == nil || !s.usableRoot(false) {
		return nil, false
	}
	path := filepath.Join(s.root, remoteFileName(identity))
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var entry RemoteEntry
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&entry); err != nil {
		return nil, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, false
	}
	if !entry.valid(identity, s.currentTime()) {
		return nil, false
	}

	branches := make(map[string]struct{}, len(entry.Branches))
	for _, branch := range entry.Branches {
		branches[branch] = struct{}{}
	}
	for _, candidate := range candidates {
		if _, present := branches[candidate]; !present {
			return nil, false
		}
	}
	return branches, true
}

// SaveRemote publishes a complete successful remote response as positive
// evidence. It ignores cache failures so callers retain uncached behavior.
func (s *Store) SaveRemote(commonDir, remote string, urls []string, branches map[string]struct{}) {
	identity, ok := remoteIdentity(commonDir, remote, urls)
	if !ok || s == nil || !s.usableRoot(true) {
		return
	}

	branchNames := make([]string, 0, len(branches))
	for branch := range branches {
		if !validRemoteBranchName(branch) {
			return
		}
		branchNames = append(branchNames, branch)
	}
	sort.Strings(branchNames)
	entry := RemoteEntry{
		SchemaVersion: RemoteCacheSchemaVersion,
		Identity:      identity,
		ObservedAt:    s.currentTime(),
		Branches:      branchNames,
	}
	if !entry.valid(identity, entry.ObservedAt) {
		return
	}

	target := filepath.Join(s.root, remoteFileName(identity))
	tmp, err := os.CreateTemp(s.root, ".remote-*")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return
	}
	encoder := json.NewEncoder(tmp)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(entry); err != nil {
		_ = tmp.Close()
		return
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return
	}
	if err := tmp.Close(); err != nil {
		return
	}
	_ = os.Rename(tmpName, target)
}

func (s *Store) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

func (entry RemoteEntry) valid(identity string, now time.Time) bool {
	if entry.SchemaVersion != RemoteCacheSchemaVersion || entry.Identity != identity || entry.Identity == "" || entry.Branches == nil {
		return false
	}
	if entry.ObservedAt.IsZero() || entry.ObservedAt.After(now) || now.Sub(entry.ObservedAt) >= RemotePositiveTTL {
		return false
	}
	for i, branch := range entry.Branches {
		if !validRemoteBranchName(branch) || (i > 0 && entry.Branches[i-1] >= branch) {
			return false
		}
	}
	return true
}

func remoteIdentity(commonDir, remote string, urls []string) (string, bool) {
	commonPath, ok := canonicalPath(commonDir)
	if !ok || commonPath == "" || remote == "" || len(urls) == 0 {
		return "", false
	}
	hash := sha256.New()
	writePart := func(value string) {
		_, _ = fmt.Fprintf(hash, "%d:", len(value))
		_, _ = io.WriteString(hash, value)
	}
	writePart(commonPath)
	writePart(remote)
	for _, url := range urls {
		if url == "" {
			return "", false
		}
		writePart(url)
	}
	return hex.EncodeToString(hash.Sum(nil)), true
}

func remoteFileName(identity string) string {
	return identity + ".remote.json"
}

func validRemoteBranchName(branch string) bool {
	return branch != "" && !strings.ContainsAny(branch, "\x00\r\n")
}
