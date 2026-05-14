package api

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/rubrical-works/gh-pmu/internal/config"
)

// fields_fallback.go: cached-metadata fallback for the ProjectV2.fields
// resolver, engaged when the bulk fetch returns ErrFieldResolverUnavailable.
// See issue #853 — added after the 2026-05-14 timestamp-fields rollout broke
// the bulk resolver across user projects. The cache lives in .gh-pmu.json
// (metadata.fields[]) and is populated by `gh pmu init`.

// cachedFieldShape mirrors config.FieldMetadata structurally so the converter
// can be tested without importing config types into test files. Kept tiny on
// purpose — only what the live ProjectField needs.
type cachedFieldShape struct {
	Name     string
	ID       string
	DataType string
	Options  []cachedOptionShape
}

type cachedOptionShape struct {
	Name string
	ID   string
}

// Module state. Defaults are production-safe; tests override via the
// SetXxxForTesting helpers below.
var (
	fieldsCacheLoader        = defaultFieldsCacheLoader
	fieldsCacheWarning       sync.Once
	fieldsCacheWarningWriter io.Writer = os.Stderr
)

// defaultFieldsCacheLoader reads .gh-pmu.json starting from the working
// directory (walks up the tree) and returns its metadata.fields[] converted
// to []ProjectField. Returns nil (no error) when the config exists but has
// no cached fields — callers treat that as "cache missing" and surface an
// actionable error.
func defaultFieldsCacheLoader() ([]ProjectField, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve working directory for cache lookup: %w", err)
	}
	cfg, err := config.LoadFromDirectory(cwd)
	if err != nil {
		return nil, err
	}
	if cfg == nil || cfg.Metadata == nil || len(cfg.Metadata.Fields) == 0 {
		return nil, nil
	}

	shapes := make([]cachedFieldShape, 0, len(cfg.Metadata.Fields))
	for _, fm := range cfg.Metadata.Fields {
		s := cachedFieldShape{Name: fm.Name, ID: fm.ID, DataType: fm.DataType}
		for _, opt := range fm.Options {
			s.Options = append(s.Options, cachedOptionShape{Name: opt.Name, ID: opt.ID})
		}
		shapes = append(shapes, s)
	}
	return projectFieldsFromMetadataShape(shapes), nil
}

// projectFieldsFromMetadataShape is a pure converter from cached metadata
// shapes to []ProjectField. FieldOption.Color is intentionally left empty:
// the cache doesn't carry it, and the live path at queries.go also leaves it
// empty for non-Color-aware queries.
func projectFieldsFromMetadataShape(shapes []cachedFieldShape) []ProjectField {
	if len(shapes) == 0 {
		return nil
	}
	out := make([]ProjectField, 0, len(shapes))
	for _, s := range shapes {
		pf := ProjectField{ID: s.ID, Name: s.Name, DataType: s.DataType}
		for _, opt := range s.Options {
			pf.Options = append(pf.Options, FieldOption{ID: opt.ID, Name: opt.Name})
		}
		out = append(out, pf)
	}
	return out
}

// emitFieldsCacheWarning writes the warning exactly once per process,
// regardless of how many GetProjectFields calls fall back. AC3: "Repeated
// calls within one command invocation must not spam — emit once per process."
func emitFieldsCacheWarning(count int) {
	fieldsCacheWarning.Do(func() {
		fmt.Fprintf(fieldsCacheWarningWriter,
			"warning: ProjectV2 field resolver unavailable (GraphQL 5xx); using cached field metadata from .gh-pmu.json (%d fields).\n",
			count)
	})
}

// loadCachedProjectFieldsOrError engages the fallback after a resolver crash.
// Returns the cached fields on success, or an actionable error on cache miss
// or loader failure. The resolver sentinel is preserved through both error
// paths so upstream callers can distinguish "resolver unavailable AND cache
// unusable" from "some other failure."
//
// origErr is the original resolver error — joined into empty-cache errors so
// the trace ID stays visible to users reporting to GitHub.
func loadCachedProjectFieldsOrError(origErr error) ([]ProjectField, error) {
	fields, loadErr := fieldsCacheLoader()
	if loadErr != nil {
		return nil, fmt.Errorf(
			"ProjectV2 field resolver unavailable and cached metadata could not be loaded — wait for GitHub to recover or run 'gh pmu init' from a working environment: %w",
			errors.Join(ErrFieldResolverUnavailable, loadErr))
	}
	if len(fields) == 0 {
		return nil, fmt.Errorf(
			"ProjectV2 field resolver unavailable and no cached metadata in .gh-pmu.json — run 'gh pmu init' from a working environment or wait for GitHub to recover: %w",
			errors.Join(ErrFieldResolverUnavailable, origErr))
	}
	emitFieldsCacheWarning(len(fields))
	return fields, nil
}

// ─── Test hooks ───
// These are exported only so external packages' tests can configure the
// fallback when integration-testing commands that consume field metadata.
// Production code should never call these.

// SetCachedFieldsLoaderForTesting overrides the loader. Returns a restore
// function (defer-friendly).
func SetCachedFieldsLoaderForTesting(fn func() ([]ProjectField, error)) func() {
	prev := fieldsCacheLoader
	fieldsCacheLoader = fn
	return func() { fieldsCacheLoader = prev }
}

// SetFieldsCacheWarningWriterForTesting redirects the warning sink.
func SetFieldsCacheWarningWriterForTesting(w io.Writer) func() {
	prev := fieldsCacheWarningWriter
	fieldsCacheWarningWriter = w
	return func() { fieldsCacheWarningWriter = prev }
}

// resetFieldsCacheWarningForTesting resets the sync.Once gate so tests can
// observe the warning fresh. Lowercase (intra-package) on purpose — the
// reset has no production use.
func resetFieldsCacheWarningForTesting() {
	fieldsCacheWarning = sync.Once{}
}
