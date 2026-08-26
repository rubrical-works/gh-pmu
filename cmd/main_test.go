package cmd

import (
	"os"
	"testing"
)

// TestMain enables repo-root write protection for every test in this package.
//
// go test ./cmd/ runs with the package directory as the working directory, and
// config.FindConfigFile walks up from there to the repository's own
// .gh-pmu.json. Any test that executes a command through PersistentPreRunE
// therefore reaches the real config. SetRepoRootProtection's own documentation
// says test setup should call it; before this nothing did, so the guard was
// inert outside the handful of init tests that toggled it themselves (#436).
func TestMain(m *testing.M) {
	SetRepoRootProtection(true)
	os.Exit(m.Run())
}

// setRepoRootProtectionForTest sets the protection flag for the duration of one
// test and restores whatever it was before.
//
// Hard-coding the restore to false — as several tests did — silently disables
// the guard for every test that runs afterwards, which is how the package-wide
// protection TestMain installs used to leak.
func setRepoRootProtectionForTest(t *testing.T, enabled bool) {
	t.Helper()
	previous := protectRepoRoot.Load()
	t.Cleanup(func() { SetRepoRootProtection(previous) })
	SetRepoRootProtection(enabled)
}
