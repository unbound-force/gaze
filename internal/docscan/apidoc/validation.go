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

// ignoredSymbolsList is the set of Go keywords and builtins that
// should not be treated as API symbol references in documentation.
// Defined as a slice to avoid mutable package-level maps (CS-007).
var ignoredSymbolsList = [...]string{
	"nil", "true", "false",
	"error", "string", "int",
	"bool", "any", "func",
	"if", "for", "return",
	"defer", "go",
}

// isIgnoredSymbol reports whether s is a Go keyword or builtin that
// should be excluded from symbol reference detection.
func isIgnoredSymbol(s string) bool {
	for _, sym := range ignoredSymbolsList {
		if s == sym {
			return true
		}
	}
	return false
}

// allLowerHyphenRe matches strings that consist entirely of lowercase
// letters and hyphens, such as command names like "go-test" or
// "golangci-lint".
var allLowerHyphenRe = regexp.MustCompile(`^[a-z][a-z-]*$`)

// genericLanguageTagsList is the exhaustive set of language tags that
// are considered generic and excluded from code block language
// validation. These tags typically represent data formats, shell
// commands, or output rather than source code in a specific
// programming language. Defined as a slice to avoid mutable
// package-level maps (CS-007).
var genericLanguageTagsList = [...]string{
	"text", "plaintext", "console",
	"shell", "bash", "sh", "zsh",
	"json", "yaml", "yml", "toml",
	"xml", "html", "css", "sql",
	"diff", "ini", "csv",
	"makefile", "dockerfile",
	"markdown", "md",
	"output", "log",
}

// isGenericLanguageTag reports whether lang is a generic language
// tag that should be excluded from code block validation.
func isGenericLanguageTag(lang string) bool {
	for _, tag := range genericLanguageTagsList {
		if lang == tag {
			return true
		}
	}
	return false
}

// GenericLanguageTags returns the exhaustive set of language tags
// considered generic. The returned map is a fresh copy safe for
// mutation by callers.
func GenericLanguageTags() map[string]bool {
	result := make(map[string]bool, len(genericLanguageTagsList))
	for _, tag := range genericLanguageTagsList {
		result[tag] = true
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
	if isIgnoredSymbol(s) {
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
			if isGenericLanguageTag(lang) {
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
