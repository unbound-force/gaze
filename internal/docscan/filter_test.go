package docscan_test

import (
	"testing"

	"github.com/unbound-force/gaze/internal/config"
	"github.com/unbound-force/gaze/internal/docscan"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name string
		rel  string
		cfg  *config.GazeConfig
		want bool
	}{
		// FR-001: default inclusion (no include/exclude patterns).
		{
			name: "default_inclusion_no_patterns",
			rel:  "docs/readme.md",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{},
				},
			},
			want: true,
		},
		// FR-001: nil config fallback (uses DefaultConfig).
		{
			name: "nil_config_fallback_included",
			rel:  "README.md",
			cfg:  nil,
			want: true,
		},
		// FR-001: nil config with excluded file (DefaultConfig excludes LICENSE).
		{
			name: "nil_config_fallback_excluded",
			rel:  "LICENSE",
			cfg:  nil,
			want: false,
		},
		// FR-001: include-pattern match (file matches include glob).
		{
			name: "include_pattern_match",
			rel:  "docs/design.md",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{
						Include: []string{"docs/**"},
					},
				},
			},
			want: true,
		},
		// FR-001: include-pattern miss (file does not match any include glob).
		{
			name: "include_pattern_miss",
			rel:  "src/main.go",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{
						Include: []string{"docs/**"},
					},
				},
			},
			want: false,
		},
		// FR-001: exclude-pattern match (file matches exclude glob).
		{
			name: "exclude_pattern_match",
			rel:  "vendor/lib/readme.md",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{
						Exclude: []string{"vendor/**"},
					},
				},
			},
			want: false,
		},
		// Include match overridden by exclude.
		{
			name: "include_match_then_exclude_match",
			rel:  "docs/internal/secret.md",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{
						Include: []string{"docs/**"},
						Exclude: []string{"docs/internal/**"},
					},
				},
			},
			want: false,
		},
		// FR-002: pattern with path separator (full-path matching).
		{
			name: "glob_full_path_match",
			rel:  "specs/001/spec.md",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{
						Include: []string{"specs/**"},
					},
				},
			},
			want: true,
		},
		// FR-002: pattern without path separator (base-name matching).
		{
			name: "glob_basename_match",
			rel:  "deep/nested/dir/README.md",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{
						Exclude: []string{"README.md"},
					},
				},
			},
			want: false,
		},
		// FR-002: pattern without separator matches basename only.
		{
			name: "glob_basename_wildcard",
			rel:  "some/path/notes.txt",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{
						Include: []string{"*.txt"},
					},
				},
			},
			want: true,
		},
		// Edge case: forward-slash path matching.
		// On Unix, filepath.ToSlash is a no-op (backslash is a valid
		// filename character, not a separator). On Windows, it
		// converts backslashes to forward slashes. This test verifies
		// that Filter handles forward-slash paths correctly on all
		// platforms. Backslash normalization is only meaningful on
		// Windows and cannot be tested on Unix.
		{
			name: "forward_slash_path_match",
			rel:  "docs/readme.md",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{
						Include: []string{"docs/**"},
					},
				},
			},
			want: true,
		},
		// Double-star prefix: exact directory name match.
		{
			name: "doublestar_exact_dir_match",
			rel:  "testdata",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{
						Exclude: []string{"testdata/**"},
					},
				},
			},
			want: false,
		},
		// Exclude pattern that does not match.
		{
			name: "exclude_pattern_no_match",
			rel:  "docs/guide.md",
			cfg: &config.GazeConfig{
				Classification: config.ClassificationConfig{
					DocScan: config.DocScan{
						Exclude: []string{"vendor/**"},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := docscan.Filter(tt.rel, tt.cfg)
			if got != tt.want {
				t.Errorf("Filter(%q, cfg) = %v, want %v", tt.rel, got, tt.want)
			}
		})
	}
}
