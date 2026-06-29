package migration

import (
	"database/sql"
	"fmt"
	"io"
	"os"
)

// Executor applies and unapplies migrations against a live database.
// Equivalent to Django's MigrationExecutor in django/db/migrations/executor.py.
//
// Django's Executor:
//   - builds a Plan (ordered list of migrations to apply)
//   - applies each migration's operations in order
//   - records each migration in django_migrations when done
//
// We do the same, but operations produce raw SQL instead of using
// Django's SchemaEditor. Each migration .go file exports a *Migration var
// that holds its Operations — we call op.SQL(dialect) to get the statements.
type Executor struct {
	db       *sql.DB
	dialect  string
	recorder *Recorder
	out      io.Writer
}

// NewExecutor creates an Executor for the given database connection.
// dialect must be "sqlite3", "mysql", or "postgres".
func NewExecutor(db *sql.DB, dialect string) *Executor {
	return &Executor{
		db:       db,
		dialect:  dialect,
		recorder: NewRecorder(db, dialect),
		out:      os.Stdout,
	}
}

// SetOutput controls where progress messages are written (default: stdout).
func (e *Executor) SetOutput(w io.Writer) {
	e.out = w
}

// EnsureTable creates the django_migrations table if it doesn't exist.
func (e *Executor) EnsureTable() error {
	return e.recorder.EnsureTable()
}

// MigratePlan applies the given list of migrations in order, skipping those
// already recorded in django_migrations.
// Equivalent to Django's MigrationExecutor.migrate() / _migrate_all_forwards().
func (e *Executor) MigratePlan(migrations []*Migration, fake bool) error {
	if err := e.recorder.EnsureTable(); err != nil {
		return fmt.Errorf("migrate: could not create django_migrations table: %w", err)
	}

	applied, err := e.recorder.Applied()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		key := [2]string{m.App, m.Name}
		if applied[key] {
			continue
		}

		fmt.Fprintf(e.out, "  Applying %s.%s...", m.App, m.Name)

		if !fake {
			if err := e.applyMigration(m); err != nil {
				fmt.Fprintln(e.out, " FAILED")
				return fmt.Errorf("migrate: error applying %s.%s: %w", m.App, m.Name, err)
			}
		}

		if err := e.recorder.Record(m.App, m.Name); err != nil {
			return fmt.Errorf("migrate: could not record %s.%s: %w", m.App, m.Name, err)
		}
		fmt.Fprintln(e.out, " OK")
	}

	return nil
}

// applyMigration executes all forward SQL for a migration in a transaction.
func (e *Executor) applyMigration(m *Migration) error {
	tx, err := e.db.Begin()
	if err != nil {
		return err
	}

	for _, op := range m.Operations {
		for _, stmt := range op.SQL(e.dialect) {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("SQL error in operation '%s': %w\nSQL: %s", op.Description(), err, stmt)
			}
		}
	}

	return tx.Commit()
}

// UnapplyPlan rolls back the given migrations in reverse order.
// Equivalent to Django's MigrationExecutor._migrate_all_backwards().
func (e *Executor) UnapplyPlan(migrations []*Migration, fake bool) error {
	applied, err := e.recorder.Applied()
	if err != nil {
		return err
	}

	// Reverse order for rollback
	for i := len(migrations) - 1; i >= 0; i-- {
		m := migrations[i]
		key := [2]string{m.App, m.Name}
		if !applied[key] {
			continue
		}

		fmt.Fprintf(e.out, "  Unapplying %s.%s...", m.App, m.Name)

		if !fake {
			if err := e.unapplyMigration(m); err != nil {
				fmt.Fprintln(e.out, " FAILED")
				return fmt.Errorf("migrate: error unapplying %s.%s: %w", m.App, m.Name, err)
			}
		}

		if err := e.recorder.Unrecord(m.App, m.Name); err != nil {
			return fmt.Errorf("migrate: could not unrecord %s.%s: %w", m.App, m.Name, err)
		}
		fmt.Fprintln(e.out, " OK")
	}

	return nil
}

func (e *Executor) unapplyMigration(m *Migration) error {
	tx, err := e.db.Begin()
	if err != nil {
		return err
	}

	// Reverse operation order, like Django's unapply()
	for i := len(m.Operations) - 1; i >= 0; i-- {
		for _, stmt := range m.Operations[i].RollbackSQL(e.dialect) {
			if _, err := tx.Exec(stmt); err != nil {
				tx.Rollback()
				return fmt.Errorf("SQL error rolling back '%s': %w\nSQL: %s",
					m.Operations[i].Description(), err, stmt)
			}
		}
	}

	return tx.Commit()
}

// ShowMigrations prints the apply status of all migrations for the given apps,
// like Django's `migrate --list` or `showmigrations`.
func (e *Executor) ShowMigrations(loader *Loader, apps []string) error {
	if err := e.recorder.EnsureTable(); err != nil {
		return err
	}

	applied, err := e.recorder.Applied()
	if err != nil {
		return err
	}

	for _, app := range apps {
		migs, err := loader.Migrations(app)
		if err != nil {
			return err
		}
		fmt.Fprintf(e.out, "%s\n", app)
		if len(migs) == 0 {
			fmt.Fprintf(e.out, " (no migrations)\n")
			continue
		}
		for _, m := range migs {
			mark := "[ ]"
			if applied[[2]string{m.App, m.Name}] {
				mark = "[x]"
			}
			fmt.Fprintf(e.out, " %s %s\n", mark, m.Name)
		}
	}
	return nil
}
