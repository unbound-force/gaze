package overspecd

import (
	"bytes"
	"log"
	"strings"
	"testing"
)

func TestProcess(t *testing.T) {
	// Capture log output — this is over-specification because
	// log output is an incidental side effect.
	var logBuf bytes.Buffer
	orig := log.Writer()
	log.SetOutput(&logBuf)
	defer log.SetOutput(orig)

	got := Process(5)
	if got != 10 {
		t.Errorf("Process(5) = %d, want 10", got)
	}

	// Over-specified: asserting on log output (incidental).
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "processed") {
		t.Errorf("expected log to contain 'processed', got %q", logOutput)
	}
}

func TestFormat(t *testing.T) {
	got := Format("item", 42)
	if got != "item-42" {
		t.Errorf("Format(\"item\", 42) = %q, want \"item-42\"", got)
	}
}
