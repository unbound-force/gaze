package apidoc

import (
	"testing"

	"github.com/unbound-force/gaze/internal/docscan"
)

func TestValidateReferences_RenamedSymbolDetected(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path: "README.md",
			Content: "line one\n" +
				"Use `OldFunctionName` to process data.\n" +
				"line three\n",
		},
	}
	symbols := map[string]bool{
		"NewFunctionName": true,
	}

	refs := ValidateReferences(docs, symbols)

	if len(refs) != 1 {
		t.Fatalf("expected 1 stale reference, got %d", len(refs))
	}
	if refs[0].Symbol != "OldFunctionName" {
		t.Errorf("expected symbol OldFunctionName, got %s", refs[0].Symbol)
	}
	if refs[0].DocFile != "README.md" {
		t.Errorf("expected doc_file README.md, got %s", refs[0].DocFile)
	}
	if refs[0].DocLine != 2 {
		t.Errorf("expected doc_line 2, got %d", refs[0].DocLine)
	}
}

func TestValidateReferences_ValidSymbolNotFlagged(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path:    "README.md",
			Content: "Call `ProcessData` to start.\n",
		},
	}
	symbols := map[string]bool{
		"ProcessData": true,
	}

	refs := ValidateReferences(docs, symbols)

	if len(refs) != 0 {
		t.Fatalf("expected 0 stale references, got %d: %+v", len(refs), refs)
	}
}

func TestValidateReferences_NonSymbolContentIgnored(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path: "README.md",
			Content: "Use `--verbose` for debug output.\n" +
				"See `/path/to/file` for details.\n" +
				"Set `$HOME` before running.\n" +
				"Returns `nil` on failure.\n" +
				"Use `true` or `false`.\n" +
				"The `error` interface.\n" +
				"A `string` value.\n" +
				"An `int` counter.\n" +
				"A `bool` flag.\n" +
				"The `any` type.\n" +
				"A `func` literal.\n" +
				"An `if` statement.\n" +
				"A `for` loop.\n" +
				"The `return` keyword.\n" +
				"Use `defer` for cleanup.\n" +
				"The `go` keyword.\n" +
				"Run `go-test` to verify.\n" +
				"Use `golangci-lint` for linting.\n",
		},
	}
	// Empty symbol set — everything not ignored would be stale.
	symbols := map[string]bool{}

	refs := ValidateReferences(docs, symbols)

	if len(refs) != 0 {
		t.Fatalf("expected 0 stale references (all should be ignored), got %d: %+v", len(refs), refs)
	}
}

func TestValidateReferences_EmptySymbolSet(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path: "README.md",
			Content: "Call `ProcessData` and `HandleRequest`.\n" +
				"Also see `Config`.\n",
		},
	}
	symbols := map[string]bool{}

	refs := ValidateReferences(docs, symbols)

	if len(refs) != 3 {
		t.Fatalf("expected 3 stale references, got %d: %+v", len(refs), refs)
	}

	// Verify all three symbols are reported.
	found := map[string]bool{}
	for _, r := range refs {
		found[r.Symbol] = true
	}
	for _, want := range []string{"ProcessData", "HandleRequest", "Config"} {
		if !found[want] {
			t.Errorf("expected stale reference for %s, not found", want)
		}
	}
}

func TestValidateReferences_EmptyDocuments(t *testing.T) {
	symbols := map[string]bool{
		"ProcessData": true,
	}

	refs := ValidateReferences(nil, symbols)

	if len(refs) != 0 {
		t.Fatalf("expected 0 stale references for empty docs, got %d", len(refs))
	}
}

func TestValidateReferences_FencedCodeBlockContentSkipped(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path: "README.md",
			Content: "Use `ValidSymbol` in your code.\n" +
				"```go\n" +
				"result := OldFunction()\n" +
				"fmt.Println(`OldFunction`)\n" +
				"```\n" +
				"After the code block, `AnotherStale` appears.\n",
		},
	}
	symbols := map[string]bool{
		"ValidSymbol": true,
	}

	refs := ValidateReferences(docs, symbols)

	// Only AnotherStale should be reported. Content inside the
	// fenced code block (OldFunction) must not be scanned.
	if len(refs) != 1 {
		t.Fatalf("expected 1 stale reference, got %d: %+v", len(refs), refs)
	}
	if refs[0].Symbol != "AnotherStale" {
		t.Errorf("expected symbol AnotherStale, got %s", refs[0].Symbol)
	}
	if refs[0].DocLine != 6 {
		t.Errorf("expected doc_line 6, got %d", refs[0].DocLine)
	}
}

