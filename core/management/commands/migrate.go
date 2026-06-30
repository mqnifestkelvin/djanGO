package commands

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/mqnifestkelvin/djanGO/client/orm"
	"github.com/mqnifestkelvin/djanGO/core/management"
	"github.com/mqnifestkelvin/djanGO/core/migration"
	"github.com/mqnifestkelvin/djanGO/core/signals"
)

// ANSI colours — match Django's migrate output exactly.
const (
	colorReset     = "\033[0m"
	colorBoldCyan  = "\033[1;36m"
	colorBoldGreen = "\033[1;32m"
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
// Django output:
//
//	Operations to perform:
//	  Apply all migrations: auth, blog, contenttypes, sessions
//	Running migrations:
//	  Applying contenttypes.0001_initial... OK
//	  Applying auth.0001_initial... OK
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

	executor := migration.NewExecutor(db, "sqlite3")
	executor.SetOutput(os.Stdout)

	if c.plan {
		if err := executor.EnsureTable(); err != nil {
			return err
		}
		applied, err := migration.NewRecorder(db, "sqlite3").Applied()
		if err != nil {
			return err
		}
		fmt.Println("Planned operations:")
		for _, m := range migs {
			if !applied[[2]string{m.App, m.Name}] {
				fmt.Printf("  %s.%s\n", m.App, m.Name)
				for _, op := range m.Operations {
					fmt.Printf("    %s\n", op.Description())
				}
			}
		}
		return nil
	}

	// Print "Operations to perform:" header — mirrors Django's migrate output.
	//
	// Django:
	//   Operations to perform:
	//     Apply all migrations: auth, blog, contenttypes, sessions
	appSet := map[string]struct{}{}
	for _, m := range migs {
		appSet[m.App] = struct{}{}
	}
	// Include contrib apps that are synced via RunSyncdb.
	for _, app := range []string{"admin", "auth", "contenttypes", "sessions"} {
		appSet[app] = struct{}{}
	}
	appList := make([]string, 0, len(appSet))
	for app := range appSet {
		appList = append(appList, app)
	}
	sort.Strings(appList)

	if targetApp != "" {
		fmt.Printf("Operations to perform:\n")
		fmt.Printf("  Apply all migrations: %s\n", targetApp)
	} else if len(appList) > 0 {
		fmt.Printf("Operations to perform:\n")
		apps := ""
		for i, a := range appList {
			if i > 0 {
				apps += ", "
			}
			apps += a
		}
		fmt.Printf("  Apply all migrations: %s\n", apps)
	}

	// Count pending file-based migrations before running — needed to decide
	// whether to print the single "Running migrations:" header and the
	// "No migrations to apply." message.
	pendingMigs, err := executor.PendingMigrations(migs)
	if err != nil {
		return err
	}

	// Count pending contrib tables (those not yet in sqlite_master).
	pendingContrib, err := c.pendingContribTables(db)
	if err != nil {
		return err
	}

	totalPending := len(pendingMigs) + len(pendingContrib)

	if totalPending == 0 {
		fmt.Println("  No migrations to apply.")
		return nil
	}

	// Print single "Running migrations:" header — mirrors Django (one header
	// for the entire migrate run, not one per batch).
	fmt.Fprintf(os.Stdout, "%sRunning migrations:%s\n", colorBoldCyan, colorReset)

	// Apply file-based migrations (app migration files) — header already printed.
	if err := executor.MigratePlan(migs, c.fake, false); err != nil {
		return err
	}

	// Apply contrib model tables via RunSyncdb — djanGO equivalent of Django's
	// built-in migrations for auth, contenttypes, sessions.
	if err := c.syncContribTables(db, pendingContrib); err != nil {
		return err
	}

	// post_migrate — mirrors Django's post_migrate signal.
	// Django fires this after every migrate run to populate django_content_type
	// and auth_permission rows for every installed model.
	if err := createContentTypesAndPermissions(db); err != nil {
		return err
	}

	// Fire post_migrate signal — allows app code to hook into the migrate lifecycle.
	// Django: post_migrate.send(sender=app_config, verbosity=options["verbosity"])
	signals.PostMigrate.Send("migrate", signals.Kwargs{"verbosity": 1})

	return nil
}

type contribMig struct {
	app   string
	name  string
	table string
}

// allContribMigrations is the ordered list of contrib tables djanGO creates via RunSyncdb,
// presented as Django-style named migrations.
// Order mirrors Django's dependency graph:
//   contenttypes → auth → admin → sessions
var allContribMigrations = []contribMig{
	{"contenttypes", "0001_initial", "django_content_type"},
	{"auth", "0001_initial", "auth_permission"},
	{"auth", "0002_initial", "auth_group"},
	{"auth", "0003_initial", "auth_group_permissions"},
	{"auth", "0004_initial", "auth_user"},
	{"auth", "0005_initial", "auth_user_groups"},
	{"auth", "0006_initial", "auth_user_user_permissions"},
	{"admin", "0001_initial", "django_admin_log"},
	{"sessions", "0001_initial", "django_session"},
}

// pendingContribTables returns the subset of contrib migrations not yet recorded
// in django_migrations — mirrors how PendingMigrations works for file-based migrations.
func (c *migrateCmd) pendingContribTables(db *sql.DB) ([]contribMig, error) {
	rec := migration.NewRecorder(db, "sqlite3")
	if err := rec.EnsureTable(); err != nil {
		return nil, err
	}
	applied, err := rec.Applied()
	if err != nil {
		return nil, err
	}
	var pending []contribMig
	for _, m := range allContribMigrations {
		if !applied[[2]string{m.app, m.name}] {
			pending = append(pending, m)
		}
	}
	return pending, nil
}

