package migration

import (
	"sort"
	"sync"
)

// registry holds all Migration objects registered via init() in migration files.
// This is the Go equivalent of Django's MigrationLoader.disk_migrations —
// Django imports Python modules; we use Go's init() mechanism instead.
var (
	registryMu sync.RWMutex
	registered []*Migration
	byKey      = make(map[[2]string]*Migration) // keyed by [app, name]
)

// RegisterMigration adds a Migration to the global in-memory registry.
// Call this from the init() of each generated migration file, or declare
// it as the var initializer — the generated files call this automatically.
//
// This mirrors what Django does when it imports a migration module:
// the Migration class is read and stored in MigrationLoader.disk_migrations.
func RegisterMigration(m *Migration) {
	registryMu.Lock()
	defer registryMu.Unlock()
	key := [2]string{m.App, m.Name}
	if _, exists := byKey[key]; !exists {
		registered = append(registered, m)
		byKey[key] = m
	}
}

// RegistryFor returns all registered migrations for a given app, sorted by number.
func RegistryFor(app string) []*Migration {
	registryMu.RLock()
	defer registryMu.RUnlock()

	var out []*Migration
	for _, m := range registered {
		if m.App == app {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return migrationNumber(out[i].Name) < migrationNumber(out[j].Name)
	})
	return out
}

// AllRegistered returns all migrations across all apps, sorted app then number.
func AllRegistered() []*Migration {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make([]*Migration, len(registered))
	copy(out, registered)
	sort.Slice(out, func(i, j int) bool {
		if out[i].App != out[j].App {
			return out[i].App < out[j].App
		}
		return migrationNumber(out[i].Name) < migrationNumber(out[j].Name)
	})
	return out
}
