package aireport

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// rubricDocPath is the published rubric, resolved from the package
// directory so the test runs under `go test ./...`.
const rubricDocPath = "../../docs/reference/cli/report.md"

// TestRubricDocMatchesCode verifies that the published Grading rubric
// table is the one grade.go implements.
//
// The document is the contract readers and CI consumers rely on, so a
// bound that moves in code without moving in the document is a test
// failure, not a documentation nit.
func TestRubricDocMatchesCode(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(rubricDocPath))
	if err != nil {
		t.Fatalf("read %s: %v", rubricDocPath, err)
	}
	doc := string(data)

	if !strings.Contains(doc, "rubric version `1`") {
		t.Errorf("document does not publish rubric version `1`")
	}

	rows := parseRubricTable(t, doc)
	if len(rows) != len(gradeLadder)+1 {
		t.Fatalf("rubric table has %d graded rows, want %d", len(rows), len(gradeLadder)+1)
	}

	for i, grade := range gradeLadder {
		cells, ok := rows[string(grade)]
		if !ok {
			t.Errorf("rubric table is missing grade %q", grade)
			continue
		}
		want := []string{
			tenthsCell(crapLoadBounds[i]),
			tenthsCell(gazeCRAPLoadBounds[i]),
			tenthsCell(lineCoverageBounds[i]),
			strconv.Itoa(contractCoverageBounds[i]),
			tenthsCell(complexityBounds[i]),
		}
		if len(cells) != len(want) {
			t.Errorf("grade %q has %d bound cells, want %d", grade, len(cells), len(want))
			continue
		}
		for j := range want {
			if cells[j] != want[j] {
				t.Errorf("grade %q cell %d = %q, want %q", grade, j+1, cells[j], want[j])
			}
		}
	}

	last := len(gradeLadder) - 1
	wantF := []string{
		"> " + tenthsCell(crapLoadBounds[last]),
		"> " + tenthsCell(gazeCRAPLoadBounds[last]),
		"< " + tenthsCell(lineCoverageBounds[last]),
		"< " + strconv.Itoa(contractCoverageBounds[last]),
		"> " + tenthsCell(complexityBounds[last]),
	}
	fCells, ok := rows["F"]
	if !ok {
		t.Fatalf("rubric table is missing the F row")
	}
	for j := range wantF {
		if j >= len(fCells) {
			t.Errorf("F row is missing cell %d, want %q", j+1, wantF[j])
			continue
		}
		if fCells[j] != wantF[j] {
			t.Errorf("F row cell %d = %q, want %q", j+1, fCells[j], wantF[j])
		}
	}
}

// tenthsCell renders a bound stored in tenths as the document's
// one-decimal value.
func tenthsCell(tenths int) string {
	return strconv.FormatFloat(float64(tenths)/10, 'f', 1, 64)
}

// parseRubricTable extracts the rubric table's data rows from doc,
// keyed by grade. Each value holds the five bound cells.
func parseRubricTable(t *testing.T, doc string) map[string][]string {
	t.Helper()

	lines := strings.Split(doc, "\n")
	start := indexOfLine(lines, "## Grading rubric")
	if start < 0 {
		t.Fatalf("document has no %q section", "## Grading rubric")
	}

	rows := make(map[string][]string)
	inTable := false
	for _, line := range lines[start+1:] {
		switch {
		case strings.HasPrefix(line, "## "):
			return rows
		case strings.HasPrefix(line, "| Grade |"):
			inTable = true
			continue
		case !inTable:
			continue
		case !strings.HasPrefix(line, "|"):
			if len(rows) > 0 {
				return rows
			}
			continue
		}

		cells := splitTableRow(line)
		if len(cells) < 2 || isTableSeparator(cells[0]) {
			continue
		}
		rows[cells[0]] = cells[1:]
	}
	return rows
}

// indexOfLine returns the index of the first line equal to want.
func indexOfLine(lines []string, want string) int {
	for i, line := range lines {
		if line == want {
			return i
		}
	}
	return -1
}

// splitTableRow splits a Markdown table row into trimmed cells,
// dropping the empty cells outside the outer pipes.
func splitTableRow(line string) []string {
	parts := strings.Split(strings.TrimSpace(line), "|")
	parts = parts[1 : len(parts)-1]
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

// isTableSeparator reports whether cell is a Markdown table's
// delimiter cell, made only of dashes and colons.
func isTableSeparator(cell string) bool {
	if cell == "" {
		return false
	}
	for _, r := range cell {
		if r != '-' && r != ':' {
			return false
		}
	}
	return true
}
