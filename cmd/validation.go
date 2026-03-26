package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rubrical-works/gh-pmu/internal/api"
	"github.com/rubrical-works/gh-pmu/internal/config"
)

// Regex patterns for checkbox detection
var (
	checkedBoxRegex    = regexp.MustCompile(`- \[x\]`)
	uncheckedBoxRegex  = regexp.MustCompile(`- \[ \]`)
	uncheckedItemRegex = regexp.MustCompile(`- \[ \] (.+)`)
)

// ValidationError represents a validation failure with actionable message
type ValidationError struct {
	IssueNumber int
	Message     string
	Suggestion  string
}

func (e *ValidationError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("Issue #%d: %s\n\n%s", e.IssueNumber, e.Message, e.Suggestion)
	}
	return fmt.Sprintf("Issue #%d: %s", e.IssueNumber, e.Message)
}

// ValidationErrors collects multiple validation failures
type ValidationErrors struct {
	Errors []ValidationError
}

func (e *ValidationErrors) Error() string {
	if len(e.Errors) == 0 {
		return ""
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Validation failed for %d issues:\n", len(e.Errors)))
	for _, err := range e.Errors {
		sb.WriteString(fmt.Sprintf("\n  - Issue #%d: %s", err.IssueNumber, err.Message))
	}
	return sb.String()
}

func (e *ValidationErrors) Add(err ValidationError) {
	e.Errors = append(e.Errors, err)
}

func (e *ValidationErrors) HasErrors() bool {
	return len(e.Errors) > 0
}

// issueValidationContext holds all info needed to validate an issue
type issueValidationContext struct {
	Number         int
	CurrentStatus  string
	CurrentRelease string
	Body           string
	ActiveReleases []string // Discovered from GitHub release tracker issues
}

// validateStatusTransition checks IDPF rules for a status transition
// Set force=true to bypass checkbox validation (but NOT body or release requirements)
func validateStatusTransition(cfg *config.Config, ctx *issueValidationContext, targetStatus, targetRelease string, force bool) *ValidationError {
	// Skip validation if not using IDPF
	if !cfg.IsIDPF() {
		return nil
	}

	// Normalize status values for comparison
	currentStatus := strings.ToLower(ctx.CurrentStatus)
	targetStatusLower := strings.ToLower(targetStatus)

	// Rule 1: Body required for in_review/done (NOT bypassed by --force)
	if targetStatusLower == "in_review" || targetStatusLower == "in review" || targetStatusLower == "done" {
		if isBodyEmpty(ctx.Body) {
			return &ValidationError{
				IssueNumber: ctx.Number,
				Message:     fmt.Sprintf("Empty body. Cannot move to '%s' without issue content.", targetStatus),
				Suggestion:  fmt.Sprintf("Use: gh issue edit %d --body \"<description>\"", ctx.Number),
			}
		}
	}

	// Rule 2: All checkboxes must be checked for in_review/done (bypassed by --force)
	if targetStatusLower == "in_review" || targetStatusLower == "in review" || targetStatusLower == "done" {
		unchecked := countUncheckedBoxes(ctx.Body)
		if unchecked > 0 && !force {
			uncheckedItems := getUncheckedItems(ctx.Body)
			itemList := ""
			if len(uncheckedItems) > 0 {
				itemList = "\n" + strings.Join(uncheckedItems, "\n")
			}
			return &ValidationError{
				IssueNumber: ctx.Number,
				Message:     fmt.Sprintf("Has %d unchecked checkbox(es):%s", unchecked, itemList),
				Suggestion:  fmt.Sprintf("Complete these items before moving to %s, or use --force to bypass.\nClaude: Review GitHub-Workflow rules before using --force.", targetStatus),
			}
		}
	}

	// Rule 3: Release required for backlog → ready/in_progress
	if currentStatus == "backlog" && (targetStatusLower == "ready" || targetStatusLower == "in progress" || targetStatusLower == "in_progress") {
		// Check if release is being set or already set
		releaseValue := targetRelease
		if releaseValue == "" {
			releaseValue = ctx.CurrentRelease
		}

		if releaseValue == "" {
			return &ValidationError{
				IssueNumber: ctx.Number,
				Message:     fmt.Sprintf("No branch assignment. Cannot move from 'backlog' to '%s' without a branch.", targetStatus),
				Suggestion:  fmt.Sprintf("Use: gh pmu move %d --branch vX.Y.Z", ctx.Number),
			}
		}

		// Validate release exists in active releases (if we have discovered releases)
		if !isReleaseActiveInContext(ctx.ActiveReleases, releaseValue) {
			suggestion := "Use 'gh pmu branch start' to create a new branch."
			if len(ctx.ActiveReleases) > 0 {
				suggestion = fmt.Sprintf("Available branches: %s\n\n%s", strings.Join(ctx.ActiveReleases, ", "), suggestion)
			}
			return &ValidationError{
				IssueNumber: ctx.Number,
				Message:     fmt.Sprintf("Branch \"%s\" not found in active branches.", releaseValue),
				Suggestion:  suggestion,
			}
		}
	}

	return nil
}

// isReleaseActiveInContext checks if a release name exists in the discovered active releases
func isReleaseActiveInContext(activeReleases []string, releaseName string) bool {
	// If no active releases discovered, allow any release (backwards compatibility)
	if len(activeReleases) == 0 {
		return true
	}

	for _, active := range activeReleases {
		if strings.EqualFold(active, releaseName) {
			return true
		}
	}

	return false
}

// discoverActiveReleases fetches active branch names from GitHub issues with "branch" label
// Returns the extracted branch names (e.g., "release/v1.2.0") from issue titles like "Branch: release/v1.2.0"
// Supports both "Branch: " (new) and "Release: " (legacy) prefixes for backwards compatibility
func discoverActiveReleases(issues []api.Issue) []string {
	var releases []string
	for _, issue := range issues {
		var version string
		if strings.HasPrefix(issue.Title, "Branch: ") {
			// Extract version from title (e.g., "Branch: release/v1.2.0" or "Branch: release/v1.2.0 (Phoenix)")
			version = strings.TrimPrefix(issue.Title, "Branch: ")
		} else if strings.HasPrefix(issue.Title, "Release: ") {
			// Legacy format: "Release: v1.2.0" or "Release: v1.2.0 (Phoenix)"
			version = strings.TrimPrefix(issue.Title, "Release: ")
		} else {
			continue
		}
		// Remove codename in parentheses if present
		if idx := strings.Index(version, " ("); idx > 0 {
			version = version[:idx]
		}
		releases = append(releases, strings.TrimSpace(version))
	}
	return releases
}

// isBodyEmpty checks if the body is empty (empty string or whitespace only)
func isBodyEmpty(body string) bool {
	return strings.TrimSpace(body) == ""
}

// stripCodeBlocks removes content inside fenced code blocks (``` or ~~~) and
// indented code blocks (4 spaces or tab) from the body. This prevents example
// checkboxes in code blocks from being counted as acceptance criteria.
func stripCodeBlocks(body string) string {
	lines := strings.Split(body, "\n")
	var filteredLines []string
	inFencedCodeBlock := false
	fenceChar := ""
	inIndentedCodeBlock := false

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check for fenced code block start/end (``` or ~~~)
		if strings.HasPrefix(trimmedLine, "```") || strings.HasPrefix(trimmedLine, "~~~") {
			currentFence := trimmedLine[:3]
			if !inFencedCodeBlock {
				// Starting a fenced code block
				inFencedCodeBlock = true
				fenceChar = currentFence
				continue // Skip the opening fence line
			} else if currentFence == fenceChar {
				// Ending a fenced code block (matching fence)
				inFencedCodeBlock = false
				fenceChar = ""
				continue // Skip the closing fence line
			}
			// Different fence inside a code block - just skip it
			continue
		}

		// Skip content inside fenced code blocks
		if inFencedCodeBlock {
			continue
		}

		// Check for indented code block (4+ spaces or tab at start)
		// But not if it's a list item (- [ ] or - [x] pattern)
		isIndentedCode := (strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")) &&
			!strings.Contains(line, "- [ ]") && !strings.Contains(line, "- [x]")

		// Check if previous line was blank (indented code blocks are preceded by blank line)
		prevLineBlank := i == 0 || strings.TrimSpace(lines[i-1]) == ""

		if isIndentedCode && (inIndentedCodeBlock || prevLineBlank) {
			inIndentedCodeBlock = true
			continue // Skip this line (it's part of an indented code block)
		} else if trimmedLine == "" && inIndentedCodeBlock {
			// Blank line might end the code block, but keep checking
			continue
		} else {
			inIndentedCodeBlock = false
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}

// countUncheckedBoxes counts the number of unchecked checkboxes in the body,
// excluding checkboxes inside code blocks (which are examples, not criteria).
func countUncheckedBoxes(body string) int {
	strippedBody := stripCodeBlocks(body)
	return len(uncheckedBoxRegex.FindAllString(strippedBody, -1))
}

// countCheckedBoxes counts the number of checked checkboxes in the body,
// excluding checkboxes inside code blocks.
func countCheckedBoxes(body string) int {
	strippedBody := stripCodeBlocks(body)
	return len(checkedBoxRegex.FindAllString(strippedBody, -1))
}

// countCodeBlockCheckboxes counts checkboxes inside code blocks (for informational messages).
// Returns the count of both checked and unchecked checkboxes found in code blocks.
func countCodeBlockCheckboxes(body string) int {
	totalBefore := len(uncheckedBoxRegex.FindAllString(body, -1)) + len(checkedBoxRegex.FindAllString(body, -1))
	strippedBody := stripCodeBlocks(body)
	totalAfter := len(uncheckedBoxRegex.FindAllString(strippedBody, -1)) + len(checkedBoxRegex.FindAllString(strippedBody, -1))
	return totalBefore - totalAfter
}

// getUncheckedItems extracts the text of all unchecked checkbox items,
// excluding checkboxes inside code blocks.
func getUncheckedItems(body string) []string {
	strippedBody := stripCodeBlocks(body)
	matches := uncheckedItemRegex.FindAllStringSubmatch(strippedBody, -1)

	var items []string
	for _, match := range matches {
		if len(match) > 1 {
			items = append(items, "  [ ] "+strings.TrimSpace(match[1]))
		}
	}
	return items
}

// getFieldValueFromSlice extracts a field value from a slice of field values
func getFieldValueFromSlice(fieldValues []api.FieldValue, fieldName string) string {
	for _, fv := range fieldValues {
		if strings.EqualFold(fv.Field, fieldName) {
			return fv.Value
		}
	}
	return ""
}

// buildValidationContext creates a validation context from project item data
func buildValidationContext(number int, body string, fieldValues []api.FieldValue, activeReleases []string) *issueValidationContext {
	// Check both "Branch" (new) and "Release" (legacy) field names for backward compatibility
	currentBranch := getFieldValueFromSlice(fieldValues, BranchFieldName)
	if currentBranch == "" {
		currentBranch = getFieldValueFromSlice(fieldValues, LegacyReleaseFieldName)
	}
	return &issueValidationContext{
		Number:         number,
		CurrentStatus:  getFieldValueFromSlice(fieldValues, "Status"),
		CurrentRelease: currentBranch,
		Body:           body,
		ActiveReleases: activeReleases,
	}
}
