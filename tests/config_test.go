package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zaidejjo/zgit/pkg/core/config"
)

func TestConfigMigratesYAMLToTOML(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".config", "zgit")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	yaml := "github:\n  user: octocat\ntheme: catppuccin\n"
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	m, err := config.New()
	if err != nil {
		t.Fatal(err)
	}
	if got := m.GetString("github.user"); got != "octocat" {
		t.Errorf("github.user = %q, want octocat", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.toml")); err != nil {
		t.Errorf("config.toml missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); !os.IsNotExist(err) {
		t.Errorf("config.yaml should be renamed away")
	}
	if _, err := os.Stat(filepath.Join(dir, "config.yaml.bak")); err != nil {
		t.Errorf("config.yaml.bak missing: %v", err)
	}

	// Second boot reads TOML directly and keeps values.
	m2, err := config.New()
	if err != nil {
		t.Fatal(err)
	}
	if got := m2.GetString("theme"); got != "catppuccin" {
		t.Errorf("theme = %q, want catppuccin", got)
	}
}
