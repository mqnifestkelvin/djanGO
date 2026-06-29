package management

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartAppCreatesExactFiles(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "blog")

	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := scaffoldApp(appDir, "blog"); err != nil {
		t.Fatalf("scaffoldApp failed: %v", err)
	}

	expected := []string{
		"init.go",
		"apps.go",
		"admin.go",
		"models.go",
		"views.go",
		"tests.go",
		"migrations/init.go",
	}

	for _, rel := range expected {
		full := filepath.Join(appDir, filepath.FromSlash(rel))
		if _, err := os.Stat(full); os.IsNotExist(err) {
			t.Errorf("expected file missing: %s", rel)
		}
	}
}

func TestStartAppNoExtraFiles(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "accounts")
	os.MkdirAll(appDir, 0755)

	if err := scaffoldApp(appDir, "accounts"); err != nil {
		t.Fatalf("scaffoldApp failed: %v", err)
	}

	count := 0
	filepath.Walk(appDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})

	if count != 7 {
		t.Errorf("expected exactly 7 files, got %d", count)
	}
}

func TestStartAppMigrationsFolderExists(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "posts")
	os.MkdirAll(appDir, 0755)

	if err := scaffoldApp(appDir, "posts"); err != nil {
		t.Fatalf("scaffoldApp failed: %v", err)
	}

	migrationsInit := filepath.Join(appDir, "migrations", "init.go")
	if _, err := os.Stat(migrationsInit); os.IsNotExist(err) {
		t.Error("expected migrations/init.go to exist")
	}
}

func TestStartAppValidateName(t *testing.T) {
	valid := []string{"blog", "my_app", "_private", "app123"}
	for _, name := range valid {
		if err := validateName(name); err != nil {
			t.Errorf("expected '%s' to be valid, got error: %v", name, err)
		}
	}

	invalid := []string{"", "123blog", "my-app", "my app"}
	for _, name := range invalid {
		if err := validateName(name); err == nil {
			t.Errorf("expected '%s' to be invalid, got nil error", name)
		}
	}
}
