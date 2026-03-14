package cmd

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/rubrical-works/gh-pmu/internal/config"
	"github.com/spf13/cobra"
)

// historyOptions holds command flags
type historyOptions struct {
	since   string // Date filter (e.g., "2024-01-01", "1 week ago")
	limit   int    // Max commits (default 50)
	output  bool   // Write to History/ directory
	force   bool   // Override safety limits
	json    bool   // JSON output format
	compact bool   // Force compact mode even for single file
	browser bool   // Open in web browser
	files   bool   // Show affected files for each commit
}

// CommitInfo represents parsed information from a git commit
type CommitInfo struct {
	Hash       string           `json:"hash"`
	Author     string           `json:"author"`
	Date       time.Time        `json:"date"`
	Subject    string           `json:"subject"`
	Body       string           `json:"body,omitempty"`
	ChangeType string           `json:"change_type"`
	References []IssueReference `json:"references,omitempty"`
	Insertions int              `json:"insertions,omitempty"`
	Deletions  int              `json:"deletions,omitempty"`
	Comments   []CommitComment  `json:"comments,omitempty"`
	Files      []string         `json:"files,omitempty"`
	FileCount  int              `json:"file_count,omitempty"`
}

// CommitComment represents a GitHub comment on a commit
type CommitComment struct {
	Author string    `json:"author"`
	Body   string    `json:"body"`
	Date   time.Time `json:"date"`
}

// IssueReference represents a parsed issue/PR reference
type IssueReference struct {
	Number int    `json:"number"`
	Owner  string `json:"owner,omitempty"`
	Repo   string `json:"repo,omitempty"`
	Type   string `json:"type"` // fixes, closes, related
	URL    string `json:"url"`
}

// Lipgloss styles
var (
	historyHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)

	hashStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	dateStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	authorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	changeTypeStyles = map[string]lipgloss.Style{
		"Fix":      lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
		"Add":      lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true),
		"Update":   lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true),
		"Remove":   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		"Refactor": lipgloss.NewStyle().Foreground(lipgloss.Color("33")),
		"Docs":     lipgloss.NewStyle().Foreground(lipgloss.Color("141")),
		"Test":     lipgloss.NewStyle().Foreground(lipgloss.Color("208")),
		"Chore":    lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		"Change":   lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
	}

	issueRefStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75"))

	summaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)
)

func newHistoryCommand() *cobra.Command {
	opts := &historyOptions{
		limit: 50,
	}

	cmd := &cobra.Command{
		Use:   "history <path> [path...]",
		Short: "Show git commit history with issue references",
		Long: `Show git commit history for file(s) or directories with issue/PR references.

Parses commit messages for issue references (#123, fixes #456) and creates
links to GitHub issues. Infers change type from commit prefixes (Fix:, Add:, etc.).

Safety protections:
  - Refuses to run from repository root without explicit path
  - Limited to 25 files by default (use --force to override)

Examples:
  gh pmu history cmd/move.go                    # History for single file
  gh pmu history cmd/                           # History for directory
  gh pmu history internal/api/ --since "1 week" # Recent changes only
  gh pmu history cmd/ --output                  # Write to History/ directory
  gh pmu history . --force                      # Override safety limits
  gh pmu history cmd/ --json                    # JSON output
  gh pmu history cmd/ --limit 100               # More commits`,
		Args: cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHistory(cmd, args, opts)
		},
	}

	cmd.Flags().StringVar(&opts.since, "since", "", "Show commits since date (e.g., '2024-01-01', '1 week ago')")
	cmd.Flags().IntVar(&opts.limit, "limit", 50, "Maximum number of commits to show")
	cmd.Flags().BoolVar(&opts.output, "output", false, "Write output to History/ directory as markdown")
	cmd.Flags().BoolVar(&opts.force, "force", false, "Override safety limits (root directory, file count)")
	cmd.Flags().BoolVar(&opts.json, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&opts.compact, "compact", false, "Force compact output even for single file")
	cmd.Flags().BoolVar(&opts.browser, "browser", false, "Open history in web browser")
	cmd.Flags().BoolVar(&opts.files, "files", false, "Show affected files for each commit")

	return cmd
}

