package database

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database connection
type DB struct {
	conn *sql.DB
}

// LogEntry represents a logged request/response
type LogEntry struct {
	ID               int64
	Timestamp        time.Time
	Endpoint         string
	Method           string
	Model            string
	Prompt           string
	Response         string
	StatusCode       int
	LatencyMs        int64
	Stream           bool
	BackendType      string
	Error            string
	FrontendURL      string // Frontend URL that received the request
	BackendURL       string // Backend URL that was called
	FrontendRequest  string // Raw frontend request JSON
	FrontendResponse string // Raw frontend response JSON
	BackendRequest   string // Raw backend request JSON
	BackendResponse  string // Raw backend response data
	LastMessage      string // Last message in the prompt (user input or tool result)
}

// New creates a new database connection and initializes the schema
func New(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.configureConnection(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to configure database: %w", err)
	}
	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// sqliteDSN applies connection-local pragmas to every connection opened by
// database/sql. SQLite's busy timeout is connection-local, so setting it once
// with Exec is not sufficient when the pool grows under concurrent traffic.
func sqliteDSN(path string) string {
	if path == ":memory:" {
		return "file::memory:?cache=shared&_pragma=busy_timeout%285000%29"
	}

	var dsn string
	if strings.HasPrefix(path, "file:") {
		dsn = path
	} else {
		databaseURL := &url.URL{Scheme: "file"}
		if filepath.IsAbs(path) {
			databaseURL.Path = path
		} else {
			// A relative URL path is rendered as file://./..., which makes
			// SQLite treat "." as an invalid URI authority. Opaque renders
			// the intended file:./... form instead.
			databaseURL.Opaque = path
		}
		dsn = databaseURL.String()
	}
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	return dsn + separator + "_pragma=busy_timeout%285000%29"
}

func (db *DB) configureConnection() error {
	var mode string
	if err := db.conn.QueryRow("PRAGMA journal_mode = WAL").Scan(&mode); err != nil {
		return err
	}
	if !strings.EqualFold(mode, "wal") {
		return fmt.Errorf("WAL journal mode unavailable (database returned %q)", mode)
	}
	return nil
}

// initSchema creates the required tables if they don't exist
func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS request (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		endpoint TEXT NOT NULL,
		method TEXT NOT NULL,
		model TEXT,
		prompt TEXT,
		response TEXT,
		status_code INTEGER,
		latency_ms INTEGER,
		stream BOOLEAN,
		backend_type TEXT,
		error TEXT,
		frontend_url TEXT,
		backend_url TEXT,
		frontend_request TEXT,
		frontend_response TEXT,
		backend_request TEXT,
		backend_response TEXT,
		last_message TEXT NOT NULL DEFAULT 'unknown'
	);

	CREATE INDEX IF NOT EXISTS idx_timestamp ON request(timestamp);
	CREATE INDEX IF NOT EXISTS idx_endpoint ON request(endpoint);
	CREATE INDEX IF NOT EXISTS idx_model ON request(model);
	CREATE INDEX IF NOT EXISTS idx_status_code ON request(status_code);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Log inserts a log entry into the database
func (db *DB) Log(entry LogEntry) error {
	query := `
		INSERT INTO request (timestamp, endpoint, method, model, prompt, response, status_code, latency_ms, stream, backend_type, error, frontend_url, backend_url, frontend_request, frontend_response, backend_request, backend_response, last_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.Exec(
		query,
		entry.Timestamp,
		entry.Endpoint,
		entry.Method,
		entry.Model,
		entry.Prompt,
		entry.Response,
		entry.StatusCode,
		entry.LatencyMs,
		entry.Stream,
		entry.BackendType,
		entry.Error,
		entry.FrontendURL,
		entry.BackendURL,
		entry.FrontendRequest,
		entry.FrontendResponse,
		entry.BackendRequest,
		entry.BackendResponse,
		entry.LastMessage,
	)

	if err != nil {
		return fmt.Errorf("failed to insert log entry: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}
