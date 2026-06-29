package management

import (
	"fmt"
	"os"
	"path/filepath"
)

// fmt is used for error messages only

func RunStartApp(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Error: you must provide an app name.")
		fmt.Fprintln(os.Stderr, "Usage: djangocli startapp <name> [directory]")
		os.Exit(1)
	}

	name := args[0]
	var target string
	if len(args) >= 2 {
		target = args[1]
	}

	if err := validateName(name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	topDir := resolveTargetDir(name, target)

	if _, err := os.Stat(topDir); err == nil {
		fmt.Fprintf(os.Stderr, "Error: '%s' already exists.\n", topDir)
		os.Exit(1)
	}

	if err := os.MkdirAll(topDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	if err := scaffoldApp(topDir, name); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

}

func scaffoldApp(topDir, name string) error {
	// Django generates exactly 7 files:
	// __init__.py, admin.py, apps.py, models.py, tests.py, views.py
	// migrations/__init__.py
	//
	// We mirror this exactly — nothing extra.
	camel := toCamelCase(name)

	files := map[string]func() string{
		// mirrors __init__.py
		"init.go": func() string {
			return fmt.Sprintf("package %s\n", name)
		},

		// mirrors apps.py
		"apps.go": func() string {
			return fmt.Sprintf(`package %s

import "github.com/mqnifestkelvin/djanGO/core/apps"

type %sConfig struct {
	apps.AppConfig
}

func init() {
	apps.MustRegister(&apps.AppConfig{
		Name:        "%s",
		Label:       "%s",
		VerboseName: "%s",
	})
}
`, name, camel, name, name, camel)
		},

		// mirrors admin.py
		"admin.go": func() string {
			return fmt.Sprintf(`package %s

// Register your models with the djanGO admin here.
`, name)
		},

		// mirrors models.py
		"models.go": func() string {
			return fmt.Sprintf(`package %s

import "github.com/mqnifestkelvin/djanGO/client/orm"

// Create your models here.
var _ = orm.NewOrm
`, name)
		},

		// mirrors views.py
		"views.go": func() string {
			return fmt.Sprintf(`package %s

import "github.com/mqnifestkelvin/djanGO/server/web"

// Create your views here.
type IndexController struct {
	web.Controller
}
`, name)
		},

		// mirrors tests.py
		"tests.go": func() string {
			return fmt.Sprintf(`package %s

import "testing"

// Create your tests here.
var _ = testing.T{}
`, name)
		},

		// mirrors migrations/__init__.py
		"migrations/init.go": func() string {
			return "package migrations\n"
		},
	}

	for relPath, contentFn := range files {
		fullPath := filepath.Join(topDir, filepath.FromSlash(relPath))

		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("could not create directory for %s: %w", relPath, err)
		}

		if err := os.WriteFile(fullPath, []byte(contentFn()), 0644); err != nil {
			return fmt.Errorf("could not write %s: %w", relPath, err)
		}
	}

	return nil
}