func runHistory(cmd *cobra.Command, args []string, opts *historyOptions) error {
	// Load config for repo info
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	cfg, err := config.LoadFromDirectory(cwd)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w\nRun 'gh pmu init' to create a configuration file", err)
	}

	// Default to current directory if no args
	paths := args
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Safety validation
	if err := validateHistorySafety(paths, opts); err != nil {
		return err
	}

	// Get commit history
	commits, err := getCommitHistory(paths, opts.since, opts.limit)
	if err != nil {
		return fmt.Errorf("failed to get commit history: %w", err)
	}

	if len(commits) == 0 {
		fmt.Println("No commits found for the specified path(s).")
		return nil
	}

	// Get repo info for issue URLs
	repoOwner, repoName := parseRepoFromConfig(cfg)

	// Determine if single file mode (detailed view)
	isSingleFile := len(paths) == 1 && !isDirectory(paths[0])

	// Parse references and infer change types
	for i := range commits {
		commits[i].ChangeType = inferChangeType(commits[i].Subject)
		commits[i].References = parseCommitReferences(commits[i].Subject, repoOwner, repoName)

		// Fetch additional details for single file mode (or JSON output)
		if isSingleFile || opts.json {
			commits[i].Insertions, commits[i].Deletions = getCommitStats(commits[i].Hash, paths)
			commits[i].Body = getCommitBody(commits[i].Hash)
			commits[i].Comments = getCommitComments(commits[i].Hash, repoOwner, repoName)
		}

		// Fetch file info for directory mode (or when --files flag is used)
		if !isSingleFile || opts.files {
			commits[i].Files, commits[i].FileCount = getCommitFiles(commits[i].Hash, paths)
		}
	}

	// Generate target path string for display
	targetPath := strings.Join(paths, ", ")

	// Output based on format
	if opts.json {
		return outputHistoryJSON(commits)
	}

	if opts.browser {
		return openHistoryInBrowser(commits, targetPath, repoOwner, repoName)
	}

	if opts.output {
		return outputMarkdown(commits, targetPath, repoOwner, repoName)
	}

	// Default: styled screen output
	if isSingleFile && !opts.compact {
		renderDetailedHistoryScreen(commits, targetPath)
	} else {
		renderHistoryScreen(commits, targetPath, opts.files)
	}
	return nil
}

// validateHistorySafety checks safety constraints
func validateHistorySafety(paths []string, opts *historyOptions) error {
	if opts.force {
		return nil
	}

	// Check for repository root
	repoRoot, err := getRepoRoot()
	if err != nil {
		return nil // Can't determine, let git handle it
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil
	}

	for _, path := range paths {
		absPath := path
		if !filepath.IsAbs(path) {
			absPath = filepath.Join(cwd, path)
		}
		absPath = filepath.Clean(absPath)

		if absPath == repoRoot || path == "." {
			// Check if we're at repo root
			if cwd == repoRoot {
				return fmt.Errorf("refusing to run at repository root\n" +
					"Specify a subdirectory or file, or use --force to override")
			}
		}
	}

	// Check file count
	totalFiles := 0
	for _, path := range paths {
		count, err := countFilesInPath(path)
		if err != nil {
			continue
		}
		totalFiles += count
	}

	if totalFiles > 25 {
		return fmt.Errorf("path contains %d files (limit is 25)\n"+
			"Use --force to override this limit", totalFiles)
	}

	return nil
}

// getRepoRoot returns the git repository root
func getRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// countFilesInPath counts tracked files in a path
func countFilesInPath(path string) (int, error) {
	cmd := exec.Command("git", "ls-files", path)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0, nil
	}
	return len(lines), nil
}

// getCommitHistory executes git log and parses the output
func getCommitHistory(paths []string, since string, limit int) ([]CommitInfo, error) {
	args := []string{
		"log",
		"--format=%h%x00%an%x00%aI%x00%s",
		fmt.Sprintf("--max-count=%d", limit),
	}

	if since != "" {
		args = append(args, fmt.Sprintf("--since=%s", since))
	}

	args = append(args, "--")
	args = append(args, paths...)

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	return parseCommitLog(strings.TrimSpace(string(output))), nil
}

// parseCommitLog parses git log output using null byte delimiters.
// Format: %h%x00%an%x00%aI%x00%s (one commit per line).
func parseCommitLog(output string) []CommitInfo {
	if output == "" {
		return nil
	}

	lines := strings.Split(output, "\n")
	var commits []CommitInfo
	for _, line := range lines {
		parts := strings.SplitN(line, "\x00", 4)
		if len(parts) != 4 {
			continue
		}

		date, _ := time.Parse(time.RFC3339, parts[2])
		commits = append(commits, CommitInfo{
			Hash:    parts[0],
			Author:  parts[1],
			Date:    date,
			Subject: parts[3],
		})
	}

	return commits
}

