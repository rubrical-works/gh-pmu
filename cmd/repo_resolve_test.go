package cmd

import (
	"testing"

	"github.com/rubrical-works/gh-pmu/internal/config"
)

func TestResolveRepoDefaults(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *config.Config
		repoFlag  string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"flag valid", nil, "o/r", "o", "r", false},
		{"flag empty owner", nil, "/r", "", "", true},
		{"flag empty repo", nil, "o/", "", "", true},
		{"flag too many parts", nil, "o/r/x", "", "", true},
		{"config fallback", &config.Config{Repositories: []string{"co/cr"}}, "", "co", "cr", false},
		{"config malformed empty component", &config.Config{Repositories: []string{"co/"}}, "", "", "", true},
		{"no flag no config", &config.Config{}, "", "", "", true},
		{"flag takes precedence over config", &config.Config{Repositories: []string{"co/cr"}}, "fo/fr", "fo", "fr", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := resolveRepoDefaults(tt.cfg, tt.repoFlag)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveRepoDefaults(%q) err=%v, wantErr=%v", tt.repoFlag, err, tt.wantErr)
			}
			if !tt.wantErr && (owner != tt.wantOwner || repo != tt.wantRepo) {
				t.Errorf("resolveRepoDefaults() = %s/%s, want %s/%s", owner, repo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

func TestApplyRepoDefaults(t *testing.T) {
	tests := []struct {
		name                string
		refOwner, refRepo   string
		defOwner, defRepo   string
		wantOwner, wantRepo string
	}{
		{"ref has both", "ro", "rr", "do", "dr", "ro", "rr"},
		{"ref empty owner uses defaults", "", "rr", "do", "dr", "do", "dr"},
		{"ref empty repo uses defaults", "ro", "", "do", "dr", "do", "dr"},
		{"ref empty both uses defaults", "", "", "do", "dr", "do", "dr"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo := applyRepoDefaults(tt.refOwner, tt.refRepo, tt.defOwner, tt.defRepo)
			if owner != tt.wantOwner || repo != tt.wantRepo {
				t.Errorf("applyRepoDefaults() = %s/%s, want %s/%s", owner, repo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}
