// Package migration implements djanGO's migration system — a 1:1 port of
// Django's django.db.migrations architecture.
//
// The flow mirrors Django exactly:
//
//	makemigrations  →  introspects orm.RegisterModel() structs
//	                   compares against applied migrations in django_migrations table
//	                   writes numbered .go files to <app>/migrations/
//
//	migrate         →  discovers all migration files via the Loader
//	                   builds a dependency-ordered Plan via the Executor
//	                   applies pending migrations and records them in django_migrations
//
// Django equivalents:
//
//	Migration struct         ←→  django.db.migrations.Migration
//	Loader                   ←→  django.db.migrations.loader.MigrationLoader
//	Executor                 ←→  django.db.migrations.executor.MigrationExecutor
//	Autodetector             ←→  django.db.migrations.autodetector.MigrationAutodetector
//	Writer                   ←→  django.db.migrations.writer.MigrationWriter
//	django_migrations table  ←→  django_migrations table
package migration

import "fmt"

// Migration is the in-memory representation of one migration file.
// Equivalent to Django's Migration class in django/db/migrations/migration.py.
type Migration struct {
	// Name is the filename stem e.g. "0001_initial" or "0002_add_field_post_title"
	Name string

	// App is the app label this migration belongs to e.g. "blog"
	App string

	// Dependencies lists migrations that must be applied before this one.
	// Each entry is [app, name] — same shape as Django's dependencies list.
	Dependencies [][2]string

	// Operations is the ordered list of schema changes.
	Operations []Operation

	// Initial is true for the first migration of an app (mirrors Django's initial = True).
	Initial bool
}

func (m *Migration) String() string {
	return fmt.Sprintf("%s.%s", m.App, m.Name)
}

// MigrationRecord is one row from the django_migrations table.
// Equivalent to Django's MigrationRecorder.Migration model.
type MigrationRecord struct {
	ID      int64
	App     string
	Name    string
	Applied string // timestamp string
}
