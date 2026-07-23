package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rubrical-works/gh-pmu/internal/integrity"
)

func TestRunIntegrityCheck_NoDrift_NoOutput(t *testing.T) {
	// ARRANGE: Config matches HEAD
	dir := t.TempDir()
	configContent := []byte(`{"project":{"owner":"test","number":1},"repositories":["test/repo"]}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	buf := new(bytes.Buffer)

	// ACT
	err := runIntegrityCheck(configPath, buf)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	output := buf.String()
	if output != "" {
		t.Errorf("Expected no output for clean config, got: %q", output)
	}
}

func TestRunIntegrityCheck_WithDrift_WarnsUser(t *testing.T) {
	// ARRANGE: Config drifted from HEAD
	dir := t.TempDir()
	original := []byte(`{"project":{"owner":"original","number":1},"repositories":["test/repo"]}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	modified := []byte(`{"project":{"owner":"changed","number":1},"repositories":["test/repo"]}`)
	if err := os.WriteFile(configPath, modified, 0644); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)

	// ACT
	err := runIntegrityCheck(configPath, buf)

	// ASSERT: Warns but doesn't error (default non-strict)
	if err != nil {
		t.Fatalf("Expected no error (warn only), got: %v", err)
	}
	output := buf.String()
	if !containsStr(output, "Warning") {
		t.Errorf("Expected warning in output, got: %q", output)
	}
}

func TestRunIntegrityCheck_StrictMode_ReturnsError(t *testing.T) {
	// ARRANGE: Config with strict mode, drifted
	dir := t.TempDir()
	original := []byte(`{"project":{"owner":"original","number":1},"repositories":["test/repo"],"configIntegrity":"strict"}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	modified := []byte(`{"project":{"owner":"changed","number":1},"repositories":["test/repo"],"configIntegrity":"strict"}`)
	if err := os.WriteFile(configPath, modified, 0644); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)

	// ACT
	err := runIntegrityCheck(configPath, buf)

	// ASSERT: Strict mode returns error
	if err == nil {
		t.Fatal("Expected error in strict mode with drift")
	}
}

// #865 Gap 1: strict mode is decided from HEAD-or-local.
func TestIsStrictModeEither(t *testing.T) {
	strict := []byte(`{"configIntegrity":"strict"}`)
	notStrict := []byte(`{"project":{"owner":"x"}}`)
	cases := []struct {
		name        string
		local, head []byte
		want        bool
	}{
		{"local strict, head not", strict, notStrict, true},
		{"local not, head strict", notStrict, strict, true},
		{"both strict", strict, strict, true},
		{"neither strict", notStrict, notStrict, false},
		{"head nil, local strict", strict, nil, true},
		{"head nil, local not", notStrict, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isStrictModeEither(tc.local, tc.head); got != tc.want {
				t.Errorf("isStrictModeEither = %v, want %v", got, tc.want)
			}
		})
	}
}

// #865 Gap 1 (AC1/AC4): removing the strict key locally must not disable
// enforcement while HEAD still declares strict.
func TestRunIntegrityCheck_TamperedStrictKeyStillEnforced(t *testing.T) {
	dir := t.TempDir()
	committed := []byte(`{"project":{"owner":"original","number":1},"repositories":["test/repo"],"configIntegrity":"strict"}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, committed, 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	// Tamper: drop the strict key AND drift a value.
	tampered := []byte(`{"project":{"owner":"changed","number":1},"repositories":["test/repo"]}`)
	if err := os.WriteFile(configPath, tampered, 0644); err != nil {
		t.Fatal(err)
	}

	if err := runIntegrityCheck(configPath, new(bytes.Buffer)); err == nil {
		t.Fatal("Expected strict-mode error: HEAD declares strict, so removing the local key must not disable enforcement")
	}
}

// #865 Gap 2 (AC2/AC4): the committed baseline is read with a cwd-relative
// pathspec, so drift is detected when the config lives in a monorepo subdirectory.
func TestRunIntegrityCheck_MonorepoSubdirectoryBaseline(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")

	subdir := filepath.Join(root, "packages", "app")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(subdir, ".gh-pmu.json")
	committed := []byte(`{"project":{"owner":"original","number":1},"repositories":["test/repo"],"configIntegrity":"strict"}`)
	if err := os.WriteFile(configPath, committed, 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "packages/app/.gh-pmu.json")
	runGit(t, root, "commit", "-m", "init")

	// Drift the subdirectory config locally.
	tampered := []byte(`{"project":{"owner":"changed","number":1},"repositories":["test/repo"],"configIntegrity":"strict"}`)
	if err := os.WriteFile(configPath, tampered, 0644); err != nil {
		t.Fatal(err)
	}

	// New (cwd-relative) pathspec reads the subdir baseline -> drift -> strict error.
	// The old repo-root pathspec would fail to find the baseline and silently skip.
	if err := runIntegrityCheck(configPath, new(bytes.Buffer)); err == nil {
		t.Fatal("Expected drift detection against the subdirectory's committed baseline")
	}
}

func TestRunIntegrityCheck_Throttled_Skips(t *testing.T) {
	// ARRANGE: Config drifted BUT throttle is active
	dir := t.TempDir()
	original := []byte(`{"project":{"owner":"original","number":1},"repositories":["test/repo"]}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, original, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	// Modify config
	modified := []byte(`{"project":{"owner":"changed","number":1},"repositories":["test/repo"]}`)
	if err := os.WriteFile(configPath, modified, 0644); err != nil {
		t.Fatal(err)
	}

	// Set throttle to today
	state := integrity.ThrottleState{
		LastCheck: time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(dir, integrity.ThrottleFileName), data, 0644); err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)

	// ACT
	err := runIntegrityCheck(configPath, buf)

	// ASSERT: Throttled, so no output and no error
	if err != nil {
		t.Fatalf("Expected no error when throttled, got: %v", err)
	}
	output := buf.String()
	if output != "" {
		t.Errorf("Expected no output when throttled, got: %q", output)
	}
}

func TestRunIntegrityCheck_Performance(t *testing.T) {
	// ARRANGE: Create config and git repo
	dir := t.TempDir()
	configContent := []byte(`{"project":{"owner":"test","number":1},"repositories":["test/repo"]}`)
	configPath := filepath.Join(dir, ".gh-pmu.json")
	if err := os.WriteFile(configPath, configContent, 0644); err != nil {
		t.Fatal(err)
	}

	runGit(t, dir, "init")
	runGit(t, dir, "add", ".gh-pmu.json")
	runGit(t, dir, "commit", "-m", "init")

	buf := new(bytes.Buffer)

	// ACT: Time the check
	start := time.Now()
	err := runIntegrityCheck(configPath, buf)
	elapsed := time.Since(start)

	// ASSERT: Under 200ms
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("Integrity check took %v, expected <200ms", elapsed)
	}
}
