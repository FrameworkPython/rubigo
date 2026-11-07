package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
)

// SQLiteDriverName is the database/sql driver name used for sessions.
const SQLiteDriverName = "sqlite"

// ErrNoSQLiteDriver is returned when no "sqlite" database/sql driver
// has been linked into the program. Add a blank import of
//
//	_ "github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/storage/sqlitedriver"
//
// (or any pure-Go/cgo driver registered as "sqlite"), or use the file
// or in-memory stores that need no driver.
var ErrNoSQLiteDriver = errors.New(`storage: no "sqlite" database/sql driver registered; import storage/sqlitedriver or use NewFileStore`)

// driverRegistered reports whether a database/sql driver is available.
func driverRegistered(name string) bool {
	for _, d := range sql.Drivers() {
		if d == name {
			return true
		}
	}
	return false
}

// SQLiteSuffix is the file extension used for SQLite sessions.
const SQLiteSuffix = ".session"

// SQLiteStore is a file-backed Store backed by SQLite. Existing session
// files created by earlier releases stay compatible.
type SQLiteStore struct {
	path string
	conn *sql.DB
	mu   sync.Mutex
}

// NewSQLiteStore opens (or creates) a SQLite-backed session file. If
// `name` does not end in `.session` the suffix is appended automatically.
func NewSQLiteStore(name string) (*SQLiteStore, error) {
	if name == "" {
		return nil, fmt.Errorf("storage: empty session name")
	}
	if filepath.Ext(name) != SQLiteSuffix {
		name = name + SQLiteSuffix
	}
	if !driverRegistered(SQLiteDriverName) {
		return nil, ErrNoSQLiteDriver
	}
	conn, err := sql.Open(SQLiteDriverName, name)
	if err != nil {
		return nil, fmt.Errorf("storage: open sqlite: %w", err)
	}
	if err := initSchema(conn); err != nil {
		conn.Close()
		return nil, err
	}
	return &SQLiteStore{path: name, conn: conn}, nil
}

func initSchema(conn *sql.DB) error {
	_, err := conn.Exec(`create table if not exists version (version integer primary key)`)
	if err != nil {
		return fmt.Errorf("storage: create version table: %w", err)
	}
	_, err = conn.Exec(`insert or ignore into version(version) values (1)`)
	if err != nil {
		return fmt.Errorf("storage: seed version: %w", err)
	}
	_, err = conn.Exec(`create table if not exists session (
		phone text primary key,
		auth text,
		guid text,
		agent text,
		private_key text)`)
	if err != nil {
		return fmt.Errorf("storage: create session table: %w", err)
	}
	return nil
}

// Load returns the single stored session row, or nil if empty.
func (s *SQLiteStore) Load() (*Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.conn.QueryRow(`select phone, auth, guid, agent, private_key from session limit 1`)
	var rec Record
	err := row.Scan(&rec.Phone, &rec.Auth, &rec.GUID, &rec.UserAgent, &rec.PrivateKey)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("storage: scan row: %w", err)
	}
	return &rec, nil
}

// Save upserts the row identified by rec.Phone (or "(default)" when empty).
func (s *SQLiteStore) Save(rec Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	phone := rec.Phone
	if phone == "" {
		phone = "(default)"
	}
	_, err := s.conn.Exec(`insert or replace into session (phone, auth, guid, agent, private_key) values (?, ?, ?, ?, ?)`,
		phone, rec.Auth, rec.GUID, rec.UserAgent, rec.PrivateKey)
	if err != nil {
		return fmt.Errorf("storage: insert: %w", err)
	}
	return nil
}

// Close releases the underlying SQLite connection.
func (s *SQLiteStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}

// Path returns the on-disk file path.
func (s *SQLiteStore) Path() string { return s.path }
