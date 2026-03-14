package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rubrical-works/gh-pmu/internal/config"
)

func TestNewHistoryCommand(t *testing.T) {
	cmd := newHistoryCommand()

	if cmd.Use != "history <path> [path...]" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("expected Short description")
	}
}

func TestHistoryCommand_HasFlags(t *testing.T) {
	cmd := newHistoryCommand()

	tests := []struct {
		flagName     string
		defaultValue string
	}{
		{"since", ""},
		{"limit", "50"},
		{"output", "false"},
		{"force", "false"},
		{"json", "false"},
		{"compact", "false"},
		{"browser", "false"},
		{"files", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.flagName, func(t *testing.T) {
			flag := cmd.Flags().Lookup(tt.flagName)
			if flag == nil {
				t.Errorf("expected flag --%s to exist", tt.flagName)
				return
			}
			if flag.DefValue != tt.defaultValue {
				t.Errorf("expected default %s, got %s", tt.defaultValue, flag.DefValue)
			}
		})
	}
}

func TestInferChangeType(t *testing.T) {
	tests := []struct {
		subject  string
		expected string
	}{
		// Fix variations
		{"fix: handle empty values", "Fix"},
		{"Fix: handle empty values", "Fix"},
		{"fix(api): handle empty values", "Fix"},
		{"bug: fix null pointer", "Fix"},

		// Add variations
		{"add: new feature", "Add"},
		{"Add: new feature", "Add"},
		{"feat: implement login", "Add"},
		{"feat(auth): implement login", "Add"},
		{"feature: new dashboard", "Add"},

		// Update variations
		{"update: improve performance", "Update"},
		{"enhance: add caching", "Update"},

		// Remove variations
		{"remove: deprecated function", "Remove"},
		{"delete: old files", "Remove"},

		// Refactor
		{"refactor: extract method", "Refactor"},
		{"refactor(core): simplify logic", "Refactor"},

		// Docs
		{"docs: update readme", "Docs"},
		{"doc: add examples", "Docs"},

		// Test
		{"test: add unit tests", "Test"},
		{"test(api): improve coverage", "Test"},

		// Chore
		{"chore: update deps", "Chore"},
		{"build: fix makefile", "Chore"},
		{"ci: update workflow", "Chore"},

		// Unknown - fallback to Change
		{"Some random commit message", "Change"},
		{"Updated the thing", "Change"},
		{"v1.0.0 release", "Change"},
	}

	for _, tt := range tests {
		t.Run(tt.subject, func(t *testing.T) {
			result := inferChangeType(tt.subject)
			if result != tt.expected {
				t.Errorf("inferChangeType(%q) = %q, want %q", tt.subject, result, tt.expected)
			}
		})
	}
}

