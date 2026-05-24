package version_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neviim/homeestoque/backend/internal/version"
)

func setup(t *testing.T, running string) {
	t.Helper()
	version.Running = running
	version.ResetCache()
}

func TestRunning_Default(t *testing.T) {
	if version.Running == "" {
		t.Fatal("Running não deve ser vazio")
	}
}

func TestAvailable_ReadsFile(t *testing.T) {
	setup(t, "0.1.0")
	dir := t.TempDir()
	want := "1.2.3"
	if err := os.WriteFile(filepath.Join(dir, "VERSION"), []byte(want+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := version.Available(dir)
	if got != want {
		t.Fatalf("Available = %q, want %q", got, want)
	}
}

func TestAvailable_FallbackWhenMissing(t *testing.T) {
	setup(t, "0.0.1")
	dir := t.TempDir() // sem arquivo VERSION

	got := version.Available(dir)
	if got != "0.0.1" {
		t.Fatalf("Available = %q, want running version %q", got, version.Running)
	}
}

func TestIsUpdateAvailable_True(t *testing.T) {
	setup(t, "0.1.0")
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.1\n"), 0o644)

	if !version.IsUpdateAvailable(dir) {
		t.Fatal("deveria detectar update disponível")
	}
}

func TestIsUpdateAvailable_False(t *testing.T) {
	setup(t, "0.1.0")
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.1.0\n"), 0o644)

	if version.IsUpdateAvailable(dir) {
		t.Fatal("não deveria detectar update quando versões iguais")
	}
}
