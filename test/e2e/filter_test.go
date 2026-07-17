//go:build e2e

package e2e

import (
	"fmt"
	"testing"
	"time"
)

// TestFilterByStatus tests filtering issues by status field.
func TestFilterByStatus(t *testing.T) {
	cfg := setupTestConfig(t)

	// Setup test branch (required for IDPF validation when moving to in_progress)
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

	// Step 1: Create issues with different status values
	t.Run("create test issues", func(t *testing.T) {
		// Create issue in Backlog (default)
		backlogIssue := createTestIssue(t, cfg, "Filter Test - Backlog")
		issueNums = append(issueNums, backlogIssue)

		// Create issue, assign to branch, and move to In Progress
		inProgressIssue := createTestIssue(t, cfg, "Filter Test - In Progress")
		issueNums = append(issueNums, inProgressIssue)
		assignIssueToBranch(t, cfg, inProgressIssue)
		result := runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", inProgressIssue), "--status", "in_progress")
		assertExitCode(t, result, 0)
	})

	// Step 2: Run filter command for backlog status
	t.Run("filter by backlog status", func(t *testing.T) {
		// Use retry for eventual consistency
		result := waitForProjectSync(t, cfg, 5,
			[]string{"list", "--status", "backlog"},
			fmt.Sprintf("#%d", issueNums[0]),
		)
		assertExitCode(t, result, 0)

		// Verify backlog issue appears
		assertContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[0]))

		// Verify in_progress issue does NOT appear
		assertNotContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[1]))
	})

	// Step 3: Run filter command for in_progress status
	t.Run("filter by in_progress status", func(t *testing.T) {
		// Use retry for eventual consistency
		result := waitForProjectSync(t, cfg, 5,
			[]string{"list", "--status", "in_progress"},
			fmt.Sprintf("#%d", issueNums[1]),
		)
		assertExitCode(t, result, 0)

		// Verify in_progress issue appears
		assertContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[1]))

		// Verify backlog issue does NOT appear
		assertNotContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[0]))
	})
}

// TestFilterByPriority tests filtering issues by priority field.
func TestFilterByPriority(t *testing.T) {
	cfg := setupTestConfig(t)

	// Track issues for cleanup
	var issueNums []int

	// Cleanup at end of test
	defer func() {
		for _, num := range issueNums {
			runCleanupAfterTest(t, num)
		}
	}()

	// Step 1: Create issues with different priority values
	t.Run("create test issues", func(t *testing.T) {
		// Create P2 priority issue (default)
		p2Issue := createTestIssue(t, cfg, "Filter Test - P2")
		issueNums = append(issueNums, p2Issue)

		// Create P0 priority issue
		p0Issue := createTestIssue(t, cfg, "Filter Test - P0")
		issueNums = append(issueNums, p0Issue)
		result := runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", p0Issue), "--priority", "p0")
		assertExitCode(t, result, 0)
	})

	// Step 2: Run filter command for P0 priority
	t.Run("filter by P0 priority", func(t *testing.T) {
		// Use retry for eventual consistency
		result := waitForProjectSync(t, cfg, 5,
			[]string{"list", "--priority", "p0"},
			fmt.Sprintf("#%d", issueNums[1]),
		)
		assertExitCode(t, result, 0)

		// Verify P0 issue appears...
		assertContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[1]))
		// ...and the P2 contrast issue does NOT (a no-op --priority would leak it).
		assertNotContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[0]))
	})

	// Step 3: Run filter command for P2 priority
	t.Run("filter by P2 priority", func(t *testing.T) {
		// Use retry for eventual consistency
		result := waitForProjectSync(t, cfg, 5,
			[]string{"list", "--priority", "p2"},
			fmt.Sprintf("#%d", issueNums[0]),
		)
		assertExitCode(t, result, 0)

		// Verify P2 issue appears...
		assertContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[0]))
		// ...and the P0 contrast issue does NOT.
		assertNotContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[1]))
	})
}

// TestFilterCombined tests filtering with multiple criteria.
func TestFilterCombined(t *testing.T) {
	cfg := setupTestConfig(t)

	// Setup test branch (required for IDPF validation when moving to in_progress)
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

	// Matching issue: in_progress AND p1 — must appear.
	issue := createTestIssue(t, cfg, "Filter Combined Test - Match")
	issueNums = append(issueNums, issue)
	assignIssueToBranch(t, cfg, issue)
	result := runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", issue), "--status", "in_progress", "--priority", "p1")
	assertExitCode(t, result, 0)

	// Contrast A: matches status ONLY (in_progress, but stays p2). Its absence proves
	// the priority half of the AND is applied. createTestIssue defaults to p2.
	statusOnly := createTestIssue(t, cfg, "Filter Combined Test - Status Only")
	issueNums = append(issueNums, statusOnly)
	assignIssueToBranch(t, cfg, statusOnly)
	result = runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", statusOnly), "--status", "in_progress")
	assertExitCode(t, result, 0)

	// Contrast B: matches priority ONLY (p1, but stays in backlog). Its absence proves
	// the status half of the AND is applied. No branch needed for a priority-only move.
	priorityOnly := createTestIssue(t, cfg, "Filter Combined Test - Priority Only")
	issueNums = append(issueNums, priorityOnly)
	result = runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", priorityOnly), "--priority", "p1")
	assertExitCode(t, result, 0)

	// Filter by both status and priority - use retry for eventual consistency
	result = waitForProjectSync(t, cfg, 5,
		[]string{"list", "--status", "in_progress", "--priority", "p1"},
		fmt.Sprintf("#%d", issue),
	)
	assertExitCode(t, result, 0)

	// AND semantics: the matching issue is present, single-criterion contrasts absent.
	assertContains(t, result.Stdout, fmt.Sprintf("#%d", issue))
	assertNotContains(t, result.Stdout, fmt.Sprintf("#%d", statusOnly))
	assertNotContains(t, result.Stdout, fmt.Sprintf("#%d", priorityOnly))
}