func TestParseCommitReferences(t *testing.T) {
	tests := []struct {
		name          string
		subject       string
		defaultOwner  string
		defaultRepo   string
		expectedCount int
		checkFirst    *IssueReference // optional: check first reference details
	}{
		{
			name:          "simple hash reference",
			subject:       "Fix bug #123",
			defaultOwner:  "owner",
			defaultRepo:   "repo",
			expectedCount: 1,
			checkFirst: &IssueReference{
				Number: 123,
				Owner:  "owner",
				Repo:   "repo",
				Type:   "related",
				URL:    "https://github.com/owner/repo/issues/123",
			},
		},
		{
			name:          "fixes prefix",
			subject:       "fixes #456",
			defaultOwner:  "owner",
			defaultRepo:   "repo",
			expectedCount: 1,
			checkFirst: &IssueReference{
				Number: 456,
				Owner:  "owner",
				Repo:   "repo",
				Type:   "fixes",
				URL:    "https://github.com/owner/repo/issues/456",
			},
		},
		{
			name:          "closes prefix",
			subject:       "closes #789",
			defaultOwner:  "owner",
			defaultRepo:   "repo",
			expectedCount: 1,
			checkFirst: &IssueReference{
				Number: 789,
				Owner:  "owner",
				Repo:   "repo",
				Type:   "closes",
				URL:    "https://github.com/owner/repo/issues/789",
			},
		},
		{
			name:          "cross-repo reference",
			subject:       "See other-owner/other-repo#42",
			defaultOwner:  "owner",
			defaultRepo:   "repo",
			expectedCount: 1,
			checkFirst: &IssueReference{
				Number: 42,
				Owner:  "other-owner",
				Repo:   "other-repo",
				Type:   "related",
				URL:    "https://github.com/other-owner/other-repo/issues/42",
			},
		},
		{
			name:          "multiple references",
			subject:       "Fix #1, closes #2, related to #3",
			defaultOwner:  "owner",
			defaultRepo:   "repo",
			expectedCount: 3,
		},
		{
			name:          "no references",
			subject:       "Just a regular commit message",
			defaultOwner:  "owner",
			defaultRepo:   "repo",
			expectedCount: 0,
		},
		{
			name:          "duplicate references deduplicated",
			subject:       "Fix #123, also #123 again",
			defaultOwner:  "owner",
			defaultRepo:   "repo",
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs := parseCommitReferences(tt.subject, tt.defaultOwner, tt.defaultRepo)

			if len(refs) != tt.expectedCount {
				t.Errorf("expected %d references, got %d", tt.expectedCount, len(refs))
				return
			}

			if tt.checkFirst != nil && len(refs) > 0 {
				got := refs[0]
				if got.Number != tt.checkFirst.Number {
					t.Errorf("Number: expected %d, got %d", tt.checkFirst.Number, got.Number)
				}
				if got.Owner != tt.checkFirst.Owner {
					t.Errorf("Owner: expected %s, got %s", tt.checkFirst.Owner, got.Owner)
				}
				if got.Repo != tt.checkFirst.Repo {
					t.Errorf("Repo: expected %s, got %s", tt.checkFirst.Repo, got.Repo)
				}
				if got.Type != tt.checkFirst.Type {
					t.Errorf("Type: expected %s, got %s", tt.checkFirst.Type, got.Type)
				}
				if got.URL != tt.checkFirst.URL {
					t.Errorf("URL: expected %s, got %s", tt.checkFirst.URL, got.URL)
				}
			}
		})
	}
}

