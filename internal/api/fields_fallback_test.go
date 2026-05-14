package api

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// Tests for the cached-metadata fallback path engaged when
// ErrFieldResolverUnavailable surfaces from the bulk fetch. See issue #853.

func TestGetProjectFields_Fallback_EngagesOnResolverUnavailable(t *testing.T) {
	resetFieldsCacheWarningForTesting()
	cached := []ProjectField{
		{ID: "f1", Name: "Status", DataType: "SINGLE_SELECT", Options: []FieldOption{
			{ID: "o1", Name: "Backlog"},
			{ID: "o2", Name: "Done"},
		}},
		{ID: "f2", Name: "Branch", DataType: "TEXT"},
	}
	restore := SetCachedFieldsLoaderForTesting(func() ([]ProjectField, error) {
		return cached, nil
	})
	defer restore()

	var stderr bytes.Buffer
	restoreW := SetFieldsCacheWarningWriterForTesting(&stderr)
	defer restoreW()

	mock := &queryMockClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("GraphQL: Something went wrong while executing your query on 2026-05-14T19:00:00Z. Please include `ABCD:1234:ABCD:1234:6A060000` when reporting this issue.")
		},
	}

	client := NewClientWithGraphQL(mock)
	fields, err := client.GetProjectFields("proj-id")

	if err != nil {
		t.Fatalf("Expected fallback to succeed; got error: %v", err)
	}
	if len(fields) != 2 {
		t.Fatalf("Expected 2 cached fields; got %d", len(fields))
	}
	if fields[0].Name != "Status" || fields[1].Name != "Branch" {
		t.Errorf("Expected cached fields preserved in order; got %v", fields)
	}
	if len(fields[0].Options) != 2 {
		t.Errorf("Expected single-select options preserved; got %d options", len(fields[0].Options))
	}
	if !strings.Contains(stderr.String(), "ProjectV2 field resolver unavailable") {
		t.Errorf("Expected stderr warning emitted; got: %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "2 fields") {
		t.Errorf("Expected warning to mention field count; got: %q", stderr.String())
	}
}

func TestGetProjectFields_Fallback_EmptyCacheReturnsActionableError(t *testing.T) {
	resetFieldsCacheWarningForTesting()
	restore := SetCachedFieldsLoaderForTesting(func() ([]ProjectField, error) {
		return nil, nil // cache loaded but empty
	})
	defer restore()

	mock := &queryMockClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("GraphQL: Something went wrong while executing your query on 2026-05-14T19:00:00Z. Please include `WXYZ:9999:WXYZ:9999:6A060000` when reporting this issue.")
		},
	}

	client := NewClientWithGraphQL(mock)
	fields, err := client.GetProjectFields("proj-id")

	if err == nil {
		t.Fatal("Expected error when cache is empty and resolver is unavailable")
	}
	if fields != nil {
		t.Errorf("Expected nil fields when fallback can't recover; got %d entries", len(fields))
	}
	msg := err.Error()
	if !strings.Contains(msg, "field resolver unavailable") {
		t.Errorf("Expected actionable error to mention resolver unavailable; got: %v", err)
	}
	if !strings.Contains(msg, "gh pmu init") {
		t.Errorf("Expected actionable error to suggest 'gh pmu init'; got: %v", err)
	}
	// Sentinel must still be detectable for upstream callers that want to
	// distinguish "the resolver crashed AND we couldn't recover" from
	// "some other failure."
	if !errors.Is(err, ErrFieldResolverUnavailable) {
		t.Errorf("Expected sentinel preserved through empty-cache path; got: %v", err)
	}
}

func TestGetProjectFields_Fallback_NonResolverErrorPropagates(t *testing.T) {
	resetFieldsCacheWarningForTesting()
	cacheCalled := false
	restore := SetCachedFieldsLoaderForTesting(func() ([]ProjectField, error) {
		cacheCalled = true
		return []ProjectField{{ID: "x", Name: "x"}}, nil
	})
	defer restore()

	mock := &queryMockClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			// Generic, non-resolver-shaped error (e.g., network failure).
			return errors.New("dial tcp: lookup api.github.com: no such host")
		},
	}

	client := NewClientWithGraphQL(mock)
	fields, err := client.GetProjectFields("proj-id")

	if err == nil {
		t.Fatal("Expected non-resolver error to propagate; got nil")
	}
	if fields != nil {
		t.Errorf("Expected nil fields for non-resolver failure; got %d", len(fields))
	}
	if cacheCalled {
		t.Error("Cache loader must NOT be called for non-resolver errors — that would mask real failures")
	}
	if errors.Is(err, ErrFieldResolverUnavailable) {
		t.Error("Non-resolver error must NOT be classified as resolver-unavailable")
	}
}

