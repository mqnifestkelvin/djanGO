package commands

import (
	"database/sql"
	"flag"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mqnifestkelvin/djanGO/core/management"
	"github.com/mqnifestkelvin/djanGO/core/migration"
)

func init() {
	management.Register(&migrateCmd{})
}

type migrateCmd struct {
	management.BaseCommand
	fake     bool
	database string
	plan     bool
}

func (c *migrateCmd) Name() string { return "migrate" }
func (c *migrateCmd) Help() string {
	return "Updates database schema by running pending migrations"
}

func (c *migrateCmd) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.fake, "fake", false, "Mark migrations as run without actually running them")
	fs.StringVar(&c.database, "database", "", "Path to SQLite database file (default: db.sqlite3)")
	fs.BoolVar(&c.plan, "plan", false, "Show what migrations would be applied")
}

// Execute mirrors Django's migrate command handle() method.
//
// Usage: go run manage.go migrate [app_label]
//
// Django's migrate:
//  1. Uses MigrationLoader to discover migration files on disk
//  2. Uses MigrationExecutor to build a plan (apply order)
//  3. Applies each migration and records it in django_migrations
//
// djanGO's migrate:
//  1. Migration files register themselves via init() → migration.RegisterMigration()
//     (This is Go's equivalent of Django importing migration modules)
//  2. migration.AllRegistered() returns them in app+number order (the Plan)
//  3. Executor applies pending ones and records them in django_migrations
func (c *migrateCmd) Execute(args []string) error {
	dbPath := c.database
	if dbPath == "" {
		dbPath = "db.sqlite3"
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("migrate: cannot open database '%s': %w", dbPath, err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("migrate: cannot connect to database: %w", err)
	}

	// Get all migrations registered via init() in migration files.
	// Filter by app label if provided.
	targetApp := ""
	if len(args) > 0 {
		targetApp = args[0]
	}

	var migs []*migration.Migration
	if targetApp != "" {
		migs = migration.RegistryFor(targetApp)
	} else {
		migs = migration.AllRegistered()
	}

	if len(migs) == 0 {
		c.Print("No migrations registered.")
		c.Print("Make sure your migrations/ packages are imported in manage.go.")
		return nil
	}

	executor := migration.NewExecutor(db, "sqlite3")
	executor.SetOutput(os.Stdout)

	if c.plan {
		// --plan: show what would be applied, like Django's migrate --plan
		if err := executor.EnsureTable(); err != nil {
			return err
		}
		applied, err := migration.NewRecorder(db, "sqlite3").Applied()
		if err != nil {
			return err
		}
		c.Print("Planned operations:")
		for _, m := range migs {
			if !applied[[2]string{m.App, m.Name}] {
				c.Print("  %s.%s", m.App, m.Name)
				for _, op := range m.Operations {
					c.Print("    %s", op.Description())
				}
			}
		}
		return nil
	}

	return executor.MigratePlan(migs, c.fake)
}
