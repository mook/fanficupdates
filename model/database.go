package model

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Open a new database, possibly read-only.
func Open(dir string, readonly bool) (*sql.DB, error) {
	mode := map[bool]string{false: "rwc", true: "ro"}
	return open(dir, url.Values{"mode": {mode[readonly]}})
}

// OpenMemory opens an in-memory database for testing.
func OpenMemory() (*sql.DB, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get working directory: %w", err)
	}
	return open(cwd, url.Values{"mode": {"memory"}})
}

// Open a database after configuration.
func open(dir string, query url.Values) (*sql.DB, error) {
	query.Add("_pragma", "auto_vacuum('incremental')")
	query.Add("_pragma", "foreign_keys(1)")
	query.Add("_pragma", "journal_mode('wal')")
	u := url.URL{
		Scheme:   "file",
		Path:     filepath.Join(dir, "fanficupdates.sqlite"),
		RawQuery: query.Encode(),
	}
	db, err := sql.Open("sqlite", u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	return db, nil
}

func Insert(ctx context.Context, db *sql.DB, book CalibreBook) error {
	fields := []string{
		"id", "uuid", "publisher", "size", "identifiers", "title", "author",
		"author_sort", "timestamp", "pubdate", "last_modified", "tags",
		"comments", "languages", "cover", "series_index"}
	stmt := fmt.Sprintf("INSERT OR REPLACE INTO BOOKS (%s)",
		strings.Join(fields, ", "))
	db.ExecContext(ctx, stmt)
	return nil
}
