package management

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStartProjectCreatesExactFiles(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "mysite")

	if err := scaffoldProject(target, "mysite", "test-secret-key"); err != nil {
		t.Fatalf("scaffoldProject failed: %v", err)
	}

	expected := []string{
		"manage.go",
		"go.mod",
		"mysite/init.go",
		"mysite/settings.go",
		"mysite/urls.go",
		"mysite/wsgi.go",
		"mysite/asgi.go",
	}

	for _, rel := range expected {
		full := filepath.Join(target, filepath.FromSlash(rel))
		if _, err := os.Stat(full); os.IsNotExist(err) {
			t.Errorf("expected file missing: %s", rel)
		}
	}
}

func TestStartProjectNoExtraFiles(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "cleansite")

	if err := scaffoldProject(target, "cleansite", "key"); err != nil {
		t.Fatalf("scaffoldProject failed: %v", err)
	}

	// Walk and count files — expect exactly 7
	count := 0
	filepath.Walk(target, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})

	if count != 7 {
		t.Errorf("expected exactly 7 files, got %d", count)
	}
}

func TestStartProjectInnerFolderNamedAfterProject(t *testing.T) {
	dir := t.TempDir()
	projectName := "myapp"
	target := filepath.Join(dir, projectName)

	if err := scaffoldProject(target, projectName, "key"); err != nil {
		t.Fatalf("scaffoldProject failed: %v", err)
	}

	innerDir := filepath.Join(target, projectName)
	info, err := os.Stat(innerDir)
	if os.IsNotExist(err) {
		t.Fatalf("inner config folder '%s' does not exist", projectName)
	}
	if !info.IsDir() {
		t.Fatalf("expected '%s' to be a directory", projectName)
	}
}

func TestStartProjectSecretKeyIsUnique(t *testing.T) {
	key1 := generateSecretKey()
	key2 := generateSecretKey()

	if key1 == key2 {
		t.Error("expected two generateSecretKey() calls to produce different values")
	}
	if !strings.HasPrefix(key1, "djangosecretkey-") {
		t.Errorf("expected key to start with 'djangosecretkey-', got: %s", key1)
	}
}

func TestStartProjectValidateName(t *testing.T) {
	valid := []string{"mysite", "my_site", "_private", "app123"}
	for _, name := range valid {
		if err := validateName(name); err != nil {
			t.Errorf("expected '%s' to be valid, got error: %v", name, err)
		}
	}

	invalid := []string{"", "123abc", "my-site", "my site", "my.site"}
	for _, name := range invalid {
		if err := validateName(name); err == nil {
			t.Errorf("expected '%s' to be invalid, got nil error", name)
		}
	}
}

func TestStartProjectSettingsContainsSecretKey(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "keycheck")
	secretKey := "test-my-secret-key-123"

	if err := scaffoldProject(target, "keycheck", secretKey); err != nil {
		t.Fatalf("scaffoldProject failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(target, "keycheck", "settings.go"))
	if err != nil {
		t.Fatalf("could not read settings.go: %v", err)
	}

	if !strings.Contains(string(data), secretKey) {
		t.Errorf("expected settings.go to contain the secret key '%s'", secretKey)
	}
}
