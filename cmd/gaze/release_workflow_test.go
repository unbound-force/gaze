package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReleaseWorkflow_MacOSSigningIdentity(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "release.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}

	contents := string(workflow)
	for _, expected := range []string{
		`if [ -n "$MACOS_SIGN_P12" ] && [ -n "$MACOS_SIGN_IDENTITY" ]; then`,
		`MACOS_SIGN_IDENTITY: ${{ secrets.MACOS_SIGN_IDENTITY }}`,
		`needs.check-signing-secrets.outputs.has_signing_secrets == 'true'`,
		`needs.check-signing-secrets.outputs.has_signing_secrets == 'false'`,
	} {
		if !strings.Contains(contents, expected) {
			t.Errorf("release workflow must contain %q", expected)
		}
	}

	for _, forbidden := range []string{
		`vars.MACOS_SIGN_IDENTITY`,
		`Developer ID Application:`,
	} {
		if strings.Contains(contents, forbidden) {
			t.Errorf("release workflow must not contain %q", forbidden)
		}
	}
}