// inferChangeType determines the change type from commit subject prefix
func inferChangeType(subject string) string {
	prefixMap := map[string]string{
		"fix":      "Fix",
		"bug":      "Fix",
		"add":      "Add",
		"feat":     "Add",
		"feature":  "Add",
		"update":   "Update",
		"enhance":  "Update",
		"remove":   "Remove",
		"delete":   "Remove",
		"refactor": "Refactor",
		"docs":     "Docs",
		"doc":      "Docs",
		"test":     "Test",
		"chore":    "Chore",
		"build":    "Chore",
		"ci":       "Chore",
	}

	lowerSubject := strings.ToLower(subject)
	for prefix, changeType := range prefixMap {
		if strings.HasPrefix(lowerSubject, prefix+":") ||
			strings.HasPrefix(lowerSubject, prefix+"(") ||
			strings.HasPrefix(lowerSubject, prefix+" ") {
			return changeType
		}
	}
	return "Change"
}

// parseCommitReferences extracts all issue/PR references from commit message
func parseCommitReferences(subject, defaultOwner, defaultRepo string) []IssueReference {
	var refs []IssueReference
	seen := make(map[int]bool)

	// Pattern: fixes #123, closes #456, resolves #789
	actionPattern := regexp.MustCompile(`(?i)(fix(?:e[sd])?|close[sd]?|resolve[sd]?)\s+#(\d+)`)
	actionTypeMap := map[string]string{
		"fix": "fixes", "fixes": "fixes", "fixed": "fixes",
		"close": "closes", "closes": "closes", "closed": "closes",
		"resolve": "resolves", "resolves": "resolves", "resolved": "resolves",
	}
	actionMatches := actionPattern.FindAllStringSubmatch(subject, -1)
	for _, match := range actionMatches {
		num, _ := strconv.Atoi(match[2])
		if !seen[num] {
			seen[num] = true
			keyword := strings.ToLower(match[1])
			refType := actionTypeMap[keyword]
			if refType == "" {
				refType = keyword
			}
			refs = append(refs, IssueReference{
				Number: num,
				Owner:  defaultOwner,
				Repo:   defaultRepo,
				Type:   refType,
				URL:    fmt.Sprintf("https://github.com/%s/%s/issues/%d", defaultOwner, defaultRepo, num),
			})
		}
	}

	// Pattern: owner/repo#123
	crossRepoPattern := regexp.MustCompile(`(\w[\w-]*)/(\w[\w-]*)#(\d+)`)
	crossMatches := crossRepoPattern.FindAllStringSubmatch(subject, -1)
	for _, match := range crossMatches {
		num, _ := strconv.Atoi(match[3])
		if !seen[num] {
			seen[num] = true
			refs = append(refs, IssueReference{
				Number: num,
				Owner:  match[1],
				Repo:   match[2],
				Type:   "related",
				URL:    fmt.Sprintf("https://github.com/%s/%s/issues/%d", match[1], match[2], num),
			})
		}
	}

	// Pattern: simple #123 (not already captured)
	simplePattern := regexp.MustCompile(`(?:^|[^/\w])#(\d+)`)
	simpleMatches := simplePattern.FindAllStringSubmatch(subject, -1)
	for _, match := range simpleMatches {
		num, _ := strconv.Atoi(match[1])
		if !seen[num] {
			seen[num] = true
			refs = append(refs, IssueReference{
				Number: num,
				Owner:  defaultOwner,
				Repo:   defaultRepo,
				Type:   "related",
				URL:    fmt.Sprintf("https://github.com/%s/%s/issues/%d", defaultOwner, defaultRepo, num),
			})
		}
	}

	return refs
}

