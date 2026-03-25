package integrity

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestComputeChecksum_ReturnsSHA256(t *testing.T) {
	// ARRANGE: Create a temp config file
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gh-pmu.json")
	content := []byte(`{"project":{"owner":"test","number":1}}`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// ACT: Compute checksum
	checksum, err := ComputeChecksum(configPath)

	// ASSERT: Returns valid SHA-256 hex string
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	expected := fmt.Sprintf("%x", sha256.Sum256(content))
	if checksum != expected {
		t.Errorf("Expected checksum %q, got %q", expected, checksum)
	}
}

func TestComputeChecksum_MissingFile_ReturnsError(t *testing.T) {
	// ARRANGE: Non-existent file
	path := filepath.Join(t.TempDir(), "missing.json")

	// ACT
	_, err := ComputeChecksum(path)

	// ASSERT
	if err == nil {
		t.Fatal("Expected error for missing file")
	}
}

func TestSaveChecksum_WritesChecksumFile(t *testing.T) {
	// ARRANGE
	dir := t.TempDir()
	checksumPath := filepath.Join(dir, ChecksumFileName)

	// ACT
	err := SaveChecksum(dir, "abc123def456")

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	data, err := os.ReadFile(checksumPath)
	if err != nil {
		t.Fatalf("Expected checksum file to exist: %v", err)
	}
	if string(data) != "abc123def456\n" {
		t.Errorf("Expected checksum content %q, got %q", "abc123def456\n", string(data))
	}
}

func TestLoadChecksum_ReadsChecksumFile(t *testing.T) {
	// ARRANGE
	dir := t.TempDir()
	checksumPath := filepath.Join(dir, ChecksumFileName)
	if err := os.WriteFile(checksumPath, []byte("abc123def456\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// ACT
	checksum, err := LoadChecksum(dir)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if checksum != "abc123def456" {
		t.Errorf("Expected %q, got %q", "abc123def456", checksum)
	}
}

func TestLoadChecksum_MissingFile_ReturnsEmpty(t *testing.T) {
	// ARRANGE
	dir := t.TempDir()

	// ACT
	checksum, err := LoadChecksum(dir)

	// ASSERT: No error, empty checksum
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if checksum != "" {
		t.Errorf("Expected empty string, got %q", checksum)
	}
}

func TestIsThrottled_NeverChecked_ReturnsFalse(t *testing.T) {
	// ARRANGE
	dir := t.TempDir()

	// ACT
	throttled, err := IsThrottled(dir)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if throttled {
		t.Error("Expected not throttled when never checked")
	}
}

func TestIsThrottled_CheckedToday_ReturnsTrue(t *testing.T) {
	// ARRANGE: Write today's date
	dir := t.TempDir()
	state := ThrottleState{
		LastCheck: time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(dir, ThrottleFileName), data, 0644); err != nil {
		t.Fatal(err)
	}

	// ACT
	throttled, err := IsThrottled(dir)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !throttled {
		t.Error("Expected throttled when checked today")
	}
}

func TestIsThrottled_CheckedYesterday_ReturnsFalse(t *testing.T) {
	// ARRANGE: Write yesterday's date
	dir := t.TempDir()
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	state := ThrottleState{
		LastCheck: yesterday.Format(time.RFC3339),
	}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(dir, ThrottleFileName), data, 0644); err != nil {
		t.Fatal(err)
	}

	// ACT
	throttled, err := IsThrottled(dir)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if throttled {
		t.Error("Expected not throttled when checked yesterday")
	}
}

func TestRecordCheck_WritesTimestamp(t *testing.T) {
	// ARRANGE
	dir := t.TempDir()

	// ACT
	err := RecordCheck(dir)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ThrottleFileName))
	if err != nil {
		t.Fatal("Expected throttle file to exist")
	}

	var state ThrottleState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("Expected valid JSON: %v", err)
	}

	parsed, err := time.Parse(time.RFC3339, state.LastCheck)
	if err != nil {
		t.Fatalf("Expected ISO 8601 timestamp: %v", err)
	}

	// Should be within the last minute
	if time.Since(parsed) > time.Minute {
		t.Errorf("Timestamp too old: %v", parsed)
	}
}

