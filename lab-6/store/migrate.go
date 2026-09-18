package store

import (
	"database/sql"
	"fmt"
)

// schemaVersion is the newest schema version this code knows how to build.
// It equals len(migrations).
var schemaVersion = len(migrations)

// migrations holds one SQL migration per schema version, in order. Index 0
// brings the schema to version 1, index 1 to version 2, and so on. Never
// edit an entry once it has been released: append new migrations instead.
var migrations = []string{
	// v1: initial notes table.
	`CREATE TABLE notes (
		id    INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT    NOT NULL,
		body  TEXT    NOT NULL DEFAULT ''
	);`,
}

// migrate brings an existing database up to schemaVersion. Each migration
// runs in its own transaction, and PRAGMA user_version records how far the
// schema has advanced so already-applied migrations are skipped on reopen.
func migrate(db *sql.DB) error {
	version, err := schemaVersion(db)
	if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for v := version + 1; v <= schemaVersion; v++ {
		if err := applyMigration(db, v); err != nil {
			return fmt.Errorf("migrate to version %d: %w", v, err)
		}
	}

	return nil
}

// schemaVersionDB reads the current schema version from PRAGMA user_version.
func schemaVersion(db *sql.DB) (int, error) {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return 0, err
	}
	return version, nil
}

// applyMigration runs migration v inside a transaction and bumps
// user_version on success, so a crash mid-migration is rolled back cleanly.
func applyMigration(db *sql.DB, v int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(migrations[v-1]); err != nil {
		return err
	}

	// PRAGMA does not accept bound parameters, so the version is formatted
	// in. v comes from our own migrations slice, never from user input.
	if _, err := tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", v)); err != nil {
		return err
	}

	return tx.Commit()
}