// parseRepoFromConfig extracts owner and repo from config
func parseRepoFromConfig(cfg *config.Config) (string, string) {
	if len(cfg.Repositories) > 0 {
		parts := strings.Split(cfg.Repositories[0], "/")
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	return "", ""
}

// renderHistoryScreen outputs styled history to terminal
func renderHistoryScreen(commits []CommitInfo, targetPath string, showFiles bool) {
	// Header
	header := historyHeaderStyle.Render(fmt.Sprintf(" %s ", targetPath))
	commitCount := fmt.Sprintf("%d commits", len(commits))

	fmt.Printf("\n%s  %s\n\n", header, summaryStyle.Render(commitCount))

	// Count change types for summary
	typeCounts := make(map[string]int)

	// Commit entries
	for _, commit := range commits {
		typeCounts[commit.ChangeType]++

		// Hash and date
		hashStr := hashStyle.Render(commit.Hash)
		dateStr := dateStyle.Render(commit.Date.Format("2006-01-02"))
		authorStr := authorStyle.Render(truncate(commit.Author, 12))

		// Change type badge
		typeStyle, ok := changeTypeStyles[commit.ChangeType]
		if !ok {
			typeStyle = changeTypeStyles["Change"]
		}
		typeStr := typeStyle.Render(fmt.Sprintf("[%s]", commit.ChangeType))

		// Subject (truncated)
		subjectStr := truncate(commit.Subject, 45)

		// Issue references
		var refStrs []string
		for _, ref := range commit.References {
			refStrs = append(refStrs, issueRefStyle.Render(fmt.Sprintf("#%d", ref.Number)))
		}
		refStr := ""
		if len(refStrs) > 0 {
			refStr = " " + strings.Join(refStrs, " ")
		}

		// File count suffix
		fileCountStr := ""
		if commit.FileCount > 0 {
			fileCountStr = fileCountStyle.Render(fmt.Sprintf(" (%d files)", commit.FileCount))
		}

		fmt.Printf("  %s  %s  %-12s  %s %s%s%s\n",
			hashStr, dateStr, authorStr, typeStr, subjectStr, refStr, fileCountStr)

		// Show file list if --files flag is used
		if showFiles && len(commit.Files) > 0 {
			for _, file := range commit.Files {
				fmt.Printf("      %s\n", filePathStyle.Render(file))
			}
			fmt.Println()
		}
	}

	// Summary line
	fmt.Println()
	var summaryParts []string
	typeOrder := []string{"Fix", "Add", "Update", "Remove", "Refactor", "Docs", "Test", "Chore", "Change"}
	for _, t := range typeOrder {
		if count, ok := typeCounts[t]; ok && count > 0 {
			style := changeTypeStyles[t]
			summaryParts = append(summaryParts, style.Render(fmt.Sprintf("%s: %d", t, count)))
		}
	}
	if len(summaryParts) > 0 {
		fmt.Printf("  %s\n\n", strings.Join(summaryParts, " | "))
	}
}

// outputHistoryJSON outputs commits as JSON
func outputHistoryJSON(commits []CommitInfo) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(commits)
}

// outputMarkdown writes history to a markdown file in History/ directory
func outputMarkdown(commits []CommitInfo, targetPath, repoOwner, repoName string) error {
	// Create History/ directory
	historyDir := "History"
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return fmt.Errorf("failed to create History directory: %w", err)
	}

	// Generate filename from path
	filename := strings.ReplaceAll(targetPath, "/", "-")
	filename = strings.ReplaceAll(filename, "\\", "-")
	filename = strings.ReplaceAll(filename, ".", "-")
	filename = strings.ReplaceAll(filename, ", ", "_")
	filename = strings.Trim(filename, "-")
	if filename == "" {
		filename = "history"
	}
	filename = filename + ".md"
	fullPath := filepath.Join(historyDir, filename)

	// Generate markdown content
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# History: %s\n\n", targetPath))
	b.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	b.WriteString("---\n\n")

	// Summary counts
	typeCounts := make(map[string]int)
	for _, commit := range commits {
		typeCounts[commit.ChangeType]++
	}

	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("**Total Commits:** %d\n\n", len(commits)))
	typeOrder := []string{"Fix", "Add", "Update", "Remove", "Refactor", "Docs", "Test", "Chore", "Change"}
	for _, t := range typeOrder {
		if count, ok := typeCounts[t]; ok && count > 0 {
			b.WriteString(fmt.Sprintf("- **%s:** %d\n", t, count))
		}
	}
	b.WriteString("\n---\n\n")

	// Commits table
	b.WriteString("## Commits\n\n")
	b.WriteString("| Commit | Date | Author | Type | Message | Issues |\n")
	b.WriteString("|--------|------|--------|------|---------|--------|\n")

	for _, commit := range commits {
		// Issue references as links
		var issueLinks []string
		for _, ref := range commit.References {
			issueLinks = append(issueLinks, fmt.Sprintf("[#%d](%s)", ref.Number, ref.URL))
		}
		issuesStr := "-"
		if len(issueLinks) > 0 {
			issuesStr = strings.Join(issueLinks, ", ")
		}

		// Escape pipe characters in subject
		subject := strings.ReplaceAll(commit.Subject, "|", "\\|")

		b.WriteString(fmt.Sprintf("| `%s` | %s | %s | %s | %s | %s |\n",
			commit.Hash,
			commit.Date.Format("2006-01-02"),
			commit.Author,
			commit.ChangeType,
			subject,
			issuesStr,
		))
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(b.String()), 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	fmt.Printf("✓ History written to %s\n", fullPath)
	return nil
}

