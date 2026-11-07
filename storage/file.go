package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileSuffix is the file extension used for JSON file sessions.
const FileSuffix = ".bin"

// FileStore persists the session record as a base64-encoded JSON array
// inside a small file. It is the lightweight alternative to SQLite.
type FileStore struct {
	path string
	mu   sync.Mutex
}

// NewFileStore opens (or creates) a JSON file session. If `name` does
// not end in `.bin` the suffix is appended.
func NewFileStore(name string) (*FileStore, error) {
	if name == "" {
		return nil, fmt.Errorf("storage: empty session name")
	}
	if filepath.Ext(name) != FileSuffix {
		name = name + FileSuffix
	}
	return &FileStore{path: name}, nil
}

// Load reads the record from disk. Returns nil (no error) when the
// file does not exist yet.
func (s *FileStore) Load() (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("storage: read file: %w", err)
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("storage: base64 decode: %w", err)
	}
	var arr []string
	if err := json.Unmarshal(decoded, &arr); err != nil {
		return nil, fmt.Errorf("storage: json decode: %w", err)
	}
	rec := &Record{}
	if len(arr) >= 1 {
		rec.Phone = arr[0]
	}
	if len(arr) >= 2 {
		rec.Auth = arr[1]
	}
	if len(arr) >= 3 {
		rec.GUID = arr[2]
	}
	if len(arr) >= 4 {
		rec.UserAgent = arr[3]
	}
	if len(arr) >= 5 {
		rec.PrivateKey = arr[4]
	}
	return rec, nil
}

// Save writes the record to disk (base64-encoded JSON array). The
// write is atomic: we write to a temp file first, then rename it over
// the target so a crash mid-write cannot corrupt the existing store.
func (s *FileStore) Save(rec Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	arr := []string{rec.Phone, rec.Auth, rec.GUID, rec.UserAgent, rec.PrivateKey}
	raw, err := json.Marshal(arr)
	if err != nil {
		return fmt.Errorf("storage: json encode: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(raw)
	// Write to a temp file then rename for atomicity.
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, []byte(encoded), 0o600); err != nil {
		return fmt.Errorf("storage: write tmp file: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("storage: rename tmp file: %w", err)
	}
	return nil
}

// Close is a no-op for file stores.
func (s *FileStore) Close() error { return nil }

// Path returns the on-disk file path.
func (s *FileStore) Path() string { return s.path }
