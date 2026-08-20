package integrity

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ChecksumFileName is the name of the checksum file stored in the project root.
const ChecksumFileName = ".gh-pmu.checksum"

// ThrottleFileName is the name of the throttle state file.
const ThrottleFileName = ".gh-pmu-integrity-check.json"

// ThrottleState tracks when the last integrity check was performed.
type ThrottleState struct {
	LastCheck string `json:"lastCheck"`
}

// ComparisonResult holds the result of comparing local vs committed config.
type ComparisonResult struct {
	Drifted   bool
	Changes   []string
	Unchanged []string
}

// ComputeChecksum returns the SHA-256 hex digest of the file at path.
func ComputeChecksum(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file for checksum: %w", err)
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// SaveChecksum writes a checksum string to the checksum file in dir.
func SaveChecksum(dir, checksum string) error {
	path := filepath.Join(dir, ChecksumFileName)
	return os.WriteFile(path, []byte(checksum+"\n"), 0600)
}

// LoadChecksum reads the stored checksum from the checksum file in dir.
// Returns empty string with no error if the file does not exist.
func LoadChecksum(dir string) (string, error) {
	path := filepath.Join(dir, ChecksumFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("failed to read checksum file: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// IsThrottled returns true if an integrity check was already performed today.
// Uses ISO 8601 date comparison (midnight boundary in UTC).
func IsThrottled(dir string) (bool, error) {
	path := filepath.Join(dir, ThrottleFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read throttle state: %w", err)
	}

	var state ThrottleState
	if err := json.Unmarshal(data, &state); err != nil {
		// Corrupted file — treat as not throttled
		return false, nil
	}

	lastCheck, err := time.Parse(time.RFC3339, state.LastCheck)
	if err != nil {
		return false, nil
	}

	// Compare dates (midnight boundary)
	now := time.Now().UTC()
	return lastCheck.UTC().Format("2006-01-02") == now.Format("2006-01-02"), nil
}

// RecordCheck writes the current timestamp to the throttle state file.
func RecordCheck(dir string) error {
	state := ThrottleState{
		LastCheck: time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal throttle state: %w", err)
	}
	path := filepath.Join(dir, ThrottleFileName)
	return os.WriteFile(path, data, 0600)
}

// UpdateChecksumForConfig computes the checksum of the config file at path
// and saves it to the checksum file in the same directory.
// Call this after config.Save() to keep the checksum in sync.
func UpdateChecksumForConfig(configPath string) error {
	checksum, err := ComputeChecksum(configPath)
	if err != nil {
		return err
	}
	return SaveChecksum(filepath.Dir(configPath), checksum)
}

// CompareContent compares local config content against committed content.
// If committed is nil or empty, reports drift (no committed version found).
func CompareContent(local, committed []byte) (*ComparisonResult, error) {
	if len(committed) == 0 {
		return &ComparisonResult{
			Drifted: true,
			Changes: []string{"No committed version found — local config has no git baseline"},
		}, nil
	}

	localHash := fmt.Sprintf("%x", sha256.Sum256(local))
	committedHash := fmt.Sprintf("%x", sha256.Sum256(committed))

	if localHash == committedHash {
		return &ComparisonResult{Drifted: false}, nil
	}

	// Parse both as JSON to find specific differences
	changes, unchanged, dropped := diffJSON(local, committed)

	// Only tool-written keys moved. Reporting this as drift would blame the
	// user for gh pmu's own write, and would block them outright under strict
	// mode (#901).
	if len(changes) == 0 && dropped > 0 {
		return &ComparisonResult{Drifted: false}, nil
	}

	return &ComparisonResult{
		Drifted:   true,
		Changes:   changes,
		Unchanged: unchanged,
	}, nil
}

// driftExcludedKeys lists dotted config paths that gh pmu writes on its own
// behalf and that must therefore not count as user-visible drift.
//
// project.view is resolved from the API and cached the first time it is needed
// (#901). version is stamped by config.RefreshVersion whenever the running
// binary differs from the one that last wrote the file (#905). Without these
// exclusions, either write makes the next integrity check report drift the user
// did not cause — and under strict mode (cmd/integrity_check.go) drift is a hard
// error that blocks every subsequent command until .gh-pmu.json is committed.
//
// The version case is the sharper one: the refresh runs in the same
// PersistentPreRunE as runDailyIntegrityCheck, three lines earlier, so without
// the exclusion a single command would write the file and then report itself.
//
// Updating the stored checksum does not help: the drift check compares the
// local file against the git HEAD blob and never reads the checksum file.
//
// Keep this list minimal and exact. It suppresses drift reporting, so a key
// added here stops being watched. Matching is on the full dotted path, so a
// top-level "view" and a nested "acceptance.version" are both unaffected, and
// the project.owner / project.number / repositories alerting from #792 is
// untouched.
var driftExcludedKeys = map[string]bool{
	"project.view": true,
	"version":      true,
}

// isDriftExcluded reports whether a change description refers to an excluded
// key. Descriptions are formatted as "Changed: project.view", "Added: x" or
// "Removed: x" by diffMaps.
func isDriftExcluded(change string) bool {
	parts := strings.SplitN(change, ": ", 2)
	if len(parts) != 2 {
		return false
	}
	return driftExcludedKeys[parts[1]]
}

// diffJSON compares two JSON documents and returns change descriptions, the
// unchanged top-level keys, and how many changes were dropped as
// tool-written (driftExcludedKeys).
//
// The dropped count is what lets CompareContent tell "nothing changed but our
// own bookkeeping" apart from "changed in some way diffMaps could not name" —
// the two are both an empty change list, but only the first is not drift.
func diffJSON(local, committed []byte) (changes []string, unchanged []string, dropped int) {
	var localMap, committedMap map[string]interface{}

	if err := json.Unmarshal(local, &localMap); err != nil {
		return []string{"Local config is not valid JSON"}, nil, 0
	}
	if err := json.Unmarshal(committed, &committedMap); err != nil {
		return []string{"Committed config is not valid JSON"}, nil, 0
	}

	diffMaps("", localMap, committedMap, &changes)

	// Drop tool-written keys before anything else looks at the list, so they
	// never reach a report or the unchanged-key tally below.
	kept := changes[:0]
	for _, c := range changes {
		if isDriftExcluded(c) {
			dropped++
			continue
		}
		kept = append(kept, c)
	}
	changes = kept

	// Bytes differ but no key does: a formatting-only edit, which has always
	// been reported. Suppress it only when an excluded key explains the
	// difference, otherwise real reformatting would go unreported.
	if len(changes) == 0 && dropped == 0 {
		changes = append(changes, "Content differs (whitespace or formatting change)")
	}

	// Identify unchanged top-level keys
	changedTopLevel := make(map[string]bool)
	for _, c := range changes {
		// Extract top-level key from "Changed: key.sub" or "Added: key" etc.
		parts := strings.SplitN(c, ": ", 2)
		if len(parts) == 2 {
			topKey := strings.SplitN(parts[1], ".", 2)[0]
			changedTopLevel[topKey] = true
		}
	}

	// Collect all top-level keys from both maps
	allKeys := make(map[string]bool)
	for k := range localMap {
		allKeys[k] = true
	}
	for k := range committedMap {
		allKeys[k] = true
	}

	// Keys present in both but not in changed set are unchanged
	for k := range allKeys {
		if !changedTopLevel[k] {
			unchanged = append(unchanged, k)
		}
	}

	// Sort for deterministic output
	sort.Strings(unchanged)

	return changes, unchanged, dropped
}

// diffMaps recursively compares two maps and appends change descriptions.
func diffMaps(prefix string, local, committed map[string]interface{}, changes *[]string) {
	for key, localVal := range local {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		committedVal, exists := committed[key]
		if !exists {
			*changes = append(*changes, fmt.Sprintf("Added: %s", fullKey))
			continue
		}

		localMap, localIsMap := localVal.(map[string]interface{})
		committedMap, committedIsMap := committedVal.(map[string]interface{})

		if localIsMap && committedIsMap {
			diffMaps(fullKey, localMap, committedMap, changes)
		} else if fmt.Sprintf("%v", localVal) != fmt.Sprintf("%v", committedVal) {
			*changes = append(*changes, fmt.Sprintf("Changed: %s", fullKey))
		}
	}

	for key := range committed {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}
		if _, exists := local[key]; !exists {
			*changes = append(*changes, fmt.Sprintf("Removed: %s", fullKey))
		}
	}
}