func TestGetProjectFields_Fallback_WarningEmittedOncePerProcess(t *testing.T) {
	resetFieldsCacheWarningForTesting()
	cached := []ProjectField{{ID: "f1", Name: "Status", DataType: "TEXT"}}
	restore := SetCachedFieldsLoaderForTesting(func() ([]ProjectField, error) {
		return cached, nil
	})
	defer restore()

	var stderr bytes.Buffer
	restoreW := SetFieldsCacheWarningWriterForTesting(&stderr)
	defer restoreW()

	mock := &queryMockClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("GraphQL: Something went wrong while executing your query")
		},
	}
	client := NewClientWithGraphQL(mock)

	// Three calls in the same "process" (test invocation).
	for i := 0; i < 3; i++ {
		_, err := client.GetProjectFields("proj-id")
		if err != nil {
			t.Fatalf("call %d: unexpected error: %v", i, err)
		}
	}

	output := stderr.String()
	occurrences := strings.Count(output, "ProjectV2 field resolver unavailable")
	if occurrences != 1 {
		t.Errorf("Expected exactly 1 warning emission across 3 fallbacks; got %d. Output: %q", occurrences, output)
	}
}

func TestGetProjectFields_Fallback_LoaderErrorReturnsActionableError(t *testing.T) {
	resetFieldsCacheWarningForTesting()
	restore := SetCachedFieldsLoaderForTesting(func() ([]ProjectField, error) {
		return nil, errors.New("no .gh-pmu.json found in current directory or any parent")
	})
	defer restore()

	mock := &queryMockClient{
		queryFunc: func(name string, query interface{}, variables map[string]interface{}) error {
			return errors.New("GraphQL: Something went wrong while executing your query")
		},
	}

	client := NewClientWithGraphQL(mock)
	fields, err := client.GetProjectFields("proj-id")

	if err == nil {
		t.Fatal("Expected error when cache loader itself fails")
	}
	if fields != nil {
		t.Errorf("Expected nil fields; got %d", len(fields))
	}
	if !errors.Is(err, ErrFieldResolverUnavailable) {
		t.Error("Sentinel must be preserved through cache-loader-failure path")
	}
	// The loader's own error should surface to the user so they can fix it.
	if !strings.Contains(err.Error(), "no .gh-pmu.json found") {
		t.Errorf("Expected loader error to surface; got: %v", err)
	}
}

func TestProjectFieldsFromMetadata_PreservesOptions(t *testing.T) {
	// Pure converter test — no network, no config file.
	// Mirrors the shape config.FieldMetadata produces from .gh-pmu.json.
	fields := projectFieldsFromMetadataShape([]cachedFieldShape{
		{Name: "Status", ID: "f1", DataType: "SINGLE_SELECT", Options: []cachedOptionShape{
			{Name: "Backlog", ID: "o1"},
			{Name: "Done", ID: "o2"},
		}},
		{Name: "Branch", ID: "f2", DataType: "TEXT"},
	})

	if len(fields) != 2 {
		t.Fatalf("Expected 2 fields; got %d", len(fields))
	}
	if fields[0].ID != "f1" || fields[0].Name != "Status" || fields[0].DataType != "SINGLE_SELECT" {
		t.Errorf("Field 0 conversion mismatch: %+v", fields[0])
	}
	if len(fields[0].Options) != 2 {
		t.Fatalf("Expected 2 options on first field; got %d", len(fields[0].Options))
	}
	if fields[0].Options[0].ID != "o1" || fields[0].Options[0].Name != "Backlog" {
		t.Errorf("Option 0 conversion mismatch: %+v", fields[0].Options[0])
	}
	if fields[1].DataType != "TEXT" || len(fields[1].Options) != 0 {
		t.Errorf("TEXT field should have no options; got: %+v", fields[1])
	}
}
