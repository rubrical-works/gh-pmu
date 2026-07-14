package version

import (
	"regexp"
	"testing"
)

// semverPattern is the shape /prepare-release writes into version.go.
// Intentionally strict: no pre-release or build-metadata suffixes are expected
// here, so anything else means the release workflow wrote a malformed version.
var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

func TestVersion_IsNotEmpty(t *testing.T) {
	if Version == "" {
		t.Error("Version constant should not be empty")
	}
}

func TestVersion_IsValidSemver(t *testing.T) {
	if !semverPattern.MatchString(Version) {
		t.Errorf("Version %q is not MAJOR.MINOR.PATCH", Version)
	}
}

// TestSemverPattern_RejectsMalformed pins the canary itself. The previous
// check only looked for a single dot anywhere in the string, so it accepted
// every input below and could not have caught a malformed release version.
func TestSemverPattern_RejectsMalformed(t *testing.T) {
	invalid := []string{
		"",
		".",
		"1.",
		".1",
		"a.b",
		"a.b.c",
		"1.4",
		"1.4.10.2",
		"v1.4.10",
		"1.4.10-rc1",
		" 1.4.10",
		"1.4.10 ",
	}

	for _, v := range invalid {
		if semverPattern.MatchString(v) {
			t.Errorf("expected %q to be rejected as a version", v)
		}
	}
}

func TestSemverPattern_AcceptsWellFormed(t *testing.T) {
	valid := []string{"0.0.0", "1.4.10", "10.20.30"}

	for _, v := range valid {
		if !semverPattern.MatchString(v) {
			t.Errorf("expected %q to be accepted as a version", v)
		}
	}
}
