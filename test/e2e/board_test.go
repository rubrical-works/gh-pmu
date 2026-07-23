//go:build e2e

package e2e

import (
	"fmt"
	"testing"
)

// TestBoardRendering tests that the board command displays issues correctly
// across different status columns.
func TestBoardRendering(t *testing.T) {
	cfg := setupTestConfig(t)

	// Setup test branch (required for IDPF validation when moving to in_progress)
	branchInfo, branchCleanup := setupTestBranch(t, cfg)
	defer branchCleanup()

	// Track issues for cleanup
	var issueNums []int

	// Cleanup at end of test
	defer func() {
		for _, num := range issueNums {
			runCleanupAfterTest(t, num)
		}
	}()

	// Step 1: Create issues in different statuses
	t.Run("create test issues", func(t *testing.T) {
		// Create issue in Backlog
		backlogIssue := createTestIssue(t, cfg, "Board Test - Backlog")
		issueNums = append(issueNums, backlogIssue)

		// Create issue, assign to branch, and move to In Progress
		inProgressIssue := createTestIssue(t, cfg, "Board Test - In Progress")
		issueNums = append(issueNums, inProgressIssue)
		assignIssueToBranch(t, cfg, inProgressIssue)
		result := runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", inProgressIssue), "--status", "in_progress")
		assertExitCode(t, result, 0)

		// Create issue, assign to branch, add body, and move to Done
		doneIssue := createTestIssue(t, cfg, "Board Test - Done")
		issueNums = append(issueNums, doneIssue)
		assignIssueToBranch(t, cfg, doneIssue)
		// Add body (required for IDPF validation when moving to done)
		runPMU(t, cfg.Dir, "edit", fmt.Sprintf("%d", doneIssue), "--body", "Test issue body for done status")
		result = runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", doneIssue), "--status", "done")
		assertExitCode(t, result, 0)

		// Log branch info for debugging
		t.Logf("Using test branch: %s (tracker #%d)", branchInfo.Name, branchInfo.TrackerIssue)
	})

	// Step 2: Run board command
	var boardOutput string
	t.Run("run board command", func(t *testing.T) {
		result := runPMU(t, cfg.Dir, "board")
		assertExitCode(t, result, 0)
		boardOutput = result.Stdout
	})

	// Step 3: Verify all status column headers appear in output
	t.Run("verify status columns", func(t *testing.T) {
		// Check for key status column headers
		assertContains(t, boardOutput, "Backlog")
		assertContains(t, boardOutput, "In progress")
		assertContains(t, boardOutput, "Done")
	})

	// Step 4: Verify issues appear under correct columns
	t.Run("verify issues in board", func(t *testing.T) {
		// Verify our test issues appear in the board output
		// We check for issue numbers and key content, not exact formatting
		for _, num := range issueNums {
			assertContains(t, boardOutput, fmt.Sprintf("#%d", num))
		}

		// Verify the issue titles contain our test markers
		// Note: Board truncates titles, so we only check for [E2E] prefix
		assertContains(t, boardOutput, "[E2E]")
	})
}

// TestBoardWithFilter tests that the board command respects status filters.
func TestBoardWithFilter(t *testing.T) {
	cfg := setupTestConfig(t)

	// A branch is required to move the contrast issue to in_progress (IDPF validation).
	_, branchCleanup := setupTestBranch(t, cfg)
	defer branchCleanup()

	// Track issues for cleanup
	var issueNums []int

	// Cleanup at end of test
	defer func() {
		for _, num := range issueNums {
			runCleanupAfterTest(t, num)
		}
	}()

	// Create an issue in backlog — it must appear under `board --status backlog`.
	backlogIssue := createTestIssue(t, cfg, "Board Filter Test - Backlog")
	issueNums = append(issueNums, backlogIssue)

	// Create a contrast issue in a different column (in_progress). It must be ABSENT
	// from the backlog-filtered board. Without this contrast, an inclusion-only check
	// passes even if --status is silently ignored and every column is rendered.
	inProgressIssue := createTestIssue(t, cfg, "Board Filter Test - In Progress")
	issueNums = append(issueNums, inProgressIssue)
	assignIssueToBranch(t, cfg, inProgressIssue)
	moveResult := runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", inProgressIssue), "--status", "in_progress")
	assertExitCode(t, moveResult, 0)

	// Run board with status filter - use retry for eventual consistency
	// GitHub API may not immediately index the new issue in project queries
	result := waitForProjectSync(t, cfg, 5,
		[]string{"board", "--status", "backlog"},
		fmt.Sprintf("#%d", backlogIssue),
	)
	assertExitCode(t, result, 0)

	// Verify the backlog issue appears...
	assertContains(t, result.Stdout, fmt.Sprintf("#%d", backlogIssue))
	// ...and the in_progress contrast issue does NOT (proves --status actually filters).
	assertNotContains(t, result.Stdout, fmt.Sprintf("#%d", inProgressIssue))
}
