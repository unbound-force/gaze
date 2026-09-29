package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestModulePath_MatchesMajorVersion verifies that the go.mod module
// path includes the /v2 suffix required by Go's Import Compatibility
// Rule for modules tagged at v2 or later. Without this suffix,
// `go install` resolves to the latest v1 release instead of v2.
func TestModulePath_MatchesMajorVersion(t *testing.T) {
	modPath := filepath.Join("..", "..", "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}

	lines := strings.Split(string(data), "\n")
	var moduleLine string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			moduleLine = trimmed
			break
		}
	}

	if moduleLine == "" {
		t.Fatal("go.mod does not contain a module directive")
	}

	// Extract the module path (everything after "module ")
	modName := strings.TrimPrefix(moduleLine, "module ")
	modName = strings.TrimSpace(modName)

	if !strings.HasSuffix(modName, "/v2") {
		t.Errorf("module path %q missing /v2 suffix; go install will resolve to v1", modName)
	}
}
