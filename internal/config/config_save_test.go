package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Save previously accepted a full file path and silently discarded the
// basename, writing to filepath.Join(filepath.Dir(path), ConfigFileName)
// regardless of the filename passed. The signature now names what it actually
// takes — a directory (#874).

func TestSave_WritesConfigFileIntoGivenDirectory(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Version: "1.4.10"}
	cfg.Project.Name = "gh-pmu"

	if err := cfg.Save(dir); err != nil {
		t.Fatalf("Expected Save to succeed; got %v", err)
	}

	written := filepath.Join(dir, ConfigFileName)
	if _, err := os.Stat(written); err != nil {
		t.Fatalf("Expected %s to exist; got %v", written, err)
	}

	loaded, err := Load(written)
	if err != nil {
		t.Fatalf("Expected the written config to load; got %v", err)
	}
	if loaded.Project.Name != "gh-pmu" {
		t.Errorf("Expected round-tripped project name gh-pmu; got %q", loaded.Project.Name)
	}
}

// The path-to-directory change is not compile-enforced (both are strings), so a
// stale caller passing the config file path must fail loudly rather than write
// to <config>/.gh-pmu.json.
func TestSave_RejectsAConfigFilePath(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{Version: "1.4.10"}

	err := cfg.Save(filepath.Join(dir, ConfigFileName))
	if err == nil {
		t.Errorf("Expected Save to reject a %s file path", ConfigFileName)
	}
}

// A nested directory that does not exist must surface an error rather than
// writing somewhere unexpected.
func TestSave_MissingDirectoryReturnsError(t *testing.T) {
	cfg := &Config{Version: "1.4.10"}

	err := cfg.Save(filepath.Join(t.TempDir(), "does", "not", "exist"))
	if err == nil {
		t.Fatal("Expected an error when the target directory is absent")
	}
}