// TestFilterByBranch tests filtering issues by branch field using --branch flag.
func TestFilterByBranch(t *testing.T) {
	cfg := setupTestConfig(t)

	// Generate unique branch name
	branchName := fmt.Sprintf("release/e2e-filter-%d", time.Now().UnixNano())

	// Track resources for cleanup
	var issueNums []int
	var trackerIssueNum int

	// Cleanup at end of test
	defer func() {
		for _, num := range issueNums {
			runCleanupAfterTest(t, num)
		}
		if trackerIssueNum > 0 {
			runCleanupAfterTest(t, trackerIssueNum)
		}
		// Ensure branch is closed
		runPMU(t, cfg.Dir, "branch", "close", "--yes")
	}()

	// Step 1: Start a branch to assign issues to
	t.Run("start branch", func(t *testing.T) {
		result := runPMU(t, cfg.Dir, "branch", "start", "--name", branchName)
		assertExitCode(t, result, 0)
		trackerIssueNum = extractIssueNumber(t, result.Stdout)
	})

	// Step 2: Create two issues - one assigned to branch, one not
	t.Run("create test issues", func(t *testing.T) {
		// Create issue that will be assigned to branch
		branchIssue := createTestIssue(t, cfg, "Filter Test - With Branch")
		issueNums = append(issueNums, branchIssue)

		// Create issue that will NOT be assigned to branch
		noBranchIssue := createTestIssue(t, cfg, "Filter Test - No Branch")
		issueNums = append(issueNums, noBranchIssue)

		// Assign first issue to branch
		result := runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", branchIssue), "--branch", "current")
		assertExitCode(t, result, 0)
	})

	// Step 3: Filter by branch - should show only assigned issue
	t.Run("filter by branch value", func(t *testing.T) {
		result := waitForProjectSync(t, cfg, 5,
			[]string{"list", "--branch", branchName},
			fmt.Sprintf("#%d", issueNums[0]),
		)
		assertExitCode(t, result, 0)

		// Verify branch-assigned issue appears
		assertContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[0]))

		// Verify non-branch issue does NOT appear
		assertNotContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[1]))
	})

	// Step 4: Filter by branch using "current" keyword
	t.Run("filter by branch current", func(t *testing.T) {
		result := waitForProjectSync(t, cfg, 5,
			[]string{"list", "--branch", "current"},
			fmt.Sprintf("#%d", issueNums[0]),
		)
		assertExitCode(t, result, 0)

		// Verify branch-assigned issue appears...
		assertContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[0]))
		// ...and the unassigned issue does NOT (a no-op --branch current would leak it).
		assertNotContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[1]))
	})
}

// TestFilterByNoBranch tests filtering issues without branch assignment using --no-branch flag.
func TestFilterByNoBranch(t *testing.T) {
	cfg := setupTestConfig(t)

	// Generate unique branch name
	branchName := fmt.Sprintf("release/e2e-nobranch-%d", time.Now().UnixNano())

	// Track resources for cleanup
	var issueNums []int
	var trackerIssueNum int

	// Cleanup at end of test
	defer func() {
		for _, num := range issueNums {
			runCleanupAfterTest(t, num)
		}
		if trackerIssueNum > 0 {
			runCleanupAfterTest(t, trackerIssueNum)
		}
		// Ensure branch is closed
		runPMU(t, cfg.Dir, "branch", "close", "--yes")
	}()

	// Step 1: Start a branch
	t.Run("start branch", func(t *testing.T) {
		result := runPMU(t, cfg.Dir, "branch", "start", "--name", branchName)
		assertExitCode(t, result, 0)
		trackerIssueNum = extractIssueNumber(t, result.Stdout)
	})

	// Step 2: Create two issues - one assigned to branch, one not
	t.Run("create test issues", func(t *testing.T) {
		// Create issue that will be assigned to branch
		branchIssue := createTestIssue(t, cfg, "No-Branch Filter Test - With Branch")
		issueNums = append(issueNums, branchIssue)

		// Create issue that will NOT be assigned to branch
		noBranchIssue := createTestIssue(t, cfg, "No-Branch Filter Test - No Branch")
		issueNums = append(issueNums, noBranchIssue)

		// Assign first issue to branch
		result := runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", branchIssue), "--branch", "current")
		assertExitCode(t, result, 0)
	})

	// Step 3: Filter by --no-branch - should show only unassigned issue
	t.Run("filter by no-branch", func(t *testing.T) {
		result := waitForProjectSync(t, cfg, 5,
			[]string{"list", "--no-branch"},
			fmt.Sprintf("#%d", issueNums[1]),
		)
		assertExitCode(t, result, 0)

		// Verify non-branch issue appears
		assertContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[1]))

		// Verify branch-assigned issue does NOT appear
		assertNotContains(t, result.Stdout, fmt.Sprintf("#%d", issueNums[0]))
	})
}