// syncContribTables runs orm.RunSyncdb for the given pending contrib tables,
// records each one in django_migrations, and prints each as "Applying app.name... OK".
// The "Running migrations:" header must already have been printed by the caller.
func (c *migrateCmd) syncContribTables(db *sql.DB, pending []contribMig) error {
	if len(pending) == 0 {
		return nil
	}
	if err := orm.RunSyncdb("default", false, false); err != nil {
		return fmt.Errorf("migrate: syncdb failed: %w", err)
	}
	rec := migration.NewRecorder(db, "sqlite3")
	for _, m := range pending {
		fmt.Fprintf(os.Stdout, "  Applying %s.%s... %sOK%s\n",
			m.app, m.name, colorBoldGreen, colorReset)
		if err := rec.Record(m.app, m.name); err != nil {
			return fmt.Errorf("migrate: could not record %s.%s: %w", m.app, m.name, err)
		}
	}
	return nil
}

// verboseNames maps model names to their Django verbose_name equivalents.
// Django derives these from the model's Meta.verbose_name — we hardcode the
// known contrib ones to match Django's output exactly.
var verboseNames = map[string]string{
	"contenttype": "content type",
	"logentry":    "log entry",
	"permission":  "permission",
	"group":       "group",
	"user":        "user",
	"session":     "session",
	"post":        "post",
}

// modelVerboseName returns the human-readable name for a model —
// mirrors Django's opts.verbose_name used in permission name strings.
func modelVerboseName(modelName string) string {
	if v, ok := verboseNames[modelName]; ok {
		return v
	}
	// Default: use model name as-is (user-defined models like "post")
	return modelName
}

// createContentTypesAndPermissions mirrors Django's post_migrate signal handlers:
//   - create_contenttypes() in django.contrib.contenttypes.management
//   - create_permissions()  in django.contrib.auth.management
//
// After every migrate Django inserts one django_content_type row per registered
// model and four auth_permission rows (add/change/delete/view) per content type.
// We replicate that here using raw SQL so it works even before the ORM is fully
// bootstrapped in the command context.
func createContentTypesAndPermissions(db *sql.DB) error {
	models := orm.GetRegisteredModels()

	for _, m := range models {
		// FullName is "pkgPath.StructName" e.g.
		//   "github.com/mqnifestkelvin/djanGO/contrib/auth.User"
		//   "mysite/blog.Post"
		// app_label = last path segment before the dot  ("auth", "blog")
		// model_name = struct name lowercased            ("user", "post")
		dotIdx := strings.LastIndex(m.FullName, ".")
		if dotIdx < 0 {
			continue
		}
		pkgPath := m.FullName[:dotIdx]
		modelName := strings.ToLower(m.FullName[dotIdx+1:])

		// Last segment of the package path is the app label.
		// e.g. ".../contrib/auth" → "auth", "mysite/blog" → "blog"
		slashIdx := strings.LastIndex(pkgPath, "/")
		appLabel := pkgPath
		if slashIdx >= 0 {
			appLabel = pkgPath[slashIdx+1:]
		}

		if appLabel == "" || modelName == "" {
			continue
		}

		// Skip internal M2M join tables — they have no semantic meaning as
		// content types (Django doesn't create content types for join tables).
		// We identify them by checking if the model name ends with known suffixes.
		internalJoin := map[string]bool{
			"grouppermission": true,
			"usergroup":       true,
			"userpermission":  true,
		}
		if internalJoin[modelName] {
			continue
		}

		// 1. Upsert django_content_type — get or create.
		var ctID int
		err := db.QueryRow(
			`SELECT id FROM django_content_type WHERE app_label=? AND model=?`,
			appLabel, modelName,
		).Scan(&ctID)
		if err == sql.ErrNoRows {
			res, insertErr := db.Exec(
				`INSERT INTO django_content_type (app_label, model) VALUES (?, ?)`,
				appLabel, modelName,
			)
			if insertErr != nil {
				return fmt.Errorf("post_migrate: insert content type %s.%s: %w", appLabel, modelName, insertErr)
			}
			id, _ := res.LastInsertId()
			ctID = int(id)
		} else if err != nil {
			return fmt.Errorf("post_migrate: query content type %s.%s: %w", appLabel, modelName, err)
		}

		// 2. Create the four default permissions (add/change/delete/view) if missing.
		// Mirrors Django's _get_builtin_permissions() / create_permissions().
		// verbose_name = model name with spaces before capitals, e.g. "contenttype" → "content type"
		verboseName := modelVerboseName(modelName)
		for _, action := range []string{"add", "change", "delete", "view"} {
			codename := action + "_" + modelName
			name := "Can " + action + " " + verboseName

			var exists int
			_ = db.QueryRow(
				`SELECT 1 FROM auth_permission WHERE content_type_id=? AND codename=?`,
				ctID, codename,
			).Scan(&exists)
			if exists == 1 {
				continue
			}
			if _, err := db.Exec(
				`INSERT INTO auth_permission (name, content_type_id, codename) VALUES (?, ?, ?)`,
				name, ctID, codename,
			); err != nil {
				return fmt.Errorf("post_migrate: insert permission %s: %w", codename, err)
			}
		}
	}
	return nil
}
