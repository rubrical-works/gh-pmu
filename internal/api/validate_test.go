package api

import (
	"strings"
	"testing"
)

func TestValidateOwner_Valid(t *testing.T) {
	valid := []string{
		"a",
		"A",
		"octocat",
		"Rubrical-Works",
		"user123",
		"a-b-c",
		strings.Repeat("a", 39),
	}
	for _, v := range valid {
		if err := validateOwner(v); err != nil {
			t.Errorf("validateOwner(%q) unexpected error: %v", v, err)
		}
	}
}

func TestValidateOwner_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"-leading",
		"trailing-",
		"has space",
		"has.dot",
		"has/slash",
		`has\backslash`,
		`has"quote`,
		strings.Repeat("a", 40),
		"a\u0000b",
	}
	for _, v := range invalid {
		if err := validateOwner(v); err == nil {
			t.Errorf("validateOwner(%q) expected error, got nil", v)
		}
	}
}

func TestValidateRepo_Valid(t *testing.T) {
	valid := []string{
		"a",
		"gh-pmu",
		"repo.with.dots",
		"repo_with_underscores",
		"Mixed-Case_1.0",
		strings.Repeat("a", 100),
	}
	for _, v := range valid {
		if err := validateRepo(v); err != nil {
			t.Errorf("validateRepo(%q) unexpected error: %v", v, err)
		}
	}
}

func TestValidateRepo_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"has space",
		"has/slash",
		`has\backslash`,
		`has"quote`,
		strings.Repeat("a", 101),
		"tab\there",
	}
	for _, v := range invalid {
		if err := validateRepo(v); err == nil {
			t.Errorf("validateRepo(%q) expected error, got nil", v)
		}
	}
}

func TestValidateLabelName_Valid(t *testing.T) {
	valid := []string{
		"bug",
		"help wanted",
		"good first issue",
		"priority:high",
		"v1.0",
		"area/backend",
		"P0",
	}
	for _, v := range valid {
		if err := validateLabelName(v); err != nil {
			t.Errorf("validateLabelName(%q) unexpected error: %v", v, err)
		}
	}
}

func TestValidateLabelName_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"has\nnewline",
		"has\rcarriage",
		"has\ttab",
		`has"doublequote`,
		`has\backslash`,
		"has\u0000null",
		strings.Repeat("x", 51),
	}
	for _, v := range invalid {
		if err := validateLabelName(v); err == nil {
			t.Errorf("validateLabelName(%q) expected error, got nil", v)
		}
	}
}

func TestValidateNodeID_Valid(t *testing.T) {
	valid := []string{
		"MDU6SXNzdWUxMjM=",
		"I_kwDOA123",
		"PVTI_lADOA1234567890",
		"abcABC012_-=",
	}
	for _, v := range valid {
		if err := validateNodeID(v); err != nil {
			t.Errorf("validateNodeID(%q) unexpected error: %v", v, err)
		}
	}
}

func TestValidateNodeID_Invalid(t *testing.T) {
	invalid := []string{
		"",
		"has space",
		`has"quote`,
		`has\backslash`,
		"has\nnewline",
	}
	for _, v := range invalid {
		if err := validateNodeID(v); err == nil {
			t.Errorf("validateNodeID(%q) expected error, got nil", v)
		}
	}
}

// End-to-end: a label name containing characters that would be dangerous in
// a quoted GraphQL string is rejected at the validator boundary before
// reaching the raw-query builder. The Client has no rawGQL client configured —
// if validation did not fire first, the call would fail with "raw GraphQL
// client not initialized" instead of the validator's "unsafe character" message.
func TestGetLabelIDs_RejectsDangerousLabel(t *testing.T) {
	c := &Client{}
	_, err := c.getLabelIDs("octocat", "gh-pmu", []string{`evil"}) { } injected(x: "`})
	if err == nil {
		t.Fatal("getLabelIDs expected error for dangerous label, got nil")
	}
	if !strings.Contains(err.Error(), "unsafe character") {
		t.Errorf("expected validator error mentioning 'unsafe character', got: %v", err)
	}
}

func TestGetLabelIDs_RejectsInvalidOwner(t *testing.T) {
	c := &Client{}
	_, err := c.getLabelIDs(`bad"owner`, "gh-pmu", []string{"bug"})
	if err == nil {
		t.Fatal("getLabelIDs expected error for invalid owner, got nil")
	}
	if !strings.Contains(err.Error(), "owner contains invalid characters") {
		t.Errorf("expected validator error mentioning owner, got: %v", err)
	}
}

// A label with a colon (e.g., `priority:high`) is a common valid GitHub label
// and must pass the validator — the round-trip proves we do not over-reject.
func TestGetLabelIDs_AllowsColonInLabel(t *testing.T) {
	if err := validateLabelName("priority:high"); err != nil {
		t.Errorf("label with colon should be accepted, got: %v", err)
	}
	if err := validateLabelName("area/backend"); err != nil {
		t.Errorf("label with slash should be accepted, got: %v", err)
	}
	if err := validateLabelName("good first issue"); err != nil {
		t.Errorf("label with spaces should be accepted, got: %v", err)
	}
}
