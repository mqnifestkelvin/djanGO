package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Loader discovers migration files on disk and parses them into Migration objects.
// Equivalent to Django's MigrationLoader in django/db/migrations/loader.py.
//
// Unlike Django (which imports Python modules), we parse the generated .go files
// directly — reading the metadata comment block at the top of each file.
// The generated format is:
//
//	// djanGO migration
//	// App: blog
//	// Name: 0001_initial
//	// Dependencies: []
//	// Operations: CreateModel:Post, CreateModel:Comment
type Loader struct {
	// MigrationsDir maps app label → path to its migrations/ directory.
	MigrationsDir map[string]string
}

// NewLoader creates a Loader. appDirs maps app label → app root directory
// (the directory that contains a migrations/ sub-folder).
func NewLoader(appDirs map[string]string) *Loader {
	dirs := make(map[string]string, len(appDirs))
	for app, dir := range appDirs {
		dirs[app] = filepath.Join(dir, "migrations")
	}
	return &Loader{MigrationsDir: dirs}
}

// Migrations returns all discovered migrations for an app, sorted by number.
func (l *Loader) Migrations(app string) ([]*Migration, error) {
	dir, ok := l.MigrationsDir[app]
	if !ok {
		return nil, fmt.Errorf("migration: no migrations directory registered for app '%s'", app)
	}

	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var migrations []*Migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		stem := strings.TrimSuffix(e.Name(), ".go")
		if stem == "init" {
			continue
		}
		m, err := parseMigrationFile(filepath.Join(dir, e.Name()), app, stem)
		if err != nil {
			return nil, fmt.Errorf("migration: could not parse %s: %w", e.Name(), err)
		}
		migrations = append(migrations, m)
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrationNumber(migrations[i].Name) < migrationNumber(migrations[j].Name)
	})
	return migrations, nil
}

// LatestNumber returns the highest migration number for an app (0 if none exist).
func (l *Loader) LatestNumber(app string) (int, error) {
	migs, err := l.Migrations(app)
	if err != nil {
		return 0, err
	}
	if len(migs) == 0 {
		return 0, nil
	}
	return migrationNumber(migs[len(migs)-1].Name), nil
}

// migrationNumber extracts the leading 4-digit number from a migration name like "0003_add_post".
func migrationNumber(name string) int {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) == 0 {
		return 0
	}
	n, _ := strconv.Atoi(parts[0])
	return n
}

// parseMigrationFile reads a generated migration .go file and reconstructs
// a Migration from its metadata header comment block.
func parseMigrationFile(path, app, name string) (*Migration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	m := &Migration{
		App:  app,
		Name: name,
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "// ") {
			continue
		}
		line = strings.TrimPrefix(line, "// ")

		switch {
		case strings.HasPrefix(line, "Initial: true"):
			m.Initial = true
		case strings.HasPrefix(line, "Dependencies: "):
			raw := strings.TrimPrefix(line, "Dependencies: ")
			raw = strings.Trim(raw, "[]")
			if raw != "" {
				for _, dep := range strings.Split(raw, ", ") {
					parts := strings.SplitN(dep, ":", 2)
					if len(parts) == 2 {
						m.Dependencies = append(m.Dependencies, [2]string{parts[0], parts[1]})
					}
				}
			}
		}
	}
	return m, nil
}