// truncate shortens a string to maxLen, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// isDirectory checks if a path is a directory
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		// If path doesn't exist locally, check if it ends with /
		return strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\")
	}
	return info.IsDir()
}

// relativeTime returns a human-readable relative time string
func relativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	case diff < 30*24*time.Hour:
		weeks := int(diff.Hours() / 24 / 7)
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	case diff < 365*24*time.Hour:
		months := int(diff.Hours() / 24 / 30)
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	default:
		years := int(diff.Hours() / 24 / 365)
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}
}

// getCommitStats returns insertions and deletions for a commit on specific paths
func getCommitStats(hash string, paths []string) (int, int) {
	args := []string{"show", "--stat", "--format=", hash, "--"}
	args = append(args, paths...)

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}

	// Parse the last line which contains: " N files changed, X insertions(+), Y deletions(-)"
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return 0, 0
	}

	lastLine := lines[len(lines)-1]

	var insertions, deletions int

	// Match insertions
	insMatch := regexp.MustCompile(`(\d+) insertion`).FindStringSubmatch(lastLine)
	if len(insMatch) > 1 {
		insertions, _ = strconv.Atoi(insMatch[1])
	}

	// Match deletions
	delMatch := regexp.MustCompile(`(\d+) deletion`).FindStringSubmatch(lastLine)
	if len(delMatch) > 1 {
		deletions, _ = strconv.Atoi(delMatch[1])
	}

	return insertions, deletions
}

// getCommitBody retrieves the full commit message body (excluding subject)
func getCommitBody(hash string) string {
	cmd := exec.Command("git", "log", "-1", "--format=%b", hash)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	body := strings.TrimSpace(string(output))
	// Remove common trailer lines (Co-Authored-By, Generated with, etc.)
	lines := strings.Split(body, "\n")
	var cleanLines []string
	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "co-authored-by:") ||
			strings.Contains(lower, "generated with") ||
			strings.HasPrefix(lower, "signed-off-by:") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}
	return strings.TrimSpace(strings.Join(cleanLines, "\n"))
}

// getCommitComments fetches GitHub comments on a commit via gh api
func getCommitComments(hash, owner, repo string) []CommitComment {
	if owner == "" || repo == "" {
		return nil
	}

	// Use gh api to fetch commit comments
	endpoint := fmt.Sprintf("repos/%s/%s/commits/%s/comments", owner, repo, hash)
	cmd := exec.Command("gh", "api", endpoint)
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// Parse JSON response
	var apiComments []struct {
		User struct {
			Login string `json:"login"`
		} `json:"user"`
		Body      string `json:"body"`
		CreatedAt string `json:"created_at"`
	}

	if err := json.Unmarshal(output, &apiComments); err != nil {
		return nil
	}

	var comments []CommitComment
	for _, c := range apiComments {
		date, _ := time.Parse(time.RFC3339, c.CreatedAt)
		comments = append(comments, CommitComment{
			Author: c.User.Login,
			Body:   c.Body,
			Date:   date,
		})
	}

	return comments
}

// getCommitFiles returns the list of files changed in a commit, filtered by paths
func getCommitFiles(hash string, paths []string) ([]string, int) {
	args := []string{"show", "--name-only", "--format=", hash}
	if len(paths) > 0 && !(len(paths) == 1 && paths[0] == ".") {
		args = append(args, "--")
		args = append(args, paths...)
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, 0
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}

	return files, len(files)
}

