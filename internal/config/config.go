// Package config handles loading and providing Gaze configuration
// from .gaze.yaml files with sensible defaults.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Thresholds defines the confidence score boundaries for
// classification labels.
type Thresholds struct {
	// Contractual is the minimum confidence for the contractual
	// label. Scores >= this value are classified as contractual.
	Contractual int `yaml:"contractual"`

	// Incidental is the upper bound for the incidental label.
	// Scores < this value are classified as incidental.
	Incidental int `yaml:"incidental"`
}

// DocScan defines document scanning configuration.
type DocScan struct {
	// Exclude is a list of glob patterns for files to exclude
	// from document scanning.
	Exclude []string `yaml:"exclude"`

	// Include is a list of glob patterns for files to include.
	// If set, only matching files are processed, overriding the
	// default full-repo scan.
	Include []string `yaml:"include"`

	// Timeout is the maximum duration for document scanning.
	Timeout time.Duration `yaml:"-"`

	// TimeoutStr is the string representation for YAML parsing.
	TimeoutStr string `yaml:"timeout"`
}

// ClassificationConfig groups all classification-related settings.
type ClassificationConfig struct {
	// Thresholds defines the confidence score boundaries.
	Thresholds Thresholds `yaml:"thresholds"`

	// DocScan defines document scanning configuration.
	DocScan DocScan `yaml:"doc_scan"`
}

// BaselineConfig defines settings for CRAP score baseline comparison.
type BaselineConfig struct {
	// File is the path to the baseline JSON file.
	// Default: ".gaze/baseline.json"
	File string `yaml:"file"`

	// Epsilon is the minimum score delta to trigger a
	// regression or improvement classification. Must be >= 0.
	// Default: 0.5
	Epsilon float64 `yaml:"epsilon"`

	// NewFunctionThreshold is the CRAP score above which a new
	// function (not in the baseline) is classified as a
	// violation. Must be > 0. Default: 30.
	NewFunctionThreshold float64 `yaml:"new_function_threshold"`
}

// GazeConfig is the top-level configuration loaded from .gaze.yaml.
type GazeConfig struct {
	// Classification holds classification-related settings.
	Classification ClassificationConfig `yaml:"classification"`

	// Baseline holds baseline comparison settings.
	Baseline BaselineConfig `yaml:"baseline"`
}

// DefaultConfig returns a GazeConfig with sensible defaults.
// The default exclude list matches FR-009.
func DefaultConfig() *GazeConfig {
	return &GazeConfig{
		Baseline: BaselineConfig{
			File:                 ".gaze/baseline.json",
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
		},
		Classification: ClassificationConfig{
			Thresholds: Thresholds{
				Contractual: 80,
				Incidental:  50,
			},
			DocScan: DocScan{
				Exclude: []string{
					"vendor/**",
					"node_modules/**",
					".git/**",
					"testdata/**",
					"CHANGELOG.md",
					"CONTRIBUTING.md",
					"CODE_OF_CONDUCT.md",
					"LICENSE",
					"LICENSE.md",
				},
				Include:    nil,
				Timeout:    30 * time.Second,
				TimeoutStr: "30s",
			},
		},
	}
}

// Load reads a .gaze.yaml configuration file from the given path.
// If the file does not exist, it returns DefaultConfig without error.
// If the file exists but is invalid, it returns an error.
func Load(path string) (*GazeConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config %q: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config %q: %w", path, err)
	}

	// Validate baseline configuration.
	if cfg.Baseline.Epsilon < 0 {
		return nil, fmt.Errorf("baseline.epsilon must be >= 0, got %g", cfg.Baseline.Epsilon)
	}
	if cfg.Baseline.NewFunctionThreshold <= 0 {
		return nil, fmt.Errorf("baseline.new_function_threshold must be > 0, got %g",
			cfg.Baseline.NewFunctionThreshold)
	}

	// Parse timeout string if provided.
	if cfg.Classification.DocScan.TimeoutStr != "" {
		d, err := time.ParseDuration(cfg.Classification.DocScan.TimeoutStr)
		if err != nil {
			return nil, fmt.Errorf("parsing doc_scan.timeout %q: %w",
				cfg.Classification.DocScan.TimeoutStr, err)
		}
		cfg.Classification.DocScan.Timeout = d
	}

	return cfg, nil
}