func TestValidateReferences_MultipleFilesAndLines(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path:    "README.md",
			Content: "`Alpha` is great.\n`Beta` too.\n",
		},
		{
			Path:    "docs/guide.md",
			Content: "See `Gamma` for details.\n",
		},
	}
	symbols := map[string]bool{
		"Beta": true,
	}

	refs := ValidateReferences(docs, symbols)

	if len(refs) != 2 {
		t.Fatalf("expected 2 stale references, got %d: %+v", len(refs), refs)
	}

	// Alpha from README.md line 1, Gamma from docs/guide.md line 1.
	found := map[string]string{}
	for _, r := range refs {
		found[r.Symbol] = r.DocFile
	}
	if found["Alpha"] != "README.md" {
		t.Errorf("expected Alpha in README.md, got %s", found["Alpha"])
	}
	if found["Gamma"] != "docs/guide.md" {
		t.Errorf("expected Gamma in docs/guide.md, got %s", found["Gamma"])
	}
}

// --- ValidateCodeBlocks tests ---

func TestValidateCodeBlocks_WrongLanguageTag(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path: "docs/tutorial.md",
			Content: "Some text.\n" +
				"```python\n" +
				"print('hello')\n" +
				"```\n",
		},
	}

	issues := ValidateCodeBlocks(docs, "go")

	if len(issues) != 1 {
		t.Fatalf("expected 1 code block issue, got %d", len(issues))
	}
	if issues[0].DeclaredLang != "python" {
		t.Errorf("expected declared_lang python, got %s", issues[0].DeclaredLang)
	}
	if issues[0].ExpectedLang != "go" {
		t.Errorf("expected expected_lang go, got %s", issues[0].ExpectedLang)
	}
	if issues[0].DocFile != "docs/tutorial.md" {
		t.Errorf("expected doc_file docs/tutorial.md, got %s", issues[0].DocFile)
	}
	if issues[0].DocLine != 2 {
		t.Errorf("expected doc_line 2, got %d", issues[0].DocLine)
	}
}

func TestValidateCodeBlocks_UntaggedCodeBlock(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path: "README.md",
			Content: "Example:\n" +
				"```\n" +
				"some output\n" +
				"```\n",
		},
	}

	issues := ValidateCodeBlocks(docs, "go")

	if len(issues) != 0 {
		t.Fatalf("expected 0 issues for untagged code block, got %d: %+v", len(issues), issues)
	}
}

func TestValidateCodeBlocks_MatchingLanguageTag(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path: "docs/api.md",
			Content: "Example:\n" +
				"```go\n" +
				"func main() {}\n" +
				"```\n",
		},
	}

	issues := ValidateCodeBlocks(docs, "go")

	if len(issues) != 0 {
		t.Fatalf("expected 0 issues for matching language tag, got %d: %+v", len(issues), issues)
	}
}

func TestValidateCodeBlocks_GenericTagNotFlagged(t *testing.T) {
	// All generic tags should be ignored regardless of expected language.
	docs := []docscan.DocumentFile{
		{
			Path: "README.md",
			Content: "```json\n{\"key\": \"value\"}\n```\n" +
				"```yaml\nkey: value\n```\n" +
				"```bash\necho hello\n```\n" +
				"```shell\nls -la\n```\n" +
				"```text\nplain text\n```\n" +
				"```diff\n+added\n```\n" +
				"```dockerfile\nFROM alpine\n```\n",
		},
	}

	issues := ValidateCodeBlocks(docs, "python")

	if len(issues) != 0 {
		t.Fatalf("expected 0 issues for generic tags, got %d: %+v", len(issues), issues)
	}
}

func TestValidateCodeBlocks_EmptyExpectedLanguage(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path: "README.md",
			Content: "```python\nprint('hello')\n```\n" +
				"```ruby\nputs 'hello'\n```\n",
		},
	}

	issues := ValidateCodeBlocks(docs, "")

	if issues != nil {
		t.Fatalf("expected nil for empty expected language, got %+v", issues)
	}
}