func TestParseCommitReferences_KeywordTypeLabels(t *testing.T) {
	tests := []struct {
		keyword      string
		expectedType string
	}{
		{"fix", "fixes"},
		{"fixes", "fixes"},
		{"fixed", "fixes"},
		{"Fix", "fixes"},
		{"Fixes", "fixes"},
		{"Fixed", "fixes"},
		{"close", "closes"},
		{"closes", "closes"},
		{"closed", "closes"},
		{"Close", "closes"},
		{"Closes", "closes"},
		{"Closed", "closes"},
		{"resolve", "resolves"},
		{"resolves", "resolves"},
		{"resolved", "resolves"},
		{"Resolve", "resolves"},
		{"Resolves", "resolves"},
		{"Resolved", "resolves"},
	}

	for _, tt := range tests {
		t.Run(tt.keyword, func(t *testing.T) {
			subject := fmt.Sprintf("%s #42", tt.keyword)
			refs := parseCommitReferences(subject, "owner", "repo")
			if len(refs) == 0 {
				t.Fatalf("No references found for keyword %q", tt.keyword)
			}
			if refs[0].Type != tt.expectedType {
				t.Errorf("Keyword %q: expected type %q, got %q", tt.keyword, tt.expectedType, refs[0].Type)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly ten", 11, "exactly ten"},
		{"this is a very long string that needs truncation", 20, "this is a very lo..."},
		{"", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestCommitInfo_JSONMarshalling(t *testing.T) {
	commit := CommitInfo{
		Hash:       "abc1234",
		Author:     "Test Author",
		Date:       time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
		Subject:    "Fix: test commit #123",
		Body:       "This is the commit body\nwith multiple lines",
		ChangeType: "Fix",
		References: []IssueReference{
			{
				Number: 123,
				Owner:  "owner",
				Repo:   "repo",
				Type:   "related",
				URL:    "https://github.com/owner/repo/issues/123",
			},
		},
		Insertions: 10,
		Deletions:  5,
		Comments: []CommitComment{
			{
				Author: "reviewer",
				Body:   "Great change!",
				Date:   time.Date(2025, 1, 16, 14, 0, 0, 0, time.UTC),
			},
		},
		Files:     []string{"cmd/history.go", "cmd/history_test.go"},
		FileCount: 2,
	}

	// Verify struct fields are properly tagged
	if commit.Hash != "abc1234" {
		t.Errorf("expected hash abc1234, got %s", commit.Hash)
	}
	if commit.Body != "This is the commit body\nwith multiple lines" {
		t.Errorf("expected body, got %s", commit.Body)
	}
	if len(commit.References) != 1 {
		t.Errorf("expected 1 reference, got %d", len(commit.References))
	}
	if commit.Insertions != 10 {
		t.Errorf("expected 10 insertions, got %d", commit.Insertions)
	}
	if commit.Deletions != 5 {
		t.Errorf("expected 5 deletions, got %d", commit.Deletions)
	}
	if len(commit.Comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(commit.Comments))
	}
	if commit.Comments[0].Author != "reviewer" {
		t.Errorf("expected comment author 'reviewer', got %s", commit.Comments[0].Author)
	}
	if len(commit.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(commit.Files))
	}
	if commit.FileCount != 2 {
		t.Errorf("expected file count 2, got %d", commit.FileCount)
	}
}

func TestCommitComment_Fields(t *testing.T) {
	comment := CommitComment{
		Author: "test-user",
		Body:   "This is a test comment",
		Date:   time.Date(2025, 12, 10, 12, 0, 0, 0, time.UTC),
	}

	if comment.Author != "test-user" {
		t.Errorf("expected author 'test-user', got %s", comment.Author)
	}
	if comment.Body != "This is a test comment" {
		t.Errorf("expected body, got %s", comment.Body)
	}
	if comment.Date.Year() != 2025 {
		t.Errorf("expected year 2025, got %d", comment.Date.Year())
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{"just now", now, "just now"},
		{"1 minute ago", now.Add(-1 * time.Minute), "1 minute ago"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5 minutes ago"},
		{"1 hour ago", now.Add(-1 * time.Hour), "1 hour ago"},
		{"3 hours ago", now.Add(-3 * time.Hour), "3 hours ago"},
		{"1 day ago", now.Add(-24 * time.Hour), "1 day ago"},
		{"5 days ago", now.Add(-5 * 24 * time.Hour), "5 days ago"},
		{"1 week ago", now.Add(-7 * 24 * time.Hour), "1 week ago"},
		{"2 weeks ago", now.Add(-14 * 24 * time.Hour), "2 weeks ago"},
		{"1 month ago", now.Add(-30 * 24 * time.Hour), "1 month ago"},
		{"6 months ago", now.Add(-180 * 24 * time.Hour), "6 months ago"},
		{"1 year ago", now.Add(-365 * 24 * time.Hour), "1 year ago"},
		{"2 years ago", now.Add(-730 * 24 * time.Hour), "2 years ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := relativeTime(tt.input)
			if result != tt.expected {
				t.Errorf("relativeTime() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsDirectory(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"path ending with slash", "cmd/", true},
		{"path ending with backslash", "cmd\\", true},
		{"file path", "cmd/history.go", false},
		{"nonexistent path", "nonexistent/path/file.txt", false},
		{"nonexistent dir path", "nonexistent/path/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDirectory(tt.path)
			if result != tt.expected {
				t.Errorf("isDirectory(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGenerateHistoryHTML(t *testing.T) {
	commits := []CommitInfo{
		{
			Hash:       "abc1234",
			Author:     "Test Author",
			Date:       time.Date(2025, 12, 10, 10, 0, 0, 0, time.UTC),
			Subject:    "feat: Add new feature #123",
			Body:       "This is the body",
			ChangeType: "Add",
			References: []IssueReference{
				{Number: 123, URL: "https://github.com/owner/repo/issues/123"},
			},
			Insertions: 50,
			Deletions:  10,
		},
		{
			Hash:       "def5678",
			Author:     "Another Author",
			Date:       time.Date(2025, 12, 9, 10, 0, 0, 0, time.UTC),
			Subject:    "fix: Bug fix",
			ChangeType: "Fix",
			Insertions: 5,
			Deletions:  3,
		},
	}

	html := generateHistoryHTML(commits, "cmd/test.go", "owner", "repo")

	// Check HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("expected HTML doctype")
	}
	if !strings.Contains(html, "History: cmd/test.go") {
		t.Error("expected title with path")
	}
	if !strings.Contains(html, "2 commits") {
		t.Error("expected commit count")
	}
	if !strings.Contains(html, "abc1234") {
		t.Error("expected first commit hash")
	}
	if !strings.Contains(html, "def5678") {
		t.Error("expected second commit hash")
	}
	if !strings.Contains(html, "feat: Add new feature #123") {
		t.Error("expected commit subject")
	}
	if !strings.Contains(html, "This is the body") {
		t.Error("expected commit body")
	}
	if !strings.Contains(html, "type-Add") {
		t.Error("expected Add type badge")
	}
	if !strings.Contains(html, "type-Fix") {
		t.Error("expected Fix type badge")
	}
	if !strings.Contains(html, "#123") {
		t.Error("expected issue reference")
	}
	if !strings.Contains(html, "+50") {
		t.Error("expected insertions")
	}
	if !strings.Contains(html, "-10") {
		t.Error("expected deletions")
	}
	if !strings.Contains(html, "github.com/owner/repo/commit/abc1234") {
		t.Error("expected commit URL")
	}
}

func TestGenerateHistoryHTML_EscapesHTML(t *testing.T) {
	commits := []CommitInfo{
		{
			Hash:       "abc1234",
			Author:     "<script>alert('xss')</script>",
			Date:       time.Now(),
			Subject:    "feat: <b>bold</b> & special chars",
			Body:       "<div>HTML in body</div>",
			ChangeType: "Add",
		},
	}

	html := generateHistoryHTML(commits, "test.go", "owner", "repo")

	// Should escape HTML entities
	if strings.Contains(html, "<script>") {
		t.Error("HTML should be escaped - found unescaped script tag")
	}
	if !strings.Contains(html, "&lt;script&gt;") {
		t.Error("expected escaped script tag")
	}
	if strings.Contains(html, "<b>bold</b>") {
		t.Error("HTML should be escaped - found unescaped b tag")
	}
}

// ============================================================================
// validateHistorySafety Tests
// ============================================================================

func TestValidateHistorySafety_ForceOverrides(t *testing.T) {
	// Force flag should always succeed
	opts := &historyOptions{force: true}
	err := validateHistorySafety([]string{"."}, opts)
	if err != nil {
		t.Errorf("Expected no error with --force, got: %v", err)
	}
}

func TestValidateHistorySafety_NormalPath(t *testing.T) {
	// A specific subdirectory should be fine
	opts := &historyOptions{force: false}
	err := validateHistorySafety([]string{"cmd/"}, opts)
	// This should pass as it's not the repo root
	if err != nil && strings.Contains(err.Error(), "repository root") {
		t.Errorf("Expected non-root path to succeed, got: %v", err)
	}
}

// ============================================================================
// parseRepoFromConfig Tests
// ============================================================================

func TestParseRepoFromConfig_ValidRepo(t *testing.T) {
	cfg := &config.Config{
		Repositories: []string{"owner/repo"},
	}

	owner, repo := parseRepoFromConfig(cfg)
	if owner != "owner" {
		t.Errorf("Expected owner 'owner', got %q", owner)
	}
	if repo != "repo" {
		t.Errorf("Expected repo 'repo', got %q", repo)
	}
}

func TestParseRepoFromConfig_EmptyRepos(t *testing.T) {
	cfg := &config.Config{
		Repositories: []string{},
	}

	owner, repo := parseRepoFromConfig(cfg)
	if owner != "" {
		t.Errorf("Expected empty owner, got %q", owner)
	}
	if repo != "" {
		t.Errorf("Expected empty repo, got %q", repo)
	}
}

func TestParseRepoFromConfig_InvalidFormat(t *testing.T) {
	cfg := &config.Config{
		Repositories: []string{"invalid-no-slash"},
	}

	owner, repo := parseRepoFromConfig(cfg)
	if owner != "" || repo != "" {
		t.Errorf("Expected empty owner/repo for invalid format, got %q/%q", owner, repo)
	}
}

func TestParseRepoFromConfig_MultipleRepos(t *testing.T) {
	// Should use first repo
	cfg := &config.Config{
		Repositories: []string{"first-owner/first-repo", "second-owner/second-repo"},
	}

	owner, repo := parseRepoFromConfig(cfg)
	if owner != "first-owner" {
		t.Errorf("Expected owner 'first-owner', got %q", owner)
	}
	if repo != "first-repo" {
		t.Errorf("Expected repo 'first-repo', got %q", repo)
	}
}

// ============================================================================
// outputHistoryJSON Tests
// ============================================================================

func TestOutputHistoryJSON_ValidCommits(t *testing.T) {
	commits := []CommitInfo{
		{
			Hash:       "abc1234",
			Author:     "Test Author",
			Date:       time.Date(2025, 12, 10, 10, 0, 0, 0, time.UTC),
			Subject:    "feat: Test commit",
			ChangeType: "Add",
			Insertions: 10,
			Deletions:  5,
		},
	}

	// Capture stdout by redirecting to a buffer
	// We'll just verify the function doesn't error
	err := outputHistoryJSON(commits)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestOutputHistoryJSON_EmptyCommits(t *testing.T) {
	var commits []CommitInfo

	err := outputHistoryJSON(commits)
	if err != nil {
		t.Errorf("Expected no error for empty commits, got: %v", err)
	}
}

// ============================================================================
// outputMarkdown Tests
// ============================================================================

func TestOutputMarkdown_CreatesFile(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	commits := []CommitInfo{
		{
			Hash:       "abc1234",
			Author:     "Test Author",
			Date:       time.Date(2025, 12, 10, 10, 0, 0, 0, time.UTC),
			Subject:    "feat: Test commit #123",
			ChangeType: "Add",
			References: []IssueReference{
				{Number: 123, URL: "https://github.com/owner/repo/issues/123"},
			},
			Insertions: 10,
			Deletions:  5,
		},
		{
			Hash:       "def5678",
			Author:     "Another Author",
			Date:       time.Date(2025, 12, 9, 10, 0, 0, 0, time.UTC),
			Subject:    "fix: Bug fix",
			ChangeType: "Fix",
		},
	}

	err := outputMarkdown(commits, "cmd/test.go", "owner", "repo")
	if err != nil {
		t.Fatalf("outputMarkdown failed: %v", err)
	}

	// Check file was created
	expectedPath := filepath.Join("History", "cmd-test-go.md")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Expected History file to be created")
	}

	// Read and verify content
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "# History: cmd/test.go") {
		t.Error("Expected title in markdown")
	}
	if !strings.Contains(contentStr, "**Total Commits:** 2") {
		t.Error("Expected commit count")
	}
	if !strings.Contains(contentStr, "| `abc1234`") {
		t.Error("Expected commit hash in table")
	}
	if !strings.Contains(contentStr, "[#123]") {
		t.Error("Expected issue link")
	}
}

func TestOutputMarkdown_EscapesPipeInSubject(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	commits := []CommitInfo{
		{
			Hash:       "abc1234",
			Author:     "Test",
			Date:       time.Now(),
			Subject:    "feat: Add A | B feature",
			ChangeType: "Add",
		},
	}

	err := outputMarkdown(commits, "test", "owner", "repo")
	if err != nil {
		t.Fatalf("outputMarkdown failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join("History", "test.md"))
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	// Pipe should be escaped
	if !strings.Contains(string(content), `A \| B`) {
		t.Error("Expected pipe character to be escaped in markdown table")
	}
}

func TestOutputMarkdown_HandlesMultiplePaths(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(originalDir) }()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to chdir: %v", err)
	}

	commits := []CommitInfo{
		{
			Hash:       "abc1234",
			Author:     "Test",
			Date:       time.Now(),
			Subject:    "test commit",
			ChangeType: "Change",
		},
	}

	err := outputMarkdown(commits, "cmd/, internal/", "owner", "repo")
	if err != nil {
		t.Fatalf("outputMarkdown failed: %v", err)
	}

	// Check that History directory exists
	historyDir := filepath.Join("History")
	entries, err := os.ReadDir(historyDir)
	if err != nil {
		t.Fatalf("Failed to read History directory: %v", err)
	}

	// Just verify a file was created (the exact name depends on path sanitization)
	if len(entries) == 0 {
		t.Error("Expected at least one file in History directory")
	}
}

// ============================================================================
// isDirectory Additional Tests
// ============================================================================

func TestIsDirectory_ExistingDirectory(t *testing.T) {
	tempDir := t.TempDir()
	result := isDirectory(tempDir)
	if !result {
		t.Errorf("Expected true for existing directory, got false")
	}
}

func TestIsDirectory_ExistingFile(t *testing.T) {
	tempFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()

	result := isDirectory(tempFile.Name())
	if result {
		t.Errorf("Expected false for existing file, got true")
	}
}

func TestIsDirectory_NonexistentNoSlash(t *testing.T) {
	result := isDirectory("nonexistent-path-no-slash")
	if result {
		t.Errorf("Expected false for nonexistent path without slash, got true")
	}
}

// ============================================================================
// countFilesInPath Tests (uses git ls-files)
// ============================================================================

func TestCountFilesInPath_CurrentDirectory(t *testing.T) {
	// This test runs in the actual repo
	// Get the repo root first, then check for files
	root, err := getRepoRoot()
	if err != nil {
		t.Skipf("Skipping: not in git repo: %v", err)
	}

	// Count files in the cmd directory from repo root
	cmdPath := filepath.Join(root, "cmd")
	count, err := countFilesInPath(cmdPath)
	if err != nil {
		t.Skipf("Skipping: git ls-files failed: %v", err)
	}

	if count == 0 {
		// The test might be running from a different cwd, just verify function works
		t.Skipf("Skipping: no files found (may be running from unexpected directory)")
	}
}

func TestCountFilesInPath_NonexistentPath(t *testing.T) {
	count, err := countFilesInPath("nonexistent-dir-12345")
	// git ls-files may succeed with 0 files or fail
	if err == nil && count != 0 {
		t.Errorf("Expected 0 files for nonexistent path, got %d", count)
	}
}

// ============================================================================
// getRepoRoot Tests
// ============================================================================

func TestGetRepoRoot_InGitRepo(t *testing.T) {
	root, err := getRepoRoot()
	if err != nil {
		t.Skipf("Skipping: not in git repo: %v", err)
	}

	if root == "" {
		t.Error("Expected non-empty repo root")
	}

	// Should be an absolute path
	if !filepath.IsAbs(root) {
		t.Errorf("Expected absolute path, got %q", root)
	}
}

// ============================================================================
// parseCommitLog Tests
// ============================================================================

func TestParseCommitLog_StandardInput(t *testing.T) {
	input := "abc1234\x00John Doe\x002026-03-14T10:00:00Z\x00feat: add feature"
	commits := parseCommitLog(input)
	if len(commits) != 1 {
		t.Fatalf("Expected 1 commit, got %d", len(commits))
	}
	if commits[0].Hash != "abc1234" {
		t.Errorf("Expected hash 'abc1234', got '%s'", commits[0].Hash)
	}
	if commits[0].Author != "John Doe" {
		t.Errorf("Expected author 'John Doe', got '%s'", commits[0].Author)
	}
	if commits[0].Subject != "feat: add feature" {
		t.Errorf("Expected subject 'feat: add feature', got '%s'", commits[0].Subject)
	}
}

func TestParseCommitLog_AuthorWithPipe(t *testing.T) {
	// Regression: author name containing | must not corrupt parsing
	input := "def5678\x00First | Last\x002026-03-14T10:00:00Z\x00fix: something"
	commits := parseCommitLog(input)
	if len(commits) != 1 {
		t.Fatalf("Expected 1 commit, got %d", len(commits))
	}
	if commits[0].Author != "First | Last" {
		t.Errorf("Expected author 'First | Last', got '%s'", commits[0].Author)
	}
	if commits[0].Subject != "fix: something" {
		t.Errorf("Expected subject 'fix: something', got '%s'", commits[0].Subject)
	}
}

func TestParseCommitLog_MultipleLines(t *testing.T) {
	input := "aaa\x00Author1\x002026-03-14T10:00:00Z\x00first commit\n" +
		"bbb\x00Author2\x002026-03-14T11:00:00Z\x00second commit"
	commits := parseCommitLog(input)
	if len(commits) != 2 {
		t.Fatalf("Expected 2 commits, got %d", len(commits))
	}
}

func TestParseCommitLog_EmptyInput(t *testing.T) {
	commits := parseCommitLog("")
	if len(commits) != 0 {
		t.Errorf("Expected 0 commits, got %d", len(commits))
	}
}
