//go:build e2e

package e2e

import (
	"fmt"
	"testing"
	"time"
)

// TestBranchLifecycle tests the complete branch workflow:
// start -> current -> add issue -> list -> close
func TestBranchLifecycle(t *testing.T) {
	cfg := setupTestConfig(t)

	// Generate unique branch name with timestamp (format: release/e2e-test-{timestamp})
	branchName := fmt.Sprintf("release/e2e-test-%d", time.Now().UnixNano())

	// Track resources for cleanup
	var testIssueNum int
	var trackerIssueNum int

	// Cleanup at end of test
	defer func() {
		if testIssueNum > 0 {
			runCleanupAfterTest(t, testIssueNum)
		}
		if trackerIssueNum > 0 {
			runCleanupAfterTest(t, trackerIssueNum)
		}
		// Ensure branch is closed even if test fails
		runPMU(t, cfg.Dir, "branch", "close", "--yes")
	}()

	// Step 1: Start a new branch with unique timestamped name
	t.Run("start branch", func(t *testing.T) {
		result := runPMU(t, cfg.Dir, "branch", "start", "--name", branchName)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, branchName)
		// Extract tracker issue number for label verification and cleanup
		trackerIssueNum = extractIssueNumber(t, result.Stdout)
	})

	// Step 1b: Verify tracker issue has 'branch' label
	t.Run("verify branch label", func(t *testing.T) {
		if trackerIssueNum == 0 {
			t.Skip("No tracker issue number available")
		}
		assertHasLabel(t, trackerIssueNum, "branch")
	})

	// Step 2: Verify branch current shows the branch
	t.Run("verify current branch", func(t *testing.T) {
		result := runPMU(t, cfg.Dir, "branch", "current")
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, branchName)
	})

	// Step 3: Create test issue and add to branch
	t.Run("add issue to branch", func(t *testing.T) {
		// Create test issue
		testIssueNum = createTestIssue(t, cfg, "Branch Test Issue")

		// Add to branch via move --branch current
		result := runPMU(t, cfg.Dir, "move", fmt.Sprintf("%d", testIssueNum), "--branch", "current")
		assertExitCode(t, result, 0)
	})

	// Step 3b: Verify the issue is actually a member of the branch via read-back.
	// A zero-exit move above proves the command ran, not that membership was set;
	// list --branch current confirms the issue is genuinely on the branch.
	t.Run("verify issue branch membership", func(t *testing.T) {
		if testIssueNum == 0 {
			t.Fatal("No test issue number available for membership check")
		}
		result := waitForProjectSync(t, cfg, 5,
			[]string{"list", "--branch", "current"},
			fmt.Sprintf("#%d", testIssueNum),
		)
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, fmt.Sprintf("#%d", testIssueNum))
	})

	// Step 4: Verify branch list shows the branch
	t.Run("verify branch list", func(t *testing.T) {
		result := runPMU(t, cfg.Dir, "branch", "list")
		assertExitCode(t, result, 0)
		assertContains(t, result.Stdout, branchName)
	})

	// Step 5: Close the branch
	t.Run("close branch", func(t *testing.T) {
		result := runPMU(t, cfg.Dir, "branch", "close", "--yes")
		assertExitCode(t, result, 0)
	})

	// Step 6: Verify no current branch
	t.Run("verify no current branch", func(t *testing.T) {
		result := runPMU(t, cfg.Dir, "branch", "current")
		// After close, `branch current` reports no active branch and exits 0
		// ("No active release"). Assert unconditionally — the old `if ExitCode == 0`
		// guard let an errored command pass this subtest with zero assertions.
		assertExitCode(t, result, 0)
		// The just-closed branch must no longer be reported as current...
		assertNotContains(t, result.Stdout, branchName)
		// ...and current must positively report that no branch is active.
		assertContains(t, result.Stdout, "No active release")
	})
}
