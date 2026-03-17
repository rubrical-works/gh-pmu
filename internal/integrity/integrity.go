package integrity

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Drifted bool
	Changes []string
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
	return os.WriteFile(path, []byte(checksum+"\n"), 0644)
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
	return os.WriteFile(path, data, 0644)
}

// CompareContent compares local config content against committed content.
// If committed is nil or empty, reports drift (no committed version found).
func CompareContent(local, committed []byte) (*ComparisonResult, error) {
	if committed == nil || len(committed) == 0 {
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
	changes := diffJSON(local, committed)

	return &ComparisonResult{
		Drifted: true,
		Changes: changes,
	}, nil
}

// diffJSON compares two JSON documents and returns human-readable change descriptions.
func diffJSON(local, committed []byte) []string {
	var localMap, committedMap map[string]interface{}

	if err := json.Unmarshal(local, &localMap); err != nil {
		return []string{"Local config is not valid JSON"}
	}
	if err := json.Unmarshal(committed, &committedMap); err != nil {
		return []string{"Committed config is not valid JSON"}
	}

	var changes []string
	diffMaps("", localMap, committedMap, &changes)

	if len(changes) == 0 {
		changes = append(changes, "Content differs (whitespace or formatting change)")
	}

	return changes
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
