package cmd

import (
	"bytes"
	"strings"
	"testing"
)

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
