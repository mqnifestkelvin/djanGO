package commands

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mqnifestkelvin/djanGO/client/orm"
	"github.com/mqnifestkelvin/djanGO/conf"
	"github.com/mqnifestkelvin/djanGO/core/management"
	"github.com/mqnifestkelvin/djanGO/core/migration"
)

func init() {
	management.Register(&makeMigrationsCmd{})
}

type makeMigrationsCmd struct {
	management.BaseCommand
	dryRun bool
	name   string
}

func (c *makeMigrationsCmd) Name() string { return "makemigrations" }
func (c *makeMigrationsCmd) Help() string {
	return "Creates new migration(s) for apps"
}

func (c *makeMigrationsCmd) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.dryRun, "dry-run", false, "Show what migrations would be made without writing them")
	fs.StringVar(&c.name, "name", "", "Use this name for the migration file")
}

// Execute mirrors Django's makemigrations command handle() method.
// Usage: go run manage.go makemigrations [app_label ...]
// With no args: detects changes for ALL apps that have registered models.
// With args:    detects changes only for the named apps.
func (c *makeMigrationsCmd) Execute(args []string) error {
	models := orm.GetRegisteredModels()
	if len(models) == 0 {
		c.Print("No models registered. Import your app packages in manage.go.")
		return nil
	}

	// Determine which apps to scan
	targetApps := args
	if len(targetApps) == 0 {
		targetApps = inferAppsFromModels(models)
	}

	if len(targetApps) == 0 {
		c.Print("No changes detected")
		return nil
	}

	// Validate every requested app label against InstalledApps — mirrors Django's:
	//   for app_label in app_labels:
	//       apps.get_app_config(app_label)  # raises LookupError if not installed
	if conf.IsConfigured() {
		for _, app := range targetApps {
			if !conf.IsInstalled(app) {
				fmt.Fprintf(os.Stderr, "\033[1;31mNo installed app with label '%s'.\033[0m\n", app)
				os.Exit(1)
			}
		}
	}

	detector := migration.NewAutodetector()
	madeAny := false

	for _, app := range targetApps {
		migrationsDir := findMigrationsDir(app)
		loader := migration.NewLoader(map[string]string{app: filepath.Dir(migrationsDir)})

		existingModels, err := collectExistingModels(loader, app)
		if err != nil {
			return fmt.Errorf("makemigrations: %w", err)
		}

		ops, err := detector.Changes(app, existingModels)
		if err != nil {
			return fmt.Errorf("makemigrations: %w", err)
		}

		if len(ops) == 0 {
			c.Print(fmt.Sprintf("No changes detected in app '%s'", app))
			continue
		}

		latestNum, err := loader.LatestNumber(app)
		if err != nil {
			return err
		}

		migName := migration.NextMigrationName(latestNum, ops)
		if c.name != "" {
			num := fmt.Sprintf("%04d", latestNum+1)
			migName = num + "_" + c.name
		}

		deps := buildDependencies(app, latestNum)
		m := &migration.Migration{
			Name:         migName,
			App:          app,
			Dependencies: deps,
			Operations:   ops,
			Initial:      latestNum == 0,
		}

		w := &migration.Writer{Migration: m}

		// Django prints: "Migrations for 'blog':" then the file name, then operations
		c.Print("Migrations for '%s':", app)
		c.Print("  %s", filepath.Join(migrationsDir, w.Filename()))
		for _, op := range ops {
			c.Print("    - %s", op.Description())
		}

		if !c.dryRun {
			if _, err := w.Write(migrationsDir); err != nil {
				return err
			}
		}
		madeAny = true
	}

	_ = madeAny
	return nil
}

// findMigrationsDir resolves the migrations/ folder for an app
// relative to the project's working directory: ./<app>/migrations/
func findMigrationsDir(app string) string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, app, "migrations")
}

// inferAppsFromModels derives unique app labels from registered models.
// Beego stores FullName as "import/path.ModelName" e.g. "mysite/blog.Post".
// We extract the last path segment of the import path before the dot.
func inferAppsFromModels(models []orm.ModelInfo) []string {
	seen := make(map[string]bool)
	var apps []string
	for _, mi := range models {
		// FullName = "import/path.ModelName" — split on "." first to get the pkg path
		dotIdx := strings.LastIndex(mi.FullName, ".")
		if dotIdx < 0 {
			continue
		}
		pkgPath := mi.FullName[:dotIdx]                   // e.g. "mysite/blog"
		slashIdx := strings.LastIndex(pkgPath, "/")
		var appLabel string
		if slashIdx >= 0 {
			appLabel = pkgPath[slashIdx+1:] // last path segment e.g. "blog"
		} else {
			appLabel = pkgPath
		}
		appLabel = strings.ToLower(appLabel)
		if !seen[appLabel] && appLabel != "main" {
			seen[appLabel] = true
			apps = append(apps, appLabel)
		}
	}
	return apps
}

// collectExistingModels returns the set of table names already covered by
// registered migration files (loaded via their init() calls).
// Uses the in-memory registry rather than file parsing so Operations are available.
func collectExistingModels(_ *migration.Loader, app string) (map[string]bool, error) {
	tables := make(map[string]bool)
	for _, m := range migration.RegistryFor(app) {
		for _, op := range m.Operations {
			if cm, ok := op.(*migration.CreateModel); ok {
				tables[cm.Table] = true
			}
		}
	}
	return tables, nil
}

// buildDependencies returns the dependency on the previous migration for this app.
func buildDependencies(app string, latestNum int) [][2]string {
	if latestNum == 0 {
		return nil
	}
	// Point to the previous migration by number prefix — e.g. "0001"
	prevNum := fmt.Sprintf("%04d", latestNum)
	return [][2]string{{app, prevNum}}
}
