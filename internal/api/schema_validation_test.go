package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// ============================================================================
// Vendored Schema Provenance Tests (#886)
// ============================================================================
//
// The vendored schema is a pinned copy of GitHub's public GraphQL schema. It is
// the oracle the operation-validation tests below check against, so its
// provenance has to be recorded and verifiable: a schema of unknown age is
// indistinguishable from a schema that has drifted.
//
// Refresh procedure: TESTING.md -> "Vendored Schema Validation".

var (
	vendoredSchemaPath   = filepath.Join("..", "..", "testdata", "graphql", "schema.docs.graphql")
	schemaProvenancePath = filepath.Join("..", "..", "testdata", "graphql", "schema-provenance.json")
)

// schemaProvenance mirrors testdata/graphql/schema-provenance.json.
type schemaProvenance struct {
	Source    string `json:"source"`
	Retrieved string `json:"retrieved"`
	Bytes     int64  `json:"bytes"`
	SHA256    string `json:"sha256"`
}

func readProvenance(t *testing.T) schemaProvenance {
	t.Helper()
	raw, err := os.ReadFile(schemaProvenancePath)
	if err != nil {
		t.Fatalf("provenance record not found at %s: %v", schemaProvenancePath, err)
	}
	var p schemaProvenance
	if err := json.Unmarshal(raw, &p); err != nil {
		t.Fatalf("provenance record is not valid JSON: %v", err)
	}
	return p
}

func TestVendoredSchema_FileExists(t *testing.T) {
	info, err := os.Stat(vendoredSchemaPath)
	if err != nil {
		t.Fatalf("vendored schema not found at %s: %v", vendoredSchemaPath, err)
	}
	if info.Size() < 1_000_000 {
		t.Errorf("vendored schema is %d bytes; expected the full public schema (>1MB)", info.Size())
	}
}

func TestVendoredSchema_ProvenanceRecordsSourceAndDate(t *testing.T) {
	p := readProvenance(t)

	if p.Source == "" {
		t.Error("provenance record has no source URL")
	}
	if p.Retrieved == "" {
		t.Error("provenance record has no retrieval date")
	}
}

// TestVendoredSchema_MatchesProvenance is what makes drift legible: if the file
// on disk no longer matches the recorded digest, either the schema was
// refreshed without updating the record, or it was edited by hand. Both are
// bugs — the vendored copy must be a verbatim upstream artifact.
func TestVendoredSchema_MatchesProvenance(t *testing.T) {
	p := readProvenance(t)

	raw, err := os.ReadFile(vendoredSchemaPath)
	if err != nil {
		t.Fatalf("vendored schema not found at %s: %v", vendoredSchemaPath, err)
	}

	if int64(len(raw)) != p.Bytes {
		t.Errorf("schema is %d bytes but provenance records %d — refresh the record or restore the file",
			len(raw), p.Bytes)
	}

	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != p.SHA256 {
		t.Errorf("schema sha256 is %s but provenance records %s — refresh the record or restore the file",
			got, p.SHA256)
	}
}
