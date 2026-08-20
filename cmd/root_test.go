package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/config"
)

func TestRefreshConfigVersionInDir_StaleVersion_Rewrites(t *testing.T) {
	dir := t.TempDir()
	cfg := baseConfig()
	cfg.Version = "1.1.0"
	path := writeTestConfig(t, dir, cfg)

	var buf bytes.Buffer
	refreshConfigVersionInDir(dir, "1.5.3", &buf)

	reloaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}
	if reloaded.Version != "1.5.3" {
		t.Errorf("Expected version rewritten to '1.5.3', got '%s'", reloaded.Version)
	}
	if buf.Len() != 0 {
		t.Errorf("Expected no warning on the success path, got: %s", buf.String())
	}
}

func TestRefreshConfigVersionInDir_NoConfig_SilentBail(t *testing.T) {
	dir := t.TempDir()

	var buf bytes.Buffer
	refreshConfigVersionInDir(dir, "1.5.3", &buf)

	// An uninitialized repo is the normal case for a fresh checkout: no warning,
	// no file created, no error surfaced to the command.
	if buf.Len() != 0 {
		t.Errorf("Expected silence when no config exists, got: %s", buf.String())
	}
	if _, err := os.Stat(filepath.Join(dir, config.ConfigFileName)); !os.IsNotExist(err) {
		t.Error("Expected no config file to be created")
	}
}

func TestRefreshConfigVersionInDir_CorruptConfig_WarnsWithoutAborting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, config.ConfigFileName)
	if err := os.WriteFile(path, []byte("{not valid json"), 0644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	var buf bytes.Buffer
	refreshConfigVersionInDir(dir, "1.5.3", &buf)

	if !strings.Contains(buf.String(), "Warning") {
		t.Errorf("Expected a warning for an unreadable config, got: %q", buf.String())
	}
}

func TestRefreshConfigVersionInDir_CurrentVersion_LeavesBytesUnchanged(t *testing.T) {
	dir := t.TempDir()
	cfg := baseConfig()
	cfg.Version = "1.5.3"
	path := writeTestConfig(t, dir, cfg)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var buf bytes.Buffer
	refreshConfigVersionInDir(dir, "1.5.3", &buf)

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to re-read config: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Error("Expected config bytes unchanged when the version already matches")
	}
}

func TestShouldRefreshVersion_HonorsExemptCommands(t *testing.T) {
	// checkAcceptance and runDailyIntegrityCheck both skip these; the version
	// refresh must agree or 'gh pmu help' would write the config.
	for name := range exemptCommands {
		if shouldRefreshVersion(name) {
			t.Errorf("Expected exempt command %q to skip the version refresh", name)
		}
	}

	for _, name := range []string{"list", "move", "board", "config"} {
		if !shouldRefreshVersion(name) {
			t.Errorf("Expected non-exempt command %q to run the version refresh", name)
		}
	}
}

func TestRootCommandHelp(t *testing.T) {
	// Test that root command executes and shows help
	cmd := NewRootCommand()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("Expected help output, got empty string")
	}

	// Verify it contains expected content
	if !bytes.Contains([]byte(output), []byte("gh pmu")) {
		t.Errorf("Expected output to contain 'gh pmu', got: %s", output)
	}
}

func TestRootCommandVersion(t *testing.T) {
	cmd := NewRootCommand()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	// Cobra uses first word of Use field for version output
	if !bytes.Contains([]byte(output), []byte("version")) {
		t.Errorf("Expected version output to contain 'version', got: %s", output)
	}
}

func TestRootCommandVersionContainsCopyright(t *testing.T) {
	cmd := NewRootCommand()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Rubrical Works") {
		t.Errorf("Expected version output to contain 'Rubrical Works', got: %s", output)
	}
	if !strings.Contains(output, "(c)") {
		t.Errorf("Expected version output to contain '(c)' copyright marker, got: %s", output)
	}
}

func TestRootCommandVersionFormat(t *testing.T) {
	cmd := NewRootCommand()

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines in version output, got %d: %q", len(lines), output)
	}
	if !strings.HasPrefix(lines[0], "gh pmu version ") {
		t.Errorf("Expected first line to start with 'gh pmu version ', got: %q", lines[0])
	}
	if lines[1] != "Rubrical Works (c) 2026" {
		t.Errorf("Expected second line to be 'Rubrical Works (c) 2026', got: %q", lines[1])
	}
}

// TestSubcommandUsageLinePrefix verifies that `gh pmu <cmd> --help` renders
// `Usage: gh pmu <cmd>...` rather than the buggy `Usage: gh <cmd>...`.
func TestSubcommandUsageLinePrefix(t *testing.T) {
	subcommands := []string{"board", "view", "move", "create"}

	for _, sub := range subcommands {
		t.Run(sub, func(t *testing.T) {
			cmd := NewRootCommand()
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetArgs([]string{sub, "--help"})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("Expected no error running %q --help, got: %v", sub, err)
			}

			output := buf.String()
			want := "Usage:\n  gh pmu " + sub
			if !strings.Contains(output, want) {
				t.Errorf("Expected output to contain %q for subcommand %q.\nFull output:\n%s", want, sub, output)
			}
			bad := "Usage:\n  gh " + sub
			if strings.Contains(output, bad) {
				t.Errorf("Output still contains buggy usage %q for subcommand %q.\nFull output:\n%s", bad, sub, output)
			}
		})
	}
}

// TestRootCommandUsageLinePrefix verifies that `gh pmu --help` renders
// `Usage: gh pmu [command]`.
func TestRootCommandUsageLinePrefix(t *testing.T) {
	cmd := NewRootCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "gh pmu [command]") {
		t.Errorf("Expected root usage to contain 'gh pmu [command]', got:\n%s", output)
	}
}