func TestValidateCodeBlocks_MultipleBlocksOnlyMismatchReported(t *testing.T) {
	// Spec scenario: ```go at line 5, ```python at line 20,
	// ``` at line 35, ```json at line 50. Expected "go".
	// Only the python block at line 20 should be reported.
	docs := []docscan.DocumentFile{
		{
			Path: "docs/tutorial.md",
			Content: "intro\n" + // line 1
				"more intro\n" + // line 2
				"even more\n" + // line 3
				"setup:\n" + // line 4
				"```go\n" + // line 5
				"func main() {}\n" + // line 6
				"```\n" + // line 7
				"text\n" + // line 8
				"text\n" + // line 9
				"text\n" + // line 10
				"text\n" + // line 11
				"text\n" + // line 12
				"text\n" + // line 13
				"text\n" + // line 14
				"text\n" + // line 15
				"text\n" + // line 16
				"text\n" + // line 17
				"text\n" + // line 18
				"now python:\n" + // line 19
				"```python\n" + // line 20
				"print('hello')\n" + // line 21
				"```\n" + // line 22
				"text\n" + // line 23
				"text\n" + // line 24
				"text\n" + // line 25
				"text\n" + // line 26
				"text\n" + // line 27
				"text\n" + // line 28
				"text\n" + // line 29
				"text\n" + // line 30
				"text\n" + // line 31
				"text\n" + // line 32
				"text\n" + // line 33
				"untagged:\n" + // line 34
				"```\n" + // line 35
				"some output\n" + // line 36
				"```\n" + // line 37
				"text\n" + // line 38
				"text\n" + // line 39
				"text\n" + // line 40
				"text\n" + // line 41
				"text\n" + // line 42
				"text\n" + // line 43
				"text\n" + // line 44
				"text\n" + // line 45
				"text\n" + // line 46
				"text\n" + // line 47
				"text\n" + // line 48
				"json data:\n" + // line 49
				"```json\n" + // line 50
				"{\"key\": \"value\"}\n" + // line 51
				"```\n", // line 52
		},
	}

	issues := ValidateCodeBlocks(docs, "go")

	if len(issues) != 1 {
		t.Fatalf("expected exactly 1 issue, got %d: %+v", len(issues), issues)
	}
	if issues[0].DeclaredLang != "python" {
		t.Errorf("expected declared_lang python, got %s", issues[0].DeclaredLang)
	}
	if issues[0].DocLine != 20 {
		t.Errorf("expected doc_line 20, got %d", issues[0].DocLine)
	}
}

func TestValidateCodeBlocks_MultipleFiles(t *testing.T) {
	docs := []docscan.DocumentFile{
		{
			Path:    "README.md",
			Content: "```rust\nfn main() {}\n```\n",
		},
		{
			Path:    "docs/guide.md",
			Content: "```go\nfunc main() {}\n```\n",
		},
	}

	issues := ValidateCodeBlocks(docs, "go")

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d: %+v", len(issues), issues)
	}
	if issues[0].DocFile != "README.md" {
		t.Errorf("expected doc_file README.md, got %s", issues[0].DocFile)
	}
	if issues[0].DeclaredLang != "rust" {
		t.Errorf("expected declared_lang rust, got %s", issues[0].DeclaredLang)
	}
}

// --- Table-driven tests for shouldIgnoreBacktickContent ---

func TestShouldIgnoreBacktickContent(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		ignore bool
	}{
		{"CLI flag double dash", "--verbose", true},
		{"CLI flag single dash", "-v", true},
		{"file path", "/path/to/file", true},
		{"URL-like", "http://example.com", true},
		{"env var", "$HOME", true},
		{"keyword nil", "nil", true},
		{"keyword true", "true", true},
		{"keyword false", "false", true},
		{"keyword error", "error", true},
		{"keyword string", "string", true},
		{"keyword int", "int", true},
		{"keyword bool", "bool", true},
		{"keyword any", "any", true},
		{"keyword func", "func", true},
		{"keyword if", "if", true},
		{"keyword for", "for", true},
		{"keyword return", "return", true},
		{"keyword defer", "defer", true},
		{"keyword go", "go", true},
		{"command name", "go-test", true},
		{"command name multi-hyphen", "golangci-lint", true},
		{"PascalCase symbol", "ProcessData", false},
		{"camelCase symbol", "processData", false},
		{"uppercase acronym", "HTTP", false},
		{"mixed case with digits", "V2Handler", false},
		{"single uppercase letter", "T", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldIgnoreBacktickContent(tt.input)
			if got != tt.ignore {
				t.Errorf("shouldIgnoreBacktickContent(%q) = %v, want %v", tt.input, got, tt.ignore)
			}
		})
	}
}

// --- Table-driven test for GenericLanguageTags completeness ---

func TestGenericLanguageTags_Completeness(t *testing.T) {
	// Verify the spec-mandated exhaustive set is present.
	expected := []string{
		"text", "plaintext", "console",
		"shell", "bash", "sh", "zsh",
		"json", "yaml", "yml", "toml",
		"xml", "html", "css", "sql",
		"diff", "ini", "csv",
		"makefile", "dockerfile",
		"markdown", "md",
		"output", "log",
	}

	tags := GenericLanguageTags()
	for _, tag := range expected {
		if !tags[tag] {
			t.Errorf("GenericLanguageTags() missing required tag %q", tag)
		}
	}

	// Verify no unexpected tags were added.
	if len(tags) != len(expected) {
		t.Errorf("GenericLanguageTags() has %d entries, expected %d", len(tags), len(expected))
	}
}