// Additional styles for detailed view
var (
	detailBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1).
			MarginBottom(1)

	detailHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("63")).
				Padding(0, 1).
				MarginBottom(1)

	separatorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	insertStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	deleteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	urlStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")).
			Underline(true)

	relTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)

	bodyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	commentStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229"))

	commentAuthorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("117")).
				Bold(true)

	filePathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	fileCountStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)
)

// renderDetailedHistoryScreen outputs detailed history for single file
func renderDetailedHistoryScreen(commits []CommitInfo, targetPath string) {
	// Header box
	headerContent := fmt.Sprintf("%s\n%d commits", targetPath, len(commits))
	if len(commits) > 0 {
		headerContent = fmt.Sprintf("%s\n%d commits | Last modified: %s",
			targetPath, len(commits), commits[0].Date.Format("2006-01-02"))
	}

	fmt.Println()
	fmt.Println(detailBoxStyle.Render(detailHeaderStyle.Render(headerContent)))
	fmt.Println()

	separator := separatorStyle.Render(strings.Repeat("─", 60))

	for i, commit := range commits {
		// Header line: hash | date | author
		hashStr := hashStyle.Render(commit.Hash)
		dateStr := dateStyle.Render(commit.Date.Format("2006-01-02"))
		authorStr := authorStyle.Render(commit.Author)
		relTime := relTimeStyle.Render(fmt.Sprintf("(%s)", relativeTime(commit.Date)))

		fmt.Printf("%s | %s | %s %s\n", hashStr, dateStr, authorStr, relTime)
		fmt.Println(separator)

		// Full subject (not truncated)
		fmt.Printf("%s\n", commit.Subject)

		// Commit body (if present)
		if commit.Body != "" {
			fmt.Printf("\n%s\n", bodyStyle.Render(commit.Body))
		}
		fmt.Println()

		// Change type
		typeStyle, ok := changeTypeStyles[commit.ChangeType]
		if !ok {
			typeStyle = changeTypeStyles["Change"]
		}
		fmt.Printf("Type: %s\n", typeStyle.Render(commit.ChangeType))

		// Issue references with full URLs
		if len(commit.References) > 0 {
			for _, ref := range commit.References {
				fmt.Printf("Issue: %s %s\n",
					issueRefStyle.Render(fmt.Sprintf("#%d", ref.Number)),
					urlStyle.Render(fmt.Sprintf("(%s)", ref.URL)))
			}
		}

		// Line stats
		if commit.Insertions > 0 || commit.Deletions > 0 {
			insStr := insertStyle.Render(fmt.Sprintf("+%d", commit.Insertions))
			delStr := deleteStyle.Render(fmt.Sprintf("-%d", commit.Deletions))
			fmt.Printf("Lines: %s %s\n", insStr, delStr)
		}

		// GitHub commit comments
		if len(commit.Comments) > 0 {
			fmt.Printf("\n💬 Comments (%d):\n", len(commit.Comments))
			for _, comment := range commit.Comments {
				authorStr := commentAuthorStyle.Render("@" + comment.Author)
				dateStr := relTimeStyle.Render(fmt.Sprintf("(%s)", relativeTime(comment.Date)))
				fmt.Printf("  %s %s:\n", authorStr, dateStr)
				// Indent comment body
				for _, line := range strings.Split(comment.Body, "\n") {
					fmt.Printf("    %s\n", commentStyle.Render(line))
				}
			}
		}

		// Separator between commits (except last)
		if i < len(commits)-1 {
			fmt.Printf("\n%s\n\n", separator)
		} else {
			fmt.Println()
		}
	}

	// Summary
	typeCounts := make(map[string]int)
	totalIns, totalDel := 0, 0
	for _, commit := range commits {
		typeCounts[commit.ChangeType]++
		totalIns += commit.Insertions
		totalDel += commit.Deletions
	}

	var summaryParts []string
	typeOrder := []string{"Fix", "Add", "Update", "Remove", "Refactor", "Docs", "Test", "Chore", "Change"}
	for _, t := range typeOrder {
		if count, ok := typeCounts[t]; ok && count > 0 {
			style := changeTypeStyles[t]
			summaryParts = append(summaryParts, style.Render(fmt.Sprintf("%s: %d", t, count)))
		}
	}

	fmt.Println(separator)
	if len(summaryParts) > 0 {
		fmt.Printf("Summary: %s\n", strings.Join(summaryParts, " | "))
	}
	if totalIns > 0 || totalDel > 0 {
		fmt.Printf("Total changes: %s %s\n",
			insertStyle.Render(fmt.Sprintf("+%d", totalIns)),
			deleteStyle.Render(fmt.Sprintf("-%d", totalDel)))
	}
	fmt.Println()
}

