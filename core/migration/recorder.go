package migration

import (
	"database/sql"
	"fmt"
	"time"
)

const createTableSQL = `CREATE TABLE IF NOT EXISTS django_migrations (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    app        VARCHAR(255) NOT NULL,
    name       VARCHAR(255) NOT NULL,
    applied    DATETIME NOT NULL
)`

const createTableSQLMySQL = `CREATE TABLE IF NOT EXISTS ` + "`django_migrations`" + ` (
    id      INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    app     VARCHAR(255) NOT NULL,
    name    VARCHAR(255) NOT NULL,
    applied DATETIME NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`

const createTableSQLPostgres = `CREATE TABLE IF NOT EXISTS "django_migrations" (
    id      SERIAL PRIMARY KEY,
    app     VARCHAR(255) NOT NULL,
    name    VARCHAR(255) NOT NULL,
    applied TIMESTAMP NOT NULL
)`

// Recorder manages the django_migrations table — mirrors Django's MigrationRecorder.
type Recorder struct {
	db      *sql.DB
	dialect string
}

// NewRecorder creates a Recorder for the given DB connection.
// dialect must be "sqlite3", "mysql", or "postgres".
func NewRecorder(db *sql.DB, dialect string) *Recorder {
	return &Recorder{db: db, dialect: dialect}
}

// EnsureTable creates the django_migrations table if it doesn't exist.
func (r *Recorder) EnsureTable() error {
	var q string
	switch r.dialect {
	case "mysql":
		q = createTableSQLMySQL
	case "postgres":
		q = createTableSQLPostgres
	default:
		q = createTableSQL
	}
	_, err := r.db.Exec(q)
	return err
}

// Applied returns the set of (app, name) pairs already applied to this DB.
func (r *Recorder) Applied() (map[[2]string]bool, error) {
	rows, err := r.db.Query("SELECT app, name FROM django_migrations")
	if err != nil {
		return nil, fmt.Errorf("migration: could not query django_migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[[2]string]bool)
	for rows.Next() {
		var app, name string
		if err := rows.Scan(&app, &name); err != nil {
			return nil, err
		}
		applied[[2]string{app, name}] = true
	}
	return applied, rows.Err()
}

// Record marks a migration as applied.
func (r *Recorder) Record(app, name string) error {
	_, err := r.db.Exec(
		"INSERT INTO django_migrations (app, name, applied) VALUES (?, ?, ?)",
		app, name, time.Now().Format("2006-01-02 15:04:05"),
	)
	return err
}

// Unrecord removes a migration from the applied set (used by migrate --fake rollback).
func (r *Recorder) Unrecord(app, name string) error {
	_, err := r.db.Exec(
		"DELETE FROM django_migrations WHERE app = ? AND name = ?",
		app, name,
	)
	return err
}