func TestCompareConfigs_NoDrift(t *testing.T) {
	// ARRANGE: Same content
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gh-pmu.json")
	content := []byte(`{"project":{"owner":"test","number":1}}`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// ACT
	result, err := CompareContent(content, content)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Drifted {
		t.Error("Expected no drift for identical content")
	}
}

func TestCompareConfigs_WithDrift(t *testing.T) {
	// ARRANGE: Different content
	local := []byte(`{"project":{"owner":"test","number":1}}`)
	committed := []byte(`{"project":{"owner":"original","number":1}}`)

	// ACT
	result, err := CompareContent(local, committed)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !result.Drifted {
		t.Error("Expected drift for different content")
	}
	if len(result.Changes) == 0 {
		t.Error("Expected at least one change reported")
	}
}

func TestUpdateChecksumForConfig_CreatesChecksumFile(t *testing.T) {
	// ARRANGE: Config file exists
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gh-pmu.json")
	content := []byte(`{"project":{"owner":"test","number":1}}`)
	if err := os.WriteFile(configPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// ACT
	err := UpdateChecksumForConfig(configPath)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Checksum file should exist and match
	stored, err := LoadChecksum(dir)
	if err != nil {
		t.Fatalf("Expected to read checksum: %v", err)
	}
	computed, _ := ComputeChecksum(configPath)
	if stored != computed {
		t.Errorf("Stored checksum %q doesn't match computed %q", stored, computed)
	}
}

func TestCompareConfigs_WithDrift_ReportsUnchangedSections(t *testing.T) {
	// ARRANGE: Local differs in one top-level key, others unchanged
	local := []byte(`{
		"version": "1.4.0",
		"project": {"owner": "test", "number": 1},
		"repositories": ["test/repo"],
		"defaults": {"priority": "p2"}
	}`)
	committed := []byte(`{
		"version": "1.1.0",
		"project": {"owner": "test", "number": 1},
		"repositories": ["test/repo"],
		"defaults": {"priority": "p2"}
	}`)

	// ACT
	result, err := CompareContent(local, committed)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !result.Drifted {
		t.Fatal("Expected drift")
	}

	// Changed should mention version
	foundVersion := false
	for _, c := range result.Changes {
		if c == "Changed: version" {
			foundVersion = true
		}
	}
	if !foundVersion {
		t.Errorf("Expected 'Changed: version' in changes, got: %v", result.Changes)
	}

	// Unchanged should include project, repositories, defaults
	unchangedMap := make(map[string]bool)
	for _, u := range result.Unchanged {
		unchangedMap[u] = true
	}
	for _, expected := range []string{"project", "repositories", "defaults"} {
		if !unchangedMap[expected] {
			t.Errorf("Expected %q in unchanged list, got: %v", expected, result.Unchanged)
		}
	}

	// version should NOT be in unchanged
	if unchangedMap["version"] {
		t.Error("version should not be in unchanged list")
	}
}

func TestCompareConfigs_NoDrift_EmptyUnchanged(t *testing.T) {
	// ARRANGE: Identical content
	content := []byte(`{"project":{"owner":"test","number":1}}`)

	// ACT
	result, err := CompareContent(content, content)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if result.Drifted {
		t.Error("Expected no drift")
	}
	if len(result.Unchanged) != 0 {
		t.Errorf("Expected empty unchanged list when no drift, got: %v", result.Unchanged)
	}
}

func TestCompareConfigs_EmptyCommitted_ReportsNewFile(t *testing.T) {
	// ARRANGE: No committed version
	local := []byte(`{"project":{"owner":"test","number":1}}`)

	// ACT
	result, err := CompareContent(local, nil)

	// ASSERT
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if !result.Drifted {
		t.Error("Expected drift when no committed version")
	}
}