// openHistoryInBrowser generates HTML and opens it in the default browser
func openHistoryInBrowser(commits []CommitInfo, targetPath, repoOwner, repoName string) error {
	htmlContent := generateHistoryHTML(commits, targetPath, repoOwner, repoName)

	// Create temp file in project tmp directory
	tmpFile, err := config.CreateTempFile("gh-pmu-history-*.html")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(htmlContent); err != nil {
		return fmt.Errorf("failed to write HTML: %w", err)
	}

	filePath := tmpFile.Name()

	// Open in browser based on OS
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	default: // linux, etc.
		cmd = exec.Command("xdg-open", filePath)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	fmt.Printf("Opened history in browser: %s\n", filePath)
	return nil
}

// generateHistoryHTML creates an HTML document for the commit history
func generateHistoryHTML(commits []CommitInfo, targetPath, repoOwner, repoName string) string {
	var sb strings.Builder

	// HTML header with styles
	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <title>History: ` + html.EscapeString(targetPath) + `</title>
  <style>
    * { box-sizing: border-box; }
    body {
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
      max-width: 900px;
      margin: 0 auto;
      padding: 20px;
      background: #0d1117;
      color: #c9d1d9;
    }
    h1 { color: #58a6ff; border-bottom: 1px solid #30363d; padding-bottom: 10px; }
    .summary { color: #8b949e; margin-bottom: 20px; }
    .commit {
      border: 1px solid #30363d;
      border-radius: 6px;
      margin-bottom: 16px;
      background: #161b22;
    }
    .commit-header {
      padding: 12px 16px;
      border-bottom: 1px solid #30363d;
      background: #21262d;
      border-radius: 6px 6px 0 0;
    }
    .commit-body { padding: 16px; }
    .hash {
      font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, monospace;
      color: #58a6ff;
      text-decoration: none;
    }
    .hash:hover { text-decoration: underline; }
    .meta { color: #8b949e; font-size: 14px; }
    .author { color: #58a6ff; }
    .subject { font-weight: 600; color: #c9d1d9; margin: 8px 0; }
    .body-text {
      color: #8b949e;
      white-space: pre-wrap;
      font-size: 14px;
      margin: 12px 0;
      padding: 12px;
      background: #0d1117;
      border-radius: 6px;
    }
    .type {
      display: inline-block;
      padding: 2px 8px;
      border-radius: 3px;
      font-size: 12px;
      font-weight: 600;
    }
    .type-Fix { background: #f85149; color: #fff; }
    .type-Add { background: #238636; color: #fff; }
    .type-Update { background: #1f6feb; color: #fff; }
    .type-Remove { background: #da3633; color: #fff; }
    .type-Refactor { background: #8957e5; color: #fff; }
    .type-Docs { background: #388bfd; color: #fff; }
    .type-Test { background: #3fb950; color: #fff; }
    .type-Chore { background: #6e7681; color: #fff; }
    .type-Change { background: #6e7681; color: #fff; }
    .stats {
      font-family: ui-monospace, SFMono-Regular, monospace;
      font-size: 14px;
    }
    .ins { color: #3fb950; }
    .del { color: #f85149; }
    .issue-link { color: #58a6ff; text-decoration: none; }
    .issue-link:hover { text-decoration: underline; }
    .details { margin-top: 8px; }
    .detail-row { margin: 4px 0; font-size: 14px; }
    .comment {
      margin: 12px 0;
      padding: 12px;
      background: #0d1117;
      border-left: 3px solid #388bfd;
      border-radius: 0 6px 6px 0;
    }
    .comment-author { color: #58a6ff; font-weight: 600; }
    .comment-date { color: #6e7681; font-size: 12px; }
    .comment-body { margin-top: 8px; white-space: pre-wrap; }
    .totals {
      margin-top: 20px;
      padding: 16px;
      background: #161b22;
      border: 1px solid #30363d;
      border-radius: 6px;
    }
  </style>
</head>
<body>
`)

	// Header
	sb.WriteString(fmt.Sprintf("  <h1>📜 History: %s</h1>\n", html.EscapeString(targetPath)))
	sb.WriteString(fmt.Sprintf("  <p class=\"summary\">%d commits</p>\n\n", len(commits)))

	// Count types and totals for summary
	typeCounts := make(map[string]int)
	totalIns, totalDel := 0, 0

	// Commits
	for _, commit := range commits {
		typeCounts[commit.ChangeType]++
		totalIns += commit.Insertions
		totalDel += commit.Deletions

		sb.WriteString("  <div class=\"commit\">\n")
		sb.WriteString("    <div class=\"commit-header\">\n")

		// Hash with link to GitHub
		commitURL := fmt.Sprintf("https://github.com/%s/%s/commit/%s", repoOwner, repoName, commit.Hash)
		sb.WriteString(fmt.Sprintf("      <a href=\"%s\" class=\"hash\" target=\"_blank\">%s</a>\n",
			commitURL, commit.Hash))

		// Meta info
		sb.WriteString(fmt.Sprintf("      <span class=\"meta\"> | %s | <span class=\"author\">%s</span> | %s</span>\n",
			commit.Date.Format("2006-01-02"),
			html.EscapeString(commit.Author),
			relativeTime(commit.Date)))

		sb.WriteString("    </div>\n")
		sb.WriteString("    <div class=\"commit-body\">\n")

		// Subject
		sb.WriteString(fmt.Sprintf("      <div class=\"subject\">%s</div>\n", html.EscapeString(commit.Subject)))

		// Body
		if commit.Body != "" {
			sb.WriteString(fmt.Sprintf("      <div class=\"body-text\">%s</div>\n", html.EscapeString(commit.Body)))
		}

		// Details row
		sb.WriteString("      <div class=\"details\">\n")

		// Type badge
		sb.WriteString(fmt.Sprintf("        <div class=\"detail-row\"><span class=\"type type-%s\">%s</span></div>\n",
			commit.ChangeType, commit.ChangeType))

		// Issue references
		for _, ref := range commit.References {
			sb.WriteString(fmt.Sprintf("        <div class=\"detail-row\">Issue: <a href=\"%s\" class=\"issue-link\" target=\"_blank\">#%d</a></div>\n",
				ref.URL, ref.Number))
		}

		// Stats
		if commit.Insertions > 0 || commit.Deletions > 0 {
			sb.WriteString(fmt.Sprintf("        <div class=\"detail-row stats\"><span class=\"ins\">+%d</span> <span class=\"del\">-%d</span></div>\n",
				commit.Insertions, commit.Deletions))
		}

		sb.WriteString("      </div>\n")

		// Comments
		if len(commit.Comments) > 0 {
			for _, comment := range commit.Comments {
				sb.WriteString("      <div class=\"comment\">\n")
				sb.WriteString(fmt.Sprintf("        <span class=\"comment-author\">@%s</span> <span class=\"comment-date\">%s</span>\n",
					html.EscapeString(comment.Author), relativeTime(comment.Date)))
				sb.WriteString(fmt.Sprintf("        <div class=\"comment-body\">%s</div>\n", html.EscapeString(comment.Body)))
				sb.WriteString("      </div>\n")
			}
		}

		sb.WriteString("    </div>\n")
		sb.WriteString("  </div>\n\n")
	}

	// Summary totals
	sb.WriteString("  <div class=\"totals\">\n")
	sb.WriteString("    <strong>Summary:</strong> ")
	typeOrder := []string{"Fix", "Add", "Update", "Remove", "Refactor", "Docs", "Test", "Chore", "Change"}
	var parts []string
	for _, t := range typeOrder {
		if count, ok := typeCounts[t]; ok && count > 0 {
			parts = append(parts, fmt.Sprintf("<span class=\"type type-%s\">%s: %d</span>", t, t, count))
		}
	}
	sb.WriteString(strings.Join(parts, " "))
	if totalIns > 0 || totalDel > 0 {
		sb.WriteString(fmt.Sprintf("<br><span class=\"stats\">Total changes: <span class=\"ins\">+%d</span> <span class=\"del\">-%d</span></span>",
			totalIns, totalDel))
	}
	sb.WriteString("\n  </div>\n")

	sb.WriteString("</body>\n</html>")

	return sb.String()
}
