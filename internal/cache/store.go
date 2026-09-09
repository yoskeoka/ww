package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// Store persists topology hints below one application cache directory.
type Store struct {
	root string
}

// New resolves the platform user-cache directory. It returns nil when the
// platform cannot provide a usable user-cache location.
func New() *Store {
	root, err := os.UserCacheDir()
	if err != nil || root == "" {
		return nil
	}
	return NewAt(filepath.Join(root, "ww"))
}

// NewAt creates a store at root. The directory is created lazily on Save.
func NewAt(root string) *Store {
	return &Store{root: filepath.Clean(root)}
}

// Root returns the configured cache directory for diagnostics and tests.
func (s *Store) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

// Load returns a validated topology hint, or a silent cache miss.
func (s *Store) Load(startDir string, sandbox bool) (Topology, bool) {
	if s == nil || !s.usableRoot(false) {
		return Topology{}, false
	}
	startPath, ok := canonicalPath(startDir)
	if !ok {
		return Topology{}, false
	}
	path := filepath.Join(s.root, fileName(startPath, sandbox))
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return Topology{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Topology{}, false
	}
	var topology Topology
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&topology); err != nil {
		return Topology{}, false
	}
	if !topology.ValidFor(startPath, sandbox) {
		return Topology{}, false
	}
	return topology, true
}

// Save atomically publishes a topology hint and ignores all cache failures.
func (s *Store) Save(topology Topology) {
	if s == nil || topology.SchemaVersion != SchemaVersion || topology.StartPath == "" {
		return
	}
	if !s.usableRoot(true) {
		return
	}
	target := filepath.Join(s.root, fileName(topology.StartPath, topology.Sandbox))
	tmp, err := os.CreateTemp(s.root, ".topology-*")
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
	if err := encoder.Encode(topology); err != nil {
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

func (s *Store) usableRoot(create bool) bool {
	if s.root == "" {
		return false
	}
	info, err := os.Lstat(s.root)
	if os.IsNotExist(err) && create {
		if err := os.MkdirAll(s.root, 0700); err != nil {
			return false
		}
		info, err = os.Lstat(s.root)
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false
	}
	if info.Mode().Perm()&0077 != 0 {
		if !create || os.Chmod(s.root, 0700) != nil {
			return false
		}
	}
	return true
}

func fileName(startPath string, sandbox bool) string {
	key := startPath + "\x00" + map[bool]string{true: "sandbox", false: "normal"}[sandbox]
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:]) + ".json"
}
