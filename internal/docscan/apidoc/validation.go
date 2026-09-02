package apidoc

import (
	"regexp"
	"strings"

	"github.com/unbound-force/gaze/internal/docscan"
)

// backtickRe matches single-backtick-quoted content. It captures the
// text between a pair of single backticks, excluding triple-backtick
// fences which are handled separately.
var backtickRe = regexp.MustCompile("`([^`]+)`")

// ignoredSymbols is the set of Go keywords and builtins that should
// not be treated as API symbol references in documentation.
var ignoredSymbols = map[string]bool{
	"nil": true, "true": true, "false": true,
	"error": true, "string": true, "int": true,
	"bool": true, "any": true, "func": true,
	"if": true, "for": true, "return": true,
	"defer": true, "go": true,
}

// allLowerHyphenRe matches strings that consist entirely of lowercase
// letters and hyphens, such as command names like "go-test" or
// "golangci-lint".
var allLowerHyphenRe = regexp.MustCompile(`^[a-z][a-z-]*$`)

// genericLanguageTags is the exhaustive set of language tags that
// are considered generic and excluded from code block language
// validation. These tags typically represent data formats, shell
// commands, or output rather than source code in a specific
// programming language.
var genericLanguageTags = map[string]bool{
	"text": true, "plaintext": true, "console": true,
	"shell": true, "bash": true, "sh": true, "zsh": true,
	"json": true, "yaml": true, "yml": true, "toml": true,
	"xml": true, "html": true, "css": true, "sql": true,
	"diff": true, "ini": true, "csv": true,
	"makefile": true, "dockerfile": true,
	"markdown": true, "md": true,
	"output": true, "log": true,
}

// GenericLanguageTags returns the exhaustive set of language tags
// considered generic. The returned map is a copy safe for mutation.
func GenericLanguageTags() map[string]bool {
	result := make(map[string]bool, len(genericLanguageTags))
	for k, v := range genericLanguageTags {
		result[k] = v
	}
	return result
}

// ValidateReferences scans documentation files for backtick-quoted
// symbol names and reports any that are not in the known symbol set.
// It identifies stale references that may indicate renamed or removed
// API elements.
//
// Content inside fenced code blocks (triple-backtick regions) is
// skipped to avoid false positives from code examples. Backtick
// content matching common non-symbol patterns (CLI flags, file paths,
// environment variables, Go keywords, and command names) is also
// excluded.
func ValidateReferences(docs []docscan.DocumentFile, symbolNames map[string]bool) []StaleReference {
	var refs []StaleReference

	for _, doc := range docs {
		lines := strings.Split(doc.Content, "\n")
		inCodeBlock := false

		for lineIdx, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Track fenced code block boundaries. A line starting
			// with ``` toggles the in-code-block state.
			if strings.HasPrefix(trimmed, "```") {
				inCodeBlock = !inCodeBlock
				continue
			}

			// Skip content inside fenced code blocks — these are
			// code examples, not symbol references.
			if inCodeBlock {
				continue
			}

			matches := backtickRe.FindAllStringSubmatch(line, -1)
			for _, m := range matches {
				symbol := m[1]

				if shouldIgnoreBacktickContent(symbol) {
					continue
				}

				if !symbolNames[symbol] {
					refs = append(refs, StaleReference{
						Symbol:  symbol,
						DocFile: doc.Path,
						DocLine: lineIdx + 1, // 1-indexed
					})
				}
			}
		}
	}

	return refs
}

// shouldIgnoreBacktickContent returns true if the backtick-quoted
// content matches a pattern that is unlikely to be an API symbol
// reference: CLI flags, file paths, environment variables, Go
// keywords/builtins, or all-lowercase-hyphenated command names.
func shouldIgnoreBacktickContent(s string) bool {
	// CLI flags: --verbose, -v
	if strings.HasPrefix(s, "-") {
		return true
	}

	// File paths and URLs: /path/to/file, http://...
	if strings.Contains(s, "/") {
		return true
	}

	// Environment variables: $HOME, $PATH
	if strings.HasPrefix(s, "$") {
		return true
	}

	// Go keywords and builtins
	if ignoredSymbols[s] {
		return true
	}

	// All-lowercase-with-hyphens command names: go-test, golangci-lint
	if allLowerHyphenRe.MatchString(s) {
		return true
	}

	return false
}

// ValidateCodeBlocks finds fenced code blocks in documentation files
// whose language tags do not match the expected language from the
// analyzer. Generic language tags (json, yaml, shell, etc.) are
// excluded from validation.
//
// When expectedLang is empty, validation is skipped and nil is
// returned.
func ValidateCodeBlocks(docs []docscan.DocumentFile, expectedLang string) []CodeBlockIssue {
	if expectedLang == "" {
		return nil
	}

	var issues []CodeBlockIssue

	for _, doc := range docs {
		lines := strings.Split(doc.Content, "\n")
		inCodeBlock := false

		for lineIdx, line := range lines {
			trimmed := strings.TrimSpace(line)

			if !strings.HasPrefix(trimmed, "```") {
				continue
			}

			// Toggle code block state. Opening fences may have a
			// language tag; closing fences do not.
			if inCodeBlock {
				inCodeBlock = false
				continue
			}

			inCodeBlock = true

			// Extract the language tag after the triple backticks.
			lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			if lang == "" {
				// Untagged code block — skip.
				continue
			}

			// Generic tags are excluded from validation.
			if genericLanguageTags[lang] {
				continue
			}

			// Matching tag — no issue.
			if lang == expectedLang {
				continue
			}

			issues = append(issues, CodeBlockIssue{
				DocFile:      doc.Path,
				DocLine:      lineIdx + 1, // 1-indexed
				DeclaredLang: lang,
				ExpectedLang: expectedLang,
			})
		}
	}

	return issues
}
